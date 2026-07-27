package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
)

const maxSSEEventBytes = 8 << 20

type ChatRequest struct {
	Input               string   `json:"input"`
	SessionUUID         *string  `json:"sessionUuid"`
	ModelName           *string  `json:"modelName"`
	PromptTemplateID    *int64   `json:"promptTemplateId"`
	OrgTags             []string `json:"orgTags"`
	TargetFileMD5s      []string `json:"targetFileMd5s"`
	Stream              bool     `json:"stream"`
	IncludeReferences   bool     `json:"includeReferences"`
	GenerateSuggestions bool     `json:"generateSuggestions"`
}

type ChatCompletionResult struct {
	RequestID    string           `json:"requestId"`
	SessionUUID  string           `json:"sessionUuid"`
	Content      string           `json:"content"`
	FinishReason string           `json:"finishReason"`
	References   []map[string]any `json:"references"`
	Usage        map[string]any   `json:"usage"`
	Suggestions  []string         `json:"suggestions"`
}

type ChatEvent struct {
	Type      string         `json:"type"`
	RequestID string         `json:"requestId"`
	Sequence  int64          `json:"sequence"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

type CancelChatResult struct {
	RequestID string `json:"requestId"`
	Status    string `json:"status"`
}

func (c *Client) CompleteChat(ctx context.Context, requestID, idempotencyKey string, request ChatRequest) (ChatCompletionResult, error) {
	request.Stream = false
	var result ChatCompletionResult
	err := c.DoContent(
		ctx, http.MethodPost, "/chat/completions", requestID, "application/json",
		map[string]string{"Accept": "application/json", "Idempotency-Key": idempotencyKey},
		mustJSONReader(request), &result,
	)
	normalizeChatResult(&result)
	return result, err
}

func (c *Client) StreamChat(
	ctx context.Context,
	requestID, idempotencyKey string,
	request ChatRequest,
	handle func(ChatEvent) error,
) error {
	if strings.TrimSpace(c.token) == "" {
		return apperr.New(apperr.ExitAuthentication, "尚未配置 API Token，请运行 emoera auth set-token")
	}
	request.Stream = true
	body, err := json.Marshal(request)
	if err != nil {
		return apperr.Wrap(apperr.ExitArguments, "请求参数无法编码", err)
	}
	httpRequest, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.endpoint+"/api/v1/agent/chat/completions", bytes.NewReader(body),
	)
	if err != nil {
		return apperr.Wrap(apperr.ExitConfiguration, "无法创建请求", err)
	}
	httpRequest.Header.Set("Authorization", "Bearer "+c.token)
	httpRequest.Header.Set("Accept", "text/event-stream")
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Idempotency-Key", idempotencyKey)
	httpRequest.Header.Set("User-Agent", c.userAgent)
	if requestID != "" {
		httpRequest.Header.Set("X-Request-Id", requestID)
	}

	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return context.Canceled
		}
		return mapTransportError(err)
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		return decodeEnvelopeError(response)
	}
	if !strings.HasPrefix(strings.ToLower(response.Header.Get("Content-Type")), "text/event-stream") {
		return apperr.New(apperr.ExitServer, "服务未返回 SSE 流")
	}
	return decodeSSE(response.Body, handle)
}

func (c *Client) CancelChat(ctx context.Context, requestID, httpRequestID string) (CancelChatResult, error) {
	var result CancelChatResult
	err := c.Do(ctx, http.MethodPost, "/chat/runs/"+requestID+"/cancel", httpRequestID, nil, &result)
	return result, err
}

func decodeSSE(reader io.Reader, handle func(ChatEvent) error) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64<<10), maxSSEEventBytes)
	var eventName string
	terminal := false
	dataLines := make([]string, 0, 1)
	dispatch := func() error {
		if len(dataLines) == 0 {
			eventName = ""
			return nil
		}
		var event ChatEvent
		if err := json.Unmarshal([]byte(strings.Join(dataLines, "\n")), &event); err != nil {
			return apperr.Wrap(apperr.ExitServer, "SSE 事件 JSON 无法解析", err)
		}
		if event.Type == "" {
			event.Type = eventName
		}
		if event.Data == nil {
			event.Data = map[string]any{}
		}
		if eventName != "" && event.Type != eventName {
			return apperr.New(apperr.ExitServer, "SSE 事件名称与数据类型不一致")
		}
		dataLines = dataLines[:0]
		eventName = ""
		if handle != nil {
			if err := handle(event); err != nil {
				return err
			}
		}
		if event.Type == "error" {
			terminal = true
			return mapChatEventError(event)
		}
		if event.Type == "done" {
			terminal = true
		}
		return nil
	}

	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			if err := dispatch(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, value, found := strings.Cut(line, ":")
		if found && strings.HasPrefix(value, " ") {
			value = value[1:]
		}
		switch field {
		case "event":
			eventName = value
		case "data":
			dataLines = append(dataLines, value)
		}
	}
	// A final SSE event does not have to be followed by another successful read.
	// Some reverse proxies close a chunked response immediately after `done` and
	// Go reports that close as io.ErrUnexpectedEOF. Dispatch any buffered event
	// first, then trust the protocol-level terminal event over the transport close.
	scanErr := scanner.Err()
	if err := dispatch(); err != nil {
		return err
	}
	if scanErr != nil {
		if terminal {
			return nil
		}
		return mapTransportError(scanErr)
	}
	if !terminal {
		return apperr.New(apperr.ExitServer, "SSE 流在完成事件前意外结束")
	}
	return nil
}

func decodeEnvelopeError(response *http.Response) error {
	content, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return apperr.Wrap(apperr.ExitNetwork, "读取服务响应失败", err)
	}
	var envelope Envelope
	if err := json.Unmarshal(content, &envelope); err != nil {
		return apperr.New(apperr.ExitServer, fmt.Sprintf("服务返回了无效响应（HTTP %d）", response.StatusCode))
	}
	return mapAPIError(response.StatusCode, envelope.Error)
}

func mapChatEventError(event ChatEvent) error {
	code := stringValue(event.Data["code"])
	message := stringValue(event.Data["message"])
	if message == "" {
		message = "问答处理失败"
	}
	exitCode := apperr.ExitServer
	switch code {
	case "INVALID_REQUEST", "VALIDATION_FAILED":
		exitCode = apperr.ExitArguments
	case "ACCESS_DENIED", "INSUFFICIENT_SCOPE":
		exitCode = apperr.ExitPermission
	case "RESOURCE_NOT_FOUND":
		exitCode = apperr.ExitNotFound
	case "IDEMPOTENCY_CONFLICT":
		exitCode = apperr.ExitConflict
	case "QUOTA_EXCEEDED":
		exitCode = apperr.ExitQuota
	case "RATE_LIMITED":
		exitCode = apperr.ExitRateLimited
	case "UPSTREAM_TIMEOUT":
		exitCode = apperr.ExitTimeout
	}
	if code != "" {
		message = code + ": " + message
	}
	return apperr.New(exitCode, message)
}

func mapTransportError(err error) error {
	var netError interface{ Timeout() bool }
	if errors.As(err, &netError) && netError.Timeout() {
		return apperr.Wrap(apperr.ExitTimeout, "请求超时", err)
	}
	return apperr.Wrap(apperr.ExitNetwork, "无法连接 Emoera 服务", err)
}

func mustJSONReader(value any) io.Reader {
	content, _ := json.Marshal(value)
	return bytes.NewReader(content)
}

func normalizeChatResult(result *ChatCompletionResult) {
	if result.References == nil {
		result.References = []map[string]any{}
	}
	if result.Usage == nil {
		result.Usage = map[string]any{}
	}
	if result.Suggestions == nil {
		result.Suggestions = []string{}
	}
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

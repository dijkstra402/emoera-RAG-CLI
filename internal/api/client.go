package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
)

type Client struct {
	endpoint   string
	token      string
	userAgent  string
	httpClient *http.Client
}

type Envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error *APIError       `json:"error,omitempty"`
	Meta  Meta            `json:"meta"`
}

type APIError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details"`
}

type Meta struct {
	RequestID  string    `json:"requestId"`
	APIVersion string    `json:"apiVersion"`
	Timestamp  time.Time `json:"timestamp"`
}

func New(endpoint, token, userAgent string, timeout time.Duration) (*Client, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, apperr.New(apperr.ExitConfiguration, "Endpoint 不是有效 URL")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname())) {
		return nil, apperr.New(
			apperr.ExitConfiguration,
			"非本机 Endpoint 必须使用 HTTPS",
		)
	}
	return &Client{
		endpoint: strings.TrimRight(endpoint, "/"), token: token, userAgent: userAgent,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (c *Client) Get(ctx context.Context, path, requestID string, target any) error {
	return c.Do(ctx, http.MethodGet, path, requestID, nil, target)
}

func (c *Client) Do(ctx context.Context, method, path, requestID string, body any, target any) error {
	var reader io.Reader
	contentType := ""
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return apperr.Wrap(apperr.ExitArguments, "请求参数无法编码", err)
		}
		reader = bytes.NewReader(encoded)
		contentType = "application/json"
	}
	return c.DoContent(ctx, method, path, requestID, contentType, nil, reader, target)
}

func (c *Client) DoContent(ctx context.Context,
	method, path, requestID, contentType string,
	headers map[string]string,
	body io.Reader,
	target any,
) error {
	if strings.TrimSpace(c.token) == "" {
		return apperr.New(apperr.ExitAuthentication, "尚未配置 API Token，请运行 emoera auth set-token")
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+"/api/v1/agent"+path, body)
	if err != nil {
		return apperr.Wrap(apperr.ExitConfiguration, "无法创建请求", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if requestID != "" {
		req.Header.Set("X-Request-Id", requestID)
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}

	response, err := c.httpClient.Do(req)
	if err != nil {
		var netError net.Error
		if errors.As(err, &netError) && netError.Timeout() {
			return apperr.Wrap(apperr.ExitTimeout, "请求超时", err)
		}
		return apperr.Wrap(apperr.ExitNetwork, "无法连接 Emoera 服务", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNoContent && target == nil {
		return nil
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return apperr.Wrap(apperr.ExitNetwork, "读取服务响应失败", err)
	}
	var envelope Envelope
	if err := json.Unmarshal(content, &envelope); err != nil {
		return apperr.New(apperr.ExitServer, fmt.Sprintf("服务返回了无效响应（HTTP %d）", response.StatusCode))
	}
	if !envelope.OK || response.StatusCode >= 400 {
		return mapAPIError(response.StatusCode, envelope.Error)
	}
	if target != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, target); err != nil {
			return apperr.Wrap(apperr.ExitServer, "服务响应结构不兼容", err)
		}
	}
	return nil
}

func mapAPIError(status int, apiError *APIError) error {
	message := http.StatusText(status)
	if apiError != nil && strings.TrimSpace(apiError.Message) != "" {
		message = apiError.Message
	}
	exitCode := apperr.ExitServer
	switch status {
	case http.StatusBadRequest:
		exitCode = apperr.ExitArguments
	case http.StatusUnauthorized:
		exitCode = apperr.ExitAuthentication
	case http.StatusForbidden:
		exitCode = apperr.ExitPermission
	case http.StatusNotFound:
		exitCode = apperr.ExitNotFound
	case http.StatusConflict:
		exitCode = apperr.ExitConflict
	case http.StatusTooManyRequests:
		exitCode = apperr.ExitRateLimited
		if apiError != nil && strings.Contains(apiError.Code, "QUOTA") {
			exitCode = apperr.ExitQuota
		}
	case http.StatusGatewayTimeout:
		exitCode = apperr.ExitTimeout
	}
	if apiError != nil && apiError.Code != "" {
		message = apiError.Code + ": " + message
	}
	return apperr.New(exitCode, message)
}

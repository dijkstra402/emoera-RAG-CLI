package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type Model struct {
	Name          string  `json:"name"`
	DisplayName   string  `json:"displayName"`
	Multiplier    float64 `json:"multiplier"`
	Available     bool    `json:"available"`
	ContextWindow *int64  `json:"contextWindow"`
	Provider      string  `json:"provider"`
	Default       bool    `json:"default"`
}

type UsageLimit struct {
	Used      int64   `json:"used"`
	Limit     int64   `json:"limit"`
	Remaining int64   `json:"remaining"`
	ResetsAt  *string `json:"resetsAt"`
}

type Quota struct {
	DailyChat      UsageLimit `json:"dailyChat"`
	MinuteRequests UsageLimit `json:"minuteRequests"`
	Storage        UsageLimit `json:"storage"`
}

type FunctionalStatus struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	LatencyMS *int64 `json:"latencyMs"`
}

type Status struct {
	Overall   string             `json:"overall"`
	Modules   []FunctionalStatus `json:"modules"`
	CheckedAt string             `json:"checkedAt"`
}

type ChatSession struct {
	UUID             string  `json:"uuid"`
	Title            string  `json:"title"`
	ModelName        *string `json:"modelName"`
	PromptTemplateID *int64  `json:"promptTemplateId"`
	Pinned           bool    `json:"pinned"`
	Starred          bool    `json:"starred"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
}

type ChatMessage struct {
	ID         string           `json:"id"`
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	CreatedAt  string           `json:"createdAt"`
	References []map[string]any `json:"references"`
}

type ChatSessionPage struct {
	Items      []ChatSession `json:"items"`
	NextCursor *string       `json:"nextCursor"`
	HasMore    bool          `json:"hasMore"`
}

type ChatMessagePage struct {
	Items      []ChatMessage `json:"items"`
	NextCursor *string       `json:"nextCursor"`
	HasMore    bool          `json:"hasMore"`
}

type ChatRunStatus struct {
	RequestID    string  `json:"requestId"`
	Status       string  `json:"status"`
	Stage        string  `json:"stage"`
	SessionUUID  *string `json:"sessionUuid"`
	LastSequence int64   `json:"lastSequence"`
	FinishReason *string `json:"finishReason"`
	ErrorCode    *string `json:"errorCode"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
	CompletedAt  *string `json:"completedAt"`
}

func (c *Client) ListModels(ctx context.Context, requestID string) ([]Model, error) {
	var response struct {
		Items []Model `json:"items"`
	}
	err := c.Get(ctx, "/models", requestID, &response)
	if response.Items == nil {
		response.Items = []Model{}
	}
	return response.Items, err
}

func (c *Client) GetQuota(ctx context.Context, requestID string) (Quota, error) {
	var result Quota
	err := c.Get(ctx, "/quota", requestID, &result)
	return result, err
}

func (c *Client) GetStatus(ctx context.Context, requestID string) (Status, error) {
	var result Status
	err := c.Get(ctx, "/status", requestID, &result)
	if result.Modules == nil {
		result.Modules = []FunctionalStatus{}
	}
	return result, err
}

func (c *Client) ListChatSessions(ctx context.Context, requestID, cursor string, limit int) (ChatSessionPage, error) {
	query := url.Values{}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	path := "/chat/sessions"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var result ChatSessionPage
	err := c.Get(ctx, path, requestID, &result)
	if result.Items == nil {
		result.Items = []ChatSession{}
	}
	return result, err
}

func (c *Client) ListChatMessages(ctx context.Context, requestID, sessionUUID, cursor string, limit int) (ChatMessagePage, error) {
	query := url.Values{}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	path := "/chat/sessions/" + url.PathEscape(sessionUUID) + "/messages"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var result ChatMessagePage
	err := c.Get(ctx, path, requestID, &result)
	if result.Items == nil {
		result.Items = []ChatMessage{}
	}
	for index := range result.Items {
		if result.Items[index].References == nil {
			result.Items[index].References = []map[string]any{}
		}
	}
	return result, err
}

func (c *Client) GetChatRun(ctx context.Context, requestID, runID string) (ChatRunStatus, error) {
	var result ChatRunStatus
	err := c.Get(ctx, "/chat/runs/"+url.PathEscape(runID), requestID, &result)
	return result, err
}

func (c *Client) CancelChatRun(ctx context.Context, requestID, runID string) (CancelChatResult, error) {
	var result CancelChatResult
	err := c.Do(ctx, http.MethodPost, "/chat/runs/"+url.PathEscape(runID)+"/cancel", requestID, nil, &result)
	return result, err
}

package api

import (
	"context"
	"net/url"
	"strconv"
)

type UsageOverview struct {
	Days              int     `json:"days"`
	TotalCalls        int64   `json:"totalCalls"`
	SuccessCalls      int64   `json:"successCalls"`
	ErrorCalls        int64   `json:"errorCalls"`
	ErrorRate         float64 `json:"errorRate"`
	AverageDurationMS float64 `json:"averageDurationMs"`
	InputTokens       int64   `json:"inputTokens"`
	OutputTokens      int64   `json:"outputTokens"`
	TotalTokens       int64   `json:"totalTokens"`
	ActiveTokens      int64   `json:"activeTokens"`
}

type UsageBucket struct {
	Date         string `json:"date,omitempty"`
	Model        string `json:"model,omitempty"`
	Channel      string `json:"channel,omitempty"`
	Calls        int64  `json:"calls"`
	InputTokens  int64  `json:"inputTokens"`
	OutputTokens int64  `json:"outputTokens"`
	TotalTokens  int64  `json:"totalTokens"`
}

type UsageTokens struct {
	Days     int           `json:"days"`
	Trend    []UsageBucket `json:"trend"`
	Models   []UsageBucket `json:"models"`
	Channels []UsageBucket `json:"channels"`
}

type HeatmapCell struct {
	Date  string `json:"date"`
	Calls int64  `json:"calls"`
}

type UsageHeatmap struct {
	Days     int           `json:"days"`
	MaxCalls int64         `json:"maxCalls"`
	Cells    []HeatmapCell `json:"cells"`
}

type UsageCall struct {
	ID            int64   `json:"id"`
	Channel       string  `json:"channel"`
	APITokenID    *int64  `json:"apiTokenId"`
	TokenPrefix   *string `json:"tokenPrefix"`
	RequestID     *string `json:"requestId"`
	Endpoint      *string `json:"endpoint"`
	Method        *string `json:"method"`
	Model         *string `json:"model"`
	Status        int     `json:"status"`
	Success       bool    `json:"success"`
	DurationMS    int64   `json:"durationMs"`
	InputTokens   *int    `json:"inputTokens"`
	OutputTokens  *int    `json:"outputTokens"`
	TotalTokens   *int    `json:"totalTokens"`
	QuotaUnits    *int    `json:"quotaUnits"`
	ErrorCode     *string `json:"errorCode"`
	ClientVersion *string `json:"clientVersion"`
	CreatedAt     string  `json:"createdAt"`
}

type UsageCallPage struct {
	Content       []UsageCall `json:"content"`
	Page          int         `json:"page"`
	Size          int         `json:"size"`
	TotalElements int64       `json:"totalElements"`
	TotalPages    int         `json:"totalPages"`
}

type UsageCallOptions struct {
	Days     int
	Page     int
	Size     int
	Endpoint string
	Success  *bool
	Model    string
	Channel  string
}

func (c *Client) UsageOverview(ctx context.Context, requestID string, days int) (UsageOverview, error) {
	var result UsageOverview
	err := c.Get(ctx, usagePath("/usage/overview", days), requestID, &result)
	return result, err
}

func (c *Client) UsageTokens(ctx context.Context, requestID string, days int) (UsageTokens, error) {
	var result UsageTokens
	err := c.Get(ctx, usagePath("/usage/tokens", days), requestID, &result)
	return result, err
}

func (c *Client) UsageHeatmap(ctx context.Context, requestID string, days int) (UsageHeatmap, error) {
	var result UsageHeatmap
	err := c.Get(ctx, usagePath("/usage/heatmap", days), requestID, &result)
	return result, err
}

func (c *Client) UsageCalls(ctx context.Context, requestID string, options UsageCallOptions) (UsageCallPage, error) {
	query := url.Values{}
	query.Set("days", strconv.Itoa(options.Days))
	query.Set("page", strconv.Itoa(options.Page))
	query.Set("size", strconv.Itoa(options.Size))
	if options.Endpoint != "" {
		query.Set("endpoint", options.Endpoint)
	}
	if options.Success != nil {
		query.Set("success", strconv.FormatBool(*options.Success))
	}
	if options.Model != "" {
		query.Set("model", options.Model)
	}
	if options.Channel != "" {
		query.Set("channel", options.Channel)
	}
	var result UsageCallPage
	err := c.Get(ctx, "/usage/calls?"+query.Encode(), requestID, &result)
	if result.Content == nil {
		result.Content = []UsageCall{}
	}
	return result, err
}

func usagePath(path string, days int) string {
	query := url.Values{}
	query.Set("days", strconv.Itoa(days))
	return path + "?" + query.Encode()
}

package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type PromptTemplate struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Content         string `json:"content"`
	Scope           string `json:"scope"`
	Enabled         bool   `json:"enabled"`
	DefaultTemplate bool   `json:"defaultTemplate"`
	Owner           string `json:"owner"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

type PromptTemplateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Scope       string `json:"scope"`
	Enabled     bool   `json:"enabled"`
}

func (c *Client) ListPrompts(ctx context.Context, requestID string) ([]PromptTemplate, error) {
	var response struct {
		Items []PromptTemplate `json:"items"`
	}
	err := c.Get(ctx, "/prompts", requestID, &response)
	if response.Items == nil {
		response.Items = []PromptTemplate{}
	}
	return response.Items, err
}

func (c *Client) GetPrompt(ctx context.Context, requestID string, id int64) (PromptTemplate, error) {
	var result PromptTemplate
	err := c.Get(ctx, "/prompts/"+url.PathEscape(formatInt64(id)), requestID, &result)
	return result, err
}

func (c *Client) CreatePrompt(ctx context.Context, requestID string, body PromptTemplateRequest) (PromptTemplate, error) {
	var result PromptTemplate
	err := c.Do(ctx, http.MethodPost, "/prompts", requestID, body, &result)
	return result, err
}

func (c *Client) UpdatePrompt(ctx context.Context, requestID string, id int64, body PromptTemplateRequest) (PromptTemplate, error) {
	var result PromptTemplate
	err := c.Do(ctx, http.MethodPut, "/prompts/"+url.PathEscape(formatInt64(id)), requestID, body, &result)
	return result, err
}

func (c *Client) DeletePrompt(ctx context.Context, requestID string, id int64) error {
	return c.DoContent(ctx, http.MethodDelete, "/prompts/"+url.PathEscape(formatInt64(id)), requestID, "", nil, nil, nil)
}

func (c *Client) SetDefaultPrompt(ctx context.Context, requestID string, id int64) (PromptTemplate, error) {
	var result PromptTemplate
	err := c.Do(ctx, http.MethodPut, "/prompts/"+url.PathEscape(formatInt64(id))+"/default", requestID, nil, &result)
	return result, err
}

func formatInt64(value int64) string {
	return strconv.FormatInt(value, 10)
}

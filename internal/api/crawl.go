package api

import (
	"context"
	"net/http"
)

type CrawlRequest struct {
	URL      string `json:"url"`
	OrgTag   string `json:"orgTag,omitempty"`
	IsPublic bool   `json:"isPublic"`
}

type CrawlResult struct {
	FileMD5    string `json:"fileMd5"`
	Title      string `json:"title"`
	TextLength int    `json:"textLength"`
	Status     string `json:"status"`
}

func (c *Client) Crawl(ctx context.Context, requestID string, body CrawlRequest) (CrawlResult, error) {
	var result CrawlResult
	err := c.Do(ctx, http.MethodPost, "/crawl", requestID, body, &result)
	return result, err
}

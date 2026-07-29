package api

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type OrgTag struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ParentID *string `json:"parentId"`
}

type Document struct {
	FileMD5         string   `json:"fileMd5"`
	FileName        string   `json:"fileName"`
	FileType        string   `json:"fileType"`
	FileSize        int64    `json:"fileSize"`
	Status          string   `json:"status"`
	Visibility      string   `json:"visibility"`
	OrgTag          string   `json:"orgTag"`
	Description     *string  `json:"description"`
	Keywords        []string `json:"keywords"`
	ProcessingError *string  `json:"processingError"`
	CreatedAt       string   `json:"createdAt"`
	UpdatedAt       string   `json:"updatedAt"`
}

type DocumentPage struct {
	Items      []Document `json:"items"`
	NextCursor *string    `json:"nextCursor"`
	HasMore    bool       `json:"hasMore"`
}

type DocumentListOptions struct {
	Cursor     string
	Limit      int
	Query      string
	OrgTag     string
	Visibility string
	Status     string
}

type DocumentUpdateRequest struct {
	OrgTag      *string   `json:"orgTag,omitempty"`
	Public      *bool     `json:"isPublic,omitempty"`
	Description *string   `json:"description,omitempty"`
	Keywords    *[]string `json:"keywords,omitempty"`
}

type UploadStatus struct {
	FileMD5        string  `json:"fileMd5"`
	FileName       string  `json:"fileName"`
	TotalSize      int64   `json:"totalSize"`
	TotalChunks    int     `json:"totalChunks"`
	UploadedChunks []int   `json:"uploadedChunks"`
	MissingChunks  []int   `json:"missingChunks"`
	Progress       float64 `json:"progress"`
	Completed      bool    `json:"completed"`
}

type Capabilities struct {
	APIVersion        string           `json:"apiVersion"`
	CLIMinimumVersion string           `json:"cliMinimumVersion"`
	AcceptedFileTypes []string         `json:"acceptedFileTypes"`
	Limits            map[string]int64 `json:"limits"`
	Features          map[string]bool  `json:"features"`
}

func (c *Client) ListOrgTags(ctx context.Context, requestID string) ([]OrgTag, error) {
	var response struct {
		Items []OrgTag `json:"items"`
	}
	err := c.Get(ctx, "/org-tags", requestID, &response)
	return response.Items, err
}

func (c *Client) Capabilities(ctx context.Context, requestID string) (Capabilities, error) {
	var result Capabilities
	err := c.Get(ctx, "/capabilities", requestID, &result)
	return result, err
}

func (c *Client) ListDocuments(ctx context.Context, requestID string, options DocumentListOptions) (DocumentPage, error) {
	query := url.Values{}
	if options.Cursor != "" {
		query.Set("cursor", options.Cursor)
	}
	if options.Limit > 0 {
		query.Set("limit", strconv.Itoa(options.Limit))
	}
	if options.Query != "" {
		query.Set("query", options.Query)
	}
	if options.OrgTag != "" {
		query.Set("orgTag", options.OrgTag)
	}
	if options.Visibility != "" {
		query.Set("visibility", options.Visibility)
	}
	if options.Status != "" {
		query.Set("status", options.Status)
	}
	path := "/documents"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var result DocumentPage
	err := c.Get(ctx, path, requestID, &result)
	return result, err
}

func (c *Client) GetDocument(ctx context.Context, requestID, fileMD5 string) (Document, error) {
	var result Document
	err := c.Get(ctx, "/documents/"+url.PathEscape(fileMD5), requestID, &result)
	return result, err
}

func (c *Client) UpdateDocument(ctx context.Context, requestID, fileMD5 string, body DocumentUpdateRequest) (Document, error) {
	var result Document
	err := c.Do(ctx, http.MethodPatch, "/documents/"+url.PathEscape(fileMD5), requestID, body, &result)
	return result, err
}

func (c *Client) DeleteDocument(ctx context.Context, requestID, fileMD5, idempotencyKey string) error {
	return c.DoContent(
		ctx,
		http.MethodDelete,
		"/documents/"+url.PathEscape(fileMD5),
		requestID,
		"",
		map[string]string{"Idempotency-Key": idempotencyKey},
		nil,
		nil,
	)
}

func (c *Client) GetUploadStatus(ctx context.Context, requestID, fileMD5 string) (UploadStatus, error) {
	var result UploadStatus
	err := c.Get(ctx, "/documents/uploads/"+url.PathEscape(fileMD5), requestID, &result)
	return result, err
}

func (c *Client) UploadMultipart(ctx context.Context, method, path, requestID, contentType, idempotencyKey string, bodyReader io.Reader, target any) error {
	headers := map[string]string{}
	if idempotencyKey != "" {
		headers["Idempotency-Key"] = idempotencyKey
	}
	return c.DoContent(ctx, method, path, requestID, contentType, headers, bodyReader, target)
}

func (c *Client) CompleteUpload(ctx context.Context, requestID, fileMD5, idempotencyKey string) (Document, error) {
	var result Document
	err := c.DoContent(
		ctx, http.MethodPost, "/documents/uploads/"+url.PathEscape(fileMD5)+"/complete",
		requestID, "", map[string]string{"Idempotency-Key": idempotencyKey}, nil, &result,
	)
	return result, err
}

func (c *Client) WaitDocument(ctx context.Context, requestID, fileMD5 string, interval time.Duration) (Document, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		document, err := c.GetDocument(ctx, requestID, fileMD5)
		if err != nil {
			return Document{}, err
		}
		if document.Status == "ready" || document.Status == "failed" {
			return document, nil
		}
		select {
		case <-ctx.Done():
			return Document{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

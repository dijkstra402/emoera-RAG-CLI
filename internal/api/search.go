package api

import (
	"context"
	"net/http"
)

type SearchOptions struct {
	TopK           int
	OrgTags        []string
	TargetFileMD5s []string
	MinScore       float64
	IncludeContent bool
	IncludeExplain bool
}

type SearchRequest struct {
	Query          string   `json:"query"`
	TopK           int      `json:"topK"`
	OrgTags        []string `json:"orgTags"`
	TargetFileMD5s []string `json:"targetFileMd5s"`
	MinScore       float64  `json:"minScore"`
	IncludeContent bool     `json:"includeContent"`
	IncludeExplain bool     `json:"includeExplain"`
}

type SearchExplain struct {
	Retrieval      []string `json:"retrieval"`
	KeywordMatched bool     `json:"keywordMatched"`
	RerankApplied  bool     `json:"rerankApplied"`
}

type SearchHit struct {
	Rank       int            `json:"rank"`
	FileMD5    string         `json:"fileMd5"`
	FileName   string         `json:"fileName"`
	ChunkID    *int           `json:"chunkId"`
	Content    *string        `json:"content"`
	Score      float64        `json:"score"`
	Reranked   bool           `json:"reranked"`
	OrgTag     string         `json:"orgTag"`
	Visibility string         `json:"visibility"`
	Explain    *SearchExplain `json:"explain"`
}

type SearchResult struct {
	Query          string      `json:"query"`
	RewrittenQuery *string     `json:"rewrittenQuery"`
	TookMS         int64       `json:"tookMs"`
	Items          []SearchHit `json:"items"`
}

func (c *Client) Search(ctx context.Context, requestID, query string, options SearchOptions) (SearchResult, error) {
	request := SearchRequest{
		Query: query, TopK: options.TopK,
		OrgTags:        append([]string(nil), options.OrgTags...),
		TargetFileMD5s: append([]string(nil), options.TargetFileMD5s...),
		MinScore:       options.MinScore, IncludeContent: options.IncludeContent,
		IncludeExplain: options.IncludeExplain,
	}
	if request.OrgTags == nil {
		request.OrgTags = []string{}
	}
	if request.TargetFileMD5s == nil {
		request.TargetFileMD5s = []string{}
	}
	var result SearchResult
	err := c.Do(ctx, http.MethodPost, "/search", requestID, request, &result)
	if result.Items == nil {
		result.Items = []SearchHit{}
	}
	return result, err
}

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestSearchSendsStableRequestAndDecodesResult(t *testing.T) {
	client, err := New("https://example.test", "em_sk_test", "test", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost || request.URL.Path != "/api/v1/agent/search" {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		var body SearchRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Query != "支付回调" || body.TopK != 8 || body.MinScore != 0.35 {
			t.Fatalf("unexpected search request: %#v", body)
		}
		if !reflect.DeepEqual(body.OrgTags, []string{"engineering"}) {
			t.Fatalf("unexpected org tags: %#v", body.OrgTags)
		}
		if !body.IncludeContent || !body.IncludeExplain {
			t.Fatalf("expected content and explain to be enabled: %#v", body)
		}
		return jsonResponse(http.StatusOK, `{
			"ok":true,
			"data":{"query":"支付回调","rewrittenQuery":null,"tookMs":18,"items":[{
				"rank":1,"fileMd5":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","fileName":"支付规范.pdf",
				"chunkId":7,"content":"必须先验签","score":0.91,"reranked":true,
				"orgTag":"engineering","visibility":"private",
				"explain":{"retrieval":["vector","bm25","rerank"],"keywordMatched":false,"rerankApplied":true}
			}]},
			"meta":{"requestId":"req_search","apiVersion":"v1","timestamp":"2026-07-23T00:00:00Z"}
		}`), nil
	})

	result, err := client.Search(context.Background(), "req_search", "支付回调", SearchOptions{
		TopK: 8, OrgTags: []string{"engineering"}, MinScore: 0.35,
		IncludeContent: true, IncludeExplain: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].Explain == nil {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Items[0].FileName != "支付规范.pdf" || result.Items[0].Score != 0.91 {
		t.Fatalf("unexpected hit: %#v", result.Items[0])
	}
}

func TestSearchNormalizesMissingCollections(t *testing.T) {
	client, err := New("https://example.test", "em_sk_test", "test", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["orgTags"] == nil || body["targetFileMd5s"] == nil {
			t.Fatalf("collections must be encoded as arrays: %#v", body)
		}
		return jsonResponse(http.StatusOK, `{"ok":true,"data":{"query":"问题","rewrittenQuery":null,"tookMs":1,"items":null},"meta":{"requestId":"req","apiVersion":"v1","timestamp":"2026-07-23T00:00:00Z"}}`), nil
	})

	result, err := client.Search(context.Background(), "", "问题", SearchOptions{TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	if result.Items == nil || len(result.Items) != 0 {
		t.Fatalf("empty items must be a non-nil slice: %#v", result.Items)
	}
}

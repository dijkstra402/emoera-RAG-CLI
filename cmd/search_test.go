package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/api"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
)

func TestSearchCommandJSONContract(t *testing.T) {
	server := searchTestServer(t, func(request api.SearchRequest) {
		if request.Query != "支付回调" || request.TopK != 8 || request.MinScore != 0.35 {
			t.Fatalf("unexpected request: %#v", request)
		}
		if len(request.OrgTags) != 1 || request.OrgTags[0] != "engineering" {
			t.Fatalf("unexpected org tags: %#v", request.OrgTags)
		}
		if len(request.TargetFileMD5s) != 1 || request.TargetFileMD5s[0] != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
			t.Fatalf("unexpected target files: %#v", request.TargetFileMD5s)
		}
		if !request.IncludeExplain || !request.IncludeContent {
			t.Fatalf("expected explain and content: %#v", request)
		}
	})
	defer server.Close()

	stdout, _, err := executeSearchCommand(t, server.URL,
		"search", "支付回调", "--top-k", "8", "--min-score", "0.35",
		"--org-tag", "engineering", "--org-tag", "engineering",
		"--file", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "--explain", "--json")
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "query": "支付回调",
  "rewrittenQuery": null,
  "tookMs": 12,
  "items": [
    {
      "rank": 1,
      "fileMd5": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "fileName": "支付规范.pdf",
      "chunkId": 7,
      "content": "支付回调必须先验签，\n再做幂等检查。",
      "score": 0.91,
      "reranked": true,
      "orgTag": "engineering",
      "visibility": "private",
      "explain": {
        "retrieval": [
          "vector",
          "bm25",
          "rerank"
        ],
        "keywordMatched": true,
        "rerankApplied": true
      }
    }
  ]
}
`
	if stdout != want {
		t.Fatalf("JSON contract changed\nwant:\n%s\ngot:\n%s", want, stdout)
	}
}

func TestSearchCommandTableIsSingleLineAndExplained(t *testing.T) {
	server := searchTestServer(t, nil)
	defer server.Close()

	stdout, stderr, err := executeSearchCommand(t, server.URL, "search", "支付回调", "--explain")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "RANK\tSCORE\tFILE\tCHUNK\tORG\tVISIBILITY\tRETRIEVAL\tCONTENT") {
		t.Fatalf("missing table headings: %s", stdout)
	}
	if !strings.Contains(stdout, "vector+bm25+rerank") || strings.Contains(stdout, "\n再做幂等") {
		t.Fatalf("unexpected table output: %s", stdout)
	}
	if !strings.Contains(stderr, "检索完成：1 条，耗时 12 ms") {
		t.Fatalf("missing summary: %s", stderr)
	}
}

func TestSearchCommandRejectsInvalidOptionsBeforeNetwork(t *testing.T) {
	_, _, err := executeSearchCommand(t, "http://127.0.0.1:1", "search", "问题", "--top-k", "51")
	if apperr.ExitCode(err) != apperr.ExitArguments {
		t.Fatalf("expected arguments exit code, got %d: %v", apperr.ExitCode(err), err)
	}
	_, _, err = executeSearchCommand(t, "http://127.0.0.1:1", "search", "问题", "--file", "not-md5")
	if apperr.ExitCode(err) != apperr.ExitArguments {
		t.Fatalf("expected arguments exit code, got %d: %v", apperr.ExitCode(err), err)
	}
}

func TestSearchCommandMapsScopeFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusForbidden)
		_, _ = writer.Write([]byte(`{"ok":false,"error":{"code":"INSUFFICIENT_SCOPE","message":"缺少 search:read 权限","retryable":false,"details":{}},"meta":{"requestId":"req","apiVersion":"v1","timestamp":"2026-07-23T00:00:00Z"}}`))
	}))
	defer server.Close()

	_, _, err := executeSearchCommand(t, server.URL, "search", "问题")
	if apperr.ExitCode(err) != apperr.ExitPermission {
		t.Fatalf("expected permission exit code, got %d: %v", apperr.ExitCode(err), err)
	}
}

func searchTestServer(t *testing.T, inspect func(api.SearchRequest)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/agent/search" || request.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer em_sk_test" {
			t.Fatalf("missing token")
		}
		var body api.SearchRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if inspect != nil {
			inspect(body)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
			"ok":true,
			"data":{"query":"支付回调","rewrittenQuery":null,"tookMs":12,"items":[{
				"rank":1,"fileMd5":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","fileName":"支付规范.pdf",
				"chunkId":7,"content":"支付回调必须先验签，\n再做幂等检查。","score":0.91,"reranked":true,
				"orgTag":"engineering","visibility":"private",
				"explain":{"retrieval":["vector","bm25","rerank"],"keywordMatched":true,"rerankApplied":true}
			}]},
			"meta":{"requestId":"req_search","apiVersion":"v1","timestamp":"2026-07-23T00:00:00Z"}
		}`))
	}))
}

func executeSearchCommand(t *testing.T, endpoint string, args ...string) (string, string, error) {
	t.Helper()
	t.Setenv("EMOERA_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	command := newRootCommand(memoryStore{})
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs(append([]string{"--endpoint", endpoint, "--token", "em_sk_test"}, args...))
	err := command.Execute()
	return stdout.String(), stderr.String(), err
}

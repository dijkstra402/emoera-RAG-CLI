package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
)

func TestCompleteChatUsesJSONContractAndIdempotencyKey(t *testing.T) {
	client, err := New("https://example.test", "em_sk_test", "test", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/api/v1/agent/chat/completions" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.Header.Get("Idempotency-Key") != "idem-12345678" || request.Header.Get("Accept") != "application/json" {
			t.Fatalf("missing chat headers: %#v", request.Header)
		}
		var body ChatRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Stream || body.Input != "总结文档" || len(body.OrgTags) != 1 {
			t.Fatalf("unexpected request: %#v", body)
		}
		return jsonResponse(http.StatusOK, `{"ok":true,"data":{"requestId":"chat_1","sessionUuid":"session-1","content":"答案","finishReason":"stop","references":null,"usage":null,"suggestions":null},"meta":{"requestId":"http_1","apiVersion":"v1","timestamp":"2026-07-23T00:00:00Z"}}`), nil
	})

	result, err := client.CompleteChat(context.Background(), "http_1", "idem-12345678", ChatRequest{
		Input: "总结文档", OrgTags: []string{"engineering"}, Stream: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "答案" || result.References == nil || result.Usage == nil || result.Suggestions == nil {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestDecodeSSEHandlesMultilineDataAndNamedEvents(t *testing.T) {
	stream := strings.Join([]string{
		": keepalive",
		"event: meta",
		"data: {\"type\":\"meta\",\"requestId\":\"chat_1\",",
		"data: \"sequence\":1,\"timestamp\":\"2026-07-23T00:00:00Z\",\"data\":{\"sessionUuid\":\"s1\"}}",
		"",
		"event: done",
		"data: {\"type\":\"done\",\"requestId\":\"chat_1\",\"sequence\":2,\"timestamp\":\"2026-07-23T00:00:01Z\",\"data\":{\"finishReason\":\"stop\"}}",
		"",
	}, "\n")
	var events []ChatEvent
	if err := decodeSSE(bytes.NewBufferString(stream), func(event ChatEvent) error {
		events = append(events, event)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Type != "meta" || events[1].Type != "done" {
		t.Fatalf("unexpected events: %#v", events)
	}
}

func TestDecodeSSEMapsStreamErrorAndUnexpectedEOF(t *testing.T) {
	errorStream := "event: error\ndata: {\"type\":\"error\",\"requestId\":\"chat_1\",\"sequence\":1,\"timestamp\":\"2026-07-23T00:00:00Z\",\"data\":{\"code\":\"QUOTA_EXCEEDED\",\"message\":\"额度不足\"}}\n\n"
	err := decodeSSE(strings.NewReader(errorStream), nil)
	if apperr.ExitCode(err) != apperr.ExitQuota {
		t.Fatalf("expected quota error, got %d: %v", apperr.ExitCode(err), err)
	}

	partial := "event: delta\ndata: {\"type\":\"delta\",\"requestId\":\"chat_1\",\"sequence\":1,\"timestamp\":\"2026-07-23T00:00:00Z\",\"data\":{\"content\":\"半截\"}}\n\n"
	err = decodeSSE(strings.NewReader(partial), nil)
	if apperr.ExitCode(err) != apperr.ExitServer {
		t.Fatalf("expected server error, got %d: %v", apperr.ExitCode(err), err)
	}
}

func TestDecodeSSEAcceptsTransportEOFOnlyAfterDone(t *testing.T) {
	complete := "event: done\ndata: {\"type\":\"done\",\"requestId\":\"chat_1\",\"sequence\":1,\"timestamp\":\"2026-07-23T00:00:00Z\",\"data\":{\"finishReason\":\"stop\"}}\n\n"
	if err := decodeSSE(&unexpectedEOFReader{content: []byte(complete)}, nil); err != nil {
		t.Fatalf("terminal event should win over transport EOF: %v", err)
	}

	partial := "event: delta\ndata: {\"type\":\"delta\",\"requestId\":\"chat_1\",\"sequence\":1,\"timestamp\":\"2026-07-23T00:00:00Z\",\"data\":{\"content\":\"半截\"}}\n\n"
	err := decodeSSE(&unexpectedEOFReader{content: []byte(partial)}, nil)
	if apperr.ExitCode(err) != apperr.ExitNetwork {
		t.Fatalf("non-terminal transport EOF must remain an error, got %d: %v", apperr.ExitCode(err), err)
	}
}

type unexpectedEOFReader struct {
	content []byte
	done    bool
}

func (reader *unexpectedEOFReader) Read(target []byte) (int, error) {
	if !reader.done {
		reader.done = true
		return copy(target, reader.content), nil
	}
	return 0, io.ErrUnexpectedEOF
}

func TestStreamChatRejectsMismatchedSSEEventName(t *testing.T) {
	client, err := New("https://example.test", "em_sk_test", "test", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient.Transport = roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream;charset=UTF-8"}},
			Body:       io.NopCloser(strings.NewReader("event: delta\ndata: {\"type\":\"done\",\"requestId\":\"chat_1\",\"sequence\":1,\"timestamp\":\"2026-07-23T00:00:00Z\",\"data\":{}}\n\n")),
		}, nil
	})
	err = client.StreamChat(context.Background(), "", "idem-12345678", ChatRequest{Input: "问题"}, nil)
	if apperr.ExitCode(err) != apperr.ExitServer {
		t.Fatalf("expected server error, got %d: %v", apperr.ExitCode(err), err)
	}
}

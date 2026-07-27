package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
)

func TestClientAddsAuthenticationAndDecodesEnvelope(t *testing.T) {
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/api/v1/agent/me" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer em_sk_test" {
			t.Fatalf("missing authorization header")
		}
		return jsonResponse(http.StatusOK, `{"ok":true,"data":{"username":"tester"},"meta":{"requestId":"req_test","apiVersion":"v1","timestamp":"2026-07-23T00:00:00Z"}}`), nil
	})

	client, err := New("https://example.test", "em_sk_test", "test", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient.Transport = transport
	var result map[string]any
	if err := client.Get(context.Background(), "/me", "req_test", &result); err != nil {
		t.Fatal(err)
	}
	if result["username"] != "tester" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestClientMapsAuthenticationError(t *testing.T) {
	client, err := New("https://example.test", "em_sk_test", "test", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient.Transport = roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusUnauthorized, `{"ok":false,"error":{"code":"INVALID_TOKEN","message":"Token 无效","retryable":false,"details":{}},"meta":{"requestId":"req_test","apiVersion":"v1","timestamp":"2026-07-23T00:00:00Z"}}`), nil
	})
	err = client.Get(context.Background(), "/me", "", &map[string]any{})
	if apperr.ExitCode(err) != apperr.ExitAuthentication {
		t.Fatalf("unexpected exit code %d for %v", apperr.ExitCode(err), err)
	}
}

func TestClientRejectsInsecureRemoteEndpoint(t *testing.T) {
	_, err := New("http://example.com", "em_sk_test", "test", time.Second)
	if apperr.ExitCode(err) != apperr.ExitConfiguration {
		t.Fatalf("expected configuration error, got %v", err)
	}
	if _, err := New("http://127.0.0.1:8081", "em_sk_test", "test", time.Second); err != nil {
		t.Fatalf("localhost development endpoint should be allowed: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

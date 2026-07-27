package api

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestOperationsUseStablePathsAndNormalizeCollections(t *testing.T) {
	client, err := New("https://example.test", "em_sk_test", "test", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path {
		case "/api/v1/agent/models":
			return jsonResponse(http.StatusOK, `{"ok":true,"data":{"items":null},"meta":{"requestId":"req","apiVersion":"v1","timestamp":"2026-07-24T00:00:00Z"}}`), nil
		case "/api/v1/agent/chat/sessions":
			if request.URL.Query().Get("cursor") != "opaque+cursor" || request.URL.Query().Get("limit") != "25" {
				t.Fatalf("unexpected query: %s", request.URL.RawQuery)
			}
			return jsonResponse(http.StatusOK, `{"ok":true,"data":{"items":null,"nextCursor":null,"hasMore":false},"meta":{"requestId":"req","apiVersion":"v1","timestamp":"2026-07-24T00:00:00Z"}}`), nil
		case "/api/v1/agent/chat/sessions/11111111-1111-4111-8111-111111111111/messages":
			return jsonResponse(http.StatusOK, `{"ok":true,"data":{"items":[{"id":"1","role":"assistant","content":"答案","createdAt":"2026-07-24T10:00:00","references":null}],"nextCursor":null,"hasMore":false},"meta":{"requestId":"req","apiVersion":"v1","timestamp":"2026-07-24T00:00:00Z"}}`), nil
		default:
			t.Fatalf("unexpected path: %s", request.URL.Path)
			return nil, nil
		}
	})

	models, err := client.ListModels(context.Background(), "req")
	if err != nil || models == nil || len(models) != 0 {
		t.Fatalf("models were not normalized: %#v %v", models, err)
	}
	sessions, err := client.ListChatSessions(context.Background(), "req", "opaque+cursor", 25)
	if err != nil || sessions.Items == nil {
		t.Fatalf("sessions were not normalized: %#v %v", sessions, err)
	}
	messages, err := client.ListChatMessages(context.Background(), "req",
		"11111111-1111-4111-8111-111111111111", "", 20)
	if err != nil || len(messages.Items) != 1 || messages.Items[0].References == nil {
		t.Fatalf("messages were not normalized: %#v %v", messages, err)
	}
}

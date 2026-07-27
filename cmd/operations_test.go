package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestOperationsCommandsRenderPagesAndStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/v1/agent/chat/sessions":
			fmt.Fprint(writer, `{"ok":true,"data":{"items":[{"uuid":"11111111-1111-4111-8111-111111111111","title":"支付问题","modelName":"qwen3-8b","createdAt":"2026-07-24T09:00:00","updatedAt":"2026-07-24T10:00:00"}],"nextCursor":"next-1","hasMore":true},"meta":{"requestId":"req","apiVersion":"v1","timestamp":"2026-07-24T00:00:00Z"}}`)
		case "/api/v1/agent/chat/sessions/11111111-1111-4111-8111-111111111111/messages":
			fmt.Fprint(writer, `{"ok":true,"data":{"items":[{"id":"8","role":"assistant","content":"先验证签名","createdAt":"2026-07-24T10:00:00","references":[]}],"nextCursor":null,"hasMore":false},"meta":{"requestId":"req","apiVersion":"v1","timestamp":"2026-07-24T00:00:00Z"}}`)
		case "/api/v1/agent/status":
			fmt.Fprint(writer, `{"ok":true,"data":{"overall":"operational","checkedAt":"2026-07-24T10:00:00+08:00","modules":[{"code":"chat","name":"AI 问答","status":"operational","latencyMs":20}]},"meta":{"requestId":"req","apiVersion":"v1","timestamp":"2026-07-24T00:00:00Z"}}`)
		default:
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
	}))
	defer server.Close()

	sessions, sessionsErr, err := executeOperationCommand(t, server.URL, "chat", "sessions")
	if err != nil || !strings.Contains(sessions, "支付问题") || !strings.Contains(sessionsErr, "--cursor next-1") {
		t.Fatalf("unexpected sessions output: %q %q %v", sessions, sessionsErr, err)
	}
	messages, _, err := executeOperationCommand(t, server.URL, "chat", "messages",
		"11111111-1111-4111-8111-111111111111")
	if err != nil || !strings.Contains(messages, "先验证签名") {
		t.Fatalf("unexpected messages output: %q %v", messages, err)
	}
	status, _, err := executeOperationCommand(t, server.URL, "status", "--quiet")
	if err != nil || strings.TrimSpace(status) != "operational" {
		t.Fatalf("unexpected status output: %q %v", status, err)
	}
}

func TestOperationsCommandsValidateBeforeNetwork(t *testing.T) {
	_, _, err := executeOperationCommand(t, "http://127.0.0.1:1", "chat", "sessions", "--limit", "101")
	if err == nil || !strings.Contains(err.Error(), "--limit") {
		t.Fatalf("expected limit validation error, got %v", err)
	}
	_, _, err = executeOperationCommand(t, "http://127.0.0.1:1", "chat", "messages", "not-a-uuid")
	if err == nil || !strings.Contains(err.Error(), "UUID") {
		t.Fatalf("expected UUID validation error, got %v", err)
	}
}

func executeOperationCommand(t *testing.T, endpoint string, args ...string) (string, string, error) {
	t.Helper()
	t.Setenv("EMOERA_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	command := newRootCommand(memoryStore{})
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs(append([]string{"--endpoint", endpoint, "--token", "em_sk_test"}, args...))
	err := command.Execute()
	return stdout.String(), stderr.String(), err
}

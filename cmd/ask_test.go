package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/api"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
)

func TestAskCommandNonStreamingJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/agent/chat/completions" || request.Header.Get("Idempotency-Key") != "fixed-idempotency" {
			t.Fatalf("unexpected request: %s %#v", request.URL.Path, request.Header)
		}
		var body api.ChatRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Stream || body.Input != "总结支付规范" || body.ModelName == nil || *body.ModelName != "qwen3-8b" {
			t.Fatalf("unexpected body: %#v", body)
		}
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"ok":true,"data":{"requestId":"chat_1","sessionUuid":"11111111-1111-4111-8111-111111111111","content":"支付回调先验签。","finishReason":"stop","references":[],"usage":{"quotaUnits":1},"suggestions":[]},"meta":{"requestId":"http_1","apiVersion":"v1","timestamp":"2026-07-23T00:00:00Z"}}`)
	}))
	defer server.Close()

	stdout, _, err := executeAskCommand(t, context.Background(), server.URL,
		"ask", "总结支付规范", "--stream=false", "--model", "qwen3-8b",
		"--idempotency-key", "fixed-idempotency", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, `"content": "支付回调先验签。"`) || !strings.Contains(stdout, `"quotaUnits": 1`) {
		t.Fatalf("unexpected JSON: %s", stdout)
	}
}

func TestAskCommandStreamingJSONLIsFlattened(t *testing.T) {
	server := askSSEServer(t, nil)
	defer server.Close()
	stdout, _, err := executeAskCommand(t, context.Background(), server.URL,
		"ask", "问题", "--jsonl", "--idempotency-key", "fixed-idempotency")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 JSONL events, got %d: %s", len(lines), stdout)
	}
	if !strings.Contains(lines[0], `"sessionUuid":"11111111-1111-4111-8111-111111111111"`) || strings.Contains(lines[0], `"data"`) {
		t.Fatalf("meta event was not flattened: %s", lines[0])
	}
	if !strings.Contains(lines[2], `"content":"答案"`) || !strings.Contains(lines[3], `"finishReason":"stop"`) {
		t.Fatalf("unexpected events: %s", stdout)
	}
}

func TestAskCommandStreamingHumanOutputUsesReadablePresentation(t *testing.T) {
	server := askSSEServer(t, nil)
	defer server.Close()
	stdout, stderr, err := executeAskCommand(t, context.Background(), server.URL,
		"ask", "问题", "--idempotency-key", "fixed-idempotency")
	if err != nil {
		t.Fatal(err)
	}
	if stdout != "答案\n" {
		t.Fatalf("answer must remain clean on stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "EMOERA  知识库问答") ||
		!strings.Contains(stderr, "组织回答") ||
		!strings.Contains(stderr, "会话  11111111-1111-4111-8111-111111111111") {
		t.Fatalf("unexpected terminal presentation: %s", stderr)
	}
	if strings.Contains(stderr, "\x1b[") {
		t.Fatalf("non-terminal output must not contain ANSI escapes: %q", stderr)
	}
}

func TestAskCommandReadsStdinAndValidatesBeforeNetwork(t *testing.T) {
	command := newRootCommand(memoryStore{})
	command.SetIn(strings.NewReader("来自标准输入的问题\n"))
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"--endpoint", "http://127.0.0.1:1", "--token", "em_sk_test", "ask", "-", "--session", "bad"})
	err := command.Execute()
	if apperr.ExitCode(err) != apperr.ExitArguments {
		t.Fatalf("expected arguments error, got %d: %v", apperr.ExitCode(err), err)
	}

	_, _, err = executeAskCommand(t, context.Background(), "http://127.0.0.1:1",
		"ask", "问题", "--input-file", filepath.Join(t.TempDir(), "question.txt"))
	if apperr.ExitCode(err) != apperr.ExitArguments {
		t.Fatalf("expected conflicting input error, got %d: %v", apperr.ExitCode(err), err)
	}
}

func TestAskCommandCancellationCallsServerCancel(t *testing.T) {
	started := make(chan struct{})
	cancelCalled := make(chan struct{})
	var once sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v1/agent/chat/completions":
			writer.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(writer, "event: meta\ndata: {\"type\":\"meta\",\"requestId\":\"chat_cancel\",\"sequence\":1,\"timestamp\":\"2026-07-23T00:00:00Z\",\"data\":{\"sessionUuid\":\"s1\"}}\n\n")
			writer.(http.Flusher).Flush()
			once.Do(func() { close(started) })
			<-request.Context().Done()
		case "/api/v1/agent/chat/runs/chat_cancel/cancel":
			writer.Header().Set("Content-Type", "application/json")
			fmt.Fprint(writer, `{"ok":true,"data":{"requestId":"chat_cancel","status":"cancelling"},"meta":{"requestId":"http_1","apiVersion":"v1","timestamp":"2026-07-23T00:00:00Z"}}`)
			close(cancelCalled)
		default:
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, _, err := executeAskCommand(t, ctx, server.URL,
			"ask", "问题", "--idempotency-key", "fixed-idempotency")
		result <- err
	}()
	select {
	case <-started:
		// 等待客户端消费 meta，获得服务端生成的 runId 后再模拟 Ctrl+C。
		time.Sleep(100 * time.Millisecond)
		cancel()
	case <-time.After(2 * time.Second):
		t.Fatal("stream did not start")
	}
	select {
	case err := <-result:
		if apperr.ExitCode(err) != apperr.ExitInterrupted {
			t.Fatalf("expected interrupted exit code, got %d: %v", apperr.ExitCode(err), err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("command did not stop")
	}
	select {
	case <-cancelCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("cancel endpoint was not called")
	}
}

func askSSEServer(t *testing.T, inspect func(api.ChatRequest)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/agent/chat/completions" || request.Header.Get("Accept") != "text/event-stream" {
			t.Fatalf("unexpected request: %s %#v", request.URL.Path, request.Header)
		}
		var body api.ChatRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if inspect != nil {
			inspect(body)
		}
		writer.Header().Set("Content-Type", "text/event-stream;charset=UTF-8")
		for _, event := range []string{
			"event: meta\ndata: {\"type\":\"meta\",\"requestId\":\"chat_1\",\"sequence\":1,\"timestamp\":\"2026-07-23T00:00:00Z\",\"data\":{\"sessionUuid\":\"11111111-1111-4111-8111-111111111111\",\"model\":\"qwen3-8b\"}}\n\n",
			"event: stage\ndata: {\"type\":\"stage\",\"requestId\":\"chat_1\",\"sequence\":2,\"timestamp\":\"2026-07-23T00:00:01Z\",\"data\":{\"stage\":\"generating\"}}\n\n",
			"event: delta\ndata: {\"type\":\"delta\",\"requestId\":\"chat_1\",\"sequence\":3,\"timestamp\":\"2026-07-23T00:00:02Z\",\"data\":{\"content\":\"答案\"}}\n\n",
			"event: done\ndata: {\"type\":\"done\",\"requestId\":\"chat_1\",\"sequence\":4,\"timestamp\":\"2026-07-23T00:00:03Z\",\"data\":{\"finishReason\":\"stop\"}}\n\n",
		} {
			fmt.Fprint(writer, event)
		}
	}))
}

func executeAskCommand(t *testing.T, ctx context.Context, endpoint string, args ...string) (string, string, error) {
	t.Helper()
	t.Setenv("EMOERA_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	command := newRootCommand(memoryStore{})
	command.SetContext(ctx)
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs(append([]string{"--endpoint", endpoint, "--token", "em_sk_test"}, args...))
	err := command.Execute()
	return stdout.String(), stderr.String(), err
}

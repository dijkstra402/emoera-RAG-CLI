package upload

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/api"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
)

func TestDirectUploadUsesSingleMultipartRequest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "中文资料.md")
	if err := os.WriteFile(path, []byte("知识库内容"), 0o600); err != nil {
		t.Fatal(err)
	}
	client := &fakeClient{directLimit: DefaultDirectMaxSize}

	document, err := New(client).Upload(context.Background(), path, Options{OrgTag: "engineering"})
	if err != nil {
		t.Fatal(err)
	}
	if document.FileMD5 == "" || client.multipartCalls != 1 {
		t.Fatalf("unexpected upload result: %#v, calls=%d", document, client.multipartCalls)
	}
	if client.paths[0] != "/documents" || !strings.Contains(client.bodies[0], "中文资料.md") {
		t.Fatalf("unexpected multipart request: %s", client.paths[0])
	}
}

func TestDirectUploadRetriesWithReusableMultipartBody(t *testing.T) {
	path := filepath.Join(t.TempDir(), "retry-direct.md")
	if err := os.WriteFile(path, []byte("repeatable body"), 0o600); err != nil {
		t.Fatal(err)
	}
	client := &fakeClient{directLimit: DefaultDirectMaxSize, failUploads: 1}

	document, err := New(client).Upload(context.Background(), path, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if document.FileMD5 == "" || client.multipartCalls != 2 {
		t.Fatalf("expected one retry before success, result=%#v calls=%d", document, client.multipartCalls)
	}
	if !strings.Contains(client.bodies[0], "repeatable body") || !strings.Contains(client.bodies[1], "repeatable body") {
		t.Fatalf("multipart body must be rebuilt for every retry")
	}
}

func TestResumableUploadSendsOnlyMissingChunksAndCompletes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.txt")
	content := make([]byte, ChunkSizeBytes+17)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	client := &fakeClient{
		directLimit:  1,
		uploadStatus: api.UploadStatus{TotalChunks: 2, UploadedChunks: []int{0}, MissingChunks: []int{1}},
	}

	document, err := New(client).Upload(context.Background(), path, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if document.FileMD5 == "" || client.multipartCalls != 1 || client.completeCalls != 1 {
		t.Fatalf("unexpected resumable result: %#v calls=%d complete=%d", document, client.multipartCalls, client.completeCalls)
	}
	if !strings.Contains(client.paths[0], "/chunks/1") {
		t.Fatalf("expected only missing chunk 1, got %v", client.paths)
	}
}

func TestResumableUploadRetriesTransientChunkFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "retry.txt")
	if err := os.WriteFile(path, []byte("retry me"), 0o600); err != nil {
		t.Fatal(err)
	}
	client := &fakeClient{
		directLimit: 1,
		statusError: apperr.New(apperr.ExitNotFound, "not initialized"),
		failUploads: 2,
	}

	if _, err := New(client).Upload(context.Background(), path, Options{}); err != nil {
		t.Fatal(err)
	}
	if client.multipartCalls != 3 || client.completeCalls != 1 {
		t.Fatalf("expected two retries before success, calls=%d complete=%d", client.multipartCalls, client.completeCalls)
	}
}

type fakeClient struct {
	directLimit    int64
	uploadStatus   api.UploadStatus
	statusError    error
	multipartCalls int
	completeCalls  int
	failUploads    int
	paths          []string
	bodies         []string
}

func (client *fakeClient) Capabilities(context.Context, string) (api.Capabilities, error) {
	return api.Capabilities{Limits: map[string]int64{"directUploadMaxBytes": client.directLimit}}, nil
}

func (client *fakeClient) GetUploadStatus(context.Context, string, string) (api.UploadStatus, error) {
	if client.statusError != nil {
		return api.UploadStatus{}, client.statusError
	}
	return client.uploadStatus, nil
}

func (client *fakeClient) GetDocument(_ context.Context, _ string, md5 string) (api.Document, error) {
	return api.Document{FileMD5: md5, Status: "ready"}, nil
}

func (client *fakeClient) UploadMultipart(_ context.Context, _, path, _, _, _ string, body io.Reader, target any) error {
	client.multipartCalls++
	client.paths = append(client.paths, path)
	content, _ := io.ReadAll(body)
	client.bodies = append(client.bodies, string(content))
	if client.failUploads > 0 {
		client.failUploads--
		return apperr.New(apperr.ExitNetwork, "temporary network failure")
	}
	if document, ok := target.(*api.Document); ok {
		document.FileMD5 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		document.Status = "processing"
	}
	return nil
}

func (client *fakeClient) CompleteUpload(_ context.Context, _, md5, _ string) (api.Document, error) {
	client.completeCalls++
	return api.Document{FileMD5: md5, Status: "processing"}, nil
}

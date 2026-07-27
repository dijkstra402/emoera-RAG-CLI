package upload

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/api"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
)

const (
	ChunkSizeBytes       = 5 * 1024 * 1024
	DefaultDirectMaxSize = 25 * 1024 * 1024
)

type Options struct {
	OrgTag      string
	Public      bool
	Description string
	Keywords    string
	RequestID   string
	Progress    func(uploaded, total int64)
}

type Manager struct {
	client Client
}

type Client interface {
	Capabilities(context.Context, string) (api.Capabilities, error)
	GetUploadStatus(context.Context, string, string) (api.UploadStatus, error)
	GetDocument(context.Context, string, string) (api.Document, error)
	UploadMultipart(context.Context, string, string, string, string, string, io.Reader, any) error
	CompleteUpload(context.Context, string, string, string) (api.Document, error)
}

func New(client Client) *Manager { return &Manager{client: client} }

func (manager *Manager) Upload(ctx context.Context, path string, options Options) (api.Document, error) {
	info, err := os.Stat(path)
	if err != nil {
		return api.Document{}, apperr.Wrap(apperr.ExitFile, "无法读取上传文件", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 {
		return api.Document{}, apperr.New(apperr.ExitFile, "只能上传非空普通文件")
	}
	fileMD5, err := checksum(path)
	if err != nil {
		return api.Document{}, err
	}
	directLimit := int64(DefaultDirectMaxSize)
	if capabilities, capabilityError := manager.client.Capabilities(ctx, options.RequestID); capabilityError == nil {
		if configured := capabilities.Limits["directUploadMaxBytes"]; configured > 0 {
			directLimit = configured
		}
	}
	if info.Size() <= directLimit {
		return manager.direct(ctx, path, fileMD5, info, options)
	}
	return manager.resumable(ctx, path, fileMD5, info, options)
}

func (manager *Manager) direct(ctx context.Context, path, fileMD5 string, info os.FileInfo, options Options) (api.Document, error) {
	idempotency := idempotencyKey("upload", fileMD5)
	var lastError error
	for attempt := 0; attempt < 3; attempt++ {
		contentType, body, err := directMultipart(path, options)
		if err != nil {
			return api.Document{}, err
		}
		var document api.Document
		lastError = manager.client.UploadMultipart(
			ctx, http.MethodPost, "/documents", options.RequestID, contentType,
			idempotency, body, &document,
		)
		if lastError == nil {
			if options.Progress != nil {
				options.Progress(info.Size(), info.Size())
			}
			return document, nil
		}
		if !retryable(lastError) || attempt == 2 {
			return api.Document{}, lastError
		}
		if err := waitRetry(ctx, attempt); err != nil {
			return api.Document{}, err
		}
	}
	return api.Document{}, lastError
}

func directMultipart(path string, options Options) (string, io.Reader, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", nil, apperr.Wrap(apperr.ExitFile, "打开上传文件失败", err)
	}
	defer file.Close()
	buffer := &bytes.Buffer{}
	writer := multipart.NewWriter(buffer)
	if err := writeFields(writer, options); err != nil {
		return "", nil, err
	}
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return "", nil, apperr.Wrap(apperr.ExitFile, "创建上传请求失败", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", nil, apperr.Wrap(apperr.ExitFile, "读取上传文件失败", err)
	}
	if err := writer.Close(); err != nil {
		return "", nil, apperr.Wrap(apperr.ExitFile, "完成上传请求失败", err)
	}
	return writer.FormDataContentType(), buffer, nil
}

func (manager *Manager) resumable(ctx context.Context, path, fileMD5 string, info os.FileInfo, options Options) (api.Document, error) {
	totalChunks := int((info.Size() + ChunkSizeBytes - 1) / ChunkSizeBytes)
	missing := make([]int, totalChunks)
	for index := range missing {
		missing[index] = index
	}
	state, err := manager.client.GetUploadStatus(ctx, options.RequestID, fileMD5)
	if err == nil {
		if state.Completed {
			return manager.client.GetDocument(ctx, options.RequestID, fileMD5)
		}
		missing = state.MissingChunks
	} else if apperr.ExitCode(err) != apperr.ExitNotFound {
		return api.Document{}, err
	}

	file, err := os.Open(path)
	if err != nil {
		return api.Document{}, apperr.Wrap(apperr.ExitFile, "打开上传文件失败", err)
	}
	defer file.Close()
	uploaded := info.Size() - missingBytes(missing, info.Size(), totalChunks)
	if options.Progress != nil {
		options.Progress(uploaded, info.Size())
	}

	for _, index := range missing {
		chunkLength := int64(ChunkSizeBytes)
		if index == totalChunks-1 {
			chunkLength = info.Size() - int64(index)*ChunkSizeBytes
		}
		content := make([]byte, chunkLength)
		if _, err := file.ReadAt(content, int64(index)*ChunkSizeBytes); err != nil && err != io.EOF {
			return api.Document{}, apperr.Wrap(apperr.ExitFile, "读取文件分片失败", err)
		}
		if err := manager.uploadChunk(ctx, filepath.Base(path), fileMD5, index, totalChunks, info.Size(), content, options); err != nil {
			return api.Document{}, err
		}
		uploaded += chunkLength
		if options.Progress != nil {
			options.Progress(uploaded, info.Size())
		}
	}
	return manager.client.CompleteUpload(
		ctx, options.RequestID, fileMD5, idempotencyKey("complete", fileMD5),
	)
}

func (manager *Manager) uploadChunk(ctx context.Context, fileName, fileMD5 string, index, totalChunks int, totalSize int64, content []byte, options Options) error {
	var lastError error
	for attempt := 0; attempt < 3; attempt++ {
		buffer := &bytes.Buffer{}
		writer := multipart.NewWriter(buffer)
		fields := map[string]string{
			"totalChunks": strconv.Itoa(totalChunks), "totalSize": strconv.FormatInt(totalSize, 10),
			"fileName": fileName, "isPublic": strconv.FormatBool(options.Public),
		}
		if options.OrgTag != "" {
			fields["orgTag"] = options.OrgTag
		}
		for name, value := range fields {
			if err := writer.WriteField(name, value); err != nil {
				return apperr.Wrap(apperr.ExitFile, "创建分片请求失败", err)
			}
		}
		part, err := writer.CreateFormFile("file", fileName+".part")
		if err != nil {
			return apperr.Wrap(apperr.ExitFile, "创建分片请求失败", err)
		}
		if _, err := part.Write(content); err != nil {
			return apperr.Wrap(apperr.ExitFile, "创建分片请求失败", err)
		}
		if err := writer.Close(); err != nil {
			return apperr.Wrap(apperr.ExitFile, "完成分片请求失败", err)
		}
		var state api.UploadStatus
		lastError = manager.client.UploadMultipart(
			ctx, http.MethodPut,
			"/documents/uploads/"+url.PathEscape(fileMD5)+"/chunks/"+strconv.Itoa(index),
			options.RequestID, writer.FormDataContentType(), "", buffer, &state,
		)
		if lastError == nil {
			return nil
		}
		if !retryable(lastError) || attempt == 2 {
			return lastError
		}
		if err := waitRetry(ctx, attempt); err != nil {
			return err
		}
	}
	return lastError
}

func waitRetry(ctx context.Context, attempt int) error {
	jitter := make([]byte, 1)
	_, _ = rand.Read(jitter)
	delay := time.Duration(1<<attempt)*300*time.Millisecond + time.Duration(jitter[0]%100)*time.Millisecond
	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return apperr.Wrap(apperr.ExitTimeout, "上传请求超时", ctx.Err())
		}
		return apperr.Wrap(apperr.ExitNetwork, "上传请求已取消", ctx.Err())
	case <-time.After(delay):
		return nil
	}
}

func writeFields(writer *multipart.Writer, options Options) error {
	fields := map[string]string{"isPublic": strconv.FormatBool(options.Public)}
	if options.OrgTag != "" {
		fields["orgTag"] = options.OrgTag
	}
	if options.Description != "" {
		fields["description"] = options.Description
	}
	if options.Keywords != "" {
		fields["keywords"] = options.Keywords
	}
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			return apperr.Wrap(apperr.ExitFile, "创建上传请求失败", err)
		}
	}
	return nil
}

func checksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", apperr.Wrap(apperr.ExitFile, "打开上传文件失败", err)
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", apperr.Wrap(apperr.ExitFile, "计算文件 MD5 失败", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func idempotencyKey(operation, fileMD5 string) string {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return operation + "-" + fileMD5
	}
	return fmt.Sprintf("%s-%s-%s", operation, fileMD5, hex.EncodeToString(random))
}

func missingBytes(missing []int, totalSize int64, totalChunks int) int64 {
	var result int64
	for _, index := range missing {
		if index == totalChunks-1 {
			result += totalSize - int64(index)*ChunkSizeBytes
		} else {
			result += ChunkSizeBytes
		}
	}
	return result
}

func retryable(err error) bool {
	switch apperr.ExitCode(err) {
	case apperr.ExitNetwork, apperr.ExitTimeout, apperr.ExitRateLimited, apperr.ExitServer:
		return true
	default:
		return false
	}
}

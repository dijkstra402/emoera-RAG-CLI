package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/api"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/output"
	"github.com/spf13/cobra"
)

const maxAskInputBytes = 1 << 20

var sessionUUIDPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type askOptions struct {
	inputFile           string
	model               string
	session             string
	promptTemplateID    int64
	orgTags             []string
	files               []string
	stream              bool
	includeReferences   bool
	generateSuggestions bool
	idempotencyKey      string
	jsonOutput          bool
	jsonlOutput         bool
}

type chatAccumulator struct {
	result api.ChatCompletionResult
}

func newAskCommand(app *application) *cobra.Command {
	options := askOptions{stream: true, includeReferences: true}
	command := &cobra.Command{
		Use: "ask [question|-]", Short: "向知识库发起 RAG 问答", Args: maximumArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			question, err := readAskInput(command, args, options.inputFile)
			if err != nil {
				return err
			}
			request, err := buildAskRequest(question, options)
			if err != nil {
				return err
			}
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			mode := runtime.Output
			if options.jsonOutput {
				mode = "json"
			}
			if options.jsonlOutput {
				mode = "jsonl"
			}
			if mode == "jsonl" && !options.stream {
				return apperr.New(apperr.ExitArguments, "jsonl 输出需要启用 --stream")
			}
			key := strings.TrimSpace(options.idempotencyKey)
			if key == "" {
				key, err = newCLIRequestID("idem")
				if err != nil {
					return err
				}
			}
			if len(key) < 8 || len(key) > 128 {
				return apperr.New(apperr.ExitArguments, "--idempotency-key 长度必须在 8 到 128 之间")
			}
			if !options.stream {
				return runNonStreamingAsk(command, client, app.requestID, key, request, mode, app.quiet)
			}
			return runStreamingAsk(command, client, app.requestID, key, request, mode, app.quiet, app.noColor)
		},
	}
	flags := command.Flags()
	flags.StringVar(&options.inputFile, "input-file", "", "从文件读取问题")
	flags.StringVar(&options.model, "model", "", "指定模型名称")
	flags.StringVar(&options.session, "session", "", "继续指定会话 UUID")
	flags.Int64Var(&options.promptTemplateID, "prompt-template-id", 0, "指定 Prompt 模板 ID")
	flags.StringArrayVar(&options.orgTags, "org-tag", nil, "限定组织标签，可重复指定")
	flags.StringArrayVar(&options.files, "file", nil, "限定文件 MD5，可重复指定")
	flags.BoolVar(&options.stream, "stream", true, "使用 SSE 流式回答")
	flags.BoolVar(&options.includeReferences, "references", true, "返回引用详情")
	flags.BoolVar(&options.generateSuggestions, "suggestions", false, "生成推荐追问")
	flags.StringVar(&options.idempotencyKey, "idempotency-key", "", "指定幂等键（默认自动生成）")
	flags.BoolVar(&options.jsonOutput, "json", false, "输出 JSON")
	flags.BoolVar(&options.jsonlOutput, "jsonl", false, "逐行输出 SSE 事件 JSON")
	return command
}

func readAskInput(command *cobra.Command, args []string, inputFile string) (string, error) {
	inputFile = strings.TrimSpace(inputFile)
	if inputFile != "" && len(args) > 0 {
		return "", apperr.New(apperr.ExitArguments, "不能同时指定位置参数和 --input-file")
	}
	var reader io.Reader
	switch {
	case inputFile != "":
		file, err := os.Open(inputFile)
		if err != nil {
			return "", apperr.Wrap(apperr.ExitFile, "无法读取问题文件", err)
		}
		defer file.Close()
		reader = file
	case len(args) == 1 && args[0] == "-":
		reader = command.InOrStdin()
	case len(args) == 1:
		return validateAskInput(args[0])
	default:
		return "", apperr.New(apperr.ExitArguments, "请提供问题、- 或 --input-file")
	}
	content, err := io.ReadAll(io.LimitReader(reader, maxAskInputBytes+1))
	if err != nil {
		return "", apperr.Wrap(apperr.ExitFile, "读取问题失败", err)
	}
	if len(content) > maxAskInputBytes {
		return "", apperr.New(apperr.ExitArguments, "问题输入文件过大")
	}
	return validateAskInput(string(content))
}

func validateAskInput(raw string) (string, error) {
	question := strings.TrimSpace(raw)
	if question == "" {
		return "", apperr.New(apperr.ExitArguments, "问题不能为空")
	}
	if !utf8.ValidString(question) {
		return "", apperr.New(apperr.ExitArguments, "问题必须是有效 UTF-8 文本")
	}
	if utf8.RuneCountInString(question) > 16000 {
		return "", apperr.New(apperr.ExitArguments, "问题最多 16000 个字符")
	}
	return question, nil
}

func buildAskRequest(question string, options askOptions) (api.ChatRequest, error) {
	orgTags := normalizeRepeatedValues(options.orgTags)
	files := normalizeRepeatedValues(options.files)
	if len(orgTags) > 20 {
		return api.ChatRequest{}, apperr.New(apperr.ExitArguments, "--org-tag 最多指定 20 个")
	}
	if len(files) > 50 {
		return api.ChatRequest{}, apperr.New(apperr.ExitArguments, "--file 最多指定 50 个")
	}
	for _, fileMD5 := range files {
		if !fileMD5Pattern.MatchString(fileMD5) {
			return api.ChatRequest{}, apperr.New(apperr.ExitArguments, "--file 必须是 32 位文件 MD5")
		}
	}
	request := api.ChatRequest{
		Input: question, OrgTags: orgTags, TargetFileMD5s: files,
		Stream: options.stream, IncludeReferences: options.includeReferences,
		GenerateSuggestions: options.generateSuggestions,
	}
	if model := strings.TrimSpace(options.model); model != "" {
		if utf8.RuneCountInString(model) > 100 {
			return api.ChatRequest{}, apperr.New(apperr.ExitArguments, "--model 最多 100 个字符")
		}
		request.ModelName = &model
	}
	if session := strings.TrimSpace(options.session); session != "" {
		if !sessionUUIDPattern.MatchString(session) {
			return api.ChatRequest{}, apperr.New(apperr.ExitArguments, "--session 必须是有效 UUID")
		}
		request.SessionUUID = &session
	}
	if options.promptTemplateID < 0 {
		return api.ChatRequest{}, apperr.New(apperr.ExitArguments, "--prompt-template-id 不能小于 0")
	}
	if options.promptTemplateID > 0 {
		request.PromptTemplateID = &options.promptTemplateID
	}
	return request, nil
}

func runNonStreamingAsk(command *cobra.Command, client *api.Client, requestID, key string, request api.ChatRequest, mode string, quiet bool) error {
	result, err := client.CompleteChat(commandContext(command), requestID, key, request)
	if err != nil {
		return err
	}
	if quiet || mode == "table" {
		fmt.Fprintln(command.OutOrStdout(), result.Content)
		if !quiet && len(result.References) > 0 {
			fmt.Fprintf(command.ErrOrStderr(), "引用 %d 条 · 会话 %s\n", len(result.References), result.SessionUUID)
		}
		return nil
	}
	return output.Write(command.OutOrStdout(), mode, result)
}

func runStreamingAsk(command *cobra.Command, client *api.Client, requestID, key string, request api.ChatRequest, mode string, quiet, noColor bool) error {
	baseContext := commandContext(command)
	ctx, stop := signal.NotifyContext(baseContext, os.Interrupt, syscall.SIGTERM)
	defer stop()
	accumulator := chatAccumulator{}
	runID := ""
	wroteDelta := false
	lastStage := ""
	presenter := newAskPresenter(command.ErrOrStderr(), noColor)
	handle := func(event api.ChatEvent) error {
		if event.RequestID != "" {
			runID = event.RequestID
		}
		accumulator.accept(event)
		if mode == "jsonl" {
			return writeChatJSONL(command.OutOrStdout(), event)
		}
		if mode == "table" && !quiet {
			switch event.Type {
			case "stage":
				stage := eventString(event, "stage")
				if stage != "" && stage != lastStage {
					presenter.stage(stage)
					lastStage = stage
				}
			case "delta":
				content := eventString(event, "content")
				if content != "" {
					if !wroteDelta {
						presenter.answerHeading()
					}
					fmt.Fprint(command.OutOrStdout(), content)
					wroteDelta = true
				}
			}
		}
		return nil
	}
	err := client.StreamChat(ctx, requestID, key, request, handle)
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		if runID != "" {
			cancelContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, cancelErr := client.CancelChat(cancelContext, runID, requestID)
			cancel()
			if cancelErr != nil && !quiet {
				fmt.Fprintf(command.ErrOrStderr(), "取消请求已中断，本地连接已关闭：%v\n", cancelErr)
			}
		}
		return apperr.New(apperr.ExitInterrupted, "操作已取消")
	}
	if mode == "table" && wroteDelta && !quiet {
		fmt.Fprintln(command.OutOrStdout())
	}
	if err != nil {
		return err
	}
	if mode == "jsonl" {
		return nil
	}
	if quiet || mode == "table" {
		if quiet {
			fmt.Fprintln(command.OutOrStdout(), accumulator.result.Content)
		} else {
			presenter.summary(accumulator.result)
		}
		return nil
	}
	return output.Write(command.OutOrStdout(), mode, accumulator.result)
}

func (accumulator *chatAccumulator) accept(event api.ChatEvent) {
	accumulator.result.RequestID = event.RequestID
	switch event.Type {
	case "meta":
		accumulator.result.SessionUUID = eventString(event, "sessionUuid")
	case "delta":
		accumulator.result.Content += eventString(event, "content")
	case "references":
		accumulator.result.References = eventMapList(event, "items")
	case "usage":
		accumulator.result.Usage = copyEventData(event.Data)
	case "suggestions":
		accumulator.result.Suggestions = eventStringList(event, "items")
	case "done":
		accumulator.result.FinishReason = eventString(event, "finishReason")
		if accumulator.result.SessionUUID == "" {
			accumulator.result.SessionUUID = eventString(event, "sessionUuid")
		}
	}
	if accumulator.result.References == nil {
		accumulator.result.References = []map[string]any{}
	}
	if accumulator.result.Usage == nil {
		accumulator.result.Usage = map[string]any{}
	}
	if accumulator.result.Suggestions == nil {
		accumulator.result.Suggestions = []string{}
	}
}

func writeChatJSONL(writer io.Writer, event api.ChatEvent) error {
	flattened := make(map[string]any, len(event.Data)+4)
	flattened["type"] = event.Type
	flattened["requestId"] = event.RequestID
	flattened["sequence"] = event.Sequence
	flattened["timestamp"] = event.Timestamp
	for key, value := range event.Data {
		if _, reserved := flattened[key]; !reserved {
			flattened[key] = value
		}
	}
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(flattened)
}

func eventString(event api.ChatEvent, key string) string {
	value, _ := event.Data[key].(string)
	return value
}

func eventMapList(event api.ChatEvent, key string) []map[string]any {
	items, ok := event.Data[key].([]any)
	if !ok {
		return []map[string]any{}
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if object, ok := item.(map[string]any); ok {
			result = append(result, object)
		}
	}
	return result
}

func eventStringList(event api.ChatEvent, key string) []string {
	items, ok := event.Data[key].([]any)
	if !ok {
		return []string{}
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if value, ok := item.(string); ok {
			result = append(result, value)
		}
	}
	return result
}

func copyEventData(data map[string]any) map[string]any {
	result := make(map[string]any, len(data))
	for key, value := range data {
		result[key] = value
	}
	return result
}

func newCLIRequestID(prefix string) (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", apperr.Wrap(apperr.ExitServer, "无法生成请求标识", err)
	}
	return prefix + "_" + hex.EncodeToString(buffer), nil
}

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/api"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/config"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/output"
	"github.com/spf13/cobra"
)

var runIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,80}$`)

func newChatCommand(app *application) *cobra.Command {
	command := &cobra.Command{Use: "chat", Short: "查询会话、消息和问答运行状态"}
	command.AddCommand(newChatSessionsCommand(app))
	command.AddCommand(newChatMessagesCommand(app))
	command.AddCommand(newChatExportCommand(app))
	command.AddCommand(newChatRunCommand(app))
	command.AddCommand(newChatCancelCommand(app))
	return command
}

type chatExport struct {
	SessionUUID string            `json:"sessionUuid"`
	ExportedAt  string            `json:"exportedAt"`
	Messages    []api.ChatMessage `json:"messages"`
}

func newChatExportCommand(app *application) *cobra.Command {
	var exportFormat string
	var destination string
	var force bool
	command := &cobra.Command{
		Use: "export <session-uuid>", Short: "将完整会话导出为 Markdown 或 JSON", Args: exactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			sessionUUID := args[0]
			if !sessionUUIDPattern.MatchString(sessionUUID) {
				return apperr.New(apperr.ExitArguments, "session UUID 格式无效")
			}
			exportFormat = strings.ToLower(strings.TrimSpace(exportFormat))
			if exportFormat != "markdown" && exportFormat != "json" {
				return apperr.New(apperr.ExitArguments, "导出格式只支持 markdown 或 json")
			}
			client, _, err := app.client()
			if err != nil {
				return err
			}
			messages, err := collectChatMessages(command, client, app.requestID, sessionUUID)
			if err != nil {
				return err
			}
			result := chatExport{
				SessionUUID: sessionUUID,
				ExportedAt:  time.Now().Format(time.RFC3339),
				Messages:    messages,
			}
			content, err := encodeChatExport(result, exportFormat)
			if err != nil {
				return apperr.Wrap(apperr.ExitServer, "无法生成会话导出内容", err)
			}
			if destination == "" {
				extension := ".md"
				if exportFormat == "json" {
					extension = ".json"
				}
				destination = sessionUUID + extension
			}
			if destination == "-" {
				_, err = command.OutOrStdout().Write(content)
				return err
			}
			if err := writeExportFile(destination, content, force); err != nil {
				return err
			}
			successMessage(command, "已导出 %d 条消息到 %s", len(messages), destination)
			return nil
		},
	}
	command.Flags().StringVar(&exportFormat, "format", "markdown", "导出格式：markdown 或 json")
	command.Flags().StringVarP(&destination, "output-file", "f", "", "输出文件；使用 - 写入标准输出")
	command.Flags().BoolVar(&force, "force", false, "覆盖已存在的文件")
	return command
}

func collectChatMessages(command *cobra.Command, client *api.Client, requestID, sessionUUID string) ([]api.ChatMessage, error) {
	messages := make([]api.ChatMessage, 0, 100)
	cursor := ""
	for {
		page, err := client.ListChatMessages(commandContext(command), requestID, sessionUUID, cursor, 100)
		if err != nil {
			return nil, err
		}
		messages = append(messages, page.Items...)
		if !page.HasMore || page.NextCursor == nil || strings.TrimSpace(*page.NextCursor) == "" {
			break
		}
		next := strings.TrimSpace(*page.NextCursor)
		if next == cursor {
			return nil, apperr.New(apperr.ExitServer, "服务返回了重复的会话游标")
		}
		cursor = next
	}
	return messages, nil
}

func encodeChatExport(result chatExport, exportFormat string) ([]byte, error) {
	if exportFormat == "json" {
		content, err := json.MarshalIndent(result, "", "  ")
		return append(content, '\n'), err
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Emoera 对话导出\n\n- 会话：`%s`\n- 导出时间：%s\n- 消息数：%d\n\n---\n\n",
		result.SessionUUID, result.ExportedAt, len(result.Messages))
	for _, message := range result.Messages {
		role := "AI 助手"
		if strings.EqualFold(message.Role, "user") {
			role = "用户"
		} else if strings.TrimSpace(message.Role) != "" && !strings.EqualFold(message.Role, "assistant") {
			role = message.Role
		}
		fmt.Fprintf(&builder, "## %s\n\n%s\n\n", role, strings.TrimSpace(message.Content))
		if len(message.References) > 0 {
			builder.WriteString("### 引用\n\n")
			for index, reference := range message.References {
				name := firstReferenceValue(reference, "fileName", "filename", "title", "fileMd5")
				if name == "" {
					name = fmt.Sprintf("引用 %d", index+1)
				}
				fmt.Fprintf(&builder, "- %s\n", name)
			}
			builder.WriteString("\n")
		}
		builder.WriteString("---\n\n")
	}
	return []byte(builder.String()), nil
}

func firstReferenceValue(reference map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(fmt.Sprint(reference[key])); value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func writeExportFile(destination string, content []byte, force bool) error {
	if strings.TrimSpace(destination) == "" {
		return apperr.New(apperr.ExitArguments, "输出文件不能为空")
	}
	if !force {
		if _, err := os.Stat(destination); err == nil {
			return apperr.New(apperr.ExitConflict, "输出文件已存在；使用 --force 覆盖")
		} else if !os.IsNotExist(err) {
			return apperr.Wrap(apperr.ExitConfiguration, "无法检查输出文件", err)
		}
	}
	parent := filepath.Dir(destination)
	if parent != "." {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return apperr.Wrap(apperr.ExitConfiguration, "无法创建输出目录", err)
		}
	}
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return apperr.Wrap(apperr.ExitConfiguration, "无法创建导出文件", err)
	}
	defer file.Close()
	if _, err := io.Copy(file, strings.NewReader(string(content))); err != nil {
		return apperr.Wrap(apperr.ExitConfiguration, "无法写入导出文件", err)
	}
	return nil
}

func newChatSessionsCommand(app *application) *cobra.Command {
	var cursor string
	var limit int
	var jsonOutput bool
	command := &cobra.Command{
		Use: "sessions", Short: "列出当前账号的会话", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if err := validatePageOptions(cursor, limit); err != nil {
				return err
			}
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			page, err := client.ListChatSessions(commandContext(command), app.requestID, cursor, limit)
			if err != nil {
				return err
			}
			if app.quiet {
				for _, item := range page.Items {
					fmt.Fprintln(command.OutOrStdout(), item.UUID)
				}
				return nil
			}
			if outputMode(runtime.Output, jsonOutput) != "table" {
				return output.Write(command.OutOrStdout(), outputMode(runtime.Output, jsonOutput), page)
			}
			fmt.Fprintln(command.OutOrStdout(), "SESSION_UUID\tMODEL\tUPDATED_AT\tTITLE")
			for _, item := range page.Items {
				fmt.Fprintf(command.OutOrStdout(), "%s\t%s\t%s\t%s\n",
					item.UUID, pointerText(item.ModelName), item.UpdatedAt, sanitizeTableCell(item.Title))
			}
			writeNextCursor(command, page.NextCursor)
			return nil
		},
	}
	command.Flags().StringVar(&cursor, "cursor", "", "下一页游标")
	command.Flags().IntVar(&limit, "limit", 20, "返回数量（1-100）")
	command.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func newChatMessagesCommand(app *application) *cobra.Command {
	var cursor string
	var limit int
	var jsonOutput bool
	command := &cobra.Command{
		Use: "messages <session-uuid>", Short: "读取指定会话的消息", Args: exactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			if !sessionUUIDPattern.MatchString(args[0]) {
				return apperr.New(apperr.ExitArguments, "session UUID 格式无效")
			}
			if err := validatePageOptions(cursor, limit); err != nil {
				return err
			}
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			page, err := client.ListChatMessages(commandContext(command), app.requestID, args[0], cursor, limit)
			if err != nil {
				return err
			}
			if app.quiet {
				for _, item := range page.Items {
					fmt.Fprintln(command.OutOrStdout(), item.Content)
				}
				return nil
			}
			if outputMode(runtime.Output, jsonOutput) != "table" {
				return output.Write(command.OutOrStdout(), outputMode(runtime.Output, jsonOutput), page)
			}
			fmt.Fprintln(command.OutOrStdout(), "ID\tROLE\tCREATED_AT\tREFERENCES\tCONTENT")
			for _, item := range page.Items {
				fmt.Fprintf(command.OutOrStdout(), "%s\t%s\t%s\t%d\t%s\n",
					item.ID, item.Role, item.CreatedAt, len(item.References),
					truncateRunes(sanitizeTableCell(item.Content), 120))
			}
			writeNextCursor(command, page.NextCursor)
			return nil
		},
	}
	command.Flags().StringVar(&cursor, "cursor", "", "下一页游标")
	command.Flags().IntVar(&limit, "limit", 20, "返回数量（1-100）")
	command.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func newChatRunCommand(app *application) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "run <request-id>", Short: "查询问答运行状态", Args: exactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			if !runIDPattern.MatchString(args[0]) {
				return apperr.New(apperr.ExitArguments, "request ID 格式无效")
			}
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			status, err := client.GetChatRun(commandContext(command), app.requestID, args[0])
			if err != nil {
				return err
			}
			if app.quiet {
				fmt.Fprintln(command.OutOrStdout(), status.Status)
				return nil
			}
			return output.Write(command.OutOrStdout(), outputMode(runtime.Output, jsonOutput), status)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func newChatCancelCommand(app *application) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "cancel <request-id>", Short: "取消正在执行的问答", Args: exactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			if !runIDPattern.MatchString(args[0]) {
				return apperr.New(apperr.ExitArguments, "request ID 格式无效")
			}
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			result, err := client.CancelChatRun(commandContext(command), app.requestID, args[0])
			if err != nil {
				return err
			}
			if app.quiet {
				fmt.Fprintln(command.OutOrStdout(), result.Status)
				return nil
			}
			return output.Write(command.OutOrStdout(), outputMode(runtime.Output, jsonOutput), result)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func newModelCommand(app *application) *cobra.Command {
	root := &cobra.Command{Use: "model", Short: "查看模型并管理当前 Profile 的默认模型"}
	var jsonOutput bool
	list := &cobra.Command{
		Use: "list", Short: "列出可用模型", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			items, err := client.ListModels(commandContext(command), app.requestID)
			if err != nil {
				return err
			}
			if app.quiet {
				for _, item := range items {
					if item.Available {
						fmt.Fprintln(command.OutOrStdout(), item.Name)
					}
				}
				return nil
			}
			if outputMode(runtime.Output, jsonOutput) != "table" {
				return output.Write(command.OutOrStdout(), outputMode(runtime.Output, jsonOutput), items)
			}
			effectiveName := effectiveModelName(items, runtime.DefaultModel)
			fmt.Fprintln(command.OutOrStdout(), "SELECTED\tNAME\tAVAILABLE\tMULTIPLIER\tPROVIDER\tDISPLAY_NAME")
			for _, item := range items {
				selected := ""
				if strings.EqualFold(item.Name, effectiveName) {
					selected = "*"
				}
				fmt.Fprintf(command.OutOrStdout(), "%s\t%s\t%t\t%.2fx\t%s\t%s\n",
					selected, item.Name, item.Available, item.Multiplier, item.Provider, sanitizeTableCell(item.DisplayName))
			}
			return nil
		},
	}
	list.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	root.AddCommand(list)
	root.AddCommand(newModelCurrentCommand(app))
	root.AddCommand(newModelUseCommand(app))
	root.AddCommand(newModelResetCommand(app))
	return root
}

func newModelCurrentCommand(app *application) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "current", Short: "显示当前问答默认使用的模型", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			items, err := client.ListModels(commandContext(command), app.requestID)
			if err != nil {
				return err
			}
			name := effectiveModelName(items, runtime.DefaultModel)
			model, found := findModel(items, name)
			result := map[string]any{
				"name":      name,
				"source":    runtime.ModelSource,
				"available": found && model.Available,
			}
			if found {
				result["displayName"] = model.DisplayName
				result["provider"] = model.Provider
				result["multiplier"] = model.Multiplier
			}
			if outputMode(runtime.Output, jsonOutput) != "table" {
				return output.Write(command.OutOrStdout(), outputMode(runtime.Output, jsonOutput), result)
			}
			fmt.Fprintf(command.OutOrStdout(), "当前模型：%s\n", name)
			fmt.Fprintf(command.OutOrStdout(), "配置来源：%s\n", modelSourceLabel(runtime.ModelSource))
			if found {
				fmt.Fprintf(command.OutOrStdout(), "显示名称：%s\n", model.DisplayName)
				fmt.Fprintf(command.OutOrStdout(), "提供方：%s · 倍率：%.2fx · 可用：%t\n", model.Provider, model.Multiplier, model.Available)
			}
			return nil
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func newModelUseCommand(app *application) *cobra.Command {
	return &cobra.Command{
		Use: "use <model>", Short: "设置当前 Profile 的默认模型", Args: exactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			items, err := client.ListModels(commandContext(command), app.requestID)
			if err != nil {
				return err
			}
			model, found := findModel(items, args[0])
			if !found {
				return apperr.New(apperr.ExitArguments, fmt.Sprintf("模型 %q 不存在，请运行 emoera model list 查看可用模型", args[0]))
			}
			if !model.Available {
				return apperr.New(apperr.ExitArguments, fmt.Sprintf("模型 %q 当前不可用", model.Name))
			}
			path, file, err := app.loadFile()
			if err != nil {
				return err
			}
			if err := config.Set(&file, runtime.Profile, "default-model", model.Name); err != nil {
				return err
			}
			if err := config.Save(path, file); err != nil {
				return err
			}
			successMessage(command, "Profile %s 的默认模型已设置为 %s", runtime.Profile, model.Name)
			return nil
		},
	}
}

func newModelResetCommand(app *application) *cobra.Command {
	return &cobra.Command{
		Use: "reset", Short: "清除 Profile 默认模型并恢复服务端默认值", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			runtime, err := app.runtime()
			if err != nil {
				return err
			}
			path, file, err := app.loadFile()
			if err != nil {
				return err
			}
			if err := config.Set(&file, runtime.Profile, "default-model", ""); err != nil {
				return err
			}
			if err := config.Save(path, file); err != nil {
				return err
			}
			if runtime.ModelSource == "environment" {
				successMessage(command, "已清除 Profile %s 的默认模型；环境变量 EMOERA_MODEL=%s 仍然生效", runtime.Profile, runtime.DefaultModel)
				return nil
			}
			successMessage(command, "已恢复 Profile %s 的服务端默认模型", runtime.Profile)
			return nil
		},
	}
}

func findModel(items []api.Model, value string) (api.Model, bool) {
	value = strings.TrimSpace(value)
	for _, item := range items {
		if strings.EqualFold(item.Name, value) || strings.EqualFold(item.DisplayName, value) {
			return item, true
		}
	}
	return api.Model{}, false
}

func effectiveModelName(items []api.Model, configured string) string {
	if configured = strings.TrimSpace(configured); configured != "" {
		if model, found := findModel(items, configured); found {
			return model.Name
		}
		return configured
	}
	for _, item := range items {
		if item.Available && item.Default {
			return item.Name
		}
	}
	for _, item := range items {
		if item.Available {
			return item.Name
		}
	}
	return ""
}

func modelSourceLabel(source string) string {
	switch source {
	case "environment":
		return "环境变量 EMOERA_MODEL"
	case "profile":
		return "当前 Profile"
	default:
		return "服务端默认值"
	}
}

func newQuotaCommand(app *application) *cobra.Command {
	root := &cobra.Command{Use: "quota", Short: "查看当前账号用量与配额"}
	var jsonOutput bool
	show := &cobra.Command{
		Use: "show", Short: "显示对话、请求和存储配额", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			quota, err := client.GetQuota(commandContext(command), app.requestID)
			if err != nil {
				return err
			}
			if outputMode(runtime.Output, jsonOutput) != "table" {
				return output.Write(command.OutOrStdout(), outputMode(runtime.Output, jsonOutput), quota)
			}
			fmt.Fprintln(command.OutOrStdout(), "TYPE\tUSED\tLIMIT\tREMAINING\tRESETS_AT")
			writeUsageLimit(command, "daily_chat", quota.DailyChat)
			writeUsageLimit(command, "minute_requests", quota.MinuteRequests)
			writeUsageLimit(command, "storage_bytes", quota.Storage)
			return nil
		},
	}
	show.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	root.AddCommand(show)
	return root
}

func newStatusCommand(app *application) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "status", Short: "查看公开业务功能状态", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			status, err := client.GetStatus(commandContext(command), app.requestID)
			if err != nil {
				return err
			}
			if app.quiet {
				fmt.Fprintln(command.OutOrStdout(), status.Overall)
				return nil
			}
			if outputMode(runtime.Output, jsonOutput) != "table" {
				return output.Write(command.OutOrStdout(), outputMode(runtime.Output, jsonOutput), status)
			}
			fmt.Fprintf(command.OutOrStdout(), "整体状态：%s（检查时间 %s）\n", status.Overall, status.CheckedAt)
			fmt.Fprintln(command.OutOrStdout(), "CODE\tSTATUS\tLATENCY_MS\tNAME")
			for _, module := range status.Modules {
				latency := "-"
				if module.LatencyMS != nil {
					latency = fmt.Sprint(*module.LatencyMS)
				}
				fmt.Fprintf(command.OutOrStdout(), "%s\t%s\t%s\t%s\n",
					module.Code, module.Status, latency, sanitizeTableCell(module.Name))
			}
			return nil
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func validatePageOptions(cursor string, limit int) error {
	if limit < 1 || limit > 100 {
		return apperr.New(apperr.ExitArguments, "--limit 必须在 1 到 100 之间")
	}
	if len(cursor) > 300 {
		return apperr.New(apperr.ExitArguments, "--cursor 长度不能超过 300")
	}
	return nil
}

func outputMode(defaultMode string, jsonOutput bool) string {
	if jsonOutput {
		return "json"
	}
	return defaultMode
}

func pointerText(value *string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return "-"
	}
	return *value
}

func writeNextCursor(command *cobra.Command, cursor *string) {
	if cursor != nil {
		fmt.Fprintf(command.ErrOrStderr(), "下一页：--cursor %s\n", *cursor)
	}
}

func writeUsageLimit(command *cobra.Command, name string, value api.UsageLimit) {
	reset := "-"
	if value.ResetsAt != nil {
		reset = *value.ResetsAt
	}
	fmt.Fprintf(command.OutOrStdout(), "%s\t%d\t%d\t%d\t%s\n",
		name, value.Used, value.Limit, value.Remaining, reset)
}

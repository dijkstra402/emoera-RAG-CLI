package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/api"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/output"
	"github.com/spf13/cobra"
)

var runIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,80}$`)

func newChatCommand(app *application) *cobra.Command {
	command := &cobra.Command{Use: "chat", Short: "查询会话、消息和问答运行状态"}
	command.AddCommand(newChatSessionsCommand(app))
	command.AddCommand(newChatMessagesCommand(app))
	command.AddCommand(newChatRunCommand(app))
	command.AddCommand(newChatCancelCommand(app))
	return command
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
	root := &cobra.Command{Use: "model", Short: "查看可用模型与倍率"}
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
			fmt.Fprintln(command.OutOrStdout(), "NAME\tAVAILABLE\tMULTIPLIER\tPROVIDER\tDISPLAY_NAME")
			for _, item := range items {
				fmt.Fprintf(command.OutOrStdout(), "%s\t%t\t%.2fx\t%s\t%s\n",
					item.Name, item.Available, item.Multiplier, item.Provider, sanitizeTableCell(item.DisplayName))
			}
			return nil
		},
	}
	list.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	root.AddCommand(list)
	return root
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

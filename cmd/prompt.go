package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/api"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/output"
	"github.com/spf13/cobra"
)

func newPromptCommand(app *application) *cobra.Command {
	command := &cobra.Command{Use: "prompt", Short: "管理个人与全局 Prompt 模板"}
	command.AddCommand(newPromptListCommand(app))
	command.AddCommand(newPromptShowCommand(app))
	command.AddCommand(newPromptCreateCommand(app))
	command.AddCommand(newPromptUpdateCommand(app))
	command.AddCommand(newPromptDeleteCommand(app))
	command.AddCommand(newPromptDefaultCommand(app))
	return command
}

func newPromptListCommand(app *application) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "list", Short: "列出可管理的 Prompt 模板", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			items, err := client.ListPrompts(commandContext(command), app.requestID)
			if err != nil {
				return err
			}
			if app.quiet {
				for _, item := range items {
					fmt.Fprintln(command.OutOrStdout(), item.ID)
				}
				return nil
			}
			if outputMode(runtime.Output, jsonOutput) != "table" {
				return output.Write(command.OutOrStdout(), outputMode(runtime.Output, jsonOutput), items)
			}
			fmt.Fprintln(command.OutOrStdout(), "ID\tDEFAULT\tENABLED\tSCOPE\tOWNER\tNAME")
			for _, item := range items {
				selected := ""
				if item.DefaultTemplate {
					selected = "*"
				}
				fmt.Fprintf(command.OutOrStdout(), "%d\t%s\t%t\t%s\t%s\t%s\n",
					item.ID, selected, item.Enabled, item.Scope,
					sanitizeTableCell(item.Owner), sanitizeTableCell(item.Name))
			}
			return nil
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func newPromptShowCommand(app *application) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "show <id>", Short: "查看 Prompt 模板完整内容", Args: exactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			id, err := parsePositiveID(args[0])
			if err != nil {
				return err
			}
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			item, err := client.GetPrompt(commandContext(command), app.requestID, id)
			if err != nil {
				return err
			}
			if app.quiet {
				fmt.Fprintln(command.OutOrStdout(), item.Content)
				return nil
			}
			return output.Write(command.OutOrStdout(), outputMode(runtime.Output, jsonOutput), item)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func newPromptCreateCommand(app *application) *cobra.Command {
	var name, description, content, contentFile, scope string
	var disabled, jsonOutput bool
	command := &cobra.Command{
		Use: "create", Short: "创建 Prompt 模板", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			resolvedContent, err := resolvePromptContent(content, contentFile)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return apperr.New(apperr.ExitArguments, "--name 不能为空")
			}
			resolvedScope, err := normalizePromptScope(scope)
			if err != nil {
				return err
			}
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			item, err := client.CreatePrompt(commandContext(command), app.requestID, api.PromptTemplateRequest{
				Name: name, Description: description, Content: resolvedContent,
				Scope: resolvedScope, Enabled: !disabled,
			})
			if err != nil {
				return err
			}
			if app.quiet {
				fmt.Fprintln(command.OutOrStdout(), item.ID)
				return nil
			}
			return output.Write(command.OutOrStdout(), outputMode(runtime.Output, jsonOutput), item)
		},
	}
	flags := command.Flags()
	flags.StringVar(&name, "name", "", "模板名称（必填）")
	flags.StringVar(&description, "description", "", "模板说明")
	flags.StringVar(&content, "content", "", "模板内容，与 --content-file 二选一")
	flags.StringVar(&contentFile, "content-file", "", "从 UTF-8 文件读取模板内容")
	flags.StringVar(&scope, "scope", "user", "模板范围：user/global")
	flags.BoolVar(&disabled, "disabled", false, "创建为停用状态")
	flags.BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func newPromptUpdateCommand(app *application) *cobra.Command {
	var name, description, content, contentFile, scope string
	var enabled, disabled, jsonOutput bool
	command := &cobra.Command{
		Use: "update <id>", Short: "修改 Prompt 模板", Args: exactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			id, err := parsePositiveID(args[0])
			if err != nil {
				return err
			}
			if enabled && disabled {
				return apperr.New(apperr.ExitArguments, "--enabled 与 --disabled 不能同时使用")
			}
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			current, err := client.GetPrompt(commandContext(command), app.requestID, id)
			if err != nil {
				return err
			}
			body := api.PromptTemplateRequest{
				Name: current.Name, Description: current.Description, Content: current.Content,
				Scope: current.Scope, Enabled: current.Enabled,
			}
			flags := command.Flags()
			changed := false
			if flags.Changed("name") {
				body.Name, changed = name, true
			}
			if flags.Changed("description") {
				body.Description, changed = description, true
			}
			if flags.Changed("content") || flags.Changed("content-file") {
				body.Content, err = resolvePromptContent(content, contentFile)
				if err != nil {
					return err
				}
				changed = true
			}
			if flags.Changed("scope") {
				body.Scope, err = normalizePromptScope(scope)
				if err != nil {
					return err
				}
				changed = true
			}
			if enabled || disabled {
				body.Enabled, changed = enabled, true
			}
			if !changed {
				return apperr.New(apperr.ExitArguments, "至少指定一个需要修改的字段")
			}
			item, err := client.UpdatePrompt(commandContext(command), app.requestID, id, body)
			if err != nil {
				return err
			}
			return output.Write(command.OutOrStdout(), outputMode(runtime.Output, jsonOutput), item)
		},
	}
	flags := command.Flags()
	flags.StringVar(&name, "name", "", "新的模板名称")
	flags.StringVar(&description, "description", "", "新的模板说明；空字符串可清除")
	flags.StringVar(&content, "content", "", "新的模板内容")
	flags.StringVar(&contentFile, "content-file", "", "从 UTF-8 文件读取新内容")
	flags.StringVar(&scope, "scope", "", "新的模板范围：user/global")
	flags.BoolVar(&enabled, "enabled", false, "启用模板")
	flags.BoolVar(&disabled, "disabled", false, "停用模板")
	flags.BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func newPromptDeleteCommand(app *application) *cobra.Command {
	var yes bool
	command := &cobra.Command{
		Use: "delete <id>", Short: "删除 Prompt 模板", Args: exactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			id, err := parsePositiveID(args[0])
			if err != nil {
				return err
			}
			if !yes {
				return apperr.New(apperr.ExitArguments, "删除不可撤销，请确认后增加 --yes")
			}
			client, _, err := app.client()
			if err != nil {
				return err
			}
			if err := client.DeletePrompt(commandContext(command), app.requestID, id); err != nil {
				return err
			}
			successMessage(command, "Prompt 模板 %d 已删除", id)
			return nil
		},
	}
	command.Flags().BoolVar(&yes, "yes", false, "确认执行不可撤销的删除")
	return command
}

func newPromptDefaultCommand(app *application) *cobra.Command {
	return &cobra.Command{
		Use: "default <id>", Short: "将 Prompt 模板设为默认", Args: exactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			id, err := parsePositiveID(args[0])
			if err != nil {
				return err
			}
			client, _, err := app.client()
			if err != nil {
				return err
			}
			item, err := client.SetDefaultPrompt(commandContext(command), app.requestID, id)
			if err != nil {
				return err
			}
			successMessage(command, "Prompt 模板 %d（%s）已设为默认", item.ID, item.Name)
			return nil
		},
	}
}

func parsePositiveID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id < 1 {
		return 0, apperr.New(apperr.ExitArguments, "ID 必须是正整数")
	}
	return id, nil
}

func resolvePromptContent(content, contentFile string) (string, error) {
	if strings.TrimSpace(content) != "" && strings.TrimSpace(contentFile) != "" {
		return "", apperr.New(apperr.ExitArguments, "--content 与 --content-file 只能使用一个")
	}
	if strings.TrimSpace(contentFile) != "" {
		data, err := os.ReadFile(contentFile)
		if err != nil {
			return "", apperr.Wrap(apperr.ExitArguments, "无法读取 Prompt 文件", err)
		}
		content = string(data)
	}
	if strings.TrimSpace(content) == "" {
		return "", apperr.New(apperr.ExitArguments, "Prompt 内容不能为空，请设置 --content 或 --content-file")
	}
	return content, nil
}

func normalizePromptScope(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "user":
		return "USER", nil
	case "global":
		return "GLOBAL", nil
	default:
		return "", apperr.New(apperr.ExitArguments, "--scope 只能是 user 或 global")
	}
}

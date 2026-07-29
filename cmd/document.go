package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/api"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/output"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/upload"
	"github.com/spf13/cobra"
)

var documentMD5Pattern = regexp.MustCompile(`^[a-fA-F0-9]{32}$`)

func newDocumentCommand(app *application) *cobra.Command {
	command := &cobra.Command{Use: "doc", Aliases: []string{"document"}, Short: "管理知识库文档"}
	command.AddCommand(newDocumentListCommand(app))
	command.AddCommand(newDocumentShowCommand(app))
	command.AddCommand(newDocumentStatusCommand(app))
	command.AddCommand(newDocumentUploadCommand(app))
	command.AddCommand(newDocumentUpdateCommand(app))
	command.AddCommand(newDocumentDeleteCommand(app))
	return command
}

func newDocumentUpdateCommand(app *application) *cobra.Command {
	var orgTag, description, keywords string
	var makePublic, makePrivate, jsonOutput bool
	command := &cobra.Command{
		Use: "update <file-md5>", Short: "修改文档组织、可见性和元数据", Args: exactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			if !documentMD5Pattern.MatchString(args[0]) {
				return apperr.New(apperr.ExitArguments, "文件 MD5 格式无效")
			}
			if makePublic && makePrivate {
				return apperr.New(apperr.ExitArguments, "--public 与 --private 不能同时使用")
			}
			flags := command.Flags()
			body := api.DocumentUpdateRequest{}
			changed := false
			if flags.Changed("org-tag") {
				body.OrgTag = &orgTag
				changed = true
			}
			if flags.Changed("description") {
				body.Description = &description
				changed = true
			}
			if flags.Changed("keywords") {
				items := splitCommaValues(keywords)
				body.Keywords = &items
				changed = true
			}
			if makePublic || makePrivate {
				value := makePublic
				body.Public = &value
				changed = true
			}
			if !changed {
				return apperr.New(apperr.ExitArguments, "至少指定一个需要修改的字段")
			}
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			document, err := client.UpdateDocument(commandContext(command), app.requestID, args[0], body)
			if err != nil {
				return err
			}
			if app.quiet {
				fmt.Fprintln(command.OutOrStdout(), document.FileMD5)
				return nil
			}
			return output.Write(command.OutOrStdout(), outputMode(runtime.Output, jsonOutput), document)
		},
	}
	flags := command.Flags()
	flags.StringVar(&orgTag, "org-tag", "", "新的组织标签")
	flags.StringVar(&description, "description", "", "新的文档描述；空字符串可清除")
	flags.StringVar(&keywords, "keywords", "", "逗号分隔关键词；空字符串可清除")
	flags.BoolVar(&makePublic, "public", false, "设为公开文档")
	flags.BoolVar(&makePrivate, "private", false, "设为私有文档")
	flags.BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func newDocumentDeleteCommand(app *application) *cobra.Command {
	var yes bool
	var idempotencyKey string
	command := &cobra.Command{
		Use: "delete <file-md5>", Short: "删除文档、对象文件和检索索引", Args: exactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			if !documentMD5Pattern.MatchString(args[0]) {
				return apperr.New(apperr.ExitArguments, "文件 MD5 格式无效")
			}
			if !yes {
				return apperr.New(apperr.ExitArguments, "删除不可撤销，请确认后增加 --yes")
			}
			if strings.TrimSpace(idempotencyKey) == "" {
				generated, err := newCLIRequestID("doc-delete")
				if err != nil {
					return err
				}
				idempotencyKey = generated
			}
			client, _, err := app.client()
			if err != nil {
				return err
			}
			if err := client.DeleteDocument(commandContext(command), app.requestID, args[0], idempotencyKey); err != nil {
				return err
			}
			if app.quiet {
				fmt.Fprintln(command.OutOrStdout(), args[0])
				return nil
			}
			successMessage(command, "文档 %s 已删除", args[0])
			return nil
		},
	}
	command.Flags().BoolVar(&yes, "yes", false, "确认执行不可撤销的删除")
	command.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "自定义幂等键")
	return command
}

func splitCommaValues(value string) []string {
	items := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		if normalized := strings.TrimSpace(item); normalized != "" {
			items = append(items, normalized)
		}
	}
	return items
}

func newDocumentListCommand(app *application) *cobra.Command {
	var options api.DocumentListOptions
	var jsonOutput bool
	command := &cobra.Command{
		Use: "list", Short: "筛选并列出文档", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			page, err := client.ListDocuments(commandContext(command), app.requestID, options)
			if err != nil {
				return err
			}
			mode := runtime.Output
			if jsonOutput {
				mode = "json"
			}
			if mode == "table" {
				return writeDocumentPage(command, page)
			}
			return output.Write(command.OutOrStdout(), mode, page)
		},
	}
	flags := command.Flags()
	flags.StringVar(&options.Cursor, "cursor", "", "下一页游标")
	flags.IntVar(&options.Limit, "limit", 20, "返回数量（1-100）")
	flags.StringVar(&options.Query, "query", "", "文件名关键词")
	flags.StringVar(&options.OrgTag, "org-tag", "", "组织标签")
	flags.StringVar(&options.Visibility, "visibility", "all", "可见性：all/public/private")
	flags.StringVar(&options.Status, "status", "", "状态：uploading/processing/ready/failed")
	flags.BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func writeDocumentPage(command *cobra.Command, page api.DocumentPage) error {
	fmt.Fprintln(command.OutOrStdout(), "FILE_MD5\tSTATUS\tVISIBILITY\tSIZE\tFILE_NAME")
	for _, document := range page.Items {
		fmt.Fprintf(command.OutOrStdout(), "%s\t%s\t%s\t%d\t%s\n",
			document.FileMD5, document.Status, document.Visibility,
			document.FileSize, document.FileName)
	}
	if page.NextCursor != nil {
		fmt.Fprintf(command.ErrOrStderr(), "下一页：--cursor %s\n", *page.NextCursor)
	}
	return nil
}

func newDocumentShowCommand(app *application) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "show <file-md5>", Short: "查看文档详情", Args: exactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			document, err := client.GetDocument(commandContext(command), app.requestID, args[0])
			if err != nil {
				return err
			}
			mode := runtime.Output
			if jsonOutput {
				mode = "json"
			}
			return output.Write(command.OutOrStdout(), mode, document)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func newDocumentStatusCommand(app *application) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "status <file-md5>", Short: "查看解析状态", Args: exactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			document, err := client.GetDocument(commandContext(command), app.requestID, args[0])
			if err != nil {
				return err
			}
			data := map[string]any{
				"fileMd5": document.FileMD5, "fileName": document.FileName,
				"status": document.Status, "processingError": document.ProcessingError,
			}
			mode := runtime.Output
			if jsonOutput {
				mode = "json"
			}
			return output.Write(command.OutOrStdout(), mode, data)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func newDocumentUploadCommand(app *application) *cobra.Command {
	var orgTag, description, keywords string
	var isPublic, wait, jsonOutput bool
	var maxWait time.Duration
	command := &cobra.Command{
		Use: "upload <path>", Short: "上传文档，小文件直传、大文件自动断点续传", Args: exactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			manager := upload.New(client)
			progress := func(uploaded, total int64) {
				if app.quiet || total <= 0 {
					return
				}
				fmt.Fprintf(command.ErrOrStderr(), "\r上传进度 %6.2f%%", float64(uploaded)*100/float64(total))
				if uploaded >= total {
					fmt.Fprintln(command.ErrOrStderr())
				}
			}
			document, err := manager.Upload(commandContext(command), args[0], upload.Options{
				OrgTag: orgTag, Public: isPublic, Description: description,
				Keywords: keywords, RequestID: app.requestID, Progress: progress,
			})
			if err != nil {
				return err
			}
			if wait {
				waitContext, cancel := context.WithTimeout(commandContext(command), maxWait)
				defer cancel()
				document, err = client.WaitDocument(waitContext, app.requestID, document.FileMD5, 2*time.Second)
				if err != nil {
					if waitContext.Err() != nil {
						return apperr.Wrap(apperr.ExitTimeout, "等待文档解析超时", waitContext.Err())
					}
					return err
				}
				if document.Status == "failed" {
					message := "文档解析失败"
					if document.ProcessingError != nil && strings.TrimSpace(*document.ProcessingError) != "" {
						message += "：" + *document.ProcessingError
					}
					return apperr.New(apperr.ExitServer, message)
				}
			}
			if app.quiet {
				fmt.Fprintln(command.OutOrStdout(), document.FileMD5)
				return nil
			}
			mode := runtime.Output
			if jsonOutput {
				mode = "json"
			}
			return output.Write(command.OutOrStdout(), mode, document)
		},
	}
	flags := command.Flags()
	flags.StringVar(&orgTag, "org-tag", "", "写入组织标签")
	flags.BoolVar(&isPublic, "public", false, "设为公开文档")
	flags.StringVar(&description, "description", "", "文档描述")
	flags.StringVar(&keywords, "keywords", "", "逗号分隔关键词")
	flags.BoolVar(&wait, "wait", false, "等待解析完成")
	flags.DurationVar(&maxWait, "max-wait", 10*time.Minute, "最长等待解析时间")
	flags.BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

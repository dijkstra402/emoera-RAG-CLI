package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/api"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/output"
	"github.com/spf13/cobra"
)

func newCrawlCommand(app *application) *cobra.Command {
	var orgTag string
	var isPublic, jsonOutput bool
	command := &cobra.Command{
		Use: "crawl <url>", Short: "抓取公开网页并写入知识库", Args: exactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			if err := validatePublicURL(args[0]); err != nil {
				return err
			}
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			result, err := client.Crawl(commandContext(command), app.requestID, api.CrawlRequest{
				URL: args[0], OrgTag: orgTag, IsPublic: isPublic,
			})
			if err != nil {
				return err
			}
			if app.quiet {
				fmt.Fprintln(command.OutOrStdout(), result.FileMD5)
				return nil
			}
			return output.Write(command.OutOrStdout(), outputMode(runtime.Output, jsonOutput), result)
		},
	}
	command.Flags().StringVar(&orgTag, "org-tag", "", "写入组织标签；默认使用主组织")
	command.Flags().BoolVar(&isPublic, "public", false, "设为公开知识文档")
	command.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func validatePublicURL(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return apperr.New(apperr.ExitArguments, "URL 必须是有效的 http:// 或 https:// 地址")
	}
	return nil
}

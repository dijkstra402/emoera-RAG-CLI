package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/api"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/output"
	"github.com/spf13/cobra"
)

const searchContentPreviewRunes = 100

var fileMD5Pattern = regexp.MustCompile(`^[a-fA-F0-9]{32}$`)

func newSearchCommand(app *application) *cobra.Command {
	options := api.SearchOptions{
		TopK: 10, MinScore: 0.3, IncludeContent: true,
	}
	var files, orgTags []string
	var noContent, jsonOutput bool
	command := &cobra.Command{
		Use: "search <query>", Short: "检索知识库并展示召回依据", Args: exactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			query := strings.TrimSpace(args[0])
			options.TargetFileMD5s = normalizeRepeatedValues(files)
			options.OrgTags = normalizeRepeatedValues(orgTags)
			options.IncludeContent = !noContent
			if err := validateSearchRequest(query, options); err != nil {
				return err
			}
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			result, err := client.Search(commandContext(command), app.requestID, query, options)
			if err != nil {
				return err
			}
			if app.quiet {
				for _, hit := range result.Items {
					fmt.Fprintln(command.OutOrStdout(), searchHitIdentifier(hit))
				}
				return nil
			}
			mode := runtime.Output
			if jsonOutput {
				mode = "json"
			}
			if mode == "table" {
				return writeSearchTable(command, result, options.IncludeExplain)
			}
			return output.Write(command.OutOrStdout(), mode, result)
		},
	}
	flags := command.Flags()
	flags.IntVar(&options.TopK, "top-k", 10, "最多返回数量（1-50）")
	flags.Float64Var(&options.MinScore, "min-score", 0.3, "最低相关度（0-1）")
	flags.StringArrayVar(&orgTags, "org-tag", nil, "限定组织标签，可重复指定")
	flags.StringArrayVar(&files, "file", nil, "限定文件 MD5，可重复指定")
	flags.BoolVar(&options.IncludeExplain, "explain", false, "返回召回与重排依据")
	flags.BoolVar(&noContent, "no-content", false, "不返回 Chunk 正文")
	flags.BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func validateSearchRequest(query string, options api.SearchOptions) error {
	if query == "" {
		return apperr.New(apperr.ExitArguments, "检索内容不能为空")
	}
	if utf8.RuneCountInString(query) > 4000 {
		return apperr.New(apperr.ExitArguments, "检索内容最多 4000 个字符")
	}
	if options.TopK < 1 || options.TopK > 50 {
		return apperr.New(apperr.ExitArguments, "--top-k 必须在 1 到 50 之间")
	}
	if options.MinScore < 0 || options.MinScore > 1 {
		return apperr.New(apperr.ExitArguments, "--min-score 必须在 0 到 1 之间")
	}
	if len(options.OrgTags) > 20 {
		return apperr.New(apperr.ExitArguments, "--org-tag 最多指定 20 个")
	}
	if len(options.TargetFileMD5s) > 50 {
		return apperr.New(apperr.ExitArguments, "--file 最多指定 50 个")
	}
	for _, fileMD5 := range options.TargetFileMD5s {
		if !fileMD5Pattern.MatchString(fileMD5) {
			return apperr.New(apperr.ExitArguments, "--file 必须是 32 位文件 MD5")
		}
	}
	return nil
}

func normalizeRepeatedValues(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func writeSearchTable(command *cobra.Command, result api.SearchResult, includeExplain bool) error {
	headings := "RANK\tSCORE\tFILE\tCHUNK\tORG\tVISIBILITY"
	if includeExplain {
		headings += "\tRETRIEVAL"
	}
	headings += "\tCONTENT"
	fmt.Fprintln(command.OutOrStdout(), headings)
	for _, hit := range result.Items {
		chunk := "-"
		if hit.ChunkID != nil {
			chunk = strconv.Itoa(*hit.ChunkID)
		}
		row := fmt.Sprintf("%d\t%.4f\t%s\t%s\t%s\t%s",
			hit.Rank, hit.Score, sanitizeTableCell(hit.FileName), chunk,
			sanitizeTableCell(hit.OrgTag), hit.Visibility)
		if includeExplain {
			retrieval := "-"
			if hit.Explain != nil && len(hit.Explain.Retrieval) > 0 {
				retrieval = strings.Join(hit.Explain.Retrieval, "+")
			}
			row += "\t" + retrieval
		}
		content := "-"
		if hit.Content != nil {
			content = truncateRunes(sanitizeTableCell(*hit.Content), searchContentPreviewRunes)
		}
		fmt.Fprintf(command.OutOrStdout(), "%s\t%s\n", row, content)
	}
	fmt.Fprintf(command.ErrOrStderr(), "检索完成：%d 条，耗时 %d ms\n", len(result.Items), result.TookMS)
	return nil
}

func searchHitIdentifier(hit api.SearchHit) string {
	if hit.ChunkID == nil {
		return hit.FileMD5
	}
	return hit.FileMD5 + ":" + strconv.Itoa(*hit.ChunkID)
}

func sanitizeTableCell(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func truncateRunes(value string, limit int) string {
	content := []rune(value)
	if len(content) <= limit {
		return value
	}
	return string(content[:limit]) + "…"
}

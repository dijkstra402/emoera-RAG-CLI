package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/api"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/output"
	"github.com/spf13/cobra"
)

func newUsageCommand(app *application) *cobra.Command {
	command := &cobra.Command{Use: "usage", Short: "查看 AI 调用、Token、渠道和错误统计"}
	command.AddCommand(newUsageOverviewCommand(app))
	command.AddCommand(newUsageTokensCommand(app))
	command.AddCommand(newUsageHeatmapCommand(app))
	command.AddCommand(newUsageCallsCommand(app))
	return command
}

func newUsageOverviewCommand(app *application) *cobra.Command {
	var days int
	var jsonOutput bool
	command := &cobra.Command{
		Use: "overview", Short: "显示 AI 调用统计概览", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if err := validateUsageDays(days); err != nil {
				return err
			}
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			result, err := client.UsageOverview(commandContext(command), app.requestID, days)
			if err != nil {
				return err
			}
			if outputMode(runtime.Output, jsonOutput) != "table" {
				return output.Write(command.OutOrStdout(), outputMode(runtime.Output, jsonOutput), result)
			}
			fmt.Fprintf(command.OutOrStdout(), "统计周期：最近 %d 天\n", result.Days)
			fmt.Fprintf(command.OutOrStdout(), "调用：%d（成功 %d / 失败 %d / 错误率 %.2f%%）\n",
				result.TotalCalls, result.SuccessCalls, result.ErrorCalls, result.ErrorRate)
			fmt.Fprintf(command.OutOrStdout(), "平均耗时：%.0f ms\n", result.AverageDurationMS)
			fmt.Fprintf(command.OutOrStdout(), "Token：输入 %d / 输出 %d / 总计 %d\n",
				result.InputTokens, result.OutputTokens, result.TotalTokens)
			return nil
		},
	}
	command.Flags().IntVar(&days, "days", 7, "统计天数（1-365）")
	command.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func newUsageTokensCommand(app *application) *cobra.Command {
	var days int
	var jsonOutput bool
	command := &cobra.Command{
		Use: "tokens", Short: "按日期、模型和渠道查看 Token 用量", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if err := validateUsageDays(days); err != nil {
				return err
			}
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			result, err := client.UsageTokens(commandContext(command), app.requestID, days)
			if err != nil {
				return err
			}
			if outputMode(runtime.Output, jsonOutput) != "table" {
				return output.Write(command.OutOrStdout(), outputMode(runtime.Output, jsonOutput), result)
			}
			fmt.Fprintln(command.OutOrStdout(), "模型用量")
			writeUsageBuckets(command, result.Models, "MODEL")
			fmt.Fprintln(command.OutOrStdout(), "\n渠道用量")
			writeUsageBuckets(command, result.Channels, "CHANNEL")
			fmt.Fprintln(command.OutOrStdout(), "\n每日趋势")
			writeUsageBuckets(command, result.Trend, "DATE")
			return nil
		},
	}
	command.Flags().IntVar(&days, "days", 7, "统计天数（1-365）")
	command.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func newUsageHeatmapCommand(app *application) *cobra.Command {
	var days int
	var jsonOutput bool
	command := &cobra.Command{
		Use: "heatmap", Short: "显示 GitHub 风格每日调用热力图", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if err := validateUsageDays(days); err != nil {
				return err
			}
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			result, err := client.UsageHeatmap(commandContext(command), app.requestID, days)
			if err != nil {
				return err
			}
			if outputMode(runtime.Output, jsonOutput) != "table" {
				return output.Write(command.OutOrStdout(), outputMode(runtime.Output, jsonOutput), result)
			}
			return writeUsageHeatmap(command, result)
		},
	}
	command.Flags().IntVar(&days, "days", 365, "统计天数（1-365）")
	command.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func newUsageCallsCommand(app *application) *cobra.Command {
	options := api.UsageCallOptions{Days: 7, Page: 1, Size: 20}
	var success string
	var jsonOutput bool
	command := &cobra.Command{
		Use: "calls", Short: "筛选并列出 AI 调用明细", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if err := validateUsageDays(options.Days); err != nil {
				return err
			}
			if options.Page < 1 || options.Size < 1 || options.Size > 100 {
				return apperr.New(apperr.ExitArguments, "--page 必须大于 0，--size 必须在 1 到 100 之间")
			}
			parsedSuccess, err := parseSuccessFilter(success)
			if err != nil {
				return err
			}
			options.Success = parsedSuccess
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			result, err := client.UsageCalls(commandContext(command), app.requestID, options)
			if err != nil {
				return err
			}
			if outputMode(runtime.Output, jsonOutput) != "table" {
				return output.Write(command.OutOrStdout(), outputMode(runtime.Output, jsonOutput), result)
			}
			fmt.Fprintln(command.OutOrStdout(), "TIME\tCHANNEL\tSTATUS\tDURATION_MS\tTOKENS\tMODEL\tENDPOINT")
			for _, item := range result.Content {
				fmt.Fprintf(command.OutOrStdout(), "%s\t%s\t%d\t%d\t%s\t%s\t%s\n",
					item.CreatedAt, item.Channel, item.Status, item.DurationMS,
					pointerIntText(item.TotalTokens), pointerText(item.Model), pointerText(item.Endpoint))
			}
			fmt.Fprintf(command.ErrOrStderr(), "第 %d/%d 页，共 %d 条\n", result.Page, result.TotalPages, result.TotalElements)
			return nil
		},
	}
	flags := command.Flags()
	flags.IntVar(&options.Days, "days", 7, "统计天数（1-365）")
	flags.IntVar(&options.Page, "page", 1, "页码")
	flags.IntVar(&options.Size, "size", 20, "每页数量（1-100）")
	flags.StringVar(&options.Endpoint, "endpoint", "", "按接口路径筛选")
	flags.StringVar(&success, "success", "all", "调用结果：all/true/false")
	flags.StringVar(&options.Model, "model", "", "按模型筛选")
	flags.StringVar(&options.Channel, "channel", "", "按渠道筛选：WEB/CLI/BOT/WIDGET/AGENT_API")
	flags.BoolVar(&jsonOutput, "json", false, "输出 JSON")
	return command
}

func validateUsageDays(days int) error {
	if days < 1 || days > 365 {
		return apperr.New(apperr.ExitArguments, "--days 必须在 1 到 365 之间")
	}
	return nil
}

func parseSuccessFilter(value string) (*bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "all":
		return nil, nil
	case "true", "success":
		value := true
		return &value, nil
	case "false", "failed", "failure":
		value := false
		return &value, nil
	default:
		return nil, apperr.New(apperr.ExitArguments, "--success 只能是 all、true 或 false")
	}
}

func writeUsageBuckets(command *cobra.Command, buckets []api.UsageBucket, label string) {
	fmt.Fprintf(command.OutOrStdout(), "%s\tCALLS\tINPUT\tOUTPUT\tTOTAL\n", label)
	for _, item := range buckets {
		name := item.Date
		if item.Model != "" {
			name = item.Model
		}
		if item.Channel != "" {
			name = item.Channel
		}
		fmt.Fprintf(command.OutOrStdout(), "%s\t%d\t%d\t%d\t%d\n",
			sanitizeTableCell(name), item.Calls, item.InputTokens, item.OutputTokens, item.TotalTokens)
	}
}

func writeUsageHeatmap(command *cobra.Command, heatmap api.UsageHeatmap) error {
	if len(heatmap.Cells) == 0 {
		fmt.Fprintln(command.OutOrStdout(), "暂无调用记录")
		return nil
	}
	firstDate, err := time.Parse("2006-01-02", heatmap.Cells[0].Date)
	if err != nil {
		return apperr.Wrap(apperr.ExitServer, "热力图日期格式不兼容", err)
	}
	offset := (int(firstDate.Weekday()) + 6) % 7
	weeks := (offset + len(heatmap.Cells) + 6) / 7
	grid := make([][]int64, 7)
	for row := range grid {
		grid[row] = make([]int64, weeks)
		for column := range grid[row] {
			grid[row][column] = -1
		}
	}
	for index, cell := range heatmap.Cells {
		position := offset + index
		grid[position%7][position/7] = cell.Calls
	}
	fmt.Fprintf(command.OutOrStdout(), "最近 %d 天 · 总调用热力图（空白 → 高）\n", heatmap.Days)
	labels := []string{"一", "二", "三", "四", "五", "六", "日"}
	for row, label := range labels {
		fmt.Fprintf(command.OutOrStdout(), "%s ", label)
		for _, calls := range grid[row] {
			fmt.Fprint(command.OutOrStdout(), heatSymbol(calls, heatmap.MaxCalls))
		}
		fmt.Fprintln(command.OutOrStdout())
	}
	fmt.Fprintf(command.OutOrStdout(), "最高单日调用：%d\n", heatmap.MaxCalls)
	return nil
}

func heatSymbol(calls, maximum int64) string {
	if calls < 0 {
		return "  "
	}
	if calls == 0 || maximum == 0 {
		return "· "
	}
	ratio := float64(calls) / float64(maximum)
	switch {
	case ratio <= .25:
		return "░ "
	case ratio <= .5:
		return "▒ "
	case ratio <= .75:
		return "▓ "
	default:
		return "█ "
	}
}

func pointerIntText(value *int) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprint(*value)
}

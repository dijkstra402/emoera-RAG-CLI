package cmd

import (
	"fmt"
	"sort"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/api"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/output"
	"github.com/spf13/cobra"
)

func newOrgCommand(app *application) *cobra.Command {
	command := &cobra.Command{Use: "org", Short: "查看可访问组织标签"}
	var tree bool
	var jsonOutput bool
	list := &cobra.Command{
		Use: "list", Short: "列出可访问组织标签", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			items, err := client.ListOrgTags(commandContext(command), app.requestID)
			if err != nil {
				return err
			}
			mode := runtime.Output
			if jsonOutput {
				mode = "json"
			}
			if tree && mode == "table" {
				return writeOrgTree(command, items)
			}
			return output.Write(command.OutOrStdout(), mode, items)
		},
	}
	list.Flags().BoolVar(&tree, "tree", false, "按父子关系显示")
	list.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	command.AddCommand(list)
	return command
}

func writeOrgTree(command *cobra.Command, items []api.OrgTag) error {
	children := map[string][]api.OrgTag{}
	known := map[string]bool{}
	for _, item := range items {
		known[item.ID] = true
	}
	for _, item := range items {
		parent := ""
		if item.ParentID != nil && known[*item.ParentID] {
			parent = *item.ParentID
		}
		children[parent] = append(children[parent], item)
	}
	for key := range children {
		sort.Slice(children[key], func(i, j int) bool { return children[key][i].ID < children[key][j].ID })
	}
	visited := map[string]bool{}
	var walk func(string, string)
	walk = func(parent, indent string) {
		for _, item := range children[parent] {
			if visited[item.ID] {
				continue
			}
			visited[item.ID] = true
			fmt.Fprintf(command.OutOrStdout(), "%s%s  %s\n", indent, item.ID, item.Name)
			walk(item.ID, indent+"  ")
		}
	}
	walk("", "")
	// Defensive fallback for malformed cyclic data: render each unreachable node once.
	for _, item := range items {
		if !visited[item.ID] {
			walk(item.ID, "")
			if !visited[item.ID] {
				visited[item.ID] = true
				fmt.Fprintf(command.OutOrStdout(), "%s  %s\n", item.ID, item.Name)
			}
		}
	}
	return nil
}

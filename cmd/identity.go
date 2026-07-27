package cmd

import (
	"github.com/dijkstra402/emoera-RAG-CLI/internal/output"
	"github.com/spf13/cobra"
)

func newWhoAmICommand(app *application) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "whoami", Short: "查看当前账号和 Token 身份", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			var data map[string]any
			if err := client.Get(commandContext(command), "/me", app.requestID, &data); err != nil {
				return err
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

func newCapabilitiesCommand(app *application) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "capabilities", Short: "查看 Agent API 版本和能力限制", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			var data map[string]any
			if err := client.Get(commandContext(command), "/capabilities", app.requestID, &data); err != nil {
				return err
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

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newAuthCommand(app *application) *cobra.Command {
	command := &cobra.Command{Use: "auth", Short: "管理 API Token"}
	command.AddCommand(&cobra.Command{
		Use: "set-token [token]", Short: "将 Token 安全保存到系统 Keychain", Args: maximumArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			profile, err := currentProfile(app)
			if err != nil {
				return err
			}
			token := ""
			if len(args) == 1 {
				token = args[0]
			} else {
				token, err = readSecret(command)
				if err != nil {
					return err
				}
			}
			if !strings.HasPrefix(strings.TrimSpace(token), "em_sk_") {
				return apperr.New(apperr.ExitArguments, "Token 格式无效，应以 em_sk_ 开头")
			}
			if err := app.store.Set(profile, token); err != nil {
				return err
			}
			successMessage(command, "Token 已保存到系统 Keychain（Profile：%s）", profile)
			return nil
		},
	})
	command.AddCommand(&cobra.Command{
		Use: "logout", Short: "删除当前 Profile 的 Keychain Token", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			profile, err := currentProfile(app)
			if err != nil {
				return err
			}
			if err := app.store.Delete(profile); err != nil {
				return err
			}
			successMessage(command, "已退出 Profile：%s", profile)
			return nil
		},
	})
	var jsonOutput bool
	status := &cobra.Command{
		Use: "status", Short: "验证当前 Token", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			client, runtime, err := app.client()
			if err != nil {
				return err
			}
			var identity map[string]any
			if err := client.Get(commandContext(command), "/me", app.requestID, &identity); err != nil {
				return err
			}
			data := map[string]any{
				"authenticated": true, "profile": runtime.Profile,
				"endpoint": runtime.Endpoint, "tokenSource": runtime.TokenSource,
				"identity": identity,
			}
			mode := runtime.Output
			if jsonOutput {
				mode = "json"
			}
			return output.Write(command.OutOrStdout(), mode, data)
		},
	}
	status.Flags().BoolVar(&jsonOutput, "json", false, "输出 JSON")
	command.AddCommand(status)
	return command
}

func currentProfile(app *application) (string, error) {
	_, file, err := app.loadFile()
	if err != nil {
		return "", err
	}
	if app.profile != "" {
		return app.profile, nil
	}
	if value := strings.TrimSpace(os.Getenv("EMOERA_PROFILE")); value != "" {
		return value, nil
	}
	return file.CurrentProfile, nil
}

func readSecret(command *cobra.Command) (string, error) {
	stdin := int(os.Stdin.Fd())
	if term.IsTerminal(stdin) {
		fmt.Fprint(command.ErrOrStderr(), "API Token：")
		content, err := term.ReadPassword(stdin)
		fmt.Fprintln(command.ErrOrStderr())
		if err != nil {
			return "", apperr.Wrap(apperr.ExitConfiguration, "读取 Token 失败", err)
		}
		return strings.TrimSpace(string(content)), nil
	}
	content, err := bufio.NewReader(command.InOrStdin()).ReadString('\n')
	if err != nil && strings.TrimSpace(content) == "" {
		return "", apperr.Wrap(apperr.ExitConfiguration, "从标准输入读取 Token 失败", err)
	}
	return strings.TrimSpace(content), nil
}

package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/api"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/auth"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/config"
	"github.com/spf13/cobra"
)

// version 会在 Release 构建时通过 -ldflags 注入；go install 构建则读取模块版本。
var version = "dev"

func resolvedVersion() string {
	if version != "" && version != "dev" {
		return strings.TrimPrefix(version, "v")
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return "dev"
}

type application struct {
	profile   string
	endpoint  string
	token     string
	output    string
	noColor   bool
	quiet     bool
	timeout   time.Duration
	requestID string
	debug     bool
	store     auth.Store
}

func Execute() error {
	root := newRootCommand(auth.KeyringStore{})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	root.SilenceErrors = true
	root.SilenceUsage = true
	return root.Execute()
}

func newRootCommand(store auth.Store) *cobra.Command {
	app := &application{store: store}
	buildVersion := resolvedVersion()
	root := &cobra.Command{
		Use:           "emoera",
		Short:         "E时代 RAG 知识库 Agent CLI",
		Version:       buildVersion,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return apperr.Wrap(apperr.ExitArguments, "命令参数错误", err)
	})
	flags := root.PersistentFlags()
	flags.StringVar(&app.profile, "profile", "", "配置环境名称")
	flags.StringVar(&app.endpoint, "endpoint", "", "临时覆盖服务地址")
	flags.StringVar(&app.token, "token", "", "临时 API Token（建议改用环境变量）")
	flags.StringVarP(&app.output, "output", "o", "", "输出格式：table、json 或 jsonl")
	flags.BoolVar(&app.noColor, "no-color", false, "禁用彩色输出")
	flags.BoolVarP(&app.quiet, "quiet", "q", false, "只输出核心结果")
	flags.DurationVar(&app.timeout, "timeout", 0, "请求超时，例如 30s")
	flags.StringVar(&app.requestID, "request-id", "", "指定请求 ID")
	flags.BoolVar(&app.debug, "debug", false, "输出已脱敏的请求诊断")

	root.AddCommand(newConfigCommand(app))
	root.AddCommand(newAuthCommand(app))
	root.AddCommand(newWhoAmICommand(app))
	root.AddCommand(newCapabilitiesCommand(app))
	root.AddCommand(newOrgCommand(app))
	root.AddCommand(newDocumentCommand(app))
	root.AddCommand(newSearchCommand(app))
	root.AddCommand(newAskCommand(app))
	root.AddCommand(newChatCommand(app))
	root.AddCommand(newModelCommand(app))
	root.AddCommand(newQuotaCommand(app))
	root.AddCommand(newStatusCommand(app))
	root.AddCommand(newPromptCommand(app))
	root.AddCommand(newCrawlCommand(app))
	root.AddCommand(newUsageCommand(app))
	return root
}

func noArgs(command *cobra.Command, args []string) error {
	return argumentError(cobra.NoArgs(command, args))
}

func exactArgs(count int) cobra.PositionalArgs {
	return func(command *cobra.Command, args []string) error {
		return argumentError(cobra.ExactArgs(count)(command, args))
	}
}

func maximumArgs(count int) cobra.PositionalArgs {
	return func(command *cobra.Command, args []string) error {
		return argumentError(cobra.MaximumNArgs(count)(command, args))
	}
}

func argumentError(err error) error {
	if err == nil {
		return nil
	}
	return apperr.Wrap(apperr.ExitArguments, "命令参数错误", err)
}

func (a *application) loadFile() (string, config.File, error) {
	path, err := config.Path()
	if err != nil {
		return "", config.File{}, err
	}
	file, err := config.Load(path)
	return path, file, err
}

func (a *application) runtime() (config.Runtime, error) {
	path, file, err := a.loadFile()
	if err != nil {
		return config.Runtime{}, err
	}
	profile := a.profile
	if profile == "" {
		profile = os.Getenv("EMOERA_PROFILE")
	}
	if profile == "" {
		profile = file.CurrentProfile
	}
	keychainToken := ""
	if a.token == "" && os.Getenv("EMOERA_API_TOKEN") == "" {
		keychainToken, err = a.store.Get(profile)
		if err != nil {
			return config.Runtime{}, err
		}
	}
	return config.Resolve(file, path, config.Overrides{
		Profile: a.profile, Endpoint: a.endpoint, Token: a.token,
		Output: a.output, Timeout: a.timeout,
	}, keychainToken)
}

func (a *application) client() (*api.Client, config.Runtime, error) {
	runtime, err := a.runtime()
	if err != nil {
		return nil, config.Runtime{}, err
	}
	client, err := api.New(runtime.Endpoint, runtime.Token, "emoera-cli/"+resolvedVersion(), runtime.Timeout)
	return client, runtime, err
}

func commandContext(command *cobra.Command) context.Context {
	if command.Context() != nil {
		return command.Context()
	}
	return context.Background()
}

func successMessage(command *cobra.Command, format string, values ...any) {
	fmt.Fprintf(command.OutOrStdout(), format+"\n", values...)
}

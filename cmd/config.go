package cmd

import (
	"sort"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/config"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/output"
	"github.com/spf13/cobra"
)

func newConfigCommand(app *application) *cobra.Command {
	command := &cobra.Command{Use: "config", Short: "管理 CLI Profile 配置"}
	command.AddCommand(&cobra.Command{
		Use: "init", Short: "初始化配置文件", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			path, err := config.Path()
			if err != nil {
				return err
			}
			if err := config.Save(path, config.DefaultFile()); err != nil {
				return err
			}
			successMessage(command, "已初始化配置：%s", path)
			return nil
		},
	})
	command.AddCommand(&cobra.Command{
		Use: "set <key> <value>", Short: "设置当前或指定 Profile", Args: exactArgs(2),
		RunE: func(command *cobra.Command, args []string) error {
			path, file, err := app.loadFile()
			if err != nil {
				return err
			}
			if err := config.Set(&file, app.profile, args[0], args[1]); err != nil {
				return err
			}
			if err := config.Save(path, file); err != nil {
				return err
			}
			successMessage(command, "配置已更新")
			return nil
		},
	})
	command.AddCommand(&cobra.Command{
		Use: "use <profile>", Short: "切换 Profile（不存在时创建）", Args: exactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			path, file, err := app.loadFile()
			if err != nil {
				return err
			}
			if err := config.Use(&file, args[0]); err != nil {
				return err
			}
			if err := config.Save(path, file); err != nil {
				return err
			}
			successMessage(command, "当前 Profile：%s", file.CurrentProfile)
			return nil
		},
	})
	command.AddCommand(&cobra.Command{
		Use: "list", Short: "列出 Profile（不显示 Token）", Args: noArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			path, file, err := app.loadFile()
			if err != nil {
				return err
			}
			items := make([]map[string]any, 0, len(file.Profiles))
			names := make([]string, 0, len(file.Profiles))
			for name := range file.Profiles {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				profile := file.Profiles[name]
				items = append(items, map[string]any{
					"name": name, "current": name == file.CurrentProfile,
					"endpoint": profile.Endpoint, "defaultOutput": profile.DefaultOutput,
				})
			}
			mode := app.output
			if mode == "" {
				mode = "table"
			}
			if err := output.Write(command.OutOrStdout(), mode, items); err != nil {
				return err
			}
			if mode == "table" {
				successMessage(command, "配置文件：%s", path)
			}
			return nil
		},
	})
	return command
}

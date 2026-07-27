# Contributing

感谢你参与 Emoera Agent CLI。

## 开发环境

- Go 1.22 或更高版本
- macOS、Linux 或 Windows
- 用于联调的 Emoera Agent API Token（测试代码不应依赖真实 Token）

## 本地检查

提交前请运行：

```bash
gofmt -w $(find . -name '*.go' -type f)
go test -race ./...
go vet ./...
go build ./...
```

新增命令或请求行为时，请同时补充单元测试和 README 示例。测试不得访问生产环境，也不得提交 API Token、私钥、手机号或其他敏感信息。

## Pull Request

1. 从 `main` 创建功能分支。
2. 保持提交内容单一、说明清晰。
3. 确保 CI 全部通过。
4. 在 PR 中说明行为变化、测试方式和兼容性影响。

提交贡献即表示你同意按本仓库 Apache License 2.0 授权相应内容。

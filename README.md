# Emoera Agent CLI

[![CI](https://github.com/dijkstra402/emoera-RAG-CLI/actions/workflows/ci.yml/badge.svg)](https://github.com/dijkstra402/emoera-RAG-CLI/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/dijkstra402/emoera-RAG-CLI?include_prereleases)](https://github.com/dijkstra402/emoera-RAG-CLI/releases)
[![License](https://img.shields.io/github/license/dijkstra402/emoera-RAG-CLI)](LICENSE)

`emoera` 是 E时代 RAG 知识库面向开发者和 Agent 的命令行客户端。它支持安全保存 Agent API Token、文档管理、混合检索、流式 RAG 问答、会话查询、模型与配额查看，既适合人工使用，也适合脚本和自动化 Agent 调用。

## 安装

从 [GitHub Releases](https://github.com/dijkstra402/emoera-RAG-CLI/releases/latest) 下载与你的系统匹配的压缩包：

| 系统 | 芯片 | 文件 |
| --- | --- | --- |
| macOS | Apple Silicon | `emoera-cli_*_darwin_arm64.tar.gz` |
| macOS | Intel | `emoera-cli_*_darwin_amd64.tar.gz` |
| Linux | x86_64 | `emoera-cli_*_linux_amd64.tar.gz` |
| Linux | ARM64 | `emoera-cli_*_linux_arm64.tar.gz` |
| Windows | x86_64 | `emoera-cli_*_windows_amd64.zip` |

macOS / Linux：

```bash
tar -xzf emoera-cli_*_darwin_arm64.tar.gz
sudo install -m 0755 emoera-cli_*/emoera /usr/local/bin/emoera
emoera --version
```

Windows：解压后将 `emoera.exe` 所在目录加入 `PATH`，然后运行：

```powershell
emoera --version
```

也可以使用 Go 直接安装最新源码：

```bash
go install github.com/dijkstra402/emoera-RAG-CLI@latest
```

## 快速开始

先在 E时代 RAG 知识库的“个人中心 → Agent API”创建 Token。Token 只会在创建时完整显示一次。

```bash
# 初始化默认配置，线上地址默认为 https://rag.emoera.cn
emoera config init

# 交互式输入 Token，并安全保存到系统 Keychain
emoera auth set-token

# 检查身份与服务能力
emoera auth status
emoera whoami
emoera capabilities

# 发起知识库问答
emoera ask "孙梦磊是谁"
```

自建部署可修改服务地址：

```bash
emoera config set endpoint https://your-rag.example.com
```

CI 或容器环境可使用环境变量，不必把 Token 写入脚本或配置文件：

```bash
export EMOERA_ENDPOINT="https://rag.emoera.cn"
export EMOERA_API_TOKEN="em_sk_xxx"
emoera ask "总结最新上传的技术文档" --json
```

## 常用命令

```bash
# 配置与认证
emoera config list
emoera config use production
emoera auth status
emoera auth logout

# 组织与文档
emoera org list
emoera doc list
emoera doc upload ./manual.pdf
emoera doc show <file-md5>
emoera doc status <file-md5>

# 可解释检索
emoera search "设备监控架构" --top-k 10 --min-score 0.3 --explain

# 流式问答、指定模型与限定文档
emoera ask "比较两份方案的差异" --model "Qwen3 8B" \
  --file <file-md5-1> --file <file-md5-2> --suggestions

# 适合 Agent 管道处理的结构化输出
emoera ask "列出风险" --json
emoera ask "列出风险" --jsonl
emoera search "部署规范" --json

# 会话、运行状态、模型和配额
emoera chat sessions
emoera chat messages <session-uuid>
emoera chat run <request-id>
emoera chat cancel <request-id>
emoera model list
emoera quota show
emoera status
```

使用 `emoera <command> --help` 查看完整参数。全局支持 `--profile`、`--endpoint`、`--output`、`--timeout`、`--request-id`、`--quiet`、`--no-color` 和 `--debug`。

## 多环境 Profile

```bash
emoera config use staging
emoera config set endpoint https://staging.example.com
emoera auth set-token

emoera config use production
emoera config set endpoint https://rag.emoera.cn
emoera auth set-token

emoera --profile staging ask "测试问题"
```

不同 Profile 的 Token 分别保存在系统 Keychain 中，配置文件只保存服务地址和默认输出格式。

## 校验下载文件

每个 Release 都附带 `SHA256SUMS`。下载压缩包与校验文件后执行：

```bash
sha256sum -c SHA256SUMS --ignore-missing      # Linux
shasum -a 256 -c SHA256SUMS                   # macOS
```

Release 还包含 CycloneDX SBOM、Sigstore 签名 bundle 和 GitHub 构建来源证明。

## 从源码开发

需要 Go 1.22 或更高版本：

```bash
git clone https://github.com/dijkstra402/emoera-RAG-CLI.git
cd emoera-RAG-CLI
go test -race ./...
go vet ./...
go build -o bin/emoera .
```

贡献前请阅读 [CONTRIBUTING.md](CONTRIBUTING.md)，安全问题请按 [SECURITY.md](SECURITY.md) 私下报告。

## License

Apache License 2.0，详见 [LICENSE](LICENSE)。

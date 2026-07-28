# Emoera Agent CLI

[![CI](https://github.com/dijkstra402/emoera-RAG-CLI/actions/workflows/ci.yml/badge.svg)](https://github.com/dijkstra402/emoera-RAG-CLI/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/dijkstra402/emoera-RAG-CLI?include_prereleases)](https://github.com/dijkstra402/emoera-RAG-CLI/releases)
[![Docs](https://img.shields.io/badge/docs-online-d9342b)](https://dijkstra402.github.io/emoera-RAG-CLI/)
[![License](https://img.shields.io/github/license/dijkstra402/emoera-RAG-CLI)](LICENSE)

`emoera-CLI` 是 E时代 RAG 知识库面向开发者和 Agent 的命令行客户端。它支持安全保存 Agent API Token、文档管理、混合检索、流式 RAG 问答、会话查询、模型与配额查看，既适合人工使用，也适合脚本和自动化 Agent 调用。

完整文档：[dijkstra402.github.io/emoera-RAG-CLI](https://dijkstra402.github.io/emoera-RAG-CLI/)。文档站基于 Cloudflare 官方 [Nimbus](https://github.com/cloudflare/nimbus) 架构构建，提供中英文内容、全文搜索、移动端适配和面向 Agent 的 `llms.txt`。

## 能力概览

- **安全认证**：Agent API Token 默认保存到系统 Keychain，不写入普通配置文件。
- **RAG 问答**：支持流式输出、指定模型、限定文档、追问建议与会话续问。
- **可解释检索**：支持 `top-k`、最低分数、召回详情和结构化结果。
- **文档自动化**：上传、列表、详情、解析状态查询均可通过脚本完成。
- **Agent 友好**：提供 table、JSON、JSONL 输出，明确退出码和请求 ID。
- **多环境隔离**：开发、预发、生产 Profile 分别保存地址、Token 与默认参数。

## 安装

### 一行命令安装（推荐）

macOS / Linux：

```bash
curl -fsSL https://raw.githubusercontent.com/dijkstra402/emoera-RAG-CLI/main/install.sh | sh
```

Windows PowerShell：

```powershell
irm https://raw.githubusercontent.com/dijkstra402/emoera-RAG-CLI/main/install.ps1 | iex
```

安装脚本会自动识别系统和 CPU，下载最新 Release，校验 `SHA256SUMS`，并安装到当前用户目录。也可以继续使用下面的图形安装包。

从 [GitHub Releases](https://github.com/dijkstra402/emoera-RAG-CLI/releases/latest) 下载与你的系统匹配的压缩包：

| 系统 | 推荐安装包 | 免安装包 |
| --- | --- | --- |
| macOS Apple Silicon | `emoera-cli_*_darwin_arm64.pkg` | `emoera-cli_*_darwin_arm64.tar.gz` |
| macOS Intel | `emoera-cli_*_darwin_amd64.pkg` | `emoera-cli_*_darwin_amd64.tar.gz` |
| Debian / Ubuntu x86_64 | `emoera-cli_*_linux_amd64.deb` | `emoera-cli_*_linux_amd64.tar.gz` |
| Debian / Ubuntu ARM64 | `emoera-cli_*_linux_arm64.deb` | `emoera-cli_*_linux_arm64.tar.gz` |
| Fedora / RHEL / Rocky | `emoera-cli_*_linux_*.rpm` | 对应 Linux `tar.gz` |
| Windows x86_64 | `emoera-cli_*_windows_amd64-setup.exe` | `emoera-cli_*_windows_amd64.zip` |

### macOS 图形安装包（推荐）

macOS 用户可以直接下载对应芯片的 `.pkg`：

- Apple Silicon：`emoera-cli_*_darwin_arm64.pkg`
- Intel：`emoera-cli_*_darwin_amd64.pkg`

双击安装包，按“继续 → 安装”完成安装，然后打开终端运行：

```bash
emoera --version
```

当前安装包尚未使用 Apple Developer ID 签名。如果 macOS 阻止打开，请在“系统设置 → 隐私与安全性”中选择“仍要打开”。后续配置和升级不会影响已经保存在系统 Keychain 中的 Token。

### Debian / Ubuntu 安装包

```bash
sudo apt install ./emoera-cli_*_linux_amd64.deb
emoera --version
```

### Fedora / RHEL / Rocky Linux 安装包

```bash
sudo dnf install ./emoera-cli_*_linux_amd64.rpm
emoera --version
```

### Windows 图形安装包

运行 `emoera-cli_*_windows_amd64-setup.exe`，安装器会为当前用户安装并更新 `PATH`。安装后重新打开 PowerShell：

```powershell
emoera --version
```

当前社区安装包未做商业代码签名。若 SmartScreen 提示未知发布者，请先核对 Release 来源与 SHA-256，再选择“更多信息 → 仍要运行”。

### 免安装压缩包

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
go install github.com/dijkstra402/emoera-RAG-CLI/cmd/emoera@latest
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
emoera ask "介绍一下知识库中的主要内容"
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

## 命令总览

| 命令 | 用途 | 示例 |
| --- | --- | --- |
| `config` | 初始化和管理 Profile | `emoera config list` |
| `auth` | 保存、检查或删除 Token | `emoera auth status` |
| `whoami` | 查看 Token 对应身份 | `emoera whoami --json` |
| `capabilities` | 查看服务端开放能力 | `emoera capabilities` |
| `org` | 查询可访问组织 | `emoera org list` |
| `doc` | 文档上传、列表和状态 | `emoera doc upload ./manual.pdf` |
| `search` | 权限内混合检索 | `emoera search "部署规范" --explain` |
| `ask` | 发起流式 RAG 问答 | `emoera ask "有哪些风险？"` |
| `chat` | 查询会话、运行或取消请求 | `emoera chat sessions` |
| `model` | 查看可用模型与倍率 | `emoera model list` |
| `quota` | 查看当前用量与配额 | `emoera quota show` |
| `status` | 查看服务状态 | `emoera status` |

### 常用组合

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

## Agent 与自动化接入

建议自动化程序使用 JSON/JSONL，固定传入请求 ID，并同时判断退出码与错误字段：

```bash
emoera --request-id "deploy-audit-20260727-001" \
  ask "检查发布风险并列出对应措施" --json --no-color
```

配置优先级从高到低为：命令行参数 → 环境变量 → 当前 Profile → 内置默认值。无桌面 Keychain 的 CI 或容器环境可安全注入变量：

```bash
export EMOERA_ENDPOINT="https://rag.emoera.cn"
export EMOERA_API_TOKEN="em_sk_xxx"
emoera search "发布回滚" --json
```

### 退出码

| 退出码 | 含义 |
| --- | --- |
| `0` | 请求成功 |
| `1` | 一般运行错误 |
| `2` | 参数或配置错误 |
| `3` | 未认证或 Token 无效 |
| `4` | 无权限访问资源 |
| `5` | 资源不存在 |
| `6` | 超过速率或用量限制 |
| `7` | 服务暂不可用或网络失败 |

## 校验下载文件

每个 Release 都附带 `SHA256SUMS`。下载压缩包与校验文件后执行：

```bash
sha256sum -c SHA256SUMS --ignore-missing      # Linux
shasum -a 256 -c SHA256SUMS                   # macOS
```

Release 还包含 CycloneDX SBOM、Sigstore 签名 bundle 和 GitHub 构建来源证明。

## 升级、卸载与排障

- macOS PKG：安装新版本即可覆盖；卸载时删除 `/usr/local/bin/emoera`。
- Debian / Ubuntu：`sudo apt remove emoera-cli`。
- Fedora / RHEL：`sudo dnf remove emoera-cli`。
- Windows：在“设置 → 应用 → 已安装的应用”中卸载 Emoera Agent CLI。
- Go 安装：重新执行 `go install ...@latest` 升级，删除 `$GOBIN/emoera` 卸载。

常用诊断顺序：

```bash
emoera auth status
emoera capabilities
emoera status
emoera --debug ask "连接测试"
```

- **命令不存在**：重新打开终端，确认安装目录已加入 `PATH`。
- **401 / Token 无效**：重新设置 Token，确认它未撤销或过期。
- **403 / 无权限**：检查 Token Scope、组织权限与文档可见范围。
- **请求超时**：适当增大 `--timeout`，并检查状态页和网络代理。
- **Linux 无 Keychain**：服务器或容器中使用 `EMOERA_API_TOKEN` 环境变量。

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

export type DocsLocale = 'zh' | 'en';

const zh = [
  {
    label: '开始使用',
    items: [
      { href: '/', label: '概览', description: '认识 Emoera Agent CLI', keywords: '终端 cli 开源 agent api 知识库' },
      { href: '/install', label: '安装', description: '三大平台安装与升级', keywords: 'macos windows linux pkg deb rpm 安装包 path 升级' },
      { href: '/quickstart', label: '快速开始', description: '创建 Token 并完成首个问答', keywords: '配置 login whoami ask 第一次问答' },
      { href: '/authentication', label: '认证与安全', description: 'Token、Scope 与 Keychain', keywords: '凭证 密钥 环境变量 401 403 credential keychain scope' }
    ]
  },
  {
    label: '使用指南',
    items: [
      { href: '/commands', label: '命令参考', description: '完整命令与常用参数', keywords: 'doc search ask model quota profile output json jsonl timeout' },
      { href: '/automation', label: 'Agent 自动化', description: 'JSON、幂等与 CI 集成', keywords: '流水线 脚本 request-id exit code 退出码 cicd' },
      { href: '/troubleshooting', label: '故障排查', description: '错误码与诊断流程', keywords: '错误 401 403 eof 网络 超时 debug 诊断' },
      { href: '/release-security', label: '发布安全', description: 'Checksum、SBOM 与签名', keywords: 'sha256 sigstore provenance checksum sbom 校验和 供应链' }
    ]
  }
] as const;

const en = [
  {
    label: 'Get started',
    items: [
      { href: '/en', label: 'Overview', description: 'Meet Emoera Agent CLI', keywords: 'terminal open source agent api knowledge base' },
      { href: '/en/install', label: 'Install', description: 'Install and update on three platforms', keywords: 'macos windows linux pkg deb rpm package path upgrade' },
      { href: '/en/quickstart', label: 'Quickstart', description: 'Create a token and ask your first question', keywords: 'configure login whoami ask first query' },
      { href: '/en/authentication', label: 'Authentication', description: 'Tokens, scopes, and secure storage', keywords: 'credential key secret environment 401 403 keychain scope' }
    ]
  },
  {
    label: 'Guides',
    items: [
      { href: '/en/commands', label: 'Command reference', description: 'Commands, flags, and output formats', keywords: 'doc search ask model quota profile output json jsonl timeout' },
      { href: '/en/automation', label: 'Agent automation', description: 'JSON, idempotency, and CI workflows', keywords: 'pipeline script request-id exit code cicd' },
      { href: '/en/troubleshooting', label: 'Troubleshooting', description: 'Errors and diagnostic workflow', keywords: 'error 401 403 eof network timeout debug diagnose' },
      { href: '/en/release-security', label: 'Release security', description: 'Checksums, SBOMs, and signatures', keywords: 'sha256 sigstore provenance checksum sbom supply chain' }
    ]
  }
] as const;

export const navGroups = { zh, en } as const;

export function getNavGroups(locale: DocsLocale) {
  return navGroups[locale];
}

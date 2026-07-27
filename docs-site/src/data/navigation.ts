export const navGroups = [
  {
    label: '开始使用',
    items: [
      { href: '/', label: '概览', description: '认识 Emoera Agent CLI' },
      { href: '/install', label: '安装', description: '三大平台安装与升级' },
      { href: '/quickstart', label: '快速开始', description: '创建 Token 并完成首个问答' },
      { href: '/authentication', label: '认证与安全', description: 'Token、Scope 与 Keychain' }
    ]
  },
  {
    label: '使用指南',
    items: [
      { href: '/commands', label: '命令参考', description: '完整命令与常用参数' },
      { href: '/automation', label: 'Agent 自动化', description: 'JSON、幂等与 CI 集成' },
      { href: '/troubleshooting', label: '故障排查', description: '错误码与诊断流程' },
      { href: '/release-security', label: '发布安全', description: 'Checksum、SBOM 与签名' }
    ]
  }
] as const;

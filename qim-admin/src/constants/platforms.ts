// 客户端平台常量（与后端 service.NormalizePlatform / 客户端 detectPlatform 的输出对齐）。
// 单一来源：SystemMessages 目标平台下拉、SystemConfig 最低发消息版本门槛字段共用；
// 新增平台（如 android/web）只需改这里。

export interface ClientPlatform {
  value: string
  label: string
}

// 含"所有平台"空值，供需要全局选项的场景（如门槛"通用"）
export const CLIENT_PLATFORMS: ClientPlatform[] = [
  { value: '', label: '所有平台' },
  { value: 'windows', label: 'Windows' },
  { value: 'macos', label: 'macOS' },
  { value: 'linux', label: 'Linux' },
]

// 可选平台（不含空值"所有平台"），用于目标平台下拉等仅列具体平台的场景
export const CLIENT_PLATFORM_OPTIONS: ClientPlatform[] = CLIENT_PLATFORMS.filter(p => p.value)

export const clientPlatformLabel = (value: string): string =>
  CLIENT_PLATFORMS.find(p => p.value === value)?.label || value

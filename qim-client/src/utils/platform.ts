// 平台检测：WS 连接上报、REST 版本头、更新检查共用同一口径。
// 与后端 service.NormalizePlatform 的输入约定一致（macos/linux/windows）。
export const detectPlatform = (): string => {
  const ua = navigator.userAgent.toLowerCase()
  return ua.includes('mac') ? 'macos'
    : ua.includes('linux') ? 'linux'
    : 'windows'
}

// 判断是否为完整的 URL（http/https/data URI）
export const isAbsoluteUrl = (url: string): boolean => {
  return url.startsWith('http') || url.startsWith('data:') || url.startsWith('blob:')
}

// 头像背景颜色池
export const AVATAR_COLORS = [
  '#4285F4', '#EA4335', '#FBBC05', '#34A853',
  '#FF6D01', '#46BDC6', '#7B1FA2', '#C2185B',
  '#673AB7', '#00BCD4', '#8BC34A', '#FF9800'
]

// 根据名称获取头像颜色
export const getAvatarColor = (name: string): string => {
  const charCode = name.charCodeAt(0)
  return AVATAR_COLORS[charCode % AVATAR_COLORS.length]
}

// 获取名称的首字符
export const getInitial = (name: string): string => {
  return name.charAt(0).toUpperCase()
}

// Canvas 头像缓存
const avatarCache = new Map<string, string>()

// 生成默认头像 URL（使用 Canvas 生成首字母彩色头像，带缓存优化）
export const generateAvatar = (name: string): string => {
  if (avatarCache.has(name)) {
    return avatarCache.get(name)!
  }

  const bgColor = getAvatarColor(name)
  const displayName = getInitial(name)

  const canvas = document.createElement('canvas')
  canvas.width = 100
  canvas.height = 100
  const ctx = canvas.getContext('2d')

  if (ctx) {
    ctx.fillStyle = bgColor
    ctx.beginPath()
    ctx.arc(50, 50, 50, 0, 2 * Math.PI)
    ctx.fill()
    ctx.fillStyle = '#fff'
    ctx.font = 'bold 40px Arial'
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    ctx.fillText(displayName, 50, 50)
  }

  const dataUrl = canvas.toDataURL()
  avatarCache.set(name, dataUrl)
  return dataUrl
}

// 处理头像 URL
export const getDisplayName = (user: { nickname?: string; username?: string; name?: string } | undefined | null): string => {
  if (!user) return '未知用户'
  return user.nickname || user.username || user.name || '未知用户'
}

export const getAvatarUrl = (avatar: string | undefined | null, name: string, serverUrl: string): string => {
  if (avatar && avatar.trim() && isAbsoluteUrl(avatar)) {
    return avatar
  }
  if (avatar && avatar.trim()) {
    const cleanServerUrl = serverUrl.replace(/\/$/, '')
    const cleanAvatar = avatar.replace(/^\//, '')
    return `${cleanServerUrl}/${cleanAvatar}`
  }
  return generateAvatar(name || '用户')
}

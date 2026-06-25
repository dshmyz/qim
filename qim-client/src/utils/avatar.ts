// 判断是否为完整的 URL（http/https/data URI）
export const isAbsoluteUrl = (url: string): boolean => {
  return url.startsWith('http') || url.startsWith('data:') || url.startsWith('blob:')
}

// 头像背景颜色池（24色，覆盖更多色相，减少撞色概率）
export const AVATAR_COLORS = [
  '#4285F4', '#EA4335', '#FBBC05', '#34A853',
  '#FF6D01', '#46BDC6', '#7B1FA2', '#C2185B',
  '#673AB7', '#00BCD4', '#8BC34A', '#FF9800',
  '#E91E63', '#2196F3', '#009688', '#FF5722',
  '#607D8B', '#795548', '#F44336', '#3F51B5',
  '#00ACC1', '#8D6E63', '#546E7A', '#26A69A'
]

// 根据名称获取头像颜色（对整个名字做 hash，避免同姓撞色）
export const getAvatarColor = (name: string): string => {
  let hash = 0
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash)
  }
  return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length]
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
    // 绘制填满整个画布的矩形背景，避免透明缝隙
    ctx.fillStyle = bgColor
    ctx.fillRect(0, 0, 100, 100)
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

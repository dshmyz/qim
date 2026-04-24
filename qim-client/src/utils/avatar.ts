// 生成默认头像 URL
export const generateAvatar = (name: string): string => {
  // 使用 DiceBear API 生成头像
  return `https://api.dicebear.com/7.x/identicon/svg?seed=${encodeURIComponent(name)}`
}

// 处理头像 URL
export const getAvatarUrl = (avatar: string | undefined | null, name: string, serverUrl: string): string => {
  let avatarUrl: string
  
  if (avatar && avatar.trim() && avatar.startsWith('http')) {
    avatarUrl = avatar
  } else if (avatar && avatar.trim()) {
    // 确保serverUrl末尾没有斜杠，avatar开头没有斜杠
    const cleanServerUrl = serverUrl.replace(/\/$/, '')
    const cleanAvatar = avatar.replace(/^\//, '')
    avatarUrl = `${cleanServerUrl}/${cleanAvatar}`
  } else {
    avatarUrl = generateAvatar(name || '用户')
  }
  
  // 尝试缓存头像（仅在 Electron 环境中）
  if (window.electron && window.electron.ipcRenderer) {
    window.electron.ipcRenderer.send('cache-avatar', avatarUrl)
  }
  
  return avatarUrl
}

// 获取缓存的头像 URL
export const getCachedAvatarUrl = (avatarUrl: string): Promise<string> => {
  return new Promise((resolve) => {
    if (window.electron && window.electron.ipcRenderer) {
      // 发送缓存请求
      window.electron.ipcRenderer.send('cache-avatar', avatarUrl)
      
      // 监听响应
      const listener = (event: any, cachedUrl: string) => {
        window.electron.ipcRenderer.once('avatar-cached', listener)
        resolve(cachedUrl)
      }
      
      window.electron.ipcRenderer.once('avatar-cached', listener)
    } else {
      // 非 Electron 环境直接返回原始 URL
      resolve(avatarUrl)
    }
  })
}

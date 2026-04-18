// 生成默认头像 URL
export const generateAvatar = (name: string): string => {
  // 使用 DiceBear API 生成头像
  return `https://api.dicebear.com/7.x/identicon/svg?seed=${encodeURIComponent(name)}`
}

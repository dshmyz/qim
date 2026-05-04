/**
 * 日期格式化工具函数
 */

/**
 * 格式化日期为友好显示格式
 * @param date - 日期对象或字符串
 * @param includeYear - 是否总是包含年份
 * @returns 格式化后的日期字符串
 */
export function formatDate(date: Date | string | null | undefined, includeYear = false): string {
  if (!date) return '-'
  
  const d = typeof date === 'string' ? new Date(date) : date
  if (isNaN(d.getTime())) return '-'
  
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)
  
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (hours < 24) return `${hours}小时前`
  if (days < 7) return `${days}天前`
  
  const year = includeYear || d.getFullYear() !== now.getFullYear() ? `${d.getFullYear()}-` : ''
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const hour = String(d.getHours()).padStart(2, '0')
  const minute = String(d.getMinutes()).padStart(2, '0')
  
  return `${year}${month}-${day} ${hour}:${minute}`
}

/**
 * 文件类型工具函数
 */

/**
 * 判断是否为图片类型
 */
export function isImage(mimeType?: string): boolean {
  return !!mimeType && mimeType.startsWith('image/')
}

/**
 * 判断是否为视频类型
 */
export function isVideo(mimeType?: string): boolean {
  return !!mimeType && mimeType.startsWith('video/')
}

/**
 * 判断是否为音频类型
 */
export function isAudio(mimeType?: string): boolean {
  return !!mimeType && mimeType.startsWith('audio/')
}

/**
 * 判断是否为文档类型
 */
export function isDocument(mimeType?: string): boolean {
  if (!mimeType) return false
  return (
    mimeType.includes('pdf') ||
    mimeType.includes('word') ||
    mimeType.includes('document') ||
    mimeType.includes('excel') ||
    mimeType.includes('sheet') ||
    mimeType.includes('powerpoint') ||
    mimeType.includes('presentation') ||
    mimeType.startsWith('text/')
  )
}

/**
 * 获取文件图标类名
 */
export function getFileIcon(mimeType?: string): string {
  if (!mimeType) return 'fas fa-file'
  if (isImage(mimeType)) return 'fas fa-image'
  if (isVideo(mimeType)) return 'fas fa-video'
  if (isAudio(mimeType)) return 'fas fa-music'
  if (mimeType.includes('pdf')) return 'fas fa-file-pdf'
  if (mimeType.includes('word') || mimeType.includes('document')) return 'fas fa-file-word'
  if (mimeType.includes('excel') || mimeType.includes('sheet')) return 'fas fa-file-excel'
  if (mimeType.includes('powerpoint') || mimeType.includes('presentation'))
    return 'fas fa-file-powerpoint'
  if (mimeType.startsWith('text/')) return 'fas fa-file-alt'
  if (mimeType.includes('zip') || mimeType.includes('rar') || mimeType.includes('7z'))
    return 'fas fa-file-archive'
  return 'fas fa-file'
}

/**
 * 获取文件图标颜色
 */
export function getFileIconColor(mimeType?: string): string {
  if (!mimeType) return 'var(--text-secondary)'
  if (isImage(mimeType)) return 'var(--color-success-500)'
  if (isVideo(mimeType)) return 'var(--color-error-500)'
  if (isAudio(mimeType)) return 'var(--color-warning-500)'
  if (mimeType.includes('pdf')) return '#e74c3c'
  if (mimeType.includes('word') || mimeType.includes('document')) return '#2b579a'
  if (mimeType.includes('excel') || mimeType.includes('sheet')) return '#217346'
  if (mimeType.includes('powerpoint') || mimeType.includes('presentation')) return '#d24726'
  if (mimeType.startsWith('text/')) return 'var(--primary-color)'
  if (mimeType.includes('zip') || mimeType.includes('rar') || mimeType.includes('7z'))
    return '#f39c12'
  return 'var(--text-secondary)'
}

/**
 * 格式化文件大小
 */
export function formatFileSize(size?: number): string {
  if (size === undefined || size === null) return '未知大小'
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  if (size < 1024 * 1024 * 1024) return `${(size / (1024 * 1024)).toFixed(1)} MB`
  return `${(size / (1024 * 1024 * 1024)).toFixed(1)} GB`
}

/**
 * 格式化文件日期
 */
export function formatFileDate(dateString?: string): string {
  if (!dateString) return ''
  const date = new Date(dateString)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

/**
 * 获取文件扩展名
 */
export function getFileExtension(filename?: string): string {
  if (!filename) return ''
  const lastDot = filename.lastIndexOf('.')
  if (lastDot === -1 || lastDot === 0) return ''
  return filename.substring(lastDot + 1).toLowerCase()
}

/**
 * 获取文件类型标签
 */
export function getFileTypeLabel(mimeType?: string): string {
  if (!mimeType) return '文件'
  if (isImage(mimeType)) return '图片'
  if (isVideo(mimeType)) return '视频'
  if (isAudio(mimeType)) return '音频'
  if (mimeType.includes('pdf')) return 'PDF'
  if (mimeType.includes('word') || mimeType.includes('document')) return 'Word'
  if (mimeType.includes('excel') || mimeType.includes('sheet')) return 'Excel'
  if (mimeType.includes('powerpoint') || mimeType.includes('presentation')) return 'PPT'
  if (mimeType.startsWith('text/')) return '文本'
  if (mimeType.includes('zip') || mimeType.includes('rar') || mimeType.includes('7z'))
    return '压缩包'
  return '文件'
}

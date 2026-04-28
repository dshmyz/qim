// 文件类型判断工具函数

/** 文件分类枚举 */
export type FileCategory =
  | 'image'
  | 'video'
  | 'audio'
  | 'document'
  | 'spreadsheet'
  | 'presentation'
  | 'archive'
  | 'code'
  | 'text'
  | 'font'
  | 'unknown'

/** MIME 类型到文件分类的映射表 */
const MIME_CATEGORY_MAP: Record<string, FileCategory> = {
  // Images
  'image/jpeg': 'image',
  'image/png': 'image',
  'image/gif': 'image',
  'image/webp': 'image',
  'image/svg+xml': 'image',
  'image/bmp': 'image',
  'image/tiff': 'image',
  'image/x-icon': 'image',
  'image/avif': 'image',
  'image/heic': 'image',
  'image/heif': 'image',

  // Videos
  'video/mp4': 'video',
  'video/webm': 'video',
  'video/ogg': 'video',
  'video/quicktime': 'video',
  'video/x-msvideo': 'video',
  'video/x-matroska': 'video',
  'video/x-flv': 'video',
  'video/x-ms-wmv': 'video',
  'video/3gpp': 'video',
  'video/3gpp2': 'video',
  'video/avi': 'video',
  'video/mpeg': 'video',
  'video/mov': 'video',

  // Audio
  'audio/mpeg': 'audio',
  'audio/wav': 'audio',
  'audio/ogg': 'audio',
  'audio/webm': 'audio',
  'audio/mp4': 'audio',
  'audio/flac': 'audio',
  'audio/aac': 'audio',
  'audio/x-wav': 'audio',
  'audio/x-flac': 'audio',
  'audio/midi': 'audio',
  'audio/x-midi': 'audio',
  'audio/x-aiff': 'audio',
  'audio/aacp': 'audio',
  'audio/opus': 'audio',

  // Documents
  'application/pdf': 'document',
  'application/msword': 'document',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document': 'document',
  'text/plain': 'document',
  'text/rtf': 'document',
  'text/markdown': 'document',
  'text/csv': 'document',
  'application/rtf': 'document',
  'application/vnd.oasis.opendocument.text': 'document',

  // Spreadsheets
  'application/vnd.ms-excel': 'spreadsheet',
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': 'spreadsheet',
  'application/vnd.oasis.opendocument.spreadsheet': 'spreadsheet',
  'application/x-iwork-numbers-sffnumbers': 'spreadsheet',

  // Presentations
  'application/vnd.ms-powerpoint': 'presentation',
  'application/vnd.openxmlformats-officedocument.presentationml.presentation': 'presentation',
  'application/vnd.oasis.opendocument.presentation': 'presentation',
  'application/x-iwork-keynote-sffkey': 'presentation',

  // Archives
  'application/zip': 'archive',
  'application/x-tar': 'archive',
  'application/gzip': 'archive',
  'application/x-7z-compressed': 'archive',
  'application/x-rar-compressed': 'archive',
  'application/x-bzip2': 'archive',
  'application/x-xz': 'archive',
  'application/vnd.rar': 'archive',
  'application/java-archive': 'archive',

  // Code
  'application/json': 'code',
  'application/xml': 'code',
  'application/javascript': 'code',
  'application/typescript': 'code',
  'text/javascript': 'code',
  'text/xml': 'code',
  'text/html': 'code',
  'text/css': 'code',
  'application/x-sh': 'code',
  'application/x-python': 'code',
  'text/x-python': 'code',
  'text/x-java': 'code',
  'text/x-c': 'code',
  'text/x-c++': 'code',
  'text/x-go': 'code',
  'text/x-ruby': 'code',
  'text/x-php': 'code',
  'text/x-swift': 'code',
  'text/x-rust': 'code',
  'text/x-kotlin': 'code',
  'text/x-scala': 'code',
  'application/x-httpd-php': 'code',
  'application/xhtml+xml': 'code',

  // Fonts
  'font/woff': 'font',
  'font/woff2': 'font',
  'font/ttf': 'font',
  'font/otf': 'font',
  'application/x-font-ttf': 'font',
  'application/x-font-otf': 'font',
}

/** MIME 类型前缀到文件分类的映射（用于处理通配类型） */
const MIME_PREFIX_MAP: Record<string, FileCategory> = {
  'image/': 'image',
  'video/': 'video',
  'audio/': 'audio',
  'font/': 'font',
  'text/': 'text',
}

/** 文件分类到 FontAwesome 图标类名的映射 */
const CATEGORY_ICON_MAP: Record<FileCategory, string> = {
  image: 'fa-solid fa-file-image',
  video: 'fa-solid fa-file-video',
  audio: 'fa-solid fa-file-audio',
  document: 'fa-solid fa-file-lines',
  spreadsheet: 'fa-solid fa-file-excel',
  presentation: 'fa-solid fa-file-powerpoint',
  archive: 'fa-solid fa-file-zipper',
  code: 'fa-solid fa-file-code',
  text: 'fa-solid fa-file-alt',
  font: 'fa-solid fa-font',
  unknown: 'fa-solid fa-file',
}

/**
 * 根据 MIME 类型判断文件分类
 * @param mimeType - 文件的 MIME 类型（如 'image/png'）
 * @returns 文件分类，如果无法识别则返回 'unknown'
 */
export function getFileCategory(mimeType: string): FileCategory {
  if (!mimeType || typeof mimeType !== 'string') {
    return 'unknown'
  }

  const normalized = mimeType.trim().toLowerCase()

  // 精确匹配
  if (MIME_CATEGORY_MAP[normalized] !== undefined) {
    return MIME_CATEGORY_MAP[normalized]
  }

  // 前缀匹配（处理 image/*、video/* 等通配类型）
  for (const [prefix, category] of Object.entries(MIME_PREFIX_MAP)) {
    if (normalized.startsWith(prefix)) {
      return category
    }
  }

  return 'unknown'
}

/**
 * 根据 MIME 类型获取 FontAwesome 图标类名
 * @param mimeType - 文件的 MIME 类型
 * @returns FontAwesome 图标类名
 */
export function getFileIcon(mimeType: string): string {
  const category = getFileCategory(mimeType)
  return CATEGORY_ICON_MAP[category] ?? CATEGORY_ICON_MAP.unknown
}

/**
 * 格式化字节数为人类可读的文件大小
 * @param bytes - 字节数
 * @returns 格式化后的字符串（如 '1.5 MB'）
 */
export function formatFileSize(bytes: number): string {
  if (bytes === 0) {
    return '0 B'
  }

  if (bytes < 0 || !Number.isFinite(bytes)) {
    return '0 B'
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const k = 1024
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  const index = Math.min(i, units.length - 1)
  const size = bytes / Math.pow(k, index)

  // 整数部分直接显示，小数部分最多保留 1 位（去掉末尾的 0）
  const formatted = size % 1 === 0 ? size.toString() : size.toFixed(1).replace(/\.0$/, '')

  return `${formatted} ${units[index]}`
}

/**
 * 判断是否为图片文件
 * @param mimeType - 文件的 MIME 类型
 * @returns 是否为图片
 */
export function isImageFile(mimeType: string): boolean {
  return getFileCategory(mimeType) === 'image'
}

/**
 * 判断是否为视频文件
 * @param mimeType - 文件的 MIME 类型
 * @returns 是否为视频
 */
export function isVideoFile(mimeType: string): boolean {
  return getFileCategory(mimeType) === 'video'
}

/**
 * 判断是否为音频文件
 * @param mimeType - 文件的 MIME 类型
 * @returns 是否为音频
 */
export function isAudioFile(mimeType: string): boolean {
  return getFileCategory(mimeType) === 'audio'
}

/**
 * 判断是否为文档文件（PDF、Word、纯文本等）
 * @param mimeType - 文件的 MIME 类型
 * @returns 是否为文档
 */
export function isDocumentFile(mimeType: string): boolean {
  return getFileCategory(mimeType) === 'document'
}

/**
 * 判断是否为电子表格文件
 * @param mimeType - 文件的 MIME 类型
 * @returns 是否为电子表格
 */
export function isSpreadsheetFile(mimeType: string): boolean {
  return getFileCategory(mimeType) === 'spreadsheet'
}

/**
 * 判断是否为演示文稿文件
 * @param mimeType - 文件的 MIME 类型
 * @returns 是否为演示文稿
 */
export function isPresentationFile(mimeType: string): boolean {
  return getFileCategory(mimeType) === 'presentation'
}

/**
 * 判断是否为压缩包文件
 * @param mimeType - 文件的 MIME 类型
 * @returns 是否为压缩包
 */
export function isArchiveFile(mimeType: string): boolean {
  return getFileCategory(mimeType) === 'archive'
}

/**
 * 判断是否为代码文件
 * @param mimeType - 文件的 MIME 类型
 * @returns 是否为代码文件
 */
export function isCodeFile(mimeType: string): boolean {
  return getFileCategory(mimeType) === 'code'
}

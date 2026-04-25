import type { Message } from '../types'

/**
 * 聊天工具函数 composable
 * 包含时间格式化、时间分隔线判断、文件图标获取、文件大小格式化、Markdown 渲染等功能
 */
export function useChatUtils() {
  // 格式化时间戳为可读时间
  const formatTime = (timestamp: number | string | null | undefined): string => {
    // 检查 timestamp 是否有效
    if (!timestamp || (typeof timestamp !== 'number' && typeof timestamp !== 'string')) {
      return '未知时间'
    }

    const date = new Date(timestamp)

    // 检查日期是否有效
    if (isNaN(date.getTime())) {
      return '未知时间'
    }

    const now = new Date()
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
    const messageDate = new Date(date.getFullYear(), date.getMonth(), date.getDate())
    const diffDays = Math.floor((today.getTime() - messageDate.getTime()) / (24 * 60 * 60 * 1000))

    if (diffDays === 0) {
      // 今天的消息，显示具体时间
      return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
    } else if (diffDays === 1) {
      // 昨天的消息，显示"昨天 时间"
      return `昨天 ${date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}`
    } else if (diffDays < 7) {
      // 本周的消息，显示星期几和时间
      const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
      const weekday = weekdays[date.getDay()]
      return `${weekday} ${date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}`
    } else {
      // 更早的消息，显示具体日期和时间
      return date.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
    }
  }

  // 判断是否应该显示时间分隔线
  const shouldShowTimeDivider = (index: number, currentMessage: Message, messages?: Message[]): boolean => {
    // 如果没有传入 messages 数组，只能判断是否是第一条消息
    if (!messages || messages.length === 0) {
      return index === 0
    }

    // 第一条消息总是显示时间
    if (index === 0) {
      return true
    }

    // 获取前一条消息
    const previousMessage = messages[index - 1]
    if (!previousMessage) {
      return true
    }

    // 计算时间差（毫秒）
    const timeDiff = currentMessage.timestamp - previousMessage.timestamp

    // 如果时间差超过5分钟，显示时间分隔线
    if (timeDiff > 5 * 60 * 1000) {
      return true
    }

    // 如果是不同的日期，显示时间分隔线
    const currentDate = new Date(currentMessage.timestamp)
    const previousDate = new Date(previousMessage.timestamp)

    if (
      currentDate.getFullYear() !== previousDate.getFullYear() ||
      currentDate.getMonth() !== previousDate.getMonth() ||
      currentDate.getDate() !== previousDate.getDate()
    ) {
      return true
    }

    return false
  }

  // 根据文件扩展名获取对应的 Font Awesome 图标
  const getFileIcon = (fileUrl: string): string => {
    const fileName = fileUrl.split('/').pop() || fileUrl
    const extension = fileName.split('.').pop()?.toLowerCase() || ''

    switch (extension) {
      // 文档类
      case 'doc':
      case 'docx':
        return 'fas fa-file-word'
      case 'xls':
      case 'xlsx':
        return 'fas fa-file-excel'
      case 'ppt':
      case 'pptx':
        return 'fas fa-file-powerpoint'
      case 'pdf':
        return 'fas fa-file-pdf'
      case 'txt':
        return 'fas fa-file-alt'
      case 'md':
        return 'fas fa-file-markdown'

      // 图片类
      case 'jpg':
      case 'jpeg':
      case 'png':
      case 'gif':
      case 'webp':
      case 'bmp':
        return 'fas fa-file-image'

      // 音频类
      case 'mp3':
      case 'wav':
      case 'ogg':
      case 'flac':
        return 'fas fa-file-audio'

      // 视频类
      case 'mp4':
      case 'avi':
      case 'mov':
      case 'wmv':
      case 'flv':
        return 'fas fa-file-video'

      // 压缩包类
      case 'zip':
      case 'rar':
      case '7z':
      case 'tar':
      case 'gz':
        return 'fas fa-file-archive'

      // 代码类
      case 'js':
      case 'ts':
      case 'jsx':
      case 'tsx':
      case 'html':
      case 'css':
      case 'scss':
      case 'less':
      case 'json':
      case 'xml':
      case 'yaml':
      case 'yml':
      case 'py':
      case 'java':
      case 'c':
      case 'cpp':
      case 'cs':
      case 'go':
      case 'php':
      case 'rb':
      case 'swift':
      case 'kt':
        return 'fas fa-file-code'

      // 默认图标
      default:
        return 'fas fa-file'
    }
  }

  // 格式化文件大小
  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 B'

    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))

    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  // 渲染 Markdown
  const renderMarkdown = (content: string): string => {
    // 简单的 Markdown 渲染
    let html = content

    // 标题
    html = html.replace(/^# (.*$)/gm, '<h1>$1</h1>')
    html = html.replace(/^## (.*$)/gm, '<h2>$1</h2>')
    html = html.replace(/^### (.*$)/gm, '<h3>$1</h3>')

    // 粗体
    html = html.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')

    // 斜体
    html = html.replace(/\*(.*?)\*/g, '<em>$1</em>')

    // 代码块
    html = html.replace(/```([\s\S]*?)```/g, '<pre><code>$1</code></pre>')

    // 行内代码
    html = html.replace(/`(.*?)`/g, '<code>$1</code>')

    // 列表
    html = html.replace(/^- (.*$)/gm, '<li>$1</li>')
    html = html.replace(/(<li>.*<\/li>)/s, '<ul>$1</ul>')

    // 链接
    html = html.replace(/\[(.*?)\]\((.*?)\)/g, '<a href="$2" target="_blank">$1</a>')

    // 换行
    html = html.replace(/\n/g, '<br>')

    return html
  }

  return {
    formatTime,
    shouldShowTimeDivider,
    getFileIcon,
    formatFileSize,
    renderMarkdown
  }
}

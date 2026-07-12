import { decodeToPlainText } from './mentions'

/**
 * 将 markdown 源码转换为可读纯文本。
 *
 * 处理规则：
 * - 围栏代码块：去掉围栏和语言标识，保留代码内容
 * - 行内代码：去掉反引号
 * - 标题/粗体/斜体：去掉 markdown 标记符号
 * - 链接：保留链接文字，去掉 URL
 * - 图片：替换为 [图片]
 * - 列表/引用/水平线：去掉标记符号
 * - mention token：解码为 @姓名
 *
 * @param content markdown 源码
 * @returns 可读纯文本
 */
export function markdownToPlainText(content: string): string {
  if (!content) return ''

  // 先解码 mention token：@{mention:3|张三} → @张三
  let text = decodeToPlainText(content)

  // 去除围栏代码块，保留代码内容
  text = text.replace(/```[a-zA-Z0-9]*\n([\s\S]*?)```/g, '$1')

  // 去除行内代码反引号
  text = text.replace(/`([^`]+)`/g, '$1')

  // 去除图片语法：![alt](url) → [图片]
  text = text.replace(/!\[([^\]]*)\]\([^)]+\)/g, '[图片]')

  // 去除链接语法：[text](url) → text
  text = text.replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')

  // 去除标题标记
  text = text.replace(/^#{1,6}\s+/gm, '')

  // 去除粗体/斜体标记
  text = text.replace(/\*\*([^*]+)\*\*/g, '$1')
  text = text.replace(/__([^_]+)__/g, '$1')
  text = text.replace(/\*([^*]+)\*/g, '$1')
  text = text.replace(/_([^_]+)_/g, '$1')

  // 去除无序列表标记
  text = text.replace(/^\s*[-*+]\s+/gm, '')

  // 去除有序列表标记
  text = text.replace(/^\s*\d+\.\s+/gm, '')

  // 去除引用标记
  text = text.replace(/^>\s*/gm, '')

  // 去除水平线
  text = text.replace(/^[-*]{3,}$/gm, '')

  return text.trim()
}

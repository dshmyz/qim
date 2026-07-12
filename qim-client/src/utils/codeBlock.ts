/**
 * 将代码与语言标记组装为 markdown 围栏代码块字符串。
 * @param code 代码内容
 * @param language 语言标识（如 'javascript'、'python'），空字符串表示无语言标记
 * @returns 形如 ```lang\ncode\n``` 的字符串
 */
export function formatCodeBlock(code: string, language: string = ''): string {
  const lang = language.trim()
  const trimmedCode = code.replace(/\n+$/, '')
  return '```' + lang + '\n' + trimmedCode + '\n```'
}

/**
 * 从 markdown 内容中提取所有围栏代码块的代码内容（不含围栏和语言标识）。
 * 行内代码（`code`）不会被提取。
 * @param content markdown 源码
 * @returns 代码块内容数组，无代码块时返回空数组
 */
export function extractCodeBlocks(content: string): string[] {
  const blocks: string[] = []
  const regex = /```[a-zA-Z0-9]*\n([\s\S]*?)```/g
  let match: RegExpExecArray | null
  while ((match = regex.exec(content)) !== null) {
    blocks.push(match[1].replace(/\n$/, ''))
  }
  return blocks
}

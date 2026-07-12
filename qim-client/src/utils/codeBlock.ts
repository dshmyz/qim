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

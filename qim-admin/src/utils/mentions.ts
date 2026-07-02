const TOKEN_REGEX = /@\{mention:(all|[1-9]\d*)(?:\|([^}]*))?\}/g

export function decodeMentionTokens(content: string): string {
  return content.replace(TOKEN_REGEX, (_token, target: string, encodedName?: string) => {
    if (target === 'all') {
      return '@所有人'
    }

    if (!encodedName) {
      return `@用户${target}`
    }

    try {
      return `@${decodeURIComponent(encodedName)}`
    } catch {
      return `@${encodedName}`
    }
  })
}

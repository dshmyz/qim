export interface MentionSpan {
  start: number
  end: number
  text: string
  userIds: number[]
  all?: boolean
}

export function reconcileMentionSpans(
  spans: MentionSpan[],
  previousText: string,
  nextText: string
): MentionSpan[] {
  if (previousText === nextText) return spans

  let prefixLength = 0
  const sharedLength = Math.min(previousText.length, nextText.length)
  while (prefixLength < sharedLength && previousText[prefixLength] === nextText[prefixLength]) {
    prefixLength++
  }

  let suffixLength = 0
  while (
    suffixLength < previousText.length - prefixLength &&
    suffixLength < nextText.length - prefixLength &&
    previousText[previousText.length - 1 - suffixLength] === nextText[nextText.length - 1 - suffixLength]
  ) {
    suffixLength++
  }

  const removedEnd = previousText.length - suffixLength
  const delta = nextText.length - previousText.length

  return spans.flatMap((span) => {
    if (span.end <= prefixLength) return [span]
    if (span.start >= removedEnd) {
      return [{ ...span, start: span.start + delta, end: span.end + delta }]
    }
    return []
  })
}

/**
 * Converts only selections made through the mention picker into persistent
 * tokens. Ordinary "@..." text is left untouched, so source code and URLs
 * cannot accidentally become mentions.
 */
export function serializeMentionTokens(content: string, spans: MentionSpan[]): string {
  const validSpans = spans
    .filter((span) => content.slice(span.start, span.end) === span.text)
    .sort((a, b) => b.start - a.start)

  let serialized = content
  for (const span of validSpans) {
    const token = span.all
      ? '@{mention:all}'
      : (() => {
          const id = span.userIds.find((userID) => Number.isSafeInteger(userID) && userID > 0)
          return id ? `@{mention:${id}|${encodeURIComponent(span.text.slice(1))}}` : span.text
        })()
    serialized = serialized.slice(0, span.start) + token + serialized.slice(span.end)
  }
  return serialized
}

const mentionTokenPattern = /@\{mention:(all|[1-9]\d*)(?:\|([^}]*))?\}/g

/** Converts stored mention tokens back to text suitable for UI and search. */
export function displayMentionTokens(content: string): string {
  return content.replace(mentionTokenPattern, (token, target: string, encodedName?: string) => {
    if (target === 'all' && encodedName === undefined) return '@所有人'
    if (target === 'all' || encodedName === undefined) return token
    try {
      return `@${decodeURIComponent(encodedName)}`
    } catch {
      return token
    }
  })
}

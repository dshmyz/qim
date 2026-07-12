import { describe, expect, it } from 'vitest'
import { formatCodeBlock, extractCodeBlocks } from '@/utils/codeBlock'

describe('formatCodeBlock', () => {
  it('wraps code with language fence', () => {
    expect(formatCodeBlock('console.log(1)', 'javascript'))
      .toBe('```javascript\nconsole.log(1)\n```')
  })

  it('wraps code with python fence', () => {
    expect(formatCodeBlock('print(1)', 'python'))
      .toBe('```python\nprint(1)\n```')
  })

  it('produces fence without language when language is empty', () => {
    expect(formatCodeBlock('hello', ''))
      .toBe('```\nhello\n```')
  })

  it('trims trailing newlines from code to keep fence clean', () => {
    expect(formatCodeBlock('hello\n\n', 'js'))
      .toBe('```js\nhello\n```')
  })

  it('trims surrounding whitespace from language identifier', () => {
    expect(formatCodeBlock('x = 1', '  python  '))
      .toBe('```python\nx = 1\n```')
  })

  it('handles multi-line code', () => {
    const code = 'function add(a, b) {\n  return a + b\n}'
    expect(formatCodeBlock(code, 'typescript'))
      .toBe('```typescript\n' + code + '\n```')
  })
})

describe('extractCodeBlocks', () => {
  it('extracts a single code block', () => {
    const md = '```javascript\nconsole.log(1)\n```'
    expect(extractCodeBlocks(md)).toEqual(['console.log(1)'])
  })

  it('extracts code block without language identifier', () => {
    const md = '```\nhello\n```'
    expect(extractCodeBlocks(md)).toEqual(['hello'])
  })

  it('extracts multiple code blocks', () => {
    const md = '```js\na\n```\ntext\n```python\nb\n```'
    expect(extractCodeBlocks(md)).toEqual(['a', 'b'])
  })

  it('preserves multi-line code content', () => {
    const code = 'function add(a, b) {\n  return a + b\n}'
    const md = '```typescript\n' + code + '\n```'
    expect(extractCodeBlocks(md)).toEqual([code])
  })

  it('returns empty array when no code block', () => {
    expect(extractCodeBlocks('just plain text')).toEqual([])
  })

  it('returns empty array for inline code only', () => {
    expect(extractCodeBlocks('use `console.log` to debug')).toEqual([])
  })
})

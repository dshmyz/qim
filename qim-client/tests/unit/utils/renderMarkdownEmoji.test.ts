import { describe, it, expect } from 'vitest'
import { useChatUtils } from '@/composables/useChatUtils'

/**
 * 聊天消息渲染（renderMarkdown）把 emoji 转成 Twemoji 图片。
 * 与频道渲染 channelMarkdown 同一链路（sanitize 后 emojiToHtml），
 * 保证 Linux 无系统 emoji 字体时也能正常显示，而非方框/乱码。
 */
describe('renderMarkdown emoji', () => {
  const { renderMarkdown } = useChatUtils()

  it('把 emoji 字符转成 Twemoji <img>', () => {
    const html = renderMarkdown('😀 你好')
    expect(html).toContain('<img')
    expect(html).toContain('emoji/72x72/1f600.png')
  })

  it('普通文本不受影响', () => {
    const html = renderMarkdown('你好，世界')
    expect(html).not.toContain('<img')
    expect(html).toContain('你好，世界')
  })

  it('Markdown 结构仍保留', () => {
    const html = renderMarkdown('**粗体** 😀')
    expect(html).toContain('<strong>粗体</strong>')
    expect(html).toContain('<img')
  })

  it('直播脚本消毒依旧生效', () => {
    const html = renderMarkdown('<script>alert(1)</script> 😀')
    expect(html).not.toContain('<script>')
    expect(html).toContain('<img')
  })
})

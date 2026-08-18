import { describe, expect, it } from 'vitest'
import { renderMarkdown } from '../../../src/composables/useMarkdownRender'

describe('useMarkdownRender note-link XSS 防护', () => {
  it('属性注入样本不再逃逸属性', () => {
    const html = renderMarkdown('[[a" onmouseover="alert(1)]]')
    // 属性闭合引号后不得出现裸 onmouseover（转义后的 &quot; onmouseover= 是属性值内的文本，安全）
    expect(html).not.toContain('" onmouseover=')
    expect(html).toContain('data-note-title="a&quot; onmouseover=&quot;alert(1)"')
  })

  it('img 注入样本被转义为纯文本', () => {
    const html = renderMarkdown('[[<img src=x onerror=alert(1)>]]')
    expect(html).not.toMatch(/<img/)
    expect(html).toContain('&lt;img')
  })

  it('script 样本不产生可执行标签', () => {
    const html = renderMarkdown('[[<script>alert(1)</script>]]')
    expect(html).not.toMatch(/<script/)
  })

  it('正常标题生成正确锚点，dataset 可还原', () => {
    const html = renderMarkdown('[[设计稿 😀]]')
    expect(html).toContain('class="note-link"')
    expect(html).toContain('data-note-title="设计稿 😀"')
  })

  it('标题中的经典标记不被 classicToHtml 污染（withEmoji）', () => {
    const html = renderMarkdown('[[[示爱]]]', { withEmoji: true })
    // 属性与锚点文本都保持标题原文 [示爱（不含实体化、不被转成 <img>）
    expect(html).toContain('data-note-title="[示爱"')
    expect(html).toContain('fa-sticky-note"></i> [示爱')
    expect(html).not.toContain('classic-emoji-img')
  })

  it('emoji 标题在 withEmoji 下属性值安全', () => {
    const html = renderMarkdown('[[设计稿 😀]]', { withEmoji: true })
    expect(html).toContain('data-note-title="设计稿 😀"')
    // 正文表情转图，但属性内不出现 img 标签
    expect(html).not.toMatch(/data-note-title="[^"]*<img/)
  })

  it('围栏 code 块内 [[title]] 不生成链接，原样恢复为 <pre><code>', () => {
    const html = renderMarkdown('```\n[[block]]\n```')
    expect(html).not.toContain('note-link')
    expect(html).toContain('<pre><code>[[block]]</code></pre>')
  })

  it('行内 code 内 [[title]] 不生成链接', () => {
    const html = renderMarkdown('前 ``[[inline]]`` 后')
    expect(html).not.toContain('note-link')
    expect(html).toContain('<code>[[inline]]</code>')
  })

  it('标题含行内 code 反引号时原文保留', () => {
    const html = renderMarkdown('[[a `b` c]]')
    expect(html).toContain('data-note-title="a `b` c"')
  })

  it('围栏 code 内容特殊字符安全转义', () => {
    const html = renderMarkdown('```html\n<a href="x" onclick="y">z</a>\n```')
    expect(html).toContain('<pre><code>&lt;a href=&quot;x&quot; onclick=&quot;y&quot;&gt;z&lt;/a&gt;</code></pre>')
  })

  it('普通文本与既有渲染不受影响', () => {
    const html = renderMarkdown('**粗体** [链接](https://example.com)')
    expect(html).toContain('<strong>粗体</strong>')
    expect(html).toContain('href="https://example.com"')
  })

  it('正文与标题共存：正文内链按需，围栏后紧跟内链', () => {
    const html = renderMarkdown('```\ncode\n```\n\n[[标题]]')
    expect(html).toContain('<pre><code>code</code></pre>')
    expect(html).toContain('data-note-title="标题"')
  })

  it('代码块紧跟说明文字一行（无空行）仍打断段落渲染为 <pre><code>', () => {
    // 回归：围栏紧跟段落末行、中间只有单个 \n 时，占位符不得并进上一段 <p> 泄漏成乱码
    const html = renderMarkdown('比如你可以这样发：\n```bash\necho hi\n```')
    expect(html).toContain('<pre><code>echo hi</code></pre>')
    expect(html).toContain('<p>比如你可以这样发：</p>')
    expect(html).not.toMatch(/\d+/)
  })

  it('列表项内缩进围栏（2 空格）正确恢复为 <pre><code>', () => {
    // 回归：列表项内缩进围栏占位行落到 0 列会把围栏提前切断、占位符逃出泄漏
    const html = renderMarkdown('- 常见写法：\n  ```\n  GOPROXY=https://goproxy.cn,direct\n  ```')
    // 恢复保留围栏内原文（含原行首缩进），与既有 extractNoteLinks 恢复行为一致
    expect(html).toContain('<pre><code>  GOPROXY=https://goproxy.cn,direct')
    expect(html).not.toMatch(/\d+/)
  })

  it('有序列表项内 4 空格缩进围栏（含语言标记）正确恢复且不泄漏', () => {
    // 回归：缩进对齐——开头围栏不重复补缩进（否则缩进翻倍成缩进代码块），
    // 占位行/收尾围栏补上原缩进，列表项内整体结构一致
    const html = renderMarkdown('1.  权限说明：\n    ```xml\n    <key>screenshot</key>\n    ```')
    expect(html).toContain('<pre><code>    &lt;key&gt;screenshot&lt;/key&gt;')
    expect(html).not.toMatch(/\d+/)
  })
})

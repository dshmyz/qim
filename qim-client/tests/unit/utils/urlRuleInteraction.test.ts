import { describe, it, expect } from 'vitest'
import { applyRenderRules } from '@/utils/renderRules'
import { escapeHTML } from '@/utils/sanitize'
import type { CompiledRule } from '@/stores/renderRules'

function makeRule(overrides: Partial<CompiledRule> = {}): CompiledRule {
  const base: CompiledRule = {
    id: 'jira_ticket',
    name: 'Jira 工单卡片化',
    enabled: true,
    priority: 10,
    scope: { groups: ['*'], exclude_groups: [], conversation_types: ['single', 'group', 'discussion'] },
    match: {
      pattern: '\\b([A-Z]{2,6})-(\\d{1,6})\\b',
      flags: 'g',
      capture_groups: { project: 1, number: 2 },
    },
    render: {
      type: 'link_card',
      url_template: 'http://jira.xxx.com/{{project}}/{{project}}-{{number}}',
      label_template: '{{project}}-{{number}}',
      title_template: '查看 {{project}}-{{number}}',
      icon: 'fab fa-jira',
      target: '_blank',
      class: 'jira-card',
    },
    compiledRegex: new RegExp('\\b([A-Z]{2,6})-(\\d{1,6})\\b', 'g'),
  }
  const merged: CompiledRule = {
    ...base,
    ...overrides,
    match: { ...base.match, ...overrides.match },
    render: { ...base.render, ...overrides.render },
  }
  merged.compiledRegex = new RegExp(merged.match.pattern, merged.match.flags || 'g')
  return merged
}

// 模拟 TextMessage.vue 的 linkifyUrls 简化版
function linkifyUrls(text: string): string {
  const urlRegex = /https?:\/\/[^\s<]+/g
  return text.replace(urlRegex, (matchedUrl) => {
    return `<a href="${matchedUrl}" target="_blank" rel="noopener noreferrer" class="message-link">${matchedUrl}</a>`
  })
}

describe('URL 与渲染规则的交互', () => {
  const rule = makeRule()

  describe('跳过 <a> 标签内部', () => {
    it('URL 中包含 NI-30000 时，不破坏 <a> 标签结构', () => {
      const rawText = '请看 http://jira.xxx.com/NI/NI-30000'
      const escaped = escapeHTML(rawText)
      const linked = linkifyUrls(escaped)
      const result = applyRenderRules(linked, [rule])

      // 期望：原始 <a> 标签完整保留，不生成嵌套 <a>
      const anchorCount = (result.match(/<a /g) || []).length
      expect(anchorCount).toBe(1)

      // 期望：href 完整未被破坏
      expect(result).toContain('href="http://jira.xxx.com/NI/NI-30000"')

      // 期望：不生成规则卡片（因为 NI-30000 在 <a> 内部，应跳过）
      expect(result).not.toContain('render-card')
    })

    it('URL 参数中包含工单号时，不破坏 <a> 标签', () => {
      const rawText = '链接 http://example.com?ticket=NI-30000 看看'
      const escaped = escapeHTML(rawText)
      const linked = linkifyUrls(escaped)
      const result = applyRenderRules(linked, [rule])

      const anchorCount = (result.match(/<a /g) || []).length
      expect(anchorCount).toBe(1)
      expect(result).toContain('href="http://example.com?ticket=NI-30000"')
      expect(result).not.toContain('render-card')
    })

    it('<a> 标签外部的 NI-30000 仍正常匹配', () => {
      const rawText = 'http://example.com/NI-30000 和 NI-30000 都看看'
      const escaped = escapeHTML(rawText)
      const linked = linkifyUrls(escaped)
      const result = applyRenderRules(linked, [rule])

      // 期望：URL 内的 NI-30000 跳过，外部的 NI-30000 匹配
      const anchorCount = (result.match(/<a /g) || []).length
      expect(anchorCount).toBe(2) // 1 个 linkify 的 + 1 个规则的

      // 外部 NI-30000 被规则匹配生成卡片
      expect(result).toContain('render-card')
      expect(result).toContain('href="http://jira.xxx.com/NI/NI-30000"')
    })

    it('纯文本中的 NI-30000 不受影响', () => {
      const rawText = '工单 NI-30000 需要处理'
      const escaped = escapeHTML(rawText)
      const linked = linkifyUrls(escaped)
      const result = applyRenderRules(linked, [rule])

      expect(result).toContain('render-card')
      expect(result).toContain('href="http://jira.xxx.com/NI/NI-30000"')
    })

    it('多个 <a> 标签之间的文本仍正常匹配', () => {
      const rawText = 'http://a.com NI-10001 http://b.com NI-10002'
      const escaped = escapeHTML(rawText)
      const linked = linkifyUrls(escaped)
      const result = applyRenderRules(linked, [rule])

      // 2 个 linkify URL + 2 个规则卡片 = 4 个 <a>
      const anchorCount = (result.match(/<a /g) || []).length
      expect(anchorCount).toBe(4)

      expect(result).toContain('href="http://jira.xxx.com/NI/NI-10001"')
      expect(result).toContain('href="http://jira.xxx.com/NI/NI-10002"')
    })

    it('linkify 生成的 <a> 标签 label 是 URL 时，内部不二次匹配', () => {
      // linkify 生成的 label 就是 URL 本身： <a href="...">http://...</a>
      const rawText = 'http://jira.xxx.com/NI-30000'
      const escaped = escapeHTML(rawText)
      const linked = linkifyUrls(escaped)
      const result = applyRenderRules(linked, [rule])

      // 只应该有 1 个 <a>（linkify 的），规则不在 label 内匹配
      const anchorCount = (result.match(/<a /g) || []).length
      expect(anchorCount).toBe(1)
      expect(result).not.toContain('render-card')
    })

    it('已有 <a> 标签前后都有规则匹配', () => {
      // 模拟 markdown 链接 + 规则匹配
      const linked = '前缀 NI-10001 <a href="http://x.com/NI-10002">label</a> 后缀 NI-10003'
      const result = applyRenderRules(linked, [rule])

      // 期望：2 个规则卡片（NI-10001, NI-10003）+ 1 个已有 <a> = 3 个 <a>
      // NI-10002 在 <a> 内部，被跳过
      const anchorCount = (result.match(/<a /g) || []).length
      expect(anchorCount).toBe(3)

      // 已有 <a> 标签内 NI-10002 不被规则匹配
      expect(result).toContain('>label</a>')
      expect(result).not.toContain('href="http://jira.xxx.com/NI/NI-10002"')

      // 外部 NI-10001 和 NI-10003 被规则匹配
      expect(result).toContain('href="http://jira.xxx.com/NI/NI-10001"')
      expect(result).toContain('href="http://jira.xxx.com/NI/NI-10003"')
    })
  })
})

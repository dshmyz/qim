import { describe, it, expect } from 'vitest'
import { applyRenderRules } from '@/utils/renderRules'
import { escapeHTML } from '@/utils/sanitize'
import type { CompiledRule } from '@/stores/renderRules'

function makeRule(overrides: Partial<CompiledRule> = {}): CompiledRule {
  const base: CompiledRule = {
    id: 'test_rule',
    name: '测试规则',
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
  // 根据最终的 match 配置重新编译正则
  merged.compiledRegex = new RegExp(merged.match.pattern, merged.match.flags || 'g')
  return merged
}

describe('applyRenderRules 功能测试', () => {
  describe('空规则', () => {
    it('规则为空时原样返回', () => {
      const text = escapeHTML('hello NI-30000')
      expect(applyRenderRules(text, [])).toBe(text)
    })
  })

  describe('link_card 类型', () => {
    it('匹配 Jira 工单号并生成卡片', () => {
      const rule = makeRule()
      const input = escapeHTML('请看 NI-30000 这个工单')
      const result = applyRenderRules(input, [rule])

      expect(result).toContain('<a ')
      expect(result).toContain('href="http://jira.xxx.com/NI/NI-30000"')
      expect(result).toContain('target="_blank"')
      expect(result).toContain('class="render-card jira-card"')
      expect(result).toContain('NI-30000')
      expect(result).toContain('<i class="fab fa-jira"></i>')
    })

    it('一条消息中匹配多个工单号', () => {
      const rule = makeRule()
      const input = escapeHTML('NI-10001 和 NI-10002 都需要处理')
      const result = applyRenderRules(input, [rule])

      expect(result).toContain('href="http://jira.xxx.com/NI/NI-10001"')
      expect(result).toContain('href="http://jira.xxx.com/NI/NI-10002"')
    })
  })

  describe('link 类型', () => {
    it('匹配 PR 号并生成链接', () => {
      const rule = makeRule({
        match: { pattern: '#PR(\\d+)', flags: 'g', capture_groups: { number: 1 } },
        render: {
          type: 'link',
          url_template: 'https://github.com/org/repo/pull/{{number}}',
          label_template: '#PR{{number}}',
          title_template: '查看 PR #{{number}}',
          target: '_blank',
          class: 'pr-link',
          url_template_: '',
        } as any,
      })
      const input = escapeHTML('请合并 #PR123')
      const result = applyRenderRules(input, [rule])

      expect(result).toContain('<a ')
      expect(result).toContain('href="https://github.com/org/repo/pull/123"')
      expect(result).toContain('class="message-link pr-link"')
      expect(result).toContain('#PR123')
    })
  })

  describe('text_chip 类型', () => {
    it('匹配 @提及并生成标签', () => {
      const rule = makeRule({
        match: { pattern: '@([一-鿿A-Za-z0-9_]+)', flags: 'g', capture_groups: { name: 1 } },
        render: {
          type: 'text_chip',
          url_template: '',
          label_template: '@{{name}}',
          class: 'mention-chip',
        } as any,
      })
      const input = escapeHTML('@张三 你看看')
      const result = applyRenderRules(input, [rule])

      expect(result).toContain('<span ')
      expect(result).toContain('class="render-chip mention-chip"')
      expect(result).toContain('@张三')
    })
  })

  describe('多规则优先级', () => {
    it('按优先级依次应用多条规则', () => {
      const jiraRule = makeRule({ id: 'jira', priority: 10 })
      const prRule = makeRule({
        id: 'pr',
        priority: 20,
        match: { pattern: '#PR(\\d+)', flags: 'g', capture_groups: { number: 1 } },
        render: {
          type: 'link',
          url_template: 'https://github.com/pull/{{number}}',
          label_template: '#PR{{number}}',
          target: '_blank',
          class: 'pr-link',
        } as any,
      })

      const input = escapeHTML('NI-30000 和 #PR123')
      const result = applyRenderRules(input, [jiraRule, prRule])

      expect(result).toContain('href="http://jira.xxx.com/NI/NI-30000"')
      expect(result).toContain('href="https://github.com/pull/123"')
    })
  })

  describe('未匹配文本保留', () => {
    it('无匹配时原样返回文本', () => {
      const rule = makeRule()
      const input = escapeHTML('今天天气不错')
      const result = applyRenderRules(input, [rule])
      expect(result).toBe(input)
    })

    it('匹配前后的文本保留', () => {
      const rule = makeRule()
      const input = escapeHTML('前缀 NI-30000 后缀')
      const result = applyRenderRules(input, [rule])
      expect(result).toContain('前缀')
      expect(result).toContain('后缀')
      expect(result).toContain('href="http://jira.xxx.com/NI/NI-30000"')
    })
  })

  describe('HTML 注入防护', () => {
    it('用户输入的 HTML 特殊字符被转义', () => {
      const rule = makeRule()
      // 模拟已 escape 的文本（包含 <script>）
      const input = escapeHTML('<script>alert(1)</script> NI-30000')
      const result = applyRenderRules(input, [rule])

      // 不应包含未转义的 <script>
      expect(result).not.toContain('<script>')
      expect(result).toContain('&lt;script&gt;')
    })

    it('规则配置中的 icon class 被转义', () => {
      const rule = makeRule({
        render: {
          ...makeRule().render,
          icon: '"><script>alert(1)</script>',
        } as any,
      })
      const input = escapeHTML('NI-30000')
      const result = applyRenderRules(input, [rule])

      // 注入的 script 应被转义（由外层 sanitize 负责）或被 escapeHTML 转义
      expect(result).not.toContain('<script>alert(1)</script>')
    })
  })
})

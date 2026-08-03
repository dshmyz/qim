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
  merged.compiledRegex = new RegExp(merged.match.pattern, merged.match.flags || 'g')
  return merged
}

describe('多规则冲突与链式匹配', () => {
  describe('链式匹配：规则 A 剩余文本被规则 B 匹配', () => {
    it('不同模式的规则可链式匹配同一段文本的不同部分', () => {
      // 规则 A: Jira 工单号 NI-30000
      const jiraRule = makeRule({
        id: 'jira',
        priority: 10,
      })
      // 规则 B: @提及
      const mentionRule = makeRule({
        id: 'mention',
        priority: 20,
        match: { pattern: '@([一-鿿A-Za-z0-9_]+)', flags: 'g', capture_groups: { name: 1 } },
        render: {
          type: 'text_chip',
          url_template: '',
          label_template: '@{{name}}',
          class: 'mention-chip',
        } as any,
      })

      const input = escapeHTML('NI-30000 由 @张三 负责')
      const result = applyRenderRules(input, [jiraRule, mentionRule])

      // 两个规则都应生效
      expect(result).toContain('href="http://jira.xxx.com/NI/NI-30000"')
      expect(result).toContain('class="render-chip mention-chip"')
      expect(result).toContain('@张三')
    })

    it('三条规则可同时生效于同一条消息', () => {
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
      const mentionRule = makeRule({
        id: 'mention',
        priority: 30,
        match: { pattern: '@([一-鿿A-Za-z0-9_]+)', flags: 'g', capture_groups: { name: 1 } },
        render: {
          type: 'text_chip',
          url_template: '',
          label_template: '@{{name}}',
          class: 'mention-chip',
        } as any,
      })

      const input = escapeHTML('NI-10001 #PR123 @李四 都处理一下')
      const result = applyRenderRules(input, [jiraRule, prRule, mentionRule])

      expect(result).toContain('href="http://jira.xxx.com/NI/NI-10001"')
      expect(result).toContain('href="https://github.com/pull/123"')
      expect(result).toContain('class="render-chip mention-chip"')
      expect(result).toContain('@李四')
    })
  })

  describe('冲突场景：同一文本被多条规则匹配', () => {
    it('优先级高的规则（priority 小）先匹配，优先级低的不再处理该段', () => {
      // 规则 A: 匹配 NI-30000（完整工单号）
      const fullRule = makeRule({
        id: 'full_jira',
        priority: 10,
        render: {
          ...makeRule().render,
          class: 'full-jira-card',
        } as any,
      })
      // 规则 B: 匹配 30000（纯数字，是 NI-30000 的子串）
      const numberRule = makeRule({
        id: 'number_only',
        priority: 20,
        match: { pattern: '\\b(\\d{4,6})\\b', flags: 'g', capture_groups: { num: 1 } },
        render: {
          type: 'text_chip',
          url_template: '',
          label_template: '#{{num}}',
          class: 'number-chip',
        } as any,
      })

      const input = escapeHTML('NI-30000')
      const result = applyRenderRules(input, [fullRule, numberRule])

      // 规则 A 先匹配 NI-30000，规则 B 看不到这段文本
      expect(result).toContain('full-jira-card')
      expect(result).not.toContain('number-chip')
      expect(result).not.toContain('#30000')
    })

    it('反转优先级后，子串规则先匹配，会"吃掉"完整规则想匹配的文本', () => {
      // 规则 B 优先级变高（priority 小）
      const numberRule = makeRule({
        id: 'number_only',
        priority: 10, // 优先级更高
        match: { pattern: '\\b(\\d{4,6})\\b', flags: 'g', capture_groups: { num: 1 } },
        render: {
          type: 'text_chip',
          url_template: '',
          label_template: '#{{num}}',
          class: 'number-chip',
        } as any,
      })
      const fullRule = makeRule({
        id: 'full_jira',
        priority: 20, // 优先级更低
        render: {
          ...makeRule().render,
          class: 'full-jira-card',
        } as any,
      })

      // 关键点：\b 在 - 之后成立（- 是非 word 字符，30000 的 3 是 word 字符）
      // 所以 number 规则会匹配 NI-30000 中的 30000
      const input = escapeHTML('工单 12345 和 NI-30000')
      const result = applyRenderRules(input, [numberRule, fullRule])

      // 12345 被规则 B（number）匹配
      expect(result).toContain('number-chip')
      expect(result).toContain('#12345')

      // NI-30000 中的 30000 也被规则 B 匹配（因为 \b 在 - 后成立）
      // 导致 NI-30000 被拆成 "NI-" + number-chip("30000")
      // 完整规则 full_jira 无法再匹配（文本已被 number 吃掉）
      expect(result).toContain('#30000')
      expect(result).not.toContain('full-jira-card')
      expect(result).not.toContain('href="http://jira.xxx.com/NI/NI-30000"')

      // 这正是"子串规则先匹配会吃掉完整规则"的冲突行为
      // 提示：配置规则时避免子串包含关系，或给完整规则更高优先级
    })

    it('规则 A 生成的 HTML 不会被规则 B 二次匹配', () => {
      // 规则 A: 生成包含 http:// 的链接卡片
      const jiraRule = makeRule({
        id: 'jira',
        priority: 10,
      })
      // 规则 B: 匹配所有 URL（模拟 linkify）
      const urlRule = makeRule({
        id: 'url_matcher',
        priority: 20,
        match: { pattern: 'https?://[^\\s<]+', flags: 'g', capture_groups: { url: 0 } },
        render: {
          type: 'link',
          url_template: '{{url}}',
          label_template: '{{url}}',
          target: '_blank',
          class: 'auto-url',
        } as any,
      })

      const input = escapeHTML('NI-30000')
      const result = applyRenderRules(input, [jiraRule, urlRule])

      // 规则 A 生成的卡片内的 URL 不应被规则 B 二次匹配
      // 期望：只有 1 个 <a>（规则 A 的）
      const anchorCount = (result.match(/<a /g) || []).length
      expect(anchorCount).toBe(1)
      expect(result).toContain('jira-card')
      expect(result).not.toContain('auto-url')
    })
  })

  describe('相同优先级的规则', () => {
    it('同优先级规则按数组顺序执行（先到先得）', () => {
      const ruleA = makeRule({
        id: 'rule_a',
        priority: 10,
        render: { ...makeRule().render, class: 'rule-a-card' } as any,
      })
      const ruleB = makeRule({
        id: 'rule_b',
        priority: 10,
        render: { ...makeRule().render, class: 'rule-b-card' } as any,
      })

      const input = escapeHTML('NI-30000')
      const resultA = applyRenderRules(input, [ruleA, ruleB])
      const resultB = applyRenderRules(input, [ruleB, ruleA])

      // 顺序不同，结果不同：先出现的规则匹配
      expect(resultA).toContain('rule-a-card')
      expect(resultA).not.toContain('rule-b-card')

      expect(resultB).toContain('rule-b-card')
      expect(resultB).not.toContain('rule-a-card')
    })
  })

  describe('多规则与 <a> 标签保护的组合', () => {
    it('已有 <a> 内的内容对所有规则都跳过', () => {
      const jiraRule = makeRule({ id: 'jira', priority: 10 })
      const mentionRule = makeRule({
        id: 'mention',
        priority: 20,
        match: { pattern: '@([一-鿿A-Za-z0-9_]+)', flags: 'g', capture_groups: { name: 1 } },
        render: {
          type: 'text_chip',
          url_template: '',
          label_template: '@{{name}}',
          class: 'mention-chip',
        } as any,
      })

      // <a> 标签内同时有 NI-30000 和 @张三
      const linked = '外部 NI-10001 <a href="http://x.com/NI-30000">@张三</a> 外部 @李四'
      const result = applyRenderRules(linked, [jiraRule, mentionRule])

      // <a> 内的 NI-30000 和 @张三 都不被匹配
      expect(result).not.toContain('href="http://jira.xxx.com/NI/NI-30000"')
      expect(result).not.toContain('render-chip mention-chip">@张三</span>')

      // <a> 外的 NI-10001 和 @李四 被匹配
      expect(result).toContain('href="http://jira.xxx.com/NI/NI-10001"')
      expect(result).toContain('@李四')
      expect(result).toContain('mention-chip')
    })
  })
})

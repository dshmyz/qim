import { describe, it, expect } from 'vitest'
import { detectRuleConflicts } from '@/utils/ruleConflictDetector'
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

describe('detectRuleConflicts', () => {
  describe('无冲突', () => {
    it('空规则列表返回空数组', () => {
      expect(detectRuleConflicts([])).toEqual([])
    })

    it('单条规则返回空数组', () => {
      const rule = makeRule()
      expect(detectRuleConflicts([rule])).toEqual([])
    })

    it('模式完全不同的规则无冲突', () => {
      const jiraRule = makeRule({ id: 'jira' })
      const mentionRule = makeRule({
        id: 'mention',
        match: { pattern: '@([一-鿿A-Za-z0-9_]+)', flags: 'g', capture_groups: { name: 1 } },
      })
      expect(detectRuleConflicts([jiraRule, mentionRule])).toEqual([])
    })
  })

  describe('模板污染：规则 A 生成的 HTML 被规则 B 匹配', () => {
    it('规则 A 的 label_template 含规则 B 的模式', () => {
      // 规则 A: Jira 工单号，label 是 "NI-30000"
      const jiraRule = makeRule({ id: 'jira', priority: 10 })
      // 规则 B: 匹配纯数字
      const numberRule = makeRule({
        id: 'number',
        priority: 20,
        match: { pattern: '\\b(\\d{4,6})\\b', flags: 'g', capture_groups: { num: 1 } },
        render: {
          type: 'text_chip',
          url_template: '',
          label_template: '#{{num}}',
          class: 'number-chip',
        } as any,
      })

      const conflicts = detectRuleConflicts([jiraRule, numberRule])

      // 规则 B 的正则会匹配规则 A 生成的 label（NI-30000 中的 30000）
      expect(conflicts.length).toBeGreaterThan(0)
      const conflict = conflicts.find(
        c => c.ruleAId === 'jira' && c.ruleBId === 'number'
      )
      expect(conflict).toBeDefined()
      expect(conflict!.type).toBe('template_pollution')
      expect(conflict!.description).toContain('30000')
    })

    it('规则 A 的 url_template 含规则 B 的模式', () => {
      // 规则 A: 生成 URL http://jira.xxx.com/NI/NI-30000
      const jiraRule = makeRule({ id: 'jira', priority: 10 })
      // 规则 B: 匹配 URL
      const urlRule = makeRule({
        id: 'url',
        priority: 20,
        match: { pattern: 'https?://[^\\s<]+', flags: 'g', capture_groups: { url: 0 } },
      })

      const conflicts = detectRuleConflicts([jiraRule, urlRule])

      // 规则 B 的正则会匹配规则 A 生成的 URL
      const conflict = conflicts.find(
        c => c.ruleAId === 'jira' && c.ruleBId === 'url'
      )
      expect(conflict).toBeDefined()
      expect(conflict!.type).toBe('template_pollution')
    })
  })

  describe('子串包含：规则 A 的 pattern 是规则 B 的子串', () => {
    it('数字规则是工单号规则的子串', () => {
      const jiraRule = makeRule({
        id: 'jira',
        priority: 10,
        match: {
          pattern: '\\b([A-Z]{2,6})-(\\d{1,6})\\b',
          flags: 'g',
          capture_groups: { project: 1, number: 2 },
        },
      })
      const numberRule = makeRule({
        id: 'number',
        priority: 20,
        match: {
          pattern: '\\b(\\d{4,6})\\b',
          flags: 'g',
          capture_groups: { num: 1 },
        },
      })

      const conflicts = detectRuleConflicts([jiraRule, numberRule])

      const conflict = conflicts.find(c => c.type === 'substring_overlap')
      expect(conflict).toBeDefined()
    })

    it('完全相同的模式报告冲突', () => {
      const ruleA = makeRule({ id: 'rule_a' })
      const ruleB = makeRule({ id: 'rule_b' })

      const conflicts = detectRuleConflicts([ruleA, ruleB])

      const conflict = conflicts.find(
        c => c.ruleAId === 'rule_a' && c.ruleBId === 'rule_b'
      )
      expect(conflict).toBeDefined()
      expect(conflict!.type).toBe('duplicate_pattern')
    })
  })

  describe('冲突结果结构', () => {
    it('返回结构包含 ruleAId/ruleBId/type/description', () => {
      const jiraRule = makeRule({ id: 'jira', priority: 10 })
      const numberRule = makeRule({
        id: 'number',
        priority: 20,
        match: { pattern: '\\b(\\d{4,6})\\b', flags: 'g', capture_groups: { num: 1 } },
      })

      const conflicts = detectRuleConflicts([jiraRule, numberRule])
      const conflict = conflicts[0]

      expect(conflict).toHaveProperty('ruleAId')
      expect(conflict).toHaveProperty('ruleBId')
      expect(conflict).toHaveProperty('type')
      expect(conflict).toHaveProperty('description')
      expect(typeof conflict.description).toBe('string')
      expect(conflict.description.length).toBeGreaterThan(0)
    })

    it('不报告自身与自身的冲突', () => {
      const rule = makeRule({ id: 'solo' })
      const conflicts = detectRuleConflicts([rule])
      expect(conflicts).toEqual([])
    })
  })

  describe('多条规则的组合冲突', () => {
    it('三条规则中能检测出多对冲突', () => {
      const jiraRule = makeRule({ id: 'jira', priority: 10 })
      const numberRule = makeRule({
        id: 'number',
        priority: 20,
        match: { pattern: '\\b(\\d{4,6})\\b', flags: 'g', capture_groups: { num: 1 } },
      })
      const urlRule = makeRule({
        id: 'url',
        priority: 30,
        match: { pattern: 'https?://[^\\s<]+', flags: 'g', capture_groups: { url: 0 } },
      })

      const conflicts = detectRuleConflicts([jiraRule, numberRule, urlRule])

      // jira vs number（子串 + 模板污染）
      // jira vs url（模板污染：jira 生成的 URL 被 url 规则匹配）
      expect(conflicts.length).toBeGreaterThanOrEqual(2)

      const jiraNumberConflicts = conflicts.filter(
        c => (c.ruleAId === 'jira' && c.ruleBId === 'number') ||
             (c.ruleAId === 'number' && c.ruleBId === 'jira')
      )
      expect(jiraNumberConflicts.length).toBeGreaterThan(0)
    })
  })
})

import { describe, it, expect } from 'vitest'
import { applyRenderRules } from '@/utils/renderRules'
import { escapeHTML, sanitizeHTML } from '@/utils/sanitize'
import type { CompiledRule } from '@/stores/renderRules'

// 构造与后端 seed 数据一致的 3 条规则
function buildTestRules(): CompiledRule[] {
  return [
    {
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
        title_template: '查看 Jira 工单 {{project}}-{{number}}',
        icon: 'fab fa-jira',
        target: '_blank',
        class: 'jira-ticket-card',
      },
      compiledRegex: new RegExp('\\b([A-Z]{2,6})-(\\d{1,6})\\b', 'g'),
    },
    {
      id: 'github_pr_link',
      name: 'GitHub PR 链接化',
      enabled: true,
      priority: 20,
      scope: { groups: ['*'], exclude_groups: [], conversation_types: ['single', 'group', 'discussion'] },
      match: {
        pattern: '#PR(\\d+)',
        flags: 'g',
        capture_groups: { number: 1 },
      },
      render: {
        type: 'link',
        url_template: 'https://github.com/org/repo/pull/{{number}}',
        label_template: '#PR{{number}}',
        title_template: '查看 GitHub PR #{{number}}',
        target: '_blank',
        class: 'github-pr-link',
      },
      compiledRegex: new RegExp('#PR(\\d+)', 'g'),
    },
    {
      id: 'mention_highlight',
      name: '@提及高亮标签',
      enabled: true,
      priority: 30,
      scope: { groups: ['*'], exclude_groups: [], conversation_types: ['single', 'group', 'discussion'] },
      match: {
        pattern: '@([一-鿿A-Za-z0-9_]+)',
        flags: 'g',
        capture_groups: { name: 1 },
      },
      render: {
        type: 'text_chip',
        url_template: '',
        label_template: '@{{name}}',
        class: 'mention-chip',
      },
      compiledRegex: new RegExp('@([一-鿿A-Za-z0-9_]+)', 'g'),
    },
  ]
}

// 基线：不应用渲染规则时的开销（escapeHTML + sanitizeHTML）
function baselineRender(text: string): string {
  const escaped = escapeHTML(text)
  return sanitizeHTML(escaped)
}

// 优化后管线：escapeHTML + applyRenderRules（无内部 sanitize）+ 外层 sanitizeHTML
// sanitize 次数：1 次
function renderOptimized(text: string, rules: CompiledRule[]): string {
  const escaped = escapeHTML(text)
  const html = applyRenderRules(escaped, rules)
  return sanitizeHTML(html)
}

// 优化前管线模拟：escapeHTML + applyRenderRules（含内部 sanitize）+ 外层 sanitizeHTML
// sanitize 次数：2 次（内部 + 外层）
function renderWithDoubleSanitize(text: string, rules: CompiledRule[]): string {
  const escaped = escapeHTML(text)
  // 模拟优化前 applyRenderRules 内部的 sanitize
  const raw = applyRenderRules(escaped, rules)
  const innerSanitized = sanitizeHTML(raw)
  // 外层 sanitize
  return sanitizeHTML(innerSanitized)
}

// 生成测试文本
function shortTextNoMatch(): string {
  return '今天天气不错，我们去吃饭吧。这个需求大概需要三天完成，包括前端和后端的联调。'
}

function shortTextWithMatch(): string {
  return '请看 NI-30000 这个工单，另外 #PR123 也需要合并。@张三 你来跟进一下。'
}

function longTextNoMatch(): string {
  const base = '这是一段普通的聊天消息，不包含任何需要特殊渲染的内容。我们在讨论项目进度和下一步计划。'
  return base.repeat(20) // ~1200 字符
}

function longTextWithMatch(): string {
  const matches = 'NI-10001 #PR200 @李四 NI-10002 #PR201 @王五 '
  const filler = '这是一段讨论项目进度的消息，需要大家确认。'
  let text = ''
  for (let i = 0; i < 20; i++) {
    text += filler + matches
  }
  return text // ~1400 字符，60 个匹配
}

function benchmark(fn: () => void, iterations: number): number {
  // 预热
  for (let i = 0; i < 10; i++) fn()

  // 正式测量
  const start = performance.now()
  for (let i = 0; i < iterations; i++) fn()
  const end = performance.now()

  return (end - start) / iterations
}

describe('applyRenderRules 性能基准', () => {
  const rules = buildTestRules()
  const ITERATIONS = 100

  it('短文本有匹配：优化前(2次sanitize) vs 优化后(1次sanitize)', () => {
    const text = shortTextWithMatch()
    const before = benchmark(() => renderWithDoubleSanitize(text, rules), ITERATIONS)
    const after = benchmark(() => renderOptimized(text, rules), ITERATIONS)
    const improvement = ((before - after) / before * 100).toFixed(1)

    console.log(`[短文本有匹配] 优化前: ${before.toFixed(4)}ms, 优化后: ${after.toFixed(4)}ms, 提升: ${improvement}%`)

    expect(after).toBeGreaterThan(0)
  })

  it('长文本有匹配：优化前(2次sanitize) vs 优化后(1次sanitize)', () => {
    const text = longTextWithMatch()
    const before = benchmark(() => renderWithDoubleSanitize(text, rules), ITERATIONS)
    const after = benchmark(() => renderOptimized(text, rules), ITERATIONS)
    const improvement = ((before - after) / before * 100).toFixed(1)

    console.log(`[长文本有匹配] 优化前: ${before.toFixed(4)}ms, 优化后: ${after.toFixed(4)}ms, 提升: ${improvement}%`)

    expect(after).toBeGreaterThan(0)
  }, 30000)

  it('无匹配场景：优化前 vs 优化后（应无差异）', () => {
    const text = longTextNoMatch()
    const before = benchmark(() => renderWithDoubleSanitize(text, rules), ITERATIONS)
    const after = benchmark(() => renderOptimized(text, rules), ITERATIONS)

    console.log(`[长文本无匹配] 优化前: ${before.toFixed(4)}ms, 优化后: ${after.toFixed(4)}ms`)

    expect(after).toBeGreaterThan(0)
  })

  it('模拟20条消息列表：优化前 vs 优化后', () => {
    const messages = [
      shortTextNoMatch(),
      shortTextWithMatch(),
      longTextNoMatch(),
      longTextWithMatch(),
    ]
    const messageCount = 20

    // 优化前（2次 sanitize）
    const beforeStart = performance.now()
    for (let i = 0; i < messageCount; i++) {
      renderWithDoubleSanitize(messages[i % messages.length], rules)
    }
    const beforeTotal = performance.now() - beforeStart

    // 优化后（1次 sanitize）
    const afterStart = performance.now()
    for (let i = 0; i < messageCount; i++) {
      renderOptimized(messages[i % messages.length], rules)
    }
    const afterTotal = performance.now() - afterStart
    const improvement = ((beforeTotal - afterTotal) / beforeTotal * 100).toFixed(1)

    console.log(`[20条消息] 优化前: ${beforeTotal.toFixed(2)}ms, 优化后: ${afterTotal.toFixed(2)}ms, 提升: ${improvement}%`)

    expect(afterTotal).toBeGreaterThan(0)
  }, 30000)
})

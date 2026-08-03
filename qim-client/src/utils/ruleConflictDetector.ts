import type { CompiledRule } from '../stores/renderRules'

/**
 * 规则冲突类型
 */
export type ConflictType = 'template_pollution' | 'substring_overlap' | 'duplicate_pattern'

/**
 * 规则冲突描述
 */
export interface RuleConflict {
  /** 生成 HTML 的规则 ID（被污染方） */
  ruleAId: string
  /** 匹配到 ruleA 生成内容的规则 ID（污染方） */
  ruleBId: string
  /** 冲突类型 */
  type: ConflictType
  /** 人类可读的冲突描述 */
  description: string
}

/**
 * 生成规则的样例 HTML（用占位符填充模板）
 * 用于检测其他规则是否会匹配它生成的内容
 */
function generateSampleHtml(rule: CompiledRule): string {
  // 用样例值填充模板
  const sampleCtx: Record<string, string> = {}
  for (const [name, idx] of Object.entries(rule.match.capture_groups)) {
    // 给不同捕获组生成样例值
    sampleCtx[name] = `SAMPLE${idx}`
  }
  const sampleLabel = fillTemplate(rule.render.label_template, sampleCtx)
  const sampleUrl = fillTemplate(rule.render.url_template || '', sampleCtx)
  return `${sampleUrl} ${sampleLabel}`
}

function fillTemplate(tmpl: string, ctx: Record<string, string>): string {
  let result = tmpl
  for (const [k, v] of Object.entries(ctx)) {
    result = result.split(`{{${k}}}`).join(v)
  }
  return result
}

/**
 * 用样例输入测试规则 A 的正则，生成真实匹配样例
 * 包含 label 和 url 两部分，用于检测模板污染和子串重叠
 */
function generateRealSample(rule: CompiledRule): string {
  // 构造一个能被规则 A 匹配的样例文本
  // 根据 capture_groups 反推样例
  const samples: Record<string, string> = {}
  for (const [name, idx] of Object.entries(rule.match.capture_groups)) {
    if (idx === 1) samples[name] = 'NI'
    else if (idx === 2) samples[name] = '30000'
    else samples[name] = `sample${idx}`
  }
  // 拼接 label 和 url，覆盖两种渲染输出
  const sampleLabel = fillTemplate(rule.render.label_template, samples)
  const sampleUrl = fillTemplate(rule.render.url_template || '', samples)
  return `${sampleUrl} ${sampleLabel}`
}

/**
 * 检测两条规则之间的冲突
 */
function detectPairConflicts(a: CompiledRule, b: CompiledRule): RuleConflict[] {
  const conflicts: RuleConflict[] = []

  // 1. 重复模式
  if (a.match.pattern === b.match.pattern) {
    conflicts.push({
      ruleAId: a.id,
      ruleBId: b.id,
      type: 'duplicate_pattern',
      description: `规则 "${a.name}" 和 "${b.name}" 使用了相同的正则模式，只会生效优先级更高（priority 更小）的那条`,
    })
    return conflicts
  }

  // 2. 模板污染：规则 A 生成的 HTML 会被规则 B 的正则匹配
  const sampleHtml = generateRealSample(a)
  try {
    const regexB = new RegExp(b.match.pattern, b.match.flags || 'g')
    regexB.lastIndex = 0
    const match = regexB.exec(sampleHtml)
    if (match && match[0]) {
      // 排除误报：如果匹配到的只是占位符本身，不算冲突
      if (!match[0].includes('SAMPLE') && !match[0].includes('sample')) {
        conflicts.push({
          ruleAId: a.id,
          ruleBId: b.id,
          type: 'template_pollution',
          description: `规则 "${a.name}" 生成的 "${match[0]}" 会被规则 "${b.name}" 的正则匹配，可能导致重复渲染（代码已隔离，但建议检查配置）`,
        })
      }
    }
  } catch {
    // 正则编译失败，跳过
  }

  // 3. 子串包含：A 的样例匹配也会被 B 单独匹配
  //    用 A 的正则匹配样例文本，再用 B 的正则匹配同一段文本
  try {
    const regexA = new RegExp(a.match.pattern, a.match.flags || 'g')
    regexA.lastIndex = 0
    const matchA = regexA.exec(sampleHtml)
    if (matchA && matchA[0]) {
      const matchedText = matchA[0]
      const regexB = new RegExp(b.match.pattern, b.match.flags || 'g')
      regexB.lastIndex = 0
      const matchB = regexB.exec(matchedText)
      if (matchB && matchB[0] && matchB[0] !== matchedText) {
        // B 能在 A 的匹配结果内部匹配到子串
        conflicts.push({
          ruleAId: a.id,
          ruleBId: b.id,
          type: 'substring_overlap',
          description: `规则 "${a.name}" 匹配 "${matchedText}"，其中 "${matchB[0]}" 也会被规则 "${b.name}" 匹配，存在子串重叠风险`,
        })
      }
    }
  } catch {
    // 正则编译失败，跳过
  }

  return conflicts
}

/**
 * 检测规则列表中的所有冲突
 *
 * 检测三种冲突：
 * 1. duplicate_pattern: 两条规则使用相同正则
 * 2. template_pollution: 规则 A 生成的 HTML 会被规则 B 的正则匹配
 * 3. substring_overlap: 规则 A 的匹配结果内部包含规则 B 的匹配
 *
 * @param rules 规则列表
 * @returns 冲突列表
 */
export function detectRuleConflicts(rules: CompiledRule[]): RuleConflict[] {
  const conflicts: RuleConflict[] = []

  for (let i = 0; i < rules.length; i++) {
    for (let j = 0; j < rules.length; j++) {
      if (i === j) continue
      const pairConflicts = detectPairConflicts(rules[i], rules[j])
      conflicts.push(...pairConflicts)
    }
  }

  // 去重：同方向的 (A→B) 冲突只保留一条
  // 但保留双向（A→B 和 B→A 是不同的冲突）
  const seen = new Set<string>()
  return conflicts.filter(c => {
    const key = `${c.ruleAId}->${c.ruleBId}:${c.type}`
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
}

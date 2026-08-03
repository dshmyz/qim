import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { request } from '../composables/useRequest'

export interface RenderRule {
  id: string
  name: string
  enabled: boolean
  priority: number
  scope: {
    groups: string[]
    exclude_groups: string[]
    conversation_types: string[]
  }
  match: {
    pattern: string
    flags: string
    capture_groups: Record<string, number>
  }
  render: {
    type: 'link' | 'link_card' | 'text_chip'
    url_template: string
    label_template: string
    title_template?: string
    icon?: string
    target?: '_blank' | '_self'
    class?: string
  }
}

// 预编译的规则（含 RegExp 对象，避免每条消息都 new RegExp）
export interface CompiledRule extends RenderRule {
  compiledRegex: RegExp
}

// glob 通配匹配（支持 external_* 这类）
function matchGlob(pattern: string, str: string): boolean {
  const re = new RegExp('^' + pattern.replace(/[.+^${}()|[\]\\]/g, '\\$&').replace(/\*/g, '.*') + '$')
  return re.test(str)
}

export const useRenderRulesStore = defineStore('renderRules', () => {
  const rules = ref<CompiledRule[]>([])
  const version = ref<number>(0)
  const loaded = ref(false)

  // 按优先级排序的启用规则
  const activeRules = computed(() =>
    rules.value
      .filter(r => r.enabled)
      .sort((a, b) => a.priority - b.priority)
  )

  // 拉取规则（带版本号增量同步）
  async function fetchRules(): Promise<void> {
    try {
      const res = await request<{ rules: RenderRule[]; version: number }>(
        `/api/v1/render-rules`,
        { method: 'GET', params: { version: version.value } }
      )
      if (res && res.rules) {
        // 预编译正则
        rules.value = res.rules.map(rule => ({
          ...rule,
          compiledRegex: new RegExp(rule.match.pattern, rule.match.flags || 'g')
        }))
        version.value = res.version
        loaded.value = true
      }
    } catch (e) {
      // 304 表示版本未变化，无需更新，静默处理
      if (e instanceof Error && e.message.includes('304')) return
      // 其他错误静默失败，不影响消息渲染
      console.warn('[renderRules] 拉取失败', e)
    }
  }

  // 判断某会话是否应用某规则（作用域过滤）
  function rulesForConversation(convType: string, groupId?: string): CompiledRule[] {
    return activeRules.value.filter(rule => {
      // 会话类型过滤
      if (rule.scope.conversation_types?.length &&
          !rule.scope.conversation_types.includes(convType)) {
        return false
      }
      // 群组黑名单
      if (groupId && rule.scope.exclude_groups?.length) {
        for (const ex of rule.scope.exclude_groups) {
          if (matchGlob(ex, groupId)) return false
        }
      }
      // 群组白名单
      if (groupId && rule.scope.groups?.length && !rule.scope.groups.includes('*')) {
        let matched = false
        for (const g of rule.scope.groups) {
          if (matchGlob(g, groupId)) { matched = true; break }
        }
        if (!matched) return false
      }
      return true
    })
  }

  return { rules, version, loaded, activeRules, fetchRules, rulesForConversation }
})

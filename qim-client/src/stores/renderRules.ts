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

  // 单飞守卫：同一时间只允许一个拉取请求，避免多条消息渲染时并发触发
  // /render-rules 造成请求/日志洪泛。拉取完成后清空，下一次需要时再拉。
  let fetchPromise: Promise<void> | null = null

  // 按优先级排序的启用规则
  const activeRules = computed(() =>
    rules.value
      .filter(r => r.enabled)
      .sort((a, b) => a.priority - b.priority)
  )

  // 拉取规则（带版本号增量同步，单飞并发去重）
  function fetchRules(): Promise<void> {
    if (fetchPromise) return fetchPromise
    fetchPromise = doFetchRules().finally(() => { fetchPromise = null })
    return fetchPromise
  }

  async function doFetchRules(): Promise<void> {
    try {
      const res = await request<{ rules?: RenderRule[]; version?: number; data?: { rules?: RenderRule[]; version?: number } }>(
        `/api/v1/render-rules`,
        { method: 'GET', params: { version: version.value } }
      )
      // 响应封装要么是 { code, data: { rules, version } }（request 返回整个 body），
      // 要么历史版本直接平铺 { rules, version }。两者都兼容。
      const data = res?.data ?? res
      if (data && Array.isArray(data.rules)) {
        // 预编译正则
        rules.value = data.rules.map(rule => ({
          ...rule,
          compiledRegex: new RegExp(rule.match.pattern, rule.match.flags || 'g')
        }))
        if (typeof data.version === 'number') version.value = data.version
        loaded.value = true
      }
    } catch (e) {
      // 304 表示版本未变化，说明本地已是最新规则，视为已加载。
      // 只有置 loaded 才不会再被 TextMessage 每次渲染触发重复拉取。
      if (e instanceof Error && e.message.includes('304')) {
        // 防御：若本地规则为空却收到 304（缓存了"无规则"的旧版本），
        // 不能视为已加载——否则会一直维持空规则、永不重新拉取，导致渲染永远失效。
        // 此时清空版本号强制下一次携带 version=0 全量拉取。
        if (rules.value.length === 0) {
          version.value = 0
          console.warn('[renderRules] 304 但本地无规则，重置版本号以重新全量拉取')
        }
        loaded.value = true
        return
      }
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

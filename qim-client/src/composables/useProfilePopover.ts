import { computed, onScopeDispose, ref } from 'vue'
import { fetchUserProfile } from './useUserProfileInfo'

/**
 * 共享「随行资料小卡」行为（ChatHeader 单聊/机器人头像 → 群成员名片同款交互）。
 * 原本 ChatHeader 把 MemberSidebar 的成员名片复刻了一份（状态机/异步富化/悬停收起/
 * Teleport/样式约 85% 重复），这里抽成单一实现：
 * - 点击头像弹出，重复点击同一身份收起
 * - 悬停卡片保持，移开 200ms 后收起
 * - 列表数据不完整时异步 fetchUserProfile 富化，失败静默保留已有数据
 * - bot/系统身份没有可富化的个人资料字段，跳过无意义的请求
 */

export interface ProfileMember {
  id: string
  name: string
  avatar: string
  username?: string
  email?: string
  mobile?: string
  department?: string
  position?: string
  signature?: string
  status?: string
  ip?: string
  type?: string
  user?: Record<string, any>
}

export interface ProfilePopoverState {
  member: ProfileMember
  anchor: { left: number; top: number; bottom: number; width: number }
}

/** 移开即消失的延迟（悬停卡片期间取消） */
export const PROFILE_POPOVER_HIDE_DELAY = 200
/** 卡片宽度，与 UserProfileCard 的 CSS 保持一致 */
export const PROFILE_POPOVER_WIDTH = 248
const PROFILE_POPOVER_GAP = 8
const PROFILE_POPOVER_MAX_HEIGHT = 360

export function useProfilePopover(buildMember: () => ProfileMember | null) {
  const popoverState = ref<ProfilePopoverState | null>(null)

  // 列表数据不完整（无部门/职位/签名/IP 等），异步拉取详情富化；失败静默保留已有数据。
  // bot/系统身份没有个人资料字段，跳过请求。
  async function enrich(member: ProfileMember) {
    if (member.type === 'bot' || member.type === 'system') return
    const { profile, success } = await fetchUserProfile(member.id, member)
    const st = popoverState.value
    if (!success || !st || st.member.id !== member.id) return
    const m = st.member
    st.member = {
      ...m,
      name: m.name || profile.name,
      username: m.username || profile.username,
      department: m.department || profile.department,
      position: m.position || profile.position,
      signature: m.signature || profile.signature,
      status: m.status || profile.status,
      ip: m.ip || profile.ip,
      email: m.email || profile.email,
      mobile: m.mobile || profile.mobile
    }
  }

  // 打开/切换：重复点击同一身份收起；否则以触发元素锚点弹出，并异步富化详情。
  function open(e: MouseEvent) {
    const member = buildMember()
    if (!member) return
    if (popoverState.value?.member.id === member.id) {
      popoverState.value = null
      return
    }
    const el = e.currentTarget as HTMLElement
    const rect = el.getBoundingClientRect()
    popoverState.value = {
      member,
      anchor: { left: rect.left, top: rect.top, bottom: rect.bottom, width: rect.width }
    }
    void enrich(member)
  }

  // 跟随头像左缘，优先放头像下方；下方放不下时上移到头像上方
  const popoverStyle = computed(() => {
    const st = popoverState.value
    if (!st) return {}
    const { anchor } = st
    const left = Math.min(Math.max(anchor.left, 8), window.innerWidth - PROFILE_POPOVER_WIDTH - 8)
    let top = anchor.bottom + PROFILE_POPOVER_GAP
    if (top + PROFILE_POPOVER_MAX_HEIGHT > window.innerHeight) {
      top = Math.max(8, anchor.top - PROFILE_POPOVER_MAX_HEIGHT - PROFILE_POPOVER_GAP)
    }
    return { left: `${left}px`, top: `${top}px` }
  })

  let hideTimer: ReturnType<typeof setTimeout> | null = null

  // 移开即消失：离开头像/卡片 200ms 后收起，悬停卡片期间保持
  function scheduleHide() {
    if (hideTimer) clearTimeout(hideTimer)
    hideTimer = setTimeout(() => {
      popoverState.value = null
      hideTimer = null
    }, PROFILE_POPOVER_HIDE_DELAY)
  }

  function cancelHide() {
    if (hideTimer) {
      clearTimeout(hideTimer)
      hideTimer = null
    }
  }

  onScopeDispose(() => {
    if (hideTimer) clearTimeout(hideTimer)
  })

  return { popoverState, popoverStyle, open, scheduleHide, cancelHide }
}
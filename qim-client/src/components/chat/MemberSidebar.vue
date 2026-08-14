<template>
  <div class="members-sidebar" :class="{ 'collapsed': !isExpanded }">
    <div class="sidebar-header-container">
      <div v-if="isExpanded" class="members-header">
        <div class="header-content">
          <button class="action-btn" @click="handleToggleExpanded" title="收起成员列表">
            <i class="fas fa-chevron-left"></i>
          </button>
          <h3>群成员 ({{ members.length }})</h3>
        </div>
        <div class="header-actions">
          <button class="action-btn" @click="handleToggleMemberSearch" title="搜索成员">
            <i class="fas fa-search"></i>
          </button>
        </div>
      </div>
      <div v-else class="collapsed-header">
        <button class="action-btn" @click="handleToggleExpanded" title="展开成员列表">
          <i class="fas fa-chevron-right"></i>
        </button>
      </div>
    </div>
    <div v-if="showSearch && isExpanded" class="members-search">
      <input
        v-model="searchQueryLocal"
        type="text"
        placeholder="搜索群成员..."
        class="member-search-input"
        @focus="handleSearchFocus"
      />
    </div>
    <div v-if="isExpanded" class="members-content">
      <div v-for="member in filteredMembers" :key="member.id" class="member-item" :class="{ 'bot-member': isBotType(member.type) }" @contextmenu.prevent="handleMemberContextMenu($event, member)" @click="handleMemberClick(member, $event)" @dblclick="handleStartPrivateChat(member)" @mouseleave="scheduleHidePopover">
        <div class="member-avatar-wrapper">
          <Avatar
            :src="member.avatar"
            :name="member.name || '未知用户'"
            :alt="member.name || '未知用户'"
            size="sm"
            class="member-avatar"
          />
          <span v-if="member.status" class="member-status-dot" :class="member.status"></span>
        </div>
        <div class="member-detail">
          <div class="member-name-row">
            <span class="member-name">{{ member.name || '未知用户' }}</span>
            <span v-if="member.role === 'owner'" class="member-role-tag owner" title="群主">群主</span>
            <span v-else-if="member.role === 'admin'" class="member-role-tag admin" title="管理员">管理</span>
            <span v-if="isBotType(member.type)" class="member-role-tag bot" title="AI助手"><i class="fas fa-robot"></i></span>
          </div>
          <span v-if="member.position" class="member-position">{{ member.position }}</span>
        </div>
      </div>
    </div>
    <div v-else class="collapsed-avatars">
      <div
        v-for="member in filteredMembers"
        :key="member.id"
        class="collapsed-avatar-item"
        @contextmenu.prevent="handleCollapsedMemberContextMenu($event, member)"
        @click="handleMemberClick(member, $event)"
        @dblclick="handleStartPrivateChat(member)"
        @mouseleave="scheduleHidePopover"
      >
        <Avatar
          :src="member.avatar"
          :name="member.name || '未知用户'"
          :alt="member.name || '未知用户'"
          :title="member.name || '未知用户'"
          size="sm"
          class="collapsed-avatar"
        />
        <span v-if="member.role === 'owner'" class="collapsed-role owner"><i class="fas fa-crown"></i></span>
        <span v-else-if="member.role === 'admin'" class="collapsed-role admin"><i class="fas fa-user-shield"></i></span>
      </div>
    </div>

    <!-- 轻量成员名片：点击成员出现，跟随成员位置，移开即消失 -->
    <Teleport to="body">
      <div
        v-if="popoverState"
        class="member-profile-popover"
        :style="popoverStyle"
        @mouseenter="cancelHidePopover"
        @mouseleave="scheduleHidePopover"
      >
        <div class="popover-header">
          <div class="popover-avatar-wrap">
            <Avatar
              :src="popoverState.member.avatar"
              :name="displayName"
              :alt="displayName"
              size="md"
              class="popover-avatar"
            />
            <span v-if="popoverState.member.status" class="popover-status-dot" :class="popoverState.member.status"></span>
          </div>
          <div class="popover-identity">
            <div class="popover-name-row">
              <span class="popover-name">{{ displayName }}</span>
              <span v-if="popoverState.member.role === 'owner'" class="popover-role owner" title="群主">群主</span>
              <span v-else-if="popoverState.member.role === 'admin'" class="popover-role admin" title="管理员">管理</span>
              <span v-if="isBotType(popoverState.member.type)" class="popover-role bot" title="AI助手"><i class="fas fa-robot"></i></span>
            </div>
            <span class="popover-status-text">{{ popoverStatusText }}</span>
          </div>
        </div>

        <div v-if="popoverInfoRows.length" class="popover-info">
          <div v-for="row in popoverInfoRows" :key="row.icon" class="popover-info-row">
            <i :class="row.icon"></i>
            <span>{{ row.text }}</span>
          </div>
        </div>

        <div class="popover-actions">
          <button class="popover-action-btn" @click.stop="handlePopoverPrivateChat">
            <i class="fas fa-comment"></i>
            <span>{{ isBotType(popoverState.member.type) ? '开始对话' : '发起私聊' }}</span>
          </button>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onUnmounted } from 'vue'
import Avatar from '../shared/Avatar.vue'
import { isVisibleGroupMember } from '../../composables/useGroup'
import { fetchUserProfile } from '../../composables/useUserProfileInfo'

interface Member {
  id: string
  name: string
  avatar: string
  role?: 'owner' | 'admin' | 'member'
  type?: string
  position?: string
  department?: string
  status?: string
  signature?: string
  ip?: string
  username?: string
  nickname?: string
  email?: string
  mobile?: string
  disabled?: boolean
  is_disabled?: boolean
  is_deleted?: boolean
  deletedAt?: string | number
  deleted_at?: string | number
  user?: Record<string, any>
}

interface Props {
  members: Member[]
  isExpanded: boolean
  showSearch: boolean
  searchQuery: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'toggle-expanded'): void
  (e: 'toggle-member-search'): void
  (e: 'search-focus'): void
  (e: 'show-member-context-menu', event: MouseEvent, member: Member): void
  (e: 'start-private-chat', member: Member): void
  (e: 'update:searchQuery', value: string): void
}>()

const searchQueryLocal = computed({
  get: () => props.searchQuery,
  set: (val) => emit('update:searchQuery', val)
})

const rolePriority: Record<string, number> = { owner: 3, admin: 2, member: 1 }

// 非人类成员（机器人 bot、系统助手 system）——样式与私聊约束与机器人一致
const isBotType = (type?: string) => type === 'bot' || type === 'system'

const filteredMembers = computed(() => {
  let members = (props.members || []).filter(isVisibleGroupMember)

  members = [...members].sort((a, b) => {
    const aPriority = rolePriority[a.role || 'member'] || 1
    const bPriority = rolePriority[b.role || 'member'] || 1
    
    if (aPriority !== bPriority) {
      return bPriority - aPriority
    }
    
    return (a.name || '').localeCompare(b.name || '')
  })
  
  if (props.searchQuery) {
    const query = props.searchQuery.toLowerCase()
    members = members.filter(member => 
      (member.name || '').toLowerCase().includes(query)
    )
  }
  
  return members
})

const handleToggleExpanded = () => {
  emit('toggle-expanded')
}

const handleToggleMemberSearch = () => {
  emit('toggle-member-search')
}

const handleSearchFocus = () => {
  emit('search-focus')
}

const handleMemberContextMenu = (event: MouseEvent, member: Member) => {
  emit('show-member-context-menu', event, member)
}

const handleCollapsedMemberContextMenu = (event: MouseEvent, member: Member) => {
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const anchoredEvent = new MouseEvent(event.type, {
    bubbles: event.bubbles,
    cancelable: event.cancelable,
    clientX: rect.right,
    clientY: rect.top,
    screenX: event.screenX,
    screenY: event.screenY,
    button: event.button,
    buttons: event.buttons,
    ctrlKey: event.ctrlKey,
    shiftKey: event.shiftKey,
    altKey: event.altKey,
    metaKey: event.metaKey,
  })

  emit('show-member-context-menu', anchoredEvent, member)
}

const handleStartPrivateChat = (member: Member) => {
  popoverState.value = null
  emit('start-private-chat', member)
}

// ---- 轻量成员名片（跟随点击位置，移开即消失） ----
const POPOVER_WIDTH = 248
const POPOVER_GAP = 10
const HIDE_DELAY = 200

interface PopoverState {
  member: Member
  anchor: { left: number; top: number; bottom: number }
}

const popoverState = ref<PopoverState | null>(null)
let hideTimer: ReturnType<typeof setTimeout> | null = null

const handleMemberClick = (member: Member, event: MouseEvent) => {
  // 点击同一个成员：收起名片
  if (popoverState.value && popoverState.value.member.id === member.id) {
    popoverState.value = null
    return
  }
  const el = event.currentTarget as HTMLElement
  const rect = el.getBoundingClientRect()
  popoverState.value = { member: { ...member }, anchor: { left: rect.left, top: rect.top, bottom: rect.bottom } }
  fetchProfileForPopover(member)
}

// 列表数据不完整（无部门/职位/签名/IP），异步拉取详情富化名片；失败静默保留已有数据
const fetchProfileForPopover = async (member: Member) => {
  const userId = member.user?.id ?? member.id
  if (!userId) return
  const { profile, success } = await fetchUserProfile(userId, member)
  if (!success || !popoverState.value || popoverState.value.member.id !== member.id) return
  popoverState.value.member = {
    ...popoverState.value.member,
    name: popoverState.value.member.name || profile.name,
    username: popoverState.value.member.username || profile.username,
    department: popoverState.value.member.department || profile.department,
    position: popoverState.value.member.position || profile.position,
    signature: popoverState.value.member.signature || profile.signature,
    status: popoverState.value.member.status || profile.status,
    ip: popoverState.value.member.ip || profile.ip,
    email: popoverState.value.member.email || profile.email,
    mobile: popoverState.value.member.mobile || profile.mobile
  }
}

const scheduleHidePopover = () => {
  if (hideTimer) clearTimeout(hideTimer)
  hideTimer = setTimeout(() => {
    popoverState.value = null
    hideTimer = null
  }, HIDE_DELAY)
}

const cancelHidePopover = () => {
  if (hideTimer) {
    clearTimeout(hideTimer)
    hideTimer = null
  }
}

const displayName = computed(() => {
  const m = popoverState.value?.member
  if (!m) return ''
  return m.name || m.username || m.nickname || '未知用户'
})

const popoverStatusText = computed(() => {
  const m = popoverState.value?.member
  if (!m) return ''
  if (isBotType(m.type)) return 'AI 助手'
  switch (m.status) {
    case 'online': return '在线'
    case 'busy': return '忙碌'
    case 'offline': return '离线'
    default: return ''
  }
})

const popoverInfoRows = computed(() => {
  const m = popoverState.value?.member
  if (!m) return []
  const rows: { icon: string; text: string }[] = []
  if (m.username) rows.push({ icon: 'fas fa-at', text: m.username })
  const org = [m.department, m.position].filter(Boolean).join(' · ')
  if (org) rows.push({ icon: 'fas fa-briefcase', text: org })
  if (m.email) rows.push({ icon: 'fas fa-envelope', text: m.email })
  if (m.mobile) rows.push({ icon: 'fas fa-mobile-alt', text: m.mobile })
  if (m.signature) rows.push({ icon: 'fas fa-quote-left', text: m.signature })
  if (m.ip) rows.push({ icon: 'fas fa-globe', text: `IP: ${m.ip}` })
  return rows
})

const popoverStyle = computed(() => {
  if (!popoverState.value) return {}
  const { anchor } = popoverState.value
  // 侧边栏贴右，名片优先放成员项左侧；放不下时放右侧
  let left = anchor.left - POPOVER_WIDTH - POPOVER_GAP
  if (left < 8) left = anchor.left + POPOVER_GAP
  left = Math.min(Math.max(left, 8), window.innerWidth - POPOVER_WIDTH - 8)
  // 垂直对齐成员项顶部；底部溢出时上移
  const top = Math.max(8, Math.min(anchor.top, window.innerHeight - 360))
  return { left: `${left}px`, top: `${top}px` }
})

const handlePopoverPrivateChat = () => {
  const m = popoverState.value?.member
  if (!m) return
  popoverState.value = null
  emit('start-private-chat', m)
}

onUnmounted(() => {
  if (hideTimer) clearTimeout(hideTimer)
})
</script>

<style scoped>
.members-sidebar {
  width: 180px;
  background: var(--sidebar-bg);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: -2px 0 10px rgba(0, 0, 0, 0.05);
  transition: width 0.3s ease;
}

.members-sidebar.collapsed {
  width: 48px;
  border-left: none;
}

.sidebar-header-container {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 8px 0 8px;
  /* border-bottom: 1px solid var(--border-color); */
}

.members-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.header-content {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.collapsed-header {
  display: flex;
  justify-content: center;
  padding: 6px 0;
}

.collapsed-avatars {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.collapsed-avatar-item {
  position: relative;
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  transition: background 0.2s;
}

.collapsed-avatar-item:hover {
  background: var(--hover-color);
}

.collapsed-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  flex-shrink: 0;
}

.collapsed-role {
  position: absolute;
  bottom: 2px;
  right: 2px;
  font-size: 8px;
  line-height: 1;
}

.collapsed-role.owner {
  color: #ffd700;
}

.collapsed-role.admin {
  color: #4facfe;
}

.members-sidebar .members-header {
  padding: 8px 12px;
  background: var(--sidebar-bg);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.members-sidebar .members-header h3 {
  margin: 0;
  font-size: var(--font-size-xs);
  font-weight: 500;
  color: var(--text-color);
}

.action-btn {
  width: 24px;
  height: 24px;
  border: none;
  background: transparent;
  border-radius: 50%;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  transition: all 0.2s;
}

.action-btn:hover {
  background: var(--hover-color);
  color: var(--text-color);
}

.members-search {
  padding: 6px 8px;
  background: var(--sidebar-bg);
}

.member-search-input {
  width: 100%;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: var(--font-size-xxs);
  outline: none;
  background: var(--sidebar-bg);
  color: var(--text-color);
  border: 1px solid var(--border-color);
}

.member-search-input:focus {
  border-color: var(--primary-color);
}

.members-sidebar .members-content {
  flex: 1;
  overflow-y: auto;
  padding: 4px 8px;
}

.members-sidebar .member-item {
  display: flex;
  align-items: center;
  gap: 8px;
  border-radius: 6px;
  padding: 6px 10px;
  transition: all 0.2s ease;
  margin-bottom: 1px;
  cursor: pointer;
}

.members-sidebar .member-item:hover {
  background: var(--hover-color);
  transform: translateY(-1px);
}

.member-avatar-wrapper {
  position: relative;
  flex-shrink: 0;
}

.members-sidebar .member-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.member-status-dot {
  position: absolute;
  bottom: 0;
  right: 0;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  border: 1.5px solid var(--sidebar-bg, #fff);
  background-color: #d9d9d9;
}

.member-status-dot.online {
  background-color: #52c41a;
}

.member-status-dot.busy {
  background-color: #ff4d4f;
}

.member-status-dot.offline {
  background-color: #d9d9d9;
}

.member-detail {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
  gap: 1px;
}

.member-name-row {
  display: flex;
  align-items: center;
  gap: 4px;
}

.members-sidebar .member-name {
  font-size: var(--font-size-xs);
  color: var(--text-color);
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-weight: 400;
  min-width: 0;
}

.member-role-tag {
  font-size: var(--font-size-tiny);
  padding: 1px 5px;
  border-radius: 8px;
  font-weight: 600;
  white-space: nowrap;
  flex-shrink: 0;
  letter-spacing: 0.2px;
  line-height: 1.3;
}

.member-role-tag.owner {
  background: rgba(255, 215, 0, 0.15);
  color: #d4a017;
  border: 1px solid rgba(255, 215, 0, 0.25);
}

.member-role-tag.admin {
  background: rgba(79, 172, 254, 0.1);
  color: #2196f3;
  border: 1px solid rgba(79, 172, 254, 0.2);
}

.member-role-tag.bot {
  color: #7c3aed;
  background: rgba(124, 58, 237, 0.08);
  border: 1px solid rgba(124, 58, 237, 0.15);
  font-size: var(--font-size-tiny);
}

.member-position {
  font-size: var(--font-size-tiny);
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  opacity: 0.7;
  line-height: 1.3;
}

.members-sidebar .bot-member {
  cursor: default;
}

/* ---- 轻量成员名片（Teleport 到 body，fixed 定位） ---- */
.member-profile-popover {
  position: fixed;
  z-index: 1200;
  width: 248px;
  background: var(--card-bg, #fff);
  border-radius: 12px;
  box-shadow: 0 8px 28px rgba(0, 0, 0, 0.16), 0 0 0 1px rgba(0, 0, 0, 0.05);
  padding: 14px 14px 12px;
  box-sizing: border-box;
  max-height: calc(100vh - 16px);
  overflow-y: auto;
  animation: popover-in 0.14s ease-out;
}

@keyframes popover-in {
  from { opacity: 0; transform: translateY(-4px); }
  to { opacity: 1; transform: translateY(0); }
}

.popover-header {
  display: flex;
  align-items: center;
  gap: 10px;
}

.popover-avatar-wrap {
  position: relative;
  flex-shrink: 0;
}

.popover-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
}

.popover-status-dot {
  position: absolute;
  bottom: 0;
  right: 0;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  border: 2px solid var(--card-bg, #fff);
  background-color: #d9d9d9;
}

.popover-status-dot.online { background-color: #52c41a; }
.popover-status-dot.busy { background-color: #ff4d4f; }
.popover-status-dot.offline { background-color: #d9d9d9; }

.popover-identity {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
  flex: 1;
}

.popover-name-row {
  display: flex;
  align-items: center;
  gap: 5px;
  min-width: 0;
}

.popover-name {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.popover-role {
  font-size: var(--font-size-tiny);
  padding: 1px 6px;
  border-radius: 8px;
  font-weight: 600;
  white-space: nowrap;
  flex-shrink: 0;
  letter-spacing: 0.2px;
  line-height: 1.3;
}

.popover-role.owner {
  background: rgba(255, 215, 0, 0.15);
  color: #d4a017;
  border: 1px solid rgba(255, 215, 0, 0.25);
}

.popover-role.admin {
  background: rgba(79, 172, 254, 0.1);
  color: #2196f3;
  border: 1px solid rgba(79, 172, 254, 0.2);
}

.popover-role.bot {
  color: #7c3aed;
  background: rgba(124, 58, 237, 0.08);
  border: 1px solid rgba(124, 58, 237, 0.15);
}

.popover-status-text {
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary);
}

.popover-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--border-color, rgba(0, 0, 0, 0.06));
}

.popover-info-row {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  line-height: 1.5;
  min-width: 0;
}

.popover-info-row i {
  flex-shrink: 0;
  width: 14px;
  text-align: center;
  font-size: var(--font-size-xxxs);
  opacity: 0.6;
}

.popover-info-row span {
  min-width: 0;
  overflow-wrap: break-word;
}

.popover-actions {
  display: flex;
  margin-top: 10px;
}

.popover-action-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 7px 0;
  border: none;
  border-radius: 8px;
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: #fff;
  background: var(--primary-color);
  cursor: pointer;
  transition: all 0.2s;
}

.popover-action-btn:hover {
  background: var(--active-color);
  transform: translateY(-1px);
}
</style>

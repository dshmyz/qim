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
      <div v-for="member in filteredMembers" :key="member.id" class="member-item" :class="{ 'bot-member': member.type === 'bot_assistant' }" @contextmenu.prevent="handleMemberContextMenu($event, member)" @dblclick="member.type !== 'bot_assistant' && handleStartPrivateChat(member)">
        <Avatar
          :src="member.avatar"
          :name="member.name || '未知用户'"
          :alt="member.name || '未知用户'"
          size="sm"
          class="member-avatar"
        />
        <div class="member-info">
          <span class="member-name">{{ member.name || '未知用户' }}</span>
          <span v-if="member.role === 'owner'" class="member-role owner" title="群主"><i class="fas fa-crown"></i></span>
          <span v-else-if="member.role === 'admin'" class="member-role admin" title="管理员"><i class="fas fa-user-shield"></i></span>
          <span v-if="member.type === 'bot_assistant'" class="member-role bot" title="AI助手"><i class="fas fa-robot"></i></span>
        </div>
      </div>
    </div>
    <div v-else class="collapsed-avatars">
      <div
        v-for="member in filteredMembers"
        :key="member.id"
        class="collapsed-avatar-item"
        @contextmenu.prevent="handleMemberContextMenu($event, member)"
        @dblclick="member.type !== 'bot_assistant' && handleStartPrivateChat(member)"
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
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Avatar from '../shared/Avatar.vue'

interface Member {
  id: string
  name: string
  avatar: string
  role?: 'owner' | 'admin' | 'member'
  type?: string
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

const filteredMembers = computed(() => {
  let members = props.members || []
  
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

const handleStartPrivateChat = (member: Member) => {
  emit('start-private-chat', member)
}
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
  font-size: 13px;
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
  font-size: 12px;
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
  font-size: 12px;
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

.members-sidebar .member-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.members-sidebar .member-name {
  font-size: 13px;
  color: var(--text-color);
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-weight: 400;
}

.members-sidebar .member-info {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 4px;
  flex: 1;
  min-width: 0;
}

.members-sidebar .member-role {
  font-size: 14px;
  padding: 1px 4px;
  border-radius: 3px;
  font-weight: 500;
  white-space: nowrap;
}

.members-sidebar .member-role.owner {
  color: #ffd700;
}

.members-sidebar .member-role.admin {
  color: #4facfe;
}

.members-sidebar .member-role.bot {
  color: #7c3aed;
}

.members-sidebar .bot-member {
  cursor: default;
}

.members-sidebar .bot-member .member-name {
  color: var(--text-secondary);
}
</style>

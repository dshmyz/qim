<template>
  <div class="members-sidebar" :class="{ 'collapsed': !isExpanded }">
    <div class="sidebar-header-container">
      <div v-if="isExpanded" class="members-header">
        <div class="header-content">
          <button class="toggle-sidebar-btn" @click="handleToggleExpanded">
            <i class="fas fa-chevron-left"></i>
          </button>
          <h3>群成员 ({{ members.length }})</h3>
        </div>
        <div class="header-actions">
          <button class="search-toggle-btn" @click="handleToggleMemberSearch">
            <i class="fas fa-search"></i>
          </button>
        </div>
      </div>
      <button v-else class="collapsed-toggle-btn" @click="handleToggleExpanded">
        <i class="fas fa-user"></i>
      </button>
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
      <div v-for="member in filteredMembers" :key="member.id" class="member-item" @contextmenu.prevent="handleMemberContextMenu($event, member)" @dblclick="handleStartPrivateChat(member)">
        <img :src="member.avatar" :alt="member.name || '未知用户'" class="member-avatar" />
        <div class="member-info">
          <span class="member-name">{{ member.name || '未知用户' }}</span>
          <span v-if="member.role === 'owner'" class="member-role owner" title="群主"><i class="fas fa-crown"></i></span>
          <span v-else-if="member.role === 'admin'" class="member-role admin" title="管理员"><i class="fas fa-user-shield"></i></span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Member {
  id: string
  name: string
  avatar: string
  role?: 'owner' | 'admin' | 'member'
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

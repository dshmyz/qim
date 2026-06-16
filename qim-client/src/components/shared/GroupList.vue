<template>
  <div v-if="filteredGroups.length === 0" class="empty-groups">
    <div class="placeholder-content">
      <i class="fas fa-users fa-4x"></i>
      <h3>暂无群聊</h3>
      <p>你还没有加入任何群聊或讨论组</p>
    </div>
  </div>
  <div v-else class="groups-list">
    <div v-for="group in filteredGroups" :key="group.id" class="group-item" :class="{ active: selectedGroup && selectedGroup.id === group.id }" @contextmenu.prevent="$emit('showContextMenu', $event, group)" @click="$emit('select', group)" @dblclick="$emit('enter', group)">
      <div class="group-avatar">
        <Avatar :src="group.avatar" :name="group.name || (group.type === 'group' ? '群聊' : '讨论组')" :server-url="serverUrl" :alt="group.name" size="md" />
        <span class="group-badge" :class="group.type === 'discussion' ? 'discussion-badge' : ''">{{ group.type === 'group' ? '群' : '讨' }}</span>
      </div>
      <div class="group-info">
        <div class="group-name">
          {{ group.name }}
          <span v-if="group.member_count" class="member-count">({{ group.member_count }}人)</span>
          <span v-if="group.type === 'discussion'" class="conversation-type-tag">讨论组</span>
        </div>
      </div>
      <div v-if="group.unread_count && group.unread_count > 0" class="unread-badge">
        {{ group.unread_count > 99 ? '99+' : group.unread_count }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import Avatar from './Avatar.vue'
import { generateAvatar, getAvatarUrl, isAbsoluteUrl } from '../../utils/avatar'
import { useServerUrl } from '../../composables/useServerUrl'
import type { Conversation, User } from '../../types'

const { serverUrl } = useServerUrl()

interface Props {
  groups: any[]
  selectedGroup: Conversation | null
  searchQuery?: string
}

const props = defineProps<Props>()

defineEmits<{
  select: [conversation: any]
  enter: [conversation: any]
  showContextMenu: [event: MouseEvent, conversation: any]
}>()

const filteredGroups = computed(() => {
  if (!props.searchQuery || !props.searchQuery.trim()) return props.groups

  const query = props.searchQuery.toLowerCase().trim()
  return props.groups.filter(group => {
    if (group.name && group.name.toLowerCase().includes(query)) return true
    return false
  })
})

const getConversationAvatarUrl = (conversation: any) => {
  return getAvatarUrl(conversation.avatar, conversation.name || (conversation.type === 'group' ? '群聊' : '讨论组'), serverUrl.value)
}
</script>

<style scoped>
.groups-list {
  flex-shrink: 0;
  /* border-right: 1px solid #e8e8e8; */
  overflow-y: auto;
  padding: 16px;
  margin: 8px 8px;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  /* background: #fafafa; */
  max-height: calc(100vh - 200px);
}

.group-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s;
  margin-bottom: 8px;
}

.group-item:hover {
  background: var(--hover-color);
}

.group-item.active {
  background: var(--hover-color);
}

.group-avatar {
  position: relative;
  margin-right: 12px;
}

.group-avatar img {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  object-fit: cover;
}

.group-badge {
  position: absolute;
  bottom: 0;
  right: 0;
  background: #1976d2;
  color: white;
  font-size: 10px;
  padding: 2px 4px;
  border-radius: 4px;
}

.discussion-badge {
  background: #ff9800;
}

.conversation-type-tag {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 3px;
  background: #f5f5f5;
  color: #666;
  margin-left: 6px;
  font-weight: normal;
}

.group-info {
  flex: 1;
  min-width: 0;
}

.group-name {
  font-weight: 500;
  color: var(--text-color);
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.member-count {
  font-size: 12px;
  color: var(--text-secondary);
  margin-left: 6px;
  font-weight: normal;
}

.unread-badge {
  display: inline-block;
  background: var(--error-color);
  color: white;
  font-size: 12px;
  min-width: 18px;
  height: 18px;
  line-height: 18px;
  text-align: center;
  border-radius: 9px;
  padding: 0 6px;
}

.empty-groups {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 300px;
}

.empty-groups .placeholder-content {
  text-align: center;
  color: var(--text-secondary, #666);
}

.empty-groups .placeholder-content i {
  color: var(--text-tertiary, #999);
  margin-bottom: 16px;
}

.empty-groups .placeholder-content h3 {
  margin: 0 0 8px 0;
  color: var(--text-primary, #333);
}

.empty-groups .placeholder-content p {
  margin: 0;
  font-size: 14px;
  color: var(--text-secondary, #666);
}
</style>

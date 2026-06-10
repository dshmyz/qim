<template>
  <div v-if="visible" class="context-menu" :style="{ left: position.x + 'px', top: position.y + 'px' }">
    <div v-if="canRemoveMember" class="context-menu-item" @click.stop="handleRemoveMember">
      <span class="context-menu-icon"><i class="fas fa-trash"></i></span>
      <span>移除群聊</span>
    </div>
    <div class="context-menu-item" @click.stop="handleViewMemberInfo">
      <span class="context-menu-icon"><i class="fas fa-user"></i></span>
      <span>查看资料</span>
    </div>
    <div v-if="canSetAdmin" class="context-menu-item" @click.stop="handleSetAdmin">
      <span class="context-menu-icon"><i class="fas fa-star"></i></span>
      <span>{{ isSelectedMemberAdmin ? '取消管理员' : '设为管理员' }}</span>
    </div>
    <div v-if="canTransferOwner" class="context-menu-item" @click.stop="handleTransferOwner">
      <span class="context-menu-icon"><i class="fas fa-crown"></i></span>
      <span>转让群主</span>
    </div>
    <div v-if="member?.type !== 'bot_assistant'" class="context-menu-item" @click.stop="handleSendPrivateMessage">
      <span class="context-menu-icon"><i class="fas fa-comment"></i></span>
      <span>发起私聊</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Conversation, User } from '../../types'

interface Props {
  visible: boolean
  position: { x: number; y: number }
  member: User | null
  currentUserId: string | number
  conversation: Conversation | undefined
}

interface Emits {
  (e: 'remove-member', memberId: string | number, memberName: string): void
  (e: 'view-member-info', member: User | null): void
  (e: 'set-admin', memberId: string | number, memberName: string, isAdmin: boolean): void
  (e: 'transfer-owner', memberId: string | number, memberName: string): void
  (e: 'send-private-message'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const currentUserRole = computed((): string => {
  if (!props.conversation?.members || !props.currentUserId) return 'member'
  const member = props.conversation.members.find((m: any) => {
    return String(m.id) === String(props.currentUserId)
  })
  return member?.role || 'member'
})

const isSelectedMemberAdmin = computed((): boolean => {
  return props.member?.role === 'admin'
})

const canRemoveMember = computed((): boolean => {
  if (!props.member || currentUserRole.value === 'member') return false
  if (props.member.role === 'owner') return false
  if (currentUserRole.value === 'admin' && props.member.role === 'admin') return false
  return true
})

const canSetAdmin = computed((): boolean => {
  if (!props.member || (currentUserRole.value !== 'owner' && currentUserRole.value !== 'admin')) return false
  if (props.member.role === 'owner') return false
  if (currentUserRole.value === 'admin' && props.member.role === 'admin') return false
  return true
})

const canTransferOwner = computed((): boolean => {
  if (!props.member || currentUserRole.value !== 'owner') return false
  if (props.member.role === 'owner') return false
  return true
})

const handleRemoveMember = () => {
  if (!props.member) return
  emit('remove-member', props.member.id, props.member.name || '未知用户')
}

const handleViewMemberInfo = () => {
  emit('view-member-info', props.member)
}

const handleSetAdmin = () => {
  if (!props.member) return
  emit('set-admin', props.member.id, props.member.name || '未知用户', isSelectedMemberAdmin.value)
}

const handleTransferOwner = () => {
  if (!props.member) return
  emit('transfer-owner', props.member.id, props.member.name || '未知用户')
}

const handleSendPrivateMessage = () => {
  emit('send-private-message')
}
</script>

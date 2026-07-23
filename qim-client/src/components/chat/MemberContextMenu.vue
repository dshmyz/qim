<template>
  <UniversalContextMenu menuId="member" :items="menuItems" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Conversation, User } from '../../types'
import UniversalContextMenu from '../shared/UniversalContextMenu.vue'
import type { ContextMenuItem } from '../shared/context-menu-types'

interface Props {
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

const isBotMember = computed((): boolean => {
  const t = props.member?.type
  return t === 'bot'
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

const menuItems = computed<ContextMenuItem[]>(() => [
  {
    label: '查看资料',
    icon: 'fas fa-user',
    action: () => emit('view-member-info', props.member)
  },
  {
    label: isSelectedMemberAdmin.value ? '取消管理员' : '设为管理员',
    icon: 'fas fa-star',
    visible: canSetAdmin.value && !isBotMember.value,
    action: () => {
      if (props.member) emit('set-admin', props.member.id, props.member.name || '未知用户', isSelectedMemberAdmin.value)
    }
  },
  {
    label: '转让群主',
    icon: 'fas fa-crown',
    visible: canTransferOwner.value && !isBotMember.value,
    action: () => {
      if (props.member) emit('transfer-owner', props.member.id, props.member.name || '未知用户')
    }
  },
  {
    label: '发起私聊',
    icon: 'fas fa-comment',
    visible: props.member?.type !== 'bot',
    action: () => emit('send-private-message')
  },
  { divider: true, visible: canRemoveMember.value },
  {
    label: '移除群聊',
    icon: 'fas fa-trash',
    danger: true,
    visible: canRemoveMember.value,
    action: () => {
      if (props.member) emit('remove-member', props.member.id, props.member.name || '未知用户')
    }
  }
])
</script>

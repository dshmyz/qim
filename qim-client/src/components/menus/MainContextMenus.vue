<template>
  <!-- 会话右键菜单 -->
  <UniversalContextMenu menuId="context" :items="conversationMenuItems" />
  <UniversalContextMenu menuId="action" :items="actionMenuItems" />
  <UniversalContextMenu menuId="user" :items="userMenuItems" />
  <UniversalContextMenu menuId="sidebar-member" :items="memberMenuItems" />
  <UniversalContextMenu menuId="group" :items="groupMenuItems" />
  <UniversalContextMenu menuId="settings" :items="settingsMenuItems" />
  <UniversalContextMenu menuId="theme" :items="themeMenuItems" />
  <UniversalContextMenu menuId="more" :items="moreMenuItems" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import UniversalContextMenu from '../shared/UniversalContextMenu.vue'
import type { ContextMenuItem } from '../shared/context-menu-types'

interface Conversation {
  id: string | number
  name: string
  type: string
  is_pinned?: boolean
  pinned?: boolean
  muted?: boolean
}

interface Theme {
  id: string
  name: string
  themeClass: string
}

const themes: Theme[] = [
  { id: 'modern-light', name: '清新白', themeClass: 'light-theme' },
  { id: 'elegant-dark', name: '炫酷黑', themeClass: 'elegant-dark-theme' },
  { id: 'monochrome-elegance', name: '单色雅', themeClass: 'monochrome-elegance-theme' },
  { id: 'crimson-red', name: '中国红', themeClass: 'crimson-red-theme' },
  { id: 'emerald-green', name: '翡翠绿', themeClass: 'emerald-green-theme' },
  { id: 'elegant-purple', name: '高雅紫', themeClass: 'elegant-purple-theme' },
  { id: 'warm-amber', name: '琥珀黄', themeClass: 'warm-amber-theme' },
  { id: 'ocean-blue', name: '海洋蓝', themeClass: 'ocean-blue-theme' },
  { id: 'mediterranean-dream', name: '地中海', themeClass: 'mediterranean-dream-theme' },
  { id: 'spring-blossom', name: '春日花', themeClass: 'spring-blossom-theme' }
]

interface Props {
  selectedConversation: Conversation | null
  selectedEmployee: any
  selectedGroupForContextMenu: Conversation | null
  isGroupOwner: boolean
  currentUser?: { isAdmin?: boolean; roles?: string[] }
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'pin': [conversation: Conversation]
  'mute': [conversation: Conversation]
  'exitGroup': [conversation?: Conversation]
  'remove': [conversation: Conversation]
  'createGroup': []
  'createDiscussion': []
  'createChannel': []
  'systemMessage': []
  'viewUserProfile': []
  'privateChat': [employee: any]
  'removeMember': []
  'viewMemberInfo': []
  'setAdmin': []
  'viewGroupMembers': [group: Conversation]
  'addMembers': [group: Conversation]
  'editAnnouncement': [group: Conversation]
  'dissolveGroup': [group: Conversation]
  'about': []
  'checkUpdate': []
  'settings': []
  'openFeedback': []
  'logout': []
  'setTheme': [theme: string]
  'showChannels': []
  'closeMoreMenu': []
  'closeAllMenus': []
}>()

const canCreateChannel = computed(() => {
  return props.currentUser?.isAdmin || props.currentUser?.roles?.includes('system_admin')
})

const canPublishSystemMessage = computed(() => {
  return props.currentUser?.isAdmin ||
    props.currentUser?.roles?.includes('system_admin') ||
    props.currentUser?.roles?.includes('system_publisher')
})

// 会话右键菜单
const conversationMenuItems = computed<ContextMenuItem[]>(() => {
  const conv = props.selectedConversation
  if (!conv) return []
  return [
    {
      label: conv.is_pinned || conv.pinned ? '取消置顶' : '置顶',
      icon: 'fas fa-thumbtack',
      action: () => { emit('pin', conv); emit('closeAllMenus') }
    },
    {
      label: conv.muted ? '取消免打扰' : '免打扰',
      icon: 'fas fa-bell-slash',
      action: () => { emit('mute', conv); emit('closeAllMenus') }
    },
    {
      label: '退出群聊',
      icon: 'fas fa-sign-out-alt',
      visible: conv.type === 'group' || conv.type === 'discussion',
      action: () => { emit('exitGroup', conv); emit('closeAllMenus') }
    },
    { divider: true },
    {
      label: '移除会话',
      icon: 'fas fa-trash',
      danger: true,
      action: () => { emit('remove', conv); emit('closeAllMenus') }
    }
  ]
})

// 动作菜单
const actionMenuItems = computed<ContextMenuItem[]>(() => [
  {
    label: '创建群聊',
    icon: 'fas fa-user-friends',
    action: () => emit('createGroup')
  },
  {
    label: '创建讨论组',
    icon: 'fas fa-comments',
    action: () => emit('createDiscussion')
  },
  {
    label: '创建频道',
    icon: 'fas fa-bullhorn',
    visible: canCreateChannel.value,
    action: () => emit('createChannel')
  },
  {
    label: '发布系统消息',
    icon: 'fas fa-broadcast-tower',
    visible: canPublishSystemMessage.value,
    action: () => emit('systemMessage')
  }
])

// 用户右键菜单
const userMenuItems = computed<ContextMenuItem[]>(() => [
  {
    label: '查看资料',
    icon: 'fas fa-user',
    action: () => emit('viewUserProfile')
  },
  {
    label: '发起私聊',
    icon: 'fas fa-comment',
    action: () => emit('privateChat', props.selectedEmployee)
  }
])

// 成员上下文菜单
const memberMenuItems = computed<ContextMenuItem[]>(() => [
  {
    label: '移除群聊',
    icon: 'fas fa-trash',
    danger: true,
    action: () => emit('removeMember')
  },
  {
    label: '查看资料',
    icon: 'fas fa-user',
    action: () => emit('viewMemberInfo')
  },
  {
    label: '设为管理员',
    icon: 'fas fa-star',
    action: () => emit('setAdmin')
  }
])

// 群聊上下文菜单
const groupMenuItems = computed<ContextMenuItem[]>(() => {
  const group = props.selectedGroupForContextMenu
  if (!group) return []
  const items: ContextMenuItem[] = [
    {
      label: '查看群成员',
      icon: 'fas fa-user-friends',
      action: () => emit('viewGroupMembers', group)
    },
    {
      label: '添加成员',
      icon: 'fas fa-plus',
      action: () => emit('addMembers', group)
    },
    {
      label: '编辑群公告',
      icon: 'fas fa-bullhorn',
      visible: props.isGroupOwner,
      action: () => emit('editAnnouncement', group)
    },
    {
      label: '解散群聊',
      icon: 'fas fa-trash-alt',
      danger: true,
      visible: props.isGroupOwner,
      action: () => emit('dissolveGroup', group)
    }
  ]
  if (!props.isGroupOwner) {
    items.push(
      { divider: true },
      {
        label: '退出群聊',
        icon: 'fas fa-sign-out-alt',
        action: () => emit('exitGroup', group)
      }
    )
  }
  return items
})

// 设置菜单
const settingsMenuItems = computed<ContextMenuItem[]>(() => [
  {
    label: '关于',
    icon: 'fas fa-info-circle',
    action: () => emit('about')
  },
  {
    label: '问题反馈',
    icon: 'fas fa-comment-dots',
    action: () => { emit('openFeedback'); emit('closeAllMenus') }
  },
  {
    label: '检查更新',
    icon: 'fas fa-sync',
    action: () => emit('checkUpdate')
  },
  {
    label: '设置',
    icon: 'fas fa-sliders',
    action: () => emit('settings')
  },
  { divider: true },
  {
    label: '退出登录',
    icon: 'fas fa-sign-out-alt',
    danger: true,
    action: () => emit('logout')
  }
])

// 主题菜单
const themeMenuItems = computed<ContextMenuItem[]>(() =>
  themes.map(t => ({
    label: t.name,
    icon: `theme-icon ${t.themeClass}`,
    action: () => emit('setTheme', t.id)
  }))
)

// 更多菜单
const moreMenuItems = computed<ContextMenuItem[]>(() => [
  {
    label: '频道',
    icon: 'fas fa-bullhorn',
    action: () => { emit('showChannels'); emit('closeMoreMenu') }
  }
])
</script>

<style>
.theme-icon {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  display: inline-block;
  vertical-align: middle;
}

.light-theme { background: #fff; border: 1px solid #ddd; }
.elegant-dark-theme { background: #333; }
.ocean-blue-theme { background: #0078d4; }
.elegant-purple-theme { background: #75629a; }
.warm-amber-theme { background: #d4893a; }
.crimson-red-theme { background: #c4352e; }
.emerald-green-theme { background: #2d8b4e; }
.mediterranean-dream-theme { background: #4a8aad; }
.monochrome-elegance-theme { background: #777; }
.spring-blossom-theme { background: #f0a1b9; }
</style>

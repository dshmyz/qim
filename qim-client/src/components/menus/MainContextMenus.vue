<template>
  <!-- 右键菜单 -->
  <div v-if="showMenu && selectedConversation" class="context-menu" :style="{ left: menuPosition.x + 'px', top: menuPosition.y + 'px' }" @click.stop>
    <div class="context-menu-item" @click="handleContextMenuAction('pin', selectedConversation)">
      {{ selectedConversation.is_pinned ? '取消置顶' : '置顶' }}
    </div>
    <div class="context-menu-item" @click="handleContextMenuAction('mute', selectedConversation)">
      {{ selectedConversation.muted ? '取消免打扰' : '免打扰' }}
    </div>
    <div v-if="selectedConversation.type === 'group' || selectedConversation.type === 'discussion'" class="context-menu-item" @click="handleContextMenuAction('exitGroup', selectedConversation)">
      退出群聊
    </div>
    <div class="context-menu-item divider"></div>
    <div class="context-menu-item" @click="handleContextMenuAction('remove', selectedConversation)">
      移除会话
    </div>
  </div>

  <!-- 动作菜单 -->
  <div v-if="showActionMenuFlag" class="action-menu" :style="{ left: actionMenuPosition.x + 'px', top: actionMenuPosition.y + 'px' }">
    <div class="action-menu-item" @click="$emit('createGroup')">
      <span class="action-menu-icon"><i class="fas fa-user-friends"></i></span>
      <span>创建群聊</span>
    </div>
    <div class="action-menu-item" @click="$emit('createDiscussion')">
      <span class="action-menu-icon"><i class="fas fa-comments"></i></span>
      <span>创建讨论组</span>
    </div>
    <div v-if="canCreateChannel" class="action-menu-item" @click="$emit('createChannel')">
      <span class="action-menu-icon"><i class="fas fa-bullhorn"></i></span>
      <span>创建频道</span>
    </div>
    <div v-if="canPublishSystemMessage" class="action-menu-item" @click="$emit('systemMessage')">
      <span class="action-menu-icon"><i class="fas fa-broadcast-tower"></i></span>
      <span>发布系统消息</span>
    </div>
  </div>

  <!-- 用户右键菜单 -->
  <div v-if="showUserContextMenuFlag" class="user-context-menu" :style="{ left: userContextMenuPosition.x + 'px', top: userContextMenuPosition.y + 'px' }" @click.stop>
    <div class="user-context-menu-item" @click="$emit('viewUserProfile')">
      <span class="user-context-menu-icon"><i class="fas fa-user"></i></span>
      <span>查看资料</span>
    </div>
    <div class="user-context-menu-item" @click="$emit('privateChat', selectedEmployee)">
      <span class="user-context-menu-icon"><i class="fas fa-comment"></i></span>
      <span>发起私聊</span>
    </div>
  </div>

  <!-- 成员上下文菜单 -->
  <div v-if="showMemberContextMenuFlag" class="context-menu" :style="{ left: memberContextMenuPosition.x + 'px', top: memberContextMenuPosition.y + 'px' }">
    <div class="context-menu-item" @click="$emit('removeMember')">
      <span class="context-menu-icon"><i class="fas fa-trash"></i></span>
      <span>移除群聊</span>
    </div>
    <div class="context-menu-item" @click="$emit('viewMemberInfo')">
      <span class="context-menu-icon"><i class="fas fa-user"></i></span>
      <span>查看资料</span>
    </div>
    <div class="context-menu-item" @click="$emit('setAdmin')">
      <span class="context-menu-icon"><i class="fas fa-star"></i></span>
      <span>设为管理员</span>
    </div>
  </div>

  <!-- 群聊上下文菜单 -->
  <div v-if="showGroupContextMenuFlag" class="context-menu" :style="{ left: groupContextMenuPosition.x + 'px', top: groupContextMenuPosition.y + 'px' }">
    <div class="context-menu-item" @click="$emit('viewGroupMembers', selectedGroupForContextMenu)">
      <span class="context-menu-icon"><i class="fas fa-user-friends"></i></span>
      <span>查看群成员</span>
    </div>
    <div class="context-menu-item" @click="$emit('addMembers', selectedGroupForContextMenu)">
      <span class="context-menu-icon"><i class="fas fa-plus"></i></span>
      <span>添加成员</span>
    </div>
    <div v-if="isGroupOwner" class="context-menu-item" @click="$emit('editAnnouncement', selectedGroupForContextMenu)">
      <span class="context-menu-icon"><i class="fas fa-bullhorn"></i></span>
      <span>编辑群公告</span>
    </div>
    <div v-if="isGroupOwner" class="context-menu-item" @click="$emit('dissolveGroup', selectedGroupForContextMenu)">
      <span class="context-menu-icon"><i class="fas fa-trash-alt"></i></span>
      <span>解散群聊</span>
    </div>
    <div class="context-menu-divider"></div>
    <div class="context-menu-item" @click="$emit('exitGroup', selectedGroupForContextMenu)">
      <span class="context-menu-icon"><i class="fas fa-sign-out-alt"></i></span>
      <span>退出群聊</span>
    </div>
  </div>

  <!-- 设置菜单 -->
  <div v-if="showSettingsMenuFlag" class="context-menu" :style="{ left: settingsMenuPosition.x + 'px', top: settingsMenuPosition.y + 'px' }">
    <div class="context-menu-item" @click="$emit('about')">
      <span class="context-menu-icon"><i class="fas fa-info-circle"></i></span>
      <span>关于</span>
    </div>
    <div class="context-menu-item" @click="handleContextMenuAction('openFeedback')">
      <span class="context-menu-icon"><i class="fas fa-comment-dots"></i></span>
      <span>问题反馈</span>
    </div>
    <div class="context-menu-item" @click="$emit('checkUpdate')">
      <span class="context-menu-icon"><i class="fas fa-sync"></i></span>
      <span>检查更新</span>
    </div>
    <div class="context-menu-item" @click="$emit('settings')">
      <span class="context-menu-icon"><i class="fas fa-sliders"></i></span>
      <span>设置</span>
    </div>
    <div class="context-menu-divider"></div>
    <div class="context-menu-item" @click="$emit('logout')">
      <span class="context-menu-icon"><i class="fas fa-sign-out-alt"></i></span>
      <span>退出登录</span>
    </div>
  </div>

  <!-- 主题菜单 -->
  <div v-if="showThemeMenuFlag" class="context-menu theme-menu" :style="{ left: themeMenuPosition.x + 'px', top: themeMenuPosition.y + 'px' }">
    <div v-for="theme in themes" :key="theme.id" class="context-menu-item" @click="$emit('setTheme', theme.id)">
      <span :class="['context-menu-icon', 'theme-icon', theme.themeClass]"></span>
      <span>{{ theme.name }}</span>
    </div>
  </div>

  <!-- 更多菜单 -->
  <div v-if="showMoreMenuFlag" class="context-menu" :style="{ left: moreMenuPosition.x + 'px', top: moreMenuPosition.y + 'px' }">
    <div class="context-menu-item" @click="$emit('showChannels'); $emit('closeMoreMenu')">
      <span class="context-menu-icon"><i class="fas fa-bullhorn"></i></span>
      <span>频道</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch, onMounted, onUnmounted } from 'vue'

interface Conversation {
  id: string | number
  name: string
  type: string
  pinned?: boolean
  muted?: boolean
}

interface Position {
  x: number
  y: number
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
  showMenu: boolean
  selectedConversation: Conversation | null
  menuPosition: Position
  showActionMenuFlag: boolean
  actionMenuPosition: Position
  showUserContextMenuFlag: boolean
  userContextMenuPosition: Position
  selectedEmployee: any
  showMemberContextMenuFlag: boolean
  memberContextMenuPosition: Position
  showGroupContextMenuFlag: boolean
  groupContextMenuPosition: Position
  selectedGroupForContextMenu: Conversation | null
  isGroupOwner: boolean
  showSettingsMenuFlag: boolean
  settingsMenuPosition: Position
  showThemeMenuFlag: boolean
  themeMenuPosition: Position
  showMoreMenuFlag: boolean
  moreMenuPosition: Position
  currentUser?: { isAdmin?: boolean; roles?: string[] }
}

const props = defineProps<Props>()

const canCreateChannel = computed(() => {
  return props.currentUser?.isAdmin || props.currentUser?.roles?.includes('system_admin')
})

const canPublishSystemMessage = computed(() => {
  return props.currentUser?.isAdmin ||
    props.currentUser?.roles?.includes('system_admin') ||
    props.currentUser?.roles?.includes('system_publisher')
})

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

const handleContextMenuAction = (event: string, payload?: any) => {
  emit(event as any, payload)
  emit('closeAllMenus')
}

const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  if (target.closest('.context-menu') || target.closest('.action-menu') || target.closest('.theme-menu')) {
    return
  }
  if (
    props.showMenu ||
    props.showActionMenuFlag ||
    props.showUserContextMenuFlag ||
    props.showMemberContextMenuFlag ||
    props.showGroupContextMenuFlag ||
    props.showSettingsMenuFlag ||
    props.showThemeMenuFlag ||
    props.showMoreMenuFlag
  ) {
    emit('closeAllMenus')
  }
}

watch(() => [
  props.showMenu,
  props.showActionMenuFlag,
  props.showUserContextMenuFlag,
  props.showMemberContextMenuFlag,
  props.showGroupContextMenuFlag,
  props.showSettingsMenuFlag,
  props.showThemeMenuFlag,
  props.showMoreMenuFlag
], (vals) => {
  const anyVisible = vals.some(v => v)
  if (anyVisible) {
    setTimeout(() => {
      document.addEventListener('click', handleClickOutside)
    }, 0)
  } else {
    document.removeEventListener('click', handleClickOutside)
  }
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.context-menu {
  position: fixed;
  background: var(--context-menu-bg);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  padding: 8px 0;
  z-index: 1000;
  min-width: 160px;
}

.context-menu-item {
  padding: 8px 16px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-color);
  font-size: 13px;
}

.context-menu-item:hover {
  background: var(--context-menu-hover);
}

.context-menu-item.divider,
.context-menu-divider {
  height: 1px;
  background: var(--border-color);
  margin: 4px 0;
  padding: 0;
}

.context-menu-icon {
  width: 16px;
  text-align: center;
}

.theme-icon {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  display: inline-block;
}

.light-theme { background: #fff; border: 1px solid #ddd; }
.elegant-dark-theme { background: #333; }
.ocean-blue-theme { background: #0078d4; }
.elegant-purple-theme { background: #6b4c9a; }
.warm-amber-theme { background: #d4893a; }
.crimson-red-theme { background: #c4352e; }
.emerald-green-theme { background: #2d8b4e; }
.mediterranean-dream-theme { background: #4a8aad; }
.monochrome-elegance-theme { background: #777; }
.spring-blossom-theme { background: #f0a1b9; }

.theme-menu {
  min-width: 140px;
}

.action-menu {
  position: fixed;
  background: var(--context-menu-bg);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  padding: 8px 0;
  z-index: 1000;
  min-width: 180px;
}

.action-menu-item {
  padding: 10px 16px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--text-color);
}

.action-menu-item:hover {
  background: var(--context-menu-hover);
}

.action-menu-icon {
  width: 20px;
  text-align: center;
  color: var(--primary-color);
}

.user-context-menu {
  position: fixed;
  background: var(--context-menu-bg);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  padding: 8px 0;
  z-index: 1000;
  min-width: 140px;
}

.user-context-menu-item {
  padding: 8px 16px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-color);
  font-size: 13px;
}

.user-context-menu-item:hover {
  background: var(--context-menu-hover);
}

.user-context-menu-icon {
  width: 16px;
  text-align: center;
}
</style>

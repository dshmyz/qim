<template>
  <div class="sidebar">
    <div class="sidebar-header">
      <div class="user-info">
        <img
          src="https://api.dicebear.com/7.x/avataaars/svg?seed=me"
          alt="avatar"
          class="user-avatar"
        />
        <span class="user-name">我的账号</span>
      </div>
      <div class="header-actions">
        <button class="icon-btn">+</button>
      </div>
    </div>

    <div class="search-box">
      <input
        v-model="searchQuery"
        type="text"
        placeholder="搜索会话..."
        class="search-input"
      />
    </div>

    <div class="conversation-list">
      <div
          v-for="conversation in filteredConversations"
          :key="conversation.id"
          class="conversation-item"
          :class="{ active: conversation.id === currentId }"
          @click="$emit('select', conversation)"
          @contextmenu.prevent="showContextMenu($event, conversation)"
        >
        <div class="conversation-avatar">
          <img :src="conversation.avatar" :alt="conversation.name" />
          <span v-if="conversation.type === 'group'" class="group-badge">群</span>
        </div>
        <div class="conversation-info">
          <div class="conversation-name">
            {{ conversation.name }}
            <span v-if="conversation.type === 'group' && conversation.members" class="member-count">
              ({{ conversation.members.length }}人)
            </span>
            <span v-if="conversation.muted" class="muted-icon" title="免打扰">🔕</span>
          </div>
          <div class="conversation-preview">
            {{ conversation.lastMessage?.content || '暂无消息' }}
          </div>
        </div>
        <div class="conversation-meta">
          <div class="conversation-time">{{ formatTime(conversation.timestamp) }}</div>
          <div v-if="conversation.unreadCount > 99" class="unread-badge">
            {{ conversation.unreadCount > 99 ? '99+' : conversation.unreadCount }}
          </div>
        </div>
      </div>
    </div>
    <div class="sidebar-footer">
      <button class="settings-btn" @click="showSettingsMenu($event)">
        <i class="fas fa-cog"></i>
      </button>
    </div>

    <!-- 右键菜单 -->
    <div
      v-if="showMenu"
      class="context-menu"
      :style="{
        left: menuPosition.x + 'px',
        top: menuPosition.y + 'px'
      }"
    >
      <div class="menu-item" @click="handleMute">
        <span class="menu-icon">🔇</span>
        <span>免打扰</span>
      </div>
      <div class="menu-divider"></div>
      <div v-if="selectedConversation?.type === 'group'" class="menu-item" @click="handleExitGroup">
        <span class="menu-icon">🚪</span>
        <span>退出群聊</span>
      </div>
      <div class="menu-item" @click="handleRemove">
        <span class="menu-icon">🗑️</span>
        <span>移除会话</span>
      </div>
    </div>
    
    <!-- 设置菜单 -->
    <div
      v-if="showSettingsMenuFlag"
      class="context-menu"
      :style="{
        left: settingsMenuPosition.x + 'px',
        top: settingsMenuPosition.y + 'px'
      }"
    >
      <div class="menu-item" @click="openSettings">
        <span class="menu-icon"><i class="fas fa-sliders"></i></span>
        <span>设置</span>
      </div>
      <div class="menu-item" @click="checkForUpdates">
        <span class="menu-icon"><i class="fas fa-sync"></i></span>
        <span>检查更新</span>
      </div>
      <div class="menu-item" @click="aboutApp">
        <span class="menu-icon"><i class="fas fa-info-circle"></i></span>
        <span>关于</span>
      </div>
      <div class="menu-divider"></div>
      <div class="menu-item" @click="logout">
        <span class="menu-icon"><i class="fas fa-sign-out-alt"></i></span>
        <span>退出登录</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { Conversation } from '../types'
import { ElMessage } from 'element-plus'

interface Props {
  conversations: Conversation[]
  currentId: string | null
}

const props = defineProps<Props>()
defineEmits<{
  select: [conversation: Conversation]
}>()

const searchQuery = ref('')
const showMenu = ref(false)
const menuPosition = ref({ x: 0, y: 0 })
const selectedConversation = ref<Conversation | null>(null)

// 设置菜单
const showSettingsMenuFlag = ref(false)
const settingsMenuPosition = ref({ x: 0, y: 0 })

const filteredConversations = computed(() => {
  if (!searchQuery.value) return props.conversations
  const query = searchQuery.value.toLowerCase()
  return props.conversations.filter(
    c =>
      c.name.toLowerCase().includes(query) ||
      c.lastMessage?.content.toLowerCase().includes(query)
  )
})

const showContextMenu = (event: MouseEvent, conversation: Conversation) => {
  event.preventDefault()
  showMenu.value = true
  menuPosition.value = {
    x: event.clientX,
    y: event.clientY
  }
  selectedConversation.value = conversation
  
  // 点击其他地方关闭菜单
  setTimeout(() => {
    document.addEventListener('click', closeContextMenu)
  }, 0)
}

const closeContextMenu = () => {
  showMenu.value = false
  selectedConversation.value = null
  document.removeEventListener('click', closeContextMenu)
}

const handleMute = async () => {
  if (selectedConversation.value) {
    // 切换免打扰状态
    selectedConversation.value.muted = !selectedConversation.value.muted
    console.log(selectedConversation.value.muted ? '设置为免打扰:' : '取消免打扰:', selectedConversation.value.name)
    
    // 保存免打扰状态到服务器
    try {
      const response = await fetch('/api/v1/conversations/mute', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          conversation_id: selectedConversation.value.id,
          muted: selectedConversation.value.muted
        })
      })
      const data = await response.json()
      if (data.code !== 0) {
        console.error('保存免打扰状态失败:', data.message)
        // 保存失败时恢复原来的状态
        selectedConversation.value.muted = !selectedConversation.value.muted
      }
    } catch (error) {
      console.error('保存免打扰状态失败:', error)
      // 保存失败时恢复原来的状态
      selectedConversation.value.muted = !selectedConversation.value.muted
    }
  }
  closeContextMenu()
}

const handleRemove = () => {
  if (selectedConversation.value) {
    // 这里可以实现移除会话逻辑
    console.log('移除:', selectedConversation.value.name)
  }
  closeContextMenu()
}

const handleExitGroup = () => {
  if (selectedConversation.value) {
    // 这里可以实现退出群聊逻辑
    console.log('退出群聊:', selectedConversation.value.name)
  }
  closeContextMenu()
}

const showSettingsMenu = (event) => {
  event.stopPropagation()
  settingsMenuPosition.value = {
    x: event.clientX,
    y: event.clientY
  }
  showSettingsMenuFlag.value = true
  
  // 点击其他地方关闭菜单
  setTimeout(() => {
    document.addEventListener('click', closeSettingsMenu)
  }, 0)
}

const closeSettingsMenu = () => {
  showSettingsMenuFlag.value = false
  document.removeEventListener('click', closeSettingsMenu)
}

const openSettings = () => {
  console.log('打开设置')
  // 这里可以实现打开设置页面的逻辑
  ElMessage.info('打开设置')
  closeSettingsMenu()
}

const checkForUpdates = () => {
  console.log('检查更新')
  // 这里可以实现检查更新的逻辑
  ElMessage.info('检查更新中...')
  setTimeout(() => {
    ElMessage.success('当前已是最新版本')
  }, 1000)
  closeSettingsMenu()
}

const aboutApp = () => {
  console.log('关于应用')
  // 这里可以实现打开关于页面的逻辑
  ElMessage.info('关于应用\n版本: 1.0.0\n© 2026 QIM')
  closeSettingsMenu()
}

const logout = () => {
  console.log('退出登录')
  // 这里可以实现退出登录的逻辑
  if (confirm('确定要退出登录吗？')) {
    ElMessage.success('已退出登录')
  }
  closeSettingsMenu()
}

function formatTime(timestamp: number): string {
  const now = Date.now()
  const diff = now - timestamp
  const minute = 60 * 1000
  const hour = 60 * minute
  const day = 24 * hour

  if (diff < minute) {
    return '刚刚'
  } else if (diff < hour) {
    return `${Math.floor(diff / minute)}分钟前`
  } else if (diff < day) {
    return `${Math.floor(diff / hour)}小时前`
  } else {
    const date = new Date(timestamp)
    return `${date.getMonth() + 1}/${date.getDate()}`
  }
}
</script>

<style scoped>
.sidebar {
  width: 300px;
  height: 100%;
  background: #f8f9fa;
  border-right: 1px solid #e8e8e8;
  display: flex;
  flex-direction: column;
}

.sidebar-header {
  padding: 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid #e8e8e8;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
}

.user-name {
  font-weight: 500;
  color: #333;
}

.icon-btn {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: none;
  background: #e3f2fd;
  color: #1976d2;
  font-size: 20px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s;
}

.icon-btn:hover {
  background: #bbdefb;
}

.search-box {
  padding: 12px 16px;
  background: #fff;
  border-bottom: 1px solid #e8e8e8;
}

.search-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #e0e0e0;
  border-radius: 20px;
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
}

.search-input:focus {
  border-color: #1976d2;
}

.conversation-list {
  flex: 1;
  overflow-y: auto;
}

.conversation-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  cursor: pointer;
  transition: background 0.2s;
}

.conversation-item:hover {
  background: #e3f2fd;
}

.conversation-item.active {
  background: #bbdefb;
}

.conversation-avatar {
  position: relative;
  margin-right: 12px;
}

.conversation-avatar img {
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

.conversation-info {
  flex: 1;
  min-width: 0;
}

.conversation-name {
  font-weight: 500;
  color: #333;
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.conversation-preview {
  font-size: 13px;
  color: #666;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.conversation-meta {
  text-align: right;
  margin-left: 8px;
}

.conversation-time {
  font-size: 12px;
  color: #999;
  margin-bottom: 4px;
}

.unread-badge {
  display: inline-block;
  background: #f44336;
  color: white;
  font-size: 12px;
  min-width: 18px;
  height: 18px;
  line-height: 18px;
  text-align: center;
  border-radius: 9px;
  padding: 0 6px;
}

.member-count {
  font-size: 12px;
  color: #666;
  margin-left: 6px;
  font-weight: normal;
}

.muted-icon {
  font-size: 12px;
  margin-left: 6px;
  color: #999;
  vertical-align: middle;
}

/* 右键菜单样式 */
.context-menu {
  position: fixed;
  background: #fff;
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.15);
  z-index: 1000;
  min-width: 160px;
  overflow: hidden;
}

.menu-item {
  display: flex;
  align-items: center;
  padding: 8px 16px;
  cursor: pointer;
  transition: background 0.2s;
  font-size: 14px;
  color: #333;
}

.menu-item:hover {
  background: #f5f5f5;
}

.menu-icon {
  margin-right: 10px;
  font-size: 16px;
  width: 20px;
  text-align: center;
}

.menu-divider {
  height: 1px;
  background: #e8e8e8;
  margin: 4px 0;
}

.sidebar-footer {
  padding: 12px 16px;
  border-top: 1px solid #e8e8e8;
  background: #f8f9fa;
  display: flex;
  justify-content: center;
  align-items: center;
}

.settings-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  border-radius: 50%;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  color: #666;
  transition: all 0.2s;
}

.settings-btn:hover {
  background: #e0e0e0;
  color: #333;
}
</style>

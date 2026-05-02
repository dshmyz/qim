# 频道界面优化实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 重构频道界面，实现现代简约风格，采用侧边栏+标签页布局，提供列表/卡片双视图和卡片/时间线双模式消息展示

**架构：** 采用侧边栏（280px）+ 标签页导航 + 内容区的三栏布局，使用 Pinia 管理频道状态，CSS Variables 实现主题，CSS Transitions/Animations 实现交互动效

**技术栈：** Vue 3 + TypeScript + Pinia + CSS Variables + CSS Animations

---

## 文件结构

### 新建文件

**状态管理：**
- `src/stores/channel.ts` - 频道状态管理（频道列表、选中状态、标签页、视图模式）

**组件文件：**
- `src/components/channel/ChannelSidebar.vue` - 左侧边栏容器
- `src/components/channel/ChannelListContainer.vue` - 频道列表容器（包含头部和视图切换）
- `src/components/channel/ChannelListItem.vue` - 列表视图项
- `src/components/channel/ChannelListCard.vue` - 卡片视图项
- `src/components/channel/TabNavigation.vue` - 标签页导航
- `src/components/channel/ChannelDetailNew.vue` - 新的频道详情组件
- `src/components/channel/ChannelHeader.vue` - 频道头部信息
- `src/components/channel/MessageList.vue` - 消息列表容器
- `src/components/channel/MessageCard.vue` - 卡片模式消息
- `src/components/channel/MessageTimeline.vue` - 时间线模式消息

**样式文件：**
- `src/styles/channel.css` - 频道专用样式变量和基础样式

### 修改文件

- `src/views/Main.vue` - 集成新的频道布局
- `src/components/shared/ChannelList.vue` - 保留作为兼容层，逐步废弃
- `src/components/channel/ChannelDetail.vue` - 保留作为兼容层，逐步废弃

---

## 任务 1：创建频道状态管理

**文件：**
- 创建：`src/stores/channel.ts`

- [ ] **步骤 1：创建频道 store 文件**

创建 `src/stores/channel.ts`：

```typescript
import { defineStore } from 'pinia'
import type { Channel } from '../types'

interface ChannelState {
  channels: Channel[]
  selectedChannelId: string | number | null
  openTabs: Array<{ id: string | number; name: string }>
  viewMode: 'list' | 'card'
  messageMode: 'card' | 'timeline'
  loading: boolean
}

export const useChannelStore = defineStore('channel', {
  state: (): ChannelState => ({
    channels: [],
    selectedChannelId: null,
    openTabs: [],
    viewMode: 'card',
    messageMode: 'card',
    loading: false,
  }),

  getters: {
    selectedChannel: (state) => {
      return state.channels.find(c => c.id === state.selectedChannelId)
    },
    subscribedChannels: (state) => {
      return state.channels.filter(c => c.is_subscribed)
    },
  },

  actions: {
    async fetchChannels() {
      this.loading = true
      try {
        const serverUrl = localStorage.getItem('serverUrl') || ''
        const response = await fetch(`${serverUrl}/api/v1/channels`, {
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('token')}`,
            'Content-Type': 'application/json'
          }
        })
        const data = await response.json()
        if (data.code === 0) {
          this.channels = data.data || []
        }
      } catch (error) {
        console.error('加载频道失败:', error)
      } finally {
        this.loading = false
      }
    },

    async subscribeChannel(channelId: string | number) {
      try {
        const serverUrl = localStorage.getItem('serverUrl') || ''
        const response = await fetch(`${serverUrl}/api/v1/channels/${channelId}/subscribe`, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('token')}`,
            'Content-Type': 'application/json'
          }
        })
        const data = await response.json()
        if (data.code === 0) {
          const channel = this.channels.find(c => c.id === channelId)
          if (channel) {
            channel.is_subscribed = true
          }
        }
      } catch (error) {
        console.error('订阅频道失败:', error)
      }
    },

    async unsubscribeChannel(channelId: string | number) {
      try {
        const serverUrl = localStorage.getItem('serverUrl') || ''
        const response = await fetch(`${serverUrl}/api/v1/channels/${channelId}/unsubscribe`, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('token')}`,
            'Content-Type': 'application/json'
          }
        })
        const data = await response.json()
        if (data.code === 0) {
          const channel = this.channels.find(c => c.id === channelId)
          if (channel) {
            channel.is_subscribed = false
          }
        }
      } catch (error) {
        console.error('取消订阅失败:', error)
      }
    },

    selectChannel(channelId: string | number) {
      this.selectedChannelId = channelId
      const channel = this.channels.find(c => c.id === channelId)
      if (channel) {
        this.addTab(channel)
      }
    },

    addTab(channel: Channel) {
      const exists = this.openTabs.find(t => t.id === channel.id)
      if (!exists) {
        this.openTabs.push({ id: channel.id, name: channel.name })
      }
    },

    removeTab(channelId: string | number) {
      const index = this.openTabs.findIndex(t => t.id === channelId)
      if (index > -1) {
        this.openTabs.splice(index, 1)
        if (this.selectedChannelId === channelId) {
          this.selectedChannelId = this.openTabs.length > 0 ? this.openTabs[0].id : null
        }
      }
    },

    setViewMode(mode: 'list' | 'card') {
      this.viewMode = mode
    },

    setMessageMode(mode: 'card' | 'timeline') {
      this.messageMode = mode
    },
  },
})
```

- [ ] **步骤 2：Commit**

```bash
git add src/stores/channel.ts
git commit -m "feat: 添加频道状态管理 store"
```

---

## 任务 2：创建频道样式文件

**文件：**
- 创建：`src/styles/channel.css`

- [ ] **步骤 1：创建频道样式文件**

创建 `src/styles/channel.css`：

```css
:root {
  --channel-primary: #000000;
  --channel-secondary: #666666;
  --channel-tertiary: #999999;
  --channel-bg-primary: #FFFFFF;
  --channel-bg-secondary: #FAFAFA;
  --channel-bg-tertiary: #F5F5F5;
  --channel-border: #E5E5E5;
  
  --channel-spacing-xs: 4px;
  --channel-spacing-sm: 8px;
  --channel-spacing-md: 12px;
  --channel-spacing-lg: 16px;
  --channel-spacing-xl: 20px;
  
  --channel-radius-sm: 6px;
  --channel-radius-md: 8px;
  --channel-radius-lg: 12px;
  
  --channel-shadow-sm: 0 2px 8px rgba(0, 0, 0, 0.08);
  --channel-shadow-md: 0 4px 16px rgba(0, 0, 0, 0.12);
  
  --channel-transition-fast: 100ms;
  --channel-transition-normal: 200ms;
  --channel-transition-slow: 300ms;
}

.channel-sidebar {
  width: 280px;
  background: var(--channel-bg-primary);
  border-right: 1px solid var(--channel-border);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.channel-sidebar-header {
  padding: var(--channel-spacing-xl);
  border-bottom: 1px solid var(--channel-border);
}

.channel-sidebar-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--channel-spacing-md);
}

.channel-view-toggle {
  display: flex;
  gap: var(--channel-spacing-sm);
  margin-top: var(--channel-spacing-md);
}

.channel-view-toggle button {
  flex: 1;
  padding: var(--channel-spacing-sm);
  border: 1px solid var(--channel-border);
  border-radius: var(--channel-radius-sm);
  background: transparent;
  color: var(--channel-secondary);
  font-size: 11px;
  cursor: pointer;
  transition: all var(--channel-transition-normal);
}

.channel-view-toggle button.active {
  background: var(--channel-bg-tertiary);
  color: var(--channel-primary);
  border-color: var(--channel-border);
}

.channel-view-toggle button:hover {
  background: var(--channel-bg-secondary);
}

.channel-tabs {
  background: var(--channel-bg-primary);
  border-bottom: 1px solid var(--channel-border);
  padding: var(--channel-spacing-md) var(--channel-spacing-xl);
  display: flex;
  align-items: center;
  gap: var(--channel-spacing-sm);
  overflow-x: auto;
}

.channel-tab {
  background: var(--channel-bg-primary);
  border: 1px solid var(--channel-border);
  border-radius: var(--channel-radius-sm);
  padding: var(--channel-spacing-sm) var(--channel-spacing-lg);
  font-size: 13px;
  color: var(--channel-secondary);
  cursor: pointer;
  white-space: nowrap;
  transition: all var(--channel-transition-normal);
  display: flex;
  align-items: center;
  gap: var(--channel-spacing-sm);
}

.channel-tab.active {
  background: var(--channel-bg-tertiary);
  color: var(--channel-primary);
  border-color: var(--channel-border);
}

.channel-tab:hover {
  background: var(--channel-bg-secondary);
}

.channel-tab-close {
  color: var(--channel-tertiary);
  font-size: 11px;
  cursor: pointer;
  transition: color var(--channel-transition-fast);
}

.channel-tab-close:hover {
  color: var(--channel-primary);
}

.channel-add-tab {
  background: transparent;
  border: 1px dashed var(--channel-border);
  border-radius: var(--channel-radius-sm);
  padding: var(--channel-spacing-sm) var(--channel-spacing-md);
  color: var(--channel-tertiary);
  cursor: pointer;
  transition: all var(--channel-transition-normal);
}

.channel-add-tab:hover {
  border-color: var(--channel-secondary);
  color: var(--channel-secondary);
}

.channel-content {
  flex: 1;
  background: var(--channel-bg-primary);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.channel-detail {
  flex: 1;
  padding: var(--channel-spacing-xl) var(--channel-spacing-xl);
  overflow-y: auto;
  max-width: 800px;
  margin: 0 auto;
  width: 100%;
}

.channel-list-item {
  background: var(--channel-bg-primary);
  border-radius: var(--channel-radius-md);
  padding: var(--channel-spacing-md);
  margin-bottom: var(--channel-spacing-sm);
  border: 1px solid var(--channel-border);
  cursor: pointer;
  transition: all var(--channel-transition-normal);
}

.channel-list-item:hover {
  background: var(--channel-bg-secondary);
  transform: translateY(-2px);
  box-shadow: var(--channel-shadow-sm);
}

.channel-list-item.selected {
  background: var(--channel-bg-tertiary);
  border-left: 3px solid var(--channel-primary);
}

.channel-card {
  background: var(--channel-bg-secondary);
  border-radius: var(--channel-radius-lg);
  padding: var(--channel-spacing-lg);
  margin-bottom: var(--channel-spacing-md);
  border: 1px solid var(--channel-border);
  cursor: pointer;
  transition: all var(--channel-transition-normal);
}

.channel-card:hover {
  background: var(--channel-bg-tertiary);
  transform: translateY(-2px);
  box-shadow: var(--channel-shadow-md);
}

.channel-card.selected {
  border: 2px solid var(--channel-primary);
}

.message-card {
  background: var(--channel-bg-secondary);
  border-radius: var(--channel-radius-lg);
  padding: var(--channel-spacing-xl);
  margin-bottom: var(--channel-spacing-md);
  border: 1px solid var(--channel-border);
  transition: all var(--channel-transition-normal);
}

.message-card:hover {
  border-color: var(--channel-secondary);
}

.message-timeline {
  position: relative;
  padding-left: 24px;
}

.message-timeline::before {
  content: '';
  position: absolute;
  left: 8px;
  top: 0;
  bottom: 0;
  width: 2px;
  background: var(--channel-border);
}

.message-timeline-item {
  position: relative;
  margin-bottom: var(--channel-spacing-xl);
}

.message-timeline-dot {
  position: absolute;
  left: -20px;
  top: 4px;
  width: 12px;
  height: 12px;
  background: var(--channel-border);
  border-radius: 50%;
  border: 2px solid var(--channel-bg-primary);
}

.message-timeline-dot.latest {
  background: var(--channel-primary);
}

@media (max-width: 768px) {
  .channel-sidebar {
    width: 100%;
    position: fixed;
    left: 0;
    top: 0;
    bottom: 0;
    z-index: 1000;
    transform: translateX(-100%);
    transition: transform var(--channel-transition-slow);
  }

  .channel-sidebar.open {
    transform: translateX(0);
  }
}
```

- [ ] **步骤 2：Commit**

```bash
git add src/styles/channel.css
git commit -m "feat: 添加频道样式文件"
```

---

## 任务 3：创建侧边栏容器组件

**文件：**
- 创建：`src/components/channel/ChannelSidebar.vue`

- [ ] **步骤 1：创建侧边栏容器组件**

创建 `src/components/channel/ChannelSidebar.vue`：

```vue
<template>
  <div class="channel-sidebar">
    <div class="channel-sidebar-header">
      <div class="channel-header-top">
        <h3>频道</h3>
        <button v-if="hasAdminPermission" class="create-channel-btn" @click="$emit('createChannel')">
          创建
        </button>
      </div>
      
      <div class="channel-tabs-toggle">
        <button 
          :class="{ active: activeTab === 'subscribed' }"
          @click="activeTab = 'subscribed'"
        >
          订阅
        </button>
        <button 
          :class="{ active: activeTab === 'discover' }"
          @click="activeTab = 'discover'"
        >
          广场
        </button>
      </div>
      
      <div class="channel-view-toggle">
        <button 
          :class="{ active: viewMode === 'list' }"
          @click="setViewMode('list')"
        >
          ☰ 列表
        </button>
        <button 
          :class="{ active: viewMode === 'card' }"
          @click="setViewMode('card')"
        >
          ▦ 卡片
        </button>
      </div>
    </div>
    
    <div class="channel-sidebar-content">
      <ChannelListContainer
        :channels="filteredChannels"
        :view-mode="viewMode"
        :loading="loading"
        :selected-channel-id="selectedChannelId"
        @select="handleSelectChannel"
        @subscribe="handleSubscribe"
        @unsubscribe="handleUnsubscribe"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useChannelStore } from '../../stores/channel'
import ChannelListContainer from './ChannelListContainer.vue'
import type { Channel } from '../../types'

const props = defineProps<{
  currentUser: Record<string, any>
}>()

const emit = defineEmits<{
  createChannel: []
}>()

const channelStore = useChannelStore()
const activeTab = ref<'subscribed' | 'discover'>('subscribed')

const viewMode = computed(() => channelStore.viewMode)
const loading = computed(() => channelStore.loading)
const selectedChannelId = computed(() => channelStore.selectedChannelId)

const filteredChannels = computed(() => {
  if (activeTab.value === 'subscribed') {
    return channelStore.subscribedChannels
  }
  return channelStore.channels
})

const hasAdminPermission = computed(() => {
  return props.currentUser.isAdmin || props.currentUser.roles?.includes('system_admin')
})

const setViewMode = (mode: 'list' | 'card') => {
  channelStore.setViewMode(mode)
}

const handleSelectChannel = (channel: Channel) => {
  channelStore.selectChannel(channel.id)
}

const handleSubscribe = (channel: Channel) => {
  channelStore.subscribeChannel(channel.id)
}

const handleUnsubscribe = (channel: Channel) => {
  channelStore.unsubscribeChannel(channel.id)
}
</script>

<style scoped>
.channel-header-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.channel-header-top h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--channel-primary);
}

.create-channel-btn {
  background: var(--channel-primary);
  color: white;
  border: none;
  padding: 8px 12px;
  border-radius: var(--channel-radius-sm);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--channel-transition-normal);
}

.create-channel-btn:hover {
  opacity: 0.8;
  transform: translateY(-1px);
}

.channel-tabs-toggle {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.channel-tabs-toggle button {
  flex: 1;
  padding: 8px;
  border: none;
  border-radius: var(--channel-radius-sm);
  background: transparent;
  color: var(--channel-secondary);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--channel-transition-normal);
}

.channel-tabs-toggle button.active {
  background: var(--channel-primary);
  color: white;
}

.channel-tabs-toggle button:hover:not(.active) {
  background: var(--channel-bg-secondary);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/channel/ChannelSidebar.vue
git commit -m "feat: 创建侧边栏容器组件"
```

---

## 任务 4：创建频道列表容器组件

**文件：**
- 创建：`src/components/channel/ChannelListContainer.vue`

- [ ] **步骤 1：创建频道列表容器组件**

创建 `src/components/channel/ChannelListContainer.vue`：

```vue
<template>
  <div class="channel-list-container">
    <div v-if="loading" class="channel-loading">
      <div class="loading-spinner"></div>
      <span>加载中...</span>
    </div>
    
    <div v-else-if="channels.length === 0" class="channel-empty">
      <i class="fas fa-bullhorn"></i>
      <h4>暂无频道</h4>
      <p>去频道广场订阅感兴趣的频道吧！</p>
    </div>
    
    <div v-else>
      <ChannelListItem
        v-if="viewMode === 'list'"
        v-for="channel in channels"
        :key="channel.id"
        :channel="channel"
        :selected="channel.id === selectedChannelId"
        @click="$emit('select', channel)"
        @subscribe="$emit('subscribe', channel)"
        @unsubscribe="$emit('unsubscribe', channel)"
      />
      
      <ChannelListCard
        v-else
        v-for="channel in channels"
        :key="channel.id"
        :channel="channel"
        :selected="channel.id === selectedChannelId"
        @click="$emit('select', channel)"
        @subscribe="$emit('subscribe', channel)"
        @unsubscribe="$emit('unsubscribe', channel)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import ChannelListItem from './ChannelListItem.vue'
import ChannelListCard from './ChannelListCard.vue'
import type { Channel } from '../../types'

defineProps<{
  channels: Channel[]
  viewMode: 'list' | 'card'
  loading: boolean
  selectedChannelId: string | number | null
}>()

defineEmits<{
  select: [channel: Channel]
  subscribe: [channel: Channel]
  unsubscribe: [channel: Channel]
}>()
</script>

<style scoped>
.channel-list-container {
  min-height: 200px;
}

.channel-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 0;
  color: var(--channel-tertiary);
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--channel-border);
  border-top: 3px solid var(--channel-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 12px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.channel-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 0;
  color: var(--channel-tertiary);
  text-align: center;
}

.channel-empty i {
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.5;
}

.channel-empty h4 {
  margin: 0 0 8px 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--channel-primary);
}

.channel-empty p {
  margin: 0;
  font-size: 13px;
  color: var(--channel-secondary);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/channel/ChannelListContainer.vue
git commit -m "feat: 创建频道列表容器组件"
```

---

## 任务 5：创建列表视图项组件

**文件：**
- 创建：`src/components/channel/ChannelListItem.vue`

- [ ] **步骤 1：创建列表视图项组件**

创建 `src/components/channel/ChannelListItem.vue`：

```vue
<template>
  <div 
    class="channel-list-item"
    :class="{ selected }"
    @click="$emit('click')"
  >
    <div class="channel-item-content">
      <img 
        :src="channel.avatar || generateAvatar(channel.name)" 
        :alt="channel.name"
        class="channel-avatar"
      />
      <div class="channel-info">
        <div class="channel-name-row">
          <span class="channel-name">{{ channel.name }}</span>
          <span v-if="unreadCount > 0" class="unread-badge">{{ unreadCount }}</span>
        </div>
        <div class="channel-preview">{{ latestMessage }}</div>
      </div>
      <span class="channel-time">{{ formatTime(channel.updated_at || channel.created_at) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { generateAvatar } from '../../utils/avatar'
import type { Channel } from '../../types'

const props = defineProps<{
  channel: Channel
  selected: boolean
}>()

defineEmits<{
  click: []
  subscribe: []
  unsubscribe: []
}>()

const unreadCount = computed(() => {
  return props.channel.unread_count || 0
})

const latestMessage = computed(() => {
  if (props.channel.messages && props.channel.messages.length > 0) {
    const latest = props.channel.messages[0]
    return latest.content.substring(0, 30) + (latest.content.length > 30 ? '...' : '')
  }
  return '暂无消息'
})

const formatTime = (date: string) => {
  if (!date) return ''
  const d = new Date(date)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  
  if (days === 0) {
    return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  } else if (days === 1) {
    return '昨天'
  } else if (days < 7) {
    return `${days}天前`
  } else {
    return d.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' })
  }
}
</script>

<style scoped>
.channel-item-content {
  display: flex;
  align-items: center;
  gap: 12px;
}

.channel-avatar {
  width: 36px;
  height: 36px;
  border-radius: var(--channel-radius-md);
  object-fit: cover;
  flex-shrink: 0;
}

.channel-info {
  flex: 1;
  min-width: 0;
}

.channel-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 2px;
}

.channel-name {
  font-weight: 600;
  font-size: 14px;
  color: var(--channel-primary);
}

.unread-badge {
  background: var(--channel-primary);
  color: white;
  padding: 2px 6px;
  border-radius: 10px;
  font-size: 10px;
  font-weight: 600;
}

.channel-preview {
  font-size: 12px;
  color: var(--channel-tertiary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.channel-time {
  font-size: 11px;
  color: var(--channel-tertiary);
  white-space: nowrap;
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/channel/ChannelListItem.vue
git commit -m "feat: 创建列表视图项组件"
```

---

## 任务 6：创建卡片视图项组件

**文件：**
- 创建：`src/components/channel/ChannelListCard.vue`

- [ ] **步骤 1：创建卡片视图项组件**

创建 `src/components/channel/ChannelListCard.vue`：

```vue
<template>
  <div 
    class="channel-card"
    :class="{ selected }"
    @click="$emit('click')"
  >
    <div class="channel-card-header">
      <img 
        :src="channel.avatar || generateAvatar(channel.name)" 
        :alt="channel.name"
        class="channel-avatar"
      />
      <div class="channel-info">
        <div class="channel-name-row">
          <span class="channel-name">{{ channel.name }}</span>
          <span v-if="unreadCount > 0" class="unread-badge">{{ unreadCount }} 条新消息</span>
        </div>
        <div class="channel-description">{{ channel.description }}</div>
        <div class="channel-latest">{{ latestMessage }}</div>
      </div>
    </div>
    
    <div class="channel-card-footer">
      <div class="channel-meta">
        <span>创建者：{{ channel.creator?.name || '未知' }}</span>
        <span>{{ channel.subscriber_count || 0 }} 订阅</span>
      </div>
      <button 
        class="subscribe-btn"
        :class="{ subscribed: channel.is_subscribed }"
        @click.stop="handleSubscribe"
      >
        {{ channel.is_subscribed ? '已订阅' : '订阅' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { generateAvatar } from '../../utils/avatar'
import type { Channel } from '../../types'

const props = defineProps<{
  channel: Channel
  selected: boolean
}>()

const emit = defineEmits<{
  click: []
  subscribe: []
  unsubscribe: []
}>()

const unreadCount = computed(() => {
  return props.channel.unread_count || 0
})

const latestMessage = computed(() => {
  if (props.channel.messages && props.channel.messages.length > 0) {
    const latest = props.channel.messages[0]
    return '最新：' + latest.content.substring(0, 20) + (latest.content.length > 20 ? '...' : '')
  }
  return '暂无消息'
})

const handleSubscribe = () => {
  if (props.channel.is_subscribed) {
    emit('unsubscribe')
  } else {
    emit('subscribe')
  }
}
</script>

<style scoped>
.channel-card-header {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 12px;
}

.channel-avatar {
  width: 48px;
  height: 48px;
  border-radius: var(--channel-radius-md);
  object-fit: cover;
  flex-shrink: 0;
}

.channel-info {
  flex: 1;
}

.channel-name-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}

.channel-name {
  font-weight: 600;
  font-size: 15px;
  color: var(--channel-primary);
}

.unread-badge {
  background: var(--channel-bg-tertiary);
  color: var(--channel-secondary);
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 600;
}

.channel-description {
  font-size: 13px;
  color: var(--channel-secondary);
  margin-bottom: 8px;
}

.channel-latest {
  font-size: 12px;
  color: var(--channel-tertiary);
}

.channel-card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 12px;
  border-top: 1px solid var(--channel-border);
}

.channel-meta {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 12px;
  color: var(--channel-tertiary);
}

.subscribe-btn {
  background: var(--channel-primary);
  color: white;
  border: none;
  padding: 6px 12px;
  border-radius: var(--channel-radius-sm);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--channel-transition-normal);
}

.subscribe-btn.subscribed {
  background: var(--channel-bg-tertiary);
  color: var(--channel-primary);
}

.subscribe-btn:hover {
  opacity: 0.8;
  transform: translateY(-1px);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/channel/ChannelListCard.vue
git commit -m "feat: 创建卡片视图项组件"
```

---

## 任务 7：创建标签页导航组件

**文件：**
- 创建：`src/components/channel/TabNavigation.vue`

- [ ] **步骤 1：创建标签页导航组件**

创建 `src/components/channel/TabNavigation.vue`：

```vue
<template>
  <div class="channel-tabs">
    <div
      v-for="tab in tabs"
      :key="tab.id"
      class="channel-tab"
      :class="{ active: tab.id === activeTabId }"
      @click="$emit('select', tab.id)"
    >
      <span>{{ tab.name }}</span>
      <span class="channel-tab-close" @click.stop="$emit('close', tab.id)">×</span>
    </div>
    <button class="channel-add-tab" @click="$emit('add')">+</button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  tabs: Array<{ id: string | number; name: string }>
  activeTabId: string | number | null
}>()

defineEmits<{
  select: [tabId: string | number]
  close: [tabId: string | number]
  add: []
}>()
</script>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/channel/TabNavigation.vue
git commit -m "feat: 创建标签页导航组件"
```

---

## 任务 8：创建频道头部组件

**文件：**
- 创建：`src/components/channel/ChannelHeader.vue`

- [ ] **步骤 1：创建频道头部组件**

创建 `src/components/channel/ChannelHeader.vue`：

```vue
<template>
  <div class="channel-header">
    <img 
      :src="channel.avatar || generateAvatar(channel.name)" 
      :alt="channel.name"
      class="channel-avatar"
    />
    <div class="channel-info">
      <h2>{{ channel.name }}</h2>
      <p>{{ channel.description }}</p>
    </div>
    <button 
      class="subscribe-btn"
      :class="{ subscribed: channel.is_subscribed }"
      @click="handleSubscribe"
    >
      {{ channel.is_subscribed ? '已订阅' : '订阅' }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { generateAvatar } from '../../utils/avatar'
import type { Channel } from '../../types'

const props = defineProps<{
  channel: Channel
}>()

const emit = defineEmits<{
  subscribe: []
  unsubscribe: []
}>()

const handleSubscribe = () => {
  if (props.channel.is_subscribed) {
    emit('unsubscribe')
  } else {
    emit('subscribe')
  }
}
</script>

<style scoped>
.channel-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--channel-border);
}

.channel-avatar {
  width: 48px;
  height: 48px;
  border-radius: var(--channel-radius-md);
  object-fit: cover;
}

.channel-info {
  flex: 1;
}

.channel-info h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--channel-primary);
}

.channel-info p {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: var(--channel-tertiary);
}

.subscribe-btn {
  background: var(--channel-primary);
  color: white;
  border: none;
  padding: 10px 20px;
  border-radius: var(--channel-radius-sm);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--channel-transition-normal);
}

.subscribe-btn.subscribed {
  background: var(--channel-bg-tertiary);
  color: var(--channel-primary);
}

.subscribe-btn:hover {
  opacity: 0.8;
  transform: translateY(-1px);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/channel/ChannelHeader.vue
git commit -m "feat: 创建频道头部组件"
```

---

## 任务 9：创建消息列表容器组件

**文件：**
- 创建：`src/components/channel/MessageList.vue`

- [ ] **步骤 1：创建消息列表容器组件**

创建 `src/components/channel/MessageList.vue`：

```vue
<template>
  <div class="message-list-container">
    <div class="message-list-header">
      <div class="message-mode-toggle">
        <button 
          :class="{ active: mode === 'card' }"
          @click="$emit('update:mode', 'card')"
        >
          卡片
        </button>
        <button 
          :class="{ active: mode === 'timeline' }"
          @click="$emit('update:mode', 'timeline')"
        >
          时间线
        </button>
      </div>
      <select v-model="sortOrder" class="message-sort">
        <option value="newest">最新优先</option>
        <option value="oldest">最早优先</option>
      </select>
    </div>
    
    <div v-if="messages.length === 0" class="message-empty">
      <i class="fas fa-comment-alt"></i>
      <p>暂无消息</p>
    </div>
    
    <div v-else>
      <MessageCard
        v-if="mode === 'card'"
        v-for="message in sortedMessages"
        :key="message.id"
        :message="message"
        :is-creator="isCreator"
      />
      
      <MessageTimeline
        v-else
        :messages="sortedMessages"
        :is-creator="isCreator"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import MessageCard from './MessageCard.vue'
import MessageTimeline from './MessageTimeline.vue'
import type { ChannelMessage } from '../../types'

const props = defineProps<{
  messages: ChannelMessage[]
  mode: 'card' | 'timeline'
  isCreator: boolean
}>()

const emit = defineEmits<{
  'update:mode': [mode: 'card' | 'timeline']
}>()

const sortOrder = ref<'newest' | 'oldest'>('newest')

const sortedMessages = computed(() => {
  const sorted = [...props.messages]
  if (sortOrder.value === 'newest') {
    sorted.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
  } else {
    sorted.sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime())
  }
  return sorted
})
</script>

<style scoped>
.message-list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.message-mode-toggle {
  display: flex;
  gap: 8px;
}

.message-mode-toggle button {
  padding: 8px 16px;
  border: 1px solid var(--channel-border);
  border-radius: var(--channel-radius-sm);
  background: transparent;
  color: var(--channel-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all var(--channel-transition-normal);
}

.message-mode-toggle button.active {
  background: var(--channel-primary);
  color: white;
  border-color: var(--channel-primary);
}

.message-mode-toggle button:hover:not(.active) {
  background: var(--channel-bg-secondary);
}

.message-sort {
  padding: 8px 12px;
  border: 1px solid var(--channel-border);
  border-radius: var(--channel-radius-sm);
  font-size: 12px;
  background: white;
  color: var(--channel-primary);
  cursor: pointer;
}

.message-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 0;
  color: var(--channel-tertiary);
}

.message-empty i {
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.5;
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/channel/MessageList.vue
git commit -m "feat: 创建消息列表容器组件"
```

---

## 任务 10：创建卡片模式消息组件

**文件：**
- 创建：`src/components/channel/MessageCard.vue`

- [ ] **步骤 1：创建卡片模式消息组件**

创建 `src/components/channel/MessageCard.vue`：

```vue
<template>
  <div class="message-card">
    <div class="message-header">
      <img 
        :src="message.sender?.avatar || generateAvatar(message.sender?.name || 'user')" 
        :alt="message.sender?.name"
        class="sender-avatar"
      />
      <div class="sender-info">
        <div class="sender-name-row">
          <span class="sender-name">{{ message.sender?.name || '未知' }}</span>
          <span v-if="isCreator" class="creator-badge">创建者</span>
        </div>
        <span class="message-time">{{ formatTime(message.created_at) }}</span>
      </div>
      <button class="more-btn">⋮</button>
    </div>
    
    <div class="message-content">
      {{ message.content }}
    </div>
    
    <div class="message-actions">
      <button class="action-btn">
        <span>👍</span>
        <span>{{ message.likes || 0 }}</span>
      </button>
      <button class="action-btn">
        <span>💬</span>
        <span>{{ message.comments || 0 }}</span>
      </button>
      <button class="action-btn">
        <span>🔗</span>
        <span>复制链接</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { generateAvatar } from '../../utils/avatar'
import type { ChannelMessage } from '../../types'

defineProps<{
  message: ChannelMessage
  isCreator: boolean
}>()

const formatTime = (date: string) => {
  if (!date) return ''
  const d = new Date(date)
  return d.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}
</script>

<style scoped>
.message-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.sender-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
}

.sender-info {
  flex: 1;
}

.sender-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 2px;
}

.sender-name {
  font-weight: 600;
  font-size: 14px;
  color: var(--channel-primary);
}

.creator-badge {
  background: var(--channel-primary);
  color: white;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 600;
}

.message-time {
  font-size: 12px;
  color: var(--channel-tertiary);
}

.more-btn {
  background: transparent;
  border: none;
  color: var(--channel-tertiary);
  cursor: pointer;
  font-size: 18px;
  padding: 4px;
}

.more-btn:hover {
  color: var(--channel-primary);
}

.message-content {
  font-size: 14px;
  line-height: 1.6;
  color: var(--channel-primary);
  margin-bottom: 16px;
}

.message-actions {
  display: flex;
  align-items: center;
  gap: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--channel-border);
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  border: none;
  color: var(--channel-secondary);
  cursor: pointer;
  font-size: 12px;
  transition: color var(--channel-transition-fast);
}

.action-btn:hover {
  color: var(--channel-primary);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/channel/MessageCard.vue
git commit -m "feat: 创建卡片模式消息组件"
```

---

## 任务 11：创建时间线模式消息组件

**文件：**
- 创建：`src/components/channel/MessageTimeline.vue`

- [ ] **步骤 1：创建时间线模式消息组件**

创建 `src/components/channel/MessageTimeline.vue`：

```vue
<template>
  <div class="message-timeline">
    <div
      v-for="(message, index) in messages"
      :key="message.id"
      class="message-timeline-item"
    >
      <div 
        class="message-timeline-dot"
        :class="{ latest: index === 0 }"
      ></div>
      <div class="message-timeline-content">
        <div class="timeline-header">
          <span class="timeline-sender">{{ message.sender?.name || '未知' }}</span>
          <span class="timeline-time">{{ formatTime(message.created_at) }}</span>
        </div>
        <div class="timeline-text">{{ message.content }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ChannelMessage } from '../../types'

defineProps<{
  messages: ChannelMessage[]
  isCreator: boolean
}>()

const formatTime = (date: string) => {
  if (!date) return ''
  const d = new Date(date)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  
  if (days === 0) {
    return '今天 ' + d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  } else if (days === 1) {
    return '昨天 ' + d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  } else if (days < 7) {
    return `${days}天前`
  } else {
    return d.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' })
  }
}
</script>

<style scoped>
.message-timeline-content {
  background: var(--channel-bg-secondary);
  border-radius: var(--channel-radius-md);
  padding: 16px;
  border: 1px solid var(--channel-border);
}

.timeline-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.timeline-sender {
  font-weight: 600;
  font-size: 13px;
  color: var(--channel-primary);
}

.timeline-time {
  font-size: 11px;
  color: var(--channel-tertiary);
}

.timeline-text {
  font-size: 13px;
  line-height: 1.5;
  color: var(--channel-primary);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/channel/MessageTimeline.vue
git commit -m "feat: 创建时间线模式消息组件"
```

---

## 任务 12：创建新的频道详情组件

**文件：**
- 创建：`src/components/channel/ChannelDetailNew.vue`

- [ ] **步骤 1：创建新的频道详情组件**

创建 `src/components/channel/ChannelDetailNew.vue`：

```vue
<template>
  <div class="channel-detail">
    <div v-if="!channel" class="channel-empty-state">
      <i class="fas fa-bullhorn"></i>
      <h3>选择一个频道</h3>
      <p>从左侧列表中选择一个频道查看详情</p>
    </div>
    
    <div v-else>
      <ChannelHeader
        :channel="channel"
        @subscribe="handleSubscribe"
        @unsubscribe="handleUnsubscribe"
      />
      
      <MessageList
        :messages="channel.messages || []"
        :mode="messageMode"
        :is-creator="isCreator"
        @update:mode="setMessageMode"
      />
      
      <div v-if="isCreator" class="message-input-area">
        <textarea 
          v-model="localMessage" 
          placeholder="输入消息..." 
          rows="2"
          class="message-textarea"
        ></textarea>
        <button 
          class="send-btn" 
          @click="handleSendMessage"
          :disabled="!localMessage.trim()"
        >
          发送
        </button>
      </div>
      <div v-else-if="channel.is_subscribed" class="message-readonly-hint">
        <i class="fas fa-bullhorn"></i>
        <span>频道为广播模式，仅创建者可发布消息</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useChannelStore } from '../../stores/channel'
import ChannelHeader from './ChannelHeader.vue'
import MessageList from './MessageList.vue'
import type { Channel } from '../../types'

const props = defineProps<{
  channel: Channel | null
}>()

const emit = defineEmits<{
  sendMessage: [channel: Channel, message: string]
}>()

const channelStore = useChannelStore()
const localMessage = ref('')

const messageMode = computed(() => channelStore.messageMode)
const isCreator = computed(() => {
  if (!props.channel) return false
  const currentUser = JSON.parse(localStorage.getItem('user') || '{}')
  return props.channel.creator?.id === currentUser.id
})

const setMessageMode = (mode: 'card' | 'timeline') => {
  channelStore.setMessageMode(mode)
}

const handleSubscribe = () => {
  if (props.channel) {
    channelStore.subscribeChannel(props.channel.id)
  }
}

const handleUnsubscribe = () => {
  if (props.channel) {
    channelStore.unsubscribeChannel(props.channel.id)
  }
}

const handleSendMessage = () => {
  if (props.channel && localMessage.value.trim()) {
    emit('sendMessage', props.channel, localMessage.value.trim())
    localMessage.value = ''
  }
}
</script>

<style scoped>
.channel-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  color: var(--channel-tertiary);
}

.channel-empty-state i {
  font-size: 64px;
  margin-bottom: 24px;
  opacity: 0.3;
}

.channel-empty-state h3 {
  margin: 0 0 8px 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--channel-primary);
}

.channel-empty-state p {
  margin: 0;
  font-size: 14px;
  color: var(--channel-secondary);
}

.message-input-area {
  margin-top: 24px;
  padding-top: 24px;
  border-top: 1px solid var(--channel-border);
  display: flex;
  gap: 12px;
  align-items: flex-end;
}

.message-textarea {
  flex: 1;
  padding: 12px;
  border: 1px solid var(--channel-border);
  border-radius: var(--channel-radius-md);
  resize: none;
  font-family: inherit;
  font-size: 14px;
  background: var(--channel-bg-primary);
  color: var(--channel-primary);
}

.message-textarea:focus {
  outline: none;
  border-color: var(--channel-primary);
}

.send-btn {
  background: var(--channel-primary);
  color: white;
  border: none;
  padding: 12px 24px;
  border-radius: var(--channel-radius-md);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--channel-transition-normal);
}

.send-btn:hover:not(:disabled) {
  opacity: 0.8;
  transform: translateY(-1px);
}

.send-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.message-readonly-hint {
  margin-top: 24px;
  padding: 12px 20px;
  border-top: 1px solid var(--channel-border);
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--channel-tertiary);
  font-size: 13px;
  background: var(--channel-bg-secondary);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/channel/ChannelDetailNew.vue
git commit -m "feat: 创建新的频道详情组件"
```

---

## 任务 13：集成到主界面

**文件：**
- 修改：`src/views/Main.vue`

- [ ] **步骤 1：在 Main.vue 中引入新组件**

在 `src/views/Main.vue` 的 `<script setup>` 部分添加：

```typescript
import ChannelSidebar from '../components/channel/ChannelSidebar.vue'
import TabNavigation from '../components/channel/TabNavigation.vue'
import ChannelDetailNew from '../components/channel/ChannelDetailNew.vue'
import { useChannelStore } from '../stores/channel'
import '../styles/channel.css'

const channelStore = useChannelStore()

const handleChannelSelect = (channelId: string | number) => {
  channelStore.selectChannel(channelId)
}

const handleTabClose = (channelId: string | number) => {
  channelStore.removeTab(channelId)
}

const handleSendMessage = async (channel: Channel, message: string) => {
  try {
    const serverUrl = localStorage.getItem('serverUrl') || ''
    const response = await fetch(`${serverUrl}/api/v1/channels/${channel.id}/messages`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ content: message })
    })
    const data = await response.json()
    if (data.code === 0) {
      channelStore.fetchChannels()
    }
  } catch (error) {
    console.error('发送消息失败:', error)
  }
}

onMounted(() => {
  channelStore.fetchChannels()
})
```

- [ ] **步骤 2：在 Main.vue 的模板中添加新布局**

在 `src/views/Main.vue` 的 `<template>` 部分，找到频道相关的区域，替换为：

```vue
<div v-if="activeOption === 'channel'" class="channel-layout">
  <ChannelSidebar 
    :current-user="currentUser"
    @createChannel="showCreateChannelModal = true"
  />
  <div class="channel-main">
    <TabNavigation
      :tabs="channelStore.openTabs"
      :active-tab-id="channelStore.selectedChannelId"
      @select="handleChannelSelect"
      @close="handleTabClose"
      @add="channelStore.selectedChannelId = null"
    />
    <div class="channel-content">
      <ChannelDetailNew
        :channel="channelStore.selectedChannel"
        @send-message="handleSendMessage"
      />
    </div>
  </div>
</div>
```

- [ ] **步骤 3：添加样式**

在 `src/views/Main.vue` 的 `<style>` 部分添加：

```css
.channel-layout {
  display: flex;
  height: 100%;
  overflow: hidden;
}

.channel-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
```

- [ ] **步骤 4：Commit**

```bash
git add src/views/Main.vue
git commit -m "feat: 集成新的频道布局到主界面"
```

---

## 任务 14：测试和优化

- [ ] **步骤 1：启动开发服务器测试**

运行：`npm run dev`

预期：应用正常启动，无编译错误

- [ ] **步骤 2：测试频道列表功能**

测试项目：
1. 切换列表/卡片视图
2. 切换订阅/广场标签
3. 点击频道查看详情
4. 订阅/取消订阅频道

预期：所有功能正常工作，无控制台错误

- [ ] **步骤 3：测试标签页功能**

测试项目：
1. 打开多个频道标签页
2. 切换不同标签页
3. 关闭标签页

预期：标签页功能正常，状态保持正确

- [ ] **步骤 4：测试消息展示**

测试项目：
1. 切换卡片/时间线模式
2. 切换排序方式
3. 创建者发送消息

预期：消息展示正常，交互流畅

- [ ] **步骤 5：测试响应式布局**

测试项目：
1. 调整浏览器窗口大小
2. 测试移动端显示

预期：响应式布局正常，移动端适配良好

- [ ] **步骤 6：性能优化检查**

检查项目：
1. 使用 Chrome DevTools Performance 分析渲染性能
2. 检查是否有不必要的重渲染
3. 优化动画性能

预期：动画流畅，无明显性能问题

- [ ] **步骤 7：最终 Commit**

```bash
git add .
git commit -m "feat: 完成频道界面优化

- 实现侧边栏+标签页布局
- 添加列表/卡片双视图切换
- 实现卡片/时间线双模式消息展示
- 优化交互动效和视觉设计
- 改善响应式布局和性能"
```

---

## 自检清单

**规格覆盖度：**
- ✅ 侧边栏+标签页布局（任务 3, 7, 13）
- ✅ 列表/卡片双视图（任务 4, 5, 6）
- ✅ 卡片/时间线双模式（任务 9, 10, 11）
- ✅ 现代简约风格（任务 2）
- ✅ 状态管理（任务 1）
- ✅ 交互动效（任务 2, 14）
- ✅ 响应式设计（任务 2, 14）

**占位符扫描：**
- ✅ 无"待定"、"TODO"等占位符
- ✅ 所有代码步骤都有完整代码
- ✅ 所有命令都有具体内容

**类型一致性：**
- ✅ Channel 类型在所有组件中一致使用
- ✅ ChannelMessage 类型在消息组件中一致使用
- ✅ Store 方法和属性名在所有引用中一致

---

**计划已完成并保存到 `docs/superpowers/plans/2026-05-02-channel-optimization.md`。两种执行方式：**

**1. 子代理驱动（推荐）** - 每个任务调度一个新的子代理，任务间进行审查，快速迭代

**2. 内联执行** - 在当前会话中使用 executing-plans 执行任务，批量执行并设有检查点

**选哪种方式？**

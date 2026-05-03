# 用户分身（Avatar）前端实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 实现分身功能的前端部分，包括类型定义、API 层、Composable、设置面板组件、聊天集成组件，后端 API 用 mock 数据暂代。

**架构：** 独立分身系统，与群组 AI 和 Bot 并行。分身在私聊中直接回复（带"AI 代回复"标记），在群聊中触发后以私聊方式回复触发者。前端通过 useRequest composable 调用后端 API，状态管理使用 composable 模式。

**技术栈：** Vue 3 + TypeScript + useRequest composable + 项目共享组件（QMessage/QMessageBox/ConfirmDialog）

---

## 文件结构

### 新建文件

| 文件 | 职责 |
|------|------|
| `src/types/avatar.ts` | 分身相关类型定义 |
| `src/api/avatar.ts` | 分身 API 封装（基于 useRequest） |
| `src/composables/useAvatar.ts` | 分身核心 Composable（配置 CRUD、会话控制、接管） |
| `src/composables/useAvatarPersona.ts` | 分身人设学习 Composable（学习触发、进度查询、预览） |
| `src/components/avatar/AvatarSettingsPanel.vue` | 分身主设置面板（Tab 容器） |
| `src/components/avatar/AvatarBasicSettings.vue` | 基础设置子面板 |
| `src/components/avatar/AvatarPersonaSettings.vue` | 人设设置子面板 |
| `src/components/avatar/AvatarTriggerSettings.vue` | 触发规则子面板 |
| `src/components/avatar/AvatarKnowledgeSettings.vue` | 知识范围子面板 |
| `src/components/avatar/AvatarReplySettings.vue` | 回复策略子面板 |
| `src/components/avatar/AvatarSessionToggle.vue` | 会话级分身开关（嵌入聊天头部） |
| `src/components/avatar/AvatarTakeoverBanner.vue` | 接管状态横幅 |
| `src/components/avatar/AvatarReplyBadge.vue` | "AI 代回复"消息标记 |
| `src/components/avatar/AvatarGroupReplyNotice.vue` | 群聊分身通知 |

### 修改文件

| 文件 | 变更 |
|------|------|
| `src/views/Main.vue` | 注册分身应用面板 + 处理分身消息通知 |
| `src/components/chat/ChatHeader.vue` | 添加 AvatarSessionToggle |
| `src/components/chat/ChatWindow.vue` | 传递分身相关 props |
| `src/components/user/UserDetailPanel.vue` | 添加"分身设置"入口 |

---

## 任务 1：类型定义

**文件：**
- 创建：`src/types/avatar.ts`

- [ ] **步骤 1：创建类型文件**

```typescript
export interface AvatarConfig {
  id: number
  userId: number
  name: string
  enabled: boolean

  autoLearnedPersona: string
  customPersonaAddon: string
  personaVersion: number
  lastLearnedAt: string | null

  knowledgeScope: AvatarKnowledgeScope
  triggerRules: AvatarTriggerRules
  replyStrategy: AvatarReplyStrategy

  modelConfigId: number | null
  useSystemConfig: boolean

  takeoverCooldown: number

  createdAt: string
  updatedAt: string
}

export interface AvatarKnowledgeScope {
  conversationHistory: boolean
  knowledgeDocs: boolean
  notes: boolean
  tasks: boolean
}

export interface AvatarTriggerRules {
  mode: 'offline' | 'keyword' | 'mention' | 'all' | 'custom'
  keywords: string[]
  timeRanges: AvatarTimeRange[]
  excludedConversations: number[]
}

export interface AvatarTimeRange {
  dayOfWeek: number[]
  startHour: number
  endHour: number
}

export interface AvatarReplyStrategy {
  maxReplyLength: 'short' | 'medium' | 'long'
  replyDelay: number
  confidenceThreshold: number
  disclaimerStyle: 'badge' | 'footer' | 'both'
}

export interface AvatarSession {
  conversationId: number
  avatarEnabled: boolean
  takeoverUntil: string | null
  lastReplyAt: string | null
}

export interface AvatarLearnStatus {
  status: 'idle' | 'learning' | 'completed' | 'failed'
  progress: number
  messageCount: number
  error: string | null
}

export interface CreateAvatarConfigRequest {
  name: string
  useSystemConfig: boolean
  modelConfigId: number | null
  triggerRules: AvatarTriggerRules
  knowledgeScope: AvatarKnowledgeScope
  replyStrategy: AvatarReplyStrategy
  takeoverCooldown: number
  customPersonaAddon: string
}

export const DEFAULT_AVATAR_CONFIG: CreateAvatarConfigRequest = {
  name: '我的分身',
  useSystemConfig: true,
  modelConfigId: null,
  triggerRules: {
    mode: 'mention',
    keywords: [],
    timeRanges: [],
    excludedConversations: []
  },
  knowledgeScope: {
    conversationHistory: true,
    knowledgeDocs: false,
    notes: false,
    tasks: false
  },
  replyStrategy: {
    maxReplyLength: 'medium',
    replyDelay: 3,
    confidenceThreshold: 0.6,
    disclaimerStyle: 'badge'
  },
  takeoverCooldown: 10,
  customPersonaAddon: ''
}
```

- [ ] **步骤 2：Commit**

```bash
git add src/types/avatar.ts
git commit -m "feat(avatar): add type definitions for avatar feature"
```

---

## 任务 2：API 层

**文件：**
- 创建：`src/api/avatar.ts`

- [ ] **步骤 1：创建 API 封装**

遵循项目 `src/api/ai.ts` 的模式，使用 `useRequest` 的 `request` 函数。

```typescript
import { request } from '../composables/useRequest'
import type {
  AvatarConfig,
  AvatarSession,
  AvatarLearnStatus,
  CreateAvatarConfigRequest
} from '../types/avatar'

export const avatarAPI = {
  async getConfig(): Promise<AvatarConfig | null> {
    const response = await request<{ code: number; data: AvatarConfig | null }>(
      '/api/v1/avatar/config',
      { method: 'GET' }
    )
    return response?.data ?? null
  },

  async createConfig(data: CreateAvatarConfigRequest): Promise<AvatarConfig> {
    const response = await request<{ code: number; data: AvatarConfig }>(
      '/api/v1/avatar/config',
      {
        method: 'POST',
        body: JSON.stringify(data),
        headers: { 'Content-Type': 'application/json' }
      }
    )
    return response!.data
  },

  async updateConfig(data: Partial<AvatarConfig>): Promise<AvatarConfig> {
    const response = await request<{ code: number; data: AvatarConfig }>(
      '/api/v1/avatar/config',
      {
        method: 'PUT',
        body: JSON.stringify(data),
        headers: { 'Content-Type': 'application/json' }
      }
    )
    return response!.data
  },

  async deleteConfig(): Promise<void> {
    await request('/api/v1/avatar/config', { method: 'DELETE' })
  },

  async triggerLearnPersona(): Promise<{ taskId: string }> {
    const response = await request<{ code: number; data: { taskId: string } }>(
      '/api/v1/avatar/learn-persona',
      {
        method: 'POST',
        body: JSON.stringify({}),
        headers: { 'Content-Type': 'application/json' }
      }
    )
    return response!.data
  },

  async getLearnStatus(): Promise<AvatarLearnStatus> {
    const response = await request<{ code: number; data: AvatarLearnStatus }>(
      '/api/v1/avatar/learn-status',
      { method: 'GET' }
    )
    return response!.data
  },

  async getLearnedPersona(): Promise<string> {
    const response = await request<{ code: number; data: string }>(
      '/api/v1/avatar/learned-persona',
      { method: 'GET' }
    )
    return response!.data
  },

  async getSessions(): Promise<AvatarSession[]> {
    const response = await request<{ code: number; data: AvatarSession[] }>(
      '/api/v1/avatar/sessions',
      { method: 'GET' }
    )
    return response?.data ?? []
  },

  async updateSession(convId: number, enabled: boolean): Promise<AvatarSession> {
    const response = await request<{ code: number; data: AvatarSession }>(
      `/api/v1/avatar/sessions/${convId}`,
      {
        method: 'PUT',
        body: JSON.stringify({ avatarEnabled: enabled }),
        headers: { 'Content-Type': 'application/json' }
      }
    )
    return response!.data
  },

  async takeoverSession(convId: number): Promise<AvatarSession> {
    const response = await request<{ code: number; data: AvatarSession }>(
      `/api/v1/avatar/sessions/${convId}/takeover`,
      {
        method: 'POST',
        body: JSON.stringify({}),
        headers: { 'Content-Type': 'application/json' }
      }
    )
    return response!.data
  },

  async previewReply(message: string): Promise<string> {
    const response = await request<{ code: number; data: { reply: string } }>(
      '/api/v1/avatar/preview',
      {
        method: 'POST',
        body: JSON.stringify({ message }),
        headers: { 'Content-Type': 'application/json' }
      }
    )
    return response!.data.reply
  }
}
```

- [ ] **步骤 2：Commit**

```bash
git add src/api/avatar.ts
git commit -m "feat(avatar): add avatar API layer using useRequest"
```

---

## 任务 3：Composable 层

**文件：**
- 创建：`src/composables/useAvatar.ts`
- 创建：`src/composables/useAvatarPersona.ts`

- [ ] **步骤 1：创建 useAvatar composable**

遵循 `useModelConfigs.ts` 的模式。

```typescript
import { ref } from 'vue'
import { avatarAPI } from '../api/avatar'
import type {
  AvatarConfig,
  AvatarSession,
  CreateAvatarConfigRequest
} from '../types/avatar'

export function useAvatar() {
  const config = ref<AvatarConfig | null>(null)
  const sessions = ref<AvatarSession[]>([])
  const loading = ref(false)
  const error = ref('')

  async function fetchConfig() {
    loading.value = true
    error.value = ''
    try {
      config.value = await avatarAPI.getConfig()
    } catch (e: any) {
      error.value = e.message || '加载分身配置失败'
    } finally {
      loading.value = false
    }
  }

  async function createConfig(data: CreateAvatarConfigRequest) {
    loading.value = true
    error.value = ''
    try {
      config.value = await avatarAPI.createConfig(data)
      return config.value
    } catch (e: any) {
      error.value = e.message || '创建分身配置失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function updateConfig(updates: Partial<AvatarConfig>) {
    loading.value = true
    error.value = ''
    try {
      config.value = await avatarAPI.updateConfig(updates)
      return config.value
    } catch (e: any) {
      error.value = e.message || '更新分身配置失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function deleteConfig() {
    loading.value = true
    error.value = ''
    try {
      await avatarAPI.deleteConfig()
      config.value = null
    } catch (e: any) {
      error.value = e.message || '删除分身配置失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function toggleEnabled(enabled: boolean) {
    if (!config.value) return
    await updateConfig({ enabled })
  }

  async function fetchSessions() {
    loading.value = true
    error.value = ''
    try {
      sessions.value = await avatarAPI.getSessions()
    } catch (e: any) {
      error.value = e.message || '加载会话分身状态失败'
    } finally {
      loading.value = false
    }
  }

  async function toggleSession(convId: number, enabled: boolean) {
    loading.value = true
    error.value = ''
    try {
      const session = await avatarAPI.updateSession(convId, enabled)
      const idx = sessions.value.findIndex(s => s.conversationId === convId)
      if (idx >= 0) {
        sessions.value[idx] = session
      } else {
        sessions.value.push(session)
      }
      return session
    } catch (e: any) {
      error.value = e.message || '切换会话分身失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function takeoverSession(convId: number) {
    loading.value = true
    error.value = ''
    try {
      const session = await avatarAPI.takeoverSession(convId)
      const idx = sessions.value.findIndex(s => s.conversationId === convId)
      if (idx >= 0) {
        sessions.value[idx] = session
      }
      return session
    } catch (e: any) {
      error.value = e.message || '接管分身失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  function getSession(convId: number): AvatarSession | undefined {
    return sessions.value.find(s => s.conversationId === convId)
  }

  function isAvatarActive(convId: number): boolean {
    const session = getSession(convId)
    if (!session || !session.avatarEnabled) return false
    if (session.takeoverUntil && new Date(session.takeoverUntil) > new Date()) return false
    return true
  }

  return {
    config,
    sessions,
    loading,
    error,
    fetchConfig,
    createConfig,
    updateConfig,
    deleteConfig,
    toggleEnabled,
    fetchSessions,
    toggleSession,
    takeoverSession,
    getSession,
    isAvatarActive
  }
}
```

- [ ] **步骤 2：创建 useAvatarPersona composable**

```typescript
import { ref } from 'vue'
import { avatarAPI } from '../api/avatar'
import type { AvatarLearnStatus } from '../types/avatar'

export function useAvatarPersona() {
  const learnStatus = ref<AvatarLearnStatus>({
    status: 'idle',
    progress: 0,
    messageCount: 0,
    error: null
  })
  const learnedPersona = ref('')
  const loading = ref(false)
  const error = ref('')
  let pollTimer: ReturnType<typeof setInterval> | null = null

  async function triggerLearn() {
    loading.value = true
    error.value = ''
    try {
      await avatarAPI.triggerLearnPersona()
      startPolling()
    } catch (e: any) {
      error.value = e.message || '触发风格学习失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchLearnStatus() {
    try {
      learnStatus.value = await avatarAPI.getLearnStatus()
      if (learnStatus.value.status === 'completed' || learnStatus.value.status === 'failed') {
        stopPolling()
      }
    } catch (e: any) {
      error.value = e.message || '查询学习进度失败'
    }
  }

  async function fetchLearnedPersona() {
    loading.value = true
    error.value = ''
    try {
      learnedPersona.value = await avatarAPI.getLearnedPersona()
    } catch (e: any) {
      error.value = e.message || '获取学习结果失败'
    } finally {
      loading.value = false
    }
  }

  async function previewReply(message: string): Promise<string> {
    loading.value = true
    error.value = ''
    try {
      return await avatarAPI.previewReply(message)
    } catch (e: any) {
      error.value = e.message || '预览回复失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  function startPolling() {
    stopPolling()
    fetchLearnStatus()
    pollTimer = setInterval(fetchLearnStatus, 3000)
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  return {
    learnStatus,
    learnedPersona,
    loading,
    error,
    triggerLearn,
    fetchLearnStatus,
    fetchLearnedPersona,
    previewReply,
    startPolling,
    stopPolling
  }
}
```

- [ ] **步骤 3：Commit**

```bash
git add src/composables/useAvatar.ts src/composables/useAvatarPersona.ts
git commit -m "feat(avatar): add useAvatar and useAvatarPersona composables"
```

---

## 任务 4：AvatarReplyBadge 组件

**文件：**
- 创建：`src/components/avatar/AvatarReplyBadge.vue`

- [ ] **步骤 1：创建消息标记组件**

这是最简单的组件，先做出来，后续消息展示会用到。

```vue
<template>
  <span :class="['avatar-reply-badge', `avatar-reply-badge--${style}`]">
    <svg viewBox="0 0 24 24" width="12" height="12" fill="currentColor">
      <path d="M12 2a2 2 0 011 .26A2 2 0 0114 2h4a2 2 0 012 2v4a2 2 0 01-.26 1A2 2 0 0120 10v4a2 2 0 01-2 2h-4a2 2 0 01-1-.26A2 2 0 0112 16H8a2 2 0 01-2-2v-4a2 2 0 01.26-1A2 2 0 016 8V4a2 2 0 012-2h4zm0 2H8v4h4V4zm6 0h-4v4h4V4zm-6 6H8v4h4v-4zm6 0h-4v4h4v-4z"/>
    </svg>
    <span class="badge-text">AI 代回复</span>
  </span>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  style?: 'badge' | 'footer' | 'both'
}>(), {
  style: 'badge'
})
</script>

<style scoped>
.avatar-reply-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 500;
  line-height: 1;
  white-space: nowrap;
}

.avatar-reply-badge--badge {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.avatar-reply-badge--footer {
  background: transparent;
  color: var(--text-secondary);
  font-size: 11px;
  padding: 0;
}

.avatar-reply-badge--both {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.badge-text {
  font-size: 11px;
}

[data-theme="elegant-dark"] .avatar-reply-badge--badge,
[data-theme="elegant-dark"] .avatar-reply-badge--both {
  background: rgba(59, 130, 246, 0.2);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/avatar/AvatarReplyBadge.vue
git commit -m "feat(avatar): add AvatarReplyBadge component"
```

---

## 任务 5：AvatarGroupReplyNotice 组件

**文件：**
- 创建：`src/components/avatar/AvatarGroupReplyNotice.vue`

- [ ] **步骤 1：创建群聊分身通知组件**

当分身在群聊中代为私聊回复时，用户本人看到的通知。

```vue
<template>
  <div class="avatar-group-reply-notice" @click="$emit('click')">
    <div class="notice-icon">
      <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
        <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
      </svg>
    </div>
    <div class="notice-content">
      <span class="notice-text">你的分身在群聊「{{ groupName }}」中代你回复了{{ triggerName }}</span>
      <span class="notice-action">点击查看</span>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  groupName: string
  triggerName: string
}>()

defineEmits<{
  click: []
}>()
</script>

<style scoped>
.avatar-group-reply-notice {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  margin: 4px 0;
  background: rgba(59, 130, 246, 0.06);
  border: 1px solid rgba(59, 130, 246, 0.15);
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s;
}

.avatar-group-reply-notice:hover {
  background: rgba(59, 130, 246, 0.12);
}

.notice-icon {
  color: #3b82f6;
  flex-shrink: 0;
}

.notice-content {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.notice-text {
  font-size: 13px;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notice-action {
  font-size: 12px;
  color: #3b82f6;
  white-space: nowrap;
  flex-shrink: 0;
}

[data-theme="elegant-dark"] .avatar-group-reply-notice {
  background: rgba(59, 130, 246, 0.1);
  border-color: rgba(59, 130, 246, 0.25);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/avatar/AvatarGroupReplyNotice.vue
git commit -m "feat(avatar): add AvatarGroupReplyNotice component"
```

---

## 任务 6：AvatarTakeoverBanner 组件

**文件：**
- 创建：`src/components/avatar/AvatarTakeoverBanner.vue`

- [ ] **步骤 1：创建接管横幅组件**

```vue
<template>
  <div v-if="visible" class="avatar-takeover-banner">
    <div class="banner-content">
      <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor" class="banner-icon">
        <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/>
      </svg>
      <span class="banner-text">分身已暂停，{{ remainingText }}后恢复</span>
    </div>
    <div class="banner-actions">
      <button class="banner-btn resume" @click="$emit('resume')">立即恢复</button>
      <button class="banner-btn extend" @click="$emit('extend')">继续暂停</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'

const props = defineProps<{
  takeoverUntil: string | null
}>()

defineEmits<{
  resume: []
  extend: []
}>()

const now = ref(Date.now())
let timer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  timer = setInterval(() => { now.value = Date.now() }, 1000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

const visible = computed(() => {
  if (!props.takeoverUntil) return false
  return new Date(props.takeoverUntil).getTime() > now.value
})

const remainingSeconds = computed(() => {
  if (!props.takeoverUntil) return 0
  const diff = new Date(props.takeoverUntil).getTime() - now.value
  return Math.max(0, Math.floor(diff / 1000))
})

const remainingText = computed(() => {
  const mins = Math.floor(remainingSeconds.value / 60)
  const secs = remainingSeconds.value % 60
  if (mins > 0) return `${mins}分${secs}秒`
  return `${secs}秒`
})
</script>

<style scoped>
.avatar-takeover-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  background: #FFF8E1;
  border-bottom: 1px solid rgba(255, 152, 0, 0.2);
  font-size: 13px;
}

.banner-content {
  display: flex;
  align-items: center;
  gap: 8px;
}

.banner-icon {
  color: #FF9800;
  flex-shrink: 0;
}

.banner-text {
  color: var(--text-primary);
}

.banner-actions {
  display: flex;
  gap: 8px;
}

.banner-btn {
  padding: 4px 12px;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  border: none;
  transition: opacity 0.2s;
}

.banner-btn:hover {
  opacity: 0.85;
}

.banner-btn.resume {
  background: var(--primary-color);
  color: white;
}

.banner-btn.extend {
  background: transparent;
  color: #FF9800;
  border: 1px solid #FF9800;
}

[data-theme="elegant-dark"] .avatar-takeover-banner {
  background: rgba(255, 152, 0, 0.1);
  border-bottom-color: rgba(255, 152, 0, 0.3);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/avatar/AvatarTakeoverBanner.vue
git commit -m "feat(avatar): add AvatarTakeoverBanner component"
```

---

## 任务 7：AvatarSessionToggle 组件

**文件：**
- 创建：`src/components/avatar/AvatarSessionToggle.vue`

- [ ] **步骤 1：创建会话级分身开关**

嵌入聊天头部的开关组件。

```vue
<template>
  <div class="avatar-session-toggle" :class="{ active: isActive }">
    <button
      class="toggle-btn"
      :title="isActive ? '分身已开启，点击关闭' : '点击开启分身'"
      @click="handleToggle"
    >
      <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
        <path d="M12 2a2 2 0 011 .26A2 2 0 0114 2h4a2 2 0 012 2v4a2 2 0 01-.26 1A2 2 0 0120 10v4a2 2 0 01-2 2h-4a2 2 0 01-1-.26A2 2 0 0112 16H8a2 2 0 01-2-2v-4a2 2 0 01.26-1A2 2 0 016 8V4a2 2 0 012-2h4zm0 2H8v4h4V4zm6 0h-4v4h4V4zm-6 6H8v4h4v-4zm6 0h-4v4h4v-4z"/>
      </svg>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAvatar } from '../../composables/useAvatar'

const props = defineProps<{
  conversationId: number
}>()

const { config, sessions, toggleSession, isAvatarActive } = useAvatar()

const isActive = computed(() => {
  return config.value?.enabled && isAvatarActive(props.conversationId)
})

async function handleToggle() {
  if (!config.value?.enabled) {
    window.$QMessage.warning('请先在分身设置中开启分身功能')
    return
  }
  try {
    await toggleSession(props.conversationId, !isActive.value)
    window.$QMessage.success(isActive.value ? '分身已关闭' : '分身已开启')
  } catch {
    window.$QMessage.error('操作失败')
  }
}
</script>

<style scoped>
.avatar-session-toggle {
  display: flex;
  align-items: center;
}

.toggle-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 6px;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.toggle-btn:hover {
  background: var(--hover-color);
  color: var(--text-primary);
}

.avatar-session-toggle.active .toggle-btn {
  color: #3b82f6;
  background: rgba(59, 130, 246, 0.1);
}

.avatar-session-toggle.active .toggle-btn:hover {
  background: rgba(59, 130, 246, 0.2);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/avatar/AvatarSessionToggle.vue
git commit -m "feat(avatar): add AvatarSessionToggle component"
```

---

## 任务 8：设置子面板组件

**文件：**
- 创建：`src/components/avatar/AvatarBasicSettings.vue`
- 创建：`src/components/avatar/AvatarPersonaSettings.vue`
- 创建：`src/components/avatar/AvatarTriggerSettings.vue`
- 创建：`src/components/avatar/AvatarKnowledgeSettings.vue`
- 创建：`src/components/avatar/AvatarReplySettings.vue`

- [ ] **步骤 1：创建 AvatarBasicSettings**

遵循 `AIBaseSettings.vue` 的 v-model 模式。

```vue
<template>
  <div class="avatar-basic-settings">
    <div class="setting-item">
      <label class="toggle-label">
        <span>启用分身</span>
        <label class="switch">
          <input type="checkbox" :checked="modelValue.enabled" @change="update('enabled', ($event.target as HTMLInputElement).checked)" />
          <span class="slider round"></span>
        </label>
      </label>
      <span class="setting-hint">开启后，分身将在你设定的规则下代替你回复消息</span>
    </div>

    <div class="setting-item">
      <label>分身名称</label>
      <input :value="modelValue.name" @input="update('name', ($event.target as HTMLInputElement).value)" class="form-input" placeholder="我的分身" />
      <span class="setting-hint">其他人在私聊中看到的分身名称</span>
    </div>

    <div class="setting-item">
      <label>模型来源</label>
      <select :value="modelValue.useSystemConfig ? 'system' : 'custom'" @change="handleModelSourceChange" class="form-select">
        <option value="system">使用系统默认模型</option>
        <option value="custom">使用我的自定义配置</option>
      </select>
    </div>

    <div v-if="!modelValue.useSystemConfig" class="setting-item">
      <label>选择配置</label>
      <select :value="modelValue.modelConfigId || ''" @change="update('modelConfigId', Number(($event.target as HTMLSelectElement).value) || null)" class="form-select">
        <option value="">请选择...</option>
        <option v-for="cfg in modelConfigs" :key="cfg.id" :value="cfg.id">
          {{ cfg.config_name }} ({{ cfg.model_name }})
        </option>
      </select>
      <span v-if="modelConfigs.length === 0" class="setting-hint error">暂无配置，请先在"我的模型配置"中添加</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { AvatarConfig } from '../../types/avatar'
import type { UserAIConfig as AIConfig } from '../../types/ai'

const props = defineProps<{
  modelValue: AvatarConfig
  modelConfigs: AIConfig[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: AvatarConfig]
}>()

function update<K extends keyof AvatarConfig>(key: K, value: AvatarConfig[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

function handleModelSourceChange(event: Event) {
  const value = (event.target as HTMLSelectElement).value
  update('useSystemConfig', value === 'system')
  if (value === 'system') {
    update('modelConfigId', null)
  }
}
</script>

<style scoped>
.avatar-basic-settings { padding: 16px; }
.setting-item { margin-bottom: 16px; }
.setting-item > label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; }
.toggle-label { display: flex; align-items: center; justify-content: space-between; cursor: pointer; }
.setting-hint { display: block; margin-top: 4px; font-size: 12px; color: var(--text-secondary); }
.setting-hint.error { color: #F44336; }
.form-select, .form-input { width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 14px; box-sizing: border-box; }
.form-select:focus, .form-input:focus { outline: none; border-color: var(--primary-color); }
.switch { position: relative; display: inline-block; width: 50px; height: 24px; min-width: 50px; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background-color: #ccc; transition: 0.4s; border-radius: 24px; }
.slider:before { position: absolute; content: ''; height: 18px; width: 18px; left: 3px; bottom: 3px; background-color: white; transition: 0.4s; border-radius: 50%; }
input:checked + .slider { background-color: var(--primary-color); }
input:checked + .slider:before { transform: translateX(26px); }
.slider.round { border-radius: 24px; }
</style>
```

- [ ] **步骤 2：创建 AvatarPersonaSettings**

```vue
<template>
  <div class="avatar-persona-settings">
    <div class="learn-section">
      <div class="learn-header">
        <h4>风格学习</h4>
        <button
          v-if="learnStatus.status !== 'learning'"
          class="learn-btn"
          @click="handleLearn"
          :disabled="learnLoading"
        >
          {{ hasLearned ? '重新学习' : '开始学习' }}
        </button>
      </div>

      <div v-if="learnStatus.status === 'learning'" class="learn-progress">
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: `${learnStatus.progress}%` }"></div>
        </div>
        <span class="progress-text">正在分析 {{ learnStatus.messageCount }} 条历史消息... {{ learnStatus.progress }}%</span>
      </div>

      <div v-else-if="learnStatus.status === 'completed'" class="learn-result">
        <div class="result-label">学习到的人设风格：</div>
        <div class="result-content">{{ learnedPersona || '暂无' }}</div>
      </div>

      <div v-else-if="learnStatus.status === 'failed'" class="learn-error">
        <span>学习失败：{{ learnStatus.error }}</span>
        <button class="retry-btn" @click="handleLearn">重试</button>
      </div>

      <div v-else class="learn-idle">
        <span class="setting-hint">分身将从你的历史消息中学习说话风格和表达习惯</span>
      </div>
    </div>

    <div class="setting-item">
      <label>补充提示词</label>
      <textarea
        :value="modelValue.customPersonaAddon"
        @input="update('customPersonaAddon', ($event.target as HTMLTextAreaElement).value)"
        class="form-textarea"
        rows="4"
        placeholder="补充描述你的说话习惯、常用表达、专业领域等..."
      ></textarea>
      <span class="setting-hint">在自动学习的基础上，手动补充分身应该知道的关于你的信息</span>
    </div>

    <div class="setting-item">
      <label>风格预览</label>
      <div class="preview-area">
        <input
          v-model="previewInput"
          class="form-input"
          placeholder="输入一段话，预览分身会怎么回复..."
          @keyup.enter="handlePreview"
        />
        <button class="preview-btn" @click="handlePreview" :disabled="previewLoading">
          {{ previewLoading ? '生成中...' : '预览' }}
        </button>
      </div>
      <div v-if="previewResult" class="preview-result">
        <div class="result-label">分身回复：</div>
        <div class="result-content">{{ previewResult }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useAvatarPersona } from '../../composables/useAvatarPersona'
import type { AvatarConfig } from '../../types/avatar'

const props = defineProps<{
  modelValue: AvatarConfig
}>()

const emit = defineEmits<{
  'update:modelValue': [value: AvatarConfig]
}>()

const {
  learnStatus,
  learnedPersona,
  loading: learnLoading,
  triggerLearn,
  fetchLearnStatus,
  fetchLearnedPersona,
  previewReply,
  stopPolling
} = useAvatarPersona()

const previewInput = ref('')
const previewResult = ref('')
const previewLoading = ref(false)

const hasLearned = computed(() => learnStatus.value.status === 'completed')

onMounted(() => {
  fetchLearnStatus()
  if (hasLearned.value) {
    fetchLearnedPersona()
  }
})

onUnmounted(() => {
  stopPolling()
})

function update<K extends keyof AvatarConfig>(key: K, value: AvatarConfig[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

async function handleLearn() {
  try {
    await triggerLearn()
    window.$QMessage.success('风格学习已开始')
  } catch {
    window.$QMessage.error('触发学习失败')
  }
}

async function handlePreview() {
  if (!previewInput.value.trim()) return
  previewLoading.value = true
  try {
    previewResult.value = await previewReply(previewInput.value.trim())
  } catch {
    window.$QMessage.error('预览失败')
  } finally {
    previewLoading.value = false
  }
}
</script>

<style scoped>
.avatar-persona-settings { padding: 16px; }
.learn-section { margin-bottom: 20px; padding: 16px; background: var(--bg-color); border-radius: 8px; border: 1px solid var(--border-color); }
.learn-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.learn-header h4 { margin: 0; font-size: 14px; }
.learn-btn { padding: 6px 14px; background: var(--primary-color); color: white; border: none; border-radius: 6px; cursor: pointer; font-size: 13px; }
.learn-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.progress-bar { height: 6px; background: var(--border-color); border-radius: 3px; overflow: hidden; margin-bottom: 8px; }
.progress-fill { height: 100%; background: var(--primary-color); border-radius: 3px; transition: width 0.3s; }
.progress-text { font-size: 12px; color: var(--text-secondary); }
.result-label { font-size: 12px; color: var(--text-secondary); margin-bottom: 6px; }
.result-content { font-size: 13px; color: var(--text-primary); line-height: 1.6; padding: 10px; background: var(--card-bg); border-radius: 6px; border: 1px solid var(--border-color); white-space: pre-wrap; }
.learn-error { color: #F44336; font-size: 13px; display: flex; align-items: center; gap: 8px; }
.retry-btn { padding: 4px 10px; background: transparent; border: 1px solid #F44336; color: #F44336; border-radius: 4px; cursor: pointer; font-size: 12px; }
.setting-item { margin-bottom: 16px; }
.setting-item > label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; }
.setting-hint { display: block; margin-top: 4px; font-size: 12px; color: var(--text-secondary); }
.form-textarea, .form-input { width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 14px; box-sizing: border-box; font-family: inherit; resize: vertical; }
.form-textarea:focus, .form-input:focus { outline: none; border-color: var(--primary-color); }
.preview-area { display: flex; gap: 8px; }
.preview-area .form-input { flex: 1; }
.preview-btn { padding: 8px 16px; background: var(--primary-color); color: white; border: none; border-radius: 6px; cursor: pointer; font-size: 13px; white-space: nowrap; }
.preview-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.preview-result { margin-top: 12px; }
</style>
```

- [ ] **步骤 3：创建 AvatarTriggerSettings**

遵循 `AITriggerSettings.vue` 的模式。

```vue
<template>
  <div class="avatar-trigger-settings">
    <div class="setting-item">
      <label>触发模式</label>
      <select :value="modelValue.triggerRules.mode" @change="updateTrigger('mode', ($event.target as HTMLSelectElement).value)" class="form-select">
        <option value="mention">被 @ 时回复</option>
        <option value="offline">离线时自动回复</option>
        <option value="keyword">关键词触发</option>
        <option value="all">所有消息</option>
        <option value="custom">自定义规则</option>
      </select>
      <span class="setting-hint">
        {{ triggerModeHint }}
      </span>
    </div>

    <div v-if="modelValue.triggerRules.mode === 'keyword' || modelValue.triggerRules.mode === 'custom'" class="setting-item">
      <label>触发关键词</label>
      <div class="keyword-input-wrapper">
        <input
          :value="keywordInput"
          @input="keywordInput = ($event.target as HTMLInputElement).value"
          @keydown.enter.prevent="addKeyword"
          class="form-input"
          placeholder="输入关键词后按回车"
        />
        <div class="keyword-tags">
          <span v-for="(kw, i) in modelValue.triggerRules.keywords" :key="i" class="keyword-tag">
            {{ kw }}
            <button class="remove-tag" @click="removeKeyword(i)">x</button>
          </span>
        </div>
      </div>
    </div>

    <div class="setting-item">
      <label>接管冷却期</label>
      <select :value="modelValue.takeoverCooldown" @change="update('takeoverCooldown', Number(($event.target as HTMLSelectElement).value))" class="form-select">
        <option :value="5">5 分钟</option>
        <option :value="10">10 分钟</option>
        <option :value="30">30 分钟</option>
        <option :value="60">1 小时</option>
      </select>
      <span class="setting-hint">你发消息后，分身暂停回复的时间</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { AvatarConfig } from '../../types/avatar'

const props = defineProps<{
  modelValue: AvatarConfig
}>()

const emit = defineEmits<{
  'update:modelValue': [value: AvatarConfig]
}>()

const keywordInput = ref('')

const triggerModeHint = computed(() => {
  const hints: Record<string, string> = {
    mention: '当有人在私聊中发消息，或在群聊中 @你时，分身会回复',
    offline: '当你离线时，分身自动回复私聊消息',
    keyword: '仅当消息包含指定关键词时，分身才回复',
    all: '分身会回复所有消息（请谨慎使用）',
    custom: '自定义触发规则，结合关键词和时间段'
  }
  return hints[props.modelValue.triggerRules.mode] || ''
})

function update<K extends keyof AvatarConfig>(key: K, value: AvatarConfig[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

function updateTrigger(key: string, value: any) {
  emit('update:modelValue', {
    ...props.modelValue,
    triggerRules: { ...props.modelValue.triggerRules, [key]: value }
  })
}

function addKeyword() {
  const kw = keywordInput.value.trim()
  if (kw && !props.modelValue.triggerRules.keywords.includes(kw)) {
    emit('update:modelValue', {
      ...props.modelValue,
      triggerRules: {
        ...props.modelValue.triggerRules,
        keywords: [...props.modelValue.triggerRules.keywords, kw]
      }
    })
  }
  keywordInput.value = ''
}

function removeKeyword(index: number) {
  const keywords = [...props.modelValue.triggerRules.keywords]
  keywords.splice(index, 1)
  emit('update:modelValue', {
    ...props.modelValue,
    triggerRules: { ...props.modelValue.triggerRules, keywords }
  })
}
</script>

<style scoped>
.avatar-trigger-settings { padding: 16px; }
.setting-item { margin-bottom: 16px; }
.setting-item > label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; }
.setting-hint { display: block; margin-top: 4px; font-size: 12px; color: var(--text-secondary); }
.form-select, .form-input { width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 14px; box-sizing: border-box; }
.form-select:focus, .form-input:focus { outline: none; border-color: var(--primary-color); }
.keyword-input-wrapper { display: flex; flex-direction: column; gap: 8px; }
.keyword-tags { display: flex; flex-wrap: wrap; gap: 6px; }
.keyword-tag { display: inline-flex; align-items: center; gap: 4px; padding: 4px 10px; background: var(--primary-color-alpha, rgba(99, 102, 241, 0.1)); color: var(--primary-color); border-radius: 12px; font-size: 13px; }
.remove-tag { background: none; border: none; color: var(--primary-color); cursor: pointer; font-size: 14px; padding: 0; width: 16px; height: 16px; display: flex; align-items: center; justify-content: center; }
</style>
```

- [ ] **步骤 4：创建 AvatarKnowledgeSettings**

```vue
<template>
  <div class="avatar-knowledge-settings">
    <div class="setting-item">
      <label class="toggle-label">
        <span>当前会话历史</span>
        <label class="switch">
          <input type="checkbox" :checked="modelValue.knowledgeScope.conversationHistory" @change="updateScope('conversationHistory', ($event.target as HTMLInputElement).checked)" />
          <span class="slider round"></span>
        </label>
      </label>
      <span class="setting-hint">分身可以参考当前会话中的历史消息来回复</span>
    </div>

    <div class="setting-item">
      <label class="toggle-label">
        <span>知识库文档</span>
        <label class="switch">
          <input type="checkbox" :checked="modelValue.knowledgeScope.knowledgeDocs" @change="updateScope('knowledgeDocs', ($event.target as HTMLInputElement).checked)" />
          <span class="slider round"></span>
        </label>
      </label>
      <span class="setting-hint">分身可以访问你上传的知识库文档</span>
    </div>

    <div class="setting-item">
      <label class="toggle-label">
        <span>用户笔记</span>
        <label class="switch">
          <input type="checkbox" :checked="modelValue.knowledgeScope.notes" @change="updateScope('notes', ($event.target as HTMLInputElement).checked)" />
          <span class="slider round"></span>
        </label>
      </label>
      <span class="setting-hint">分身可以读取你的笔记内容</span>
    </div>

    <div class="setting-item">
      <label class="toggle-label">
        <span>用户任务</span>
        <label class="switch">
          <input type="checkbox" :checked="modelValue.knowledgeScope.tasks" @change="updateScope('tasks', ($event.target as HTMLInputElement).checked)" />
          <span class="slider round"></span>
        </label>
      </label>
      <span class="setting-hint">分身可以读取你的任务列表</span>
    </div>

    <div class="privacy-notice">
      <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
        <path d="M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4zm0 10.99h7c-.53 4.12-3.28 7.79-7 8.94V12H5V6.3l7-3.11v8.8z"/>
      </svg>
      <span>分身仅在你允许的范围内读取信息，不会访问未授权的数据</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { AvatarConfig, AvatarKnowledgeScope } from '../../types/avatar'

const props = defineProps<{
  modelValue: AvatarConfig
}>()

const emit = defineEmits<{
  'update:modelValue': [value: AvatarConfig]
}>()

function updateScope(key: keyof AvatarKnowledgeScope, value: boolean) {
  emit('update:modelValue', {
    ...props.modelValue,
    knowledgeScope: { ...props.modelValue.knowledgeScope, [key]: value }
  })
}
</script>

<style scoped>
.avatar-knowledge-settings { padding: 16px; }
.setting-item { margin-bottom: 16px; }
.toggle-label { display: flex; align-items: center; justify-content: space-between; cursor: pointer; }
.setting-hint { display: block; margin-top: 4px; font-size: 12px; color: var(--text-secondary); }
.switch { position: relative; display: inline-block; width: 50px; height: 24px; min-width: 50px; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background-color: #ccc; transition: 0.4s; border-radius: 24px; }
.slider:before { position: absolute; content: ''; height: 18px; width: 18px; left: 3px; bottom: 3px; background-color: white; transition: 0.4s; border-radius: 50%; }
input:checked + .slider { background-color: var(--primary-color); }
input:checked + .slider:before { transform: translateX(26px); }
.slider.round { border-radius: 24px; }
.privacy-notice { display: flex; align-items: center; gap: 6px; padding: 10px 12px; background: rgba(59, 130, 246, 0.06); border-radius: 6px; font-size: 12px; color: var(--text-secondary); margin-top: 8px; }
</style>
```

- [ ] **步骤 5：创建 AvatarReplySettings**

```vue
<template>
  <div class="avatar-reply-settings">
    <div class="setting-item">
      <label>回复长度</label>
      <select :value="modelValue.replyStrategy.maxReplyLength" @change="updateStrategy('maxReplyLength', ($event.target as HTMLSelectElement).value)" class="form-select">
        <option value="short">简短（1-2 句）</option>
        <option value="medium">适中（3-5 句）</option>
        <option value="long">详细（6 句以上）</option>
      </select>
    </div>

    <div class="setting-item">
      <label>回复延迟</label>
      <select :value="modelValue.replyStrategy.replyDelay" @change="updateStrategy('replyDelay', Number(($event.target as HTMLSelectElement).value))" class="form-select">
        <option :value="0">无延迟</option>
        <option :value="3">3 秒</option>
        <option :value="5">5 秒</option>
        <option :value="10">10 秒</option>
      </select>
      <span class="setting-hint">模拟真人思考时间，避免回复过快显得不自然</span>
    </div>

    <div class="setting-item">
      <label>置信度阈值</label>
      <div class="threshold-slider">
        <input type="range" :value="modelValue.replyStrategy.confidenceThreshold" @input="updateStrategy('confidenceThreshold', Number(($event.target as HTMLInputElement).value))" min="0" max="1" step="0.1" class="slider-input" />
        <span class="threshold-value">{{ (modelValue.replyStrategy.confidenceThreshold * 100).toFixed(0) }}%</span>
      </div>
      <span class="setting-hint">低于此阈值时分身不会回复，而是通知你亲自回复</span>
    </div>

    <div class="setting-item">
      <label>AI 标记样式</label>
      <select :value="modelValue.replyStrategy.disclaimerStyle" @change="updateStrategy('disclaimerStyle', ($event.target as HTMLSelectElement).value)" class="form-select">
        <option value="badge">徽章标记</option>
        <option value="footer">底部标注</option>
        <option value="both">两者都有</option>
      </select>
      <span class="setting-hint">分身回复消息中"AI 代回复"标记的展示方式</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { AvatarConfig, AvatarReplyStrategy } from '../../types/avatar'

const props = defineProps<{
  modelValue: AvatarConfig
}>()

const emit = defineEmits<{
  'update:modelValue': [value: AvatarConfig]
}>()

function updateStrategy<K extends keyof AvatarReplyStrategy>(key: K, value: AvatarReplyStrategy[K]) {
  emit('update:modelValue', {
    ...props.modelValue,
    replyStrategy: { ...props.modelValue.replyStrategy, [key]: value }
  })
}
</script>

<style scoped>
.avatar-reply-settings { padding: 16px; }
.setting-item { margin-bottom: 16px; }
.setting-item > label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; }
.setting-hint { display: block; margin-top: 4px; font-size: 12px; color: var(--text-secondary); }
.form-select { width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 14px; box-sizing: border-box; }
.form-select:focus { outline: none; border-color: var(--primary-color); }
.threshold-slider { display: flex; align-items: center; gap: 12px; }
.slider-input { flex: 1; }
.threshold-value { font-size: 14px; font-weight: 500; color: var(--primary-color); min-width: 40px; }
</style>
```

- [ ] **步骤 6：Commit**

```bash
git add src/components/avatar/AvatarBasicSettings.vue src/components/avatar/AvatarPersonaSettings.vue src/components/avatar/AvatarTriggerSettings.vue src/components/avatar/AvatarKnowledgeSettings.vue src/components/avatar/AvatarReplySettings.vue
git commit -m "feat(avatar): add avatar settings sub-panels"
```

---

## 任务 9：AvatarSettingsPanel 主面板

**文件：**
- 创建：`src/components/avatar/AvatarSettingsPanel.vue`

- [ ] **步骤 1：创建主设置面板**

遵循 `GroupAIPanel.vue` 的 Tab 容器模式，使用项目共享组件替换 alert/confirm。

```vue
<template>
  <div class="avatar-settings-panel">
    <div v-if="loading && !config" class="loading-state">
      <LoadingSpinner />
    </div>

    <div v-else-if="!config" class="empty-state">
      <EmptyState
        icon="fas fa-user-astronaut"
        title="还没有分身"
        description="创建你的 AI 分身，在你不在时代替你回复消息"
      />
      <button class="create-btn" @click="handleCreate">
        <i class="fas fa-plus"></i> 创建分身
      </button>
    </div>

    <template v-else>
      <div class="tab-bar">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          :class="['tab-btn', { active: activeTab === tab.key }]"
          @click="activeTab = tab.key"
        >
          <i :class="tab.icon"></i>
          <span>{{ tab.label }}</span>
        </button>
      </div>

      <div class="tab-content">
        <AvatarBasicSettings
          v-if="activeTab === 'basic'"
          v-model="config"
          :model-configs="modelConfigs"
        />
        <AvatarPersonaSettings
          v-if="activeTab === 'persona'"
          v-model="config"
        />
        <AvatarTriggerSettings
          v-if="activeTab === 'trigger'"
          v-model="config"
        />
        <AvatarKnowledgeSettings
          v-if="activeTab === 'knowledge'"
          v-model="config"
        />
        <AvatarReplySettings
          v-if="activeTab === 'reply'"
          v-model="config"
        />
      </div>

      <div class="tab-footer">
        <button class="btn btn-danger" @click="handleDelete" v-if="config">
          <i class="fas fa-trash"></i> 删除分身
        </button>
        <button class="btn btn-primary" @click="handleSave" :disabled="saving">
          {{ saving ? '保存中...' : '保存设置' }}
        </button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAvatar } from '../../composables/useAvatar'
import { useModelConfigs } from '../../composables/useModelConfigs'
import LoadingSpinner from '../shared/LoadingSpinner.vue'
import EmptyState from '../shared/EmptyState.vue'
import AvatarBasicSettings from './AvatarBasicSettings.vue'
import AvatarPersonaSettings from './AvatarPersonaSettings.vue'
import AvatarTriggerSettings from './AvatarTriggerSettings.vue'
import AvatarKnowledgeSettings from './AvatarKnowledgeSettings.vue'
import AvatarReplySettings from './AvatarReplySettings.vue'
import { DEFAULT_AVATAR_CONFIG } from '../../types/avatar'
import type { AvatarConfig } from '../../types/avatar'

const {
  config,
  loading,
  fetchConfig,
  createConfig,
  updateConfig,
  deleteConfig
} = useAvatar()

const { configs: modelConfigs, fetchConfigs } = useModelConfigs()

const activeTab = ref('basic')
const saving = ref(false)

const tabs = [
  { key: 'basic', label: '基础设置', icon: 'fas fa-cog' },
  { key: 'persona', label: '人设风格', icon: 'fas fa-palette' },
  { key: 'trigger', label: '触发规则', icon: 'fas fa-bolt' },
  { key: 'knowledge', label: '知识范围', icon: 'fas fa-book' },
  { key: 'reply', label: '回复策略', icon: 'fas fa-sliders-h' }
]

onMounted(async () => {
  await Promise.all([fetchConfig(), fetchConfigs()])
})

async function handleCreate() {
  try {
    await createConfig(DEFAULT_AVATAR_CONFIG)
    window.$QMessage.success('分身创建成功')
  } catch {
    window.$QMessage.error('创建失败')
  }
}

async function handleSave() {
  if (!config.value) return
  saving.value = true
  try {
    await updateConfig(config.value)
    window.$QMessage.success('设置已保存')
  } catch {
    window.$QMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

async function handleDelete() {
  try {
    await window.$QMessageBox.confirm('确定删除分身吗？删除后所有会话的分身都将关闭。', '删除分身')
    await deleteConfig()
    window.$QMessage.success('分身已删除')
  } catch {
    // 用户取消
  }
}
</script>

<style scoped>
.avatar-settings-panel {
  background: var(--card-bg);
  border-radius: 8px;
  overflow: hidden;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px;
}

.empty-state {
  text-align: center;
  padding: 40px 20px;
}

.create-btn {
  margin-top: 16px;
  padding: 10px 24px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.create-btn:hover {
  opacity: 0.9;
}

.tab-bar {
  display: flex;
  border-bottom: 1px solid var(--border-color);
}

.tab-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 12px 8px;
  border: none;
  background: none;
  cursor: pointer;
  font-size: 13px;
  color: var(--text-secondary);
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
}

.tab-btn:hover {
  color: var(--text-color);
  background: var(--hover-color);
}

.tab-btn.active {
  color: var(--primary-color);
  border-bottom-color: var(--primary-color);
  background: var(--primary-color-alpha, rgba(99, 102, 241, 0.05));
}

.tab-content {
  flex: 1;
  overflow-y: auto;
}

.tab-footer {
  padding: 12px 20px;
  border-top: 1px solid var(--border-color);
  display: flex;
  justify-content: space-between;
}

.btn {
  padding: 8px 20px;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  border: none;
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.btn-primary {
  background: var(--primary-color);
  color: white;
}

.btn-primary:hover {
  opacity: 0.9;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-danger {
  background: transparent;
  color: #F44336;
  border: 1px solid #F44336;
}

.btn-danger:hover {
  background: #FFEBEE;
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/avatar/AvatarSettingsPanel.vue
git commit -m "feat(avatar): add AvatarSettingsPanel main component"
```

---

## 任务 10：集成到 Main.vue 和 ChatHeader

**文件：**
- 修改：`src/views/Main.vue`
- 修改：`src/components/chat/ChatHeader.vue`
- 修改：`src/components/chat/ChatWindow.vue`

- [ ] **步骤 1：在 Main.vue 中注册分身应用**

在 Main.vue 的 import 区域添加：

```typescript
import AvatarSettingsPanel from '../components/avatar/AvatarSettingsPanel.vue'
```

在模板的应用面板条件渲染区域添加（与 AIAssistantApp 同级）：

```vue
<div v-else-if="activeOption === 'apps' && selectedAppId === 'avatar'" class="right-content">
  <AvatarSettingsPanel @back="backToAppList" @toggleSidebar="toggleSidebar" />
</div>
```

在应用分类列表中添加：

```typescript
{ id: 'avatar', name: 'AI 分身', icon: 'fas fa-user-astronaut' }
```

- [ ] **步骤 2：在 ChatHeader.vue 中添加 AvatarSessionToggle**

在 ChatHeader 的 `header-info` 区域内，添加 AvatarSessionToggle：

```vue
<AvatarSessionToggle
  v-if="currentUser"
  :conversation-id="conversation.id"
/>
```

需要 import：

```typescript
import AvatarSessionToggle from '../avatar/AvatarSessionToggle.vue'
```

- [ ] **步骤 3：在 ChatWindow.vue 中添加 AvatarTakeoverBanner**

在 ChatWindow 的聊天头部下方，添加接管横幅：

```vue
<AvatarTakeoverBanner
  v-if="avatarTakeoverUntil"
  :takeover-until="avatarTakeoverUntil"
  @resume="handleAvatarResume"
  @extend="handleAvatarExtend"
/>
```

需要添加 props 和方法：

```typescript
import AvatarTakeoverBanner from '../avatar/AvatarTakeoverBanner.vue'
import { useAvatar } from '../../composables/useAvatar'

const { isAvatarActive, takeoverSession, getSession } = useAvatar()

const avatarTakeoverUntil = computed(() => {
  if (!props.conversation) return null
  const session = getSession(props.conversation.id)
  return session?.takeoverUntil || null
})

async function handleAvatarResume() {
  if (!props.conversation) return
  await takeoverSession(props.conversation.id)
}

async function handleAvatarExtend() {
  // 延长接管 10 分钟
  if (!props.conversation) return
  await takeoverSession(props.conversation.id)
}
```

- [ ] **步骤 4：Commit**

```bash
git add src/views/Main.vue src/components/chat/ChatHeader.vue src/components/chat/ChatWindow.vue
git commit -m "feat(avatar): integrate avatar components into chat and main views"
```

---

## 任务 11：消息展示集成

**文件：**
- 修改：`src/components/chat/MessageItem.vue`（或消息气泡组件）

- [ ] **步骤 1：在消息组件中添加分身标记**

找到消息气泡组件中渲染消息内容的位置，添加分身标记检测：

```vue
<AvatarReplyBadge
  v-if="message.is_avatar_reply"
  :style="avatarDisclaimerStyle"
/>
```

需要 import：

```typescript
import AvatarReplyBadge from '../avatar/AvatarReplyBadge.vue'
```

当 `message.is_avatar_reply` 为 true 时，给消息容器添加 `avatar-reply` class，用于区分背景色：

```css
.message.avatar-reply {
  background: rgba(59, 130, 246, 0.04);
}
```

- [ ] **步骤 2：在 Main.vue 的 handleNewMessage 中处理分身通知**

当收到 WebSocket 消息且 `is_avatar_reply` 为 true 时，如果是群聊触发的私聊回复，显示 AvatarGroupReplyNotice。

在 `handleNewMessage` 函数中，消息处理逻辑内添加：

```typescript
if (msg.is_avatar_reply && msg.avatar_group_name) {
  // 这是群聊触发的分身私聊回复，给分身用户显示通知
  // 通知内容：你的分身在群聊 XXX 中代你回复了 YYY
}
```

- [ ] **步骤 3：Commit**

```bash
git add src/components/chat/MessageItem.vue src/views/Main.vue
git commit -m "feat(avatar): add avatar reply badge and group reply notice in messages"
```

---

## 任务 12：UserDetailPanel 入口

**文件：**
- 修改：`src/components/user/UserDetailPanel.vue`

- [ ] **步骤 1：在用户详情面板添加分身设置入口**

在 UserDetailPanel 的操作按钮区域，添加"分身设置"按钮：

```vue
<button class="action-btn" @click="$emit('open-avatar-settings')">
  <i class="fas fa-user-astronaut"></i>
  分身设置
</button>
```

添加 emit：

```typescript
defineEmits([...existingEmits, 'open-avatar-settings'])
```

- [ ] **步骤 2：在 Main.vue 中处理 open-avatar-settings 事件**

在 UserDetailPanel 的使用处添加事件监听：

```vue
<UserDetailPanel
  ...
  @open-avatar-settings="openAvatarSettings"
/>
```

添加方法：

```typescript
function openAvatarSettings() {
  selectedAppId.value = 'avatar'
  activeOption.value = 'apps'
}
```

- [ ] **步骤 3：Commit**

```bash
git add src/components/user/UserDetailPanel.vue src/views/Main.vue
git commit -m "feat(avatar): add avatar settings entry in user detail panel"
```

---

## 任务 13：验证与 lint

- [ ] **步骤 1：运行 TypeScript 类型检查**

```bash
cd /Users/gracegaoya/work/project/qim/qim-client && npx vue-tsc --noEmit
```

预期：无类型错误。如果有错误，逐一修复。

- [ ] **步骤 2：运行 ESLint**

```bash
cd /Users/gracegaoya/work/project/qim/qim-client && npx eslint src/types/avatar.ts src/api/avatar.ts src/composables/useAvatar.ts src/composables/useAvatarPersona.ts src/components/avatar/
```

预期：无 lint 错误。如果有，逐一修复。

- [ ] **步骤 3：运行开发服务器验证页面渲染**

```bash
cd /Users/gracegaoya/work/project/qim/qim-client && npm run dev
```

验证：
1. 应用列表中出现"AI 分身"入口
2. 点击进入分身设置面板，各 Tab 正常切换
3. 聊天头部出现分身开关按钮
4. 消息中 is_avatar_reply 标记正常显示

- [ ] **步骤 4：Commit 最终修复**

```bash
git add -A
git commit -m "fix(avatar): fix type and lint errors from integration"
```

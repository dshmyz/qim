<template>
  <div class="group-ai-panel">
    <!-- 加载状态 -->
    <div v-if="loading" class="loading-overlay">
      <div class="loading-spinner"></div>
    </div>
    
    <!-- 审批状态提示 -->
    <div v-if="approvalStatus === 'pending'" class="approval-notice pending">
      <i class="icon-clock"></i>
      <span>AI助手申请已提交，等待系统管理员审批</span>
    </div>
    <div v-if="approvalStatus === 'rejected'" class="approval-notice rejected">
      <i class="icon-warning"></i>
      <span>AI助手申请被拒绝：{{ rejectReason || '未提供原因' }}</span>
    </div>

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
      <AIBaseSettings
        v-if="activeTab === 'base'"
        v-model="settings"
      />
      <AIPersonaSettings
        v-if="activeTab === 'persona'"
        v-model="settings"
      />
      <AITriggerSettings
        v-if="activeTab === 'trigger'"
        v-model="settings"
      />
      <AIKnowledgeSettings
        v-if="activeTab === 'knowledge'"
        :group-id="groupId"
        :server-url="serverUrl"
        :documents="documents"
        @add="handleAddDocuments"
        @remove="handleRemoveDocument"
        @retry="handleRetryDocument"
        @refresh="handleRefreshDocuments"
      />
      <!-- 群记忆管理 -->
      <div v-if="activeTab === 'memory'" class="memory-tab">
        <div class="memory-toolbar">
          <input
            v-model="searchQuery"
            class="memory-search-input"
            placeholder="搜索群记忆..."
            @keyup.enter="searchMemories"
          />
          <button class="btn btn-sm" @click="searchMemories">搜索</button>
          <button class="btn btn-sm" @click="loadMemories">刷新</button>
          <button class="btn btn-sm btn-danger" @click="clearMemories">清空全部</button>
        </div>
        <div v-if="memories.length === 0" class="memory-empty">
          暂无群记忆。群助手在群聊中遇到值得记的内容（群决定、约定、项目关键信息等）会自动沉淀到这里。
        </div>
        <div v-else class="memory-list">
          <div v-for="m in memories" :key="m.doc_id" class="memory-item">
            <div class="memory-content">{{ m.content }}</div>
            <div class="memory-meta">
              <span v-if="m.metadata?.remembered_at" class="memory-time">{{ formatMemoryTime(m.metadata.remembered_at) }}</span>
              <button class="btn-link" @click="deleteMemory(m.doc_id)">删除</button>
            </div>
          </div>
        </div>
      </div>
      <!-- 群知识图谱 -->
      <AIGraph
        v-if="activeTab === 'graph'"
        :group-id="groupId"
        :server-url="serverUrl"
      />
    </div>

    <div class="tab-footer">
      <button class="btn btn-primary" @click="saveSettings" :disabled="saving">
        {{ saving ? '保存中...' : '保存设置' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import AIBaseSettings from './ai-settings/AIBaseSettings.vue'
import AIPersonaSettings from './ai-settings/AIPersonaSettings.vue'
import AITriggerSettings from './ai-settings/AITriggerSettings.vue'
import AIKnowledgeSettings from './ai-settings/AIKnowledgeSettings.vue'
import AIGraph from './ai-settings/AIGraph.vue'
import type { GroupAISettings, GroupDocument, GroupMemory } from '../../types/ai'
import { request } from '../../composables/useRequest'

interface Props {
  groupId: number
  serverUrl: string
  aiEnabled?: boolean
  aiAssistantName?: string
  aiReplyMode?: string
  aiPersonality?: string
  aiCustomPrompt?: string
  aiLanguage?: string
  aiMaxLength?: string
  aiMentionReplyMode?: string
  aiAntiSpamInterval?: number
  aiTriggerKeywords?: string[]
  aiLearnEnabled?: boolean
  aiExtractTodos?: boolean
  approvalStatus?: 'pending' | 'approved' | 'rejected'
  rejectReason?: string
}

const props = withDefaults(defineProps<Props>(), {
  aiEnabled: false,
  aiAssistantName: 'AI助手',
  aiReplyMode: 'mention_only',
  aiPersonality: 'professional',
  aiCustomPrompt: '',
  aiLanguage: 'auto',
  aiMaxLength: 'medium',
  aiMentionReplyMode: 'mention',
  aiAntiSpamInterval: 0,
  aiTriggerKeywords: () => [],
  aiLearnEnabled: true,
  aiExtractTodos: false,
  approvalStatus: 'approved',
  rejectReason: ''
})

const emit = defineEmits<{
  (e: 'update', settings: GroupAISettings): void
}>()

const activeTab = ref('base')

const tabs = [
  { key: 'base', label: '基础设置', icon: 'fas fa-cog' },
  { key: 'persona', label: '人设风格', icon: 'fas fa-palette' },
  { key: 'trigger', label: '触发规则', icon: 'fas fa-bolt' },
  { key: 'knowledge', label: '知识库', icon: 'fas fa-book' },
  { key: 'memory', label: '群记忆', icon: 'fas fa-brain' },
  { key: 'graph', label: '知识图谱', icon: 'fas fa-project-diagram' }
]

const settings = ref<GroupAISettings>({
  aiEnabled: props.aiEnabled,
  aiAssistantName: props.aiAssistantName,
  aiReplyMode: props.aiReplyMode,
  aiPersonality: props.aiPersonality,
  aiCustomPrompt: props.aiCustomPrompt,
  aiLanguage: props.aiLanguage,
  aiMaxLength: props.aiMaxLength,
  aiMentionReplyMode: props.aiMentionReplyMode,
  aiAntiSpamInterval: props.aiAntiSpamInterval,
  aiTriggerKeywords: [...props.aiTriggerKeywords],
  aiLearnEnabled: props.aiLearnEnabled,
  aiExtractTodos: props.aiExtractTodos
})

const approvalStatus = ref<'pending' | 'approved' | 'rejected'>(props.approvalStatus)
const rejectReason = ref(props.rejectReason)
const loading = ref(false)

async function loadAISettings() {
  loading.value = true
  try {
    const response = await request(`/api/v1/groups/${props.groupId}/ai-settings`)
    if (response.code === 0 && response.data) {
      const data = response.data
      settings.value = {
        aiEnabled: data.ai_enabled !== undefined ? data.ai_enabled : props.aiEnabled,
        aiAssistantName: data.ai_assistant_name || props.aiAssistantName,
        aiReplyMode: data.ai_reply_mode || props.aiReplyMode,
        aiPersonality: data.ai_personality || props.aiPersonality,
        aiCustomPrompt: data.ai_custom_prompt || props.aiCustomPrompt,
        aiLanguage: data.ai_language || props.aiLanguage,
        aiMaxLength: data.ai_max_length || props.aiMaxLength,
        aiMentionReplyMode: data.ai_mention_reply_mode || props.aiMentionReplyMode,
        aiAntiSpamInterval: data.ai_anti_spam_interval !== undefined ? data.ai_anti_spam_interval : props.aiAntiSpamInterval,
        aiTriggerKeywords: data.ai_trigger_keywords ? data.ai_trigger_keywords.split(',').filter((k: string) => k.trim()) : [...props.aiTriggerKeywords],
        aiLearnEnabled: data.ai_learn_enabled !== undefined ? data.ai_learn_enabled : props.aiLearnEnabled,
        aiExtractTodos: data.ai_extract_todos !== undefined ? data.ai_extract_todos : props.aiExtractTodos
      }
      if (data.approval_status) {
        approvalStatus.value = data.approval_status as 'pending' | 'approved' | 'rejected'
      }
      if (data.reject_reason) {
        rejectReason.value = data.reject_reason
      }
    }
  } catch (error) {
    console.error('加载AI设置失败', error)
  } finally {
    loading.value = false
  }
}

watch(() => [props.aiEnabled, props.aiAssistantName, props.aiReplyMode, props.aiPersonality, props.aiCustomPrompt, props.aiLanguage, props.aiMaxLength, props.aiMentionReplyMode, props.aiAntiSpamInterval, props.aiTriggerKeywords, props.aiLearnEnabled, props.aiExtractTodos, props.approvalStatus, props.rejectReason], (newVal, oldVal) => {
  if (!oldVal) return
  settings.value = {
    aiEnabled: props.aiEnabled,
    aiAssistantName: props.aiAssistantName,
    aiReplyMode: props.aiReplyMode,
    aiPersonality: props.aiPersonality,
    aiCustomPrompt: props.aiCustomPrompt,
    aiLanguage: props.aiLanguage,
    aiMaxLength: props.aiMaxLength,
    aiMentionReplyMode: props.aiMentionReplyMode,
    aiAntiSpamInterval: props.aiAntiSpamInterval,
    aiTriggerKeywords: [...props.aiTriggerKeywords],
    aiLearnEnabled: props.aiLearnEnabled,
    aiExtractTodos: props.aiExtractTodos
  }
  approvalStatus.value = props.approvalStatus
  rejectReason.value = props.rejectReason
})

const saving = ref(false)

async function saveSettings() {
  saving.value = true
  try {
    emit('update', { ...settings.value })
  } finally {
    saving.value = false
  }
}

const documents = ref<GroupDocument[]>([])

// 文档处理状态轮询：后端处理是异步的（POST /process 起 goroutine 立即返回），
// 若不定时刷新，列表会一直显示"处理中"，重试后也看不到结果。
// 这里仅当 knowledge tab 可见且存在非终态（pending/processing）文档时才轮询，
// 全部进入终态即自动停止，避免无谓的后台请求。
let docStatusTimer: ReturnType<typeof setInterval> | undefined
const hasActiveDocProcess = () =>
  documents.value.some(d => d.process_status === 'pending' || d.process_status === 'processing')

function startDocStatusPolling() {
  if (docStatusTimer || activeTab.value !== 'knowledge' || !hasActiveDocProcess()) return
  docStatusTimer = setInterval(() => {
    loadDocuments()
  }, 3000)
}

function stopDocStatusPolling() {
  if (docStatusTimer) {
    clearInterval(docStatusTimer)
    docStatusTimer = undefined
  }
}

async function loadDocuments() {
  if (activeTab.value !== 'knowledge' || !hasActiveDocProcess()) {
    // 无处理中的文档（或不在知识库页）时停止轮询
    stopDocStatusPolling()
  }
  try {
    const response = await fetch(`${props.serverUrl}/api/v1/groups/${props.groupId}/ai-documents`, {
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    })
    const data = await response.json()
    if (data.code === 0) {
      documents.value = data.data || []
    }
  } catch (e) {
    console.error('加载知识库失败', e)
  } finally {
    // 每次刷新后评估：存在处理中的文档则开始（或保持）轮询
    scheduleDocStatusPolling()
  }
}

function scheduleDocStatusPolling() {
  if (activeTab.value !== 'knowledge' || !hasActiveDocProcess()) {
    stopDocStatusPolling()
    return
  }
  startDocStatusPolling()
}

async function handleAddDocuments(fileIds: number[]) {
  for (const fileId of fileIds) {
    try {
      await fetch(`${props.serverUrl}/api/v1/groups/${props.groupId}/ai-documents`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({ file_id: fileId })
      })
    } catch (e) {
      console.error('添加文档失败', e)
    }
  }
  // 提交向量化处理任务
  for (const fileId of fileIds) {
    try {
      await fetch(`${props.serverUrl}/api/v1/groups/${props.groupId}/ai-documents/${fileId}/process`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
      })
    } catch (e) {
      console.error('提交处理任务失败', e)
    }
  }
  await loadDocuments()
}

async function handleRemoveDocument(fileId: number) {
  try {
    await fetch(`${props.serverUrl}/api/v1/groups/${props.groupId}/ai-documents/${fileId}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    })
    await loadDocuments()
  } catch (e) {
    console.error('移除文档失败', e)
  }
}

async function handleRetryDocument(doc: any) {
  try {
    await fetch(`${props.serverUrl}/api/v1/groups/${props.groupId}/ai-documents/${doc.file_id}/process`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    })
    await loadDocuments()
  } catch (e) {
    console.error('重试处理失败', e)
  }
}

// 手动刷新文档列表与处理状态
async function handleRefreshDocuments() {
  await loadDocuments()
}

// ===== 群记忆 =====
const memories = ref<GroupMemory[]>([])
const searchQuery = ref('')

function formatMemoryTime(ts: string): string {
  const n = Number(ts)
  if (!n) return ''
  return new Date(n * 1000).toLocaleString('zh-CN', { hour12: false })
}

async function loadMemories() {
  try {
    const res = await request(`/api/v1/groups/${props.groupId}/group-memories`)
    if (res.code === 0) {
      memories.value = res.data || []
    }
  } catch (e) {
    console.error('加载群记忆失败', e)
  }
}

async function deleteMemory(docId: string) {
  try {
    const res = await request(`/api/v1/groups/${props.groupId}/group-memories/${docId}`, { method: 'DELETE' })
    if (res.code === 0) {
      memories.value = memories.value.filter(m => m.doc_id !== docId)
    }
  } catch (e) {
    console.error('删除群记忆失败', e)
  }
}

async function clearMemories() {
  if (!confirm('确定清空本群全部群记忆？此操作不可恢复。')) return
  try {
    const res = await request(`/api/v1/groups/${props.groupId}/group-memories`, { method: 'DELETE' })
    if (res.code === 0) {
      memories.value = []
    }
  } catch (e) {
    console.error('清空群记忆失败', e)
  }
}

async function searchMemories() {
  if (!searchQuery.value.trim()) {
    await loadMemories()
    return
  }
  try {
    const res = await request(`/api/v1/groups/${props.groupId}/group-memories/search`, {
      method: 'POST',
      body: JSON.stringify({ query: searchQuery.value.trim(), top_k: 10 })
    })
    if (res.code === 0) {
      memories.value = res.data || []
    }
  } catch (e) {
    console.error('搜索群记忆失败', e)
  }
}

onMounted(() => {
  loadAISettings()
  loadDocuments()
  scheduleDocStatusPolling()
})

// 切到知识库 tab：若存在处理中的文档则立即刷新并开始轮询；切走则停止轮询
watch(activeTab, () => {
  if (activeTab.value === 'knowledge') {
    loadDocuments()
    scheduleDocStatusPolling()
  } else {
    stopDocStatusPolling()
  }
})

onUnmounted(() => {
  stopDocStatusPolling()
})
</script>

<style scoped>
.group-ai-panel { background: var(--card-bg); border-radius: 8px; overflow: hidden; position: relative; }
.loading-overlay { position: absolute; top: 0; left: 0; right: 0; bottom: 0; background: rgba(255, 255, 255, 0.8); display: flex; align-items: center; justify-content: center; z-index: 10; }
.loading-spinner { width: 40px; height: 40px; border: 4px solid #f3f3f3; border-top: 4px solid var(--primary-color); border-radius: 50%; animation: spin 1s linear infinite; }
@keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }
.approval-notice { padding: 12px 16px; border-radius: 8px; margin-bottom: 16px; display: flex; align-items: center; gap: 8px; }
.approval-notice.pending { background-color: #fff3cd; border: 1px solid #ffc107; color: #856404; }
.approval-notice.rejected { background-color: #f8d7da; border: 1px solid #f5c6cb; color: #721c24; }
.approval-notice i { font-size: 16px; }
.tab-bar { display: flex; }
.tab-btn { flex: 1; display: flex; align-items: center; justify-content: center; gap: 6px; padding: 12px 8px; border: none; background: none; cursor: pointer; font-size: 13px; color: var(--text-secondary); border-bottom: 2px solid transparent; transition: all 0.2s; }
.tab-btn:hover { color: var(--text-color); background: var(--hover-color); }
.tab-btn.active { color: var(--primary-color); border-bottom-color: var(--primary-color); background: var(--primary-color-alpha, rgba(99, 102, 241, 0.05)); }
.tab-content { min-height: 200px; }
.tab-footer { padding: 12px 20px; display: flex; justify-content: flex-end; }
.btn { padding: 8px 20px; border-radius: 6px; font-size: 14px; cursor: pointer; border: none; font-weight: 500; }
.btn-primary { background: var(--primary-color); color: white; }
.btn-primary:hover { opacity: 0.9; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-sm { padding: 4px 12px; font-size: 13px; background: var(--card-bg, #f5f5f5); color: var(--text-color, #333); border: 1px solid var(--border-color, #ddd); }
.btn-sm:hover { background: var(--primary-color-alpha, rgba(99, 102, 241, 0.08)); }
.btn-danger { color: #e5484d; border-color: #e5484d; background: transparent; }
.btn-danger:hover { background: rgba(229, 72, 77, 0.08); }
.memory-tab { padding: 16px 20px; }
.memory-toolbar { display: flex; gap: 8px; margin-bottom: 16px; flex-wrap: wrap; }
.memory-search-input { flex: 1; min-width: 200px; padding: 6px 12px; border: 1px solid var(--border-color, #ddd); border-radius: 6px; font-size: 14px; }
.memory-empty { color: var(--text-secondary, #999); padding: 24px 0; text-align: center; font-size: 14px; }
.memory-list { display: flex; flex-direction: column; gap: 10px; }
.memory-item { padding: 12px; border: 1px solid var(--border-color, #eee); border-radius: 8px; background: var(--card-bg, #fafafa); }
.memory-content { font-size: 14px; line-height: 1.6; color: var(--text-color, #333); white-space: pre-wrap; word-break: break-word; }
.memory-meta { display: flex; align-items: center; gap: 12px; margin-top: 8px; font-size: 12px; color: var(--text-secondary, #999); }
.btn-link { background: none; border: none; color: #e5484d; cursor: pointer; font-size: 12px; padding: 0; }
.btn-link:hover { text-decoration: underline; }
</style>

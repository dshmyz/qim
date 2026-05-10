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
import { ref, watch, onMounted } from 'vue'
import AIBaseSettings from './ai-settings/AIBaseSettings.vue'
import AIPersonaSettings from './ai-settings/AIPersonaSettings.vue'
import AITriggerSettings from './ai-settings/AITriggerSettings.vue'
import AIKnowledgeSettings from './ai-settings/AIKnowledgeSettings.vue'
import type { GroupAISettings, GroupDocument } from '../../types/ai'
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
  aiAntiSpamInterval: 5,
  aiTriggerKeywords: () => [],
  aiLearnEnabled: false,
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
  { key: 'knowledge', label: '知识库', icon: 'fas fa-book' }
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
  aiLearnEnabled: props.aiLearnEnabled
})

const approvalStatus = ref<'pending' | 'approved' | 'rejected'>(props.approvalStatus)
const rejectReason = ref(props.rejectReason)
const loading = ref(false)

async function loadAISettings() {
  loading.value = true
  try {
    const response = await request(`/api/v1/conversations/${props.groupId}/ai-settings`)
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
        aiLearnEnabled: data.ai_learn_enabled !== undefined ? data.ai_learn_enabled : props.aiLearnEnabled
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

watch(() => [props.aiEnabled, props.aiAssistantName, props.aiReplyMode, props.aiPersonality, props.aiCustomPrompt, props.aiLanguage, props.aiMaxLength, props.aiMentionReplyMode, props.aiAntiSpamInterval, props.aiTriggerKeywords, props.aiLearnEnabled, props.approvalStatus, props.rejectReason], (newVal, oldVal) => {
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
    aiLearnEnabled: props.aiLearnEnabled
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

async function loadDocuments() {
  try {
    const response = await fetch(`${props.serverUrl}/api/v1/conversations/${props.groupId}/ai-documents`, {
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    })
    const data = await response.json()
    if (data.code === 0) {
      documents.value = data.data || []
    }
  } catch (e) {
    console.error('加载知识库失败', e)
  }
}

async function handleAddDocuments(fileIds: number[]) {
  for (const fileId of fileIds) {
    try {
      await fetch(`${props.serverUrl}/api/v1/conversations/${props.groupId}/ai-documents`, {
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
      await fetch(`${props.serverUrl}/api/v1/conversations/${props.groupId}/ai-documents/${fileId}/process`, {
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
    await fetch(`${props.serverUrl}/api/v1/conversations/${props.groupId}/ai-documents/${fileId}`, {
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
    await fetch(`${props.serverUrl}/api/v1/conversations/${props.groupId}/ai-documents/${doc.file_id}/process`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    })
    await loadDocuments()
  } catch (e) {
    console.error('重试处理失败', e)
  }
}

onMounted(() => {
  loadAISettings()
  loadDocuments()
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
.tab-bar { display: flex; border-bottom: 1px solid var(--border-color); }
.tab-btn { flex: 1; display: flex; align-items: center; justify-content: center; gap: 6px; padding: 12px 8px; border: none; background: none; cursor: pointer; font-size: 13px; color: var(--text-secondary); border-bottom: 2px solid transparent; transition: all 0.2s; }
.tab-btn:hover { color: var(--text-color); background: var(--hover-color); }
.tab-btn.active { color: var(--primary-color); border-bottom-color: var(--primary-color); background: var(--primary-color-alpha, rgba(99, 102, 241, 0.05)); }
.tab-content { min-height: 200px; }
.tab-footer { padding: 12px 20px; border-top: 1px solid var(--border-color); display: flex; justify-content: flex-end; }
.btn { padding: 8px 20px; border-radius: 6px; font-size: 14px; cursor: pointer; border: none; font-weight: 500; }
.btn-primary { background: var(--primary-color); color: white; }
.btn-primary:hover { opacity: 0.9; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
</style>

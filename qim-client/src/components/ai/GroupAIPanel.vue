<template>
  <div class="group-ai-panel">
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
        :documents="documents"
        @add="handleAddDocuments"
        @remove="handleRemoveDocument"
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
  aiLearnEnabled: false
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

watch(() => [props.aiEnabled, props.aiAssistantName, props.aiReplyMode, props.aiPersonality, props.aiLanguage], () => {
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

onMounted(() => {
  loadDocuments()
})
</script>

<style scoped>
.group-ai-panel { background: var(--card-bg); border-radius: 8px; overflow: hidden; }
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

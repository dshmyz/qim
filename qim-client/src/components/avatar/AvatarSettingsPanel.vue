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

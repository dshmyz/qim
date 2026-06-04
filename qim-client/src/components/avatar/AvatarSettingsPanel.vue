<template>
  <div class="avatar-settings-panel">
    <div v-if="loading" class="loading-state">
      <LoadingSpinner />
    </div>

    <div v-else-if="!config" class="empty-state">
      <EmptyState
        icon="fas fa-user-astronaut"
        title="还没有分身"
        description="创建你的 AI 分身，在你不在时代替回复消息"
      />
      <button class="create-btn" @click="handleCreate">
        <i class="fas fa-plus"></i> 创建分身
      </button>
    </div>

    <template v-else>
      <div class="tab-bar">
        <button
          v-for="tab in mainTabs"
          :key="tab.key"
          :class="['tab-btn', { active: activeMainTab === tab.key }]"
          @click="activeMainTab = tab.key"
        >
          <i :class="tab.icon"></i>
          <span>{{ tab.label }}</span>
        </button>
      </div>

      <div class="tab-content">
        <template v-if="activeMainTab === 'basic'">
          <div class="settings-section">
            <h3 class="section-title">基础配置</h3>
            <AvatarBasicSettingsSimple
              v-model="config"
            />
          </div>

          <div class="settings-section">
            <h3 class="section-title">知识来源</h3>
            <AvatarKnowledgeSettings
              v-model="config"
            />
          </div>

          <div class="settings-section">
            <h3 class="section-title">记忆管理</h3>
            <AvatarMemoryPanel
              :user-id="userId"
            />
          </div>
        </template>

        <template v-else-if="activeMainTab === 'advanced'">
          <div class="settings-section">
            <h3 class="section-title">模型配置</h3>
            <AvatarModelSettings
              v-model="config"
              :model-configs="modelConfigs"
            />
          </div>

          <div class="settings-section">
            <h3 class="section-title">人设风格</h3>
            <AvatarPersonaSettings
              v-model="config"
            />
          </div>

          <div class="settings-section">
            <h3 class="section-title">回复策略</h3>
            <AvatarReplySettings
              v-model="config"
            />
          </div>
        </template>
      </div>

      <div class="tab-footer">
        <button class="btn btn-primary" @click="handleSave" :disabled="saving">
          {{ saving ? '保存中...' : '保存设置' }}
        </button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useAvatar } from '../../composables/useAvatar'
import { useModelConfigs } from '../../composables/useModelConfigs'
import LoadingSpinner from '../shared/LoadingSpinner.vue'
import EmptyState from '../shared/EmptyState.vue'
import AvatarBasicSettingsSimple from './AvatarBasicSettingsSimple.vue'
import AvatarKnowledgeSettings from './AvatarKnowledgeSettings.vue'
import AvatarMemoryPanel from './AvatarMemoryPanel.vue'
import AvatarModelSettings from './AvatarModelSettings.vue'
import AvatarPersonaSettings from './AvatarPersonaSettings.vue'
import AvatarReplySettings from './AvatarReplySettings.vue'
import { DEFAULT_AVATAR_CONFIG } from '../../types/avatar'

const {
  config,
  loading,
  error,
  fetchConfig,
  createConfig,
  updateConfig
} = useAvatar()

const { configs: modelConfigs, fetchConfigs } = useModelConfigs()

const activeMainTab = ref<'basic' | 'advanced'>('basic')
const saving = ref(false)

const mainTabs = [
  { key: 'basic', label: '普通设置', icon: 'fas fa-cog' },
  { key: 'advanced', label: '高级设置', icon: 'fas fa-sliders-h' }
]

const userId = ref(0)

watch(activeMainTab, (newTab) => {
  localStorage.setItem('avatar-settings-tab', newTab)
})

onMounted(async () => {
  await Promise.all([fetchConfig(true), fetchConfigs()])
  
  const savedTab = localStorage.getItem('avatar-settings-tab')
  if (savedTab === 'basic' || savedTab === 'advanced') {
    activeMainTab.value = savedTab
  }
  
  const userStr = localStorage.getItem('user')
  if (userStr) {
    try {
      const user = JSON.parse(userStr)
      userId.value = user.id
    } catch (e) {
      console.error('解析用户信息失败', e)
    }
  }
})

async function handleCreate() {
  if (loading.value) return
  try {
    await createConfig(DEFAULT_AVATAR_CONFIG)
    window.$QMessage.success('分身创建成功，已自动提交审批申请')
  } catch (e: any) {
    window.$QMessage.error(error.value || '创建失败')
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
  padding: 14px 12px;
  border: none;
  background: none;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
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

.settings-section {
  margin-bottom: 24px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 12px 0;
  padding: 0 16px;
  color: var(--text-color);
}

.tab-footer {
  padding: 12px 20px;
  border-top: 1px solid var(--border-color);
  display: flex;
  justify-content: flex-end;
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

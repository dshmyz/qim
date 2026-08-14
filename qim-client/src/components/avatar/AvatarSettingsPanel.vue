<template>
  <div class="avatar-settings-panel">
    <AppHeader title="数字分身" @back="$emit('back')" />

    <div v-if="loading" class="loading-state">
      <LoadingSpinner />
    </div>

    <div v-else-if="!config" class="empty-state">
      <div class="empty-illustration">
        <i class="fas fa-user-astronaut"></i>
      </div>
      <h3 class="empty-title">还没有创建数字分身</h3>
      <p class="empty-desc">创建你的 AI 分身，在你不在时代替你回复消息</p>
      <button class="create-btn" @click="handleCreate">
        <i class="fas fa-plus"></i> 创建分身
      </button>
    </div>

    <template v-else>
      <!-- 状态概览 -->
      <div class="avatar-overview">
        <div class="overview-main">
          <div class="avatar-icon">
            <i class="fas fa-user-astronaut"></i>
          </div>
          <div class="overview-info">
            <h3>我的数字分身</h3>
            <span class="overview-status" :class="config.enabled ? 'active' : 'inactive'">
              {{ config.enabled ? '运行中' : '已关闭' }}
            </span>
          </div>
          <button
            :class="['power-btn', { on: config.enabled }]"
            @click="handleToggle"
            :disabled="toggleBusy"
          >
            <i class="fas fa-power-off"></i>
          </button>
        </div>
        <div v-if="learningStatus" class="overview-meta">
          <span class="meta-label">学习进度</span>
          <div class="meta-bar">
            <div class="meta-bar-fill" :style="{ width: learningProgress + '%' }"></div>
          </div>
          <span class="meta-value">{{ learningProgress }}%</span>
        </div>
      </div>

      <!-- 设置 Tabs -->
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

        <template v-else-if="activeMainTab === 'graph'">
          <div class="settings-section">
            <h3 class="section-title">记忆管理</h3>
            <AvatarMemoryPanel
              :user-id="userId"
            />
          </div>

          <div class="settings-section">
            <h3 class="section-title">知识图谱</h3>
            <AvatarGraph />
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
import { ref, computed, onMounted, watch } from 'vue'
import { useAvatar } from '../../composables/useAvatar'
import { useAvatarPersona } from '../../composables/useAvatarPersona'
import { useModelConfigs } from '../../composables/useModelConfigs'
import LoadingSpinner from '../shared/LoadingSpinner.vue'
import AppHeader from '../apps/AppHeader.vue'
import AvatarBasicSettingsSimple from './AvatarBasicSettingsSimple.vue'
import AvatarKnowledgeSettings from './AvatarKnowledgeSettings.vue'
import AvatarMemoryPanel from './AvatarMemoryPanel.vue'
import AvatarModelSettings from './AvatarModelSettings.vue'
import AvatarPersonaSettings from './AvatarPersonaSettings.vue'
import AvatarReplySettings from './AvatarReplySettings.vue'
import AvatarGraph from './AvatarGraph.vue'
import { DEFAULT_AVATAR_CONFIG } from '../../types/avatar'

defineEmits(['back'])

const {
  config,
  loading,
  error,
  fetchConfig,
  createConfig,
  updateConfig,
  applyForApproval
} = useAvatar()

const personaState = useAvatarPersona()
const { configs: modelConfigs, fetchConfigs } = useModelConfigs()

const learningProgress = computed(() => personaState.learnStatus.value.progress)
const learningStatus = computed(() => personaState.learnStatus.value.status)
const toggleBusy = ref(false)

async function handleToggle() {
  if (!config.value || toggleBusy.value) return
  toggleBusy.value = true
  try {
    if (config.value.enabled) {
      // 关闭分身 - 直接更新
      await updateConfig({ ...config.value, enabled: false })
      window.$QMessage.success('分身已关闭')
    } else {
      // 开启分身 - 调用审批接口（已审批通过的会直接启用）
      const result = await applyForApproval()
      if (result?.approvalStatus === 'approved') {
        window.$QMessage.success('分身已开启')
      } else {
        window.$QMessage.success('已提交开启申请，等待审批')
      }
    }
  } catch (e: any) {
    window.$QMessage.error(e?.response?.data?.message || '操作失败')
  } finally {
    toggleBusy.value = false
  }
}

const activeMainTab = ref<'basic' | 'advanced' | 'graph'>('basic')
const saving = ref(false)

const mainTabs = [
  { key: 'basic' as const, label: '普通设置', icon: 'fas fa-cog' },
  { key: 'advanced' as const, label: '高级设置', icon: 'fas fa-sliders-h' },
  { key: 'graph' as const, label: '知识图谱', icon: 'fas fa-project-diagram' }
]

const userId = ref(0)

watch(activeMainTab, (newTab) => {
  localStorage.setItem('avatar-settings-tab', newTab)
})

onMounted(async () => {
  await Promise.all([fetchConfig(true), fetchConfigs(), personaState.fetchLearnStatus()])
  
  const savedTab = localStorage.getItem('avatar-settings-tab')
  if (savedTab === 'basic' || savedTab === 'advanced' || savedTab === 'graph') {
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
  background: var(--content-bg, #f5f5f5);
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px;
}

/* ---- 空状态 ---- */
.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 24px;
}

.empty-illustration {
  width: 100px;
  height: 100px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--primary-dark, #003d99) 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 24px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
}

.empty-illustration i {
  font-size: 44px;
  color: #fff;
}

.empty-title {
  font-size: var(--font-size-xl);
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 8px 0;
}

.empty-desc {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  margin: 0 0 28px 0;
}

.create-btn {
  padding: 12px 32px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: var(--font-size-sm);
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  transition: all 0.2s;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.create-btn:hover {
  opacity: 0.9;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

/* ---- 状态概览 ---- */
.avatar-overview {
  background: var(--card-bg, #fff);
  padding: 20px;
  margin: 12px 12px 0;
  border-radius: 10px;
  flex-shrink: 0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}

.overview-main {
  display: flex;
  align-items: center;
  gap: 16px;
}

.avatar-icon {
  width: 56px;
  height: 56px;
  border-radius: 14px;
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--primary-dark, #003d99) 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: var(--font-size-2xl);
  flex-shrink: 0;
}

.overview-info {
  flex: 1;
  min-width: 0;
}

.overview-info h3 {
  margin: 0 0 4px 0;
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--text-primary);
}

.overview-status {
  font-size: var(--font-size-xxs);
  font-weight: 500;
  padding: 2px 10px;
  border-radius: 10px;
}

.overview-status.active {
  background: rgba(38, 179, 97, 0.1);
  color: #26b361;
}

.overview-status.inactive {
  background: var(--hover-color);
  color: var(--text-secondary);
}

.power-btn {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  border: 2px solid var(--border-color);
  background: var(--card-bg, #fff);
  color: var(--text-secondary);
  font-size: var(--font-size-base);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
  flex-shrink: 0;
}

.power-btn.on {
  border-color: #26b361;
  color: #26b361;
  background: rgba(38, 179, 97, 0.08);
}

.power-btn:hover {
  transform: scale(1.05);
}

.power-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.overview-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px solid var(--border-color);
}

.meta-label {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  flex-shrink: 0;
}

.meta-bar {
  flex: 1;
  height: 6px;
  background: var(--hover-color);
  border-radius: 3px;
  overflow: hidden;
}

.meta-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--primary-color), #26b361);
  border-radius: 3px;
  transition: width 0.3s;
}

.meta-value {
  font-size: var(--font-size-xxs);
  font-weight: 600;
  color: var(--text-primary);
  flex-shrink: 0;
}

/* ---- Tab 栏 ---- */
.tab-bar {
  display: flex;
  margin: 12px 12px 0;
  background: var(--card-bg, #fff);
  border-radius: 10px;
  flex-shrink: 0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  overflow: hidden;
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
  font-size: var(--font-size-sm);
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
}

/* ---- 内容区 ---- */
.tab-content {
  flex: 1;
  overflow-y: auto;
  margin: 12px;
  background: var(--card-bg, #fff);
  border-radius: 10px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}

.settings-section {
  background: var(--card-bg, #fff);
  border-radius: 10px;
  border: 1px solid var(--border-color);
  padding: 20px;
  margin-bottom: 16px;
}

.section-title {
  font-size: var(--font-size-sm);
  font-weight: 600;
  margin: 0 0 16px 0;
  color: var(--text-color);
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-color);
}

/* ---- 底部 ---- */
.tab-footer {
  padding: 12px 20px;
  margin: 0 12px 12px;
  background: var(--card-bg, #fff);
  border-radius: 10px;
  display: flex;
  justify-content: flex-end;
  flex-shrink: 0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}

.btn {
  padding: 10px 24px;
  border-radius: 8px;
  font-size: var(--font-size-sm);
  cursor: pointer;
  border: none;
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s;
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
</style>

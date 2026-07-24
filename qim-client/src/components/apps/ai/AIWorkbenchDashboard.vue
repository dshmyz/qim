<template>
  <div class="ai-workbench">
    <template v-if="viewMode === 'dashboard'">
      <QuickStartCards @action="handleQuickAction" />
      <MyAssetsSection
        :bots="bots"
        :configs="configs"
        :has-avatar-config="!!avatarConfig"
        :avatar-enabled="avatarEnabled"
        :avatar-approval-status="avatarApprovalStatus"
        :learning-progress="learningProgress"
        :learning-status="learningStatus"
        @create-bot="showCreateBot"
        @use-bot="handleUseBot"
        @config-bot="showConfigBot"
        @add-config="showAddConfig"
        @edit-config="handleEditConfig"
        @test-config="handleTestConfig"
        @delete-config="handleDeleteConfig"
        @open-avatar="viewMode = 'avatar-settings'"
        @toggle-avatar="handleToggleAvatar"
        @delete-bot="handleDeleteBot"
      />
    </template>

    <template v-else-if="viewMode === 'avatar-settings'">
      <div class="settings-header">
        <button class="back-btn" @click="viewMode = 'dashboard'">
          <i class="fas fa-chevron-left"></i>
          返回
        </button>
        <h2>数字分身设置</h2>
      </div>
      <div class="settings-content">
        <AvatarSettingsPanel />
      </div>
    </template>

    <QDialog v-model:visible="showCreateModal" title="创建机器人" width="600px">
      <CreateBotWizard @close="showCreateModal = false" @created="onBotCreated" />
    </QDialog>

    <QDialog v-model:visible="showBotConfigModal" :title="`配置 - ${configBot?.name || ''}`" width="620px">
      <BotConfigDialog v-if="configBot" :bot="configBot" @close="showBotConfigModal = false" @saved="onConfigSaved" />
    </QDialog>

    <ModelConfigFormModal
      v-model="showConfigModal"
      :config="editingConfig"
      @close="closeConfigModal"
      @save="handleSaveConfig"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import QuickStartCards from './QuickStartCards.vue'
import MyAssetsSection from './MyAssetsSection.vue'
import CreateBotWizard from './CreateBotWizard.vue'
import BotConfigDialog from './BotConfigDialog.vue'
import ModelConfigFormModal from './ModelConfigFormModal.vue'
import AvatarSettingsPanel from '../../avatar/AvatarSettingsPanel.vue'
import QDialog from '../../shared/QDialog.vue'
import { useBots } from '../../../composables/useBots'
import QMessageBox from '../../../utils/qmessagebox'
import { useModelConfigs } from '../../../composables/useModelConfigs'
import { useAvatar } from '../../../composables/useAvatar'
import { useAvatarPersona } from '../../../composables/useAvatarPersona'
import type { UserAIConfig, CreateConfigRequest } from '../../../types/ai'

const QMessage = (window as any).$QMessage

interface Bot {
  id: number
  name: string
  description?: string
  avatar?: string
  approval_status: string
}

const emit = defineEmits(['use-bot', 'back'])

const bots = ref<Bot[]>([])
const { fetchMyBots, deleteBot } = useBots()
const {
  configs,
  loading: configsLoading,
  fetchConfigs,
  createConfig,
  updateConfig,
  deleteConfig,
  testConfig
} = useModelConfigs()

const avatar = useAvatar()
const personaState = useAvatarPersona()

const viewMode = ref<'dashboard' | 'chat' | 'avatar-settings'>('dashboard')
const showCreateModal = ref(false)
const showConfigModal = ref(false)
const showBotConfigModal = ref(false)
const configBot = ref<any>(null)
const editingConfig = ref<UserAIConfig | null>(null)

const avatarConfig = computed(() => avatar.config.value)
const avatarEnabled = computed(() => avatarConfig.value?.enabled ?? false)
const avatarApprovalStatus = computed(() => avatar.avatarApprovalStatus.value)
const learningProgress = computed(() => personaState.learnStatus.value.progress)
const learningStatus = computed(() => personaState.learnStatus.value.status)

function handleQuickAction(action: string) {
  switch (action) {
    case 'chat':
      emit('use-bot', null)
      break
    case 'create':
      showCreateModal.value = true
      break
    case 'avatar':
      viewMode.value = 'avatar-settings'
      break
  }
}

function handleUseBot(bot: any) {
  emit('use-bot', bot)
}

function showCreateBot() {
  showCreateModal.value = true
}

function showConfigBot(bot: any) {
  configBot.value = bot
  showBotConfigModal.value = true
}

async function onConfigSaved() {
  bots.value = (await fetchMyBots()) || []
}

async function onBotCreated() {
  bots.value = (await fetchMyBots()) || []
}

async function handleDeleteBot(bot: Bot) {
  const result = await QMessageBox.confirm(
    `确定要删除机器人「${bot.name}」吗？删除后不可恢复。`,
    '删除机器人',
    { confirmButtonText: '删除', type: 'warning' }
  )
  if (result.action !== 'confirm') return
  try {
    const resp = await deleteBot(bot.id)
    if (!resp) {
      QMessage.error('删除失败')
      return
    }
    bots.value = (await fetchMyBots()) || []
    QMessage.success('已删除')
  } catch (e: any) {
    QMessage.error(e?.response?.data?.message || e?.message || '删除失败')
  }
}

function showAddConfig() {
  console.log('[AIWorkbench] showAddConfig called')
  editingConfig.value = null
  showConfigModal.value = true
  console.log('[AIWorkbench] showConfigModal =', showConfigModal.value)
}

function handleEditConfig(config: UserAIConfig) {
  console.log('[AIWorkbench] handleEditConfig called, config =', config)
  editingConfig.value = config
  showConfigModal.value = true
}

function closeConfigModal() {
  showConfigModal.value = false
  editingConfig.value = null
}

async function handleSaveConfig(data: CreateConfigRequest) {
  if (editingConfig.value) {
    await updateConfig(editingConfig.value.id, data)
  } else {
    await createConfig(data)
  }
  closeConfigModal()
}

async function handleTestConfig(id: number) {
  try {
    const result = await testConfig(id)
    if (result.success) {
      QMessage.success('连接测试成功')
    } else {
      QMessage.error(`连接失败: ${result.message}`)
    }
  } catch (e: any) {
    QMessage.error(`测试失败: ${e?.response?.data?.message || '未知错误'}`)
  }
}

async function handleDeleteConfig(config: UserAIConfig) {
  const result = await QMessageBox.confirm(
    `确定要删除配置 "${config.config_name}" 吗？`,
    '删除配置',
    { confirmButtonText: '删除', type: 'warning' }
  )
  if (result.action !== 'confirm') return
  await deleteConfig(config.id)
}

async function handleToggleAvatar() {
  if (!avatarConfig.value) return
  try {
    if (avatarEnabled.value) {
      // 关闭分身：直接关闭
      await avatar.toggleEnabled(false)
    } else {
      // 开启分身：走申请流程（后端根据审批状态决定直接启用还是走审批）
      await avatar.toggleEnabled(true)
    }
  } catch (e: any) {
    const msg = e?.response?.data?.message || '操作失败'
    window.$QMessage?.error(msg)
  }
}

onMounted(async () => {
  const [botsData] = await Promise.all([
    fetchMyBots(),
    fetchConfigs(),
    avatar.fetchConfig(),
    personaState.fetchLearnStatus()
  ])
  bots.value = botsData || []
})
</script>

<style scoped>
.ai-workbench {
  padding: 24px;
  height: 100%;
  overflow-y: auto;
}

.settings-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
}

.back-btn {
  padding: 8px 14px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--hover-color);
  color: var(--text-primary);
  font-size: 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s;
}

.back-btn:hover {
  background: var(--border-color);
}

.settings-header h2 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.settings-content {
  background: var(--card-bg);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-color);
  overflow: hidden;
  height: calc(100% - 60px);
}
</style>
<template>
  <div class="ai-workbench">
    <template v-if="viewMode === 'dashboard'">
      <QuickStartCards @action="handleQuickAction" />
      <MyAssetsSection
        :bots="bots"
        :configs="configs"
        @create-bot="showCreateBot"
        @use-bot="handleUseBot"
        @config-bot="showConfigBot"
        @add-config="showAddConfig"
        @edit-config="handleEditConfig"
        @test-config="handleTestConfig"
        @delete-config="handleDeleteConfig"
        @delete-bot="handleDeleteBot"
      />
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
import { ref, onMounted } from 'vue'
import QuickStartCards from './QuickStartCards.vue'
import MyAssetsSection from './MyAssetsSection.vue'
import CreateBotWizard from './CreateBotWizard.vue'
import BotConfigDialog from './BotConfigDialog.vue'
import ModelConfigFormModal from './ModelConfigFormModal.vue'
import QDialog from '../../shared/QDialog.vue'
import { useBots } from '../../../composables/useBots'
import QMessageBox from '../../../utils/qmessagebox'
import { useModelConfigs } from '../../../composables/useModelConfigs'
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

const viewMode = ref<'dashboard' | 'chat'>('dashboard')
const showCreateModal = ref(false)
const showConfigModal = ref(false)
const showBotConfigModal = ref(false)
const configBot = ref<any>(null)
const editingConfig = ref<UserAIConfig | null>(null)

function handleQuickAction(action: string) {
  switch (action) {
    case 'chat':
      emit('use-bot', null)
      break
    case 'create':
      showCreateModal.value = true
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

onMounted(async () => {
  const [botsData] = await Promise.all([
    fetchMyBots(),
    fetchConfigs()
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
</style>
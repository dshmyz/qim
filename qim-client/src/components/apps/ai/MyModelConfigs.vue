<template>
  <div class="my-model-configs">
    <div class="configs-header">
      <h3>我的模型配置</h3>
      <button class="add-btn" @click="showModal = true" :disabled="configs.length >= 5">
        <i class="fas fa-plus"></i>
        添加配置
      </button>
    </div>

    <div v-if="loading" class="loading">
      <i class="fas fa-spinner fa-spin"></i>
      <span>加载中...</span>
    </div>

    <div v-else-if="configs.length === 0" class="empty-state">
      <i class="fas fa-key"></i>
      <p>暂无配置</p>
      <p class="hint">添加你的API配置，用于创建自定义机器人</p>
    </div>

    <div v-else class="configs-list">
      <ModelConfigCard
        v-for="config in configs"
        :key="config.id"
        :config="config"
        @edit="editConfig(config)"
        @test="testConfigItem(config.id)"
        @delete="confirmDelete(config)"
      />
    </div>

    <div v-if="configs.length >= 5" class="limit-hint">
      <i class="fas fa-info-circle"></i>
      配置数量已达上限（5个）
    </div>

    <ModelConfigFormModal
      v-if="showModal"
      :config="editingConfig"
      @close="closeModal"
      @save="handleSave"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useModelConfigs } from '../../../composables/useModelConfigs'
import ModelConfigCard from './ModelConfigCard.vue'
import ModelConfigFormModal from './ModelConfigFormModal.vue'
import type { UserAIConfig, CreateConfigRequest } from '../../../types/ai'

const {
  configs,
  loading,
  fetchConfigs,
  createConfig,
  updateConfig,
  deleteConfig,
  testConfig
} = useModelConfigs()

const showModal = ref(false)
const editingConfig = ref<UserAIConfig | null>(null)

onMounted(() => {
  fetchConfigs()
})

function editConfig(config: UserAIConfig) {
  editingConfig.value = config
  showModal.value = true
}

function closeModal() {
  showModal.value = false
  editingConfig.value = null
}

async function handleSave(data: CreateConfigRequest) {
  if (editingConfig.value) {
    await updateConfig(editingConfig.value.id, data)
  } else {
    await createConfig(data)
  }
  closeModal()
}

async function testConfigItem(id: number) {
  const result = await testConfig(id)
  alert(result.success ? '连接测试成功' : `连接失败: ${result.message}`)
}

function confirmDelete(config: UserAIConfig) {
  if (confirm(`确定要删除配置 "${config.config_name}" 吗？`)) {
    deleteConfig(config.id)
  }
}
</script>

<style scoped>
.my-model-configs {
  padding: 20px;
}

.configs-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.configs-header h3 {
  margin: 0;
  font-size: 16px;
}

.add-btn {
  padding: 8px 16px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
}

.add-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.loading {
  text-align: center;
  padding: 40px;
  color: var(--text-secondary);
}

.loading i {
  font-size: 24px;
  margin-right: 8px;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: var(--text-secondary);
}

.empty-state i {
  font-size: 48px;
  margin-bottom: 12px;
  color: var(--text-secondary);
}

.hint {
  font-size: 13px;
  color: var(--text-secondary);
}

.configs-list {
  display: grid;
  gap: 16px;
}

.limit-hint {
  margin-top: 16px;
  padding: 12px;
  background: #FFF8E1;
  border-radius: 6px;
  color: #FF9800;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>

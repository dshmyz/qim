<template>
  <div class="create-bot-wizard">
    <div class="wizard-header">
      <button class="back-btn" @click="$emit('close')">
        <i class="fas fa-arrow-left"></i>
      </button>
      <h3>创建机器人</h3>
    </div>

    <div class="wizard-body">
      <div v-if="!step" class="method-selector">
        <div class="method-option" @click="step = 'template'">
          <i class="fas fa-layer-group"></i>
          <h4>使用模板</h4>
          <p>从预设模板快速创建</p>
        </div>
        <div class="method-option" @click="step = 'custom'">
          <i class="fas fa-edit"></i>
          <h4>自定义</h4>
          <p>完全自定义配置</p>
        </div>
      </div>

      <div v-else-if="step === 'template'" class="template-list">
        <div v-if="templates.length === 0" class="empty-templates">
          <i class="fas fa-inbox"></i>
          <p>暂无可用模板</p>
          <button class="switch-btn" @click="step = 'custom'">切换到自定义</button>
        </div>
        <div v-else class="templates">
          <div v-for="tpl in templates" :key="tpl.id" class="template-item" @click="createFromTemplate(tpl)">
            <div class="template-avatar">
              <img :src="tpl.avatar" :alt="tpl.name" v-if="tpl.avatar">
              <i class="fas fa-robot" v-else></i>
            </div>
            <div class="template-info">
              <h4>{{ tpl.name }}</h4>
              <p>{{ tpl.description }}</p>
            </div>
          </div>
        </div>
      </div>

      <div v-else class="custom-form">
        <div class="form-group">
          <label>名称</label>
          <input v-model="form.name" placeholder="机器人名称">
        </div>
        <div class="form-group">
          <label>描述</label>
          <textarea v-model="form.description" rows="3" placeholder="机器人描述"></textarea>
        </div>
        <div class="form-group">
          <label>模型来源</label>
          <select v-model="form.useSystemConfig">
            <option :value="true">使用系统默认模型</option>
            <option :value="false">使用我的自定义配置</option>
          </select>
        </div>
        <div v-if="!form.useSystemConfig" class="form-group">
          <label>选择配置</label>
          <select v-model="form.configId">
            <option value="">请选择...</option>
            <option v-for="cfg in myConfigs" :key="cfg.id" :value="cfg.id">
              {{ cfg.config_name }} ({{ cfg.model_name }})
            </option>
          </select>
          <p class="hint" v-if="myConfigs.length === 0">
            暂无配置，请先在"我的模型配置"中添加
          </p>
        </div>
        <div class="form-group">
          <label>系统提示词</label>
          <textarea v-model="form.system_prompt" rows="5" placeholder="定义机器人的行为和角色..."></textarea>
        </div>
        <button class="submit-btn" @click="handleSubmit" :disabled="creating">
          {{ creating ? '创建中...' : '创建' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useBots } from '../../../composables/useBots'
import { useModelConfigs } from '../../../composables/useModelConfigs'
import type { UserAIConfig } from '../../../types/ai'

const emit = defineEmits<{
  close: []
}>()

const step = ref<'template' | 'custom' | null>(null)
const templates = ref<any[]>([])
const creating = ref(false)

const { fetchTemplates, createBot } = useBots()
const { configs: myConfigs, fetchConfigs } = useModelConfigs()

const form = ref({
  name: '',
  description: '',
  useSystemConfig: true,
  configId: null as number | null,
  system_prompt: ''
})

onMounted(async () => {
  templates.value = await fetchTemplates()
  await fetchConfigs()
})

async function createFromTemplate(tpl: any) {
  creating.value = true
  try {
    const response = await createBot({
      name: tpl.name,
      description: tpl.description,
      system_prompt: tpl.system_prompt || '',
      is_template: true,
      use_system_config: true
    })

    if (response.code === 0) {
      alert('机器人创建成功')
      emit('close')
    } else {
      alert(response.message || '创建失败')
    }
  } catch (e: any) {
    alert('创建失败: ' + (e.response?.data?.message || e.message))
  } finally {
    creating.value = false
  }
}

async function handleSubmit() {
  if (!form.value.name.trim()) {
    alert('请输入机器人名称')
    return
  }

  if (!form.value.useSystemConfig && !form.value.configId) {
    alert('请选择一个模型配置')
    return
  }

  creating.value = true
  try {
    const response = await createBot({
      name: form.value.name,
      description: form.value.description,
      system_prompt: form.value.system_prompt,
      use_system_config: form.value.useSystemConfig,
      user_config_id: form.value.configId
    })

    if (response.code === 0) {
      const msg = form.value.useSystemConfig ? '已提交审批，等待管理员审核' : '机器人创建成功'
      alert(msg)
      emit('close')
    } else {
      alert(response.message || '创建失败')
    }
  } catch (e: any) {
    alert('创建失败: ' + (e.response?.data?.message || e.message))
  } finally {
    creating.value = false
  }
}
</script>

<style scoped>
.create-bot-wizard {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.wizard-header {
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  gap: 12px;
}

.wizard-header h3 {
  margin: 0;
  font-size: 16px;
}

.back-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 6px;
  color: var(--text-primary);
}

.back-btn:hover {
  background: var(--hover-color);
}

.wizard-body {
  padding: 20px;
  flex: 1;
  overflow-y: auto;
}

.method-selector {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.method-option {
  padding: 24px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  text-align: center;
  cursor: pointer;
  transition: all 0.2s;
}

.method-option:hover {
  border-color: var(--primary-color);
  background: var(--hover-color);
}

.method-option i {
  font-size: 32px;
  color: var(--primary-color);
  margin-bottom: 12px;
}

.method-option h4 {
  margin: 0 0 8px;
  font-size: 16px;
}

.method-option p {
  margin: 0;
  font-size: 13px;
  color: var(--text-secondary);
}

.template-list {
  display: flex;
  flex-direction: column;
}

.empty-templates {
  text-align: center;
  padding: 60px 20px;
  color: var(--text-secondary);
}

.empty-templates i {
  font-size: 48px;
  margin-bottom: 12px;
  color: var(--text-tertiary);
}

.switch-btn {
  margin-top: 16px;
  padding: 8px 16px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
}

.switch-btn:hover {
  opacity: 0.9;
}

.templates {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.template-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.template-item:hover {
  border-color: var(--primary-color);
  background: var(--hover-color);
}

.template-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--bg-color);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  flex-shrink: 0;
}

.template-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.template-avatar i {
  font-size: 24px;
  color: var(--primary-color);
}

.template-info h4 {
  margin: 0 0 4px;
  font-size: 15px;
}

.template-info p {
  margin: 0;
  font-size: 13px;
  color: var(--text-secondary);
}

.custom-form {
  max-width: 600px;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
}

.form-group input,
.form-group select,
.form-group textarea {
  width: 100%;
  padding: 10px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-color);
  color: var(--text-primary);
  box-sizing: border-box;
  font-family: inherit;
}

.form-group input:focus,
.form-group select:focus,
.form-group textarea:focus {
  outline: none;
  border-color: var(--primary-color);
}

.form-group textarea {
  resize: vertical;
}

.hint {
  margin-top: 8px;
  font-size: 13px;
  color: var(--text-secondary);
}

.submit-btn {
  width: 100%;
  padding: 12px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}

.submit-btn:hover:not(:disabled) {
  opacity: 0.9;
}

.submit-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>

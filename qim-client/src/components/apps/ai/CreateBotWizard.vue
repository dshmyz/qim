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
          <div v-for="tpl in templates" :key="tpl.id" class="template-item" @click="selectTemplate(tpl)">
            <div class="template-avatar">
              <Avatar v-if="tpl.avatar" :src="tpl.avatar" :name="tpl.name" :alt="tpl.name" size="sm" />
              <i class="fas fa-robot" v-else></i>
            </div>
            <div class="template-info">
              <h4>{{ tpl.name }}</h4>
              <p>{{ tpl.description }}</p>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="step === 'connect'" class="connect-step">
        <div class="connect-header">
          <i class="fas fa-check-circle"></i>
          <h3>{{ createdBotName }} 创建成功</h3>
          <p v-if="createdApprovalStatus === 'pending'" class="approval-hint">
            <i class="fas fa-clock"></i>
            当前审批状态为待审核，请联系管理员审批后方可签发 Token
          </p>
          <p v-else class="ready-hint">机器人已就绪，下面开始接入</p>
        </div>

        <div class="connect-section">
          <h4><i class="fas fa-terminal"></i> 安装 CLI 工具</h4>
          <div class="code-block" @click="copyText('go install github.com/user/qim-server/cmd/qim-mcp@latest', 'go')">
            <code>go install github.com/user/qim-server/cmd/qim-mcp@latest</code>
            <span class="copy-hint">{{ copiedField === 'go' ? '已复制' : '点击复制' }}</span>
          </div>
        </div>

        <div class="connect-section">
          <h4><i class="fas fa-key"></i> 签发 Bot Token</h4>
          <div class="token-input-row">
            <input
              v-model="mcpToken"
              type="text"
              placeholder="粘贴已签发的 Token（qbot_xxx）"
              class="token-input"
            >
          </div>
          <p class="section-hint">
            在 Bot 配置面板的「Token 管理」中签发，签发后会显示在此处
          </p>
        </div>

        <div class="connect-section">
          <h4><i class="fas fa-plug"></i> MCP Server 配置</h4>
          <p class="section-desc">将以下内容加入 Claude Code / Cursor 的 MCP 配置：</p>
          <div class="code-block" @click="copyText(mcpConfigJson, 'mcp')">
            <pre><code>{{ mcpConfigJson }}</code></pre>
            <span class="copy-hint">{{ copiedField === 'mcp' ? '已复制' : '点击复制' }}</span>
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

        <!-- 回复模式 + Webhook 配置 + 知识库（统一子组件） -->
        <BotReplyConfigFields
          v-model:mode="form.mode"
          v-model:use-creator-notes="form.use_creator_notes"
          v-model:webhook-url="form.webhook_url"
          v-model:webhook-secret="form.webhook_secret"
          secret-placeholder="••••••（留空不设置）"
          secret-hint="留空则不设置密钥；建议生成随机值保存到 agent 配置中"
          :vector-enabled="vectorEnabled"
        />

        <button class="submit-btn" @click="handleSubmit" :disabled="creating">
          {{ creating ? '创建中...' : '创建' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import Avatar from '../../shared/Avatar.vue'
import { useBots } from '../../../composables/useBots'
import { useModelConfigs } from '../../../composables/useModelConfigs'
import { getStoredServerUrl } from '../../../composables/useServerUrl'
import { useSystemConfigStore } from '../../../stores/systemConfig'
import type { UserAIConfig } from '../../../types/ai'
import BotReplyConfigFields from './BotReplyConfigFields.vue'

const QMessage = (window as any).$QMessage

const emit = defineEmits<{
  close: []
  created: [botId: number]
}>()

const step = ref<'template' | 'custom' | 'connect' | null>(null)
const templates = ref<any[]>([])
const creating = ref(false)

const { fetchTemplates, createBot } = useBots()
const { configs: myConfigs, fetchConfigs } = useModelConfigs()
const systemConfigStore = useSystemConfigStore()
const vectorEnabled = computed(() => systemConfigStore.vectorEnabled)

const form = ref({
  name: '',
  description: '',
  useSystemConfig: true,
  configId: null as number | null,
  system_prompt: '你是一个友好的AI助手，能够帮助用户回答问题、完成任务。请用简洁清晰的语言回复。',
  // 回复路由配置（与 BotConfigDialog 同字段，创建时一并提交）
  mode: 'internal_ai' as 'internal_ai' | 'external_webhook',
  use_creator_notes: false,
  webhook_url: '',
  webhook_secret: '',
})

// Connect step state
const createdBotId = ref<number | null>(null)
const createdBotName = ref('')
const createdApprovalStatus = ref('')
const mcpToken = ref('')
const copiedField = ref<string | null>(null)
let copyTimer: ReturnType<typeof setTimeout> | null = null

const serverUrl = computed(() => getStoredServerUrl())

const mcpConfigJson = computed(() => {
  const url = serverUrl.value
  return JSON.stringify({
    mcpServers: {
      'qim-bot': {
        type: 'stdio',
        command: 'qim-mcp',
        args: ['--server', url, '--token', mcpToken.value || '<YOUR_TOKEN>']
      }
    }
  }, null, 2)
})

async function copyText(text: string, field: string) {
  try {
    await navigator.clipboard.writeText(text)
    if (copyTimer) clearTimeout(copyTimer)
    copiedField.value = field
    copyTimer = setTimeout(() => { copiedField.value = null }, 2000)
  } catch {
    QMessage.warning('复制失败，请手动复制')
  }
}

onMounted(async () => {
  templates.value = await fetchTemplates()
  await fetchConfigs()
  // 加载公开配置以获取向量库状态（决定知识库开关是否可用）
  if (!systemConfigStore.loaded) {
    systemConfigStore.fetchPublicConfig()
  }
})

function selectTemplate(tpl: any) {
  form.value.name = tpl.name
  form.value.description = tpl.description
  form.value.useSystemConfig = true
  form.value.configId = null

  // 默认值，模板未提供时回退到这些值
  let systemPrompt = ''
  let mode: 'internal_ai' | 'external_webhook' = 'internal_ai'
  let useCreatorNotes = false
  let webhookUrl = ''

  if (tpl.config) {
    try {
      const config = typeof tpl.config === 'string' ? JSON.parse(tpl.config) : tpl.config
      systemPrompt = config.system_prompt || ''
      if (config.mode === 'external_webhook') {
        mode = 'external_webhook'
        webhookUrl = config.webhook_url || ''
      } else {
        mode = 'internal_ai'
        // 仅 internal_ai 模式下读取知识库开关
        useCreatorNotes = config.use_creator_notes === true
      }
    } catch (e) {
      console.error('解析模板配置失败', e)
    }
  }

  form.value.system_prompt = systemPrompt || '你是一个友好的AI助手，能够帮助用户回答问题、完成任务。请用简洁清晰的语言回复。'
  form.value.mode = mode
  form.value.use_creator_notes = useCreatorNotes
  form.value.webhook_url = webhookUrl
  // webhook_secret 不从模板预填：模板是共享配置，不应携带密钥；用户需自己生成或留空
  form.value.webhook_secret = ''
  step.value = 'custom'
}

async function handleSubmit() {
  if (!form.value.name.trim()) {
    QMessage.warning('请输入机器人名称')
    return
  }

  if (!form.value.useSystemConfig && !form.value.configId) {
    QMessage.warning('请选择一个模型配置')
    return
  }

  creating.value = true
  try {
    // 拼装 config：基础 AI 配置 + 回复路由配置（mode/webhook/知识库）
    const config: Record<string, any> = {
      system_prompt: form.value.system_prompt,
      use_system_config: form.value.useSystemConfig,
      user_config_id: form.value.configId,
      mode: form.value.mode,
      use_creator_notes: form.value.mode === 'internal_ai' && form.value.use_creator_notes === true,
    }
    if (form.value.mode === 'external_webhook') {
      config.webhook_url = form.value.webhook_url
      if (form.value.webhook_secret) config.webhook_secret = form.value.webhook_secret
    }

    const response = await createBot({
      name: form.value.name,
      description: form.value.description,
      type: 'assistant',
      config,
    })

    if (response.code === 0) {
      const bot = response.data || {}
      createdBotId.value = bot.id
      createdBotName.value = form.value.name
      createdApprovalStatus.value = bot.approval_status || 'pending'
      mcpToken.value = ''
      step.value = 'connect'
      emit('created', bot.id)
    } else {
      QMessage.error(response.message || '创建失败')
    }
  } catch (e: any) {
    QMessage.error('创建失败: ' + (e.response?.data?.message || e.message))
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

.connect-step {
  max-width: 600px;
}

.connect-header {
  text-align: center;
  margin-bottom: 28px;
}

.connect-header > i {
  font-size: 40px;
  color: #52c41a;
  margin-bottom: 8px;
}

.connect-header h3 {
  margin: 0 0 8px;
  font-size: 18px;
}

.approval-hint {
  color: #faad14;
  font-size: 13px;
  margin: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.ready-hint {
  color: var(--text-secondary);
  font-size: 13px;
  margin: 0;
}

.connect-section {
  margin-bottom: 24px;
}

.connect-section h4 {
  margin: 0 0 10px;
  font-size: 14px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-primary);
}

.connect-section h4 i {
  color: var(--primary-color);
  font-size: 13px;
}

.code-block {
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  padding: 12px 14px;
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 12.5px;
  line-height: 1.5;
  cursor: pointer;
  position: relative;
  transition: border-color 0.2s;
  word-break: break-all;
}

.code-block:hover {
  border-color: var(--primary-color);
}

.code-block pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}

.code-block code {
  font-family: inherit;
}

.copy-hint {
  position: absolute;
  top: 8px;
  right: 10px;
  font-size: 11px;
  color: var(--text-tertiary);
  font-family: inherit;
}

.token-input-row {
  display: flex;
  gap: 8px;
}

.token-input {
  flex: 1;
  padding: 10px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-color);
  color: var(--text-primary);
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 13px;
}

.token-input:focus {
  outline: none;
  border-color: var(--primary-color);
}

.section-desc {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0 0 10px;
}

.section-hint {
  font-size: 12px;
  color: var(--text-tertiary);
  margin: 6px 0 0;
}
</style>

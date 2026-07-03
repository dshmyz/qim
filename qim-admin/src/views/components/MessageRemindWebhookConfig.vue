<template>
  <div class="webhook-config-section">
    <el-divider content-position="left">消息提醒 Webhook</el-divider>

    <el-form label-width="120px" v-loading="loading">
      <el-form-item label="启用">
        <el-switch v-model="config.enabled" active-text="开启" inactive-text="关闭" />
        <span class="desc" style="margin-left: 8px">（开启后右键消息"发送提醒"将调用外部系统）</span>
      </el-form-item>

      <el-form-item label="请求地址">
        <el-input
          v-model="config.url"
          placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx"
          clearable
        />
      </el-form-item>

      <el-form-item label="请求方法">
        <el-select v-model="config.method" style="width: 140px">
          <el-option label="POST" value="POST" />
          <el-option label="PUT" value="PUT" />
        </el-select>
      </el-form-item>

      <el-form-item label="超时秒数">
        <el-input-number v-model="config.timeout_seconds" :min="1" :max="30" :step="1" />
      </el-form-item>

      <el-form-item label="签名密钥">
        <el-input
          v-model="config.secret"
          type="password"
          placeholder="HMAC-SHA256 密钥（可选，留空则不签名）"
          show-password
        />
      </el-form-item>

      <el-form-item label="自定义请求头">
        <div class="headers-list">
          <div v-for="(item, index) in headerList" :key="index" class="header-row">
            <el-input v-model="item.key" placeholder="Header 名" style="width: 200px" />
            <el-input v-model="item.value" placeholder="Header 值" style="width: 320px" />
            <el-button type="danger" size="small" @click="removeHeader(index)">删除</el-button>
          </div>
          <el-button size="small" @click="addHeader">+ 添加请求头</el-button>
        </div>
      </el-form-item>

      <el-form-item label="请求体模板">
        <el-input
          v-model="config.body_template"
          type="textarea"
          :rows="6"
          placeholder='{"msgtype":"text","text":{"content":"{{.SenderNickname}} 提醒你：{{.MessageContentPreview}}"}}'
        />
      </el-form-item>

      <el-form-item label="快速填充模板">
        <el-button size="small" @click="applyTemplate('wechat')">企业微信</el-button>
        <el-button size="small" @click="applyTemplate('feishu')">飞书</el-button>
        <el-button size="small" @click="applyTemplate('slack')">Slack</el-button>
        <el-button size="small" @click="applyTemplate('custom')">清空</el-button>
      </el-form-item>

      <el-form-item label="可用变量">
        <div class="variables-help">
          <div v-for="v in variables" :key="v.name" class="variable-item">
            <code>{{ v.name }}</code>
            <span>{{ v.desc }}</span>
          </div>
        </div>
      </el-form-item>

      <el-form-item>
        <el-button type="primary" :loading="submitting" @click="handleSave">保存配置</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getSystemConfig, updateSystemConfig } from '@/api/systemConfig'

interface WebhookConfig {
  enabled: boolean
  url: string
  method: string
  secret: string
  timeout_seconds: number
  headers: Record<string, string>
  body_template: string
}

const loading = ref(false)
const submitting = ref(false)

const config = reactive<WebhookConfig>({
  enabled: false,
  url: '',
  method: 'POST',
  secret: '',
  timeout_seconds: 10,
  headers: {},
  body_template: ''
})

const headerList = ref<Array<{ key: string; value: string }>>([])

const templates: Record<string, string> = {
  wechat: '{"msgtype":"text","text":{"content":"{{.SenderNickname}} 提醒你查看 QIM 消息：{{.MessageContentPreview}}"}}',
  feishu: '{"msg_type":"text","content":{"text":"{{.SenderNickname}} 提醒你查看 QIM 消息：{{.MessageContentPreview}}"}}',
  slack: '{"text":"{{.SenderNickname}} 提醒你查看 QIM 消息：{{.MessageContentPreview}}"}',
  custom: ''
}

const variables = [
  { name: '{{.SenderNickname}}', desc: '发送者昵称' },
  { name: '{{.SenderUsername}}', desc: '发送者用户名' },
  { name: '{{.SenderEmail}}', desc: '发送者邮箱' },
  { name: '{{.RecipientNickname}}', desc: '接收者昵称' },
  { name: '{{.RecipientEmail}}', desc: '接收者邮箱' },
  { name: '{{.MessageContentPreview}}', desc: '消息内容前 100 字符' },
  { name: '{{.MessageURL}}', desc: '消息跳转链接' },
  { name: '{{.MessageType}}', desc: '消息类型' },
  { name: '{{.MessageSentAt}}', desc: '消息发送时间' },
  { name: '{{.ConversationType}}', desc: '会话类型' }
]

const applyTemplate = (type: string) => {
  config.body_template = templates[type] || ''
}

const addHeader = () => {
  headerList.value.push({ key: '', value: '' })
}

const removeHeader = (index: number) => {
  headerList.value.splice(index, 1)
}

const syncHeadersToConfig = () => {
  const headers: Record<string, string> = {}
  headerList.value.forEach(h => {
    if (h.key.trim()) headers[h.key.trim()] = h.value
  })
  config.headers = headers
}

const loadConfig = async () => {
  loading.value = true
  try {
    const { data } = await getSystemConfig()
    const raw = (data.data as any).message_remind_webhook
    if (raw) {
      const parsed: WebhookConfig = typeof raw === 'string' ? JSON.parse(raw) : raw
      Object.assign(config, parsed)
      // 还原 headers 列表
      headerList.value = Object.entries(config.headers || {}).map(([key, value]) => ({ key, value }))
    }
  } catch (e) {
    // 配置未初始化或其他错误，使用默认值
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  submitting.value = true
  try {
    syncHeadersToConfig()
    await updateSystemConfig({
      message_remind_webhook: JSON.stringify(config)
    } as any)
    ElMessage.success('Webhook 配置保存成功')
  } catch (e) {
    ElMessage.error('保存失败')
  } finally {
    submitting.value = false
  }
}

onMounted(loadConfig)
</script>

<style scoped>
.webhook-config-section {
  margin-top: 20px;
}

.desc {
  color: var(--color-text-muted, #909399);
  font-size: 12px;
}

.headers-list {
  width: 100%;
}

.header-row {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}

.variables-help {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px 16px;
  font-size: 12px;
  color: var(--color-text-muted, #909399);
}

.variable-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.variable-item code {
  background: var(--color-bg-light, #f5f7fa);
  padding: 2px 6px;
  border-radius: 3px;
  font-family: 'Monaco', 'Menlo', monospace;
  color: var(--color-text-primary, #303133);
  white-space: nowrap;
}
</style>

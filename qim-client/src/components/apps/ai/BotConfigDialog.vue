<template>
  <div class="bot-config-dialog">
    <div v-if="loading" class="loading">
      <i class="fas fa-spinner fa-spin"></i><span>加载中...</span>
    </div>

    <template v-else>
      <!-- 模式切换 -->
      <section class="config-section">
        <h4>回复模式</h4>
        <div class="mode-options">
          <label class="mode-option" :class="{ active: form.mode === 'internal_ai' }">
            <input type="radio" v-model="form.mode" value="internal_ai" />
            <div class="mode-text">
              <span class="mode-title">内置 AI</span>
              <span class="mode-desc">用户回复走 QIM 内置大模型，无需额外配置</span>
            </div>
          </label>
          <label class="mode-option" :class="{ active: form.mode === 'external_webhook' }">
            <input type="radio" v-model="form.mode" value="external_webhook" />
            <div class="mode-text">
              <span class="mode-title">外部 Agent（Webhook）</span>
              <span class="mode-desc">用户回复转发到外部 agent（如 Claude Code），agent 经 Bot API 回复</span>
            </div>
          </label>
        </div>
      </section>

      <!-- Webhook 配置（仅 external 模式） -->
      <section v-if="form.mode === 'external_webhook'" class="config-section">
        <h4>Webhook 配置</h4>
        <div class="form-field">
          <label>回调地址</label>
          <input v-model="form.webhook_url" placeholder="https://your-agent.example/qim-webhook" />
          <p class="field-hint">QIM 把用户回复 POST 到此地址（HMAC-SHA256 签名）</p>
        </div>
        <div class="form-field">
          <label>签名密钥</label>
          <div class="secret-field">
            <input
              :type="showSecret ? 'text' : 'password'"
              v-model="form.webhook_secret"
              :placeholder="secretPlaceholder"
            />
            <button type="button" class="mini-btn" @click="generateSecret">生成随机</button>
            <button type="button" class="mini-btn" @click="showSecret = !showSecret">
              {{ showSecret ? '隐藏' : '显示' }}
            </button>
          </div>
          <p class="field-hint">留空则不修改既有密钥（仅写入，服务端不回显）</p>
        </div>
      </section>

      <!-- Token 管理 -->
      <section class="config-section">
        <div class="section-header">
          <h4>访问令牌</h4>
          <button class="action-btn primary" @click="onIssueToken" :disabled="issuing">
            <i class="fas fa-plus"></i> 签发新令牌
          </button>
        </div>
        <p class="field-hint">agent 经 qim CLI / Bot API 调用时用此令牌鉴权（Authorization: Bearer）</p>

        <!-- 新签发明文（仅本次显示） -->
        <div v-if="newToken" class="new-token-box">
          <i class="fas fa-exclamation-triangle"></i>
          <div class="new-token-text">
            <strong>令牌仅显示一次，请立即复制保存：</strong>
            <code>{{ newToken.token }}</code>
          </div>
          <button class="mini-btn" @click="copyToken(newToken.token)"><i class="fas fa-copy"></i> 复制</button>
        </div>

        <div v-if="tokens.length === 0" class="empty-tokens">暂无令牌</div>
        <table v-else class="token-table">
          <thead>
            <tr><th>名称</th><th>创建时间</th><th>最近使用</th><th>操作</th></tr>
          </thead>
          <tbody>
            <tr v-for="t in tokens" :key="t.id">
              <td>{{ t.name || '（未命名）' }}</td>
              <td>{{ formatTime(t.created_at) }}</td>
              <td>{{ t.last_used_at ? formatTime(t.last_used_at) : '—' }}</td>
              <td><button class="action-btn danger mini" @click="onRevoke(t)">撤销</button></td>
            </tr>
          </tbody>
        </table>
      </section>

      <div class="dialog-footer">
        <button class="action-btn" @click="$emit('close')">取消</button>
        <button class="action-btn primary" @click="onSave" :disabled="saving">
          <i class="fas fa-save"></i> 保存配置
        </button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useBotConfig } from '../../../composables/useBotConfig'
import type { BotTokenInfo, BotWebhookConfig } from '../../../types/bot'

const QMessage = (window as any).$QMessage
const props = defineProps<{ bot: any }>()
const emit = defineEmits(['close', 'saved'])

const { loading, issueToken, listTokens, revokeToken, updateConfig } = useBotConfig()

const form = ref<BotWebhookConfig>({ mode: 'internal_ai', webhook_url: '', webhook_secret: '' })
const secretPlaceholder = ref('••••••（留空不修改）')
const showSecret = ref(false)
const tokens = ref<BotTokenInfo[]>([])
const newToken = ref<{ token: string; id: number } | null>(null)
const issuing = ref(false)
const saving = ref(false)

// 从 bot.config（JSON 字符串）解析当前 webhook 配置
const parseCurrentConfig = () => {
  let cfg: any = {}
  try {
    cfg = props.bot?.config ? (typeof props.bot.config === 'string' ? JSON.parse(props.bot.config) : props.bot.config) : {}
  } catch {
    cfg = {}
  }
  form.value = {
    mode: cfg.mode === 'external_webhook' ? 'external_webhook' : 'internal_ai',
    webhook_url: cfg.webhook_url || '',
    webhook_secret: '', // 不回显，留空表示不修改
  }
}

const loadTokens = async () => {
  tokens.value = await listTokens(props.bot.id)
}

onMounted(async () => {
  parseCurrentConfig()
  await loadTokens()
})

const generateSecret = () => {
  const arr = new Uint8Array(32)
  crypto.getRandomValues(arr)
  const hex = Array.from(arr).map(b => b.toString(16).padStart(2, '0')).join('')
  form.value.webhook_secret = hex
  showSecret.value = true
}

const copyToken = async (token: string) => {
  try {
    await navigator.clipboard.writeText(token)
    QMessage.success('已复制')
  } catch {
    QMessage.warning('复制失败，请手动选择复制')
  }
}

const onIssueToken = async () => {
  issuing.value = true
  try {
    const name = window.prompt('令牌名称（可选，如 claude-code）', 'claude-code') || ''
    const resp = await issueToken(props.bot.id, name)
    if (resp) {
      newToken.value = { token: resp.token, id: resp.token_id }
      await loadTokens()
    }
  } finally {
    issuing.value = false
  }
}

const onRevoke = async (t: BotTokenInfo) => {
  if (!confirm(`确定撤销令牌「${t.name || '未命名'}」？撤销后立即失效。`)) return
  try {
    await revokeToken(props.bot.id, t.id)
    await loadTokens()
  } catch (e: any) {
    QMessage.error(e.message || '撤销失败')
  }
}

const onSave = async () => {
  saving.value = true
  try {
    // 只送 webhook 字段；secret 留空时不送（服务端保留既有）
    const payload: BotWebhookConfig = { mode: form.value.mode }
    if (form.value.mode === 'external_webhook') {
      payload.webhook_url = form.value.webhook_url
      if (form.value.webhook_secret) payload.webhook_secret = form.value.webhook_secret
    }
    await updateConfig(props.bot.id, payload)
    QMessage.success('配置已保存')
    emit('saved')
    emit('close')
  } catch (e: any) {
    QMessage.error(e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const formatTime = (s: string) => {
  try {
    const d = new Date(s)
    return d.toLocaleString('zh-CN', { hour12: false })
  } catch {
    return s
  }
}
</script>

<style scoped>
.bot-config-dialog { padding: 8px 4px; display: flex; flex-direction: column; gap: 20px; max-height: 70vh; overflow-y: auto; }
.loading { padding: 40px; text-align: center; color: #888; }
.loading i { margin-right: 8px; }
.config-section { border-top: 1px solid #eee; padding-top: 14px; }
.config-section:first-child { border-top: none; padding-top: 0; }
.config-section h4 { margin: 0 0 10px; font-size: 14px; color: #333; }
.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; }
.mode-options { display: flex; flex-direction: column; gap: 8px; }
.mode-option { display: flex; align-items: flex-start; gap: 10px; padding: 10px 12px; border: 1px solid #e0e0e0; border-radius: 8px; cursor: pointer; transition: all 0.15s; }
.mode-option:hover { border-color: #bbb; }
.mode-option.active { border-color: var(--primary-color, #4f7cff); background: rgba(79, 124, 255, 0.06); }
.mode-option input { margin-top: 3px; }
.mode-text { display: flex; flex-direction: column; gap: 2px; }
.mode-title { font-size: 13px; font-weight: 600; color: #333; }
.mode-desc { font-size: 12px; color: #888; }
.form-field { margin-bottom: 14px; }
.form-field label { display: block; font-size: 13px; color: #555; margin-bottom: 5px; }
.form-field input { width: 100%; padding: 7px 10px; border: 1px solid #ddd; border-radius: 6px; font-size: 13px; box-sizing: border-box; }
.field-hint { font-size: 11px; color: #aaa; margin: 4px 0 0; }
.secret-field { display: flex; gap: 6px; }
.secret-field input { flex: 1; }
.mini-btn { padding: 6px 10px; border: 1px solid #ddd; border-radius: 6px; background: #fafafa; cursor: pointer; font-size: 12px; white-space: nowrap; }
.mini-btn:hover { background: #f0f0f0; }
.new-token-box { display: flex; align-items: center; gap: 10px; padding: 10px 12px; background: #fff8e1; border: 1px solid #ffe082; border-radius: 8px; margin: 10px 0; }
.new-token-box i { color: #f5a623; }
.new-token-text { flex: 1; display: flex; flex-direction: column; gap: 4px; }
.new-token-text code { background: #fff; padding: 4px 8px; border-radius: 4px; font-size: 12px; word-break: break-all; }
.empty-tokens { color: #aaa; font-size: 13px; padding: 12px 0; text-align: center; }
.token-table { width: 100%; border-collapse: collapse; font-size: 12px; }
.token-table th, .token-table td { padding: 7px 8px; text-align: left; border-bottom: 1px solid #f0f0f0; }
.token-table th { color: #888; font-weight: 500; }
.dialog-footer { display: flex; justify-content: flex-end; gap: 10px; padding-top: 10px; border-top: 1px solid #eee; }
.action-btn { padding: 7px 14px; border: 1px solid #ddd; border-radius: 6px; background: #fff; cursor: pointer; font-size: 13px; display: inline-flex; align-items: center; gap: 5px; }
.action-btn:hover { background: #f5f5f5; }
.action-btn.primary { background: var(--primary-color, #4f7cff); color: #fff; border-color: transparent; }
.action-btn.primary:hover { opacity: 0.9; }
.action-btn.danger { color: #e74c3c; border-color: #f5b8b0; }
.action-btn.danger:hover { background: #fdeae7; }
.action-btn.mini { padding: 4px 10px; font-size: 12px; }
</style>

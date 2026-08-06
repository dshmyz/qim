<template>
  <div class="bot-config-dialog">
    <div v-if="loading" class="loading">
      <i class="fas fa-spinner fa-spin"></i><span>加载中...</span>
    </div>

    <template v-else>
      <div v-if="isPending" class="pending-banner">
        <i class="fas fa-clock"></i>
        <span>机器人当前为「{{ bot.approval_status === 'rejected' ? '已拒绝' : '待审批' }}」状态，配置和令牌签发功能暂不可用，需管理员审批通过后方可操作。</span>
      </div>
      <!-- 模型来源 -->
      <section class="config-section">
        <div class="section-header">
          <h4>模型来源</h4>
        </div>
        <div class="model-source">
          <label class="radio-label">
            <input
              type="radio"
              :checked="getUseSystemConfig()"
              @change="setUseSystemConfig(true)"
            />
            <span>使用系统默认模型（推荐）</span>
          </label>
          <label class="radio-label">
            <input
              type="radio"
              :checked="!getUseSystemConfig()"
              @change="setUseSystemConfig(false)"
            />
            <span>使用我的自定义配置</span>
          </label>
        </div>
        <div v-if="!getUseSystemConfig()" class="setting-item">
          <label>选择配置</label>
          <select
            :value="getUserConfigId() || ''"
            @change="setUserConfigId(Number(($event.target as HTMLSelectElement).value) || null)"
            class="form-select"
          >
            <option value="">请选择...</option>
            <option v-for="cfg in myConfigs" :key="cfg.id" :value="cfg.id">
              {{ cfg.config_name }} ({{ cfg.model_name }})
            </option>
          </select>
          <span v-if="myConfigs.length === 0" class="setting-hint error">
            暂无配置，请先在「我的模型配置」中添加
          </span>
        </div>
      </section>

      <!-- 回复模式 + Webhook 配置 + 知识库（统一子组件） -->
      <BotReplyConfigFields
        v-model:mode="form.mode"
        v-model:use-creator-notes="form.use_creator_notes"
        v-model:webhook-url="form.webhook_url"
        v-model:webhook-secret="form.webhook_secret"
        secret-placeholder="••••••（留空不修改）"
        secret-hint="留空则不修改既有密钥（仅写入，服务端不回显）"
        :vector-enabled="vectorEnabled"
      />

      <!-- Token 管理 -->
      <section class="config-section">
        <div class="section-header">
          <h4>访问令牌</h4>
        </div>
        <p class="field-hint">agent 经 qim CLI / Bot API 调用时用此令牌鉴权（Authorization: Bearer）</p>
        <div class="issue-token-row">
          <input v-model="tokenName" class="token-name-input" placeholder="令牌名称（可选）" />
          <button class="action-btn primary" @click="onIssueToken" :disabled="issuing || isPending">
            <i class="fas fa-plus"></i> 签发
          </button>
        </div>

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

      <!-- MCP 接入引导 -->
      <section class="config-section">
        <div class="section-header">
          <h4>MCP 接入（推荐）</h4>
          <button type="button" class="mini-btn" @click="showMcpConfig = !showMcpConfig">
            {{ showMcpConfig ? '收起' : '展开' }}
          </button>
        </div>
        <p class="field-hint">支持 MCP 的 agent（Claude Code / Cursor / Claude Desktop）可免脚本接入，注册后直接调用 QIM 消息工具。</p>
        <div v-if="showMcpConfig" class="mcp-box">
          <div class="mcp-mode-tabs">
            <button :class="['mode-tab', { active: mcpMode === 'stdio' }]" @click="mcpMode = 'stdio'">本地 (stdio)</button>
            <button :class="['mode-tab', { active: mcpMode === 'http' }]" @click="mcpMode = 'http'">远程 (HTTP)</button>
          </div>
          <div class="form-field">
            <label v-if="mcpMode === 'stdio'">QIM 服务端地址</label>
            <label v-else>MCP HTTP 端点</label>
            <input v-if="mcpMode === 'stdio'" v-model="mcpServerUrl" placeholder="http://localhost:8080" />
            <input v-else v-model="mcpHttpUrl" placeholder="http://your-server:8082/mcp" />
          </div>
          <div class="form-field">
            <label>Bot 令牌</label>
            <input v-model="mcpToken" placeholder="qbot_...（粘贴上方刚签发的令牌）" />
          </div>
          <div class="form-field">
            <label>MCP 配置（复制到 agent 的 .mcp.json）</label>
            <pre class="mcp-config-pre">{{ mcpConfigJson }}</pre>
            <button type="button" class="action-btn primary mini" @click="copyMcpConfig">
              <i class="fas fa-copy"></i> 复制配置
            </button>
          </div>
          <p class="field-hint" v-if="mcpMode === 'stdio'">需先安装 <code>qim-mcp</code> 二进制并加入 PATH（由服务端 <code>cmd/qim-mcp</code> 构建）。</p>
          <p class="field-hint" v-else>远程模式：先在服务器启动 <code>qim-mcp --transport http --addr :8082 --server http://localhost:8080</code>，token 经请求头传入，无需在启动命令中暴露。</p>
        </div>
      </section>

      <div class="dialog-footer">
        <button class="action-btn" @click="$emit('close')">取消</button>
        <button class="action-btn primary" @click="onSave" :disabled="saving || isPending">
          <i class="fas fa-save"></i> 保存配置
        </button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useBotConfig } from '../../../composables/useBotConfig'
import { useModelConfigs } from '../../../composables/useModelConfigs'
import { getStoredServerUrl } from '../../../composables/useServerUrl'
import { useSystemConfigStore } from '../../../stores/systemConfig'
import type { BotTokenInfo, BotWebhookConfig } from '../../../types/bot'
import QMessageBox from '../../../utils/qmessagebox'
import BotReplyConfigFields from './BotReplyConfigFields.vue'

const QMessage = (window as any).$QMessage
const props = defineProps<{ bot: any }>()
const emit = defineEmits(['close', 'saved'])

const { loading, issueToken, listTokens, revokeToken, updateConfig } = useBotConfig()
const { configs: myConfigs, fetchConfigs } = useModelConfigs()
const systemConfigStore = useSystemConfigStore()
const vectorEnabled = computed(() => systemConfigStore.vectorEnabled)

const form = ref<BotWebhookConfig>({ mode: 'internal_ai', webhook_url: '', webhook_secret: '', use_creator_notes: false, use_system_config: true, user_config_id: null })
const tokens = ref<BotTokenInfo[]>([])
const newToken = ref<{ token: string; id: number } | null>(null)
const issuing = ref(false)
const saving = ref(false)
const tokenName = ref('claude-code')

const isPending = computed(() => {
  const s = props.bot.approval_status
  return s === 'pending' || s === 'rejected'
})

// MCP 接入引导
const showMcpConfig = ref(false)
const mcpServerUrl = ref(getStoredServerUrl() || 'http://localhost:8080')
const mcpHttpUrl = ref('http://localhost:8082/mcp')
const mcpToken = ref('')
const mcpMode = ref<'stdio' | 'http'>('stdio')
const mcpConfigJson = computed(() => {
  const token = mcpToken.value || 'qbot_REPLACE_ME'
  const server = mcpServerUrl.value || 'http://localhost:8080'
  if (mcpMode.value === 'http') {
    return JSON.stringify({
      mcpServers: {
        qim: {
          type: 'url',
          url: mcpHttpUrl.value || 'http://localhost:8082/mcp',
          headers: { Authorization: 'Bearer ' + token },
        },
      },
    }, null, 2)
  }
  return JSON.stringify({
    mcpServers: {
      qim: {
        command: 'qim-mcp',
        args: ['--server', server, '--token', token],
      },
    },
  }, null, 2)
})
const copyMcpConfig = async () => {
  try {
    await navigator.clipboard.writeText(mcpConfigJson.value)
    QMessage.success('已复制 MCP 配置')
  } catch {
    QMessage.warning('复制失败，请手动选择复制')
  }
}

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
    use_creator_notes: cfg.use_creator_notes === true,
    use_system_config: cfg.use_system_config !== false, // 缺省视为系统默认
    user_config_id: cfg.user_config_id || null,
  }
}

// 模型来源访问器：form 可能为 undefined 字段，统一兜底（模板里用，避免 TypeScript 报错）
function getUseSystemConfig(): boolean {
  return form.value.use_system_config !== false
}
function getUserConfigId(): number | null {
  return form.value.user_config_id || null
}
function setUseSystemConfig(v: boolean) {
  form.value.use_system_config = v
  if (v) form.value.user_config_id = null // 切回系统默认时清空自定义配置引用
}
function setUserConfigId(id: number | null) {
  form.value.user_config_id = id
  if (id != null) form.value.use_system_config = false // 选了配置即视为自定义
}

const loadTokens = async () => {
  tokens.value = await listTokens(props.bot.id)
}

onMounted(async () => {
  parseCurrentConfig()
  await loadTokens()
  await fetchConfigs() // 模型来源下拉：加载我的模型配置
  // 触发公开配置加载（若其他组件已加载过，store 内 loaded 标志会避免重复请求语义，此处仍保证最新）
  if (!systemConfigStore.loaded) {
    systemConfigStore.fetchPublicConfig()
  }
})

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
    const name = tokenName.value.trim() || 'claude-code'
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
  const result = await QMessageBox.confirm(
    `确定撤销令牌「${t.name || '未命名'}」？撤销后立即失效。`,
    '撤销令牌',
    { confirmButtonText: '撤销', type: 'warning' }
  )
  if (result.action !== 'confirm') return
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
    // internal_ai 模式下送 use_creator_notes；external_webhook 模式下也送（保留 false，避免模式切换后状态遗留）
    payload.use_creator_notes = form.value.mode === 'internal_ai' && form.value.use_creator_notes === true
    // 模型来源：始终携带，服务端按合并语义更新（切回系统默认时服务端会清空 user_config_id）
    payload.use_system_config = form.value.use_system_config !== false
    if (!payload.use_system_config) {
      payload.user_config_id = form.value.user_config_id || null
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
.issue-token-row { display: flex; gap: 8px; margin-bottom: 10px; }
.token-name-input { flex: 1; padding: 7px 10px; border: 1px solid #ddd; border-radius: 6px; font-size: 13px; background: #fff; }
.mcp-mode-tabs { display: flex; gap: 0; margin-bottom: 14px; border: 1px solid #ddd; border-radius: 6px; overflow: hidden; }
.mode-tab { flex: 1; padding: 7px 12px; border: none; background: #fff; cursor: pointer; font-size: 13px; color: #666; transition: all 0.15s; }
.mode-tab:not(:last-child) { border-right: 1px solid #ddd; }
.mode-tab.active { background: var(--primary-color, #4f7cff); color: #fff; }
.pending-banner { display: flex; align-items: center; gap: 8px; padding: 10px 14px; margin-bottom: 16px; border-radius: 6px; background: #fff7e6; border: 1px solid #ffd591; color: #ad6800; font-size: 13px; }
.mcp-box { margin-top: 10px; padding: 12px; background: #f9fafc; border: 1px solid #e8ecf3; border-radius: 8px; }
.mcp-config-pre { background: #1e1e2e; color: #cdd6f4; padding: 10px 12px; border-radius: 6px; font-size: 12px; line-height: 1.5; overflow-x: auto; margin: 6px 0 8px; white-space: pre-wrap; word-break: break-all; }
.mcp-config-pre code, .mcp-box code { background: #eef1f6; color: #333; padding: 1px 5px; border-radius: 3px; font-size: 11px; }
.model-source { display: flex; flex-direction: column; gap: 8px; margin-bottom: 10px; }
.radio-label { display: flex; align-items: center; gap: 8px; cursor: pointer; font-size: 13px; color: #555; }
.radio-label input[type="radio"] { cursor: pointer; }
.setting-item { margin-top: 4px; }
.setting-item > label { display: block; margin-bottom: 5px; font-size: 13px; color: #555; }
.form-select { width: 100%; padding: 7px 10px; border: 1px solid #ddd; border-radius: 6px; font-size: 13px; box-sizing: border-box; color: #333; background: #fff; }
.form-select:focus { outline: none; border-color: var(--primary-color, #4f7cff); }
.setting-hint { display: block; margin-top: 4px; font-size: 12px; color: #aaa; }
.setting-hint.error { color: #F44336; }
</style>

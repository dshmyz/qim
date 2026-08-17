<template>
  <div class="developer-settings">
    <div class="settings-section">
      <div class="settings-section-header">
        <h4>开发者令牌</h4>
      </div>
      <p class="field-hint">
        用于 qim CLI / qim-mcp 以本人身份调用用户 API。在 agent 侧通过
        <code>--user-token qusr_…</code> 传入（如 qim-mcp），请求时以
        <code>Authorization: Bearer</code> 携带。令牌仅签发时显示一次，请立即复制保存。
      </p>

      <div class="issue-token-row">
        <input
          v-model="tokenName"
          class="token-name-input"
          placeholder="令牌名称（可选，如 cli / qim-mcp）"
          @keyup.enter="onIssueToken"
        />
        <button class="action-btn primary" @click="onIssueToken" :disabled="issuing || loading">
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
        <button class="mini-btn" @click="copyToken(newToken.token)">
          <i class="fas fa-copy"></i> 复制
        </button>
      </div>

      <div v-if="tokens.length === 0" class="empty-tokens">暂无令牌</div>
      <table v-else class="token-table">
        <thead>
          <tr><th>名称</th><th>签发时间</th><th>最近使用</th><th>操作</th></tr>
        </thead>
        <tbody>
          <tr v-for="t in tokens" :key="t.id">
            <td>{{ t.name || '（未命名）' }}</td>
            <td>{{ formatTime(t.created_at) }}</td>
            <td>{{ t.last_used_at ? formatTime(t.last_used_at) : '—' }}</td>
            <td>
              <button class="action-btn danger mini" @click="onRevoke(t)">撤销</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useUserToken, type UserTokenInfo, type IssuedUserToken } from '../../composables/useUserToken'
import { copyToClipboard } from '../../utils/clipboard'
import QMessage from '../../utils/qmessage'
import QMessageBox from '../../utils/qmessagebox'

const { loading, issueToken, listTokens, revokeToken } = useUserToken()

const tokenName = ref('')
const tokens = ref<UserTokenInfo[]>([])
const newToken = ref<IssuedUserToken | null>(null)
const issuing = ref(false)

const loadTokens = async () => {
  tokens.value = await listTokens()
}

const onIssueToken = async () => {
  const name = tokenName.value.trim()
  issuing.value = true
  try {
    const result = await issueToken(name)
    if (result) {
      newToken.value = result
      tokenName.value = ''
      await loadTokens()
    } else {
      QMessage.error('签发失败，请稍后重试')
    }
  } finally {
    issuing.value = false
  }
}

const copyToken = async (token: string) => {
  const ok = await copyToClipboard(token)
  if (ok) QMessage.success('已复制')
  else QMessage.warning('复制失败，请手动选择复制')
}

const onRevoke = async (t: UserTokenInfo) => {
  const result = await QMessageBox.confirm(
    `确定撤销令牌「${t.name || '未命名'}」吗？撤销后立即失效。`,
    '撤销令牌',
    { type: 'warning', confirmButtonText: '撤销' }
  )
  if (result.action !== 'confirm') return
  // revokeToken 经 safeRequest 吞错返回 null，必须校验结果：
  // 撤销失败（令牌已不存在/服务端异常）时不能显示成功，掩盖令牌实际仍有效。
  const ok = await revokeToken(t.id)
  if (ok) {
    QMessage.success('令牌已撤销')
  } else {
    QMessage.error('撤销失败，令牌可能仍有效，请稍后重试')
  }
  await loadTokens()
}

const formatTime = (s: string) => {
  if (!s) return '—'
  return new Date(s).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

onMounted(loadTokens)
</script>

<style scoped>
.field-hint {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
  line-height: 1.6;
  margin: 0 0 12px;
}

.field-hint code {
  background: var(--hover-color);
  border-radius: 4px;
  padding: 1px 5px;
  font-size: var(--font-size-xxs);
}

.issue-token-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 12px;
}

.token-name-input {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--bg-color);
  color: var(--text-primary);
  font-size: var(--font-size-sm);
  font-family: inherit;
  outline: none;
}

.token-name-input:focus {
  border-color: var(--primary-color);
}

.action-btn {
  border: 1px solid var(--border-color);
  background: transparent;
  color: var(--text-primary);
  border-radius: 8px;
  padding: 7px 14px;
  font-size: var(--font-size-sm);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s;
  font-family: inherit;
}

.action-btn.primary {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: #fff;
}

.action-btn.danger {
  color: #d32f2f;
  border-color: #d32f2f;
}

.action-btn.mini {
  padding: 2px 10px;
  font-size: var(--font-size-xxs);
}

.action-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.new-token-box {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border: 1px solid #ffcdd2;
  border-radius: 8px;
  background: rgba(255, 235, 238, 0.6);
  color: #b71c1c;
  font-size: var(--font-size-xs);
  margin-bottom: 12px;
}

.new-token-text {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.new-token-text code {
  font-size: var(--font-size-xxs);
  word-break: break-all;
  color: var(--text-primary);
}

.mini-btn {
  border: 1px solid var(--border-color);
  background: #fff;
  color: var(--text-primary);
  border-radius: 6px;
  padding: 4px 10px;
  font-size: var(--font-size-xxs);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.empty-tokens {
  padding: 16px;
  text-align: center;
  color: var(--text-secondary);
  font-size: var(--font-size-xs);
  border: 1px dashed var(--border-color);
  border-radius: 8px;
}

.token-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--font-size-sm);
}

.token-table th,
.token-table td {
  text-align: left;
  padding: 8px 10px;
  border-bottom: 1px solid var(--border-color);
  font-size: var(--font-size-xs);
}

.token-table th {
  color: var(--text-secondary);
  font-weight: 500;
}
</style>
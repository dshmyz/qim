<template>
  <!-- 回复模式 -->
  <section class="config-section">
    <h4>回复模式</h4>
    <div class="mode-options">
      <label class="mode-option" :class="{ active: mode === 'internal_ai' }">
        <input type="radio" :value="'internal_ai'" v-model="mode" />
        <div class="mode-text">
          <span class="mode-title">内置 AI</span>
          <span class="mode-desc">用户回复走 QIM 内置大模型，无需额外配置</span>
        </div>
      </label>
      <label class="mode-option" :class="{ active: mode === 'external_webhook' }">
        <input type="radio" :value="'external_webhook'" v-model="mode" />
        <div class="mode-text">
          <span class="mode-title">外部 Agent（Webhook）</span>
          <span class="mode-desc">用户回复转发到外部 agent（如 Claude Code），agent 经 Bot API 回复</span>
        </div>
      </label>
    </div>
  </section>

  <!-- Webhook 配置（仅 external 模式） -->
  <section v-if="mode === 'external_webhook'" class="config-section">
    <h4>Webhook 配置</h4>
    <div class="form-field">
      <label>回调地址</label>
      <input v-model="webhookUrl" placeholder="https://your-agent.example/qim-webhook" />
      <p v-if="webhookUrl" class="field-hint">QIM 把用户回复 POST 到此地址（HMAC-SHA256 签名），agent 收到后可即时回复</p>
      <p v-else class="field-hint field-hint-warn">
        <span class="warn-badge">pull 模式</span>
        未填回调地址：QIM 不会投递，机器人被 @ 不会自动回复。需用 CLI/MCP 主动轮询
        <code>GET /bot/messages</code> 并由你的 agent 进程回发；否则对用户表现为「无反应」。
      </p>
    </div>
    <div class="form-field">
      <label>签名密钥</label>
      <div class="secret-field">
        <input
          :type="showSecret ? 'text' : 'password'"
          v-model="webhookSecret"
          :placeholder="secretPlaceholder"
        />
        <button type="button" class="mini-btn" @click="generateSecret">生成随机</button>
        <button type="button" class="mini-btn" @click="showSecret = !showSecret">
          {{ showSecret ? '隐藏' : '显示' }}
        </button>
      </div>
      <p class="field-hint">{{ secretHint }}</p>
    </div>
  </section>

  <!-- 知识库配置（仅 internal_ai 模式） -->
  <section v-if="mode === 'internal_ai'" class="config-section">
    <h4>知识库</h4>
    <label class="mode-option" :class="{ active: useCreatorNotes }">
      <input type="checkbox" v-model="useCreatorNotes" :disabled="!vectorEnabled" />
      <div class="mode-text">
        <span class="mode-title">读取我的笔记作为知识库</span>
        <span class="mode-desc">机器人回答时按相关性检索你本人的笔记并注入上下文，仅可读你自己的笔记（不会泄露给他人）</span>
      </div>
    </label>
    <p v-if="vectorEnabled" class="field-hint">需服务端已配置向量库；未配置时开关无效，机器人将退回纯 prompt 模式</p>
    <p v-else class="field-hint field-hint-warn">
      <span class="warn-badge">未连接</span>
      服务端未配置向量库，本开关无效。请联系管理员或自托管部署时启用向量服务。
    </p>
  </section>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

// 用 defineModel 双向绑定 4 个字段，父组件用 v-model:xxx 即可
const mode = defineModel<'internal_ai' | 'external_webhook'>('mode', { default: 'internal_ai' })
const useCreatorNotes = defineModel<boolean>('useCreatorNotes', { default: false })
const webhookUrl = defineModel<string>('webhookUrl', { default: '' })
const webhookSecret = defineModel<string>('webhookSecret', { default: '' })

// secret 显示控制自包含在子组件
const showSecret = ref(false)

// 是否为编辑场景（编辑时留空表示不修改，创建时留空表示不设置）
const props = defineProps<{
  // secretPlaceholder: 编辑场景显示"留空不修改"，创建场景显示"留空不设置"
  secretPlaceholder?: string
  secretHint?: string
  // 向量库是否可用（由父组件从 /api/v1/system/public-config 查询后传入）
  // false 时禁用「读取我的笔记」checkbox 并显示警告徽章
  vectorEnabled?: boolean
}>()

// 用 computed 保持响应性：父组件异步加载 store 后能更新
// 默认 true 避免加载完成前误显示「未连接」警告
const vectorEnabled = computed(() => props.vectorEnabled ?? true)
const secretPlaceholder = computed(() => props.secretPlaceholder || '••••••（留空不设置）')
const secretHint = computed(() => props.secretHint || '留空则不设置密钥（仅写入，服务端不回显）')

const generateSecret = () => {
  const arr = new Uint8Array(32)
  crypto.getRandomValues(arr)
  const hex = Array.from(arr).map(b => b.toString(16).padStart(2, '0')).join('')
  webhookSecret.value = hex
  showSecret.value = true
}
</script>

<style scoped>
.config-section { border-top: 1px solid #eee; padding-top: 14px; }
.config-section:first-child { border-top: none; padding-top: 0; }
.config-section h4 { margin: 0 0 10px; font-size: var(--font-size-sm); color: var(--text-color, #333); }
.mode-options { display: flex; flex-direction: column; gap: 8px; }
.mode-option { display: flex; align-items: flex-start; gap: 10px; padding: 10px 12px; border: 1px solid #e0e0e0; border-radius: 8px; cursor: pointer; transition: all 0.15s; }
.mode-option:hover { border-color: #bbb; }
.mode-option.active { border-color: var(--primary-color, #4f7cff); background: rgba(79, 124, 255, 0.06); }
.mode-option input { margin-top: 3px; }
.mode-text { display: flex; flex-direction: column; gap: 2px; }
.mode-title { font-size: var(--font-size-xs); font-weight: 600; color: var(--text-color, #333); }
.mode-desc { font-size: var(--font-size-xxs); color: #888; }
.form-field { margin-bottom: 14px; }
.form-field label { display: block; font-size: var(--font-size-xs); color: #555; margin-bottom: 5px; }
.form-field input { width: 100%; padding: 7px 10px; border: 1px solid #ddd; border-radius: 6px; font-size: var(--font-size-xs); box-sizing: border-box; }
.field-hint { font-size: var(--font-size-xxxs); color: #aaa; margin: 4px 0 0; }
.field-hint-warn { color: #b88200; }
.warn-badge {
  display: inline-block;
  padding: 1px 6px;
  background: #fff3cd;
  color: #856404;
  border: 1px solid #ffe08a;
  border-radius: 4px;
  font-size: var(--font-size-tiny);
  font-weight: 600;
  margin-right: 4px;
}
.secret-field { display: flex; gap: 6px; }
.secret-field input { flex: 1; }
.mini-btn { padding: 6px 10px; border: 1px solid #ddd; border-radius: 6px; background: #fafafa; cursor: pointer; font-size: var(--font-size-xxs); white-space: nowrap; }
.mini-btn:hover { background: #f0f0f0; }
</style>

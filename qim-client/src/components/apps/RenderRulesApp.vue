<template>
  <div class="render-rules-app">
    <AppHeader title="渲染规则管理" icon="fas fa-magic" @back="$emit('back')">
      <template #actions>
        <button class="create-rule-btn" @click="openCreate">
          <i class="fas fa-plus"></i>
          <span>新建规则</span>
        </button>
      </template>
    </AppHeader>

    <div class="rr-content">
      <div v-if="loading" class="rr-loading">
        <i class="fas fa-spinner fa-spin"></i>
        <span>加载中...</span>
      </div>

      <template v-else>
        <!-- 规则列表 -->
        <div v-if="rules.length > 0" class="rr-list">
          <div class="rr-list-header">
            <div class="rr-col rr-col-name">规则名称</div>
            <div class="rr-col rr-col-status">状态</div>
            <div class="rr-col rr-col-priority">优先级</div>
            <div class="rr-col rr-col-pattern">匹配模式</div>
            <div class="rr-col rr-col-type">渲染类型</div>
            <div class="rr-col rr-col-actions">操作</div>
          </div>
          <div v-for="rule in sortedRules" :key="rule.id" class="rr-list-item">
            <div class="rr-col rr-col-name">
              <span class="rr-rule-name">{{ rule.name }}</span>
              <span class="rr-rule-id">{{ rule.id }}</span>
            </div>
            <div class="rr-col rr-col-status">
              <Switch
                :model-value="rule.enabled"
                size="small"
                @update:modelValue="(v: boolean) => toggleEnabled(rule.id, v)"
              />
            </div>
            <div class="rr-col rr-col-priority">{{ rule.priority }}</div>
            <div class="rr-col rr-col-pattern">
              <code class="rr-pattern-code">{{ rule.match.pattern }}</code>
            </div>
            <div class="rr-col rr-col-type">
              <span class="rr-type-badge" :class="`rr-type-${rule.render.type}`">
                {{ renderTypeLabel(rule.render.type) }}
              </span>
            </div>
            <div class="rr-col rr-col-actions">
              <button class="rr-action-btn" title="测试" @click="openTest(rule)">
                <i class="fas fa-vial"></i>
              </button>
              <button class="rr-action-btn" title="编辑" @click="openEdit(rule)">
                <i class="fas fa-edit"></i>
              </button>
              <button class="rr-action-btn rr-action-danger" title="删除" @click="confirmDelete(rule)">
                <i class="fas fa-trash"></i>
              </button>
            </div>
          </div>
        </div>

        <!-- 空状态 -->
        <div v-else class="rr-empty">
          <i class="fas fa-magic rr-empty-icon"></i>
          <p class="rr-empty-text">暂无渲染规则</p>
          <p class="rr-empty-hint">创建规则后，消息中的特定文本会自动渲染为卡片或链接</p>
          <button class="create-rule-btn" @click="openCreate">
            <i class="fas fa-plus"></i>
            <span>新建第一条规则</span>
          </button>
        </div>
      </template>
    </div>

    <!-- 编辑弹窗 -->
    <ModalContainer
      :visible="showEditor"
      :title="editingRule ? '编辑规则' : '新建规则'"
      width="720px"
      max-height="85vh"
      @close="closeEditor"
    >
      <RenderRuleEditor
        :rule="editingRule"
        @save="handleSave"
        @cancel="closeEditor"
        @error="handleValidationError"
      />
    </ModalContainer>

    <!-- 测试弹窗 -->
    <ModalContainer
      :visible="showTester"
      :title="`测试规则：${testingRule?.name || ''}`"
      width="600px"
      @close="closeTester"
    >
      <RenderRuleTester :rule="testingRule" />
    </ModalContainer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import AppHeader from '../apps/AppHeader.vue'
import Switch from '../common/Switch.vue'
import ModalContainer from '../shared/ModalContainer.vue'
import RenderRuleEditor from './render-rules/RenderRuleEditor.vue'
import RenderRuleTester from './render-rules/RenderRuleTester.vue'
import QMessage from '../../utils/qmessage'
import QMessageBox from '../../utils/qmessagebox'
import type { RenderRule, CompiledRule } from '../../stores/renderRules'
import { fetchAllRenderRules, saveRenderRules } from '../../api/renderRules'
import { detectRuleConflicts } from '../../utils/ruleConflictDetector'

defineEmits<{
  back: []
  toggleSidebar: []
}>()

const rules = ref<RenderRule[]>([])
const loading = ref(true)
const saving = ref(false)

// 按优先级排序
const sortedRules = computed(() =>
  [...rules.value].sort((a, b) => a.priority - b.priority)
)

// 编辑器状态
const showEditor = ref(false)
const editingRule = ref<RenderRule | null>(null)

// 测试器状态
const showTester = ref(false)
const testingRule = ref<RenderRule | null>(null)

onMounted(async () => {
  await loadRules()
})

async function loadRules() {
  loading.value = true
  try {
    const { rules: list } = await fetchAllRenderRules()
    rules.value = list
  } catch (e) {
    QMessage.error(e instanceof Error ? e.message : '加载规则失败')
  } finally {
    loading.value = false
  }
}

function renderTypeLabel(type: string): string {
  switch (type) {
    case 'link_card': return '卡片'
    case 'link': return '链接'
    case 'text_chip': return '标签'
    default: return type
  }
}

// 新建
function openCreate() {
  editingRule.value = null
  showEditor.value = true
}

// 编辑
function openEdit(rule: RenderRule) {
  editingRule.value = { ...rule }
  showEditor.value = true
}

function closeEditor() {
  showEditor.value = false
  editingRule.value = null
}

// 保存（新建/编辑统一处理，后端批量覆盖）
// 保存前检测规则冲突，有冲突时提示用户确认
async function handleSave(rule: RenderRule) {
  if (saving.value) return
  saving.value = true

  try {
    // 先更新本地规则列表（临时），用于冲突检测
    const candidateRules = [...rules.value]
    const index = candidateRules.findIndex(r => r.id === rule.id)
    if (index >= 0) {
      candidateRules.splice(index, 1, rule)
    } else {
      candidateRules.push(rule)
    }

    // 检测冲突（只检测启用的规则）
    const enabledRules = candidateRules
      .filter(r => r.enabled)
      .map(r => ({
        ...r,
        compiledRegex: new RegExp(r.match.pattern, r.match.flags || 'g'),
      })) as CompiledRule[]
    const conflicts = detectRuleConflicts(enabledRules)

    // 有冲突时弹窗确认
    if (conflicts.length > 0) {
      const conflictList = conflicts.map(c => `• ${c.description}`).join('\n')
      const result = await QMessageBox.confirm(
        `检测到 ${conflicts.length} 条潜在冲突：\n\n${conflictList}\n\n仍要保存吗？`,
        '规则冲突警告',
        { confirmButtonText: '仍要保存', type: 'warning' }
      )
      if (result.action !== 'confirm') {
        saving.value = false
        return
      }
    }

    // 确认保存
    rules.value = candidateRules
    await saveRenderRules(rules.value)
    QMessage.success(editingRule.value ? '规则已更新' : '规则已创建')
    closeEditor()
  } catch (e) {
    QMessage.error(e instanceof Error ? e.message : '保存失败')
    // 保存失败时重新加载，避免本地状态与服务端不一致
    await loadRules()
  } finally {
    saving.value = false
  }
}

function handleValidationError(msg: string) {
  QMessage.warning(msg)
}

// 启停切换
async function toggleEnabled(ruleId: string, enabled: boolean) {
  const rule = rules.value.find(r => r.id === ruleId)
  if (!rule) return

  const oldEnabled = rule.enabled
  rule.enabled = enabled

  try {
    await saveRenderRules(rules.value)
    QMessage.success(enabled ? '规则已启用' : '规则已禁用')
  } catch (e) {
    // 回滚
    rule.enabled = oldEnabled
    QMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}

// 删除
async function confirmDelete(rule: RenderRule) {
  const result = await QMessageBox.confirm(
    `确定要删除规则「${rule.name}」吗？删除后不可恢复。`,
    '删除规则',
    { confirmButtonText: '删除', type: 'warning' }
  )
  if (result.action !== 'confirm') return

  try {
    rules.value = rules.value.filter(r => r.id !== rule.id)
    await saveRenderRules(rules.value)
    QMessage.success('规则已删除')
  } catch (e) {
    QMessage.error(e instanceof Error ? e.message : '删除失败')
    await loadRules()
  }
}

// 测试
function openTest(rule: RenderRule) {
  testingRule.value = { ...rule }
  showTester.value = true
}

function closeTester() {
  showTester.value = false
  testingRule.value = null
}
</script>

<style scoped>
.render-rules-app {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-color);
}
.rr-content {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}
.rr-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 60px 0;
  color: var(--text-secondary);
  font-size: 14px;
}

/* 列表 */
.rr-list {
  background: var(--card-bg);
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--border-color);
}
.rr-list-header,
.rr-list-item {
  display: grid;
  grid-template-columns: 2fr 80px 70px 2.5fr 90px 120px;
  gap: 12px;
  padding: 12px 16px;
  align-items: center;
}
.rr-list-header {
  background: var(--hover-color);
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.rr-list-item {
  border-top: 1px solid var(--border-color);
  font-size: 13px;
  color: var(--text-color);
  transition: background 0.15s;
}
.rr-list-item:hover {
  background: var(--hover-color);
}
.rr-col-name {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.rr-rule-name {
  font-weight: 500;
  color: var(--text-color);
}
.rr-rule-id {
  font-size: 11px;
  color: var(--text-secondary);
  font-family: 'SF Mono', monospace;
}
.rr-pattern-code {
  font-family: 'SF Mono', 'Consolas', monospace;
  font-size: 12px;
  color: #e6a23c;
  background: rgba(230, 162, 60, 0.08);
  padding: 2px 6px;
  border-radius: 4px;
  word-break: break-all;
}
.rr-type-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 500;
}
.rr-type-link_card {
  background: rgba(103, 194, 58, 0.1);
  color: #67c23a;
}
.rr-type-link {
  background: rgba(64, 158, 255, 0.1);
  color: #409eff;
}
.rr-type-text_chip {
  background: rgba(144, 147, 153, 0.1);
  color: #909399;
}
.rr-col-actions {
  display: flex;
  gap: 4px;
}
.rr-action-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: none;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  color: var(--text-secondary);
  transition: all 0.15s;
}
.rr-action-btn:hover {
  background: var(--primary-light);
  color: var(--primary-color);
}
.rr-action-danger:hover {
  background: rgba(245, 108, 108, 0.1);
  color: #f56c6c;
}

/* 空状态 */
.rr-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 20px;
  gap: 8px;
}
.rr-empty-icon {
  font-size: 48px;
  color: var(--text-secondary);
  opacity: 0.4;
  margin-bottom: 8px;
}
.rr-empty-text {
  font-size: 16px;
  font-weight: 500;
  color: var(--text-color);
  margin: 0;
}
.rr-empty-hint {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0 0 16px 0;
}

/* 新建按钮 */
.create-rule-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 14px;
  height: 32px;
  background: var(--primary-color);
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}
.create-rule-btn:hover {
  background: var(--active-color);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}
</style>

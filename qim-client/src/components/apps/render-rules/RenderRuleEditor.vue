<template>
  <div class="render-rule-editor">
    <!-- 基本信息 -->
    <section class="rr-section">
      <h4 class="rr-section-title">基本信息</h4>
      <div class="rr-form-row">
        <div class="rr-form-group">
          <label class="rr-label">规则名称 <span class="rr-required">*</span></label>
          <input v-model="form.name" type="text" class="rr-input" placeholder="如：Jira 工单卡片化" />
        </div>
        <div class="rr-form-group rr-form-group-sm">
          <label class="rr-label">规则 ID <span class="rr-required">*</span></label>
          <input
            v-model="form.id"
            type="text"
            class="rr-input"
            placeholder="如：jira_ticket"
            :disabled="!!originalId"
          />
          <p class="rr-hint" v-if="originalId">ID 创建后不可修改</p>
          <p class="rr-hint" v-else>唯一标识，用英文小写+下划线</p>
        </div>
        <div class="rr-form-group rr-form-group-sm">
          <label class="rr-label">优先级</label>
          <input v-model.number="form.priority" type="number" min="0" class="rr-input" />
          <p class="rr-hint">数字越小越先执行</p>
        </div>
      </div>
    </section>

    <!-- 匹配规则 -->
    <section class="rr-section">
      <h4 class="rr-section-title">匹配规则</h4>
      <div class="rr-form-row">
        <div class="rr-form-group rr-form-group-lg">
          <label class="rr-label">正则表达式 <span class="rr-required">*</span></label>
          <input v-model="form.match.pattern" type="text" class="rr-input rr-input-mono" placeholder="如：\b([A-Z]{2,6})-(\d{1,6})\b" />
        </div>
        <div class="rr-form-group rr-form-group-xs">
          <label class="rr-label">标志位</label>
          <input v-model="form.match.flags" type="text" class="rr-input" placeholder="g" />
        </div>
      </div>
      <div class="rr-form-group">
        <label class="rr-label">捕获组映射</label>
        <CaptureGroupsEditor v-model="form.match.capture_groups" />
      </div>
    </section>

    <!-- 渲染配置 -->
    <section class="rr-section">
      <h4 class="rr-section-title">渲染配置</h4>
      <div class="rr-form-row">
        <div class="rr-form-group rr-form-group-sm">
          <label class="rr-label">渲染类型</label>
          <select v-model="form.render.type" class="rr-input">
            <option value="link_card">链接卡片（带图标）</option>
            <option value="link">普通链接</option>
            <option value="text_chip">文本标签</option>
          </select>
        </div>
        <div class="rr-form-group rr-form-group-sm">
          <label class="rr-label">打开方式</label>
          <select v-model="form.render.target" class="rr-input">
            <option value="_blank">新窗口</option>
            <option value="_self">当前窗口</option>
          </select>
        </div>
        <div class="rr-form-group rr-form-group-sm">
          <label class="rr-label">图标类名</label>
          <input v-model="form.render.icon" type="text" class="rr-input" placeholder="如：fab fa-jira" />
        </div>
      </div>
      <div class="rr-form-group">
        <label class="rr-label">URL 模板 <span class="rr-required" v-if="form.render.type !== 'text_chip'">*</span></label>
        <input v-model="form.render.url_template" type="text" class="rr-input rr-input-mono" placeholder="如：http://jira.xxx.com/{{project}}/{{project}}-{{number}}" />
      </div>
      <div class="rr-form-row">
        <div class="rr-form-group">
          <label class="rr-label">标签模板 <span class="rr-required">*</span></label>
          <input v-model="form.render.label_template" type="text" class="rr-input rr-input-mono" placeholder="如：{{project}}-{{number}}" />
        </div>
        <div class="rr-form-group">
          <label class="rr-label">标题模板（可选）</label>
          <input v-model="form.render.title_template" type="text" class="rr-input" placeholder="如：查看 Jira 工单 {{project}}-{{number}}" />
        </div>
      </div>
      <div class="rr-form-group">
        <label class="rr-label">CSS 类名（可选）</label>
        <input v-model="form.render.class" type="text" class="rr-input" placeholder="如：jira-ticket-card" />
      </div>
    </section>

    <!-- 作用域 -->
    <section class="rr-section">
      <h4 class="rr-section-title">作用域</h4>
      <div class="rr-form-group">
        <label class="rr-label">适用的会话类型</label>
        <div class="rr-checkbox-group">
          <label class="rr-checkbox">
            <input type="checkbox" value="single" v-model="convTypes" />
            <span>单聊</span>
          </label>
          <label class="rr-checkbox">
            <input type="checkbox" value="group" v-model="convTypes" />
            <span>群聊</span>
          </label>
          <label class="rr-checkbox">
            <input type="checkbox" value="discussion" v-model="convTypes" />
            <span>讨论组</span>
          </label>
        </div>
      </div>
      <div class="rr-form-row">
        <div class="rr-form-group">
          <label class="rr-label">群组白名单</label>
          <input v-model="groupsText" type="text" class="rr-input" placeholder="* 表示全部，多个用逗号分隔" />
          <p class="rr-hint">只有这些群组会应用规则</p>
        </div>
        <div class="rr-form-group">
          <label class="rr-label">群组黑名单</label>
          <input v-model="excludeGroupsText" type="text" class="rr-input" placeholder="多个用逗号分隔" />
          <p class="rr-hint">这些群组不应用规则</p>
        </div>
      </div>
    </section>

    <!-- 操作按钮 -->
    <div class="rr-editor-actions">
      <button type="button" class="rr-btn rr-btn-default" @click="$emit('cancel')">取消</button>
      <button type="button" class="rr-btn rr-btn-primary" @click="handleSave">保存</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, computed } from 'vue'
import type { RenderRule } from '../../../stores/renderRules'
import CaptureGroupsEditor from './CaptureGroupsEditor.vue'

const props = defineProps<{
  rule: RenderRule | null
}>()

const emit = defineEmits<{
  save: [rule: RenderRule]
  cancel: []
  error: [message: string]
}>()

const originalId = ref<string>('')

// 表单数据（深拷贝，避免直接修改父组件数据）
const form = reactive<RenderRule>(createEmptyForm())

// 会话类型用独立数组绑定 checkbox
const convTypes = ref<string[]>([])
// 群组白名单/黑名单用逗号分隔文本简化输入
const groupsText = ref('')
const excludeGroupsText = ref('')

function createEmptyForm(): RenderRule {
  return {
    id: '',
    name: '',
    enabled: false,
    priority: 10,
    scope: { groups: ['*'], exclude_groups: [], conversation_types: ['single', 'group', 'discussion'] },
    match: { pattern: '', flags: 'g', capture_groups: {} },
    render: { type: 'link_card', url_template: '', label_template: '', target: '_blank' }
  }
}

// 从 props.rule 同步表单数据
function syncFromProp() {
  const source = props.rule
  if (!source) {
    Object.assign(form, createEmptyForm())
    originalId.value = ''
  } else {
    originalId.value = source.id
    form.id = source.id
    form.name = source.name
    form.enabled = source.enabled
    form.priority = source.priority
    form.scope = {
      groups: [...(source.scope.groups || [])],
      exclude_groups: [...(source.scope.exclude_groups || [])],
      conversation_types: [...(source.scope.conversation_types || [])]
    }
    form.match = {
      pattern: source.match.pattern,
      flags: source.match.flags || 'g',
      capture_groups: { ...(source.match.capture_groups || {}) }
    }
    form.render = {
      type: source.render.type,
      url_template: source.render.url_template || '',
      label_template: source.render.label_template || '',
      title_template: source.render.title_template || '',
      icon: source.render.icon || '',
      target: source.render.target || '_blank',
      class: source.render.class || ''
    }
  }
  convTypes.value = [...(form.scope.conversation_types || [])]
  groupsText.value = (form.scope.groups || []).join(', ')
  excludeGroupsText.value = (form.scope.exclude_groups || []).join(', ')
}

watch(() => props.rule, syncFromProp, { immediate: true })

function handleSave() {
  // 校验必填字段
  if (!form.id.trim()) {
    emitError('请填写规则 ID')
    return
  }
  if (!form.name.trim()) {
    emitError('请填写规则名称')
    return
  }
  if (!form.match.pattern.trim()) {
    emitError('请填写正则表达式')
    return
  }
  if (!form.render.label_template.trim()) {
    emitError('请填写标签模板')
    return
  }
  if (form.render.type !== 'text_chip' && !form.render.url_template.trim()) {
    emitError('非文本标签类型必须填写 URL 模板')
    return
  }

  // 逗号分隔文本转数组
  form.scope.groups = parseCsv(groupsText.value)
  form.scope.exclude_groups = parseCsv(excludeGroupsText.value)
  form.scope.conversation_types = [...convTypes.value]

  // 清理空的可选字段
  const result: RenderRule = {
    id: form.id.trim(),
    name: form.name.trim(),
    enabled: form.enabled,
    priority: form.priority,
    scope: { ...form.scope },
    match: { ...form.match },
    render: { ...form.render }
  }
  // 移除空字符串的可选字段
  if (!result.render.title_template) delete result.render.title_template
  if (!result.render.icon) delete result.render.icon
  if (!result.render.class) delete result.render.class

  emit('save', result)
}

function emitError(msg: string) {
  emit('error', msg)
}

function parseCsv(text: string): string[] {
  return text
    .split(',')
    .map(s => s.trim())
    .filter(s => s.length > 0)
}
</script>

<style scoped>
.render-rule-editor {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.rr-section {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 16px;
  background: #fafbfc;
}
.rr-section-title {
  margin: 0 0 14px 0;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  padding-bottom: 8px;
  border-bottom: 1px solid #ebeef5;
}
.rr-form-row {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
}
.rr-form-group {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.rr-form-group-sm { flex: 0 0 180px; }
.rr-form-group-xs { flex: 0 0 80px; }
.rr-form-group-lg { flex: 2; }
.rr-label {
  font-size: 13px;
  color: #606266;
  font-weight: 500;
}
.rr-required {
  color: #f56c6c;
}
.rr-hint {
  font-size: 12px;
  color: #909399;
  margin: 2px 0 0 0;
}
.rr-input {
  width: 100%;
  padding: 8px 10px;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  font-size: 13px;
  color: #303133;
  background: #fff;
  transition: border-color 0.2s;
  box-sizing: border-box;
}
.rr-input:focus {
  outline: none;
  border-color: #409eff;
}
.rr-input:disabled {
  background: #f5f7fa;
  color: #909399;
  cursor: not-allowed;
}
.rr-input-mono {
  font-family: 'SF Mono', 'Consolas', 'Monaco', monospace;
  font-size: 12px;
}
.rr-checkbox-group {
  display: flex;
  gap: 16px;
}
.rr-checkbox {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  font-size: 13px;
  color: #606266;
}
.rr-checkbox input {
  cursor: pointer;
}
.rr-editor-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding-top: 12px;
  border-top: 1px solid #ebeef5;
}
.rr-btn {
  padding: 8px 18px;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  border: 1px solid transparent;
  transition: all 0.2s;
}
.rr-btn-primary {
  background: #409eff;
  color: #fff;
}
.rr-btn-primary:hover {
  background: #337ecc;
}
.rr-btn-default {
  background: #fff;
  color: #606266;
  border-color: #dcdfe6;
}
.rr-btn-default:hover {
  color: #409eff;
  border-color: #c6e2ff;
}
.rr-btn-text {
  background: none;
  border: none;
  color: #409eff;
  padding: 4px 8px;
  font-size: 12px;
}
.rr-btn-text:hover {
  color: #337ecc;
}
</style>

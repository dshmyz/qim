<template>
  <QDialog
    :visible="visible"
    title="导出短链接"
    width="450px"
    :close-on-click-mask="false"
    @update:visible="$emit('update:visible', $event)"
    @close="handleClose"
  >
    <div class="export-content">
      <!-- 导出格式选择 -->
      <div class="form-group">
        <label class="form-label">导出格式</label>
        <div class="format-options">
          <label
            v-for="format in formatOptions"
            :key="format.value"
            :class="['format-option', { active: selectedFormat === format.value }]"
          >
            <input
              v-model="selectedFormat"
              type="radio"
              :value="format.value"
            />
            <div class="format-info">
              <i :class="format.icon"></i>
              <span class="format-name">{{ format.label }}</span>
              <span class="format-desc">{{ format.description }}</span>
            </div>
          </label>
        </div>
      </div>

      <!-- 导出范围 -->
      <div class="form-group">
        <label class="form-label">导出范围</label>
        <div class="scope-options">
          <label class="radio-option">
            <input
              v-model="exportScope"
              type="radio"
              value="all"
            />
            <span class="radio-label">全部短链接 ({{ totalCount }} 条)</span>
          </label>
          <label v-if="selectedCount > 0" class="radio-option">
            <input
              v-model="exportScope"
              type="radio"
              value="selected"
            />
            <span class="radio-label">已选中的 ({{ selectedCount }} 条)</span>
          </label>
        </div>
      </div>

      <!-- 导出字段 -->
      <div class="form-group">
        <label class="form-label">导出字段</label>
        <div class="field-options">
          <label
            v-for="field in fieldOptions"
            :key="field.value"
            class="checkbox-option"
          >
            <input
              v-model="selectedFields"
              type="checkbox"
              :value="field.value"
            />
            <span class="checkbox-label">{{ field.label }}</span>
          </label>
        </div>
      </div>

      <!-- 预览 -->
      <div class="export-preview">
        <div class="preview-header">
          <span>预览</span>
          <span class="preview-count">将导出 {{ exportCount }} 条记录</span>
        </div>
        <div class="preview-content">
          <code>{{ previewText }}</code>
        </div>
      </div>
    </div>

    <template #footer>
      <button class="q-btn q-btn--default" @click="handleClose">取消</button>
      <button
        class="q-btn q-btn--primary"
        :disabled="selectedFields.length === 0 || exportCount === 0"
        @click="handleExport"
      >
        <i class="fas fa-download"></i> 导出
      </button>
    </template>
  </QDialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import QDialog from '../../shared/QDialog.vue'
import QMessage from '../../../utils/qmessage'
import type { ShortLink } from './ShortLinkItem.vue'

interface Props {
  visible: boolean
  links: ShortLink[]
  selectedIds?: number[]
}

interface Emits {
  (e: 'update:visible', value: boolean): void
}

const props = withDefaults(defineProps<Props>(), {
  selectedIds: () => []
})

const emit = defineEmits<Emits>()

const selectedFormat = ref<'csv' | 'json'>('csv')
const exportScope = ref<'all' | 'selected'>('all')
const selectedFields = ref<string[]>(['original_url', 'short_url', 'visit_count', 'created_at'])

const formatOptions = [
  {
    value: 'csv',
    label: 'CSV',
    icon: 'fas fa-file-csv',
    description: '适合Excel打开'
  },
  {
    value: 'json',
    label: 'JSON',
    icon: 'fas fa-file-code',
    description: '适合程序处理'
  }
]

const fieldOptions = [
  { value: 'id', label: 'ID' },
  { value: 'original_url', label: '原始URL' },
  { value: 'short_url', label: '短链接' },
  { value: 'visit_count', label: '访问次数' },
  { value: 'created_at', label: '创建时间' }
]

// 总数
const totalCount = computed(() => props.links.length)

// 选中数量
const selectedCount = computed(() => props.selectedIds.length)

// 导出数量
const exportCount = computed(() => {
  if (exportScope.value === 'selected') {
    return selectedCount.value
  }
  return totalCount.value
})

// 要导出的链接
const linksToExport = computed(() => {
  if (exportScope.value === 'selected' && props.selectedIds.length > 0) {
    return props.links.filter(link => props.selectedIds.includes(link.id))
  }
  return props.links
})

// 预览文本
const previewText = computed(() => {
  if (selectedFormat.value === 'csv') {
    const headers = selectedFields.value.map(f => fieldOptions.find(fo => fo.value === f)?.label || f)
    return headers.join(', ')
  } else {
    const sample: Record<string, unknown> = {}
    selectedFields.value.forEach(f => {
      sample[f] = '...'
    })
    return JSON.stringify(sample, null, 2)
  }
})

// 导出为CSV
const exportToCSV = () => {
  const headers = selectedFields.value.map(f => fieldOptions.find(fo => fo.value === f)?.label || f)
  const rows = linksToExport.value.map(link => {
    return selectedFields.value.map(field => {
      const value = link[field as keyof ShortLink]
      // 处理包含逗号或引号的值
      if (typeof value === 'string' && (value.includes(',') || value.includes('"'))) {
        return `"${value.replace(/"/g, '""')}"`
      }
      return value
    }).join(',')
  })

  const csv = [headers.join(','), ...rows].join('\n')
  downloadFile(csv, 'shortlinks.csv', 'text/csv;charset=utf-8;')
}

// 导出为JSON
const exportToJSON = () => {
  const data = linksToExport.value.map(link => {
    const item: Record<string, unknown> = {}
    selectedFields.value.forEach(field => {
      item[field] = link[field as keyof ShortLink]
    })
    return item
  })

  const json = JSON.stringify(data, null, 2)
  downloadFile(json, 'shortlinks.json', 'application/json')
}

// 下载文件
const downloadFile = (content: string, filename: string, mimeType: string) => {
  // 添加 BOM 以支持中文
  const BOM = '\uFEFF'
  const blob = new Blob([BOM + content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

// 执行导出
const handleExport = () => {
  if (selectedFields.value.length === 0) {
    QMessage.warning('请至少选择一个导出字段')
    return
  }

  if (exportCount.value === 0) {
    QMessage.warning('没有可导出的数据')
    return
  }

  try {
    if (selectedFormat.value === 'csv') {
      exportToCSV()
    } else {
      exportToJSON()
    }
    QMessage.success(`成功导出 ${exportCount.value} 条记录`)
    handleClose()
  } catch (error) {
    console.error('导出失败:', error)
    QMessage.error('导出失败')
  }
}

// 关闭对话框
const handleClose = () => {
  emit('update:visible', false)
}
</script>

<style scoped>
.export-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.form-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.format-options {
  display: flex;
  gap: 12px;
}

.format-option {
  flex: 1;
  cursor: pointer;
}

.format-option input {
  display: none;
}

.format-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 16px;
  border: 2px solid var(--border-color);
  border-radius: var(--radius-md);
  transition: all var(--transition-fast);
  background: var(--right-content-bg);
}

.format-option.active .format-info {
  border-color: var(--primary-color);
  background: var(--primary-bg, rgba(59, 130, 246, 0.05));
}

.format-info i {
  font-size: 24px;
  color: var(--primary-color);
}

.format-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.format-desc {
  font-size: 12px;
  color: var(--text-secondary);
}

.scope-options,
.field-options {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.radio-option,
.checkbox-option {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.radio-option input,
.checkbox-option input {
  cursor: pointer;
}

.radio-label,
.checkbox-label {
  font-size: 14px;
  color: var(--text-color);
}

.field-options {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
}

.export-preview {
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.preview-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  background: var(--color-gray-50, #f9fafb);
  border-bottom: 1px solid var(--border-color);
  font-size: 13px;
  font-weight: 500;
  color: var(--text-color);
}

.preview-count {
  font-size: 12px;
  font-weight: normal;
  color: var(--text-secondary);
}

.preview-content {
  padding: 12px;
  background: var(--color-gray-50, #f9fafb);
  max-height: 80px;
  overflow: auto;
}

.preview-content code {
  font-size: 12px;
  color: var(--text-secondary);
  white-space: pre-wrap;
  word-break: break-all;
}

.q-btn {
  padding: 8px 20px;
  border-radius: var(--radius-md);
  font-size: var(--font-size-base);
  font-weight: var(--font-weight-medium);
  cursor: pointer;
  transition: all var(--transition-fast);
  min-width: 80px;
  border: 1px solid var(--border-color);
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.q-btn--default {
  background: var(--right-content-bg);
  color: var(--text-color);
}

.q-btn--default:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.q-btn--primary {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.q-btn--primary:hover:not(:disabled) {
  background: var(--primary-dark);
  border-color: var(--primary-dark);
}

.q-btn--primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>

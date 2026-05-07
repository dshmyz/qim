<template>
  <div class="ai-knowledge-settings">
    <div class="setting-section">
      <div class="section-header">
        <label class="section-label">绑定文档</label>
        <button class="add-btn" @click="toggleFilePicker">
          <i :class="showFilePicker ? 'fas fa-minus' : 'fas fa-plus'"></i>
          {{ showFilePicker ? '收起' : '添加文档' }}
        </button>
      </div>

      <div v-if="documents.length === 0 && !showFilePicker" class="empty-state">
        <i class="fas fa-folder-open"></i>
        <p>暂未绑定任何文档</p>
      </div>

      <div class="document-list">
        <div v-for="doc in documents" :key="doc.id" class="document-item">
          <div class="doc-info">
            <i :class="getFileIcon(doc.file?.type || '')" class="doc-icon"></i>
            <div class="doc-name">{{ doc.file?.name || '未知文件' }}</div>
            <div class="doc-size">{{ formatSize(doc.file?.size || 0) }}</div>
          </div>
          <button class="remove-btn" @click="removeDocument(doc)" title="移除">
            <i class="fas fa-trash-alt"></i>
          </button>
        </div>
      </div>

      <div v-if="showFilePicker" class="file-picker-section">
        <div class="picker-header">
          <span class="picker-title">选择文档</span>
          <div class="picker-actions">
            <button class="btn btn-secondary" @click="cancelFilePicker">取消</button>
            <button class="btn btn-primary" @click="confirmAddDocuments" :disabled="selectedFileIds.length === 0">
              确认添加
            </button>
          </div>
        </div>

        <div class="file-picker-list">
          <div v-for="file in availableFiles" :key="file.id" class="file-option" @click="toggleFile(file)">
            <input type="checkbox" :checked="isFileSelected(file.id)" />
            <i :class="getFileIcon(file.type || '')" class="file-icon"></i>
            <span class="file-name">{{ file.name }}</span>
            <span class="file-size">{{ formatSize(file.size || 0) }}</span>
          </div>
          <div v-if="availableFiles.length === 0" class="empty-picker">
            <i class="fas fa-spinner fa-spin" v-if="loadingFiles"></i>
            <span v-else>暂无可用文件</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { GroupDocument } from '../../../types/ai'

interface Props {
  groupId: number
  serverUrl: string
  documents: GroupDocument[]
}

interface Emits {
  (e: 'add', fileIds: number[]): void
  (e: 'remove', fileId: number): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const showFilePicker = ref(false)
const availableFiles = ref<any[]>([])
const selectedFileIds = ref<number[]>([])
const loadingFiles = ref(false)

function toggleFilePicker() {
  showFilePicker.value = !showFilePicker.value
  if (showFilePicker.value && availableFiles.value.length === 0) {
    loadAvailableFiles()
  }
}

async function loadAvailableFiles() {
  loadingFiles.value = true
  try {
    const response = await fetch(`${props.serverUrl}/api/v1/files?page_size=100&type=document`, {
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    })
    const data = await response.json()
    availableFiles.value = data.data?.files || []
  } catch (e) {
    console.error('加载文件列表失败', e)
  } finally {
    loadingFiles.value = false
  }
}

function isFileSelected(fileId: number) {
  return selectedFileIds.value.includes(fileId)
}

function toggleFile(file: any) {
  const idx = selectedFileIds.value.indexOf(file.id)
  if (idx >= 0) {
    selectedFileIds.value.splice(idx, 1)
  } else {
    selectedFileIds.value.push(file.id)
  }
}

function confirmAddDocuments() {
  if (selectedFileIds.value.length > 0) {
    emit('add', [...selectedFileIds.value])
    selectedFileIds.value = []
    showFilePicker.value = false
  }
}

function cancelFilePicker() {
  selectedFileIds.value = []
  showFilePicker.value = false
}

function removeDocument(doc: GroupDocument) {
  emit('remove', doc.file_id)
}

function getFileIcon(type: string) {
  if (type.includes('pdf')) return 'fas fa-file-pdf'
  if (type.includes('word') || type.includes('document')) return 'fas fa-file-word'
  if (type.includes('excel') || type.includes('sheet')) return 'fas fa-file-excel'
  if (type.includes('text')) return 'fas fa-file-alt'
  return 'fas fa-file'
}

function formatSize(bytes: number) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}
</script>

<style scoped>
.ai-knowledge-settings { padding: 16px; }
.setting-section { margin-bottom: 20px; }
.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.section-label { font-size: 14px; font-weight: 500; }
.add-btn { display: inline-flex; align-items: center; gap: 6px; padding: 6px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 13px; cursor: pointer; }
.add-btn:hover { border-color: var(--primary-color); color: var(--primary-color); }
.empty-state { text-align: center; padding: 32px; color: var(--text-secondary); }
.empty-state i { font-size: 40px; margin-bottom: 8px; display: block; }
.document-list { display: flex; flex-direction: column; gap: 8px; }
.document-item { display: flex; justify-content: space-between; align-items: center; padding: 10px 12px; background: var(--bg-color); border: 1px solid var(--border-color); border-radius: 8px; }
.doc-info { display: flex; align-items: center; gap: 10px; }
.doc-icon { font-size: 20px; color: var(--text-secondary); }
.doc-name { font-size: 14px; }
.doc-size { font-size: 12px; color: var(--text-secondary); }
.remove-btn { background: none; border: none; color: var(--text-secondary); cursor: pointer; padding: 6px; font-size: 14px; border-radius: 4px; }
.remove-btn:hover { color: #ef4444; background: rgba(239, 68, 68, 0.1); }

.file-picker-section {
  margin-top: 16px;
  padding: 16px;
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 8px;
}

.picker-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-color);
}

.picker-title {
  font-size: 14px;
  font-weight: 500;
}

.picker-actions {
  display: flex;
  gap: 8px;
}

.file-picker-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 300px;
  overflow-y: auto;
}

.file-option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  border-radius: 6px;
  cursor: pointer;
}

.file-option:hover {
  background: var(--hover-color);
}

.file-icon {
  font-size: 16px;
  color: var(--text-secondary);
}

.file-name {
  flex: 1;
  font-size: 14px;
}

.file-size {
  font-size: 12px;
  color: var(--text-secondary);
}

.empty-picker {
  text-align: center;
  padding: 20px;
  color: var(--text-secondary);
}

.btn {
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  border: none;
}

.btn-primary {
  background: var(--primary-color);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  opacity: 0.9;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--bg-color);
  color: var(--text-color);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}
</style>
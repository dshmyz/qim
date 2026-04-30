<template>
  <div class="ai-knowledge-settings">
    <div class="setting-section">
      <div class="section-header">
        <label class="section-label">绑定文档</label>
        <button class="add-btn" @click="showFilePicker = true">
          <i class="fas fa-plus"></i> 添加文档
        </button>
      </div>

      <div v-if="documents.length === 0" class="empty-state">
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
    </div>

    <div v-if="showFilePicker" class="modal-overlay" @click="showFilePicker = false">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>选择文档</h3>
          <button class="close-btn" @click="showFilePicker = false">&times;</button>
        </div>
        <div class="modal-body">
          <div class="file-picker-list">
            <div v-for="file in availableFiles" :key="file.id" class="file-option" @click="toggleFile(file)">
              <input type="checkbox" :checked="isFileSelected(file.id)" />
              <span>{{ file.name }}</span>
            </div>
            <div v-if="availableFiles.length === 0" class="empty-picker">暂无可用文件</div>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn btn-secondary" @click="showFilePicker = false">取消</button>
          <button class="btn btn-primary" @click="confirmAddDocuments">确认</button>
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

async function loadAvailableFiles() {
  try {
    const response = await fetch('/api/v1/files?page_size=100', {
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    })
    const data = await response.json()
    availableFiles.value = data.data?.files || []
  } catch (e) {
    console.error('加载文件列表失败', e)
  }
}

watch(showFilePicker, async (val) => {
  if (val) {
    await loadAvailableFiles()
    selectedFileIds.value = []
  }
})

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
  }
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
.modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0, 0, 0, 0.5); display: flex; align-items: center; justify-content: center; z-index: 2000; }
.modal-content { background: var(--sidebar-bg); border-radius: 12px; width: 90%; max-width: 500px; max-height: 80vh; overflow: hidden; display: flex; flex-direction: column; }
.modal-header { display: flex; justify-content: space-between; padding: 16px 20px; border-bottom: 1px solid var(--border-color); }
.modal-header h3 { margin: 0; font-size: 16px; }
.modal-header .close-btn { background: none; border: none; font-size: 20px; cursor: pointer; color: var(--text-secondary); }
.modal-body { padding: 16px 20px; overflow-y: auto; flex: 1; }
.file-picker-list { display: flex; flex-direction: column; gap: 4px; }
.file-option { display: flex; align-items: center; gap: 8px; padding: 8px; border-radius: 6px; cursor: pointer; }
.file-option:hover { background: var(--hover-color); }
.empty-picker { text-align: center; padding: 20px; color: var(--text-secondary); }
.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 12px 20px; border-top: 1px solid var(--border-color); }
.btn { padding: 8px 16px; border-radius: 6px; font-size: 14px; cursor: pointer; border: none; }
.btn-primary { background: var(--primary-color); color: white; }
.btn-secondary { background: var(--bg-color); color: var(--text-color); border: 1px solid var(--border-color); }
</style>

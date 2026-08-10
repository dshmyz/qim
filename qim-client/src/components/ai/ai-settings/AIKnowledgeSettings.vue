<template>
  <div class="ai-knowledge-settings">
    <div class="setting-section">
      <div class="section-header">
        <label class="section-label">绑定文档</label>
        <div class="header-actions">
          <button class="add-btn" @click="refresh">
            <i class="fas fa-sync-alt"></i> 刷新
          </button>
          <button class="add-btn" @click="toggleFilePicker">
            <i :class="showFilePicker ? 'fas fa-minus' : 'fas fa-plus'"></i>
            {{ showFilePicker ? '收起' : '添加文档' }}
          </button>
        </div>
      </div>

      <div v-if="documents.length === 0 && !showFilePicker" class="empty-state">
        <i class="fas fa-folder-open"></i>
        <p>暂未绑定任何文档</p>
      </div>

      <div class="document-list">
        <div v-for="doc in documents" :key="doc.id" class="document-item">
          <div class="doc-info">
            <i :class="getFileIcon(doc.file?.type || '')" class="doc-icon"></i>
            <div class="doc-details">
              <div class="doc-name">{{ doc.file?.name || '未知文件' }}</div>
              <div class="doc-size">{{ formatSize(doc.file?.size || 0) }}</div>
              <div v-if="doc.process_status === 'failed' && doc.process_error" class="doc-error" :title="doc.process_error">
                {{ doc.process_error }}
              </div>
            </div>
          </div>
          <div class="doc-status">
            <span v-if="!doc.process_status || doc.process_status === 'pending'" class="status-badge status-pending">
              <i class="fas fa-clock"></i> 等待处理
              <!-- pending 也可能是绑定后未能触达（如向量服务当时未就绪）而卡住的文档，
                   给一个「重新处理」入口，点击后复用 /process 幂等触发，与 failed 一致可恢复。 -->
              <button v-if="!isRetrying(doc.id)" class="retry-btn" @click="retryDocument(doc)" title="重新处理">重新处理</button>
              <button v-else class="retry-btn" :disabled="true">处理中...</button>
            </span>
            <span v-else-if="doc.process_status === 'processing'" class="status-badge status-processing">
              <i class="fas fa-spinner fa-spin"></i> 处理中...
              <button v-if="!isRetrying(doc.id)" class="retry-btn" @click="retryDocument(doc)" title="重新处理">重试</button>
            </span>
            <span v-else-if="doc.process_status === 'completed'" class="status-badge status-completed">
              <i class="fas fa-check-circle"></i> 已就绪
            </span>
            <span v-else-if="doc.process_status === 'failed'" class="status-badge status-failed">
              <i class="fas fa-exclamation-circle"></i> 失败
              <button class="retry-btn" @click="retryDocument(doc)" title="重试" :disabled="isRetrying(doc.id)">
                {{ isRetrying(doc.id) ? '重试中...' : '重试' }}
              </button>
            </span>
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
            <button class="btn btn-secondary" @click="triggerUpload" :disabled="uploading">
              <i class="fas fa-cloud-upload-alt"></i>
              {{ uploading ? '上传中...' : '上传文档' }}
            </button>
            <input
              ref="fileInput"
              type="file"
              multiple
              class="file-input-hidden"
              accept=".pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.csv,.md,application/pdf,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document,application/vnd.ms-excel,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/vnd.ms-powerpoint,application/vnd.openxmlformats-officedocument.presentationml.presentation,text/plain,text/csv,text/markdown"
              @change="onUpload"
            />
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
import QMessage from '../../../utils/qmessage'
import { groupFiles } from '../../../api/groupFiles'
import { uploadFilesWithLimit } from '../../../composables/useFileUpload'

interface Props {
  groupId: number
  serverUrl: string
  documents: GroupDocument[]
}

interface Emits {
  (e: 'add', fileIds: number[]): void
  (e: 'remove', fileId: number): void
  (e: 'retry', doc: any): void
  (e: 'refresh'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const showFilePicker = ref(false)
const availableFiles = ref<any[]>([])
const selectedFileIds = ref<number[]>([])
const loadingFiles = ref(false)
const uploading = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

function toggleFilePicker() {
  showFilePicker.value = !showFilePicker.value
  if (showFilePicker.value && availableFiles.value.length === 0) {
    loadAvailableFiles()
  }
}

async function loadAvailableFiles() {
  loadingFiles.value = true
  try {
    // 群 AI 的可绑定文档应来自「群文件空间」而非个人的 personal files API：
    // 之前的实现调 GET /api/v1/files（只覆盖 scope_type='user' 的个人文件），
    // 群文档上传后落在 scope_type='conversation'，故下拉永远为空。
    // 这里改为读群文件接口，并带上 all=1 列出整个群空间的全部文件：
    // 默认的无 folder_id 查询只返回根目录（folder_id IS NULL），而文档可能被
    // 上传/移动到群文件夹里，会漏掉；all=1 忽略文件夹层级。
    const response = await fetch(
      `${props.serverUrl}/api/v1/groups/${props.groupId}/files?page_size=200&all=1`,
      {
        headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
      }
    )
    const data = await response.json()
    const files: any[] = data.data?.files || []
    availableFiles.value = files.filter(isBindableDocument)
  } catch (e) {
    console.error('加载文件列表失败', e)
  } finally {
    loadingFiles.value = false
  }
}

// 与后端 AddGroupDocument 允许的文档 MIME 白名单保持一致，
// 保证「可选文档」里列出的都能真正绑定成功（不过滤则会把图片/视频等也列出，点了又报错）。
//
// 入参分两种来源，字段名不同需都兼容：
//  - 后端文件对象（loadAvailableFiles）：MIME 在 mime_type
//  - 浏览器本地 File 对象（onUpload 上传成功后）：MIME 在 type
// 只读 mime_type 会把上传的本地 docx 等误判为非文档类型（type 缺失 → 空串 → 未绑定）。
//
// 另外 OOXML 文档（docx/xlsx/pptx）是 ZIP 容器，服务端检测成 application/zip 不在白名单，
// 需用文件扩展名兜底，与后端 AddGroupDocument 的兜底一致，保证列表能勾选且绑定能成功。
function isBindableDocument(file: any): boolean {
  const mime = String(file.mime_type ?? (file.type || '')).split(';')[0].trim()
  const docMimes = [
    'application/pdf',
    'application/msword',
    'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    'application/vnd.ms-excel',
    'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    'application/vnd.ms-powerpoint',
    'application/vnd.openxmlformats-officedocument.presentationml.presentation',
    'text/plain',
    'text/csv',
    'text/markdown'
  ]
  if (docMimes.includes(mime)) return true
  // 扩展名兜底：OOXML（zip 容器）及历史 mime 异常的文档仍按扩展名识别
  const name = String(file.name || file.original_name || '')
  const extMatch = /\.([^.]+)$/.exec(name)
  const ext = extMatch ? ('.' + extMatch[1]).toLowerCase() : ''
  return ['.pdf', '.doc', '.docx', '.xls', '.xlsx', '.ppt', '.pptx', '.txt', '.csv', '.md'].includes(ext)
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

function triggerUpload() {
  fileInput.value?.click()
}

/**
 * 直接上传本地文档到群知识库：
 * 1. 分片上传 + 挂载到群文件空间（复用群文件面板的通用上传器）
 * 2. 仅对文档类型（与后端 MIME 白名单一致）的文件自动 emit('add') 触发绑定与向量化
 */
async function onUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const list = input.files ? Array.from(input.files) : []
  input.value = ''
  if (list.length === 0 || uploading.value) return

  uploading.value = true
  try {
    const results = await uploadFilesWithLimit(list, undefined, {
      onFileUploaded: async (_file, fileId) => {
        await groupFiles.attach(props.groupId, fileId)
      }
    })

    // 仅文档类型可绑定；其余类型（图片/视频等）上报但自动跳过
    const bound = results.filter(r => r.success && r.fileId && isBindableDocument(r.file))
    const uploaded = results.filter(r => r.success).length
    const failed = results.length - uploaded

    if (bound.length > 0) {
      emit('add', bound.map(r => r.fileId!))
    }

    if (uploaded > 0 && bound.length < uploaded) {
      QMessage.warning(`已上传 ${uploaded} 个文件，其中 ${uploaded - bound.length} 个非文档类型未绑定`)
    } else if (bound.length === 0 && uploaded > 0) {
      QMessage.warning('上传的文件均非文档类型，未绑定到知识库')
    } else if (failed > 0 && bound.length === 0) {
      QMessage.error('上传失败')
    } else if (bound.length > 0) {
      QMessage.success(`已上传并绑定 ${bound.length} 个文档`)
    }

    selectedFileIds.value = []
    showFilePicker.value = false
    // 刷新可用文件列表，让上传的非文档文件也能在下次勾选时出现
    await loadAvailableFiles()
  } catch (e) {
    console.error('上传文档失败', e)
    QMessage.error('上传文档失败')
  } finally {
    uploading.value = false
  }
}

function removeDocument(doc: any) {
  emit('remove', doc.file_id)
}

// 手动刷新：左上角「刷新」应同时刷新两边——
// 上方「已绑定文档」交给父组件重拉（emit refresh），
// 下方「可添加文档」文件列表本地重载（否则点刷新列表不更新）。
function refresh() {
  emit('refresh')
  loadAvailableFiles()
}

// 每文档重试 in-flight 标记：点击重试后立即禁用该文档的重试按钮，防止"能一直点"。
// 仅当父组件刷新后该文档离开处理中（进入终态）才会清除，恢复可重试。
const retryingIds = ref<Set<number>>(new Set())

function isRetrying(id: number) {
  return retryingIds.value.has(id)
}

function retryDocument(doc: any) {
  const next = new Set(retryingIds.value)
  next.add(doc.id)
  retryingIds.value = next
  emit('retry', doc)
}

// 刷新后的新列表中若某文档已不再处理中，说明重试已进入终态，清除其 in-flight 标记
watch(
  () => props.documents,
  (docs) => {
    let changed = false
    const next = new Set<number>()
    for (const id of retryingIds.value) {
      const d = docs.find((x) => x.id === id)
      const active = d && (d.process_status === 'processing' || d.process_status === 'pending')
      if (active) next.add(id)
      else changed = true
    }
    if (changed) retryingIds.value = next
  }
)

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
.header-actions { display: flex; gap: 8px; }
.section-label { font-size: 14px; font-weight: 500; }
.add-btn { display: inline-flex; align-items: center; gap: 6px; padding: 6px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 13px; cursor: pointer; }
.add-btn:hover { border-color: var(--primary-color); color: var(--primary-color); }
.empty-state { text-align: center; padding: 32px; color: var(--text-secondary); }
.empty-state i { font-size: 40px; margin-bottom: 8px; display: block; }
.document-list { display: flex; flex-direction: column; gap: 8px; }
.document-item { display: flex; align-items: center; gap: 8px; padding: 10px 12px; background: var(--bg-color); border: 1px solid var(--border-color); border-radius: 8px; }
.doc-info { display: flex; align-items: center; gap: 10px; flex: 1; min-width: 0; }
.doc-icon { font-size: 20px; color: var(--text-secondary); flex-shrink: 0; }
.doc-details { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.doc-name { font-size: 14px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.doc-size { font-size: 12px; color: var(--text-secondary); }
.doc-error { font-size: 12px; color: #dc2626; line-height: 1.4; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; word-break: break-all; }
.doc-status { display: flex; align-items: center; gap: 6px; flex-shrink: 0; }
.status-badge { display: inline-flex; align-items: center; gap: 4px; padding: 2px 8px; border-radius: 12px; font-size: 12px; white-space: nowrap; }
.status-badge i { font-size: 12px; }
.status-pending { background: #f0f0f0; color: #666; }
.status-processing { background: #e0f2fe; color: #0284c7; }
.status-completed { background: #dcfce7; color: #16a34a; }
.status-failed { background: #fee2e2; color: #dc2626; }
.retry-btn { margin-left: 4px; padding: 1px 6px; border: 1px solid #dc2626; border-radius: 4px; background: white; color: #dc2626; font-size: 11px; cursor: pointer; flex-shrink: 0; }
.retry-btn:hover { background: #dc2626; color: white; }
.retry-btn:disabled { opacity: 0.5; cursor: not-allowed; background: white; color: #dc2626; }
.remove-btn { background: none; border: none; color: var(--text-secondary); cursor: pointer; padding: 6px; font-size: 14px; border-radius: 4px; flex-shrink: 0; }
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
  /* 覆盖全局 main.css 里的 .file-name{text-align:center}：
     该全局规则会把「选择文档」列表的文件名居中，需在此显式改回左对齐。 */
  text-align: left;
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

.file-input-hidden {
  display: none;
}
</style>
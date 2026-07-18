<template>
  <div class="group-files-backdrop" @click.self="emit('close')">
    <section class="group-files-panel" aria-label="群文件">
      <header class="group-files-panel__header">
        <div>
          <h2><i class="fas fa-folder-open"></i> 群文件</h2>
          <p>群成员可查看和下载文件</p>
        </div>
        <button class="icon-button" type="button" aria-label="关闭群文件" @click="emit('close')">
          <i class="fas fa-times"></i>
        </button>
      </header>

      <div class="group-files-panel__toolbar">
        <label class="primary-action upload-action" title="选择文件上传到群文件">
          <i class="fas fa-cloud-upload-alt"></i> 上传文件
          <input type="file" @change="handleUpload" />
        </label>
        <button v-if="canManage" class="secondary-action" type="button" @click="createFolder">
          <i class="fas fa-folder-plus"></i> 新建文件夹
        </button>
        <label class="search-field">
          <i class="fas fa-search"></i>
          <input v-model="search" type="search" placeholder="搜索文件" @keyup.enter="loadFiles(1)" />
        </label>
      </div>

      <div class="group-files-panel__content">
        <aside class="folder-tree" aria-label="目录树">
          <button class="folder-tree__item" :class="{ active: currentFolderId === null }" type="button" @click="openFolder(null)">
            <i class="fas fa-folder"></i> 全部文件
          </button>
          <button
            v-for="folder in folders"
            :key="folder.id"
            class="folder-tree__item"
            :class="{ active: currentFolderId === folder.id }"
            type="button"
            @click="openFolder(folder.id)"
          >
            <i class="fas fa-folder"></i> {{ folder.name }}
          </button>
        </aside>

        <main class="file-list">
          <div v-if="referenceFileId" class="reference-callout">
            <span>选择当前目录后保存聊天附件。</span>
            <button class="primary-action" type="button" @click="saveReference">保存到当前目录</button>
          </div>
          <p v-if="loading" class="file-list__state">正在加载…</p>
          <p v-else-if="files.length === 0" class="file-list__state">当前目录没有文件</p>
          <table v-else>
            <thead>
              <tr><th>名称</th><th>上传者</th><th>时间</th><th>大小</th><th>操作</th></tr>
            </thead>
            <tbody>
              <tr v-for="file in files" :key="file.id">
                <td><i class="fas fa-file"></i> {{ file.name }}</td>
                <td>{{ uploaderName(file) }}</td>
                <td>{{ formatDate(file.created_at) }}</td>
                <td>{{ formatSize(file.size) }}</td>
                <td class="file-list__actions">
                  <button type="button" @click="download(file)"><i class="fas fa-download"></i> 下载</button>
                  <button v-if="canManage" type="button" @click="move(file)"><i class="fas fa-folder"></i> 移动</button>
                  <button v-if="canManage" class="danger" type="button" @click="remove(file)"><i class="fas fa-trash"></i> 删除</button>
                </td>
              </tr>
            </tbody>
          </table>
          <footer v-if="total > pageSize" class="pagination">
            <button type="button" :disabled="page === 1" @click="loadFiles(page - 1)">上一页</button>
            <span>第 {{ page }} 页，共 {{ Math.ceil(total / pageSize) }} 页</span>
            <button type="button" :disabled="page * pageSize >= total" @click="loadFiles(page + 1)">下一页</button>
          </footer>
        </main>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import QMessage from '../../utils/qmessage'
import { groupFiles, type GroupFile, type GroupFolder } from '../../api/groupFiles'

const props = withDefaults(defineProps<{
  groupId: string | number
  canManage: boolean
  referenceFileId?: number | null
}>(), {
  referenceFileId: null,
})

const emit = defineEmits<{ close: [] }>()
const files = ref<GroupFile[]>([])
const folders = ref<GroupFolder[]>([])
const loading = ref(false)
const search = ref('')
const currentFolderId = ref<number | null>(null)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const loadFiles = async (targetPage = page.value) => {
  loading.value = true
  try {
    const response = await groupFiles.list(props.groupId, {
      folder_id: currentFolderId.value,
      page: targetPage,
      page_size: pageSize.value,
      search: search.value || undefined,
    })
    const data = response.data.data
    files.value = data.files
    folders.value = data.folders
    page.value = data.page
    pageSize.value = data.page_size
    total.value = data.total
  } catch (error: any) {
    QMessage.error(error?.message || '加载群文件失败')
  } finally {
    loading.value = false
  }
}

const openFolder = (folderId: number | null) => {
  currentFolderId.value = folderId
  loadFiles(1)
}

const createFolder = async () => {
  const name = window.prompt('文件夹名称')?.trim()
  if (!name) return
  try {
    await groupFiles.createFolder(props.groupId, name, currentFolderId.value)
    await loadFiles(1)
  } catch (error: any) {
    QMessage.error(error?.message || '新建文件夹失败')
  }
}

const move = async (file: GroupFile) => {
  const folderId = window.prompt('目标文件夹 ID（留空移到根目录）')
  if (folderId === null) return
  const target = folderId.trim() ? Number(folderId) : null
  if (target !== null && (!Number.isInteger(target) || target <= 0)) {
    QMessage.error('请输入有效的文件夹 ID')
    return
  }
  try {
    await groupFiles.move(props.groupId, file.id, target)
    await loadFiles()
  } catch (error: any) {
    QMessage.error(error?.message || '移动文件失败')
  }
}

const remove = async (file: GroupFile) => {
  if (!window.confirm(`确定删除“${file.name}”吗？`)) return
  try {
    await groupFiles.remove(props.groupId, file.id)
    await loadFiles()
  } catch (error: any) {
    QMessage.error(error?.message || '删除文件失败')
  }
}

const saveReference = async () => {
  if (!props.referenceFileId) return
  try {
    await groupFiles.shareReference(props.groupId, props.referenceFileId, currentFolderId.value)
    QMessage.success('已保存到群文件')
    emit('close')
  } catch (error: any) {
    QMessage.error(error?.message || '保存到群文件失败')
  }
}

const handleUpload = (event: Event) => {
  const input = event.target as HTMLInputElement
  if (!input.files?.length) return
  // Task 3 intentionally exposes only list/manage/reference endpoints. Do not upload
  // to the personal-file route here: that would create an inaccessible user-scoped file.
  QMessage.warning('服务器暂未提供群文件上传接口')
  input.value = ''
}

const download = (_file: GroupFile) => {
  // A group-scoped download route is likewise required server-side so authorization
  // remains in FileSpaceService rather than falling back to personal-file endpoints.
  QMessage.warning('服务器暂未提供群文件下载接口')
}

const uploaderName = (file: GroupFile) => file.uploader?.name || `用户 ${file.user_id}`
const formatDate = (value: string) => value ? new Date(value).toLocaleString() : '-'
const formatSize = (size: number) => {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

watch(() => props.groupId, () => {
  currentFolderId.value = null
  loadFiles(1)
}, { immediate: true })
</script>

<style scoped>
.group-files-backdrop { position: fixed; inset: 0; z-index: 3100; display: grid; place-items: center; background: rgba(0, 0, 0, .42); }
.group-files-panel { width: min(960px, calc(100vw - 32px)); max-height: min(700px, calc(100vh - 32px)); overflow: auto; border-radius: 12px; background: var(--card-bg, #fff); color: var(--text-color, #222); box-shadow: 0 20px 48px rgba(0, 0, 0, .22); }
.group-files-panel__header, .group-files-panel__toolbar, .group-files-panel__content { display: flex; gap: 12px; padding: 16px 20px; }
.group-files-panel__header { align-items: center; justify-content: space-between; border-bottom: 1px solid var(--border-color, #e5e7eb); }
.group-files-panel__header h2 { margin: 0; font-size: 18px; }.group-files-panel__header p { margin: 4px 0 0; color: var(--text-secondary, #6b7280); font-size: 13px; }
.group-files-panel__toolbar { align-items: center; border-bottom: 1px solid var(--border-color, #e5e7eb); }.primary-action, .secondary-action, .file-list__actions button, .pagination button, .icon-button { border: 0; border-radius: 6px; cursor: pointer; padding: 8px 10px; background: transparent; color: inherit; }.primary-action { background: var(--primary-color, #4f46e5); color: #fff; }.secondary-action { border: 1px solid var(--border-color, #d1d5db); }.upload-action input { display: none; }.search-field { margin-left: auto; display: flex; align-items: center; gap: 7px; border: 1px solid var(--border-color, #d1d5db); border-radius: 6px; padding: 7px 9px; }.search-field input { width: 180px; border: 0; outline: 0; background: transparent; color: inherit; }
.group-files-panel__content { min-height: 340px; align-items: stretch; }.folder-tree { width: 190px; flex: 0 0 auto; border-right: 1px solid var(--border-color, #e5e7eb); padding-right: 12px; }.folder-tree__item { display: block; width: 100%; border: 0; border-radius: 6px; background: transparent; color: inherit; cursor: pointer; padding: 9px; text-align: left; }.folder-tree__item.active, .folder-tree__item:hover { background: var(--hover-color, #f3f4f6); }.file-list { min-width: 0; flex: 1; }.file-list table { width: 100%; border-collapse: collapse; font-size: 13px; }.file-list th, .file-list td { border-bottom: 1px solid var(--border-color, #e5e7eb); padding: 11px 8px; text-align: left; }.file-list th { color: var(--text-secondary, #6b7280); font-weight: 500; }.file-list__state { padding: 36px; text-align: center; color: var(--text-secondary, #6b7280); }.file-list__actions { white-space: nowrap; }.file-list__actions button:hover { background: var(--hover-color, #f3f4f6); }.file-list__actions .danger { color: #dc2626; }.reference-callout { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 12px; padding: 10px; border-radius: 6px; background: #eef2ff; color: #3730a3; }.pagination { display: flex; align-items: center; justify-content: flex-end; gap: 12px; padding-top: 14px; font-size: 13px; }.pagination button:disabled { opacity: .45; cursor: not-allowed; }
@media (max-width: 650px) { .group-files-panel__toolbar { flex-wrap: wrap; }.search-field { margin-left: 0; }.group-files-panel__content { padding: 12px; }.folder-tree { width: 130px; }.file-list table th:nth-child(2), .file-list table td:nth-child(2), .file-list table th:nth-child(3), .file-list table td:nth-child(3) { display: none; } }
</style>

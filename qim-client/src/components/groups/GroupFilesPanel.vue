<template>
  <div class="group-files-backdrop" @click.self="emit('close')">
    <section class="group-files-panel" aria-label="群文件">
      <!-- Header -->
      <header class="gfp-header">
        <div class="gfp-header__info">
          <h2 class="gfp-header__title"><i class="fas fa-folder-open"></i> 群文件</h2>
          <p class="gfp-header__subtitle">群成员可查看和下载文件</p>
        </div>
        <button class="gfp-close" type="button" aria-label="关闭群文件" @click="emit('close')">
          <i class="fas fa-times"></i>
        </button>
      </header>

      <!-- Toolbar -->
      <div class="gfp-toolbar">
        <label
          class="gfp-btn gfp-btn--primary"
          :class="{ 'is-loading': uploading }"
          :title="uploading ? '正在上传…' : '选择文件上传到群文件'"
        >
          <i :class="uploading ? 'fas fa-circle-notch fa-spin' : 'fas fa-cloud-upload-alt'"></i>
          <span>{{ uploading ? '上传中…' : '上传文件' }}</span>
          <input type="file" multiple :disabled="uploading" @change="handleUpload" />
        </label>
        <button v-if="canManage" class="gfp-btn gfp-btn--secondary" type="button" @click="createFolder">
          <i class="fas fa-folder-plus"></i> 新建文件夹
        </button>
        <div class="gfp-toolbar__spacer"></div>
        <div class="gfp-search">
          <i class="fas fa-search"></i>
          <input v-model="search" type="search" placeholder="搜索文件…" @keyup.enter="triggerSearch" />
          <button v-if="search" class="gfp-search__clear" type="button" aria-label="清空搜索" @click="clearSearch">
            <i class="fas fa-times"></i>
          </button>
        </div>
      </div>

      <!-- Content -->
      <div
        class="gfp-content"
        :class="{ 'is-dragover': dragOver }"
        @dragenter="onDragEnter"
        @dragover="onDragOver"
        @dragleave="onDragLeave"
        @drop="onDrop"
      >
        <!-- File list -->
        <main class="gfp-files">
          <!-- Batch toolbar -->
          <div v-if="selectedCount > 0" class="gfp-batch-bar">
            <span class="gfp-batch-bar__count">已选 {{ selectedCount }} 项</span>
            <div class="gfp-batch-bar__actions">
              <button class="gfp-btn gfp-btn--secondary gfp-btn--sm" type="button" :disabled="batchDownloading" @click="batchDownload">
                <i :class="batchDownloading ? 'fas fa-circle-notch fa-spin' : 'fas fa-download'"></i>
                {{ batchDownloading ? '下载中…' : '批量下载' }}
              </button>
              <button v-if="canManage" class="gfp-btn gfp-btn--danger gfp-btn--sm" type="button" :disabled="batchDeleting" @click="batchDelete">
                <i :class="batchDeleting ? 'fas fa-circle-notch fa-spin' : 'fas fa-trash'"></i>
                批量删除
              </button>
              <button class="gfp-btn gfp-btn--ghost gfp-btn--sm" type="button" @click="clearSelection">取消选择</button>
            </div>
          </div>

          <!-- Breadcrumb -->
          <nav v-if="breadcrumbs.length > 1" class="gfp-breadcrumb" aria-label="目录路径">
            <template v-for="(crumb, idx) in breadcrumbs" :key="crumb.id ?? 'root'">
              <button
                class="gfp-breadcrumb__item"
                :class="{ active: idx === breadcrumbs.length - 1 }"
                type="button"
                @click="openFolder(crumb.id)"
              >
                {{ crumb.name }}
              </button>
              <i v-if="idx < breadcrumbs.length - 1" class="fas fa-chevron-right gfp-breadcrumb__sep"></i>
            </template>
          </nav>

          <!-- Back button -->
          <button
            v-if="currentFolderId !== null"
            class="gfp-back-btn"
            type="button"
            @click="goBack"
          >
            <i class="fas fa-arrow-left"></i> 返回上级
          </button>

          <!-- Reference callout -->
          <div v-if="referenceMessageId && referenceFileId" class="gfp-callout">
            <i class="fas fa-paperclip"></i>
            <span>选择当前目录后保存聊天附件。</span>
            <button class="gfp-btn gfp-btn--primary gfp-btn--sm" type="button" @click="saveReference">
              <i class="fas fa-save"></i> 保存到当前目录
            </button>
          </div>

          <!-- Loading -->
          <div v-if="loading" class="gfp-state">
            <div class="gfp-spinner"></div>
            <span>正在加载…</span>
          </div>

          <!-- Empty -->
          <div v-else-if="files.length === 0 && currentSubfolders.length === 0" class="gfp-state gfp-state--empty">
            <div class="gfp-empty-icon">
              <i class="fas fa-folder-open"></i>
            </div>
            <p class="gfp-empty-title">当前目录没有文件</p>
            <p class="gfp-empty-desc">点击「上传文件」将文件添加到群文件空间</p>
          </div>

          <!-- Subfolder grid (inline, within content area) -->
          <div v-if="!loading && currentSubfolders.length > 0" class="gfp-subfolders">
            <button
              v-for="folder in currentSubfolders"
              :key="folder.id"
              class="gfp-subfolder-card"
              type="button"
              @click="openFolder(folder.id)"
              @dblclick="openFolder(folder.id)"
            >
              <i class="fas fa-folder gfp-subfolder-card__icon"></i>
              <span class="gfp-subfolder-card__name">{{ folder.name }}</span>
              <i class="fas fa-chevron-right gfp-subfolder-card__arrow"></i>
            </button>
          </div>

          <!-- Table -->
          <table v-if="!loading && files.length > 0" class="gfp-table">
            <colgroup>
              <col class="gfp-col-check" />
              <col class="gfp-col-name" />
              <col class="gfp-col-uploader" />
              <col class="gfp-col-date" />
              <col class="gfp-col-size" />
              <col class="gfp-col-actions" />
            </colgroup>
            <thead>
              <tr>
                <th class="gfp-table__check">
                  <input
                    type="checkbox"
                    :checked="allSelected"
                    :indeterminate.prop="someSelected"
                    aria-label="全选当前页"
                    @change="toggleSelectAll"
                  />
                </th>
                <th>
                  <button class="gfp-sort" :class="{ active: sortField === 'name' }" type="button" @click="toggleSort('name')">
                    名称
                    <i :class="sortField === 'name' ? (sortOrder === 'asc' ? 'fas fa-arrow-up' : 'fas fa-arrow-down') : 'fas fa-sort'"></i>
                  </button>
                </th>
                <th>上传者</th>
                <th>
                  <button class="gfp-sort" :class="{ active: sortField === 'created_at' }" type="button" @click="toggleSort('created_at')">
                    时间
                    <i :class="sortField === 'created_at' ? (sortOrder === 'asc' ? 'fas fa-arrow-up' : 'fas fa-arrow-down') : 'fas fa-sort'"></i>
                  </button>
                </th>
                <th>
                  <button class="gfp-sort" :class="{ active: sortField === 'size' }" type="button" @click="toggleSort('size')">
                    大小
                    <i :class="sortField === 'size' ? (sortOrder === 'asc' ? 'fas fa-arrow-up' : 'fas fa-arrow-down') : 'fas fa-sort'"></i>
                  </button>
                </th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="file in files" :key="file.id" class="gfp-table__row" :class="{ selected: selectedIds.includes(file.id) }">
                <td class="gfp-table__check">
                  <input
                    type="checkbox"
                    :checked="selectedIds.includes(file.id)"
                    :aria-label="`选择 ${file.name}`"
                    @change="toggleSelect(file.id)"
                  />
                </td>
                <td class="gfp-table__name">
                  <div class="gfp-file-icon" :style="{ color: fileIcon(file.name).color, background: fileIcon(file.name).bg }">
                    <i :class="fileIcon(file.name).icon"></i>
                  </div>
                  <Tooltip :text="file.name" overflow-only>
                    <span class="gfp-file-name">{{ file.name }}</span>
                  </Tooltip>
                </td>
                <td class="gfp-table__uploader">{{ uploaderName(file) }}</td>
                <td class="gfp-table__date">{{ formatDate(file.created_at) }}</td>
                <td class="gfp-table__size">{{ formatSize(file.size) }}</td>
                <td class="gfp-table__actions">
                  <button type="button" title="下载" @click="download(file)">
                    <i class="fas fa-download"></i>
                  </button>
                  <button v-if="canManage" type="button" title="移动" @click="move(file)">
                    <i class="fas fa-folder"></i>
                  </button>
                  <button v-if="canManage" type="button" class="gfp-danger" title="删除" @click="remove(file)">
                    <i class="fas fa-trash"></i>
                  </button>
                </td>
              </tr>
            </tbody>
          </table>

          <!-- Pagination -->
          <footer v-if="total > pageSize" class="gfp-pagination">
            <button type="button" :disabled="page === 1" @click="loadFiles(page - 1)">
              <i class="fas fa-chevron-left"></i>
            </button>
            <span class="gfp-pagination__info">第 {{ page }} / {{ Math.ceil(total / pageSize) }} 页</span>
            <button type="button" :disabled="page * pageSize >= total" @click="loadFiles(page + 1)">
              <i class="fas fa-chevron-right"></i>
            </button>
          </footer>
        </main>

        <!-- Drop overlay -->
        <div v-if="dragOver" class="gfp-dropzone">
          <div class="gfp-dropzone__inner">
            <i class="fas fa-cloud-upload-alt"></i>
            <span>松开鼠标即可上传到当前目录</span>
          </div>
        </div>
      </div>

      <!-- Dialog overlay -->
      <div v-if="dialogMode" class="gfp-dialog-backdrop" @click.self="closeDialog">
        <form class="gfp-dialog" role="dialog" aria-modal="true" @submit.prevent="submitDialog">
          <h3 class="gfp-dialog__title">
            <i :class="dialogMode === 'create' ? 'fas fa-folder-plus' : dialogMode === 'move' ? 'fas fa-arrows-alt' : 'fas fa-exclamation-triangle'"></i>
            {{ dialogTitle }}
          </h3>

          <div class="gfp-dialog__body">
            <label v-if="dialogMode === 'create'" class="gfp-field">
              <span class="gfp-field__label">文件夹名称</span>
              <input
                v-model="folderName"
                class="gfp-input"
                aria-label="文件夹名称"
                autocomplete="off"
                maxlength="255"
                autofocus
                placeholder="输入文件夹名称"
              />
            </label>
            <label v-else-if="dialogMode === 'move'" class="gfp-field">
              <span class="gfp-field__label">目标目录</span>
              <select v-model="targetFolderId" class="gfp-input" aria-label="目标目录">
                <option value="">根目录</option>
                <option v-for="folder in folders" :key="folder.id" :value="String(folder.id)">{{ folder.name }}</option>
              </select>
            </label>
            <p v-else-if="dialogMode === 'remove'" class="gfp-dialog__confirm">
              确定删除「<strong>{{ pendingFile?.name }}</strong>」吗？此操作不可恢复。
            </p>
            <p v-else-if="dialogMode === 'batchRemove'" class="gfp-dialog__confirm">
              确定删除选中的 <strong>{{ selectedCount }}</strong> 个文件吗？此操作不可恢复。
            </p>
          </div>

          <div class="gfp-dialog__footer">
            <button class="gfp-btn gfp-btn--ghost" type="button" @click="closeDialog">取消</button>
            <button
              class="gfp-btn"
              :class="(dialogMode === 'remove' || dialogMode === 'batchRemove') ? 'gfp-btn--danger' : 'gfp-btn--primary'"
              type="submit"
              :disabled="batchDeleting"
            >
              <i v-if="batchDeleting" class="fas fa-circle-notch fa-spin"></i>
              {{ (dialogMode === 'remove' || dialogMode === 'batchRemove') ? '删除' : '确认' }}
            </button>
          </div>
        </form>
      </div>

      <!-- Shared upload progress (teleported to body) -->
      <UploadProgressBar :visible="true" />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import QMessage from '../../utils/qmessage'
import { groupFiles, type GroupFile, type GroupFolder } from '../../api/groupFiles'
import { uploadFilesWithLimit } from '../../composables/useFileUpload'
import UploadProgressBar from '../common/UploadProgressBar.vue'
import Tooltip from '../shared/Tooltip.vue'

const props = withDefaults(defineProps<{
  groupId: string | number
  canManage: boolean
  referenceMessageId?: number | null
  referenceFileId?: number | null
}>(), {
  referenceMessageId: null,
  referenceFileId: null,
})

const emit = defineEmits<{ close: [] }>()
const files = ref<GroupFile[]>([])
const folders = ref<GroupFolder[]>([])
const loading = ref(false)
const search = ref('')
const currentFolderId = ref<number | null>(null)
/** Breadcrumb trail: always starts with root */
const breadcrumbs = ref<Array<{ id: number | null; name: string }>>([{ id: null, name: '全部文件' }])
/** Subfolders shown in the file list area (children of currentFolderId) */
const currentSubfolders = ref<GroupFolder[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const latestListRequest = ref(0)
const sortField = ref<'name' | 'size' | 'created_at'>('created_at')
const sortOrder = ref<'asc' | 'desc'>('desc')

/* ---- Batch selection ---- */
const selectedIds = ref<number[]>([])
const batchDeleting = ref(false)
const batchDownloading = ref(false)
const selectedCount = computed(() => selectedIds.value.length)
const allSelected = computed(() =>
  files.value.length > 0 && files.value.every(f => selectedIds.value.includes(f.id)),
)
const someSelected = computed(() =>
  files.value.some(f => selectedIds.value.includes(f.id)) && !allSelected.value,
)
const toggleSelect = (id: number) => {
  const idx = selectedIds.value.indexOf(id)
  if (idx >= 0) selectedIds.value.splice(idx, 1)
  else selectedIds.value.push(id)
}
const toggleSelectAll = () => {
  if (allSelected.value) {
    const ids = new Set(files.value.map(f => f.id))
    selectedIds.value = selectedIds.value.filter(id => !ids.has(id))
  } else {
    const ids = new Set(selectedIds.value)
    files.value.forEach(f => ids.add(f.id))
    selectedIds.value = Array.from(ids)
  }
}
const clearSelection = () => { selectedIds.value = [] }

const toggleSort = (field: 'name' | 'size' | 'created_at') => {
  if (sortField.value === field) {
    sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortField.value = field
    sortOrder.value = field === 'name' ? 'asc' : 'desc'
  }
  page.value = 1
  loadFiles(1)
}
const dialogMode = ref<'create' | 'move' | 'remove' | 'batchRemove' | null>(null)
const folderName = ref('')
const targetFolderId = ref('')
const pendingFile = ref<GroupFile | null>(null)
const dialogTitle = computed(() => ({
  create: '新建文件夹',
  move: '移动文件',
  remove: '删除文件',
  batchRemove: '批量删除文件',
}[dialogMode.value || 'create']))

/* ---- File type icon helper ---- */
const FILE_ICON_MAP: Record<string, { icon: string; color: string; bg: string }> = {
  // 文档
  doc: { icon: 'fas fa-file-word', color: '#2b7cd3', bg: '#e8f0fb' },
  docx: { icon: 'fas fa-file-word', color: '#2b7cd3', bg: '#e8f0fb' },
  pdf: { icon: 'fas fa-file-pdf', color: '#e53935', bg: '#fdecea' },
  txt: { icon: 'fas fa-file-alt', color: '#607d8b', bg: '#eceff1' },
  md: { icon: 'fas fa-file-alt', color: '#607d8b', bg: '#eceff1' },
  // 表格
  xls: { icon: 'fas fa-file-excel', color: '#1b8a3e', bg: '#e8f5e9' },
  xlsx: { icon: 'fas fa-file-excel', color: '#1b8a3e', bg: '#e8f5e9' },
  csv: { icon: 'fas fa-file-excel', color: '#1b8a3e', bg: '#e8f5e9' },
  // PPT
  ppt: { icon: 'fas fa-file-powerpoint', color: '#d04423', bg: '#fbe9e7' },
  pptx: { icon: 'fas fa-file-powerpoint', color: '#d04423', bg: '#fbe9e7' },
  // 图片
  jpg: { icon: 'fas fa-file-image', color: '#00897b', bg: '#e0f2f1' },
  jpeg: { icon: 'fas fa-file-image', color: '#00897b', bg: '#e0f2f1' },
  png: { icon: 'fas fa-file-image', color: '#00897b', bg: '#e0f2f1' },
  gif: { icon: 'fas fa-file-image', color: '#00897b', bg: '#e0f2f1' },
  svg: { icon: 'fas fa-file-image', color: '#00897b', bg: '#e0f2f1' },
  webp: { icon: 'fas fa-file-image', color: '#00897b', bg: '#e0f2f1' },
  // 视频
  mp4: { icon: 'fas fa-file-video', color: '#7b1fa2', bg: '#f3e5f5' },
  avi: { icon: 'fas fa-file-video', color: '#7b1fa2', bg: '#f3e5f5' },
  mov: { icon: 'fas fa-file-video', color: '#7b1fa2', bg: '#f3e5f5' },
  mkv: { icon: 'fas fa-file-video', color: '#7b1fa2', bg: '#f3e5f5' },
  // 音频
  mp3: { icon: 'fas fa-file-audio', color: '#c2185b', bg: '#fce4ec' },
  wav: { icon: 'fas fa-file-audio', color: '#c2185b', bg: '#fce4ec' },
  flac: { icon: 'fas fa-file-audio', color: '#c2185b', bg: '#fce4ec' },
  // 压缩包
  zip: { icon: 'fas fa-file-archive', color: '#f57c00', bg: '#fff3e0' },
  rar: { icon: 'fas fa-file-archive', color: '#f57c00', bg: '#fff3e0' },
  '7z': { icon: 'fas fa-file-archive', color: '#f57c00', bg: '#fff3e0' },
  tar: { icon: 'fas fa-file-archive', color: '#f57c00', bg: '#fff3e0' },
  gz: { icon: 'fas fa-file-archive', color: '#f57c00', bg: '#fff3e0' },
  // 代码
  js: { icon: 'fas fa-file-code', color: '#00838f', bg: '#e0f7fa' },
  ts: { icon: 'fas fa-file-code', color: '#00838f', bg: '#e0f7fa' },
  html: { icon: 'fas fa-file-code', color: '#00838f', bg: '#e0f7fa' },
  css: { icon: 'fas fa-file-code', color: '#00838f', bg: '#e0f7fa' },
  json: { icon: 'fas fa-file-code', color: '#00838f', bg: '#e0f7fa' },
  py: { icon: 'fas fa-file-code', color: '#00838f', bg: '#e0f7fa' },
  go: { icon: 'fas fa-file-code', color: '#00838f', bg: '#e0f7fa' },
  java: { icon: 'fas fa-file-code', color: '#00838f', bg: '#e0f7fa' },
}
const DEFAULT_FILE_ICON = { icon: 'fas fa-file', color: '#9e9e9e', bg: '#f5f5f5' }

const fileIcon = (name: string) => {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  return FILE_ICON_MAP[ext] || DEFAULT_FILE_ICON
}

/* ---- Core logic (unchanged) ---- */
const loadFiles = async (targetPage = page.value) => {
  const requestVersion = ++latestListRequest.value
  loading.value = true
  try {
    const response = await groupFiles.list(props.groupId, {
      folder_id: currentFolderId.value,
      page: targetPage,
      page_size: pageSize.value,
      search: search.value || undefined,
      sort_by: sortField.value,
      sort_order: sortOrder.value,
    })
    if (requestVersion !== latestListRequest.value) return
    const data = response.data.data
    files.value = data.files
    folders.value = data.folders
    currentSubfolders.value = data.folders
    selectedIds.value = []
    page.value = data.page
    pageSize.value = data.page_size
    total.value = data.total
  } catch (error: any) {
    if (requestVersion !== latestListRequest.value) return
    QMessage.error(error?.message || '加载群文件失败')
  } finally {
    if (requestVersion === latestListRequest.value) {
      loading.value = false
    }
  }
}

/* ---- Search: debounce + clear ---- */
let searchTimer: ReturnType<typeof setTimeout> | undefined
const triggerSearch = () => {
  if (searchTimer) clearTimeout(searchTimer)
  page.value = 1
  loadFiles(1)
}
const clearSearch = () => {
  search.value = ''
  if (searchTimer) clearTimeout(searchTimer)
  page.value = 1
  loadFiles(1)
}
watch(search, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    page.value = 1
    loadFiles(1)
  }, 300)
})

const openFolder = (folderId: number | null) => {
  // If clicking root, reset breadcrumbs entirely
  if (folderId === null) {
    breadcrumbs.value = [{ id: null, name: '全部文件' }]
    currentFolderId.value = null
    loadFiles(1)
    return
  }

  // Find if this folder is already in the breadcrumb trail
  const existingIdx = breadcrumbs.value.findIndex(b => b.id === folderId)
  if (existingIdx >= 0) {
    // Truncate trail to this point (re-navigating to an ancestor)
    breadcrumbs.value = breadcrumbs.value.slice(0, existingIdx + 1)
  } else {
    // Push new folder onto trail
    const folderName = currentSubfolders.value.find(f => f.id === folderId)?.name
      || folders.value.find(f => f.id === folderId)?.name
      || '文件夹'
    breadcrumbs.value = [...breadcrumbs.value, { id: folderId, name: folderName }]
  }

  currentFolderId.value = folderId
  loadFiles(1)
}

const goBack = () => {
  if (breadcrumbs.value.length <= 1) return
  breadcrumbs.value = breadcrumbs.value.slice(0, -1)
  const parent = breadcrumbs.value[breadcrumbs.value.length - 1]
  currentFolderId.value = parent.id
  loadFiles(1)
}

const createFolder = async () => {
  folderName.value = ''
  dialogMode.value = 'create'
}

const move = (file: GroupFile) => {
  pendingFile.value = file
  targetFolderId.value = file.folder_id ? String(file.folder_id) : ''
  dialogMode.value = 'move'
}

const remove = (file: GroupFile) => {
  pendingFile.value = file
  dialogMode.value = 'remove'
}

const batchDelete = () => {
  if (selectedIds.value.length === 0) return
  dialogMode.value = 'batchRemove'
}

const batchDownload = async () => {
  if (selectedIds.value.length === 0 || batchDownloading.value) return
  batchDownloading.value = true
  const ids = [...selectedIds.value]
  let success = 0
  let failed = 0
  try {
    for (const id of ids) {
      const file = files.value.find(f => f.id === id)
      try {
        const response = await groupFiles.download(props.groupId, id)
        const url = URL.createObjectURL(response.data)
        const link = document.createElement('a')
        link.href = url
        link.download = file?.name || `file-${id}`
        link.click()
        URL.revokeObjectURL(url)
        success++
      } catch {
        failed++
      }
    }
    if (failed === 0) QMessage.success(`已开始下载 ${success} 个文件`)
    else if (success === 0) QMessage.error('下载群文件失败')
    else QMessage.warning(`下载完成：${success} 个成功，${failed} 个失败`)
  } finally {
    batchDownloading.value = false
  }
}

const closeDialog = () => {
  dialogMode.value = null
  pendingFile.value = null
}

const submitDialog = async () => {
  const mode = dialogMode.value
  if (!mode) return
  try {
    if (mode === 'create') {
      const name = folderName.value.trim()
      if (!name) return
      await groupFiles.createFolder(props.groupId, name, currentFolderId.value)
      await loadFiles(1)
    } else if (mode === 'move' && pendingFile.value) {
      const target = targetFolderId.value ? Number(targetFolderId.value) : null
      await groupFiles.move(props.groupId, pendingFile.value.id, target)
      await loadFiles()
    } else if (mode === 'remove' && pendingFile.value) {
      await groupFiles.remove(props.groupId, pendingFile.value.id)
      await loadFiles()
    } else if (mode === 'batchRemove') {
      const ids = [...selectedIds.value]
      if (ids.length === 0) return
      batchDeleting.value = true
      let success = 0
      let failed = 0
      try {
        for (const id of ids) {
          try {
            await groupFiles.remove(props.groupId, id)
            success++
          } catch {
            failed++
          }
        }
        await loadFiles()
        if (failed === 0) QMessage.success(`已删除 ${success} 个文件`)
        else if (success === 0) QMessage.error('删除群文件失败')
        else QMessage.warning(`删除完成：${success} 个成功，${failed} 个失败`)
      } finally {
        batchDeleting.value = false
      }
    }
    closeDialog()
  } catch (error: any) {
    const fallback = mode === 'create' ? '新建文件夹失败' : mode === 'move' ? '移动文件失败' : '删除文件失败'
    QMessage.error(error?.message || fallback)
  }
}

const saveReference = async () => {
  if (!props.referenceMessageId || !props.referenceFileId) return
  try {
    await groupFiles.shareReference(props.groupId, props.referenceMessageId, props.referenceFileId, currentFolderId.value)
    QMessage.success('已保存到群文件')
    emit('close')
  } catch (error: any) {
    QMessage.error(error?.message || '保存到群文件失败')
  }
}

/* ---- Upload: reuses the generic chunked uploader + progress bar ---- */
const uploading = ref(false)

const uploadFiles = async (fileList: File[] | FileList) => {
  const list = Array.from(fileList)
  if (list.length === 0) return
  uploading.value = true
  try {
    // 使用并发限制上传（最多同时 3 个文件），上传成功后挂载到群文件
    const results = await uploadFilesWithLimit(list, undefined, {
      onFileUploaded: async (_file, fileId) => {
        await groupFiles.attach(props.groupId, fileId, currentFolderId.value)
      }
    })
    const success = results.filter(r => r.success).length
    const failed = results.length - success
    await loadFiles(1)
    if (failed === 0) QMessage.success(`已上传 ${success} 个文件到群文件`)
    else if (success === 0) QMessage.error('上传群文件失败')
    else QMessage.warning(`上传完成：${success} 个成功，${failed} 个失败`)
  } finally {
    uploading.value = false
  }
}

const handleUpload = async (event: Event) => {
  const input = event.target as HTMLInputElement
  if (!input.files || input.files.length === 0) return
  await uploadFiles(input.files)
  input.value = ''
}

/* ---- Drag & drop ---- */
const dragOver = ref(false)
let dragCounter = 0
const onDragEnter = (e: DragEvent) => {
  if (!e.dataTransfer?.types?.includes('Files')) return
  e.preventDefault()
  dragCounter++
  dragOver.value = true
}
const onDragOver = (e: DragEvent) => {
  if (!e.dataTransfer?.types?.includes('Files')) return
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy'
}
const onDragLeave = (e: DragEvent) => {
  e.preventDefault()
  dragCounter = Math.max(0, dragCounter - 1)
  if (dragCounter === 0) dragOver.value = false
}
const onDrop = async (e: DragEvent) => {
  e.preventDefault()
  dragCounter = 0
  dragOver.value = false
  const files = e.dataTransfer?.files
  if (!files || files.length === 0) return
  await uploadFiles(files)
}

const download = async (file: GroupFile) => {
  try {
    const response = await groupFiles.download(props.groupId, file.id)
    const url = URL.createObjectURL(response.data)
    const link = document.createElement('a')
    link.href = url
    link.download = file.name
    link.click()
    URL.revokeObjectURL(url)
  } catch (error: any) {
    QMessage.error(error?.message || '下载群文件失败')
  }
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
  breadcrumbs.value = [{ id: null, name: '全部文件' }]
  loadFiles(1)
}, { immediate: true })

// Mark the modal as open so the shared upload progress bar (teleported to body,
// default z-index 1020) is raised above this modal's backdrop (z-index 3100).
onMounted(() => document.body.classList.add('gfp-modal-open'))
onUnmounted(() => document.body.classList.remove('gfp-modal-open'))
</script>

<style scoped>
/* ===================================================
   GroupFilesPanel — Modern UI
   =================================================== */

/* --- Backdrop & Panel Shell --- */
.group-files-backdrop {
  position: fixed;
  inset: 0;
  z-index: 3100;
  display: grid;
  place-items: center;
  background: rgba(0, 0, 0, .45);
  backdrop-filter: blur(4px);
  animation: gfp-fadeIn .2s ease;
}

.group-files-panel {
  width: min(820px, calc(100vw - 32px));
  height: min(560px, calc(100vh - 32px));
  display: flex;
  flex-direction: column;
  border-radius: var(--radius-lg, 12px);
  background: var(--card-bg, #fff);
  color: var(--text-color, #222);
  box-shadow: var(--shadow-2xl, 0 25px 50px -12px rgba(0,0,0,.25));
  overflow: hidden;
  animation: gfp-scaleIn .25s cubic-bezier(.16,1,.3,1);
}

/* --- Header --- */
.gfp-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-5, 20px) var(--spacing-6, 24px);
  border-bottom: 1px solid var(--border-color, #e5e7eb);
  background: var(--color-gray-50, #fefefe);
}

.gfp-header__info { min-width: 0; }

.gfp-header__title {
  margin: 0;
  font-size: var(--font-size-xl, 20px);
  font-weight: var(--font-weight-semibold, 600);
  display: flex;
  align-items: center;
  gap: 10px;
}

.gfp-header__title i {
  color: var(--primary-color, #3385ff);
}

.gfp-header__subtitle {
  margin: 4px 0 0;
  font-size: var(--font-size-xs, 12px);
  color: var(--text-secondary, #6b7280);
}

.gfp-close {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: var(--radius-full, 50%);
  background: transparent;
  color: var(--text-secondary, #6b7280);
  font-size: var(--font-size-sm);
  cursor: pointer;
  transition: all var(--transition-fast, 150ms ease);
  flex-shrink: 0;
}

.gfp-close:hover {
  background: var(--hover-color, #f3f4f6);
  color: var(--text-color, #222);
}

/* --- Toolbar --- */
.gfp-toolbar {
  display: flex;
  align-items: center;
  gap: var(--spacing-3, 12px);
  padding: var(--spacing-3, 12px) var(--spacing-6, 24px);
  border-bottom: 1px solid var(--border-color, #e5e7eb);
}

.gfp-toolbar__spacer { flex: 1; }

.gfp-search {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 0 10px;
  height: 32px;
  flex: 0 0 auto;
  width: clamp(200px, 30vw, 300px);
  border: 1px solid var(--border-color, #d1d5db);
  border-radius: 16px;
  transition: border-color var(--transition-fast, 150ms ease);
}

.gfp-search:focus-within {
  border-color: var(--primary-color, #3385ff);
  box-shadow: 0 0 0 2px rgba(51, 133, 255, .15);
}

.gfp-search i {
  color: var(--text-secondary, #a0a0a0);
  font-size: var(--font-size-xxs);
}

.gfp-search input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: inherit;
  font-size: var(--font-size-xs, 12px);
}

.gfp-search input::placeholder {
  color: var(--text-secondary, #a0a0a0);
}

.gfp-search__clear {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  border: 0;
  border-radius: 50%;
  background: var(--color-gray-200, #e5e5e5);
  color: var(--text-secondary, #6b7280);
  font-size: 9px;
  cursor: pointer;
  transition: all var(--transition-fast, 150ms ease);
}

.gfp-search__clear:hover {
  background: var(--color-gray-400, #c0c0c0);
  color: #fff;
}

/* --- Buttons --- */
.gfp-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 0;
  border-radius: var(--radius-md, 8px);
  padding: 7px 14px;
  font-size: var(--font-size-xs, 12px);
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast, 150ms ease);
  white-space: nowrap;
}

.gfp-btn--primary {
  background: var(--primary-color, #3385ff);
  color: #fff;
}

.gfp-btn--primary:hover {
  background: var(--primary-dark, #1a6bff);
  box-shadow: 0 2px 8px rgba(51, 133, 255, .3);
}

.gfp-btn.is-loading {
  opacity: .75;
  cursor: wait;
  pointer-events: none;
}

.gfp-btn--secondary {
  background: transparent;
  color: var(--text-color, #222);
  border: 1px solid var(--border-color, #d1d5db);
}

.gfp-btn--secondary:hover {
  background: var(--hover-color, #f3f4f6);
  border-color: var(--color-gray-400, #c0c0c0);
}

.gfp-btn--ghost {
  background: transparent;
  color: var(--text-secondary, #6b7280);
}

.gfp-btn--ghost:hover {
  background: var(--hover-color, #f3f4f6);
  color: var(--text-color, #222);
}

.gfp-btn--danger {
  background: var(--error-color, #f34040);
  color: #fff;
}

.gfp-btn--danger:hover {
  background: var(--color-error-600, #c23030);
  box-shadow: 0 2px 8px rgba(243, 64, 64, .3);
}

.gfp-btn--sm {
  padding: 5px 10px;
  font-size: var(--font-size-xxs);
}

.gfp-btn:has(input[type="file"]) input {
  display: none;
}

/* --- Batch toolbar --- */
.gfp-batch-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: var(--spacing-3, 12px);
  padding: 8px 14px;
  border-radius: var(--radius-md, 8px);
  background: var(--primary-light, #f8faff);
  border: 1px solid var(--color-primary-200, #cce0ff);
  font-size: var(--font-size-xs, 12px);
}

.gfp-batch-bar__count {
  color: var(--primary-dark, #0052cc);
  font-weight: 500;
}

.gfp-batch-bar__actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

/* --- Content Layout --- */
.gfp-content {
  position: relative;
  display: flex;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.gfp-content.is-dragover::after {
  content: '';
  position: absolute;
  inset: 8px;
  border: 2px dashed var(--primary-color, #3385ff);
  border-radius: var(--radius-md, 8px);
  background: rgba(51, 133, 255, .06);
  pointer-events: none;
  z-index: 5;
}

.gfp-dropzone {
  position: absolute;
  inset: 0;
  z-index: 6;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
}

.gfp-dropzone__inner {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 28px 40px;
  border-radius: var(--radius-lg, 12px);
  background: var(--card-bg, #fff);
  box-shadow: var(--shadow-xl, 0 20px 25px -5px rgba(0,0,0,.1));
  color: var(--primary-color, #3385ff);
  font-size: var(--font-size-sm, 14px);
  font-weight: 500;
}

.gfp-dropzone__inner i {
  font-size: var(--font-size-3xl);
}

/* --- File List --- */
.gfp-files {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  padding: var(--spacing-4, 16px) var(--spacing-5, 20px);
}

/* Reference callout */
.gfp-callout {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: var(--spacing-3, 12px);
  padding: 10px 14px;
  border-radius: var(--radius-md, 8px);
  background: var(--primary-light, #f8faff);
  border: 1px solid var(--color-primary-200, #cce0ff);
  font-size: var(--font-size-xs, 12px);
  color: var(--primary-dark, #0052cc);
}

.gfp-callout > i {
  color: var(--primary-color, #3385ff);
  flex-shrink: 0;
}

.gfp-callout span { flex: 1; }

/* Loading / Empty states */
.gfp-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  flex: 1;
  padding: var(--spacing-12, 48px);
  color: var(--text-secondary, #6b7280);
  font-size: var(--font-size-sm, 14px);
}

.gfp-spinner {
  width: 28px;
  height: 28px;
  border: 3px solid var(--border-color, #e5e7eb);
  border-top-color: var(--primary-color, #3385ff);
  border-radius: 50%;
  animation: gfp-spin .7s linear infinite;
}

.gfp-state--empty {
  gap: 8px;
}

.gfp-empty-icon {
  width: 64px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-full, 50%);
  background: var(--color-gray-100, #f8f8f8);
  margin-bottom: 8px;
}

.gfp-empty-icon i {
  font-size: var(--font-size-2xl);
  color: var(--color-gray-400, #c0c0c0);
}

.gfp-empty-title {
  margin: 0;
  font-weight: var(--font-weight-medium, 500);
  color: var(--text-color, #222);
}

.gfp-empty-desc {
  margin: 0;
  font-size: var(--font-size-xs, 12px);
}

/* --- Table --- */
.gfp-table {
  width: 100%;
  table-layout: fixed;
  border-collapse: collapse;
  font-size: var(--font-size-xs, 12px);
}

/* 固定列宽：名称列吃掉剩余空间，超长文件名靠 ellipsis 截断而非撑破表格 */
.gfp-col-check { width: 40px; }
.gfp-col-uploader { width: 110px; }
.gfp-col-date { width: 132px; }
.gfp-col-size { width: 78px; }
.gfp-col-actions { width: 116px; }

.gfp-table thead th {
  padding: 8px 10px;
  text-align: left;
  font-weight: 500;
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary, #a0a0a0);
  text-transform: uppercase;
  letter-spacing: .3px;
  border-bottom: 1px solid var(--border-color, #e5e7eb);
}

.gfp-sort {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-transform: inherit;
  letter-spacing: inherit;
  cursor: pointer;
  padding: 0;
  transition: color var(--transition-fast, 150ms ease);
}

.gfp-sort i {
  font-size: 9px;
  opacity: .5;
}

.gfp-sort:hover {
  color: var(--text-color, #222);
}

.gfp-sort.active {
  color: var(--primary-color, #3385ff);
}

.gfp-sort.active i {
  opacity: 1;
}

.gfp-table__row {
  transition: background var(--transition-fast, 150ms ease);
}

.gfp-table__row:hover {
  background: var(--color-gray-50, #fefefe);
}

.gfp-table__row.selected {
  background: var(--primary-light, #f8faff);
}

.gfp-table__row.selected:hover {
  background: var(--color-primary-100, #e8f1ff);
}

/* Checkbox column */
.gfp-table__check {
  width: 36px;
}

.gfp-table thead th.gfp-table__check {
  padding: 8px 8px;
  text-align: center;
}

.gfp-table__row td.gfp-table__check {
  padding: 10px 8px;
  text-align: center;
}

.gfp-table__check input[type="checkbox"] {
  width: 15px;
  height: 15px;
  cursor: pointer;
  accent-color: var(--primary-color, #3385ff);
}

.gfp-table__row td {
  padding: 10px;
  border-bottom: 1px solid var(--border-color, #e5e7eb);
  vertical-align: middle;
}

/* Name cell */
.gfp-table__name {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.gfp-file-icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md, 8px);
  flex-shrink: 0;
  font-size: var(--font-size-sm);
}

.gfp-file-name {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 500;
  min-width: 0;
}

/* Secondary cells */
.gfp-table__uploader,
.gfp-table__date {
  color: var(--text-secondary, #6b7280);
  white-space: nowrap;
}

.gfp-table__size {
  color: var(--text-secondary, #6b7280);
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

/* Actions cell */
.gfp-table__actions {
  display: flex;
  align-items: center;
  gap: 4px;
  opacity: 0;
  transition: opacity var(--transition-fast, 150ms ease);
}

.gfp-table__row:hover .gfp-table__actions {
  opacity: 1;
}

.gfp-table__actions button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: var(--radius-sm, 4px);
  background: transparent;
  color: var(--text-secondary, #6b7280);
  font-size: var(--font-size-xxs);
  cursor: pointer;
  transition: all var(--transition-fast, 150ms ease);
}

.gfp-table__actions button:hover {
  background: var(--hover-color, #f3f4f6);
  color: var(--text-color, #222);
}

.gfp-table__actions .gfp-danger:hover {
  background: var(--color-error-50, #fff6f6);
  color: var(--error-color, #f34040);
}

/* --- Pagination --- */
.gfp-pagination {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  padding-top: var(--spacing-4, 16px);
  font-size: var(--font-size-xs, 12px);
  color: var(--text-secondary, #6b7280);
}

.gfp-pagination button {
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-color, #d1d5db);
  border-radius: var(--radius-md, 8px);
  background: transparent;
  color: var(--text-color, #222);
  font-size: var(--font-size-xxxs);
  cursor: pointer;
  transition: all var(--transition-fast, 150ms ease);
}

.gfp-pagination button:hover:not(:disabled) {
  background: var(--hover-color, #f3f4f6);
  border-color: var(--color-gray-400, #c0c0c0);
}

.gfp-pagination button:disabled {
  opacity: .35;
  cursor: not-allowed;
}

.gfp-pagination__info {
  font-variant-numeric: tabular-nums;
  min-width: 80px;
  text-align: center;
}

/* --- Dialog --- */
.gfp-dialog-backdrop {
  position: absolute;
  inset: 0;
  z-index: 10;
  display: grid;
  place-items: center;
  background: rgba(0, 0, 0, .35);
  backdrop-filter: blur(2px);
  animation: gfp-fadeIn .15s ease;
}

.gfp-dialog {
  width: min(380px, calc(100vw - 40px));
  display: flex;
  flex-direction: column;
  padding: var(--spacing-5, 20px);
  border-radius: var(--radius-lg, 12px);
  background: var(--card-bg, #fff);
  box-shadow: var(--shadow-xl, 0 20px 25px -5px rgba(0,0,0,.1));
  animation: gfp-scaleIn .2s cubic-bezier(.16,1,.3,1);
}

.gfp-dialog__title {
  margin: 0 0 var(--spacing-4, 16px);
  font-size: var(--font-size-sm, 14px);
  font-weight: var(--font-weight-semibold, 600);
  display: flex;
  align-items: center;
  gap: 8px;
}

.gfp-dialog__title i {
  color: var(--primary-color, #3385ff);
}

.gfp-dialog__body {
  margin-bottom: var(--spacing-5, 20px);
}

.gfp-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.gfp-field__label {
  font-size: var(--font-size-xs, 12px);
  color: var(--text-secondary, #6b7280);
  font-weight: 500;
}

.gfp-input {
  width: 100%;
  box-sizing: border-box;
  padding: 8px 12px;
  border: 1px solid var(--border-color, #d1d5db);
  border-radius: var(--radius-md, 8px);
  background: var(--input-bg, #fff);
  color: inherit;
  font-size: var(--font-size-sm, 14px);
  transition: border-color var(--transition-fast, 150ms ease), box-shadow var(--transition-fast, 150ms ease);
}

.gfp-input:focus {
  outline: 0;
  border-color: var(--primary-color, #3385ff);
  box-shadow: 0 0 0 2px rgba(51, 133, 255, .15);
}

.gfp-dialog__confirm {
  margin: 0;
  font-size: var(--font-size-sm, 14px);
  line-height: 1.6;
  color: var(--text-color, #222);
}

.gfp-dialog__confirm strong {
  font-weight: 600;
}

.gfp-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

/* --- Breadcrumb --- */
.gfp-breadcrumb {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
  margin-bottom: var(--spacing-3, 12px);
  font-size: var(--font-size-xs, 12px);
}

.gfp-breadcrumb__item {
  border: 0;
  background: transparent;
  color: var(--primary-color, #3385ff);
  cursor: pointer;
  padding: 2px 6px;
  border-radius: var(--radius-sm, 4px);
  font-size: inherit;
  font-weight: 500;
  transition: all var(--transition-fast, 150ms ease);
}

.gfp-breadcrumb__item:hover {
  background: var(--hover-color, #f3f4f6);
  text-decoration: underline;
}

.gfp-breadcrumb__item.active {
  color: var(--text-color, #222);
  cursor: default;
  font-weight: 600;
}

.gfp-breadcrumb__item.active:hover {
  background: transparent;
  text-decoration: none;
}

.gfp-breadcrumb__sep {
  color: var(--color-gray-400, #c0c0c0);
  font-size: 9px;
}

/* --- Back button --- */
.gfp-back-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 0;
  background: transparent;
  color: var(--text-secondary, #6b7280);
  cursor: pointer;
  padding: 4px 8px;
  border-radius: var(--radius-sm, 4px);
  font-size: var(--font-size-xs, 12px);
  margin-bottom: var(--spacing-3, 12px);
  transition: all var(--transition-fast, 150ms ease);
  align-self: flex-start;
}

.gfp-back-btn:hover {
  background: var(--hover-color, #f3f4f6);
  color: var(--primary-color, #3385ff);
}

.gfp-back-btn i {
  font-size: var(--font-size-xxxs);
}

/* --- Subfolder cards (inline in file list) --- */
.gfp-subfolders {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-bottom: var(--spacing-3, 12px);
}

.gfp-subfolder-card {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  border: 0;
  border-radius: var(--radius-md, 8px);
  background: transparent;
  color: var(--text-color, #222);
  cursor: pointer;
  padding: 9px 12px;
  text-align: left;
  font-size: var(--font-size-xs, 12px);
  transition: all var(--transition-fast, 150ms ease);
}

.gfp-subfolder-card:hover {
  background: var(--hover-color, #f3f4f6);
}

.gfp-subfolder-card:hover .gfp-subfolder-card__icon {
  color: var(--primary-color, #3385ff);
}

.gfp-subfolder-card:hover .gfp-subfolder-card__arrow {
  opacity: 1;
  transform: translateX(2px);
}

.gfp-subfolder-card__icon {
  color: var(--color-warning-500, #f7a826);
  font-size: var(--font-size-base);
  flex-shrink: 0;
  transition: color var(--transition-fast, 150ms ease);
}

.gfp-subfolder-card__name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 500;
}

.gfp-subfolder-card__arrow {
  color: var(--text-secondary, #a0a0a0);
  font-size: var(--font-size-tiny);
  opacity: 0;
  transition: all var(--transition-fast, 150ms ease);
  flex-shrink: 0;
}

/* --- Animations --- */
@keyframes gfp-fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes gfp-scaleIn {
  from { opacity: 0; transform: scale(.96); }
  to { opacity: 1; transform: scale(1); }
}

@keyframes gfp-spin {
  to { transform: rotate(360deg); }
}

/* --- Responsive --- */
@media (max-width: 650px) {
  .gfp-toolbar {
    flex-wrap: wrap;
  }

  .gfp-toolbar__spacer {
    display: none;
  }

  .gfp-search {
    width: 100%;
    margin-top: 4px;
  }

  .gfp-search input {
    flex: 1;
    width: auto;
  }

  .gfp-col-uploader,
  .gfp-col-date {
    width: 0;
  }

  .gfp-table thead th:nth-child(3),
  .gfp-table__row td:nth-child(3),
  .gfp-table thead th:nth-child(4),
  .gfp-table__row td:nth-child(4) {
    display: none;
  }

  .gfp-table__actions {
    opacity: 1;
  }
}
</style>

<!-- Non-scoped: raise the shared upload progress bar above this modal while open.
     Scoped to the body class set in onMounted so FileManagementApp's instance is unaffected. -->
<style>
body.gfp-modal-open .upload-progress-bar {
  z-index: 3200;
}
</style>

<template>
  <div class="notes-app" :class="{ fullscreen: isFullscreen }">
    <AppHeader title="笔记" @back="$emit('back')" v-show="!isFullscreen">
      <template #actions>
        <button class="import-btn" @click="triggerImport"><i class="fas fa-file-import"></i> 导入</button>
        <button class="create-note-btn" @click="handleCreate">+ 新建笔记</button>
      </template>
    </AppHeader>
    
    <div class="notes-content">
      <div class="notes-sidebar" v-show="!isFullscreen">
        <div class="notes-search-box">
          <input
            v-model="searchQuery"
            type="text"
            class="notes-search-input"
            placeholder="搜索笔记..."
          />
          <i class="fas fa-search notes-search-icon"></i>
        </div>
        <NoteTagFilter
          :all-tags="allTags"
          :selected-tag="selectedTag"
          @select="selectedTag = $event"
          @clear="selectedTag = null"
        />
        <div class="notes-list">
          <NoteCard
            v-for="note in filteredNotes"
            :key="note.id"
            :note="note"
            :is-active="selectedNoteId === note.id"
            @select="selectNote(note.id)"
            @edit="editNote(note)"
            @delete="handleDelete(note.id)"
            @filter-tag="selectedTag = $event"
            @toggle-ai-access="toggleAiAccess(note.id, note.ai_accessible === false)"
            @contextmenu.prevent="handleNoteContextMenu(note, $event)"
          />
          <div v-if="filteredNotes.length === 0" class="empty-notes">
            <p>没有找到匹配的笔记</p>
          </div>
        </div>
      </div>
      
      <div class="note-main">
        <template v-if="selectedNote">
          <NoteToolbar
            v-model:mode="editorMode"
            :saving="saving"
            :save-status="saveStatus"
            :analyzing="analyzing"
            :formatting="formatting"
            :fullscreen="isFullscreen"
            :ai-accessible="selectedNote.ai_accessible !== false"
            @update:ai-accessible="toggleAiAccess(selectedNote.id, $event)"
            @format="handleFormat"
            @insert-link="handleInsertLink"
            @save="handleSave"
            @analyze="handleAnalyze"
            @ai-format="handleAiFormat"
            @import="triggerImport"
            @export="handleExport"
            @share="handleShare"
            @delete="handleDelete(selectedNote.id)"
            @toggle-fullscreen="toggleFullscreen"
            @show-shortcuts="showShortcutsHelp = true"
            @insert-table="noteEditorRef?.insertTable()"
            @insert-code-block="noteEditorRef?.insertBlock('```\n', '\n```')"
          />
          <NoteEditor
            ref="noteEditorRef"
            v-model:title="selectedNote.title"
            v-model:content="selectedNote.content"
            :mode="editorMode"
            :note-list="noteListForLinks"
            @save="handleSave"
            @navigate-note="handleNavigateNote"
          />
        </template>
        <div v-else class="empty-note">
          <div class="empty-icon"><i class="fas fa-book"></i></div>
          <p>选择一个笔记或创建新笔记</p>
        </div>
      </div>
    </div>
    
    <input
      ref="fileInputRef"
      type="file"
      accept=".md,.markdown,.txt,.html,.htm,.json,.docx,.pdf"
      style="display: none"
      @change="handleFileSelect"
    />
    
    <AIAnalysisModal
      :visible="showAnalysisModal"
      :result="analysisResult"
      @close="showAnalysisModal = false"
      @confirm="handleAnalysisConfirm"
    />

    <AiFormatModal
      :visible="showFormatModal"
      :original="selectedNote?.content || ''"
      :formatted="formatResult?.content || ''"
      :truncated="formatResult?.truncated"
      @close="showFormatModal = false"
      @confirm="handleFormatConfirm"
    />

    <UniversalContextMenu menuId="note-card" :items="noteContextMenuItems" />

    <!-- 快捷键帮助弹窗 -->
    <Teleport to="body">
      <div v-if="showShortcutsHelp" class="shortcuts-overlay" @click.self="showShortcutsHelp = false">
        <div class="shortcuts-modal">
          <div class="shortcuts-header">
            <h3><i class="fas fa-keyboard"></i> 快捷键</h3>
            <button class="shortcuts-close" @click="showShortcutsHelp = false"><i class="fas fa-times"></i></button>
          </div>
          <div class="shortcuts-body">
            <div class="shortcuts-group">
              <h4>通用</h4>
              <div class="shortcut-item"><span class="shortcut-desc">保存</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>S</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">新建笔记</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>N</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">搜索替换</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>H</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">查找</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>F</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">全屏</span><span class="shortcut-keys"><kbd>F11</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">显示快捷键</span><span class="shortcut-keys"><kbd>?</kbd></span></div>
            </div>
            <div class="shortcuts-group">
              <h4>格式</h4>
              <div class="shortcut-item"><span class="shortcut-desc">粗体</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>B</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">斜体</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>I</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">删除线</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>X</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">链接</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>K</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">行内代码</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>C</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">代码块</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>B</kbd></span></div>
            </div>
            <div class="shortcuts-group">
              <h4>结构</h4>
              <div class="shortcut-item"><span class="shortcut-desc">一级标题</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>1</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">二级标题</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>2</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">三级标题</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>3</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">无序列表</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>U</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">有序列表</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>O</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">任务列表</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>T</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">引用</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>Q</kbd></span></div>
            </div>
            <div class="shortcuts-group">
              <h4>编辑</h4>
              <div class="shortcut-item"><span class="shortcut-desc">撤销</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>Z</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">重做</span><span class="shortcut-keys"><kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>Z</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">缩进</span><span class="shortcut-keys"><kbd>Tab</kbd></span></div>
              <div class="shortcut-item"><span class="shortcut-desc">减少缩进</span><span class="shortcut-keys"><kbd>Shift</kbd>+<kbd>Tab</kbd></span></div>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import AppHeader from './AppHeader.vue'
import NoteCard from './notes/NoteCard.vue'
import NoteToolbar from './notes/NoteToolbar.vue'
import NoteEditor from './notes/NoteEditor.vue'
import NoteTagFilter from './notes/NoteTagFilter.vue'
import AIAnalysisModal from './notes/AIAnalysisModal.vue'
import AiFormatModal from './notes/AiFormatModal.vue'
import UniversalContextMenu from '../shared/UniversalContextMenu.vue'
import { useNotes } from '../../composables/useNotes'
import { useAutoSave } from '../../composables/useAutoSave'
import { useNoteDraft } from '../../composables/useNoteDraft'
import { openMenu } from '../../composables/useUI'
import type { ContextMenuItem } from '../shared/context-menu-types'
import QMessage from '../../utils/qmessage'
import QMessageBox from '../../utils/qmessagebox'
import type { Note, AIAnalyzeResult, NoteFormatResult } from '../../types/note'
import mammoth from 'mammoth'
import * as pdfjsLib from 'pdfjs-dist'
import pdfjsWorkerUrl from 'pdfjs-dist/build/pdf.worker.min.mjs?url'
pdfjsLib.GlobalWorkerOptions.workerSrc = pdfjsWorkerUrl

const emit = defineEmits(['back'])

const {
  fetchNotes,
  createNote,
  updateNote,
  deleteNote,
  analyzeNote,
  formatNote,
  updateNoteTags,
  updateNoteSummary,
  exportNote,
  setNoteAiAccessible,
  error: notesError
} = useNotes()

const notes = ref<Note[]>([])
const selectedNoteId = ref<number | null>(null)
const selectedNote = ref<Note | null>(null)
const searchQuery = ref('')
const selectedTag = ref<string | null>(null)
const editorMode = ref<'edit' | 'split' | 'preview'>('edit')
const analyzing = ref(false)
const showAnalysisModal = ref(false)
const analysisResult = ref<AIAnalyzeResult | null>(null)
const formatting = ref(false)
const showFormatModal = ref(false)
const formatResult = ref<NoteFormatResult | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)
const isFullscreen = ref(false)
const noteEditorRef = ref<InstanceType<typeof NoteEditor> | null>(null)
const showShortcutsHelp = ref(false)

const draft = useNoteDraft()

// 自动保存：监听 title/content 变化，防抖 5 秒后调用 updateNote
// localStorage 即时落盘保底，服务器同步为后台行为
const autoSave = useAutoSave(
  (id: number, data) => updateNote(id, data),
  { delay: 5000, maxWait: 30000 }
)
// 手动保存进行中标志，与自动保存状态合并后用于禁用保存按钮
const manualSaving = ref(false)
const saving = computed(() => manualSaving.value || autoSave.status.value === 'saving')
// 本地草稿已落盘但后端尚未同步标志
const draftSaved = ref(false)
const saveStatus = computed(() => {
  // 手动保存/后端同步中优先显示
  if (manualSaving.value || autoSave.status.value === 'saving') return 'saving'
  // 后端同步成功
  if (autoSave.status.value === 'saved') return 'saved'
  // 后端保存失败
  if (autoSave.status.value === 'error') return 'error'
  // 有未同步的本地草稿
  if (draftSaved.value) return 'draft'
  return 'idle'
})
// 切换笔记时跳过一次自动保存 watch，避免把"载入新笔记内容"误判为用户编辑
let skipNextAutoSave = false
// 跟踪自动保存对应的笔记 ID，供 status → saved 时清理 localStorage 草稿
let autoSavedNoteId: number | null = null
// 右键菜单：当前右键的笔记
const contextMenuNote = ref<Note | null>(null)

function handleNoteContextMenu(note: Note, event: MouseEvent) {
  contextMenuNote.value = note
  openMenu('note-card', event.clientX, event.clientY)
}

const noteContextMenuItems = computed<ContextMenuItem[]>(() => {
  const note = contextMenuNote.value
  if (!note) return []
  return [
    {
      label: '编辑',
      icon: 'fas fa-edit',
      action: () => { editNote(note) }
    },
    {
      label: 'AI 分析',
      icon: 'fas fa-robot',
      action: () => { selectNote(note.id).then(() => handleAnalyze()) }
    },
    {
      label: '导出',
      icon: 'fas fa-download',
      action: () => { exportNote(note.id, note.title) }
    },
    {
      label: '分享',
      icon: 'fas fa-share-alt',
      action: () => {
        window.dispatchEvent(new CustomEvent('openShareModal', {
          detail: { type: 'note', data: note }
        }))
      }
    },
    {
      label: note.ai_accessible === false ? '允许分身读取' : '禁止分身读取',
      icon: note.ai_accessible === false ? 'fas fa-eye' : 'fas fa-eye-slash',
      action: () => { toggleAiAccess(note.id, note.ai_accessible === false) }
    },
    { divider: true },
    {
      label: '删除',
      icon: 'fas fa-trash',
      danger: true,
      action: () => { handleDelete(note.id) }
    }
  ]
})

const allTags = computed(() => {
  const tags = new Set<string>()
  notes.value.forEach(n => n.tags?.forEach(t => tags.add(t)))
  return Array.from(tags)
})

// 供 NoteEditor 内链使用
const noteListForLinks = computed(() => notes.value.map(n => ({ id: n.id, title: n.title })))

function handleNavigateNote(noteId: number) {
  const note = notes.value.find(n => n.id === noteId)
  if (note) selectNote(note.id)
}

const filteredNotes = computed(() => {
  let result = notes.value
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    result = result.filter(n => 
      n.title.toLowerCase().includes(q) || 
      n.content.toLowerCase().includes(q)
    )
  }
  if (selectedTag.value) {
    result = result.filter(n => n.tags?.includes(selectedTag.value!))
  }
  return result
})

async function selectNote(id: number) {
  const oldNoteId = selectedNoteId.value
  // 切换前先把当前笔记的待保存内容落库，避免把 A 笔记的内容写到 B 笔记
  const flushResult = await autoSave.flush()
  // 仅在确认落库成功、且保存期间没有新编辑（flush 后无 pending）时才清理草稿；
  // 否则草稿里是最新的未同步内容，清掉会丢数据（flush 失败 / 保存期间又输入）
  if (oldNoteId !== null && flushResult === 'saved' && autoSave.status.value !== 'pending') {
    draft.clear(oldNoteId)
  }
  skipNextAutoSave = true
  draftSaved.value = false
  selectedNoteId.value = id
  selectedNote.value = notes.value.find(n => n.id === id) || null
  // 恢复 localStorage 草稿（断网、崩溃等未同步的内容）
  nextTick(() => restoreDraft())
}

function editNote(note: Note) {
  void selectNote(note.id)
}

function toggleFullscreen() {
  isFullscreen.value = !isFullscreen.value
}

function handleFormat(prefix: string, suffix: string) {
  noteEditorRef.value?.insertFormat(prefix, suffix)
}

function handleInsertLink() {
  noteEditorRef.value?.insertLink()
}

async function handleCreate() {
  const note = await createNote({ title: '新笔记', content: '' })
  if (note) {
    notes.value.unshift(note)
    selectNote(note.id)
  }
}

function triggerImport() {
  fileInputRef.value?.click()
}

async function handleFileSelect(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  const ext = file.name.split('.').pop()?.toLowerCase() || ''
  const baseName = file.name.replace(/\.[^.]+$/, '')

  const reader = new FileReader()
  reader.onload = async (e) => {
    const raw = e.target?.result as string

    try {
      // DOCX → mammoth 提取 HTML 后转纯文本
      if (ext === 'docx') {
        const arrayBuffer = e.target?.result as ArrayBuffer
        const { value: html } = await mammoth.convertToHtml({ arrayBuffer })
        const doc = new DOMParser().parseFromString(html, 'text/html')
        const content = doc.body?.innerText?.trim() || ''
        const note = await createNote({ title: baseName, content })
        if (note) { notes.value.unshift(note); selectNote(note.id) }
        QMessage.success('导入成功')
        return
      }

      // PDF → pdfjs-dist 逐页提取文本
      if (ext === 'pdf') {
        const arrayBuffer = e.target?.result as ArrayBuffer
        const pdf = await pdfjsLib.getDocument({ data: arrayBuffer }).promise
        const pages: string[] = []
        for (let i = 1; i <= pdf.numPages; i++) {
          const page = await pdf.getPage(i)
          const textContent = await page.getTextContent()
          const lines: string[] = []
          let line = ''
          for (const item of textContent.items as any[]) {
            line += item.str
            if (item.hasEOL) {
              lines.push(line)
              line = ''
            }
          }
          if (line) lines.push(line)
          pages.push(lines.join('\n'))
        }
        const content = pages.join('\n\n').trim()
        const note = await createNote({ title: baseName, content })
        if (note) { notes.value.unshift(note); selectNote(note.id) }
        QMessage.success('导入成功')
        return
      }

      // HTML → 提取 body 纯文本
      if (ext === 'html' || ext === 'htm') {
        const doc = new DOMParser().parseFromString(raw, 'text/html')
        const content = doc.body?.innerText?.trim() || raw
        const note = await createNote({ title: baseName, content })
        if (note) { notes.value.unshift(note); selectNote(note.id) }
        QMessage.success('导入成功')
        return
      }

      // JSON → 单条或数组
      if (ext === 'json') {
        let data = JSON.parse(raw)
        if (!Array.isArray(data)) data = [data]
        const created: Note[] = []
        for (const item of data) {
          const title = item.title || baseName
          const content = item.content || ''
          if (!content && !item.title) continue
          const note = await createNote({ title, content })
          if (note) created.push(note)
        }
        if (created.length) {
          notes.value = [...created, ...notes.value]
          selectNote(created[0].id)
          QMessage.success(`成功导入 ${created.length} 篇笔记`)
        } else {
          QMessage.warning('未找到有效笔记数据')
        }
        return
      }

      // .md / .markdown / .txt / 其他 → 直接读文本
      const note = await createNote({ title: baseName, content: raw })
      if (note) {
        notes.value.unshift(note)
        selectNote(note.id)
        QMessage.success('导入成功')
      }
    } catch (err) {
      console.error('导入失败:', err)
      QMessage.error('导入失败，请检查文件格式')
    }
  }
  // docx/pdf 需要 ArrayBuffer，其余用文本
  if (ext === 'docx' || ext === 'pdf') {
    reader.readAsArrayBuffer(file)
  } else {
    reader.readAsText(file)
  }
  input.value = ''
}

async function handleSave() {
  if (!selectedNote.value) return
  // 取消可能正在 pending 的自动保存，避免与手动保存重复请求
  autoSave.cancel()
  manualSaving.value = true
  // 保存内容快照：保存期间用户若继续输入，草稿保留新内容，不清不丢
  const snapshot = {
    title: selectedNote.value.title,
    content: selectedNote.value.content,
    tags: selectedNote.value.tags
  }
  const ok = await updateNote(selectedNote.value.id, snapshot)
  manualSaving.value = false
  if (ok) {
    // 同步状态：保存期间若有新编辑则保持 pending，否则置 saved
    autoSave.markManuallySaved()
    // 仅当保存期间没有新编辑（内容与快照一致）才清理草稿与草稿标志
    if (selectedNote.value.title === snapshot.title && selectedNote.value.content === snapshot.content) {
      draft.clear(selectedNote.value.id)
      draftSaved.value = false
    }
    QMessage.success('保存成功')
  } else {
    QMessage.error(notesError.value || '保存失败')
  }
}

async function handleAnalyze() {
  if (!selectedNote.value) return
  analyzing.value = true
  const result = await analyzeNote(selectedNote.value.id)
  analyzing.value = false
  if (result) {
    analysisResult.value = result
    showAnalysisModal.value = true
  } else {
    QMessage.error(notesError.value || 'AI 分析失败，请稍后重试')
  }
}

async function handleAiFormat() {
  if (!selectedNote.value) return
  formatting.value = true
  const result = await formatNote(selectedNote.value.id)
  formatting.value = false
  if (result) {
    formatResult.value = result
    showFormatModal.value = true
  } else {
    QMessage.error(notesError.value || 'AI 格式化失败，请稍后重试')
  }
}

async function handleFormatConfirm() {
  if (!selectedNote.value || !formatResult.value) return
  // 后端对超长笔记只格式化了前一部分（noteFormatMaxRunes=10000 截断），
  // 此时整篇覆盖会把剩余内容永久删除——拒绝保存并提示分段格式化
  if (formatResult.value.truncated) {
    QMessage.warning('笔记超过 10000 字，AI 只格式化了前一部分。为避免丢失剩余内容，请分段格式化后再替换')
    return
  }
  const ok = await updateNote(selectedNote.value.id, {
    title: selectedNote.value.title,
    content: formatResult.value.content,
    tags: selectedNote.value.tags
  })
  if (ok) {
    // v-model:content 双向绑定，改 content 即同步编辑器
    selectedNote.value.content = formatResult.value.content
    showFormatModal.value = false
    QMessage.success('已替换为格式化内容')
  } else {
    QMessage.error(notesError.value || '保存失败')
  }
}

async function handleAnalysisConfirm(summary: string, tags: string[]) {
  if (!selectedNote.value) return
  await updateNoteSummary(selectedNote.value.id, summary)
  await updateNoteTags(selectedNote.value.id, tags)
  selectedNote.value.summary = summary
  selectedNote.value.tags = tags
  showAnalysisModal.value = false
  QMessage.success('已保存摘要和标签')
}

function handleExport() {
  if (!selectedNote.value) return
  exportNote(selectedNote.value.id, selectedNote.value.title)
}

function handleShare() {
  if (!selectedNote.value) return
  window.dispatchEvent(new CustomEvent('openShareModal', {
    detail: { type: 'note', data: selectedNote.value }
  }))
}

// 切换「允许分身读取」：打开→向量化进分身知识库；关闭→移除向量。成功后同步本地状态。
async function toggleAiAccess(id: number, accessible: boolean) {
  const updated = await setNoteAiAccessible(id, accessible)
  if (!updated) {
    QMessage.error(notesError.value || '设置失败')
    return
  }
  const idx = notes.value.findIndex(n => n.id === id)
  if (idx !== -1) notes.value[idx] = updated
  if (selectedNoteId.value === id) {
    selectedNote.value = { ...updated, tags: updated.tags }
  }
}

async function handleDelete(id: number) {
  const result = await QMessageBox.confirm('确定要删除这个笔记吗？', '删除笔记', { confirmButtonText: '删除', type: 'warning' })
  if (result.action !== 'confirm') return
  const ok = await deleteNote(id)
  if (ok) {
    // 若删除的是当前笔记，丢弃其待保存内容
    if (selectedNoteId.value === id) {
      autoSave.cancel()
    }
    draft.clear(id)
    notes.value = notes.value.filter(n => n.id !== id)
    if (selectedNoteId.value === id) {
      selectedNoteId.value = null
      selectedNote.value = null
    }
    QMessage.success('删除成功')
  }
}

function handleVisibilityChange() {
  if (document.visibilityState === 'hidden') void autoSave.flush()
}

onMounted(async () => {
  notes.value = await fetchNotes()

  document.addEventListener('keydown', handleKeydown)
  document.addEventListener('visibilitychange', handleVisibilityChange)
  window.addEventListener('beforeunload', handleVisibilityChange)
})

// 监听标题/正文变化触发自动保存；tags/summary 走独立接口，不纳入防抖源
watch(
  () => selectedNote.value ? [selectedNote.value.title, selectedNote.value.content] : null,
  (val) => {
    if (skipNextAutoSave) {
      skipNextAutoSave = false
      return
    }
    if (!val || !selectedNote.value) return
    // localStorage 即时落盘：每次击键 <1ms，数据不丢失
    draft.save(selectedNote.value.id, {
      title: val[0],
      content: val[1],
      tags: selectedNote.value.tags || []
    })
    draftSaved.value = true
    // 服务器同步走防抖，后台静默完成
    autoSavedNoteId = selectedNote.value.id
    autoSave.schedule(selectedNote.value.id, {
      title: val[0],
      content: val[1],
      tags: selectedNote.value.tags || []
    })
  }
)

// 服务器同步成功 → 清理 localStorage 草稿 + 清除草稿标志
watch(
  () => autoSave.status.value,
  (newStatus) => {
    if (newStatus === 'saved' && autoSavedNoteId !== null) {
      draft.clear(autoSavedNoteId)
      autoSavedNoteId = null
      draftSaved.value = false
    }
  }
)

/** 从 localStorage 恢复未同步的草稿 */
function restoreDraft() {
  if (!selectedNote.value) return
  const cached = draft.load(selectedNote.value.id)
  if (!cached) return
  // 服务器内容不早于本地草稿（草稿已同步 / 已过期）：以服务器为准，丢弃草稿。
  // 旧格式草稿无 savedAt，保守保留走内容比较；updated_at 缺失时视为无服务器版本，恢复草稿
  const serverTime = selectedNote.value.updated_at ? new Date(selectedNote.value.updated_at).getTime() : 0
  if (cached.savedAt !== undefined && cached.savedAt <= serverTime) {
    draft.clear(selectedNote.value.id)
    return
  }
  if (cached.content === selectedNote.value.content && cached.title === selectedNote.value.title) {
    draft.clear(selectedNote.value.id)
    return
  }
  skipNextAutoSave = true
  selectedNote.value.title = cached.title
  selectedNote.value.content = cached.content
  if (cached.tags?.length) selectedNote.value.tags = cached.tags
}

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  window.removeEventListener('beforeunload', handleVisibilityChange)
  // 尝试保存未提交的改动；fire-and-forget，组件卸载不等候
  void autoSave.flush()
})

function handleKeydown(e: KeyboardEvent) {
  // 快捷键帮助弹窗打开时不拦截
  if (showShortcutsHelp.value) {
    if (e.key === 'Escape') showShortcutsHelp.value = false
    return
  }
  if (e.key === 'Escape' && isFullscreen.value) {
    isFullscreen.value = false
    return
  }
  if (e.key === 'F11') {
    e.preventDefault()
    toggleFullscreen()
    return
  }
  if ((e.ctrlKey || e.metaKey) && e.key === 's') {
    e.preventDefault()
    handleSave()
    return
  }
  if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
    e.preventDefault()
    handleCreate()
    return
  }
  // ? 键打开快捷键帮助（排除在输入框/编辑器内的场景）
  if (e.key === '?' && !e.ctrlKey && !e.metaKey && !e.altKey) {
    const el = e.target as HTMLElement
    if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.closest('.cm-editor') || el.isContentEditable) return
    e.preventDefault()
    showShortcutsHelp.value = true
  }
}
</script>

<style scoped>
.notes-app {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--content-bg);
  overflow: hidden;
  box-shadow: var(--shadow-lg);
}

.notes-app.fullscreen {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1000;
  border-radius: 0;
  margin: 0;
}

.notes-app.fullscreen .note-main {
  padding: var(--spacing-4);
}

.notes-content {
  flex: 1;
  display: flex;
  overflow: hidden;
  background: var(--content-bg);
}

.notes-sidebar {
  width: 240px;
  min-width: 200px;
  background: var(--card-bg);
  /* border-right: 1px solid var(--border-color); */
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.notes-search-box {
  position: relative;
  padding: var(--spacing-3);
  /* border-bottom: 1px solid var(--border-color); */
  background: var(--card-bg);
}

.notes-search-input {
  width: 100%;
  padding: var(--spacing-2) var(--spacing-3);
  padding-right: 36px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  font-size: var(--font-size-xxs);
  color: var(--text-color);
  background: transparent;
  box-sizing: border-box;
  transition: all var(--transition-base);
}

.notes-search-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 4px rgba(51, 133, 255, 0.1);
}

.notes-search-input::placeholder {
  color: var(--text-secondary);
}

.notes-search-icon {
  position: absolute;
  right: 20px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-secondary);
  font-size: var(--font-size-xxs);
}

.notes-list {
  padding: var(--spacing-2);
  flex: 1;
  overflow-y: auto;
}

.empty-notes {
  padding: var(--spacing-10) var(--spacing-5);
  text-align: center;
  color: var(--text-secondary);
}

.empty-notes p {
  margin: 0;
  font-size: var(--font-size-sm);
}

.note-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: var(--spacing-3) var(--spacing-5);
  overflow: hidden;
  background: var(--content-bg);
}

.empty-note {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  background: var(--card-bg);
  border-radius: var(--radius-xl);
  border: 2px dashed var(--border-color);
  /* margin: var(--spacing-4); */
}

.empty-icon {
  font-size: 80px;
  margin-bottom: var(--spacing-5);
  background: linear-gradient(135deg, var(--primary-color), var(--color-primary-600));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  opacity: 0.8;
}

.empty-note p {
  font-size: var(--font-size-sm);
  margin: 0;
  font-weight: var(--font-weight-medium);
}

.import-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 14px;
  height: 32px;
  border: 1px solid var(--border-color);
  background: var(--card-bg);
  border-radius: 6px;
  cursor: pointer;
  color: var(--text-color);
  font-size: var(--font-size-xs);
  font-weight: 500;
  transition: all 0.2s ease;
}

.import-btn:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
  color: var(--primary-color);
}

@media (max-width: 768px) {
  .notes-content {
    flex-direction: column;
  }
  
  .notes-sidebar {
    width: 100%;
    max-height: 300px;
    border-right: none;
    border-bottom: 1px solid var(--border-color);
  }
  
  .note-main {
    padding: var(--spacing-4);
  }
}

/* 快捷键帮助弹窗（Teleport 到 body，需要 :global） */
:global(.shortcuts-overlay) {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

:global(.shortcuts-modal) {
  background: var(--card-bg, #fff);
  border-radius: 12px;
  width: 520px;
  max-height: 80vh;
  overflow-y: auto;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
}

:global(.shortcuts-header) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
}

:global(.shortcuts-header h3) {
  margin: 0;
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--text-color);
  display: flex;
  align-items: center;
  gap: 8px;
}

:global(.shortcuts-close) {
  width: 28px;
  height: 28px;
  border: none;
  background: none;
  border-radius: 6px;
  cursor: pointer;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
}

:global(.shortcuts-close:hover) {
  background: var(--hover-color);
}

:global(.shortcuts-body) {
  padding: 16px 20px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

:global(.shortcuts-group h4) {
  margin: 0 0 8px 0;
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--primary-color);
}

:global(.shortcut-item) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 0;
}

:global(.shortcut-desc) {
  font-size: var(--font-size-xs);
  color: var(--text-color);
}

:global(.shortcut-keys) {
  display: flex;
  align-items: center;
  gap: 2px;
  font-size: var(--font-size-xxs);
}

:global(.shortcut-keys kbd) {
  display: inline-block;
  padding: 1px 6px;
  font-size: var(--font-size-xxxs);
  font-family: inherit;
  color: var(--text-color);
  background: var(--content-bg, #f5f5f5);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  box-shadow: 0 1px 0 var(--border-color);
}
</style>

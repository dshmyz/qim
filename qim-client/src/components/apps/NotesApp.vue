<template>
  <div class="notes-app" :class="{ fullscreen: isFullscreen }">
    <AppHeader title="笔记" @back="$emit('back')" v-show="!isFullscreen">
      <template #extra-buttons>
        <ToggleSidebarBtn
          icon="fas fa-compress"
          title="收起侧边栏"
          @click="$emit('toggleSidebar')"
        />
      </template>
      <template #actions>
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
            :analyzing="analyzing"
            :fullscreen="isFullscreen"
            @format="handleFormat"
            @insert-link="handleInsertLink"
            @save="handleSave"
            @analyze="handleAnalyze"
            @import="triggerImport"
            @export="handleExport"
            @share="handleShare"
            @delete="handleDelete(selectedNote.id)"
            @toggle-fullscreen="toggleFullscreen"
          />
          <NoteEditor
            ref="noteEditorRef"
            v-model:title="selectedNote.title"
            v-model:content="selectedNote.content"
            :mode="editorMode"
            @save="handleSave"
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
      accept=".md,.markdown"
      style="display: none"
      @change="handleFileSelect"
    />
    
    <AIAnalysisModal
      :visible="showAnalysisModal"
      :result="analysisResult"
      @close="showAnalysisModal = false"
      @confirm="handleAnalysisConfirm"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import AppHeader from './AppHeader.vue'
import ToggleSidebarBtn from '../shared/ToggleSidebarBtn.vue'
import NoteCard from './notes/NoteCard.vue'
import NoteToolbar from './notes/NoteToolbar.vue'
import NoteEditor from './notes/NoteEditor.vue'
import NoteTagFilter from './notes/NoteTagFilter.vue'
import AIAnalysisModal from './notes/AIAnalysisModal.vue'
import { useNotes } from '../../composables/useNotes'
import QMessage from '../../utils/qmessage'
import type { Note, AIAnalyzeResult } from '../../types/note'

const emit = defineEmits(['back', 'toggleSidebar'])

const { 
  fetchNotes, 
  createNote, 
  updateNote, 
  deleteNote, 
  analyzeNote,
  updateNoteTags,
  updateNoteSummary,
  exportNote,
  error: notesError
} = useNotes()

const notes = ref<Note[]>([])
const selectedNoteId = ref<number | null>(null)
const selectedNote = ref<Note | null>(null)
const searchQuery = ref('')
const selectedTag = ref<string | null>(null)
const editorMode = ref<'edit' | 'split' | 'preview'>('edit')
const saving = ref(false)
const analyzing = ref(false)
const showAnalysisModal = ref(false)
const analysisResult = ref<AIAnalyzeResult | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)
const isFullscreen = ref(false)
const noteEditorRef = ref<InstanceType<typeof NoteEditor> | null>(null)

const allTags = computed(() => {
  const tags = new Set<string>()
  notes.value.forEach(n => n.tags?.forEach(t => tags.add(t)))
  return Array.from(tags)
})

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

function selectNote(id: number) {
  selectedNoteId.value = id
  selectedNote.value = notes.value.find(n => n.id === id) || null
}

function editNote(note: Note) {
  selectNote(note.id)
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
  
  const reader = new FileReader()
  reader.onload = async (e) => {
    const content = e.target?.result as string
    const fileName = file.name.replace(/\.(md|markdown)$/i, '')
    
    const note = await createNote({ title: fileName, content })
    if (note) {
      notes.value.unshift(note)
      selectNote(note.id)
      QMessage.success('导入成功')
    }
  }
  reader.readAsText(file)
  input.value = ''
}

async function handleSave() {
  if (!selectedNote.value) return
  saving.value = true
  const ok = await updateNote(selectedNote.value.id, {
    title: selectedNote.value.title,
    content: selectedNote.value.content,
    tags: selectedNote.value.tags
  })
  saving.value = false
  if (ok) {
    QMessage.success('保存成功')
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

async function handleDelete(id: number) {
  if (!confirm('确定要删除这个笔记吗？')) return
  const ok = await deleteNote(id)
  if (ok) {
    notes.value = notes.value.filter(n => n.id !== id)
    if (selectedNoteId.value === id) {
      selectedNoteId.value = null
      selectedNote.value = null
    }
    QMessage.success('删除成功')
  }
}

onMounted(async () => {
  notes.value = await fetchNotes()
  
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})

function handleKeydown(e: KeyboardEvent) {
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
  font-size: var(--font-size-xs);
  color: var(--text-color);
  background: var(--content-bg);
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
  font-size: var(--font-size-xs);
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
  font-size: var(--font-size-base);
  margin: 0;
  font-weight: var(--font-weight-medium);
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
</style>

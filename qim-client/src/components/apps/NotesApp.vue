<template>
  <div class="notes-app">
    <div class="notes-header">
      <div class="header-left">
        <button class="back-btn" @click="$emit('back')">
          <i class="fas fa-chevron-left"></i>
        </button>
        <button class="toggle-sidebar-btn" @click="$emit('toggleSidebar')">
          <i class="fas fa-compress"></i>
        </button>
        <div class="notes-header-info">
          <h2>笔记</h2>
        </div>
      </div>
      <button class="create-note-btn" @click="handleCreate">+ 新建笔记</button>
    </div>
    
    <div class="notes-content">
      <div class="notes-sidebar">
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
            @save="handleSave"
            @analyze="handleAnalyze"
            @import="triggerImport"
            @export="handleExport"
            @share="handleShare"
            @delete="handleDelete(selectedNote.id)"
          />
          <NoteEditor
            v-model:title="selectedNote.title"
            v-model:content="selectedNote.content"
            :mode="editorMode"
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
import { ref, computed, onMounted } from 'vue'
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
  exportNote 
} = useNotes()

const notes = ref<Note[]>([])
const selectedNoteId = ref<number | null>(null)
const selectedNote = ref<Note | null>(null)
const searchQuery = ref('')
const selectedTag = ref<string | null>(null)
const editorMode = ref<'edit' | 'preview'>('edit')
const saving = ref(false)
const analyzing = ref(false)
const showAnalysisModal = ref(false)
const analysisResult = ref<AIAnalyzeResult | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)

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
})
</script>

<style scoped>
.notes-app {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--content-bg);
  overflow: hidden;
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-lg);
}

.notes-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-4) var(--spacing-5);
  background: var(--card-bg);
  box-shadow: var(--shadow-sm);
  height: 72px;
  box-sizing: border-box;
  border-bottom: 1px solid var(--border-color);
}

.header-left {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
}

.back-btn,
.toggle-sidebar-btn {
  width: 36px;
  height: 36px;
  border: 1px solid var(--border-color);
  background: var(--btn-bg);
  border-radius: var(--radius-md);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  transition: all var(--transition-base);
}

.back-btn:hover,
.toggle-sidebar-btn:hover {
  background: var(--primary-light);
  border-color: var(--primary-color);
  color: var(--primary-color);
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

.notes-header-info h2 {
  margin: 0;
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-bold);
  color: var(--text-color);
  letter-spacing: -0.02em;
}

.create-note-btn {
  padding: var(--spacing-2) var(--spacing-4);
  background: linear-gradient(135deg, var(--primary-color), var(--color-primary-600));
  color: white;
  border: none;
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  cursor: pointer;
  transition: all var(--transition-base);
  box-shadow: 0 2px 8px rgba(51, 133, 255, 0.3);
}

.create-note-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(51, 133, 255, 0.4);
}

.create-note-btn:active {
  transform: translateY(0);
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
  border-right: 1px solid var(--border-color);
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.notes-search-box {
  position: relative;
  padding: var(--spacing-3);
  border-bottom: 1px solid var(--border-color);
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
  padding: var(--spacing-6);
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
  margin: var(--spacing-4);
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

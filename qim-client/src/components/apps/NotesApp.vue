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
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.notes-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background-color: var(--card-bg);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  height: 72px;
  box-sizing: border-box;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.back-btn,
.toggle-sidebar-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: var(--hover-color);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  color: var(--primary-color);
  transition: background 0.2s;
}

.back-btn:hover,
.toggle-sidebar-btn:hover {
  background: var(--primary-light);
}

.notes-header-info h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
}

.create-note-btn {
  padding: 8px 16px;
  background-color: var(--primary-color);
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
}

.create-note-btn:hover {
  background-color: var(--active-color);
}

.notes-content {
  flex: 1;
  display: flex;
  overflow: hidden;
  background: var(--content-bg);
}

.notes-sidebar {
  width: 280px;
  background: var(--card-bg);
  border-right: 1px solid var(--border-color);
  overflow-y: auto;
}

.notes-search-box {
  position: relative;
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
}

.notes-search-input {
  width: 100%;
  padding: 10px 40px 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: 14px;
  color: var(--text-primary);
  background: var(--bg-color);
  box-sizing: border-box;
}

.notes-search-input:focus {
  outline: none;
  border-color: var(--primary-color);
}

.notes-search-icon {
  position: absolute;
  right: 24px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-secondary);
  font-size: 14px;
}

.notes-list {
  padding: 12px;
}

.empty-notes {
  padding: 40px 20px;
  text-align: center;
  color: var(--text-secondary);
}

.empty-notes p {
  margin: 0;
  font-size: 14px;
}

.note-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 24px;
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
}

.empty-icon {
  font-size: 64px;
  margin-bottom: 20px;
  color: var(--primary-color);
  opacity: 0.5;
}

.empty-note p {
  font-size: 16px;
  margin: 0;
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
    padding: 16px;
  }
}
</style>

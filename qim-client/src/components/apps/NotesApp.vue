<template>
  <div class="notes-app">
    <div class="notes-header">
      <div class="header-left">
        <button class="back-btn" @click="$emit('back')">
          <i class="fas fa-arrow-left"></i>
        </button>
        <div class="notes-header-info">
          <h2>笔记</h2>
        </div>
      </div>
      <button class="create-note-btn" @click="createNote">+ 新建笔记</button>
    </div>
    <div class="notes-content">
      <div class="notes-sidebar">
        <!-- 笔记搜索框 -->
        <div class="notes-search-box">
          <input
            v-model="noteSearchQuery"
            type="text"
            class="notes-search-input"
            placeholder="搜索笔记..."
          />
          <i class="fas fa-search notes-search-icon"></i>
        </div>
        <div class="notes-list">
          <div
            v-for="note in filteredNotes"
            :key="note.id"
            class="note-item"
            :class="{ active: selectedNoteId === note.id }"
            @click="selectNote(note.id)"
          >
            <div class="note-title">{{ note.title }}</div>
            <div class="note-preview">{{ note.content.substring(0, 50) }}...</div>
            <div class="note-date">{{ formatNoteDate(note.timestamp) }}</div>
          </div>
          <div v-if="filteredNotes.length === 0" class="empty-notes">
            <p>没有找到匹配的笔记</p>
          </div>
        </div>
      </div>
      <div class="note-editor">
        <div v-if="selectedNote" class="editor-content">
          <div class="note-header-actions">
            <button class="save-btn" @click="saveNote(selectedNote)"><i class="fas fa-save"></i> 保存</button>
            <button class="share-btn" @click="openShareModal('note', selectedNote)"><i class="fas fa-share-alt"></i> 分享</button>
            <button class="delete-btn" @click="deleteNote(selectedNote.id.toString())"><i class="fas fa-trash"></i> 删除</button>
          </div>
          <input
            v-model="selectedNote.title"
            class="note-title-input"
            placeholder="笔记标题"
          />
          <div class="editor-layout">
            <div class="editor-left">
              <div class="editor-toolbar">
                <button class="toolbar-btn" @click="insertMarkdown('**', '**')"><strong>B</strong></button>
                <button class="toolbar-btn" @click="insertMarkdown('*', '*')"><em>I</em></button>
                <button class="toolbar-btn" @click="insertMarkdown('```', '```')">代码</button>
                <button class="toolbar-btn" @click="insertMarkdown('# ', '')">标题</button>
                <button class="toolbar-btn" @click="insertMarkdown('- ', '')">列表</button>
              </div>
              <textarea
                v-model="selectedNote.content"
                class="note-content-input"
                placeholder="使用 Markdown 编写笔记..."
              ></textarea>
            </div>
            <div class="editor-right">
              <div class="note-preview-panel">
                <h3>预览</h3>
                <div class="preview-content" v-html="renderMarkdown(selectedNote.content)"></div>
              </div>
            </div>
          </div>
        </div>
        <div v-else class="empty-note">
          <div class="empty-icon"><i class="fas fa-book"></i></div>
          <p>选择一个笔记或创建新笔记</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { API_BASE_URL } from '../../config'

// 服务器URL
const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

// 定义事件
const emit = defineEmits(['back'])

// 笔记相关状态
const notes = ref<any[]>([])
const noteSearchQuery = ref('')
const selectedNoteId = ref('')

// 计算属性
const selectedNote = computed(() => {
  return notes.value.find(note => note.id === selectedNoteId.value)
})

const filteredNotes = computed(() => {
  if (!noteSearchQuery.value) {
    return notes.value
  }
  return notes.value.filter(note => 
    note.title.toLowerCase().includes(noteSearchQuery.value.toLowerCase()) ||
    note.content.toLowerCase().includes(noteSearchQuery.value.toLowerCase())
  )
})

// 获取token
const getToken = () => {
  return localStorage.getItem('token')
}

// 加载笔记数据
const loadNotes = async () => {
  try {
    const token = getToken()
    const response = await axios.get(`${serverUrl.value}/api/v1/notes`, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    // 过滤出普通笔记数据
    notes.value = response.data.data.filter((note: any) => note.type === 'note' || !note.type)
    // 如果有笔记，默认选择第一个
    if (notes.value.length > 0) {
      selectedNoteId.value = notes.value[0].id.toString()
    }
  } catch (error) {
    console.error('加载笔记失败:', error)
    ElMessage.error('加载笔记失败，请稍后重试')
  }
}

// 创建笔记
const createNote = async () => {
  try {
    const token = getToken()
    const response = await axios.post(`${serverUrl.value}/api/v1/notes`, {
      title: '新笔记',
      content: '',
      type: 'note'
    }, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    const newNote = response.data.data
    notes.value.unshift(newNote)
    selectedNoteId.value = newNote.id ? newNote.id.toString() : ''
  } catch (error) {
    console.error('创建笔记失败:', error)
    ElMessage.error('创建笔记失败，请稍后重试')
  }
}

// 选择笔记
const selectNote = (noteId: string) => {
  selectedNoteId.value = noteId
}

// 保存笔记
const saveNote = async (note: any) => {
  try {
    const token = getToken()
    const response = await axios.put(`${serverUrl.value}/api/v1/notes/${note.id}`, {
      title: note.title,
      content: note.content
    }, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    console.log('保存笔记成功:', response.data)
  } catch (error) {
    console.error('保存笔记失败:', error)
  }
}

// 删除笔记
const deleteNote = async (noteId: string) => {
  try {
    const token = getToken()
    await axios.delete(`${serverUrl.value}/api/v1/notes/${noteId}`, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    notes.value = notes.value.filter(note => note.id.toString() !== noteId)
    if (selectedNoteId.value === noteId) {
      selectedNoteId.value = notes.value.length > 0 ? notes.value[0].id.toString() : ''
    }
  } catch (error) {
    console.error('删除笔记失败:', error)
    // 直接更新本地数据
    notes.value = notes.value.filter(note => note.id.toString() !== noteId)
    if (selectedNoteId.value === noteId) {
      selectedNoteId.value = notes.value.length > 0 ? notes.value[0].id.toString() : ''
    }
  }
}

// 插入 Markdown 格式
const insertMarkdown = (prefix: string, suffix: string) => {
  if (selectedNote.value) {
    selectedNote.value.content += prefix + suffix
  }
}

// 渲染 Markdown
const renderMarkdown = (content: string): string => {
  // 简单的 Markdown 渲染
  let html = content
  
  // 标题
  html = html.replace(/^# (.*$)/gm, '<h1>$1</h1>')
  html = html.replace(/^## (.*$)/gm, '<h2>$1</h2>')
  html = html.replace(/^### (.*$)/gm, '<h3>$1</h3>')
  
  // 粗体
  html = html.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
  
  // 斜体
  html = html.replace(/\*(.*?)\*/g, '<em>$1</em>')
  
  // 代码块
  html = html.replace(/```([\s\S]*?)```/g, '<pre><code>$1</code></pre>')
  
  // 行内代码
  html = html.replace(/`(.*?)`/g, '<code>$1</code>')
  
  // 列表
  html = html.replace(/^- (.*$)/gm, '<li>$1</li>')
  html = html.replace(/(<li>.*<\/li>)/s, '<ul>$1</ul>')
  
  // 链接
  html = html.replace(/\[(.*?)\]\((.*?)\)/g, '<a href="$2" target="_blank">$1</a>')
  
  // 换行
  html = html.replace(/\n/g, '<br>')
  
  return html
}

// 格式化日期
const formatNoteDate = (timestamp: number): string => {
  const date = new Date(timestamp)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 打开分享模态框
const openShareModal = (type: string, data: any) => {
  window.dispatchEvent(new CustomEvent('openShareModal', {
    detail: { type, data }
  }))
}

// 组件挂载时加载笔记数据
onMounted(async () => {
  await loadNotes()
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
  border-bottom: 1px solid var(--border-color);
  background-color: var(--card-bg);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
  height: 72px;
  box-sizing: border-box;
}

.notes-header:hover {
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.back-btn {
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
  transition: background 0.2s;
  color: var(--primary-color);
}

.back-btn:hover {
  background: var(--primary-light);
}

.notes-header-info h2 {
  margin: 0 0 4px 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  transition: color 0.3s ease;
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
  display: flex;
  align-items: center;
  gap: 6px;
}

.create-note-btn:hover {
  background-color: var(--active-color);
  color: white;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
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
  transition: all 0.3s ease;
}

.notes-search-box {
  position: relative;
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
  background: var(--card-bg);
}

.notes-search-input {
  width: 100%;
  padding: 10px 40px 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: 14px;
  color: var(--text-primary);
  background: var(--bg-color);
  transition: all 0.3s ease;
  box-sizing: border-box;
}

.notes-search-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.notes-search-icon {
  position: absolute;
  right: 24px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-secondary);
  font-size: 14px;
  transition: color 0.3s ease;
}

.notes-list {
  padding: 12px;
}

.note-item {
  padding: 16px;
  margin-bottom: 8px;
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
}

.note-item:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.note-item.active {
  background: var(--hover-color);
  border-color: var(--primary-color);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.15);
}

.note-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 6px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  transition: color 0.3s ease;
}

.note-preview {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 8px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  transition: color 0.3s ease;
}

.note-date {
  font-size: 12px;
  color: var(--text-tertiary);
  transition: color 0.3s ease;
}

.note-editor {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 24px;
  overflow-y: auto;
  background: var(--content-bg);
}

.empty-note {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  opacity: 0.7;
  transition: all 0.3s ease;
}

.empty-note:hover {
  opacity: 1;
}

.empty-icon {
  font-size: 64px;
  margin-bottom: 20px;
  color: var(--primary-color);
  opacity: 0.5;
  transition: all 0.3s ease;
}

.empty-note:hover .empty-icon {
  opacity: 0.8;
  transform: scale(1.1);
}

.empty-note p {
  font-size: 16px;
  margin: 0;
  transition: color 0.3s ease;
}

.editor-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 20px;
  background: var(--card-bg);
  border-radius: 8px;
  padding: 24px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.editor-content:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.note-header-actions {
  display: flex;
  gap: 12px;
  margin-bottom: 8px;
  padding: 0;
}

.save-btn, .delete-btn, .share-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.3s ease;
}

.save-btn {
  background-color: #4CAF50;
  color: white;
}

.save-btn:hover {
  background-color: #45a049;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(76, 175, 80, 0.3);
}

.share-btn {
  background-color: var(--primary-color);
  color: white;
}

.share-btn:hover {
  background-color: var(--primary-hover);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

.delete-btn {
  background-color: #f44336;
  color: white;
}

.delete-btn:hover {
  background-color: #da190b;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(244, 67, 54, 0.3);
}

.note-title-input {
  padding: 12px 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
  outline: none;
  transition: all 0.3s ease;
  background: var(--bg-color);
  box-sizing: border-box;
}

.note-title-input:focus {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  transform: translateY(-1px);
}

.editor-layout {
  flex: 1;
  display: flex;
  gap: 16px;
  overflow: hidden;
}

.editor-left {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow: hidden;
}

.editor-right {
  width: 35%;
  min-width: 250px;
  overflow-y: auto;
}

.editor-toolbar {
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  transition: all 0.3s ease;
}

.editor-toolbar:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.toolbar-btn {
  padding: 8px 16px;
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.3s ease;
  color: var(--text-primary);
  font-weight: 500;
}

.toolbar-btn:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
  color: var(--primary-color);
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.note-content-input {
  flex: 1;
  padding: 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: 14px;
  color: var(--text-primary);
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  line-height: 1.6;
  resize: none;
  outline: none;
  transition: all 0.3s ease;
  background: var(--bg-color);
  min-height: 400px;
  box-sizing: border-box;
  overflow-y: auto;
}

.note-content-input:focus {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  transform: translateY(-1px);
}

.note-preview-panel {
  padding: 20px;
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  transition: all 0.3s ease;
  min-height: 400px;
}

.note-preview-panel:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.note-preview-panel h3 {
  margin-top: 0;
  margin-bottom: 16px;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  transition: color 0.3s ease;
}

.preview-content {
  font-size: 14px;
  color: var(--text-primary);
  line-height: 1.6;
  transition: color 0.3s ease;
  min-height: 300px;
}

.preview-content h1,
.preview-content h2,
.preview-content h3,
.preview-content h4,
.preview-content h5,
.preview-content h6 {
  margin-top: 20px;
  margin-bottom: 12px;
  font-weight: 600;
  color: var(--text-primary);
  transition: color 0.3s ease;
}

.preview-content h1 {
  font-size: 24px;
  border-bottom: 1px solid var(--border-color);
  padding-bottom: 8px;
}

.preview-content h2 {
  font-size: 20px;
  border-bottom: 1px solid var(--border-color);
  padding-bottom: 6px;
}

.preview-content h3 {
  font-size: 18px;
}

.preview-content ul,
.preview-content ol {
  margin-left: 24px;
  margin-bottom: 16px;
  color: var(--text-primary);
}

.preview-content li {
  margin-bottom: 6px;
  transition: color 0.3s ease;
}

.preview-content pre {
  background: var(--list-bg);
  padding: 16px;
  border-radius: 8px;
  overflow-x: auto;
  margin-bottom: 16px;
  border: 1px solid var(--border-color);
  transition: all 0.3s ease;
}

.preview-content pre:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.preview-content code {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  color: var(--text-primary);
  background: var(--list-bg);
  padding: 2px 6px;
  border-radius: 4px;
  transition: all 0.3s ease;
}

.preview-content a {
  color: var(--primary-color);
  text-decoration: none;
  transition: all 0.3s ease;
  font-weight: 500;
}

.preview-content a:hover {
  text-decoration: underline;
  color: var(--primary-hover);
}

.preview-content img {
  max-width: 100%;
  border-radius: 8px;
  margin: 20px 0;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  transition: all 0.3s ease;
}

.preview-content img:hover {
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
  transform: scale(1.01);
}

.empty-notes {
  padding: 40px 20px;
  text-align: center;
  color: var(--text-secondary);
  opacity: 0.7;
  transition: all 0.3s ease;
}

.empty-notes:hover {
  opacity: 1;
}

.empty-notes p {
  margin: 0;
  font-size: 14px;
  transition: color 0.3s ease;
}

/* 响应式设计 */
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
  
  .note-editor {
    padding: 16px;
  }
  
  .editor-content {
    padding: 16px;
  }
  
  .editor-layout {
    flex-direction: column;
  }
  
  .editor-right {
    width: 100%;
    min-width: unset;
  }
  
  .note-title-input {
    font-size: 18px;
    padding: 10px 12px;
  }
  
  .note-content-input {
    min-height: 300px;
    padding: 12px;
  }
  
  .note-preview-panel {
    padding: 16px;
    min-height: 300px;
  }
  
  .note-header-actions {
    flex-wrap: wrap;
  }
  
  .save-btn, .delete-btn, .share-btn {
    padding: 6px 12px;
    font-size: 13px;
  }
  
  .toolbar-btn {
    padding: 6px 12px;
    font-size: 13px;
  }
}

/* 动画效果 */
@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.note-item {
  animation: fadeIn 0.3s ease;
}

.editor-content {
  animation: fadeIn 0.3s ease;
}
</style>
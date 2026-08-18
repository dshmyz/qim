<template>
  <div class="sticky-notes-app">
    <AppHeader title="便签" @back="$emit('back')">
      <template #actions>
        <div class="header-right">
          <div class="">
            <input 
              type="text" 
              placeholder="搜索便签..." 
              v-model="searchQuery"
              class="search-input"
            >
            <i class="fas fa-search search-icon"></i>
          </div>
          <button class="create-note-btn" @click="showCreateNoteModal">+ 新建便签</button>
        </div>
      </template>
    </AppHeader>
    <div class="sticky-notes-content">
      <StickyTagFilter
        :all-tags="allTags"
        :selected-tag="selectedTag"
        @select="selectedTag = $event"
        @clear="selectedTag = null"
      />
      <div class="sticky-notes-grid">
        <StickyNoteCard
          v-for="(note, index) in filteredStickyNotes"
          :key="note.id"
          :note="note"
          :index="index"
          @click="showEditNoteModal"
          @share="shareNote"
          @delete="deleteStickyNote"
          @filter-tag="selectedTag = $event"
          @dragstart="onDragStart"
          @drop="onDrop"
        />
        <div v-if="stickyNotes.length === 0" class="empty-notes">
          <div class="empty-icon"><i class="fas fa-sticky-note"></i></div>
          <p>暂无便签</p>
          <p class="empty-hint">点击右上角按钮创建新便签</p>
        </div>
      </div>
    </div>

    <!-- 便签模态框 -->
    <ModalContainer
      :visible="showNoteModal"
      :title="selectedNote ? '编辑便签' : '创建便签'"
      width="640px"
      @close="closeNoteModal"
    >
      <div class="sticky-note-preview-wrap">
        <StickyNoteCard
          :note="previewNote"
          :index="0"
          class="sticky-note-preview-card"
        />
      </div>
      <div class="sticky-note-form-group">
        <label for="sticky-note-title">标题</label>
        <input
          id="sticky-note-title"
          ref="titleInputRef"
          type="text"
          class="sticky-note-form-input"
          v-model="formData.title"
          placeholder="便签标题"
        >
      </div>
      <div class="sticky-note-form-group">
        <div class="sticky-note-content-header">
          <label for="sticky-note-content">内容</label>
          <span class="sticky-note-form-counter">{{ formData.content.length }} 字</span>
        </div>
        <textarea
          id="sticky-note-content"
          ref="contentInputRef"
          class="sticky-note-form-textarea"
          v-model="formData.content"
          placeholder="便签内容"
          @input="autoResizeContent"
        ></textarea>
      </div>
      <div class="sticky-note-form-group">
        <label>颜色</label>
        <div class="sticky-note-color-picker">
          <div
            v-for="color in noteColors"
            :key="color.value"
            class="sticky-note-color-option"
            :class="{ active: formData.color === color.value }"
            :style="{ background: colorPreviewMap[color.value] }"
            :title="color.name"
            @click="formData.color = color.value"
          ></div>
        </div>
      </div>
      <div class="sticky-note-form-grid">
        <div class="sticky-note-form-group">
          <label>纸张样式</label>
          <select class="sticky-note-form-select" v-model="formData.paperStyle">
            <option value="plain">普通纸张</option>
            <option value="lined">横线纸张</option>
            <option value="grid">网格纸张</option>
            <option value="dotted">点阵纸张</option>
          </select>
        </div>
        <div class="sticky-note-form-group">
          <label>字体</label>
          <select class="sticky-note-form-select" v-model="formData.fontFamily">
            <option value="Arial, 'Microsoft YaHei', sans-serif">Arial</option>
            <option value="'KaiTi', 'STKaiti', serif">楷体</option>
            <option value="'SimSun', 'STSong', serif">宋体</option>
            <option value="'SimHei', 'STHeiti', sans-serif">黑体</option>
            <option value="'FangSong', 'STFangsong', serif">仿宋</option>
            <option value="'Courier New', monospace">Courier New</option>
          </select>
        </div>
      </div>
      <div class="sticky-note-form-group">
        <label>标签</label>
        <div class="sticky-note-tag-input-wrap" @click="tagInputEl?.focus()">
          <span v-for="(tag, index) in selectedTags" :key="tag" class="sticky-note-tag-chip">
            {{ tag }}
            <button type="button" class="remove-btn" title="删除标签" @click.stop="removeTag(index)">×</button>
          </span>
          <input
            ref="tagInputEl"
            class="sticky-note-tag-input"
            v-model="tagInputText"
            placeholder="输入标签后回车"
            @keydown="handleTagInputKeydown"
            @blur="addTagFromInput"
          >
        </div>
        <p class="sticky-note-form-hint">回车或逗号生成标签，点击 × 删除</p>
      </div>

      <template #footer>
        <div class="modal-footer-left">
          <span class="sticky-note-shortcut-hint">Ctrl+Enter 保存 · Esc 关闭</span>
        </div>
        <div class="modal-footer-right">
          <button class="sticky-note-modal-btn sticky-note-cancel-btn" @click="closeNoteModal">取消</button>
          <button class="sticky-note-modal-btn sticky-note-confirm-btn" @click="handleSave">{{ selectedNote ? '更新' : '创建' }}</button>
        </div>
      </template>
    </ModalContainer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import axios from 'axios'
import QMessage from '../../utils/qmessage'
import { useServerUrl } from '../../composables/useServerUrl'
import { logger } from '../../utils/logger';
import AppHeader from './AppHeader.vue'
import ModalContainer from '../../components/shared/ModalContainer.vue'
import StickyTagFilter from './sticky/StickyTagFilter.vue'
import StickyNoteCard from './sticky/StickyNoteCard.vue'

// 服务器URL
const { serverUrl } = useServerUrl()

// 获取token
const getToken = () => {
  return localStorage.getItem('token')
}

// 便签应用相关状态
const stickyNotes = ref<any[]>([])
const showNoteModal = ref(false)
const selectedTag = ref<string | null>(null)
const newNote = ref({
  title: '',
  content: '',
  color: 'yellow',
  tags: '',
  paperStyle: 'plain',
  fontFamily: "Arial, 'Microsoft YaHei', sans-serif"
})
const selectedNote = ref<any>(null)
const formData = ref({
  title: '',
  content: '',
  color: 'yellow',
  paperStyle: 'plain',
  fontFamily: "Arial, 'Microsoft YaHei', sans-serif"
})
// 已生成的标签 chips 与标签输入框临时文本
const selectedTags = ref<string[]>([])
const tagInputText = ref('')
const tagInputEl = ref<HTMLInputElement | null>(null)
const draggedNoteId = ref<string | null>(null)
const searchQuery = ref('')
const searchTimeout = ref<number | null>(null)
const titleInputRef = ref<HTMLInputElement | null>(null)
const contentInputRef = ref<HTMLTextAreaElement | null>(null)

// 解析样式 JSON
const parseStyle = (styleStr: string | undefined) => {
  if (!styleStr || styleStr === '{}') {
    return { color: 'yellow', paperStyle: 'plain', fontFamily: "Arial, 'Microsoft YaHei', sans-serif" }
  }
  try {
    const style = JSON.parse(styleStr)
    return {
      color: style.color || 'yellow',
      paperStyle: style.paperStyle || 'plain',
      fontFamily: style.fontFamily || "Arial, 'Microsoft YaHei', sans-serif"
    }
  } catch {
    return { color: 'yellow', paperStyle: 'plain', fontFamily: "Arial, 'Microsoft YaHei', sans-serif" }
  }
}

const serializeStyle = () => {
  return JSON.stringify({
    color: formData.value.color,
    paperStyle: formData.value.paperStyle,
    fontFamily: formData.value.fontFamily
  })
}

// 实时预览便签：随表单输入即时渲染，所见即所得
const previewNote = computed(() => ({
  id: 'preview',
  title: formData.value.title.trim() || '便签标题',
  content: formData.value.content.trim() || '便签内容会显示在这里',
  style: serializeStyle(),
  tags: getFinalTags(),
  created_at: new Date().toISOString()
}))

const noteColors = [
  { name: '黄色', value: 'yellow' },
  { name: '蓝色', value: 'blue' },
  { name: '绿色', value: 'green' },
  { name: '红色', value: 'red' },
  { name: '紫色', value: 'purple' },
  { name: '粉色', value: 'pink' }
]

const colorPreviewMap: Record<string, string> = {
  yellow: 'linear-gradient(145deg, #fff9c4, #fff59d)',
  blue: 'linear-gradient(145deg, #e1f5fe, #b3e5fc)',
  green: 'linear-gradient(145deg, #e8f5e9, #c8e6c9)',
  red: 'linear-gradient(145deg, #fce4ec, #f8bbd0)',
  purple: 'linear-gradient(145deg, #f3e5f5, #e1bee7)',
  pink: 'linear-gradient(145deg, #fce4ec, #f8bbd0)'
}

// 加载便签数据
const loadStickyNotes = async () => {
  try {
    const token = getToken()
    const response = await axios.get(`${serverUrl.value}/api/v1/notes`, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    // 过滤出便签数据
    stickyNotes.value = response.data.data.filter((note: any) => note.type === 'sticky')
  } catch (error) {
    console.error('加载便签失败:', error)
    QMessage.error('加载便签失败，请稍后重试')
  }
}

// 创建便签
const createStickyNote = async () => {
  try {
    const token = getToken()

    const response = await axios.post(`${serverUrl.value}/api/v1/notes`, {
      title: formData.value.title,
      content: formData.value.content,
      color: formData.value.color,
      tags: JSON.stringify(getFinalTags()),
      type: 'sticky',
      style: serializeStyle()
    }, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    stickyNotes.value.push(response.data.data)
    closeNoteModal()
  } catch (error) {
    console.error('创建便签失败:', error)
    QMessage.error('创建便签失败，请稍后重试')
  }
}

// 更新便签
const updateStickyNote = async () => {
  try {
    const token = getToken()

    const response = await axios.put(`${serverUrl.value}/api/v1/notes/${selectedNote.value.id}`, {
      title: formData.value.title,
      content: formData.value.content,
      color: formData.value.color,
      tags: JSON.stringify(getFinalTags()),
      type: 'sticky',
      style: serializeStyle()
    }, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    const index = stickyNotes.value.findIndex(note => note.id === selectedNote.value.id)
    if (index !== -1) {
      stickyNotes.value[index] = response.data.data
    }
    closeNoteModal()
  } catch (error) {
    console.error('更新便签失败:', error)
    QMessage.error('更新便签失败，请稍后重试')
  }
}

// 删除便签
const deleteStickyNote = async (noteId: string) => {
  try {
    // 找到要删除的便签元素
    const noteElement = document.querySelector(`.sticky-note[data-note-id="${noteId}"]`)
    if (noteElement) {
      // 添加删除动画类
      noteElement.classList.add('deleting')
      // 等待动画完成后再删除
      setTimeout(async () => {
        const token = getToken()
        await axios.delete(`${serverUrl.value}/api/v1/notes/${noteId}`, {
          headers: {
            'Content-Type': 'application/json',
            ...(token ? { 'Authorization': `Bearer ${token}` } : {})
          }
        })
        stickyNotes.value = stickyNotes.value.filter(note => note.id !== noteId)
      }, 300)
    } else {
      // 如果找不到元素，直接删除
      const token = getToken()
      await axios.delete(`${serverUrl.value}/api/v1/notes/${noteId}`, {
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { 'Authorization': `Bearer ${token}` } : {})
        }
      })
      stickyNotes.value = stickyNotes.value.filter(note => note.id !== noteId)
    }
  } catch (error) {
    console.error('删除便签失败:', error)
    // 直接更新本地数据
    const noteElement = document.querySelector(`.sticky-note[data-note-id="${noteId}"]`)
    if (noteElement) {
      noteElement.classList.add('deleting')
      setTimeout(() => {
        stickyNotes.value = stickyNotes.value.filter(note => note.id !== noteId)
      }, 300)
    } else {
      stickyNotes.value = stickyNotes.value.filter(note => note.id !== noteId)
    }
  }
}

// 显示创建便签模态框
const showCreateNoteModal = () => {
  formData.value = {
    title: '',
    content: '',
    color: 'yellow',
    paperStyle: 'plain',
    fontFamily: "Arial, 'Microsoft YaHei', sans-serif"
  }
  selectedTags.value = []
  tagInputText.value = ''
  selectedNote.value = null
  showNoteModal.value = true
}

// 将后端存储的标签（JSON 数组字符串或数组）转为标签数组
const tagsToArray = (tags: any): string[] => {
  if (Array.isArray(tags)) return tags.filter(Boolean)
  if (typeof tags === 'string' && tags) {
    try {
      const parsed = JSON.parse(tags)
      return Array.isArray(parsed) ? parsed.filter(Boolean) : []
    } catch {
      // 兼容历史逗号分隔数据
      return tags.split(/[,，、]/).map(t => t.trim()).filter(Boolean)
    }
  }
  return []
}

// 把输入框文本生成一个标签 chip（去空、去重）
const addTagFromInput = () => {
  const text = tagInputText.value.trim()
  if (!text) return
  if (!selectedTags.value.includes(text)) {
    selectedTags.value.push(text)
  }
  tagInputText.value = ''
}

const removeTag = (index: number) => {
  selectedTags.value.splice(index, 1)
}

// 输入处理：回车/逗号/顿号生成标签；空输入框退格删除最后一个标签
const handleTagInputKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Enter' || e.key === ',' || e.key === '，' || e.key === '、') {
    e.preventDefault()
    addTagFromInput()
  } else if (e.key === 'Backspace' && !tagInputText.value && selectedTags.value.length > 0) {
    removeTag(selectedTags.value.length - 1)
  }
}

// 最终标签：已生成的 chips + 输入框残留文本（按逗号/顿号拆分），保存时兜底不丢
const getFinalTags = (): string[] => {
  const tags = [...selectedTags.value]
  if (tagInputText.value.trim()) {
    tagInputText.value.split(/[,，、]/).map(t => t.trim()).filter(Boolean).forEach(t => {
      if (!tags.includes(t)) tags.push(t)
    })
  }
  return tags
}

// 显示编辑便签模态框
const showEditNoteModal = (note: any) => {
  selectedNote.value = { ...note }
  const parsedStyle = parseStyle(note.style)
  formData.value = {
    title: note.title,
    content: note.content,
    color: parsedStyle.color,
    paperStyle: parsedStyle.paperStyle,
    fontFamily: parsedStyle.fontFamily
  }
  selectedTags.value = tagsToArray(note.tags)
  tagInputText.value = ''
  showNoteModal.value = true
}

// 关闭便签模态框
const closeNoteModal = () => {
  showNoteModal.value = false
  selectedNote.value = null
  selectedTags.value = []
  tagInputText.value = ''
  formData.value = {
    title: '',
    content: '',
    color: 'yellow',
    paperStyle: 'plain',
    fontFamily: "Arial, 'Microsoft YaHei', sans-serif"
  }
}

// 弹窗打开时自动聚焦标题，并恢复内容框高度
watch(showNoteModal, async (visible) => {
  if (!visible) return
  await nextTick()
  titleInputRef.value?.focus()
  autoResizeContent()
})

// 内容输入框随内容自动增高
const autoResizeContent = () => {
  const el = contentInputRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = `${Math.min(el.scrollHeight, 320)}px`
}

// 表单校验：标题/内容至少填一项
const validateForm = (): boolean => {
  if (!formData.value.title.trim() && !formData.value.content.trim()) {
    QMessage.warning('请至少填写标题或内容')
    titleInputRef.value?.focus()
    return false
  }
  return true
}

// 保存入口：先校验再创建/更新
const handleSave = () => {
  if (!validateForm()) return
  if (selectedNote.value) {
    updateStickyNote()
  } else {
    createStickyNote()
  }
}

// 格式化日期
const formatDate = (dateString: string) => {
  const date = new Date(dateString)
  if (isNaN(date.getTime())) {
    return ''
  }
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 分享便签
const shareNote = (note: any) => {
  // 触发分享事件，通知父组件打开分享弹窗
  window.dispatchEvent(new CustomEvent('shareStickyNote', {
    detail: note
  }))
}

// 拖拽排序相关函数
const onDragStart = (event: DragEvent, noteId: string) => {
  draggedNoteId.value = noteId
  if (event.target) {
    (event.target as HTMLElement).classList.add('dragging')
  }
}

const onDrop = (event: DragEvent, targetIndex: number) => {
  event.preventDefault()
  if (draggedNoteId.value) {
    const draggedIndex = stickyNotes.value.findIndex(note => note.id === draggedNoteId.value)
    if (draggedIndex !== -1 && draggedIndex !== targetIndex) {
      const [draggedNote] = stickyNotes.value.splice(draggedIndex, 1)
      stickyNotes.value.splice(targetIndex, 0, draggedNote)
    }
  }
  document.querySelectorAll('.sticky-note.dragging').forEach(el => {
    el.classList.remove('dragging')
  })
  draggedNoteId.value = null
}

// 转发笔记到聊天窗口
const forwardNoteToChat = (note: any) => {
  // 构建转发消息内容
  const messageContent = `【笔记转发】\n标题：${note.title}\n内容：${note.content}`
  
  // 触发全局事件，通知聊天窗口接收转发内容
  window.dispatchEvent(new CustomEvent('forwardNoteToChat', {
    detail: { content: messageContent }
  }))
  
  logger.log('转发笔记到聊天窗口:', note)
}


// 处理键盘快捷键
const handleKeydown = (event: KeyboardEvent) => {
  // 新建便签: Ctrl/Cmd + N（弹窗打开时不响应，避免误触重置正在编辑的内容）
  if ((event.ctrlKey || event.metaKey) && event.key === 'n' && !showNoteModal.value) {
    event.preventDefault()
    showCreateNoteModal()
  }
  // 聚焦搜索框: Ctrl/Cmd + F
  if ((event.ctrlKey || event.metaKey) && event.key === 'f' && !showNoteModal.value) {
    event.preventDefault()
    const searchInput = document.querySelector('.search-input') as HTMLInputElement
    if (searchInput) {
      searchInput.focus()
    }
  }
  // 关闭模态框: Esc
  if (event.key === 'Escape' && showNoteModal.value) {
    closeNoteModal()
  }
  // 保存便签: Ctrl/Cmd + Enter (在模态框中)
  if (event.key === 'Enter' && event.ctrlKey && showNoteModal.value) {
    event.preventDefault()
    handleSave()
  }
}

// 组件挂载时加载便签数据
onMounted(async () => {
  await loadStickyNotes()
  // 添加键盘事件监听器
  window.addEventListener('keydown', handleKeydown)
})

// 组件卸载时移除事件监听器
onUnmounted(() => {
  // 移除键盘事件监听器
  window.removeEventListener('keydown', handleKeydown)
})

// 防抖搜索值
const debouncedSearchQuery = ref('')

// 监听搜索输入，添加防抖
watch(searchQuery, (newValue) => {
  if (searchTimeout.value) {
    clearTimeout(searchTimeout.value)
  }
  searchTimeout.value = window.setTimeout(() => {
    debouncedSearchQuery.value = newValue
  }, 300)
})

// 过滤便签的计算属性
const filteredStickyNotes = computed(() => {
  let result = [...stickyNotes.value]
  
  // 按创建时间降序排序，最新的在前面
  result.sort((a, b) => {
    const dateA = new Date(a.created_at).getTime()
    const dateB = new Date(b.created_at).getTime()
    return dateB - dateA
  })
  
  if (debouncedSearchQuery.value) {
    const query = debouncedSearchQuery.value.toLowerCase()
    result = result.filter(note => {
      return (
        note.title.toLowerCase().includes(query) ||
        note.content.toLowerCase().includes(query)
      )
    })
  }
  
  // 标签筛选
  if (selectedTag.value) {
    result = result.filter(note => {
      const tags = note.tags ? JSON.parse(note.tags) : []
      return tags.includes(selectedTag.value)
    })
  }
  
  return result
})

// 所有标签
const allTags = computed(() => {
  const tags = new Set<string>()
  stickyNotes.value.forEach(note => {
    if (note.tags) {
      try {
        const noteTags = JSON.parse(note.tags)
        if (Array.isArray(noteTags)) {
          noteTags.forEach((tag: string) => tags.add(tag))
        }
      } catch (e) {
        // 忽略解析错误
      }
    }
  })
  return Array.from(tags)
})

</script>

<style scoped>
.sticky-notes-app {
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: var(--bg-color);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.sticky-notes-content {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
}

.sticky-notes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 24px;
  margin-top: 16px;
  transition: all 0.3s ease;
}

.empty-notes {
  grid-column: 1 / -1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
  color: var(--text-tertiary);
  transition: all 0.3s ease;
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.5;
  transition: opacity 0.3s ease;
}

.empty-notes:hover .empty-icon {
  opacity: 0.8;
}

.empty-notes p {
  margin: 0 0 8px 0;
  font-size: var(--font-size-base);
  transition: color 0.3s ease;
}

.empty-hint {
  font-size: var(--font-size-sm) !important;
  opacity: 0.7;
  transition: opacity 0.3s ease;
}

.empty-notes:hover .empty-hint {
  opacity: 1;
}

/* 模态框样式 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  animation: fadeIn 0.3s ease;
}

.modal-content {
  background-color: var(--card-bg);
  border-radius: 8px;
  width: 90%;
  max-width: 500px;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  animation: slideIn 0.3s ease;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  background-color: var(--bg-color);
  border-radius: 8px 8px 0 0;
}

.modal-header h3 {
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  transition: color 0.3s ease;
}

.modal-close {
  width: 24px;
  height: 24px;
  border: none;
  background-color: transparent;
  color: var(--text-secondary);
  font-size: var(--font-size-xl);
  font-weight: bold;
  cursor: pointer;
  border-radius: 50%;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-close:hover {
  background-color: var(--hover-color);
  color: var(--text-primary);
  transform: rotate(90deg);
}

.modal-body {
  padding: 20px;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 6px;
  transition: color 0.3s ease;
}

.form-hint {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  margin-top: 4px;
  margin-bottom: 0;
  opacity: 0.8;
}

.form-input,
.form-textarea,
.form-select {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: var(--font-size-sm);
  color: var(--text-primary);
  background-color: var(--bg-color);
  transition: all 0.3s ease;
  box-sizing: border-box;
}

.form-input:focus,
.form-textarea:focus,
.form-select:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-textarea {
  resize: vertical;
  min-height: 150px;
}

.color-picker {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.color-option {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  cursor: pointer;
  transition: all 0.3s ease;
  border: 2px solid transparent;
}

.color-option:hover {
  transform: scale(1.1);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

.color-option.active {
  border-color: var(--border-color, #333);
  transform: scale(1.1);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 0 20px 20px;
}

.modal-btn {
  padding: 8px 24px;
  border: none;
  border-radius: 6px;
  font-size: var(--font-size-sm);
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
}

.modal-btn.cancel-btn {
  background-color: var(--bg-color);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.modal-btn.cancel-btn:hover {
  background-color: var(--hover-color);
  color: var(--text-primary);
  border-color: var(--primary-color);
  transform: translateY(-1px);
}

.modal-btn.confirm-btn {
  background-color: var(--primary-color);
  color: white;
}

.modal-btn.confirm-btn:hover {
  background-color: var(--active-color);
  color: white;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

/* 动画效果 */
@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes fadeOut {
  from {
    opacity: 1;
  }
  to {
    opacity: 0;
  }
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(-20px) scale(0.95);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

@keyframes slideOut {
  from {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
  to {
    opacity: 0;
    transform: translateY(20px) scale(0.95);
  }
}

/* 响应式设计 */
@media (max-width: 768px) {
  .sticky-notes-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
    padding: 16px 20px;
    height: auto;
  }
  
  .header-right {
    width: 100%;
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }
  
  .sticky-notes-content {
    padding: 16px 20px;
  }
  
  .sticky-notes-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 480px) {
  .empty-notes {
    padding: 40px 20px;
  }
  
  .empty-icon {
    font-size: var(--font-size-3xl);
    margin-bottom: 12px;
  }
  
  .empty-notes p {
    font-size: var(--font-size-sm);
  }
  
  .empty-hint {
    font-size: var(--font-size-xxs) !important;
  }
}

/* 弹窗 footer 布局 */
.modal-footer-left,
.modal-footer-right {
  display: flex;
  gap: var(--spacing-2);
}

.modal-footer-left {
  margin-right: auto;
}
</style>

<style>
.sticky-note-modal-overlay {
  position: fixed !important;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
  animation: fadeIn 0.3s ease;
}

.sticky-note-modal-content {
  background-color: var(--card-bg);
  border-radius: 8px;
  width: 90%;
  max-width: 500px;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  animation: slideIn 0.3s ease;
}

.sticky-note-modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  background-color: var(--bg-color);
  border-radius: 8px 8px 0 0;
}

.sticky-note-modal-header h3 {
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.sticky-note-modal-close {
  width: 24px;
  height: 24px;
  border: none;
  background-color: transparent;
  color: var(--text-secondary);
  font-size: var(--font-size-xl);
  font-weight: bold;
  cursor: pointer;
  border-radius: 50%;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  justify-content: center;
}

.sticky-note-modal-close:hover {
  background-color: var(--hover-color);
  color: var(--text-primary);
  transform: rotate(90deg);
}

.sticky-note-modal-body {
  padding: 20px;
}

.sticky-note-form-group {
  margin-bottom: 16px;
}

.sticky-note-form-group label {
  display: block;
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 6px;
}

.sticky-note-form-hint {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  margin-top: 4px;
  margin-bottom: 0;
  opacity: 0.8;
}

.sticky-note-form-input,
.sticky-note-form-textarea,
.sticky-note-form-select {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: var(--font-size-sm);
  color: var(--text-primary);
  background-color: var(--bg-color);
  transition: all 0.3s ease;
  box-sizing: border-box;
}

.sticky-note-form-input:focus,
.sticky-note-form-textarea:focus,
.sticky-note-form-select:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.sticky-note-form-textarea {
  resize: vertical;
  min-height: 150px;
}

.sticky-note-color-picker {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.sticky-note-color-option {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  cursor: pointer;
  transition: all 0.2s ease;
  border: 3px solid transparent;
  position: relative;
}

.sticky-note-color-option:hover {
  transform: scale(1.15);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

.sticky-note-color-option.active {
  border-color: #5d4037;
  transform: scale(1.1);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.sticky-note-color-option.active::after {
  content: '✓';
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: var(--font-size-sm);
  font-weight: bold;
  color: rgba(0, 0, 0, 0.5);
}

.sticky-note-modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 0 20px 20px;
}

.sticky-note-modal-btn {
  padding: 8px 24px;
  border: none;
  border-radius: 6px;
  font-size: var(--font-size-sm);
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
}

.sticky-note-cancel-btn {
  background-color: var(--bg-color);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.sticky-note-cancel-btn:hover {
  background-color: var(--hover-color);
  color: var(--text-primary);
  border-color: var(--primary-color);
  transform: translateY(-1px);
}

.sticky-note-confirm-btn {
  background-color: var(--primary-color);
  color: white;
}

.sticky-note-confirm-btn:hover {
  background-color: var(--active-color);
  color: white;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

/* 实时预览卡片 */
.sticky-note-preview-wrap {
  margin-bottom: 20px;
  pointer-events: none;
}

.sticky-note-preview-wrap .sticky-note {
  min-height: 120px;
  cursor: default;
}

/* 表单两列布局 */
.sticky-note-form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 16px;
  align-items: start;
}

/* 内容字数统计 */
.sticky-note-content-header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}

.sticky-note-form-counter {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  opacity: 0.75;
}

/* 标签 chips 输入 */
.sticky-note-tag-input-wrap {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background-color: var(--bg-color);
  transition: all 0.3s ease;
  cursor: text;
}

.sticky-note-tag-input-wrap:focus-within {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.sticky-note-tag-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 6px 2px 10px;
  background: var(--primary-light);
  color: var(--primary-color);
  border-radius: 12px;
  font-size: var(--font-size-xxs);
  font-weight: 500;
  white-space: nowrap;
}

.sticky-note-tag-chip .remove-btn {
  border: none;
  background: transparent;
  color: inherit;
  cursor: pointer;
  font-size: var(--font-size-sm);
  line-height: 1;
  padding: 0 2px;
  border-radius: 50%;
  display: flex;
  align-items: center;
}

.sticky-note-tag-chip .remove-btn:hover {
  background: rgba(0, 0, 0, 0.12);
}

.sticky-note-tag-input {
  flex: 1;
  min-width: 90px;
  border: none;
  outline: none;
  background: transparent;
  padding: 2px 4px;
  font-size: var(--font-size-sm);
  color: var(--text-primary);
}

/* 快捷键提示 */
.sticky-note-shortcut-hint {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  opacity: 0.75;
  align-self: center;
  white-space: nowrap;
  margin-right: 4px;
}

@media (max-width: 768px) {
  .sticky-note-form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
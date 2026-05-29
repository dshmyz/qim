<template>
  <div class="sticky-notes-app" :class="{ fullscreen: isFullscreen }">
    <AppHeader title="便签" @back="$emit('back')">
      <template #extra-buttons>
        <ToggleSidebarBtn
          icon="fas fa-compress"
          title="收起侧边栏"
          @click="$emit('toggleSidebar')"
        />
      </template>
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
      @close="closeNoteModal"
    >
      <div class="sticky-note-form-group">
        <label>标题</label>
        <input type="text" class="sticky-note-form-input" v-model="formData.title" placeholder="便签标题">
      </div>
      <div class="sticky-note-form-group">
        <label>内容</label>
        <textarea class="sticky-note-form-textarea" v-model="formData.content" placeholder="便签内容"></textarea>
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
            @click="formData.color = color.value"
          ></div>
        </div>
      </div>
      <div class="sticky-note-form-group">
        <label>提醒时间</label>
        <input 
          type="datetime-local" 
          class="sticky-note-form-input"
          :value="formatReminderForInput(formData.reminder)"
          @input="formData.reminder = ($event.target as HTMLInputElement).value"
          placeholder="设置提醒时间"
        >
      </div>
      <div class="sticky-note-form-group">
        <label>标签</label>
        <input 
          type="text" 
          class="sticky-note-form-input"
          v-model="formData.tags"
          placeholder="输入标签，用逗号分隔"
        >
        <p class="sticky-note-form-hint">示例：工作, 个人, 重要</p>
      </div>
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
      
      <template #footer>
        <div class="modal-footer-left">
          <button 
            v-if="selectedNote" 
            class="sticky-note-modal-btn sticky-note-ai-btn" 
            @click="analyzeStickyNote"
            title="AI 分析"
          >
            <i class="fas fa-magic"></i>
          </button>
          <button 
            class="sticky-note-modal-btn sticky-note-fullscreen-btn" 
            @click="toggleFullscreen"
            :title="isFullscreen ? '退出全屏' : '全屏'"
          >
            <i :class="isFullscreen ? 'fas fa-compress' : 'fas fa-expand'"></i>
          </button>
        </div>
        <div class="modal-footer-right">
          <button class="sticky-note-modal-btn sticky-note-cancel-btn" @click="closeNoteModal">取消</button>
          <button class="sticky-note-modal-btn sticky-note-confirm-btn" @click="selectedNote ? updateStickyNote() : createStickyNote()">{{ selectedNote ? '更新' : '创建' }}</button>
        </div>
      </template>
    </ModalContainer>
    
    <!-- AI 分析结果弹窗 -->
    <div v-if="showAIAnalysis" class="ai-analysis-overlay" @click.self="showAIAnalysis = false">
      <div class="ai-analysis-modal">
        <div class="ai-analysis-header">
          <h3>AI 分析结果</h3>
          <button class="close-btn" @click="showAIAnalysis = false">
            <i class="fas fa-times"></i>
          </button>
        </div>
        <div class="ai-analysis-body">
          <div class="ai-section">
            <h4>摘要</h4>
            <p>{{ aiAnalysisResult?.summary || '暂无摘要' }}</p>
          </div>
          <div class="ai-section">
            <h4>推荐标签</h4>
            <div class="ai-tags">
              <span 
                v-for="tag in aiAnalysisResult?.tags || []" 
                :key="tag" 
                class="ai-tag"
              >
                {{ tag }}
              </span>
            </div>
          </div>
          <div class="ai-section" v-if="aiAnalysisResult?.action_items?.length">
            <h4>行动项</h4>
            <ul>
              <li v-for="item in aiAnalysisResult.action_items" :key="item">{{ item }}</li>
            </ul>
          </div>
        </div>
        <div class="ai-analysis-footer">
          <button class="ai-btn cancel" @click="showAIAnalysis = false">取消</button>
          <button class="ai-btn confirm" @click="applyAIResult(aiAnalysisResult?.summary, aiAnalysisResult?.tags)">应用标签</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import axios from 'axios'
import QMessage from '../../utils/qmessage'
import { useServerUrl } from '../../composables/useServerUrl'
import { logger } from '../../utils/logger';
import AppHeader from './AppHeader.vue'
import ToggleSidebarBtn from '../shared/ToggleSidebarBtn.vue'
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
const isFullscreen = ref(false)
const selectedTag = ref<string | null>(null)
const showAIAnalysis = ref(false)
const aiAnalysisResult = ref<any>(null)
const analyzingNote = ref<any>(null)
const newNote = ref({
  title: '',
  content: '',
  color: 'yellow',
  reminder: '',
  tags: '',
  paperStyle: 'plain',
  fontFamily: "Arial, 'Microsoft YaHei', sans-serif"
})
const selectedNote = ref<any>(null)
const formData = ref({
  title: '',
  content: '',
  color: 'yellow',
  reminder: '',
  tags: '',
  paperStyle: 'plain',
  fontFamily: "Arial, 'Microsoft YaHei', sans-serif"
})
const draggedNoteId = ref<string | null>(null)
const searchQuery = ref('')
const searchTimeout = ref<number | null>(null)

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

const formatReminderForInput = (reminder: string): string => {
  if (!reminder) return ''
  const d = new Date(reminder)
  if (isNaN(d.getTime())) return ''
  const year = d.getFullYear()
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const hours = String(d.getHours()).padStart(2, '0')
  const minutes = String(d.getMinutes()).padStart(2, '0')
  return `${year}-${month}-${day}T${hours}:${minutes}`
}
const serializeStyle = () => {
  return JSON.stringify({
    color: formData.value.color,
    paperStyle: formData.value.paperStyle,
    fontFamily: formData.value.fontFamily
  })
}

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
    const tagsArray = formData.value.tags
      ? formData.value.tags.split(',').map(tag => tag.trim()).filter(tag => tag)
      : []
    
    const response = await axios.post(`${serverUrl.value}/api/v1/notes`, {
      title: formData.value.title,
      content: formData.value.content,
      color: formData.value.color,
      reminder: formData.value.reminder,
      tags: tagsArray,
      type: 'sticky',
      style: serializeStyle()
    }, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    stickyNotes.value.push(response.data.data)
    if (formData.value.reminder) {
      setupReminder(response.data.data)
    }
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
    const tagsArray = formData.value.tags
      ? formData.value.tags.split(',').map(tag => tag.trim()).filter(tag => tag)
      : []
    
    const response = await axios.put(`${serverUrl.value}/api/v1/notes/${selectedNote.value.id}`, {
      title: formData.value.title,
      content: formData.value.content,
      color: formData.value.color,
      reminder: formData.value.reminder,
      tags: tagsArray,
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
      if (formData.value.reminder) {
        setupReminder(response.data.data)
      }
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
    reminder: '',
    tags: '',
    paperStyle: 'plain',
    fontFamily: "Arial, 'Microsoft YaHei', sans-serif"
  }
  selectedNote.value = null
  showNoteModal.value = true
}

// 显示编辑便签模态框
const showEditNoteModal = (note: any) => {
  selectedNote.value = { ...note }
  const parsedStyle = parseStyle(note.style)
  formData.value = {
    title: note.title,
    content: note.content,
    color: parsedStyle.color,
    reminder: note.reminder || '',
    tags: Array.isArray(note.tags) ? note.tags.join(', ') : note.tags || '',
    paperStyle: parsedStyle.paperStyle,
    fontFamily: parsedStyle.fontFamily
  }
  showNoteModal.value = true
}

// 关闭便签模态框
const closeNoteModal = () => {
  showNoteModal.value = false
  selectedNote.value = null
  isFullscreen.value = false
  formData.value = {
    title: '',
    content: '',
    color: 'yellow',
    reminder: '',
    tags: '',
    paperStyle: 'plain',
    fontFamily: "Arial, 'Microsoft YaHei', sans-serif"
  }
}

// 全屏切换
const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value
}

// AI 分析便签
const analyzeStickyNote = async () => {
  if (!selectedNote.value) return
  
  analyzingNote.value = selectedNote.value
  try {
    const response = await axios.post(
      `${serverUrl.value}/api/v1/notes/${selectedNote.value.id}/analyze`,
      {},
      {
        headers: { Authorization: `Bearer ${getToken()}` }
      }
    )
    
    if (response.data.code === 0) {
      aiAnalysisResult.value = response.data.data
      showAIAnalysis.value = true
    }
  } catch (error) {
    QMessage.error('AI 分析失败')
  }
}

// 应用 AI 分析结果
const applyAIResult = (summary: string, tags: string[]) => {
  if (selectedNote.value) {
    formData.value.tags = JSON.stringify(tags)
  }
  showAIAnalysis.value = false
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

// 接收添加到笔记事件
const handleAddToNote = async (event: CustomEvent) => {
  const { title, content } = event.detail
  // 填充表单数据
  formData.value = {
    title: title,
    content: content,
    color: 'yellow',
    reminder: '',
    tags: '',
    paperStyle: 'plain',
    fontFamily: "Arial, 'Microsoft YaHei', sans-serif"
  }
  selectedNote.value = null
  // 自动创建笔记并保存到后端
  await createStickyNote()
  logger.log('收到添加到笔记:', { title, content })
}

// 处理键盘快捷键
const handleKeydown = (event: KeyboardEvent) => {
  // 新建便签: Ctrl/Cmd + N
  if ((event.ctrlKey || event.metaKey) && event.key === 'n') {
    event.preventDefault()
    showCreateNoteModal()
  }
  // 聚焦搜索框: Ctrl/Cmd + F
  if ((event.ctrlKey || event.metaKey) && event.key === 'f') {
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
  // 保存便签: Enter (在模态框中)
  if (event.key === 'Enter' && event.ctrlKey && showNoteModal.value) {
    event.preventDefault()
    if (selectedNote.value) {
      updateStickyNote()
    } else {
      createStickyNote()
    }
  }
}

// 组件挂载时加载便签数据
onMounted(async () => {
  await loadStickyNotes()
  // 添加添加到笔记事件监听器
  window.addEventListener('addToNote', handleAddToNote as unknown as EventListener)
  // 添加键盘事件监听器
  window.addEventListener('keydown', handleKeydown)
})

// 组件卸载时移除事件监听器
onUnmounted(() => {
  // 移除添加到笔记事件监听器
  window.removeEventListener('addToNote', handleAddToNote as unknown as EventListener)
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

// 设置提醒
const setupReminder = (note: any) => {
  const reminderTime = new Date(note.reminder).getTime()
  const now = new Date().getTime()
  
  if (reminderTime > now) {
    const timeUntilReminder = reminderTime - now
    
    setTimeout(() => {
      // 显示提醒通知
      if ('Notification' in window && Notification.permission === 'granted') {
        new Notification('便签提醒', {
          body: `${note.title}\n${note.content}`,
          icon: '/favicon.ico'
        })
      } else if ('Notification' in window && Notification.permission !== 'denied') {
        Notification.requestPermission().then(permission => {
          if (permission === 'granted') {
            new Notification('便签提醒', {
              body: `${note.title}\n${note.content}`,
              icon: '/favicon.ico'
            })
          }
        })
      }
      
      // 可以在这里添加其他提醒方式，如声音、弹窗等
    }, timeUntilReminder)
  }
}
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
  font-size: 16px;
  transition: color 0.3s ease;
}

.empty-hint {
  font-size: 14px !important;
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
  font-size: 16px;
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
  font-size: 20px;
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
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 6px;
  transition: color 0.3s ease;
}

.form-hint {
  font-size: 12px;
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
  font-size: 14px;
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
  border-color: #333;
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
  font-size: 14px;
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
    font-size: 32px;
    margin-bottom: 12px;
  }
  
  .empty-notes p {
    font-size: 14px;
  }
  
  .empty-hint {
    font-size: 12px !important;
  }
}

/* 全屏模式 */
.sticky-notes-app.fullscreen {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1000;
  background: var(--content-bg);
}

.sticky-notes-app.fullscreen .sticky-notes-header,
.sticky-notes-app.fullscreen .sticky-notes-grid {
  display: none;
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

.sticky-note-ai-btn {
  background: linear-gradient(135deg, var(--primary-color), var(--color-primary-600)) !important;
  border-color: transparent !important;
  color: white !important;
}

.sticky-note-ai-btn:hover {
  opacity: 0.9;
}

.sticky-note-fullscreen-btn {
  background: var(--btn-bg) !important;
}

.sticky-note-fullscreen-btn:hover {
  background: var(--primary-light) !important;
  color: var(--primary-color);
}

/* AI 分析弹窗 */
.ai-analysis-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1001;
}

.ai-analysis-modal {
  background: var(--card-bg);
  border-radius: var(--radius-xl);
  width: 90%;
  max-width: 480px;
  max-height: 80vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-xl);
  border: 1px solid var(--border-color);
}

.ai-analysis-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-4) var(--spacing-5);
  border-bottom: 1px solid var(--border-color);
  background: linear-gradient(135deg, var(--primary-color), var(--color-primary-600));
}

.ai-analysis-header h3 {
  margin: 0;
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-bold);
  color: white;
}

.ai-analysis-header .close-btn {
  background: rgba(255, 255, 255, 0.2);
  border: none;
  color: white;
  cursor: pointer;
  font-size: var(--font-size-base);
  padding: var(--spacing-2);
  border-radius: var(--radius-md);
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.ai-analysis-body {
  padding: var(--spacing-5);
  overflow-y: auto;
}

.ai-section {
  margin-bottom: var(--spacing-4);
}

.ai-section:last-child {
  margin-bottom: 0;
}

.ai-section h4 {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
  margin: 0 0 var(--spacing-2) 0;
}

.ai-section p {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  margin: 0;
  padding: var(--spacing-3);
  background: var(--content-bg);
  border-radius: var(--radius-md);
}

.ai-tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-2);
}

.ai-tag {
  font-size: var(--font-size-sm);
  padding: var(--spacing-2) var(--spacing-3);
  background: var(--primary-light);
  color: var(--primary-color);
  border-radius: var(--radius-full);
}

.ai-section ul {
  margin: 0;
  padding-left: var(--spacing-5);
}

.ai-section li {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  line-height: 1.8;
}

.ai-analysis-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--spacing-3);
  padding: var(--spacing-4) var(--spacing-5);
  border-top: 1px solid var(--border-color);
}

.ai-btn {
  padding: var(--spacing-2) var(--spacing-5);
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  cursor: pointer;
  transition: all var(--transition-base);
}

.ai-btn.cancel {
  background: var(--btn-bg);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.ai-btn.cancel:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.ai-btn.confirm {
  background: linear-gradient(135deg, var(--primary-color), var(--color-primary-600));
  color: white;
  border: none;
}

.ai-btn.confirm:hover {
  opacity: 0.9;
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
  font-size: 16px;
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
  font-size: 20px;
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
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 6px;
}

.sticky-note-form-hint {
  font-size: 12px;
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
  font-size: 14px;
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
  font-size: 14px;
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
  font-size: 14px;
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
</style>
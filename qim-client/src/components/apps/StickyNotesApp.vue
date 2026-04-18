<template>
  <div class="sticky-notes-app">
    <div class="sticky-notes-header">
      <div class="header-left">
        <button class="back-btn" @click="$emit('back')">
          <i class="fas fa-arrow-left"></i>
        </button>
        <h2>便签</h2>
      </div>
      <div class="header-right">
        <div class="search-box">
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
    </div>
    <div class="sticky-notes-content">
      <div class="sticky-notes-grid">
        <div 
          v-for="(note, index) in filteredStickyNotes" 
          :key="note.id"
          class="sticky-note"
          :class="[note.color, note.paperStyle]"
          :style="{ fontFamily: note.fontFamily || 'Comic Sans MS, Marker Felt, cursive' }"
          :data-note-id="note.id"
          @click="showEditNoteModal(note)"
          draggable="true"
          @dragstart="onDragStart($event, note.id)"
          @dragover.prevent
          @drop="onDrop($event, index)"
        >
          <div class="sticky-note-pin">
            <i class="fas fa-thumbtack"></i>
          </div>
          <div class="sticky-note-header">
            <div class="sticky-note-title-container">
              <h3 class="sticky-note-title">{{ note.title }}</h3>
              <span v-if="note.reminder" class="sticky-note-reminder">
                <i class="fas fa-bell"></i>
              </span>
            </div>
            <div class="sticky-note-actions">
              <button class="sticky-note-action" @click.stop="shareNote(note)">
                <i class="fas fa-share"></i>
              </button>
              <button class="sticky-note-delete" @click.stop="deleteStickyNote(note.id)">
                <i class="fas fa-trash"></i>
              </button>
            </div>
          </div>
          <div class="sticky-note-content">{{ note.content }}</div>
          <div v-if="note.tags && note.tags.length > 0" class="sticky-note-tags">
            <span 
              v-for="(tag, index) in note.tags" 
              :key="index"
              class="sticky-note-tag"
            >
              {{ tag }}
            </span>
          </div>
          <div class="sticky-note-footer">
            <span class="sticky-note-date">{{ formatDate(note.createdAt) }}</span>
          </div>
        </div>
        <div v-if="stickyNotes.length === 0" class="empty-notes">
          <div class="empty-icon"><i class="fas fa-sticky-note"></i></div>
          <p>暂无便签</p>
          <p class="empty-hint">点击右上角按钮创建新便签</p>
        </div>
      </div>
    </div>

    <!-- 便签模态框 -->
    <div v-if="showNoteModal" class="modal-overlay" @click="closeNoteModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>{{ selectedNote ? '编辑便签' : '创建便签' }}</h3>
          <button class="modal-close" @click="closeNoteModal">×</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>标题</label>
            <input type="text" class="form-input" v-model="formData.title" placeholder="便签标题">
          </div>
          <div class="form-group">
            <label>内容</label>
            <textarea class="form-textarea" v-model="formData.content" placeholder="便签内容"></textarea>
          </div>
          <div class="form-group">
            <label>颜色</label>
            <div class="color-picker">
              <div 
                v-for="color in noteColors" 
                :key="color.value"
                class="color-option"
                :class="{ active: formData.color === color.value }"
                :style="{ backgroundColor: color.value }"
                @click="formData.color = color.value"
              ></div>
            </div>
          </div>
          <div class="form-group">
            <label>提醒时间</label>
            <input 
              type="datetime-local" 
              class="form-input" 
              v-model="formData.reminder"
              placeholder="设置提醒时间"
            >
          </div>
          <div class="form-group">
            <label>标签</label>
            <input 
              type="text" 
              class="form-input" 
              v-model="formData.tags"
              placeholder="输入标签，用逗号分隔"
            >
            <p class="form-hint">示例：工作, 个人, 重要</p>
          </div>
          <div class="form-group">
            <label>纸张样式</label>
            <select class="form-select" v-model="formData.paperStyle">
              <option value="plain">普通纸张</option>
              <option value="lined">横线纸张</option>
              <option value="grid">网格纸张</option>
              <option value="dotted">点阵纸张</option>
            </select>
          </div>
          <div class="form-group">
            <label>字体</label>
            <select class="form-select" v-model="formData.fontFamily">
              <option value="'Comic Sans MS', 'Marker Felt', cursive">手写体</option>
              <option value="'Arial', 'Helvetica', sans-serif">Arial</option>
              <option value="'Georgia', 'Times New Roman', serif">Georgia</option>
              <option value="'Courier New', monospace">Courier New</option>
            </select>
          </div>
        </div>
        <div class="modal-footer">
          <button class="modal-btn cancel-btn" @click="closeNoteModal">取消</button>
          <button class="modal-btn confirm-btn" @click="selectedNote ? updateStickyNote() : createStickyNote()">{{ selectedNote ? '更新' : '创建' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { API_BASE_URL } from '../../config'

// 服务器URL
const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

// 获取token
const getToken = () => {
  return localStorage.getItem('token')
}

// 便签应用相关状态
const stickyNotes = ref<any[]>([])
const showNoteModal = ref(false)
const newNote = ref({
  title: '',
  content: '',
  color: 'yellow',
  reminder: '',
  tags: '',
  paperStyle: 'plain',
  fontFamily: "'Comic Sans MS', 'Marker Felt', cursive"
})
const selectedNote = ref<any>(null)
const formData = ref({
  title: '',
  content: '',
  color: 'yellow',
  reminder: '',
  tags: '',
  paperStyle: 'plain',
  fontFamily: "'Comic Sans MS', 'Marker Felt', cursive"
})
const draggedNoteId = ref<string | null>(null)
const searchQuery = ref('')
const searchTimeout = ref<number | null>(null)

// 便签颜色选项
const noteColors = [
  { name: '黄色', value: 'yellow' },
  { name: '蓝色', value: 'blue' },
  { name: '绿色', value: 'green' },
  { name: '红色', value: 'red' },
  { name: '紫色', value: 'purple' },
  { name: '粉色', value: 'pink' }
]

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
    ElMessage.error('加载便签失败，请稍后重试')
  }
}

// 创建便签
const createStickyNote = async () => {
  try {
    const token = getToken()
    // 处理标签，将逗号分隔的字符串转换为数组
    const tagsArray = formData.value.tags
      ? formData.value.tags.split(',').map(tag => tag.trim()).filter(tag => tag)
      : []
    
    const response = await axios.post(`${serverUrl.value}/api/v1/notes`, {
      ...formData.value,
      tags: tagsArray,
      type: 'sticky'
    }, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    stickyNotes.value.push(response.data.data)
    // 设置提醒
    if (formData.value.reminder) {
      setupReminder(response.data.data)
    }
    closeNoteModal()
  } catch (error) {
    console.error('创建便签失败:', error)
    ElMessage.error('创建便签失败，请稍后重试')
  }
}

// 更新便签
const updateStickyNote = async () => {
  try {
    const token = getToken()
    // 处理标签，将逗号分隔的字符串转换为数组
    const tagsArray = formData.value.tags
      ? formData.value.tags.split(',').map(tag => tag.trim()).filter(tag => tag)
      : []
    
    const updatedNote = {
      ...selectedNote.value,
      title: formData.value.title,
      content: formData.value.content,
      color: formData.value.color,
      reminder: formData.value.reminder,
      tags: tagsArray,
      paperStyle: formData.value.paperStyle,
      fontFamily: formData.value.fontFamily,
      createdAt: selectedNote.value.createdAt || new Date().toISOString()
    }
    const response = await axios.put(`${serverUrl.value}/api/v1/notes/${selectedNote.value.id}`, updatedNote, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    const index = stickyNotes.value.findIndex(note => note.id === selectedNote.value.id)
    if (index !== -1) {
      stickyNotes.value[index] = response.data.data
      // 更新提醒
      if (formData.value.reminder) {
        setupReminder(response.data.data)
      }
    }
    closeNoteModal()
  } catch (error) {
    console.error('更新便签失败:', error)
    ElMessage.error('更新便签失败，请稍后重试')
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
    fontFamily: "'Comic Sans MS', 'Marker Felt', cursive"
  }
  selectedNote.value = null
  showNoteModal.value = true
}

// 显示编辑便签模态框
const showEditNoteModal = (note: any) => {
  selectedNote.value = { ...note }
  formData.value = {
    title: note.title,
    content: note.content,
    color: note.color,
    reminder: note.reminder || '',
    tags: Array.isArray(note.tags) ? note.tags.join(', ') : note.tags || '',
    paperStyle: note.paperStyle || 'plain',
    fontFamily: note.fontFamily || "'Comic Sans MS', 'Marker Felt', cursive"
  }
  showNoteModal.value = true
}

// 关闭便签模态框
const closeNoteModal = () => {
  showNoteModal.value = false
  selectedNote.value = null
  formData.value = {
    title: '',
    content: '',
    color: 'yellow',
    reminder: '',
    tags: '',
    paperStyle: 'plain',
    fontFamily: "'Comic Sans MS', 'Marker Felt', cursive"
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
      // 移除被拖拽的便签
      const [draggedNote] = stickyNotes.value.splice(draggedIndex, 1)
      // 插入到新位置
      stickyNotes.value.splice(targetIndex, 0, draggedNote)
    }
  }
  // 移除拖拽状态
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
  
  console.log('转发笔记到聊天窗口:', note)
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
    fontFamily: "'Comic Sans MS', 'Marker Felt', cursive"
  }
  selectedNote.value = null
  // 自动创建笔记并保存到后端
  await createStickyNote()
  console.log('收到添加到笔记:', { title, content })
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
  window.addEventListener('addToNote', handleAddToNote as EventListener)
  // 添加键盘事件监听器
  window.addEventListener('keydown', handleKeydown)
})

// 组件卸载时移除事件监听器
onUnmounted(() => {
  // 移除添加到笔记事件监听器
  window.removeEventListener('addToNote', handleAddToNote as EventListener)
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
    const dateA = new Date(a.createdAt).getTime()
    const dateB = new Date(b.createdAt).getTime()
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
  
  return result
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

.sticky-notes-header {
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

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
}

.search-input {
  padding: 8px 12px 8px 36px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 14px;
  background-color: var(--bg-color);
  color: var(--text-primary);
  transition: all 0.3s ease;
  width: 200px;
}

.search-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  width: 250px;
}

.search-icon {
  position: absolute;
  left: 12px;
  color: var(--text-secondary);
  font-size: 14px;
  transition: color 0.3s ease;
}

.search-input:focus + .search-icon {
  color: var(--primary-color);
}

.sticky-notes-header:hover {
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

.sticky-notes-header h2 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
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

.sticky-notes-content {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
}

.sticky-notes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 20px;
  transition: all 0.3s ease;
}

.sticky-note {
  background-color: #fffb96;
  border-radius: 4px;
  padding: 14px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.12);
  cursor: pointer;
  transition: all 0.3s ease;
  min-height: 180px;
  position: relative;
  overflow: hidden;
  transform: rotate(-0.5deg);
  font-family: 'Comic Sans MS', 'Marker Felt', cursive;
  padding-top: 28px;
  animation: noteAppear 0.5s ease-out;
}

@keyframes noteAppear {
  from {
    opacity: 0;
    transform: rotate(-0.5deg) scale(0.8) translateY(20px);
  }
  to {
    opacity: 1;
    transform: rotate(-0.5deg) scale(1) translateY(0);
  }
}

.sticky-note.deleting {
  animation: noteDisappear 0.3s ease-in forwards;
}

@keyframes noteDisappear {
  from {
    opacity: 1;
    transform: rotate(-0.5deg) scale(1);
  }
  to {
    opacity: 0;
    transform: rotate(-0.5deg) scale(0.8) translateY(-20px);
  }
}

.sticky-note::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  width: 0;
  height: 0;
  border-style: solid;
  border-width: 0 20px 20px 0;
  border-color: transparent #e6e288 transparent transparent;
  box-shadow: -2px 2px 5px rgba(0, 0, 0, 0.1);
  transform-origin: top left;
  transform: rotate(-5deg);
  transition: all 0.3s ease;
  z-index: 0;
}

.sticky-note:hover::before {
  transform: rotate(-8deg) scale(1.05);
  box-shadow: -3px 3px 8px rgba(0, 0, 0, 0.15);
}

.sticky-note-pin {
  position: absolute;
  top: 3px;
  left: 15px;
  width: 16px;
  height: 16px;
  background-color: #a0a0a0;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
  z-index: 1;
  transition: all 0.3s ease;
}

.sticky-note-pin i {
  font-size: 10px;
  color: #666;
  transform: rotate(45deg);
}

.sticky-note:hover .sticky-note-pin {
  transform: rotate(10deg);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.4);
}

.sticky-note:hover {
  transform: rotate(0.5deg) translateY(-3px);
  box-shadow: 0 3px 10px rgba(0, 0, 0, 0.18);
}

.sticky-note.dragging {
  opacity: 0.6;
  transform: rotate(5deg) scale(1.05);
  box-shadow: 0 5px 15px rgba(0, 0, 0, 0.2);
  z-index: 100;
}

.sticky-note.yellow {
  background-color: #fffb96;
}

.sticky-note.yellow::before {
  border-color: transparent #e6e288 transparent transparent;
}

.sticky-note.blue {
  background-color: #96e0ff;
}

.sticky-note.blue::before {
  border-color: transparent #83c6e6 transparent transparent;
}

.sticky-note.green {
  background-color: #96ff9e;
}

.sticky-note.green::before {
  border-color: transparent #83e689 transparent transparent;
}

.sticky-note.red {
  background-color: #ff9696;
}

.sticky-note.red::before {
  border-color: transparent #e68383 transparent transparent;
}

.sticky-note.purple {
  background-color: #d996ff;
}

.sticky-note.purple::before {
  border-color: transparent #c083e6 transparent transparent;
}

.sticky-note.pink {
  background-color: #ff96d9;
}

.sticky-note.pink::before {
  border-color: transparent #e683c0 transparent transparent;
}

/* 纸张样式 */
.sticky-note.lined {
  background-image: linear-gradient(#f0f0f0 1px, transparent 1px);
  background-size: 100% 18px;
  background-position: 0 28px;
}

.sticky-note.grid {
  background-image: 
    linear-gradient(#f0f0f0 1px, transparent 1px),
    linear-gradient(90deg, #f0f0f0 1px, transparent 1px);
  background-size: 18px 18px;
  background-position: 0 28px, 0 0;
}

.sticky-note.dotted {
  background-image: radial-gradient(#f0f0f0 1px, transparent 1px);
  background-size: 18px 18px;
  background-position: 9px 37px;
}

.sticky-note-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
  transition: all 0.3s ease;
}

.sticky-note-title-container {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.sticky-note-reminder {
  color: #ff9800;
  font-size: 12px;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    transform: scale(1.1);
    opacity: 0.8;
  }
  100% {
    transform: scale(1);
    opacity: 1;
  }
}

.sticky-note-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.sticky-note-title {
  font-size: 14px;
  font-weight: 700;
  color: #333;
  margin: 0;
  transition: color 0.3s ease;
  word-break: break-word;
}

.sticky-note-action,
.sticky-note-delete {
  width: 24px;
  height: 24px;
  border: none;
  background-color: rgba(0, 0, 0, 0.1);
  color: #666;
  border-radius: 50%;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transform: scale(0.8);
}

.sticky-note:hover .sticky-note-action,
.sticky-note:hover .sticky-note-delete {
  opacity: 1;
  transform: scale(1);
}

.sticky-note-action:hover {
  background-color: rgba(59, 130, 246, 0.2);
  color: #3b82f6;
  transform: scale(1.1);
}

.sticky-note-delete:hover {
  background-color: rgba(255, 0, 0, 0.2);
  color: #ff0000;
  transform: scale(1.1);
}

.sticky-note-content {
  font-size: 13px;
  line-height: 1.45;
  color: #444;
  margin-bottom: 12px;
  flex: 1;
  transition: color 0.3s ease;
  word-break: break-word;
  white-space: pre-wrap;
  opacity: 0.9;
}

.sticky-note-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
}

.sticky-note-tag {
  background-color: rgba(0, 0, 0, 0.1);
  color: #333;
  font-size: 10px;
  padding: 2px 8px;
  border-radius: 10px;
  transition: all 0.3s ease;
}

.sticky-note:hover .sticky-note-tag {
  background-color: rgba(0, 0, 0, 0.15);
  transform: scale(1.05);
}

.sticky-note-footer {
  position: absolute;
  bottom: 8px;
  left: 14px;
  right: 14px;
  font-size: 11px;
  color: #666;
  transition: color 0.3s ease;
  opacity: 0.7;
}

.sticky-note-date {
  transition: opacity 0.3s ease;
}

.sticky-note:hover .sticky-note-date {
  opacity: 1;
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
  
  .search-box {
    width: 100%;
  }
  
  .search-input {
    width: 100%;
  }
  
  .search-input:focus {
    width: 100%;
  }
  
  .create-note-btn {
    align-self: stretch;
  }
  
  .sticky-notes-content {
    padding: 16px 20px;
  }
  
  .sticky-notes-grid {
    grid-template-columns: 1fr;
  }
  
  .sticky-note {
    min-height: 180px;
    padding: 12px;
  }
  
  .sticky-note-title {
    font-size: 14px;
  }
  
  .sticky-note-content {
    font-size: 13px;
  }
  
  .modal-content {
    width: 95%;
    margin: 20px;
  }
  
  .modal-header h3 {
    font-size: 14px;
  }
  
  .modal-body {
    padding: 16px;
  }
  
  .form-group label {
    font-size: 13px;
  }
  
  .form-input,
  .form-textarea,
  .form-select {
    font-size: 14px;
    padding: 8px 12px;
  }
  
  .form-textarea {
    min-height: 120px;
  }
  
  .modal-footer {
    padding: 0 16px 16px;
  }
  
  .modal-btn {
    padding: 8px 20px;
    font-size: 13px;
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
</style>
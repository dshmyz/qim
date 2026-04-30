<template>
  <div class="tasks-app">
    <AppHeader title="任务管理" @back="$emit('back')">
      <template #extra-buttons>
        <button class="icon-btn" @click="$emit('toggleSidebar')">
          <i class="fas fa-compress"></i>
        </button>
      </template>
      <template #actions>
        <button class="action-btn primary" @click="showCreateTaskModal = true">+ 新建任务</button>
      </template>
    </AppHeader>
    <div class="tasks-content">
      <div class="tasks-search-box">
        <input 
          type="text" 
          v-model="taskSearchQuery" 
          placeholder="搜索任务..." 
          class="tasks-search-input"
        />
        <i class="fas fa-search tasks-search-icon"></i>
      </div>
      <div class="tasks-board">
        <!-- 待办任务 -->
        <div class="task-column">
          <div class="task-column-header">
            <h3>待办</h3>
            <span class="task-count">{{ todoTasks.length }}</span>
          </div>
          <div class="task-list">
            <div 
              v-for="task in todoTasks" 
              :key="task.id"
              class="task-card"
              @click="selectTask(task)"
            >
              <div class="task-title">{{ task.title }}</div>
              <div class="task-description">{{ task.description }}</div>
              <div class="task-meta">
                <span class="task-due-date">{{ formatTaskDate(task.due_date) }}</span>
                <span :class="['task-priority', task.priority]">{{ task.priority }}</span>
              </div>
              <div class="task-actions">
                <button class="task-action-btn" @click.stop="updateTaskStatus(task.id, 'in_progress')">开始</button>
                <button class="task-action-btn delete-btn" @click.stop="deleteTask(task.id)">删除</button>
              </div>
            </div>
          </div>
        </div>
        
        <!-- 进行中任务 -->
        <div class="task-column">
          <div class="task-column-header">
            <h3>进行中</h3>
            <span class="task-count">{{ inProgressTasks.length }}</span>
          </div>
          <div class="task-list">
            <div 
              v-for="task in inProgressTasks" 
              :key="task.id"
              class="task-card"
              @click="selectTask(task)"
            >
              <div class="task-title">{{ task.title }}</div>
              <div class="task-description">{{ task.description }}</div>
              <div class="task-meta">
                <span class="task-due-date">{{ formatTaskDate(task.due_date) }}</span>
                <span :class="['task-priority', task.priority]">{{ task.priority }}</span>
              </div>
              <div class="task-actions">
                <button class="task-action-btn" @click.stop="updateTaskStatus(task.id, 'completed')">完成</button>
                <button class="task-action-btn delete-btn" @click.stop="deleteTask(task.id)">删除</button>
              </div>
            </div>
          </div>
        </div>
        
        <!-- 已完成任务 -->
        <div class="task-column">
          <div class="task-column-header">
            <h3>已完成</h3>
            <span class="task-count">{{ completedTasks.length }}</span>
          </div>
          <div class="task-list">
            <div 
              v-for="task in completedTasks" 
              :key="task.id"
              class="task-card completed"
              @click="selectTask(task)"
            >
              <div class="task-title">{{ task.title }}</div>
              <div class="task-description">{{ task.description }}</div>
              <div class="task-meta">
                <span class="task-due-date">{{ formatTaskDate(task.due_date) }}</span>
                <span :class="['task-priority', task.priority]">{{ task.priority }}</span>
              </div>
              <div class="task-actions">
                <button class="task-action-btn" @click.stop="updateTaskStatus(task.id, 'todo')">重新开始</button>
                <button class="task-action-btn delete-btn" @click.stop="deleteTask(task.id)">删除</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 创建任务模态框 -->
    <ModalContainer
      :visible="showCreateTaskModal"
      :title="selectedTask ? '编辑任务' : '创建任务'"
      @close="closeCreateTaskModal"
    >
      <div class="task-form-group">
        <label>任务标题</label>
        <input type="text" class="task-form-input" v-model="taskForm.title" placeholder="请输入任务标题">
      </div>
      <div class="task-form-group">
        <label>任务描述</label>
        <textarea class="task-form-textarea" v-model="taskForm.description" placeholder="请输入任务描述"></textarea>
      </div>
      <div class="task-form-group">
        <label>截止日期</label>
        <input type="date" class="task-form-input" v-model="taskForm.due_date">
      </div>
      <div class="task-form-group">
        <label>优先级</label>
        <select class="task-form-select" v-model="taskForm.priority">
          <option value="low">低</option>
          <option value="medium">中</option>
          <option value="high">高</option>
        </select>
      </div>
      <div class="task-form-group">
        <label>状态</label>
        <select class="task-form-select" v-model="taskForm.status">
          <option value="todo">待办</option>
          <option value="in_progress">进行中</option>
          <option value="completed">已完成</option>
        </select>
      </div>
      
      <template #footer>
        <button class="task-modal-btn task-cancel-btn" @click="closeCreateTaskModal">取消</button>
        <button class="task-modal-btn task-confirm-btn" @click="selectedTask ? updateTask() : createTask()">{{ selectedTask ? '更新' : '创建' }}</button>
      </template>
    </ModalContainer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'
import QMessage from '../../utils/qmessage'
import { API_BASE_URL } from '../../config'
import ModalContainer from '../../components/shared/ModalContainer.vue'
import AppHeader from './AppHeader.vue'

// 服务器URL
const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

// 定义事件
const emit = defineEmits(['back', 'toggleSidebar'])

// 任务管理相关状态
const tasks = ref<any[]>([])
const taskSearchQuery = ref('')
const showCreateTaskModal = ref(false)
const selectedTask = ref<any>(null)
const taskForm = ref({
  title: '',
  description: '',
  due_date: '',
  priority: 'medium',
  status: 'todo'
})

// 计算属性：过滤后的任务
const filteredTasks = computed(() => {
  if (!taskSearchQuery.value) {
    return tasks.value
  }
  const query = taskSearchQuery.value.toLowerCase()
  return tasks.value.filter(task => 
    task.title.toLowerCase().includes(query) ||
    task.description.toLowerCase().includes(query)
  )
})

// 计算属性：按状态分类的任务
const todoTasks = computed(() => {
  return filteredTasks.value.filter(task => task.status === 'todo')
})

const inProgressTasks = computed(() => {
  return filteredTasks.value.filter(task => task.status === 'in_progress')
})

const completedTasks = computed(() => {
  return filteredTasks.value.filter(task => task.status === 'completed')
})

// 获取token
const getToken = () => {
  return localStorage.getItem('token')
}

// 加载任务列表
const loadTasks = async () => {
  try {
    const token = getToken()
    const response = await axios.get(`${serverUrl.value}/api/v1/tasks`, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    if (response.data.code === 0) {
      tasks.value = response.data.data
    }
  } catch (error) {
    console.error('加载任务失败:', error)
    QMessage.error('加载任务失败，请稍后重试')
  }
}

// 创建任务
const createTask = async () => {
  if (!taskForm.value.title.trim()) {
    QMessage.error('请输入任务标题')
    return
  }

  try {
    const token = getToken()
    const response = await axios.post(`${serverUrl.value}/api/v1/tasks`, taskForm.value, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    if (response.data.code === 0) {
      tasks.value.push(response.data.data)
      closeCreateTaskModal()
      QMessage.success('任务创建成功')
    } else {
      QMessage.error(response.data.message || '创建任务失败')
    }
  } catch (error) {
    console.error('创建任务失败:', error)
    QMessage.error('创建任务失败，请稍后重试')
  }
}

// 更新任务
const updateTask = async () => {
  if (!taskForm.value.title.trim()) {
    QMessage.error('请输入任务标题')
    return
  }

  try {
    const token = getToken()
    const response = await axios.put(`${serverUrl.value}/api/v1/tasks/${selectedTask.value.id}`, taskForm.value, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    if (response.data.code === 0) {
      const index = tasks.value.findIndex(task => task.id === selectedTask.value.id)
      if (index !== -1) {
        tasks.value[index] = response.data.data
      }
      closeCreateTaskModal()
      QMessage.success('任务更新成功')
    } else {
      QMessage.error(response.data.message || '更新任务失败')
    }
  } catch (error) {
    console.error('更新任务失败:', error)
    QMessage.error('更新任务失败，请稍后重试')
  }
}

// 删除任务
const deleteTask = async (taskId: string) => {
  try {
    const token = getToken()
    await axios.delete(`${serverUrl.value}/api/v1/tasks/${taskId}`, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    tasks.value = tasks.value.filter(task => task.id !== taskId)
  } catch (error) {
    console.error('删除任务失败:', error)
    QMessage.error('删除任务失败，请稍后重试')
  }
}

// 更新任务状态
const updateTaskStatus = async (taskId: string, status: string) => {
  try {
    const token = getToken()
    await axios.patch(`${serverUrl.value}/api/v1/tasks/${taskId}/status`, { status }, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    const task = tasks.value.find(task => task.id === taskId)
    if (task) {
      task.status = status
    }
  } catch (error) {
    console.error('更新任务状态失败:', error)
    QMessage.error('更新任务状态失败，请稍后重试')
  }
}

// 选择任务
const selectTask = (task: any) => {
  selectedTask.value = { ...task }
  taskForm.value = {
    title: task.title,
    description: task.description,
    due_date: task.due_date,
    priority: task.priority,
    status: task.status
  }
  showCreateTaskModal.value = true
}

// 关闭创建任务模态框
const closeCreateTaskModal = () => {
  showCreateTaskModal.value = false
  selectedTask.value = null
  taskForm.value = {
    title: '',
    description: '',
    due_date: '',
    priority: 'medium',
    status: 'todo'
  }
}

// 格式化任务日期
const formatTaskDate = (dateString: string) => {
  if (!dateString) return ''
  const date = new Date(dateString)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}

// 组件挂载时加载任务列表
onMounted(async () => {
  await loadTasks()
})
</script>

<style scoped>
.tasks-app {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--content-bg);
  overflow: hidden;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  min-width: 0;
}

.tasks-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 16px;
}

.tasks-search-box {
  position: relative;
  margin-bottom: 16px;
  flex-shrink: 0;
}

.tasks-search-input {
  width: 100%;
  padding: 10px 40px 10px 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: 14px;
  background: var(--input-bg);
  color: var(--text-color);
  transition: all 0.2s ease;
  box-sizing: border-box;
}

.tasks-search-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.tasks-search-icon {
  position: absolute;
  right: 14px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-secondary);
  font-size: 16px;
}

.tasks-board {
  flex: 1;
  display: flex;
  gap: 16px;
  overflow-x: auto;
  padding-bottom: 16px;
  min-height: 0;
  max-width: 100%;
  transition: all 0.3s ease;
}

.task-column {
  flex: 1;
  min-width: 240px;
  max-width: 320px;
  background: var(--card-bg);
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition: all 0.2s ease;
}

.task-column:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.task-column-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
  background: var(--card-header-bg);
}

.task-column-header h3 {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color);
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.task-count {
  background: var(--text-secondary);
  color: white;
  font-size: 12px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 10px;
  min-width: 20px;
  text-align: center;
}

.task-list {
  flex: 1;
  padding: 16px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 0;
}

.task-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  animation: fadeIn 0.3s ease;
}

.task-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  transform: translateY(-1px);
  border-color: var(--primary-color);
}

.task-card.completed {
  opacity: 0.7;
  background: var(--success-light);
  border-color: var(--success-color);
}

.task-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color);
  margin-bottom: 8px;
  line-height: 1.4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.task-description {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 12px;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
}

.task-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  font-size: 12px;
}

.task-due-date {
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: 4px;
}

.task-due-date::before {
  content: '📅';
  font-size: 12px;
}

.task-priority {
  padding: 2px 8px;
  border-radius: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  font-size: 11px;
}

.task-priority.low {
  background: var(--success-light);
  color: var(--success-color);
}

.task-priority.medium {
  background: var(--warning-light);
  color: var(--warning-color);
}

.task-priority.high {
  background: var(--danger-light);
  color: var(--danger-color);
}

.task-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

.task-action-btn {
  padding: 6px 12px;
  border: 1px solid var(--border-color);
  background: var(--card-bg);
  color: var(--text-color);
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
}

.task-action-btn:hover {
  background: var(--hover-bg);
  border-color: var(--primary-color);
  color: var(--primary-color);
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.task-action-btn.delete-btn:hover {
  background: var(--danger-light);
  border-color: var(--danger-color);
  color: var(--danger-color);
}

/* 模态框样式 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  animation: fadeIn 0.3s ease;
}

.modal-content {
  background: var(--card-bg);
  border-radius: 8px;
  width: 90%;
  max-width: 500px;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
  animation: slideIn 0.3s ease;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
  margin: 0;
}

.modal-close {
  background: none;
  border: none;
  font-size: 24px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: color 0.2s ease;
  padding: 4px 8px;
  border-radius: 4px;
}

.modal-close:hover {
  background: var(--hover-color);
  color: var(--text-color);
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
  color: var(--text-color);
  margin-bottom: 8px;
}

.form-input,
.form-textarea,
.form-select {
  width: 100%;
  padding: 10px 14px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 14px;
  background: var(--input-bg);
  color: var(--text-color);
  transition: all 0.3s ease;
  box-sizing: border-box;
}

.form-input:focus,
.form-textarea:focus,
.form-select:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  transform: translateY(-1px);
}

.form-textarea {
  resize: vertical;
  min-height: 100px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 20px;
  border-top: 1px solid var(--border-color);
  background: var(--card-bg);
}

.modal-btn {
  padding: 8px 16px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
}

.modal-btn.cancel-btn {
  background: var(--card-bg);
  color: var(--text-color);
}

.modal-btn.cancel-btn:hover {
  background: var(--hover-bg);
  transform: translateY(-1px);
}

.modal-btn.confirm-btn {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.modal-btn.confirm-btn:hover {
  background: var(--primary-hover);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
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

@keyframes slideIn {
  from {
    transform: translateY(-20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

/* 响应式设计 */
@media (max-width: 768px) {
  .tasks-content {
    padding: 12px;
  }
  
  .tasks-board {
    gap: 12px;
  }
  
  .task-column {
    min-width: 250px;
  }
  
  .task-card {
    padding: 12px;
  }
  
  .modal-content {
    width: 95%;
    margin: 20px;
  }
  
  .tasks-search-input {
    padding: 8px 32px 8px 12px;
  }
  
  .task-action-btn {
    padding: 4px 8px;
    font-size: 11px;
  }
}

/* 滚动条样式 */
.task-list::-webkit-scrollbar {
  width: 6px;
}

.task-list::-webkit-scrollbar-track {
  background: var(--content-bg);
  border-radius: 3px;
}

.task-list::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
}

.task-list::-webkit-scrollbar-thumb:hover {
  background: var(--text-secondary);
}

.tasks-board::-webkit-scrollbar {
  height: 6px;
}

.tasks-board::-webkit-scrollbar-track {
  background: var(--content-bg);
  border-radius: 3px;
}

.tasks-board::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
}

.tasks-board::-webkit-scrollbar-thumb:hover {
  background: var(--text-secondary);
}

.task-form-group {
  margin-bottom: 16px;
}

.task-form-group label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 6px;
}

.task-form-input,
.task-form-textarea,
.task-form-select {
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

.task-form-input:focus,
.task-form-textarea:focus,
.task-form-select:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.task-form-textarea {
  resize: vertical;
  min-height: 100px;
}

.task-modal-btn {
  padding: 8px 24px;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
}

.task-cancel-btn {
  background-color: var(--bg-color);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.task-cancel-btn:hover {
  background-color: var(--hover-color);
  color: var(--text-primary);
  border-color: var(--primary-color);
  transform: translateY(-1px);
}

.task-confirm-btn {
  background-color: var(--primary-color);
  color: white;
}

.task-confirm-btn:hover {
  background-color: var(--active-color);
  color: white;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}
</style>
<template>
  <div class="app-management-app">
    <AppHeader title="应用管理" @back="$emit('back')">
      <template #extra-buttons>
        <button class="toggle-sidebar-btn" @click="$emit('toggleSidebar')">
          <i class="fas fa-compress"></i>
        </button>
      </template>
      <template #actions>
        <button class="create-app-btn" @click="showCreateAppModal">+ 创建应用</button>
      </template>
    </AppHeader>
    
    <!-- 应用列表 -->
    <div class="app-list" v-if="userApps.length > 0">
      <div class="app-list-header">
        <div>应用名称</div>
        <div>图标</div>
        <div>应用链接</div>
        <div>分类</div>
        <div>操作</div>
      </div>
      <div class="app-list-body">
        <div 
          v-for="app in userApps" 
          :key="app.id"
          class="app-list-item"
          @click="openApp(app)"
        >
          <div class="app-list-name">{{ app.name }}</div>
          <div class="app-list-icon"><i :class="app.icon"></i></div>
          <div class="app-list-url">{{ app.url || '无' }}</div>
          <div class="app-list-category">{{ app.category }}</div>
          <div class="app-list-actions">
            <button class="app-action-btn edit-btn" @click.stop="showEditAppModal(app)">
              <i class="fas fa-edit"></i>
            </button>
            <button class="app-action-btn delete-btn" @click.stop="deleteApp(app.id)">
              <i class="fas fa-trash"></i>
            </button>
          </div>
        </div>
      </div>
    </div>
    
    <!-- 空状态 -->
    <div v-else class="empty-apps">
      <div class="empty-icon"><i class="fas fa-app-store"></i></div>
      <p>暂无应用</p>
      <p class="empty-hint">点击右上角按钮创建新应用</p>
    </div>
    
    <!-- 创建/编辑应用模态框 -->
    <ModalContainer
      :visible="showAppModal"
      :title="selectedApp ? '编辑应用' : '创建应用'"
      @close="closeAppModal"
    >
      <div class="app-form-group">
        <label>应用名称</label>
        <input 
          v-model="formData.name" 
          type="text" 
          class="app-form-input" 
          placeholder="请输入应用名称"
        />
      </div>
      <div class="app-form-group">
        <label>应用图标</label>
        <input 
          v-model="formData.icon" 
          type="text" 
          class="app-form-input" 
          placeholder="请输入图标类名 (例如: fas fa-star)"
        />
      </div>
      <div class="app-form-group">
        <label>应用链接</label>
        <input 
          v-model="formData.url" 
          type="url" 
          class="app-form-input" 
          placeholder="请输入应用URL"
        />
      </div>
      <div class="app-form-group">
        <label>应用分类</label>
        <select v-model="formData.category" class="app-form-select">
          <option value="productivity">生产力</option>
          <option value="communication">通讯</option>
          <option value="entertainment">娱乐</option>
          <option value="education">教育</option>
          <option value="other">其他</option>
        </select>
      </div>
      <div class="app-form-group">
        <label>应用类型</label>
        <select v-model="formData.openType" class="app-form-select">
          <option value="in-app">内嵌应用</option>
          <option value="external">外链应用</option>
        </select>
      </div>
      
      <template #footer>
        <button class="app-modal-btn app-cancel-btn" @click="closeAppModal">取消</button>
        <button class="app-modal-btn app-confirm-btn" @click="saveApp">{{ selectedApp ? '保存' : '创建' }}</button>
      </template>
    </ModalContainer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { API_BASE_URL } from '../../config'
import { logger } from '../../utils/logger';
import AppHeader from './AppHeader.vue'
import ModalContainer from '../../components/shared/ModalContainer.vue'

// 定义事件
const emit = defineEmits(['back', 'toggleSidebar'])

// 应用列表
const userApps = ref<any[]>([])

// 模态框状态
const showAppModal = ref(false)
const selectedApp = ref<any>(null)
const formData = ref({
  name: '',
  icon: 'fas fa-star',
  category: 'productivity',
  url: '',
  status: 'active',
  openType: 'in-app' // in-app: 在应用内打开, external: 使用默认浏览器打开
})

// 加载用户应用
const loadApps = async () => {
  try {
    const token = localStorage.getItem('token')
    const serverUrl = localStorage.getItem('serverUrl') || API_BASE_URL
    logger.log('加载应用列表，服务器地址:', serverUrl)
    const response = await axios.get(`${serverUrl}/api/v1/apps`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    logger.log('加载应用列表响应:', response.data)
    if (response.data.code === 0) {
      // 后端返回的数据结构是 { list: [...], pagination: {...} }
      const appsArray = response.data.data.list || response.data.data
      if (Array.isArray(appsArray)) {
        // 处理后端返回的open_type字段
        userApps.value = appsArray.map((app: any) => ({
          ...app,
          openType: app.open_type || app.openType || 'in-app' // 默认为在应用内打开
        }))
        logger.log('应用列表加载成功:', userApps.value)
      } else {
        console.error('应用列表数据格式异常:', response.data.data)
      }
    } else {
      console.error('加载应用列表失败:', response.data.message)
    }
  } catch (error) {
    console.error('加载应用列表异常:', error)
  }
}

// 打开应用
const openApp = (app: any) => {
  logger.log('打开应用:', app.name)
  
  // 触发自定义事件，通知父组件（Main.vue）打开应用
  const event = new CustomEvent('open-user-app', {
    detail: app
  })
  window.dispatchEvent(event)
  logger.log('已发送打开应用事件:', app)
}

// 显示创建应用模态框
const showCreateAppModal = () => {
  formData.value = {
    name: '',
    icon: 'fas fa-star',
    category: 'productivity',
    url: '',
    status: 'active',
    openType: 'in-app'
  }
  selectedApp.value = null
  showAppModal.value = true
}

// 显示编辑应用模态框
const showEditAppModal = (app: any) => {
  selectedApp.value = { ...app }
  formData.value = {
    name: app.name,
    icon: app.icon,
    category: app.category,
    url: app.url,
    status: app.status,
    openType: app.openType || 'in-app' // 默认为在应用内打开
  }
  showAppModal.value = true
}

// 关闭应用模态框
const closeAppModal = () => {
  showAppModal.value = false
  selectedApp.value = null
}

// 保存应用
const saveApp = async () => {
  if (!formData.value.name || formData.value.name.trim() === '') {
    alert('请输入应用名称')
    return
  }
  
  try {
    const token = localStorage.getItem('token')
    const serverUrl = localStorage.getItem('serverUrl') || API_BASE_URL
    let response
    
    // 转换 openType 为后端期望的 open_type 字段
    const { openType, ...restFormData } = formData.value
    const payload = {
      ...restFormData,
      open_type: openType
    }
    
    if (selectedApp.value) {
      logger.log('编辑应用:', payload)
      response = await axios.put(`${serverUrl}/api/v1/apps/${selectedApp.value.id}`, payload, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })
    } else {
      logger.log('创建应用:', payload)
      response = await axios.post(`${serverUrl}/api/v1/apps`, payload, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })
    }
    
    logger.log('保存应用响应:', response.data)
    if (response.data.code === 0) {
      closeAppModal()
      await loadApps()
      window.dispatchEvent(new CustomEvent('refresh-user-apps'))
      logger.log('应用保存成功')
    } else {
      console.error('应用保存失败:', response.data.message)
    }
  } catch (error) {
    console.error('应用保存异常:', error)
  }
}

// 删除应用
const deleteApp = async (appId: number) => {
  if (confirm('确定要删除这个应用吗？')) {
    try {
      const token = localStorage.getItem('token')
      const serverUrl = localStorage.getItem('serverUrl') || API_BASE_URL
      logger.log('删除应用:', appId)
      const response = await axios.delete(`${serverUrl}/api/v1/apps/${appId}`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })
      logger.log('删除应用响应:', response.data)
      if (response.data.code === 0) {
        await loadApps()
        // 通知父组件重新加载用户应用
        window.dispatchEvent(new CustomEvent('refresh-user-apps'))
        logger.log('应用删除成功')
      } else {
        console.error('应用删除失败:', response.data.message)
      }
    } catch (error) {
      console.error('应用删除异常:', error)
    }
  }
}

// 组件挂载时加载应用
onMounted(() => {
  loadApps()
})
</script>

<style scoped>
.app-management-app {
  height: 100%;
  overflow-y: auto;
}

.app-list {
  background: var(--list-bg);
  border-radius: 8px;
  overflow: hidden;
  box-shadow: var(--shadow-sm);
  margin: 20px;
}

.app-list-header {
  display: grid;
  grid-template-columns: 2fr 1fr 2fr 1fr 1fr;
  gap: 12px;
  padding: 12px 16px;
  background: var(--content-bg);
  border-bottom: 1px solid var(--border-color);
  font-weight: 600;
  font-size: 14px;
  color: var(--text-color);
}

.app-list-body {
  max-height: 400px;
  overflow-y: auto;
}

.app-list-item {
  display: grid;
  grid-template-columns: 2fr 1fr 2fr 1fr 1fr;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  align-items: center;
  transition: background-color 0.2s ease;
  cursor: pointer;
}

.app-list-item:hover {
  background: var(--hover-color);
}

.app-list-item:last-child {
  border-bottom: none;
}

.app-list-icon i {
  font-size: 18px;
  color: var(--primary-color);
}

.app-list-url {
  font-size: 14px;
  color: var(--text-secondary);
  word-break: break-all;
}

.app-list-category {
  font-size: 14px;
  color: var(--text-color);
}

.app-list-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

.app-action-btn {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  font-size: 14px;
}

.app-action-btn.edit-btn {
  background: rgba(59, 130, 246, 0.1);
  color: var(--primary-color);
}

.app-action-btn.edit-btn:hover {
  background: rgba(59, 130, 246, 0.2);
  transform: scale(1.05);
}

.app-action-btn.delete-btn {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.app-action-btn.delete-btn:hover {
  background: rgba(239, 68, 68, 0.2);
  transform: scale(1.05);
}

.empty-apps {
  padding: 40px 20px;
  text-align: center;
  color: var(--text-secondary);
  font-size: 14px;
  margin: 20px;
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
  color: var(--text-secondary);
  opacity: 0.6;
}

.empty-hint {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 8px;
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
}

.modal-content {
  background: var(--list-bg);
  border-radius: 12px;
  width: 480px;
  max-width: 90%;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  overflow: hidden;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  background: var(--header-panel-bg);
  border-bottom: 1px solid var(--border-color);
}

.modal-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
}

.modal-close {
  border: none;
  background: transparent;
  font-size: 20px;
  cursor: pointer;
  color: var(--text-secondary);
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  transition: all 0.2s ease;
}

.modal-close:hover {
  background: var(--hover-color);
  color: var(--text-color);
  transform: scale(1.05);
}

.modal-body {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  background: var(--content-bg);
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-group label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.form-input,
.form-select {
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--content-bg);
  color: var(--text-color);
  font-size: 14px;
  transition: all 0.3s ease;
}

.form-input:focus,
.form-select:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
}

.modal-footer {
  padding: 0 24px 24px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  background: var(--content-bg);
}

.modal-btn {
  padding: 10px 24px;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  border: 1px solid transparent;
  box-shadow: var(--shadow-sm);
}

.modal-btn.cancel-btn {
  background: var(--list-bg);
  color: var(--text-color);
}

.modal-btn.cancel-btn:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
}

.modal-btn.confirm-btn {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.modal-btn.confirm-btn:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
  opacity: 0.95;
  color: var(--text-light);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .app-list-header,
  .app-list-item {
    grid-template-columns: 1fr;
    gap: 8px;
    text-align: left;
  }
  
  .app-list-actions {
    justify-content: flex-start;
  }
  
  .create-app-btn {
    padding: 6px 12px;
  }
}

.app-form-group {
  margin-bottom: 16px;
}

.app-form-group label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 6px;
}

.app-form-input,
.app-form-select {
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

.app-form-input:focus,
.app-form-select:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.app-modal-btn {
  padding: 8px 24px;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
}

.app-cancel-btn {
  background-color: var(--bg-color);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.app-cancel-btn:hover {
  background-color: var(--hover-color);
  color: var(--text-primary);
  border-color: var(--primary-color);
  transform: translateY(-1px);
}

.app-confirm-btn {
  background-color: var(--primary-color);
  color: white;
}

.app-confirm-btn:hover {
  background-color: var(--active-color);
  color: white;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}
</style>

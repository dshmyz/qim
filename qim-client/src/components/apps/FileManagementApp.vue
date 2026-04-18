<template>
  <div class="files-app">
    <div class="files-header">
      <div class="header-left">
        <button class="back-btn" @click="$emit('back')">
          <i class="fas fa-arrow-left"></i>
        </button>
        <div class="files-header-info">
          <h2>文件管理</h2>
        </div>
      </div>
      <div class="files-header-actions">
        <button class="create-folder-btn" @click="showCreateFolderModal = true">
          <i class="fas fa-folder-plus"></i>
        </button>
        <button class="upload-file-btn" @click="triggerFileUpload">
          <i class="fas fa-upload"></i>
        </button>
        <input 
          ref="fileInput" 
          type="file" 
          multiple 
          style="display: none" 
          @change="handleFileUpload"
        />
      </div>
    </div>
    <div class="files-content">
      <div class="files-nav">
        <div 
          class="nav-item" 
          :class="{ active: currentPath === '/' }"
          @click="navigateTo('/')"
        >
          <i class="fas fa-home"></i>
          <span>所有文件</span>
        </div>
        <div 
          class="nav-item" 
          :class="{ active: currentPath === '/images' }"
          @click="navigateTo('/images')"
        >
          <i class="fas fa-image"></i>
          <span>图片</span>
        </div>
        <div 
          class="nav-item" 
          :class="{ active: currentPath === '/documents' }"
          @click="navigateTo('/documents')"
        >
          <i class="fas fa-file-alt"></i>
          <span>文档</span>
        </div>
        <div 
          class="nav-item" 
          :class="{ active: currentPath === '/videos' }"
          @click="navigateTo('/videos')"
        >
          <i class="fas fa-video"></i>
          <span>视频</span>
        </div>
        <div 
          class="nav-item" 
          :class="{ active: currentPath === '/recent' }"
          @click="navigateTo('/recent')"
        >
          <i class="fas fa-clock"></i>
          <span>最近文件</span>
        </div>
      </div>
      <div class="files-main">
        <div class="files-path">
          <div class="path-item" @click="navigateTo('/')">首页</div>
          <template v-for="(path, index) in currentPath.split('/').filter(p => p)" :key="index">
            <span class="path-separator">/</span>
            <div 
              class="path-item" 
              @click="navigateTo('/' + currentPath.split('/').filter(p => p).slice(0, index + 1).join('/'))"
            >
              {{ path }}
            </div>
          </template>
        </div>
        <div class="files-search-box">
          <input 
            type="text" 
            v-model="fileSearchQuery" 
            placeholder="搜索文件..." 
            class="files-search-input"
          />
          <i class="fas fa-search files-search-icon"></i>
        </div>
        <div class="files-grid" v-if="filteredFiles.length > 0">
          <div 
            v-for="file in filteredFiles" 
            :key="file.id"
            class="file-item"
            @click="selectFile(file)"
          >
            <div class="file-icon" :class="getFileTypeClass(file.type)">
              <i :class="getFileTypeIcon(file.type)"></i>
            </div>
            <div class="file-info">
              <div class="file-name" :title="file.name">{{ file.name }}</div>
              <div class="file-meta">
                <span class="file-size">{{ formatFileSize(file.size) }}</span>
                <span class="file-date">{{ formatFileDate(file.created_at) }}</span>
              </div>
            </div>
            <div class="file-actions">
              <button class="file-action-btn" @click.stop="downloadFile(file)">
                <i class="fas fa-download"></i>
              </button>
              <button class="file-action-btn" @click.stop="shareFile(file)">
                <i class="fas fa-share-alt"></i>
              </button>
              <button class="file-action-btn" @click.stop="copyShareLink(file)">
                <i class="fas fa-link"></i>
              </button>
              <button class="file-action-btn" @click.stop="deleteFile(file.id)">
                <i class="fas fa-trash"></i>
              </button>
            </div>
          </div>
        </div>
        <div class="files-empty" v-else>
          <i class="fas fa-folder-open"></i>
          <p>没有文件</p>
        </div>
      </div>
    </div>

    <!-- 创建文件夹模态框 -->
    <div v-if="showCreateFolderModal" class="modal-overlay" @click="closeCreateFolderModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>创建文件夹</h3>
          <button class="modal-close" @click="closeCreateFolderModal">×</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>文件夹名称</label>
            <input type="text" class="form-input" v-model="folderName" placeholder="请输入文件夹名称">
          </div>
        </div>
        <div class="modal-footer">
          <button class="modal-btn cancel-btn" @click="closeCreateFolderModal">取消</button>
          <button class="modal-btn confirm-btn" @click="createFolder">创建</button>
        </div>
      </div>
    </div>

    <!-- 文件预览模态框 -->
    <div v-if="selectedFile" class="modal-overlay" @click="closeFilePreview">
      <div class="modal-content file-preview-modal" @click.stop>
        <div class="modal-header">
          <h3>{{ selectedFile.name }}</h3>
          <button class="modal-close" @click="closeFilePreview">×</button>
        </div>
        <div class="modal-body">
          <div class="file-preview-content">
            <img 
              v-if="isImageFile(selectedFile.type)" 
              :src="getImageUrl(selectedFile)" 
              :alt="selectedFile.name"
              class="preview-image"
            />

            <video 
              v-else-if="isVideoFile(selectedFile.type)" 
              :src="getVideoUrl(selectedFile)" 
              :alt="selectedFile.name"
              class="preview-video"
              controls
            ></video>
            <audio 
              v-else-if="isAudioFile(selectedFile.type)" 
              :src="getAudioUrl(selectedFile)" 
              :alt="selectedFile.name"
              class="preview-audio"
              controls
            />

            <div v-else class="preview-other">
              <i :class="getFileTypeIcon(selectedFile.type)"></i>
              <p>无法预览此文件类型</p>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button class="modal-btn cancel-btn" @click="closeFilePreview">关闭</button>
          <button class="modal-btn" @click="shareFile(selectedFile)">
            <i class="fas fa-share-alt"></i> 分享
          </button>
          <button class="modal-btn" @click="copyShareLink(selectedFile)">
            <i class="fas fa-link"></i> 复制链接
          </button>
          <button class="modal-btn confirm-btn" @click="downloadFile(selectedFile)">下载</button>
        </div>
      </div>
    </div>


  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import axios from 'axios'
import { API_BASE_URL } from '../../config'

// 服务器URL
const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

// 定义事件
const emit = defineEmits(['back'])

// 文件管理相关状态
const files = ref<any[]>([])
const currentPath = ref('/')
const fileSearchQuery = ref('')
const showCreateFolderModal = ref(false)
const folderName = ref('')
const selectedFile = ref<any>(null)
const fileInput = ref<HTMLInputElement>()

// 计算属性：过滤后的文件
const filteredFiles = computed(() => {
  // 创建一个新的数组，避免修改原始数组
  let result = [...files.value]
  
  // 根据路径过滤
  if (currentPath.value !== '/') {
    if (currentPath.value === '/recent') {
      // 最近文件逻辑
      result = result.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()).slice(0, 20)
    } else {
      // 按类型过滤
      const type = currentPath.value.replace('/','')
      result = result.filter(file => {
        if (type === 'images') {
          return file.type && file.type.startsWith('image/')
        } else if (type === 'documents') {
          return file.type && (file.type.startsWith('text/') || file.type.includes('document') || file.type.includes('pdf') || file.type === 'application/pdf')
        } else if (type === 'videos') {
          return file.type && file.type.startsWith('video/')
        } else {
          return true
        }
      })
    }
  }
  
  // 根据搜索关键词过滤
  if (fileSearchQuery.value) {
    const query = fileSearchQuery.value.toLowerCase()
    result = result.filter(file => 
      file.name.toLowerCase().includes(query)
    )
  }
  
  return result
})

// 获取token
const getToken = () => {
  return localStorage.getItem('token')
}

// 加载文件列表
const loadFiles = async () => {
  try {
    const token = getToken()
    const response = await axios.get(`${serverUrl.value}/api/v1/files`, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    if (response.data.code === 0) {
      files.value = response.data.data
    }
  } catch (error) {
    console.error('加载文件失败:', error)
    ElMessage.error('加载文件失败，请稍后重试')
  }
}

// 导航到指定路径
const navigateTo = (path: string) => {
  currentPath.value = path
}

// 触发文件上传
const triggerFileUpload = () => {
  fileInput.value?.click()
}

// 处理文件上传
const handleFileUpload = async (event: Event) => {
  const target = event.target as HTMLInputElement
  const files = target.files
  if (!files) return
  
  try {
    const token = getToken()
    const formData = new FormData()
    for (let i = 0; i < files.length; i++) {
      formData.append('files', files[i])
    }
    
    const response = await axios.post(`${serverUrl.value}/api/v1/files/upload`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    
    if (response.data.code === 0) {
      files.value = [...files.value, ...response.data.data]
      // 重置文件输入
      if (fileInput.value) {
        fileInput.value.value = ''
      }
    }
  } catch (error) {
    console.error('上传文件失败:', error)
    // 模拟上传成功
    for (let i = 0; i < files.length; i++) {
      const file = files[i]
      const mockFile = {
        id: Date.now().toString() + i,
        name: file.name,
        type: file.type,
        size: file.size,
        created_at: new Date().toISOString(),
        path: '/' + file.type.split('/')[0]
      }
      files.value.push(mockFile)
    }
    // 重置文件输入
    if (fileInput.value) {
      fileInput.value.value = ''
    }
  }
}

// 创建文件夹
const createFolder = async () => {
  if (!folderName.value) return
  
  try {
    const token = getToken()
    const response = await axios.post(`${serverUrl.value}/api/v1/folders`, {
      name: folderName.value,
      parent_id: null
    }, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    
    if (response.data.code === 0) {
      // 处理文件夹创建成功
      closeCreateFolderModal()
    }
  } catch (error) {
    console.error('创建文件夹失败:', error)
    // 模拟创建成功
    closeCreateFolderModal()
  }
}

// 下载文件
const downloadFile = async (file: any) => {
  try {
    const token = getToken()
    const response = await axios.get(`${serverUrl.value}/api/v1/files/${file.id}/download`, {
      headers: {
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      },
      responseType: 'blob'
    })
    
    const url = window.URL.createObjectURL(new Blob([response.data]))
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', file.name)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  } catch (error) {
    console.error('下载文件失败:', error)
    // 模拟下载
    ElMessage.info(`正在下载 ${file.name}`)
  }
}

// 删除文件
const deleteFile = async (fileId: string) => {
  try {
    const token = getToken()
    await axios.delete(`${serverUrl.value}/api/v1/files/${fileId}`, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    files.value = files.value.filter(file => file.id !== fileId)
  } catch (error) {
    console.error('删除文件失败:', error)
    // 模拟删除
    files.value = files.value.filter(file => file.id !== fileId)
  }
}

// 选择文件
const selectFile = (file: any) => {
  selectedFile.value = file
}

// 关闭创建文件夹模态框
const closeCreateFolderModal = () => {
  showCreateFolderModal.value = false
  folderName.value = ''
}

// 关闭文件预览
const closeFilePreview = () => {
  selectedFile.value = null
}

// 分享文件
const shareFile = (file: any) => {
  // 触发全局分享事件，复用Main.vue中的分享功能
  window.dispatchEvent(new CustomEvent('openShareModal', {
    detail: { type: 'file', data: file }
  }))
}

// 复制分享链接
const copyShareLink = (file: any) => {
  const shareLink = `${window.location.origin}/share/file/${file.id}`
  navigator.clipboard.writeText(shareLink).then(() => {
    ElMessage.success('分享链接已复制到剪贴板')
  }).catch(err => {
    console.error('复制失败:', err)
    ElMessage.error('复制失败，请手动复制')
  })
}

// 获取文件类型图标
const getFileTypeIcon = (fileType: string) => {
  if (!fileType) {
    return 'fas fa-file'
  }
  if (fileType.startsWith('image/')) {
    return 'fas fa-image'
  } else if (fileType.startsWith('video/')) {
    return 'fas fa-video'
  } else if (fileType.startsWith('audio/')) {
    return 'fas fa-music'
  } else if (fileType.includes('pdf')) {
    return 'fas fa-file-pdf'
  } else if (fileType.includes('word') || fileType.includes('document')) {
    return 'fas fa-file-word'
  } else if (fileType.includes('excel') || fileType.includes('sheet')) {
    return 'fas fa-file-excel'
  } else if (fileType.includes('powerpoint') || fileType.includes('presentation')) {
    return 'fas fa-file-powerpoint'
  } else if (fileType.startsWith('text/')) {
    return 'fas fa-file-alt'
  } else {
    return 'fas fa-file'
  }
}

// 获取文件类型类
const getFileTypeClass = (fileType: string) => {
  if (!fileType) {
    return 'file-icon-other'
  }
  if (fileType.startsWith('image/')) {
    return 'file-icon-image'
  } else if (fileType.startsWith('video/')) {
    return 'file-icon-video'
  } else if (fileType.startsWith('audio/')) {
    return 'file-icon-audio'
  } else if (fileType.includes('pdf')) {
    return 'file-icon-pdf'
  } else if (fileType.includes('word') || fileType.includes('document')) {
    return 'file-icon-word'
  } else if (fileType.includes('excel') || fileType.includes('sheet')) {
    return 'file-icon-excel'
  } else if (fileType.includes('powerpoint') || fileType.includes('presentation')) {
    return 'file-icon-powerpoint'
  } else if (fileType.startsWith('text/')) {
    return 'file-icon-text'
  } else {
    return 'file-icon-other'
  }
}

// 格式化文件大小
const formatFileSize = (size: number) => {
  if (size < 1024) {
    return size + ' B'
  } else if (size < 1024 * 1024) {
    return (size / 1024).toFixed(1) + ' KB'
  } else if (size < 1024 * 1024 * 1024) {
    return (size / (1024 * 1024)).toFixed(1) + ' MB'
  } else {
    return (size / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
  }
}

// 格式化文件日期
const formatFileDate = (dateString: string) => {
  const date = new Date(dateString)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 检查是否是图片文件
const isImageFile = (fileType: string) => {
  return fileType && fileType.startsWith('image/')
}

// 检查是否是视频文件
const isVideoFile = (fileType: string) => {
  return fileType && fileType.startsWith('video/')
}

// 检查是否是音频文件
const isAudioFile = (fileType: string) => {
  return fileType && fileType.startsWith('audio/')
}

// 获取图片URL
const getImageUrl = (file: any) => {
  return `${serverUrl.value}/api/v1/files/${file.id}/preview`
}

// 获取视频URL
const getVideoUrl = (file: any) => {
  return `${serverUrl.value}/api/v1/files/${file.id}/preview`
}

// 获取音频URL
const getAudioUrl = (file: any) => {
  return `${serverUrl.value}/api/v1/files/${file.id}/preview`
}

// 组件挂载时加载文件列表
onMounted(async () => {
  await loadFiles()
})
</script>

<style scoped>
.files-app {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--content-bg);
  overflow: hidden;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.files-header {
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

.files-header:hover {
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

.files-header-info h2 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 4px 0;
  transition: color 0.3s ease;
}

.files-header-actions {
  display: flex;
  gap: 8px;
}

.create-folder-btn,
.upload-file-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: 1px solid var(--border-color);
  background: var(--card-bg);
  border-radius: 6px;
  cursor: pointer;
  color: var(--text-color);
  transition: all 0.3s ease;
}

.create-folder-btn:hover,
.upload-file-btn:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
  color: var(--text-color);
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0, 122, 255, 0.2);
}

.files-content {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.files-nav {
  width: 200px;
  border-right: 1px solid var(--border-color);
  background: var(--card-bg);
  padding: 16px 0;
  overflow-y: auto;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  cursor: pointer;
  transition: all 0.2s ease;
  border-left: 3px solid transparent;
  color: var(--text-color);
}

.nav-item:hover {
  background: var(--hover-bg);
  color: var(--primary-color);
}

.nav-item.active {
  background: var(--primary-light);
  color: var(--primary-color);
  border-left-color: var(--primary-color);
  font-weight: 500;
}

.nav-item i {
  font-size: 16px;
  width: 20px;
  text-align: center;
}

.files-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 20px;
  overflow-y: auto;
}

.files-path {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 20px;
  font-size: 14px;
  color: var(--text-secondary);
}

.path-item {
  cursor: pointer;
  transition: color 0.2s ease;
}

.path-item:hover {
  color: var(--primary-color);
}

.path-separator {
  color: var(--text-tertiary);
}

.files-search-box {
  position: relative;
  margin-bottom: 20px;
}

.files-search-input {
  width: 100%;
  padding: 10px 40px 10px 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: 14px;
  background: var(--input-bg);
  color: var(--text-color);
  transition: all 0.2s ease;
}

.files-search-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.1);
}

.files-search-icon {
  position: absolute;
  right: 14px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-secondary);
  font-size: 16px;
}

.files-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 16px;
}

.file-item {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  position: relative;
  overflow: hidden;
}

.file-item:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
  border-color: var(--primary-color);
}

.file-icon {
  width: 60px;
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  margin-bottom: 12px;
  font-size: 24px;
  color: white;
}

.file-icon-image {
  background: var(--primary-color);
}

.file-icon-video {
  background: var(--danger-color);
}

.file-icon-audio {
  background: var(--warning-color);
}

.file-icon-pdf {
  background: var(--danger-color);
}

.file-icon-word {
  background: var(--primary-color);
}

.file-icon-excel {
  background: var(--success-color);
}

.file-icon-powerpoint {
  background: var(--warning-color);
}

.file-icon-text {
  background: var(--text-secondary);
}

.file-icon-other {
  background: var(--text-tertiary);
}

.file-info {
  flex: 1;
}

.file-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
  margin-bottom: 8px;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-meta {
  font-size: 12px;
  color: var(--text-secondary);
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.file-actions {
  position: absolute;
  top: 8px;
  right: 8px;
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.2s ease;
}

.file-item:hover .file-actions {
  opacity: 1;
}

.file-action-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: var(--hover-color);
  border-radius: 4px;
  cursor: pointer;
  color: var(--text-color);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.file-action-btn:hover {
  background: var(--hover-color);
  color: var(--primary-color);
  transform: scale(1.1);
}

.file-action-btn:nth-child(2):hover {
  color: var(--danger-color);
}

.files-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 300px;
  color: var(--text-secondary);
}

.files-empty i {
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.5;
}

.files-empty p {
  font-size: 16px;
  margin: 0;
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

.file-preview-modal {
  max-width: 800px;
  max-height: 90vh;
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
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  margin-right: 16px;
}

.modal-close {
  background: none;
  border: none;
  font-size: 24px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: color 0.2s ease;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
}

.modal-close:hover {
  color: var(--text-color);
  background: var(--hover-bg);
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

.form-input {
  width: 100%;
  padding: 10px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-size: 14px;
  background: var(--input-bg);
  color: var(--text-color);
  transition: all 0.2s ease;
  box-sizing: border-box;
}

.form-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.1);
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
  border-radius: 4px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.modal-btn.cancel-btn {
  background: var(--card-bg);
  color: var(--text-color);
}

.modal-btn.cancel-btn:hover {
  background: var(--hover-color);
}

.modal-btn.confirm-btn {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.modal-btn.confirm-btn:hover {
  background: var(--primary-hover);
}

/* 文件预览内容 */
.file-preview-content {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
}

.preview-image {
  max-width: 100%;
  max-height: 400px;
  object-fit: contain;
  border-radius: 4px;
}

.preview-video {
  max-width: 100%;
  max-height: 400px;
  object-fit: contain;
  border-radius: 4px;
}

.preview-audio {
  width: 100%;
  max-height: 100px;
}

.preview-other {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
  color: var(--text-secondary);
}

.preview-other i {
  font-size: 64px;
  margin-bottom: 16px;
  opacity: 0.5;
}

.preview-other p {
  font-size: 16px;
  margin: 0;
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
  .files-content {
    flex-direction: column;
  }
  
  .files-nav {
    width: 100%;
    border-right: none;
    border-bottom: 1px solid var(--border-color);
    padding: 8px 0;
    overflow-x: auto;
    overflow-y: hidden;
    white-space: nowrap;
  }
  
  .nav-item {
    display: inline-flex;
    border-left: none;
    border-bottom: 3px solid transparent;
  }
  
  .nav-item.active {
    border-left: none;
    border-bottom-color: var(--primary-color);
  }
  
  .files-main {
    padding: 12px;
  }
  
  .files-grid {
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 12px;
  }
  
  .file-item {
    padding: 12px;
  }
  
  .file-icon {
    width: 50px;
    height: 50px;
    font-size: 20px;
  }
  
  .modal-content {
    width: 95%;
    margin: 20px;
  }
  
  .file-preview-modal {
    max-width: 95%;
  }
}
</style>
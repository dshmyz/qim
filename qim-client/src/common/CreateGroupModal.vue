<template>
  <div v-if="visible" class="user-profile-modal" @click="handleClose">
    <div class="user-profile-content" @click.stop>
      <div class="user-profile-header">
        <h3>{{ title }}</h3>
        <button class="close-btn" @click="handleClose">×</button>
      </div>
      <div class="user-profile-body">
        <div class="profile-info">
          <!-- 名称输入 -->
          <div class="info-item">
            <label>{{ props && props.type === 'group' ? '群聊名称' : '讨论组名称' }}</label>
            <input type="text" v-model="name" class="profile-input" :placeholder="'请输入' + (props && props.type === 'group' ? '群聊' : '讨论组') + (props && props.type === 'group' ? '名称' : '名称（可选）')" />
          </div>
          
          <!-- 头像上传 -->
          <div class="info-item avatar-item">
            <label>头像（可选）</label>
            <div class="avatar-upload">
              <div class="avatar-preview" @click="triggerAvatarUpload">
                <img v-if="avatar" :src="avatar" alt="头像" />
                <div v-else class="avatar-placeholder">
                  <i class="fas fa-camera"></i>
                  <span>上传头像</span>
                </div>
              </div>
            </div>
          </div>
          
          <!-- 成员选择 -->
          <div class="info-item">
            <div class="member-selector-header">
              <label>{{ props && props.type === 'group' ? '群聊成员' : '讨论组成员' }}</label>
              <div class="selected-count" v-if="selectedMembers.length > 0">
                已选择 {{ selectedMembers.length }} 人
                <button class="clear-btn" @click="clearSelectedMembers">清空</button>
              </div>
            </div>
            <div class="member-search-box">
              <div class="search-input-wrapper">
                <i class="fas fa-search search-icon"></i>
                <input 
                  type="text" 
                  v-model="searchQuery" 
                  placeholder="搜索成员..." 
                  class="member-search-input" 
                  @input="handleSearchInput"
                />
                <button v-if="searchQuery" class="clear-search-btn" @click="clearSearch">×</button>
              </div>
            </div>
            <div class="member-selector" ref="memberSelectorRef">
              <div v-if="isLoading" class="loading-state">
                <div class="loading-spinner"></div>
                <p>加载中...</p>
              </div>
              <div v-else-if="filteredMembers.length === 0" class="empty-state">
                <p v-if="searchQuery">没有找到匹配的成员</p>
                <p v-else>暂无成员</p>
              </div>
              <div v-else class="member-list">
                <div 
                  v-for="employee in displayedMembers" 
                  :key="employee.id" 
                  class="member-item"
                  @click="toggleMember(employee)"
                >
                  <input 
                    type="checkbox" 
                    :checked="isMemberSelected(employee)" 
                    @change.stop="toggleMember(employee)"
                  />
                  <div class="member-info">
                    <div class="member-avatar">
                    <img :src="(employee.avatar && employee.avatar.startsWith('http')) ? employee.avatar : (employee.avatar ? serverUrl + employee.avatar : generateAvatar(employee.name))" :alt="employee.name" />
                  </div>
                    <div class="member-details">
                      <span class="member-name">{{ employee.name }}</span>
                      <span class="member-position" v-if="employee.position">{{ employee.position }}</span>
                    </div>
                  </div>
                </div>
                <div v-if="hasMoreMembers" class="load-more" @click="loadMoreMembers">
                  <button class="load-more-btn">加载更多</button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="user-profile-footer">
        <button class="cancel-btn" @click="handleClose">取消</button>
        <button class="save-btn" @click="createConversation" :disabled="((props && props.type === 'group') && !name) || selectedMembers.length === 0">
          创建 ({{ selectedMembers.length }})
        </button>
      </div>
    </div>
  </div>
  
  <!-- 头像上传 input -->
  <input type="file" ref="avatarInput" style="display: none" accept="image/*" @change="handleAvatarUpload" />
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { API_BASE_URL } from '../config'
import { generateAvatar } from '../utils/avatar'

// Props
const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  type: {
    type: String,
    default: 'group',
    validator: (value: string) => ['group', 'discussion'].includes(value)
  },
  title: {
    type: String,
    default: '创建群聊'
  },
  members: {
    type: Array,
    default: () => []
  }
})

// Emits
const emit = defineEmits(['close', 'created'])

// 服务器URL
const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

// 内部状态
const name = ref('')
const avatar = ref('')
const selectedMembers = ref<any[]>([])
const searchQuery = ref('')
const avatarInput = ref<HTMLInputElement | null>(null)
const memberSelectorRef = ref<HTMLElement | null>(null)

// 分页相关
const pageSize = 20
const currentPage = ref(1)
const isLoading = ref(false)
const searchTimeout = ref<number | null>(null)

// 过滤成员
const filteredMembers = computed(() => {
  if (!searchQuery.value) {
    return props.members
  }
  const query = searchQuery.value.toLowerCase()
  return props.members.filter(member => 
    member.name.toLowerCase().includes(query) ||
    (member.position && member.position.toLowerCase().includes(query)) ||
    (member.department && member.department.toLowerCase().includes(query))
  )
})

// 显示的成员（分页）
const displayedMembers = computed(() => {
  return filteredMembers.value.slice(0, currentPage.value * pageSize)
})

// 是否有更多成员
const hasMoreMembers = computed(() => {
  return displayedMembers.value.length < filteredMembers.value.length
})

// 监听搜索查询变化
watch(searchQuery, () => {
  // 重置分页
  currentPage.value = 1
})

// 监听visible变化，重置表单
watch(() => props.visible, (newValue) => {
  if (newValue) {
    resetForm()
  }
})

// 重置表单
const resetForm = () => {
  name.value = ''
  avatar.value = ''
  selectedMembers.value = []
  searchQuery.value = ''
  currentPage.value = 1
  isLoading.value = false
}

// 防抖搜索
const handleSearchInput = () => {
  if (searchTimeout.value) {
    clearTimeout(searchTimeout.value)
  }
  searchTimeout.value = window.setTimeout(() => {
    currentPage.value = 1
  }, 300)
}

// 清空搜索
const clearSearch = () => {
  searchQuery.value = ''
  currentPage.value = 1
}

// 加载更多成员
const loadMoreMembers = () => {
  if (hasMoreMembers.value) {
    currentPage.value++
  }
}

// 检查成员是否被选择
const isMemberSelected = (member: any) => {
  return selectedMembers.value.some(m => m.id === member.id)
}

// 切换成员选择状态
const toggleMember = (member: any) => {
  const index = selectedMembers.value.findIndex(m => m.id === member.id)
  if (index === -1) {
    selectedMembers.value.push(member)
  } else {
    selectedMembers.value.splice(index, 1)
  }
}

// 清空已选择的成员
const clearSelectedMembers = () => {
  selectedMembers.value = []
}

// 触发头像上传
const triggerAvatarUpload = () => {
  avatarInput.value?.click()
}

// 处理头像上传
const handleAvatarUpload = async (event: Event) => {
  const input = event.target as HTMLInputElement
  if (input.files && input.files.length > 0) {
    const file = input.files[0]
    
    // 先显示本地预览
    const reader = new FileReader()
    reader.onload = (e) => {
      avatar.value = e.target?.result as string
    }
    reader.readAsDataURL(file)
    
    const formData = new FormData()
    formData.append('file', file)
    
    try {
      const token = localStorage.getItem('token')
      const response = await fetch(`${serverUrl.value}/api/v1/upload`, {
        method: 'POST',
        headers: {
          ...(token ? { 'Authorization': `Bearer ${token}` } : {})
        },
        body: formData
      })
      
      if (response.ok) {
        const data = await response.json()
        if (data.code === 0) {
          // 上传成功后更新为服务器地址
          avatar.value = data.data.url
        } else {
          ElMessage.error('头像上传失败: ' + data.message)
          // 上传失败后清除预览
          avatar.value = ''
        }
      } else {
        ElMessage.error('头像上传失败: 服务器错误')
        // 上传失败后清除预览
        avatar.value = ''
      }
    } catch (error) {
      console.error('头像上传失败:', error)
      ElMessage.error('头像上传失败: 网络错误')
      // 上传失败后清除预览
      avatar.value = ''
    }
  }
}

// 处理关闭
const handleClose = () => {
  resetForm()
  emit('close')
}

// 创建会话
const createConversation = async () => {
  try {
    // 检查props是否存在
    if (!props) {
      ElMessage.error('组件参数错误，请重试')
      return
    }
    
    // 准备请求数据
    const requestData: any = {
      member_ids: selectedMembers.value.map((member: any) => parseInt(member.id))
    }
    
    // 为群聊添加名称
    if (props.type === 'group') {
      if (!name.value) {
        ElMessage.error('请输入群聊名称')
        return
      }
      requestData.name = name.value
    } else {
      // 为讨论组生成默认名称
      if (!name.value && selectedMembers.value.length > 0) {
        const memberNames = selectedMembers.value.slice(0, 3).map((member: any) => member.name).join('、')
        if (selectedMembers.value.length > 3) {
          requestData.name = `${memberNames} 等${selectedMembers.value.length}人`
        } else {
          requestData.name = `${memberNames}的讨论组`
        }
      } else if (name.value) {
        requestData.name = name.value
      }
    }
    
    // 只有当avatar有值且不是DataURL时才添加到请求数据中
    if (avatar.value && !avatar.value.startsWith('data:')) {
      requestData.avatar = avatar.value
    }
    
    const token = localStorage.getItem('token')
    const endpoint = props.type === 'group' ? '/api/v1/conversations/group' : '/api/v1/conversations/discussion'
    
    const response = await fetch(`${serverUrl.value}${endpoint}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      },
      body: JSON.stringify(requestData)
    })
    
    if (response.ok) {
      const data = await response.json()
      if (data.code === 0) {
        // 关闭弹窗
        handleClose()
        // 触发创建成功事件
        emit('created')
        // 显示成功提示
        ElMessage.success(`${props.type === 'group' ? '群聊' : '讨论组'}创建成功`)
      } else {
        ElMessage.error('创建失败: ' + data.message)
      }
    } else {
      ElMessage.error('创建失败: 服务器错误')
    }
  } catch (error) {
    console.error('创建失败:', error)
    ElMessage.error('创建失败，请重试')
  }
}
</script>

<style scoped>
/* 模态框样式 */
.user-profile-modal {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
  backdrop-filter: blur(2px);
}

.user-profile-content {
  background-color: var(--card-bg, white);
  border-radius: 12px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
  width: 90%;
  max-width: 520px;
  max-height: 85vh;
  overflow-y: auto;
  position: relative;
}

.user-profile-header {
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-color, #f0f0f0);
  display: flex;
  justify-content: space-between;
  align-items: center;
  background-color: var(--secondary-color, #f9fafb);
  border-radius: 12px 12px 0 0;
}

.user-profile-header h3 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--text-color, #1f2937);
}

.user-profile-header .close-btn {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: var(--text-secondary, #6b7280);
  padding: 0;
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
}

.user-profile-header .close-btn:hover {
  background-color: var(--hover-color, #f3f4f6);
  color: var(--text-color, #374151);
}

.user-profile-body {
  padding: 24px;
}

/* 表单样式 */
.profile-info {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.member-selector-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.info-item label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color, #374151);
  display: flex;
  align-items: center;
  gap: 6px;
}

.info-item label::after {
  content: '*';
  color: #ef4444;
  font-size: 14px;
}

/* 讨论组名称标签不需要星号 */
.info-item:has(input[placeholder*="讨论组名称（可选）"]) label::after {
  content: '';
}

/* 头像标签不需要星号 */
.info-item.avatar-item label::after {
  content: '';
}

/* 成员选择器的标签样式 */
.member-selector-header label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color, #374151);
  display: flex;
  align-items: center;
  gap: 6px;
}

.member-selector-header label::after {
  content: '*';
  color: #ef4444;
  font-size: 14px;
}

.selected-count {
  font-size: 12px;
  color: var(--text-secondary, #6b7280);
  display: flex;
  align-items: center;
  gap: 8px;
}

.clear-btn {
  background: none;
  border: none;
  color: var(--primary-color, #3b82f6);
  font-size: 12px;
  cursor: pointer;
  padding: 2px 6px;
  border-radius: 4px;
}

.clear-btn:hover {
  background-color: var(--primary-light, #f0f7ff);
  text-decoration: underline;
}

.profile-input {
  width: 100%;
  padding: 12px 16px;
  border: 1px solid var(--border-color, #d1d5db);
  border-radius: 8px;
  font-size: 14px;
  box-sizing: border-box;
  background-color: var(--secondary-color, #f9fafb);
  color: var(--text-color, #1f2937);
}

.profile-input:focus {
  outline: none;
  border-color: var(--primary-color, #3b82f6);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  background-color: var(--card-bg, white);
}

.profile-input::placeholder {
  color: var(--text-secondary, #9ca3af);
  font-size: 14px;
}

/* 搜索框样式 */
.member-search-box {
  margin-bottom: 12px;
}

.search-input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 12px;
  color: var(--text-secondary, #9ca3af);
  font-size: 14px;
}

.member-search-input {
  width: 100%;
  padding: 10px 14px 10px 36px;
  border: 1px solid var(--border-color, #d1d5db);
  border-radius: 6px;
  font-size: 14px;
  box-sizing: border-box;
  background-color: var(--secondary-color, #f9fafb);
  color: var(--text-color, #1f2937);
}

.member-search-input:focus {
  outline: none;
  border-color: var(--primary-color, #3b82f6);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  background-color: var(--card-bg, white);
}

.clear-search-btn {
  position: absolute;
  right: 12px;
  background: none;
  border: none;
  color: var(--text-secondary, #9ca3af);
  font-size: 18px;
  cursor: pointer;
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
}

.clear-search-btn:hover {
  background-color: var(--hover-color, #f3f4f6);
  color: var(--text-color, #374151);
}

/* 头像上传样式 */
.avatar-upload {
  margin-top: 8px;
}

.avatar-preview {
  width: 80px;
  height: 80px;
  border-radius: 10px;
  border: 2px dashed var(--border-color, #d1d5db);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  overflow: hidden;
  background-color: var(--secondary-color, #f9fafb);
}

.avatar-preview:hover {
  border-color: var(--primary-color, #3b82f6);
  background-color: var(--primary-light, #f0f7ff);
}

.avatar-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary, #9ca3af);
}

.avatar-placeholder i {
  font-size: 20px;
  margin-bottom: 6px;
}

.avatar-placeholder span {
  font-size: 12px;
  font-weight: 400;
}

.avatar-preview:hover .avatar-placeholder {
  color: var(--primary-color, #3b82f6);
}

/* 成员选择样式 */
.member-selector {
  max-height: 300px;
  overflow-y: auto;
  border: 1px solid var(--border-color, #d1d5db);
  border-radius: 8px;
  padding: 8px;
  background-color: var(--secondary-color, #f9fafb);
}

.member-selector:hover {
  border-color: var(--text-secondary, #9ca3af);
}

.member-selector::-webkit-scrollbar {
  width: 6px;
}

.member-selector::-webkit-scrollbar-track {
  background: var(--secondary-color, #f1f1f1);
  border-radius: 3px;
}

.member-selector::-webkit-scrollbar-thumb {
  background: var(--border-color, #d1d5db);
  border-radius: 3px;
}

.member-selector::-webkit-scrollbar-thumb:hover {
  background: var(--text-secondary, #9ca3af);
}

.member-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.member-item {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  border-radius: 6px;
  cursor: pointer;
  background-color: var(--card-bg, white);
  border: 1px solid transparent;
}

.member-item:hover {
  background-color: var(--hover-color, #f3f4f6);
  border-color: var(--border-color, #e5e7eb);
}

.member-item input[type="checkbox"] {
  margin-right: 12px;
  width: 18px;
  height: 18px;
  cursor: pointer;
  accent-color: var(--primary-color, #3b82f6);
}

.member-info {
  display: flex;
  align-items: center;
  flex: 1;
}

.member-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  overflow: hidden;
  margin-right: 12px;
  border: 2px solid var(--border-color, #e5e7eb);
}

.member-item:hover .member-avatar {
  border-color: var(--primary-color, #3b82f6);
}

.member-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.member-details {
  flex: 1;
}

.member-name {
  display: block;
  font-size: 14px;
  color: var(--text-color, #374151);
  font-weight: 400;
  margin-bottom: 2px;
}

.member-position {
  display: block;
  font-size: 12px;
  color: var(--text-secondary, #6b7280);
}

/* 加载状态 */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: var(--text-secondary, #6b7280);
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--secondary-color, #f3f4f6);
  border-top: 3px solid var(--primary-color, #3b82f6);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 12px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* 空状态样式 */
.empty-state {
  text-align: center;
  padding: 32px 24px;
  color: var(--text-secondary, #9ca3af);
  background-color: var(--card-bg, white);
  border-radius: 6px;
  border: 1px dashed var(--border-color, #d1d5db);
}

.empty-state p {
  margin: 0;
  font-size: 14px;
  font-weight: 400;
}

/* 加载更多 */
.load-more {
  text-align: center;
  padding: 16px;
  margin-top: 8px;
}

.load-more-btn {
  background-color: var(--card-bg, white);
  border: 1px solid var(--border-color, #d1d5db);
  border-radius: 6px;
  padding: 8px 16px;
  font-size: 14px;
  color: var(--text-color, #374151);
  cursor: pointer;
}

.load-more-btn:hover {
  border-color: var(--primary-color, #3b82f6);
  color: var(--primary-color, #3b82f6);
  background-color: var(--primary-light, #f0f7ff);
}

/* 底部按钮样式 */
.user-profile-footer {
  padding: 20px 24px;
  border-top: 1px solid var(--border-color, #f0f0f0);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  background-color: var(--secondary-color, #f9fafb);
  border-radius: 0 0 12px 12px;
}

.cancel-btn,
.save-btn {
  padding: 10px 20px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  border: 1px solid transparent;
  min-width: 80px;
  text-align: center;
}

.cancel-btn {
  background-color: var(--card-bg, white);
  border-color: var(--border-color, #d1d5db);
  color: var(--text-color, #374151);
}

.cancel-btn:hover {
  border-color: var(--text-secondary, #9ca3af);
  color: var(--text-color, #1f2937);
  background-color: var(--secondary-color, #f9fafb);
}

.save-btn {
  background-color: var(--primary-color, #3b82f6);
  color: white;
  border-color: var(--primary-color, #3b82f6);
  box-shadow: 0 2px 4px rgba(59, 130, 246, 0.2);
}

.save-btn:hover {
  background-color: var(--active-color, #2563eb);
  border-color: var(--active-color, #2563eb);
  box-shadow: 0 4px 8px rgba(59, 130, 246, 0.3);
}

.save-btn:active {
  box-shadow: 0 2px 4px rgba(59, 130, 246, 0.2);
}

.save-btn:disabled {
  background-color: var(--secondary-color, #f3f4f6);
  color: var(--text-secondary, #9ca3af);
  border-color: var(--border-color, #d1d5db);
  cursor: not-allowed;
  box-shadow: none;
}

.save-btn:disabled:hover {
  background-color: var(--secondary-color, #f3f4f6);
  border-color: var(--border-color, #d1d5db);
  color: var(--text-secondary, #9ca3af);
}

/* 响应式设计 */
@media (max-width: 480px) {
  .user-profile-content {
    width: 95%;
    max-height: 90vh;
  }
  
  .user-profile-header {
    padding: 16px 20px;
  }
  
  .user-profile-header h3 {
    font-size: 18px;
  }
  
  .user-profile-body {
    padding: 20px;
  }
  
  .user-profile-footer {
    padding: 16px 20px;
  }
  
  .avatar-preview {
    width: 80px;
    height: 80px;
  }
  
  .member-selector {
    max-height: 250px;
  }
}
</style>
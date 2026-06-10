<template>
  <div class="right-content">
    <div class="panel-header">
      <div class="header-left-group">
        <ToggleSidebarBtn
          icon="fas fa-compress"
          title="收起侧边栏"
          @click="$emit('toggleSidebar')"
        />
        <h2>用户资料</h2>
      </div>
    </div>
    <div class="user-profile-container">
      <div class="user-profile-header-bg"></div>
      
      <div class="user-profile-card">
        <div class="user-profile-avatar-section">
          <div class="user-avatar-container">
            <Avatar
              :src="detail.avatar"
              :name="detail.name"
              :server-url="serverUrl"
              :alt="detail.name"
              size="xl"
              class="user-avatar"
            />
          </div>
          <div class="user-basic-info">
            <h2 class="user-full-name">{{ detail.name }}</h2>
            <p class="user-department">{{ detail.department || '暂无部门' }}</p>
            <p v-if="detail.signature" class="user-signature">{{ detail.signature }}</p>
          </div>
        </div>
        
        <div class="user-info-sections">
          <div class="info-section">
            <div class="section-title">
              <i class="fas fa-user-circle"></i>
              <h3>基本信息</h3>
            </div>
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">姓名</span>
                <span class="info-value">{{ detail.name }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">账号</span>
                <span class="info-value">{{ detail.username || '暂无' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">邮箱</span>
                <span class="info-value">{{ detail.email || '暂无' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">手机</span>
                <span class="info-value">{{ detail.mobile || '暂无' }}</span>
              </div>
            </div>
          </div>
          
          <div class="info-section">
            <div class="section-title">
              <i class="fas fa-briefcase"></i>
              <h3>工作信息</h3>
            </div>
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">部门</span>
                <span class="info-value">{{ detail.department || '暂无' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">职位</span>
                <span class="info-value">{{ detail.position || '暂无' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">IP</span>
                <span class="info-value">{{ detail.ip || '暂无' }}</span>
              </div>
            </div>
          </div>
        </div>
        
        <div class="user-action-buttons">
          <button v-if="detail.type !== 'bot_assistant'" class="action-btn primary" @click="$emit('privateChat', detail)">
            <i class="fas fa-comment"></i>
            <span>发起私聊</span>
          </button>
          <button class="action-btn secondary" @click="$emit('showProfile', detail)">
            <i class="fas fa-id-card"></i>
            <span>详细资料</span>
          </button>
          <button v-if="isCurrentUser" class="action-btn secondary" @click="$emit('open-avatar-settings')">
            <i class="fas fa-user-astronaut"></i>
            <span>分身设置</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { getCurrentUser } from '../../utils/user'
import { request } from '../../composables/useRequest'
import Avatar from '../shared/Avatar.vue'
import ToggleSidebarBtn from '../shared/ToggleSidebarBtn.vue'

interface User {
  id: string | number
  name: string
  username?: string
  email?: string
  mobile?: string
  department?: string
  position?: string
  ip?: string
  avatar?: string
  signature?: string
}

interface Props {
  user: User
  serverUrl: string
}

const props = defineProps<Props>()

const detail = ref<any>({ ...props.user })

const fetchUserDetail = async () => {
  detail.value = { ...props.user }
  try {
    const response = await request(`/api/v1/users/${props.user.id}`)
    if (response.code === 0 && response.data) {
      detail.value = {
        ...props.user,
        ...response.data,
        mobile: response.data.mobile || response.data.phone || props.user.mobile || '',
        signature: response.data.signature || '',
        ip: response.data.ip || props.user.ip || '',
      }
    }
  } catch {
    // fallback to prop data
  }
}

watch(() => props.user, fetchUserDetail, { immediate: true })

const isCurrentUser = computed(() => {
  const currentUser = getCurrentUser()
  if (!currentUser || !currentUser.id) return false
  return String(currentUser.id) === String(props.user.id)
})

defineEmits<{
  'toggleSidebar': []
  'privateChat': [user: User]
  'showProfile': [user: User]
  'open-avatar-settings': []
}>()
</script>

<style scoped>
.right-content {
  flex: 1;
  background: var(--right-content-bg, #f5f5f5);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-header {
  padding: 0 20px;
  height: 56px;
  background: var(--right-content-header-bg, #fff);
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  flex-shrink: 0;
}

.panel-header h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color, #333);
}

.header-left-group {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-profile-container {
  position: relative;
  padding: 16px;
  margin: 10px 5px;
  flex: 1;
  overflow-y: auto;
}

.user-profile-header-bg {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 80px;
  background: linear-gradient(135deg, var(--primary-light), var(--active-color));
  border-radius: 8px 8px 0 0;
  z-index: 1;
}

.user-profile-card {
  position: relative;
  background: var(--card-bg);
  border-radius: 10px;
  padding: 16px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  z-index: 2;
  margin-top: 40px;
  animation: cardSlideIn 0.4s ease-out;
}

@keyframes cardSlideIn {
  from {
    transform: translateY(20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.user-profile-avatar-section {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 0;
}

.user-avatar-container {
  position: relative;
  margin-right: 16px;
}

.user-avatar {
  width: 56px;
  height: 56px;
  border-radius: 10px;
  object-fit: cover;
  border: 3px solid white;
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
  transition: transform 0.3s ease;
}

.user-avatar:hover {
  transform: scale(1.05);
}

.user-basic-info {
  flex: 1;
}

.user-full-name {
  margin: 0 0 4px 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
  letter-spacing: 0.3px;
}

.user-department {
  margin: 0;
  font-size: 12px;
  color: var(--text-secondary);
}

.user-signature {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--text-secondary);
}

.user-info-sections {
  margin-bottom: 20px;
}

.info-section {
  border-radius: 8px;
  padding: 16px;
}

.section-title {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 0;
}

.section-title i {
  font-size: 14px;
  color: var(--primary-color);
  margin-right: 8px;
}

.section-title h3 {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-color);
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.info-label {
  font-size: 12px;
  color: #64748b;
  font-weight: 600;
}

.info-value {
  font-size: 14px;
  color: var(--text-color);
  font-weight: 500;
}

.user-action-buttons {
  display: flex;
  gap: 8px;
  justify-content: center;
  padding-top: 12px;
  margin-bottom: 16px;
}

.action-btn {
  padding: 8px 16px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 5px;
  min-width: 90px;
  justify-content: center;
}

.action-btn.primary {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
}

.action-btn.primary:hover {
  background: var(--active-color);
  border-color: var(--active-color);
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(102, 126, 234, 0.4);
}

.action-btn.secondary {
  background: var(--input-bg);
  border-color: var(--border-color);
  color: var(--text-color);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.action-btn.secondary:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.2);
}

@media (max-width: 768px) {
  .user-profile-container {
    padding: 10px;
  }
  
  .user-profile-card {
    padding: 20px;
    margin-top: 40px;
  }
  
  .info-grid {
    grid-template-columns: 1fr;
  }
  
  .user-action-buttons {
    flex-direction: column;
  }
  
  .action-btn {
    width: 100%;
  }
}
</style>

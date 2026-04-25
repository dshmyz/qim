<template>
  <div class="right-content">
    <div class="right-content-header">
      <h2>用户资料</h2>
      <button class="toggle-sidebar-btn" @click="$emit('toggleSidebar')">
        <i class="fas fa-compress"></i>
      </button>
    </div>
    <div class="user-profile-container">
      <div class="user-profile-header-bg"></div>
      
      <div class="user-profile-card">
        <div class="user-profile-avatar-section">
          <div class="user-avatar-container">
            <img
              :src="getAvatarUrl(user.avatar)"
              :alt="user.name"
              class="user-avatar"
            />
            <div class="online-status-indicator"></div>
          </div>
          <div class="user-basic-info">
            <h2 class="user-full-name">{{ user.name }}</h2>
            <p class="user-department">{{ user.department || '暂无部门' }}</p>
            <p class="user-position">{{ user.position || '暂无职位' }}</p>
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
                <span class="info-value">{{ user.name }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">账号</span>
                <span class="info-value">{{ user.username || '暂无' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">邮箱</span>
                <span class="info-value">{{ user.email || '暂无' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">手机</span>
                <span class="info-value">{{ user.mobile || '暂无' }}</span>
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
                <span class="info-value">{{ user.department || '暂无' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">IP</span>
                <span class="info-value">{{ user.ip || '暂无' }}</span>
              </div>
            </div>
          </div>
        </div>
        
        <div class="user-action-buttons">
          <button class="action-btn primary" @click="$emit('privateChat', user)">
            <i class="fas fa-comment"></i>
            <span>发起私聊</span>
          </button>
          <button class="action-btn secondary" @click="$emit('showProfile', user)">
            <i class="fas fa-id-card"></i>
            <span>详细资料</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
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
}

interface Props {
  user: User
  serverUrl: string
  getAvatarUrl: (avatar: string | undefined) => string
}

defineProps<Props>()

defineEmits<{
  'toggleSidebar': []
  'privateChat': [user: User]
  'showProfile': [user: User]
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

.right-content-header {
  padding: 16px 20px;
  background: var(--right-content-header-bg, #fff);
  height: 72px;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.right-content-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 500;
  color: var(--text-color, #333);
}

.toggle-sidebar-btn {
  background: none;
  border: none;
  cursor: pointer;
  padding: 8px;
  color: var(--text-color, #333);
}

.user-profile-container {
  flex: 1;
  overflow-y: auto;
  position: relative;
}

.user-profile-header-bg {
  height: 120px;
  background: linear-gradient(135deg, var(--primary-color, #409eff), #67c23a);
}

.user-profile-card {
  background: var(--card-bg, #fff);
  border-radius: 12px;
  margin: -60px 20px 20px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.user-profile-avatar-section {
  text-align: center;
  margin-bottom: 24px;
}

.user-avatar-container {
  position: relative;
  display: inline-block;
  margin-bottom: 16px;
}

.user-avatar {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  border: 4px solid var(--card-bg, #fff);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.online-status-indicator {
  position: absolute;
  bottom: 4px;
  right: 4px;
  width: 16px;
  height: 16px;
  background: #67c23a;
  border-radius: 50%;
  border: 3px solid var(--card-bg, #fff);
}

.user-full-name {
  margin: 0 0 8px 0;
  font-size: 24px;
  color: var(--text-color, #333);
}

.user-department,
.user-position {
  margin: 0 0 4px 0;
  color: var(--text-secondary, #999);
}

.user-info-sections {
  display: flex;
  flex-direction: column;
  gap: 20px;
  margin-bottom: 24px;
}

.info-section .section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  color: var(--text-color, #333);
}

.section-title h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-label {
  font-size: 12px;
  color: var(--text-secondary, #999);
}

.info-value {
  font-size: 14px;
  color: var(--text-color, #333);
}

.user-action-buttons {
  display: flex;
  gap: 12px;
}

.action-btn {
  flex: 1;
  padding: 12px;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 14px;
}

.action-btn.primary {
  background: var(--primary-color, #409eff);
  color: white;
}

.action-btn.secondary {
  background: var(--secondary-bg, #f0f0f0);
  color: var(--text-color, #333);
}
</style>

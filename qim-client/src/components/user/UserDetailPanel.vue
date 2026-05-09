<template>
  <div class="right-content">
    <div class="right-content-header">
      <div class="header-left-group">
        <button class="toggle-sidebar-btn" @click="$emit('toggleSidebar')">
          <i class="fas fa-compress"></i>
        </button>
        <h2>用户资料</h2>
      </div>
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
          <button class="action-btn secondary" @click="$emit('open-avatar-settings')">
            <i class="fas fa-user-astronaut"></i>
            <span>分身设置</span>
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

.header-left-group {
  display: flex;
  align-items: center;
  gap: 12px;
}

.toggle-sidebar-btn {
  background: none;
  border: none;
  cursor: pointer;
  padding: 8px;
  color: var(--text-color, #333);
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
  /* height: 120px; */
  /* background: linear-gradient(135deg, var(--primary-color, #409eff), #67c23a); */
}

.user-profile-card {
  /* background: var(--card-bg, #fff);
  border-radius: 12px;
  margin: -60px 20px 20px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1); */
    margin: -60px 20px 20px;

  position: relative;
  background: var(--card-bg);
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  /* border: 1px solid var(--border-color); */
  z-index: 2;
  margin-top: 40px;
  animation: cardSlideIn 0.4s ease-out;
}

.user-profile-avatar-section {
  /* text-align: center; */
  /* margin-bottom: 24px; */
   display: flex;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border-color);
   flex-direction: column;
    text-align: center;
    gap: 15px;
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
  /* display: flex;
  flex-direction: column;
  gap: 20px;
  margin-bottom: 24px; */
}
.user-info-sections {
  margin-bottom: 20px;
}

.info-section {
  margin-bottom: 16px;
  /* background: var(--list-bg); */
  border-radius: 8px;
  padding: 16px;
  /* border: 1px solid var(--border-color); */
  transition: all 0.3s ease;
}

.info-section:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  transform: translateY(-2px);
}

.section-title {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-color);
}

.section-title i {
  font-size: 16px;
  color: var(--primary-color);
  margin-right: 8px;
}

.section-title h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color);
}


.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.info-label {
  font-size: 11px;
  color: #64748b;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.info-value {
  font-size: 13px;
  color: var(--text-color);
  font-weight: 500;
  padding: 5px 8px;
  background: var(--input-bg);
  border-radius: 4px;
  border: 1px solid var(--border-color);
  transition: all 0.2s ease;
}

.info-value:hover {
  border-color: var(--primary-color);
  box-shadow: 0 2px 8px rgba(102, 126, 234, 0.15);
}

.user-action-buttons {
  display: flex;
  gap: 10px;
  justify-content: center;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
}

.action-btn {
  padding: 10px 20px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 100px;
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
  background: var(--primary-light);
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

/* 响应式设计 */
@media (max-width: 768px) {
  .user-profile-container {
    padding: 10px;
  }
  
  .user-profile-card {
    padding: 20px;
    margin-top: 40px;
  }
  
  .user-avatar-container {
    margin-right: 0;
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

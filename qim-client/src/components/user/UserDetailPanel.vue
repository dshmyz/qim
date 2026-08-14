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
      <div class="user-profile-card">
        <!-- 头像 + 基本信息 -->
        <div class="user-info-header">
          <div class="user-avatar-wrapper">
            <Avatar
              :src="detail.avatar"
              :name="detail.name"
              :server-url="serverUrl"
              :alt="detail.name"
              size="xl"
              shape="circle"
              class="user-avatar"
            />
          </div>
          <h2 class="user-name">
            {{ detail.name }}
            <span v-if="roleLabel" class="user-role-tag">{{ roleLabel.text }}</span>
          </h2>
          <div class="user-sub">
            <span class="user-department">{{ detail.department || '暂无部门' }}<template v-if="detail.position"> · {{ detail.position }}</template></span>
            <span v-if="statusLabel" class="user-status-badge" :class="{ 'is-online': statusInfo?.status === 'online' }">
              <span class="status-dot" :class="statusInfo?.status || 'offline'"></span>{{ statusLabel }}
            </span>
          </div>
        </div>

        <!-- 签名 -->
        <div v-if="detail.signature" class="signature-bar">
          <span>{{ detail.signature }}</span>
        </div>

        <!-- 信息列表 -->
        <div class="info-section">
          <div class="section-title">基本信息</div>
          <div class="info-list">
            <div class="info-row">
              <span class="info-row-icon"><i class="fas fa-user"></i></span>
              <div class="info-row-text">
                <span class="info-row-label">账号</span>
                <span class="info-row-value">{{ detail.username || '暂无' }}</span>
              </div>
            </div>
            <div class="info-row">
              <span class="info-row-icon"><i class="fas fa-envelope"></i></span>
              <div class="info-row-text">
                <span class="info-row-label">邮箱</span>
                <span class="info-row-value">{{ detail.email || '暂无' }}</span>
              </div>
            </div>
            <div class="info-row">
              <span class="info-row-icon"><i class="fas fa-mobile-alt"></i></span>
              <div class="info-row-text">
                <span class="info-row-label">手机</span>
                <span class="info-row-value">{{ detail.mobile || '暂无' }}</span>
              </div>
            </div>
            <div class="info-row">
              <span class="info-row-icon"><i class="fas fa-building"></i></span>
              <div class="info-row-text">
                <span class="info-row-label">部门</span>
                <span class="info-row-value">{{ detail.department || '暂无' }}</span>
              </div>
            </div>
            <div class="info-row">
              <span class="info-row-icon"><i class="fas fa-briefcase"></i></span>
              <div class="info-row-text">
                <span class="info-row-label">职位</span>
                <span class="info-row-value">{{ detail.position || '暂无' }}</span>
              </div>
            </div>
            <div v-if="detail.ip" class="info-row">
              <span class="info-row-icon"><i class="fas fa-globe"></i></span>
              <div class="info-row-text">
                <span class="info-row-label">IP</span>
                <span class="info-row-value muted">{{ detail.ip }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="action-bar">
          <button v-if="detail.type !== 'bot'" class="action-btn primary" @click="$emit('privateChat', detail)">
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
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { getCurrentUser } from '../../utils/user'
import { request } from '../../composables/useRequest'
import { useUserStatus } from '../../composables/useUserStatus'
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

// 用户在线状态
const { subscribeUserStatus, unsubscribeUserStatus, getUserStatus, formatLastOnline } = useUserStatus()

// 订阅用户状态（切换用户时更新）
watch(() => props.user?.id, (newId, oldId) => {
  if (oldId) unsubscribeUserStatus(Number(oldId))
  if (newId) subscribeUserStatus(Number(newId))
}, { immediate: true })

onUnmounted(() => {
  if (props.user?.id) unsubscribeUserStatus(Number(props.user.id))
})

const statusInfo = computed(() => getUserStatus(Number(props.user?.id)))

const avatarBadge = computed(() => {
  if (detail.value.type === 'bot' || detail.value.type === 'system') {
    return { type: 'icon' as const, icon: 'fas fa-robot', title: 'AI 助手', color: 'var(--primary-color, #1890ff)' }
  }
  const s = statusInfo.value?.status
  if (s) {
    return { type: 'status' as const, status: s, title: s === 'online' ? '在线' : s === 'busy' ? '忙碌' : '离线' }
  }
  return null
})

const statusLabel = computed(() => {
  if (detail.value.type === 'bot' || detail.value.type === 'system') return 'AI 助手'
  const s = statusInfo.value?.status
  if (s === 'online') return '在线'
  if (s === 'busy') return '忙碌'
  if (s === 'offline') {
    const lastOnline = statusInfo.value?.lastOnline
    if (lastOnline) return formatLastOnline(lastOnline) || '离线'
    return '离线'
  }
  return ''
})

const roleLabel = computed(() => {
  const r = detail.value.role
  if (r === 'owner') return { text: '群主', cls: 'owner' }
  if (r === 'admin') return { text: '管理员', cls: 'admin' }
  return null
})

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
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--text-color, #333);
}

.header-left-group {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-profile-container {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
}

.user-profile-card {
  padding: 0;
}

/* ---- 头像 + 基本信息区 ---- */
.user-info-header {
  padding: 28px 20px 20px;
  text-align: center;
  background: var(--card-bg, #fff);
}

.user-avatar-wrapper {
  position: relative;
  display: inline-block;
  margin-bottom: 14px;
}

.user-avatar {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  border: 3px solid var(--border-color, rgba(0,0,0,0.08));
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.user-name {
  margin: 0 0 4px;
  font-size: var(--font-size-lg);
  font-weight: 700;
  color: var(--text-color, #333);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.user-role-tag {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: var(--font-size-xxxs);
  font-weight: 600;
  background: var(--primary-color);
  color: #fff;
  opacity: 0.85;
}

.user-sub {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin-top: 4px;
}

.user-department {
  margin: 0;
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
}

.user-status-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary);
  background: var(--hover-color, rgba(0,0,0,0.04));
  padding: 2px 10px;
  border-radius: 20px;
}

.user-status-badge.is-online {
  color: #52c41a;
}

.status-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: #d9d9d9;
  flex-shrink: 0;
}

.status-dot.online { background-color: #52c41a; }
.status-dot.busy { background-color: #ff4d4f; }
.status-dot.offline { background-color: #d9d9d9; }

/* ---- 签名区 ---- */
.signature-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 20px;
  background: var(--card-bg, #fff);
  border-bottom: 1px solid var(--border-color, rgba(0,0,0,0.06));
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
  line-height: 1.5;
}

/* ---- 信息列表 ---- */
.info-section {
  padding: 16px 20px;
}

.section-title {
  font-size: var(--font-size-xxs);
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.8px;
  margin-bottom: 12px;
}

.info-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px 8px;
}

.info-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 8px;
  min-width: 0;
  transition: background-color 0.15s;
}

.info-row:hover {
  background: var(--hover-color, rgba(0,0,0,0.03));
}

.info-row-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 8px;
  background: var(--hover-color, rgba(0,0,0,0.04));
  color: var(--text-secondary);
  font-size: var(--font-size-xxs);
  flex-shrink: 0;
}

.info-row-text {
  flex: 1;
  min-width: 0;
}

.info-row-label {
  display: block;
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary);
  margin-bottom: 1px;
}

.info-row-value {
  display: block;
  font-size: var(--font-size-xs);
  color: var(--text-color);
  font-weight: 500;
  word-break: break-all;
  line-height: 1.4;
}

.info-row-value.muted {
  color: var(--text-secondary);
  font-weight: 400;
  font-size: var(--font-size-xxs);
}

/* ---- 操作按钮 ---- */
.action-bar {
  display: flex;
  gap: 8px;
  padding: 16px 20px;
  border-top: 1px solid var(--border-color, rgba(0,0,0,0.06));
}

.action-btn {
  flex: 1;
  padding: 10px 12px;
  border: none;
  border-radius: 10px;
  font-size: var(--font-size-xs);
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.action-btn.primary {
  background: var(--primary-color);
  color: #fff;
  box-shadow: 0 2px 8px rgba(102, 126, 234, 0.25);
}

.action-btn.primary:hover {
  background: var(--active-color);
  box-shadow: 0 4px 14px rgba(102, 126, 234, 0.35);
  transform: translateY(-1px);
}

.action-btn.secondary {
  background: var(--hover-color, rgba(0,0,0,0.04));
  color: var(--text-color);
}

.action-btn.secondary:hover {
  background: var(--border-color, rgba(0,0,0,0.08));
  transform: translateY(-1px);
}

@media (max-width: 768px) {
  .info-row {
    padding: 8px 10px;
  }

  .action-bar {
    flex-direction: column;
    gap: 6px;
  }
}
</style>

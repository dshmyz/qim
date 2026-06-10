<template>
  <!-- 关于对话框 -->
  <div v-if="showAboutDialog" class="about-dialog-overlay" @click="$emit('closeAbout')">
    <div class="about-dialog" @click.stop>
      <div class="about-dialog-header">
        <h3>关于</h3>
        <button class="about-dialog-close" @click="$emit('closeAbout')">×</button>
      </div>
      <div class="about-dialog-content">
        <div class="about-dialog-logo">
          <AppLogo size="extraLarge" />
        </div>
        <h2>{{ productFullName }}</h2>
        <p class="version">版本: {{ appVersion }}</p>
        <p class="date">发布日期: 2026-04-11</p>
        <div class="credits-section">
          <p class="credit-item">
            <i class="fas fa-pencil-ruler"></i>
            产品设计: Huang Qun
          </p>
          <p class="credit-item">
            <i class="fas fa-robot"></i>
            代码实现: AI Assistant
          </p>
        </div>
        <p class="contact">
          <i class="fas fa-envelope"></i>
          联系邮箱: <a href="mailto:huangqun@buaa.edu.cn">huangqun@buaa.edu.cn</a>
        </p>
        <p class="copyright">{{ copyrightText }}</p>
        <p class="description">一款现代化的即时通讯应用，致力于提供简洁、高效、智能化的沟通体验，让团队协作更顺畅。</p>
      </div>
      <div class="about-dialog-footer">
        <button class="about-dialog-button secondary" @click="$emit('openFeedback')">
          <i class="fas fa-comments"></i> 意见反馈
        </button>
        <button class="about-dialog-button" @click="$emit('closeAbout')">确定</button>
      </div>
    </div>
  </div>

  <!-- 退出登录确认对话框 -->
  <div v-if="showLogoutDialog" class="logout-dialog-overlay" @click="$emit('cancelLogout')">
    <div class="logout-dialog" @click.stop>
      <div class="logout-dialog-header">
        <h3>退出登录</h3>
        <button class="logout-dialog-close" @click="$emit('cancelLogout')">×</button>
      </div>
      <div class="logout-dialog-content">
        <p class="logout-dialog-message">确定要退出登录吗？</p>
      </div>
      <div class="logout-dialog-footer">
        <button class="logout-dialog-button cancel-button" @click="$emit('cancelLogout')">取消</button>
        <button class="logout-dialog-button confirm-button" @click="$emit('confirmLogout')">确定</button>
      </div>
    </div>
  </div>

  <!-- 检查更新对话框 -->
  <div v-if="showUpdateDialog" class="update-dialog-overlay" @click="!forceUpdate && $emit('closeUpdate')">
    <div class="update-dialog" @click.stop>
      <div class="update-dialog-header">
        <h3>{{ forceUpdate ? '版本更新' : '检查更新' }}</h3>
        <button v-if="!forceUpdate" class="update-dialog-close" @click="$emit('closeUpdate')">×</button>
      </div>
      <div class="update-dialog-content">
        <div v-if="isCheckingUpdate" class="update-loading">
          <div class="loading-spinner"></div>
          <p class="loading-text">正在检查更新...</p>
        </div>
        <div v-else-if="isDownloading" class="update-downloading">
          <div class="download-icon"><i class="fas fa-download"></i></div>
          <p class="download-text">正在下载更新...</p>
          <div class="download-progress">
            <div class="progress-bar">
              <div class="progress-fill" :style="{ width: downloadProgress + '%' }"></div>
            </div>
            <p class="progress-text">{{ downloadProgress }}%</p>
          </div>
        </div>
        <div v-else class="update-result">
          <div class="result-icon" :class="{ 'new-version': hasNewVersion }">
            <i v-if="hasNewVersion" class="fas fa-arrow-circle-up"></i>
            <i v-else class="fas fa-check-circle"></i>
          </div>
          <p class="result-text">{{ updateResult }}</p>
          <div v-if="hasNewVersion && updateInfo" class="update-info">
            <div class="update-version-compare">
              <div class="update-version-item">
                <span class="update-version-label">当前版本</span>
                <span class="update-version-value">v{{ APP_CONFIG.version }}</span>
              </div>
              <div class="update-version-arrow"><i class="fas fa-arrow-right"></i></div>
              <div class="update-version-item update-version-new">
                <span class="update-version-label">新版本</span>
                <span class="update-version-value">v{{ updateInfo.version }}</span>
              </div>
            </div>
            <div v-if="updateInfo.releaseDate" class="update-info-row">
              <span>发布时间</span>
              <strong>{{ formatReleaseDate(updateInfo.releaseDate) }}</strong>
            </div>
            <div v-if="updateInfo.releaseNotes" class="update-release-notes">
              <div class="notes-title">发布说明</div>
              <div class="notes-box">{{ updateInfo.releaseNotes }}</div>
            </div>
          </div>
          <p v-else class="version-info">当前版本: v{{ APP_CONFIG.version }}</p>
          <p v-if="forceUpdate && hasNewVersion" class="force-update-tip">此版本为重要更新，需要升级后才能继续使用</p>
        </div>
      </div>
      <div class="update-dialog-footer">
        <button v-if="isUpdateReadyToInstall" class="update-dialog-button update-button" @click="$emit('installUpdate')">立即重启安装</button>
        <button v-else-if="hasNewVersion && !isDownloading" class="update-dialog-button update-button" @click="$emit('downloadUpdate')">{{ forceUpdate ? '立即升级' : '升级' }}</button>
        <button v-if="!forceUpdate" class="update-dialog-button" @click="$emit('closeUpdate')">关闭</button>
      </div>
    </div>
  </div>

  <!-- 系统消息发布模态框 -->
  <div v-if="showSystemMessageModal" class="user-profile-modal" @click="$emit('closeSystemMessage')">
    <div class="user-profile-content" @click.stop>
      <div class="user-profile-header">
        <h3>发布系统消息</h3>
        <button class="close-btn" @click="$emit('closeSystemMessage')">×</button>
      </div>
      <div class="user-profile-body">
        <div class="profile-info">
          <div class="info-item">
            <label>消息标题</label>
            <input type="text" v-model="localMessage.title" class="profile-input" placeholder="请输入消息标题" />
          </div>
          <div class="info-item">
            <label>消息内容</label>
            <textarea v-model="localMessage.content" class="profile-textarea" placeholder="请输入消息内容" rows="4"></textarea>
          </div>
          <div class="info-item">
            <label>发送范围</label>
            <select v-model="localMessage.target" class="profile-input" @change="onTargetChange">
              <option value="all">所有用户</option>
              <option value="department">指定部门</option>
              <option value="group">指定群聊</option>
              <option value="user">指定用户</option>
            </select>
          </div>
          <div v-if="localMessage.target === 'department'" class="info-item">
            <label>选择部门</label>
            <div class="user-search-box">
              <input
                type="text"
                v-model="deptSearchQuery"
                placeholder="搜索部门..."
                class="profile-input"
                @input="handleDeptSearchInput"
              />
              <div v-if="isSearchingDept" class="user-search-loading">搜索中...</div>
              <div class="user-search-results" v-if="showDeptResults">
                <div v-if="filteredDeptList.length === 0" class="user-search-empty">没有找到匹配的部门</div>
                <div
                  v-for="dept in filteredDeptList"
                  :key="dept.id"
                  class="user-search-item"
                  :class="{ selected: (localMessage.targetIds || []).includes(dept.id) }"
                  @click="selectDept(dept)"
                >
                  <span>{{ dept.name }}</span>
                </div>
              </div>
            </div>
            <div v-if="selectedDeptNames.length > 0" class="user-selected-info">
              <div class="selected-tags">
                <span v-for="name in selectedDeptNames" :key="name" class="selected-tag">
                  {{ name }}
                  <i class="fas fa-times" @click="removeDept(name)"></i>
                </span>
              </div>
            </div>
          </div>
          <div v-if="localMessage.target === 'group'" class="info-item">
            <label>选择群聊</label>
            <select v-model="localMessage.groupId" class="profile-input">
              <option value="" disabled>请选择群聊</option>
              <option v-for="group in groupConversations" :key="group.id" :value="group.id">{{ group.name }}</option>
            </select>
          </div>
          <div v-if="localMessage.target === 'user'" class="info-item">
            <label>选择用户</label>
            <div class="user-search-box">
              <input
                type="text"
                v-model="userSearchQuery"
                placeholder="搜索用户..."
                class="profile-input"
                @input="handleUserSearchInput"
              />
              <div v-if="isSearchingUser" class="user-search-loading">搜索中...</div>
              <div class="user-search-results" v-if="showUserResults">
                <div v-if="filteredUserList.length === 0" class="user-search-empty">没有找到匹配的用户</div>
                <div
                  v-for="user in filteredUserList"
                  :key="user.id"
                  class="user-search-item"
                  :class="{ selected: (localMessage.targetIds || []).includes(user.id) }"
                  @click="selectSystemMessageUser(user)"
                >
                  <span>{{ user.name }}</span>
                </div>
              </div>
            </div>
            <div v-if="selectedUserNames.length > 0" class="user-selected-info">
              <div class="selected-tags">
                <span v-for="name in selectedUserNames" :key="name" class="selected-tag">
                  {{ name }}
                  <i class="fas fa-times" @click="removeUser(name)"></i>
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="user-profile-footer">
        <button class="cancel-btn" @click="$emit('closeSystemMessage')">取消</button>
        <button class="save-btn" @click="$emit('sendSystemMessage', localMessage)" :disabled="!localMessage.title || !localMessage.content">发布</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import AppLogo from '../shared/AppLogo.vue'
import { APP_CONFIG, getCopyrightText } from '../../config/appConfig'
import { useServerUrl } from '../../composables/useServerUrl'

const { serverUrl } = useServerUrl()

const productFullName = APP_CONFIG.productFullName
const appVersion = APP_CONFIG.version
const copyrightText = getCopyrightText()

interface Conversation {
  id: string | number
  name: string
  type: string
}

interface SystemMessage {
  title: string
  content: string
  target: string
  groupId?: string | number
  userId?: string | number
  targetIds?: (string | number)[]
}

interface UpdateInfo {
  version: string
  releaseDate?: string
  releaseNotes?: string
}

interface Props {
  showAboutDialog: boolean
  showLogoutDialog: boolean
  showUpdateDialog: boolean
  showSystemMessageModal: boolean
  isCheckingUpdate: boolean
  isDownloading: boolean
  isUpdateReadyToInstall: boolean
  downloadProgress: number
  hasNewVersion: boolean
  forceUpdate: boolean
  updateResult: string
  updateInfo?: UpdateInfo | null
  groupConversations: Conversation[]
  allEmployees: any[]
  systemMessage: SystemMessage
  orgStructure?: any[]
}

const props = defineProps<Props>()

const formatReleaseDate = (value: string) => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  })
}

const emit = defineEmits<{
  'closeAbout': []
  'cancelLogout': []
  'confirmLogout': []
  'closeUpdate': []
  'downloadUpdate': []
  'installUpdate': []
  'closeSystemMessage': []
  'sendSystemMessage': [message: SystemMessage]
  'openFeedback': []
}>()

const localMessage = ref<SystemMessage>({ targetIds: [], ...props.systemMessage })

watch(() => props.systemMessage, (val) => {
  localMessage.value = { targetIds: [], ...val }
}, { deep: true })

// ========== 搜索通用 ==========
const searchTimeout = ref<number | null>(null)

// ========== 用户搜索与多选 ==========
const userSearchQuery = ref('')
const isSearchingUser = ref(false)
const userSearchResults = ref<any[]>([])

const showUserResults = computed(() => {
  return localMessage.value.target === 'user' && userSearchQuery.value.trim().length > 0
})

const filteredUserList = computed(() => {
  if (!userSearchQuery.value.trim()) return []
  return userSearchResults.value
})

const selectedUserNames = computed(() => {
  const ids = localMessage.value.targetIds || []
  if (ids.length === 0) return []
  const names: string[] = []
  for (const id of ids) {
    const found = props.allEmployees.find((e: any) => e.id.toString() === id.toString())
    if (found) names.push(found.name)
    else {
      const fromResults = userSearchResults.value.find((u: any) => u.id.toString() === id.toString())
      names.push(fromResults ? (fromResults.name || fromResults.username) : `用户 ${id}`)
    }
  }
  return names
})

const handleUserSearchInput = () => {
  if (searchTimeout.value) {
    clearTimeout(searchTimeout.value)
  }
  searchTimeout.value = window.setTimeout(async () => {
    const query = userSearchQuery.value.trim()
    if (!query) {
      userSearchResults.value = []
      return
    }

    isSearchingUser.value = true
    try {
      const token = localStorage.getItem('token')
      const response = await fetch(
        `${serverUrl.value}/api/v1/users/search?q=${encodeURIComponent(query)}`,
        {
          headers: {
            ...(token ? { 'Authorization': `Bearer ${token}` } : {})
          }
        }
      )

      if (response.ok) {
        const data = await response.json()
        if (data.code === 0) {
          userSearchResults.value = (data.data || []).map((u: any) => ({
            id: u.id,
            name: u.name || u.username || '',
            username: u.username || ''
          }))
        } else {
          userSearchResults.value = []
        }
      } else {
        userSearchResults.value = []
      }
    } catch (error) {
      console.error('搜索用户失败:', error)
      userSearchResults.value = []
    } finally {
      isSearchingUser.value = false
    }
  }, 300)
}

const selectSystemMessageUser = (user: any) => {
  if (!localMessage.value.targetIds) {
    localMessage.value.targetIds = []
  }
  const exists = localMessage.value.targetIds.some((id: any) => id.toString() === user.id.toString())
  if (!exists) {
    localMessage.value.targetIds.push(user.id)
  }
  userSearchQuery.value = ''
  userSearchResults.value = []
}

const removeUser = (name: string) => {
  // Find the ID from the selected names map
  const ids = localMessage.value.targetIds || []
  const allUsers = [...props.allEmployees, ...userSearchResults.value]
  const user = allUsers.find((u: any) => u.name === name || u.username === name)
  if (user) {
    localMessage.value.targetIds = ids.filter((id: any) => id.toString() !== user.id.toString())
  } else {
    // fallback: remove by index
    const idx = selectedUserNames.value.indexOf(name)
    if (idx >= 0) {
      localMessage.value.targetIds = ids.filter((_: any, i: number) => i !== idx)
    }
  }
}

// ========== 部门搜索与多选 ==========
const deptSearchQuery = ref('')
const isSearchingDept = ref(false)
const deptSearchResults = ref<any[]>([])

// 从 props.orgStructure 扁平化部门列表，避免重复请求 /api/v1/organization/tree
const flattenDepartments = (nodes: any[]): any[] => {
  if (!Array.isArray(nodes)) return []
  const result: any[] = []
  for (const n of nodes) {
    if (!n.id) continue
    result.push({ id: n.id, name: n.name || '' })
    if (Array.isArray(n.subDepartments) && n.subDepartments.length > 0) {
      result.push(...flattenDepartments(n.subDepartments))
    }
  }
  return result
}

const allDeptList = computed(() => flattenDepartments(props.orgStructure || []))

const showDeptResults = computed(() => {
  return localMessage.value.target === 'department' && deptSearchQuery.value.trim().length > 0
})

const filteredDeptList = computed(() => {
  if (!deptSearchQuery.value.trim()) return []
  const q = deptSearchQuery.value.trim().toLowerCase()
  return allDeptList.value.filter((d: any) => d.name.toLowerCase().includes(q))
})

const selectedDeptNames = computed(() => {
  const ids = localMessage.value.targetIds || []
  if (ids.length === 0) return []
  return ids.map((id: any) => {
    const found = allDeptList.value.find((d: any) => d.id.toString() === id.toString())
    return found ? found.name : `部门 ${id}`
  })
})

const handleDeptSearchInput = () => {
  if (searchTimeout.value) {
    clearTimeout(searchTimeout.value)
  }
  searchTimeout.value = window.setTimeout(() => {
    // computed 会自动更新
  }, 100)
}

const selectDept = (dept: any) => {
  if (!localMessage.value.targetIds) {
    localMessage.value.targetIds = []
  }
  const exists = localMessage.value.targetIds.some((id: any) => id.toString() === dept.id.toString())
  if (!exists) {
    localMessage.value.targetIds.push(dept.id)
  }
  deptSearchQuery.value = ''
}

const removeDept = (name: string) => {
  const ids = localMessage.value.targetIds || []
  const dept = allDeptList.value.find((d: any) => d.name === name)
  if (dept) {
    localMessage.value.targetIds = ids.filter((id: any) => id.toString() !== dept.id.toString())
  }
}

// 切换发送范围时重置
const onTargetChange = () => {
  localMessage.value.targetIds = []
  localMessage.value.groupId = undefined
  localMessage.value.userId = undefined
  userSearchQuery.value = ''
  deptSearchQuery.value = ''
  userSearchResults.value = []
}

</script>

<style scoped>
.about-dialog-overlay,
.logout-dialog-overlay,
.update-dialog-overlay {
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

.about-dialog,
.logout-dialog,
.update-dialog {
  background: var(--modal-bg, #fff);
  border-radius: 12px;
  width: 460px;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
}

.about-dialog-header,
.logout-dialog-header,
.update-dialog-header {
  padding: 16px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.about-dialog-header h3,
.logout-dialog-header h3,
.update-dialog-header h3 {
  margin: 0;
  font-size: 16px;
  color: var(--text-color, #333);
}

.about-dialog-close,
.logout-dialog-close,
.update-dialog-close {
  background: none;
  border: none;
  font-size: 20px;
  cursor: pointer;
  color: var(--text-secondary, #999);
}

.about-dialog-content {
  padding: 24px 20px;
  text-align: center;
}

.about-dialog-logo {
  margin-bottom: 16px;
  color: var(--primary-color, #409eff);
}

.about-dialog-logo img {
  width: 80px;
  height: 80px;
  object-fit: contain;
}

.about-dialog-content h2 {
  margin: 0 0 8px 0;
  color: var(--text-color, #333);
}

.version,
.date,
.copyright,
.description {
  margin: 4px 0;
  font-size: 14px;
  color: var(--text-secondary, #999);
}

.credits-section {
  margin: 12px 0;
  padding: 12px;
  background: rgba(0, 0, 0, 0.03);
  border-radius: 8px;
}

.credit-item {
  margin: 6px 0;
  font-size: 14px;
  color: var(--text-color, #666);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.credit-item i {
  color: var(--primary-color, #409eff);
}

.contact {
  margin: 8px 0;
  font-size: 13px;
  color: var(--text-secondary, #999);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.contact a {
  color: var(--primary-color, #409eff);
  text-decoration: none;
}

.contact a:hover {
  text-decoration: underline;
}

.about-dialog-footer,
.logout-dialog-footer,
.update-dialog-footer {
  padding: 16px 20px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.about-dialog-button,
.logout-dialog-button,
.update-dialog-button {
  padding: 8px 24px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.about-dialog-button {
  background: var(--primary-color, #409eff);
  color: white;
}

.about-dialog-button.secondary {
  background: var(--btn-bg, #f5f5f5);
  color: var(--text-color, #333);
  display: flex;
  align-items: center;
  gap: 6px;
}

.about-dialog-button.secondary i {
  color: var(--primary-color, #409eff);
}

.logout-dialog-button.cancel-button,
.update-dialog-button {
  background: var(--btn-bg, #f5f5f5);
  color: var(--text-color, #333);
}

.logout-dialog-button.confirm-button {
  background: #f56c6c;
  color: white;
}

.update-dialog-button.update-button {
  background: var(--primary-color, #409eff);
  color: white;
}

.logout-dialog-content {
  padding: 24px 20px;
}

.logout-dialog-message {
  margin: 0;
  text-align: center;
  font-size: 16px;
  color: var(--text-color, #333);
}

.update-dialog-content {
  padding: 20px 24px;
  min-height: 120px;
}

.update-loading,
.update-downloading,
.update-result {
  text-align: center;
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border-color, #eee);
  border-top-color: var(--primary-color, #409eff);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 12px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.loading-text,
.download-text,
.result-text,
.version-info {
  margin: 8px 0;
  color: var(--text-color, #333);
}

.update-info {
  margin: 16px 0 0;
  padding: 14px;
  border: 1px solid var(--border-color, #eee);
  border-radius: 8px;
  background: var(--card-bg, #fafafa);
  text-align: left;
}

.update-version-compare {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-color, #eee);
}

.update-version-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  min-width: 100px;
}

.update-version-label {
  font-size: 12px;
  color: var(--text-secondary, #999);
}

.update-version-value {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color, #333);
}

.update-version-new .update-version-value {
  color: var(--primary-color, #409eff);
}

.update-version-arrow {
  color: var(--text-secondary, #ccc);
  font-size: 14px;
}

.update-info-row {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  margin: 6px 0;
  font-size: 13px;
  color: var(--text-secondary, #666);
}

.update-info-row strong {
  color: var(--text-color, #333);
  font-weight: 600;
  text-align: right;
}

.update-release-notes {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--border-color, #e7ecf3);
}

.notes-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  font-size: 13px;
  font-weight: 650;
  color: var(--text-color, #374151);
}

.notes-title::before {
  content: "";
  width: 4px;
  height: 14px;
  border-radius: 99px;
  background: var(--primary-color, #409eff);
  flex-shrink: 0;
}

.notes-box {
  max-height: 150px;
  overflow-y: auto;
  padding: 14px 16px;
  border-radius: 12px;
  background: var(--card-bg, #fff);
  border: 1px solid var(--border-color, #e7ecf3);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.9);
  color: var(--text-color, #1f2937);
  font-size: 13px;
  line-height: 1.75;
  white-space: pre-wrap;
}

.notes-box::-webkit-scrollbar {
  width: 5px;
}

.notes-box::-webkit-scrollbar-track {
  background: transparent;
}

.notes-box::-webkit-scrollbar-thumb {
  background: var(--border-color, #dde3ed);
  border-radius: 99px;
}

.force-update-tip {
  margin: 8px 0;
  padding: 8px 12px;
  background-color: #fff3e0;
  border-left: 3px solid #ff9800;
  border-radius: 4px;
  color: #e65100;
  font-size: 13px;
}

.download-icon {
  font-size: 32px;
  color: var(--primary-color, #409eff);
  margin-bottom: 12px;
}

.download-progress {
  margin-top: 16px;
}

.progress-bar {
  height: 8px;
  background: var(--border-color, #eee);
  border-radius: 4px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--primary-color, #409eff);
  transition: width 0.3s ease;
}

.progress-text {
  font-size: 12px;
  color: var(--text-secondary, #999);
}

.result-icon {
  font-size: 36px;
  margin-bottom: 8px;
}

.result-icon.new-version {
  color: var(--primary-color, #409eff);
}

.result-icon:not(.new-version) {
  color: #67c23a;
}

.user-profile-modal {
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

.user-profile-content {
  background: var(--modal-bg, #fff);
  border-radius: 12px;
  width: 500px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}

.user-profile-header {
  padding: 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.user-profile-header h3 {
  margin: 0;
  font-size: 18px;
  color: var(--text-color, #333);
}

.close-btn {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: var(--text-secondary, #999);
}

.user-profile-body {
  padding: 20px;
  overflow: visible;
  flex: 1;
}

.profile-info {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-item label {
  font-size: 12px;
  color: var(--text-secondary, #999);
}

.profile-input,
.profile-textarea {
  padding: 8px 12px;
  border: 1px solid var(--border-color, #ddd);
  border-radius: 4px;
}

.profile-textarea {
  width: 100%;
  min-height: 80px;
  padding: 8px 12px;
  border: 1px solid var(--border-color, #dcdfe6);
  border-radius: 6px;
  font-size: 14px;
  font-family: inherit;
  background: var(--card-bg, #fff);
  color: var(--text-color, #333);
  outline: none;
  resize: vertical;
  box-sizing: border-box;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.profile-textarea:focus {
  border-color: var(--primary-color, #409eff);
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.12);
}

.profile-textarea::placeholder {
  color: #c0c4cc;
}

.user-profile-footer {
  padding: 16px 20px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.cancel-btn {
  padding: 8px 24px;
  border: none;
  border-radius: 4px;
  background: var(--btn-bg, #f5f5f5);
  color: var(--text-color, #333);
  cursor: pointer;
}

.save-btn {
  padding: 8px 24px;
  border: none;
  border-radius: 4px;
  background: var(--primary-color, #409eff);
  color: white;
  cursor: pointer;
}

.save-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.user-search-box {
  position: relative;
}

.user-search-loading {
  padding: 6px 0;
  font-size: 12px;
  color: var(--text-secondary, #999);
  text-align: center;
}

.user-search-results {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  z-index: 2000;
  background: var(--card-bg, #fff);
  border: 1px solid var(--border-color, #e4e7ed);
  border-radius: 6px;
  max-height: 200px;
  overflow-y: auto;
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.12);
  margin-top: 2px;
}

.user-search-empty {
  padding: 10px 14px;
  font-size: 13px;
  color: var(--text-secondary, #999);
  text-align: center;
}

.user-search-item {
  padding: 8px 14px;
  cursor: pointer;
  font-size: 14px;
  color: var(--text-color, #333);
  transition: background 0.15s;
}

.user-search-item:hover {
  background: var(--hover-color, #f0f5ff);
}

.user-search-item.selected {
  background: var(--primary-light, #e6f0ff);
  color: var(--primary-color, #409eff);
  font-weight: 500;
}

.user-selected-info {
  margin-top: 6px;
}

.selected-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.selected-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 10px;
  font-size: 12px;
  background: var(--primary-color, #409eff);
  color: #fff;
  border-radius: 4px;
  line-height: 1.4;
}

.selected-tag i {
  cursor: pointer;
  font-size: 11px;
  opacity: 0.75;
  transition: opacity 0.15s;
}

.selected-tag i:hover {
  opacity: 1;
}
</style>

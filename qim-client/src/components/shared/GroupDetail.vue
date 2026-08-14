<template>
  <div v-if="group" class="group-detail-panel">
    <div class="group-profile-container">
      <div class="group-profile-card">
        <!-- 头像 + 群信息 -->
        <div class="group-info-header">
          <div class="group-avatar-wrapper" :class="{ 'is-owner': isGroupOwner(group) }" @click="isGroupOwner(group) && triggerAvatarUpload()">
            <Avatar
              :src="group.avatar"
              :name="group.name"
              :server-url="serverUrl"
              :alt="group.name"
              size="xl"
              shape="circle"
              class="group-avatar"
            />
            <div v-if="isGroupOwner(group)" class="avatar-overlay">
              <i class="fas fa-camera"></i>
            </div>
          </div>
          <input ref="avatarInput" type="file" accept="image/*" style="display: none" @change="handleAvatarChange" />
          <div class="group-name-row">
            <h2>{{ group.name }}</h2>
            <button v-if="isGroupOwner(group) || isGroupAdmin(group)" class="edit-name-btn" @click="$emit('editGroupName')">
              <i class="fas fa-pen"></i>
            </button>
          </div>
          <div class="group-detail-meta">
            <span class="group-badge-pill"><i class="fas fa-users"></i> {{ group.members ? group.members.length : 0 }} 人</span>
            <span v-if="group.type === 'group'" class="group-badge-pill type">群聊</span>
            <span v-else class="group-badge-pill type">讨论组</span>
          </div>
        </div>

        <!-- 群公告 -->
        <div class="announcement-bar" v-if="group.announcement || isGroupOwner(group)">
          <i class="fas fa-bullhorn ann-icon"></i>
          <span class="ann-text">{{ group.announcement || '暂无公告' }}</span>
          <button v-if="isGroupOwner(group)" class="edit-inline-btn" @click="$emit('editAnnouncement')">
            <i class="fas fa-pen"></i>
          </button>
        </div>

        <!-- 群聊信息 + 邀请权限合并 -->
        <div class="info-section">
          <div class="section-title">群信息</div>
          <div class="info-list">
            <div class="info-row">
              <span class="info-row-icon"><i class="fas fa-crown"></i></span>
              <div class="info-row-text">
                <span class="info-row-label">群主</span>
                <span class="info-row-value">{{ getGroupOwner(group) || '未知' }}</span>
              </div>
            </div>
            <!-- 非群主：只读显示 -->
            <div class="info-row" v-if="group.type === 'group' && !isGroupOwner(group)">
              <span class="info-row-icon"><i class="fas fa-user-lock"></i></span>
              <div class="info-row-text">
                <span class="info-row-label">邀请权限</span>
                <span class="info-row-value">{{ getInvitePermissionText(group.invite_permission) }}</span>
              </div>
            </div>
            <!-- 群主：可编辑 -->
            <div class="info-row" v-if="group.type === 'group' && isGroupOwner(group)">
              <span class="info-row-icon"><i class="fas fa-user-lock"></i></span>
              <div class="info-row-text">
                <span class="info-row-label">邀请权限</span>
                <select
                  v-model="invitePermission"
                  @change="updateInvitePermission"
                  @click.stop @mousedown.stop @mouseup.stop
                  class="permission-select"
                >
                  <option value="owner_admin">群主和管理员</option>
                  <option value="all">所有成员</option>
                </select>
              </div>
            </div>
          </div>
        </div>

        <!-- 操作按钮（合并为一行 flex-wrap） -->
        <div class="action-bar">
          <button class="action-btn primary" @click="$emit('enter', group)">
            <i class="fas fa-comment"></i>
            <span>进入群聊</span>
          </button>
          <button class="action-btn secondary" @click="$emit('invite', group)">
            <i class="fas fa-user-plus"></i>
            <span>邀请</span>
          </button>
          <button v-if="group.type === 'group'" class="action-btn secondary" @click="showGroupFiles = true">
            <i class="fas fa-folder-open"></i>
            <span>群文件</span>
          </button>
          <button v-if="group.type === 'group'" class="action-btn secondary" @click="$emit('openAISettings')">
            <i class="fas fa-robot"></i>
            <span>AI</span>
          </button>
          <button v-if="group.type === 'group' && (isGroupOwner(group) || isGroupAdmin(group))" class="action-btn secondary" title="添加外部 agent 机器人" @click="$emit('addBotToGroup', group)">
            <i class="fas fa-plus"></i>
            <span>机器人</span>
          </button>
        </div>

        <!-- 群成员列表 -->
        <div class="members-section">
          <div class="section-header">
            <span class="section-title">群成员</span>
            <span class="section-count">{{ group.members ? group.members.length : 0 }}</span>
          </div>
          <div class="members-grid">
            <div v-for="member in group.members" :key="member.id" class="member-card" @click="$emit('openChat', member)" @contextmenu.prevent="$emit('showMemberContextMenu', $event, member)">
              <Avatar :src="member.avatar" :name="member.name" :server-url="serverUrl" :alt="member.name" size="sm" shape="circle" class="member-avatar" />
              <span class="member-name">{{ member.name }}</span>
              <span v-if="member.role === 'owner'" class="member-role-badge owner">群主</span>
              <span v-else-if="member.role === 'admin'" class="member-role-badge admin">管理</span>
            </div>
          </div>
        </div>
      </div>
    </div>
    <GroupFilesPanel
      v-if="showGroupFiles"
      :group-id="group.id"
      :can-manage="isGroupOwner(group) || isGroupAdmin(group)"
      @close="showGroupFiles = false"
    />
  </div>
  <div v-else class="group-detail-placeholder">
    <div class="placeholder-content">
      <i class="fas fa-users fa-4x"></i>
      <h3>选择一个群聊查看详情</h3>
      <p>点击左侧的群聊列表项，查看群聊的详细信息</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import Avatar from './Avatar.vue'
import { generateAvatar, getAvatarUrl, isAbsoluteUrl } from '../../utils/avatar'
import { getStoredServerUrl } from '../../composables/useServerUrl'
import type { Conversation, User } from '../../types'
import { logger } from '../../utils/logger';
import QMessage from '../../utils/qmessage'
import { getCurrentUser } from '../../utils/user'
import GroupFilesPanel from '../groups/GroupFilesPanel.vue'

const serverUrl = getStoredServerUrl()
const avatarInput = ref<HTMLInputElement | null>(null)
const showGroupFiles = ref(false)

interface Props {
  group: Conversation | null
}

const props = defineProps<Props>()

defineEmits<{
  enter: [conversation: Conversation]
  invite: [conversation: Conversation]
  editAnnouncement: []
  editGroupName: []
  openAISettings: []
  addBotToGroup: [conversation: Conversation]
  showMemberContextMenu: [event: MouseEvent, member: any]
  openChat: [user: any]
}>()

// 获取群聊头像
const getGroupAvatar = (group: Conversation) => {
  return getAvatarUrl(group.avatar, group.name || '群聊', serverUrl)
}

// 获取成员头像
const getMemberAvatar = (member: User) => {
  if (!member) return generateAvatar('成员')
  if (member.avatar && isAbsoluteUrl(member.avatar)) {
    return member.avatar
  }
  if (member.avatar) {
    return serverUrl + member.avatar
  }
  return generateAvatar(member.name || '成员')
}

// 获取群聊群主
const getGroupOwner = (group: Conversation | null) => {
  if (!group || !group.members) return ''
  const owner = group.members.find((member: User) => member.role === 'owner')
  if (owner) return owner.name

  const ownerId = (group as any).ownerId || (group as any).owner_id || (group as any).creator_id
  if (!ownerId) return ''

  const ownerById = group.members.find((member: User) => String(member.id) === String(ownerId))
  return ownerById ? (ownerById.name || ownerById.nickname || ownerById.username || '') : ''
}

const getCurrentUserGroupRole = (group: Conversation | null) => {
  if (!group || !group.members) return false
  const currentUser = getCurrentUser()
  if (!currentUser || !currentUser.id) return false

  const currentMember = group.members.find((member: User) =>
    String((member as any).user?.id ?? member.id) === String(currentUser.id)
  )

  logger.log('权限检查:', { 
    currentUserId: currentUser.id, 
    currentUserRole: currentMember?.role,
  })
  return currentMember?.role || 'member'
}

// 检查当前用户是否是群主
const isGroupOwner = (group: Conversation | null) => {
  return getCurrentUserGroupRole(group) === 'owner'
}

// 检查当前用户是否是群管理员
const isGroupAdmin = (group: Conversation | null) => {
  return getCurrentUserGroupRole(group) === 'admin'
}

// 邀请权限
const invitePermission = ref('owner_admin')

// 监听group变化，更新邀请权限
watch(
  () => props.group,
  (newGroup) => {
    if (newGroup) {
      invitePermission.value = newGroup.invite_permission || 'owner_admin'
    }
  },
  { immediate: true }
)

// 获取邀请权限文本
const getInvitePermissionText = (permission: string | undefined) => {
  switch (permission) {
    case 'all':
      return '所有成员'
    case 'owner_admin':
    default:
      return '群主和管理员'
  }
}

// 更新邀请权限
const updateInvitePermission = async () => {
  if (!props.group) return
  
  try {
    const response = await fetch(`${serverUrl}/api/v1/groups/${props.group.id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({
        invite_permission: invitePermission.value
      })
    })
    
    if (response.ok) {
      // 更新成功
      logger.log('邀请权限更新成功')
    } else {
      // 恢复原设置
      if (props.group) {
        invitePermission.value = props.group.invite_permission || 'owner_admin'
      }
      console.error('邀请权限更新失败')
    }
  } catch (error) {
    console.error('更新邀请权限时出错:', error)
    // 恢复原设置
    if (props.group) {
      invitePermission.value = props.group.invite_permission || 'owner_admin'
    }
  }
}

// 触发头像上传
const triggerAvatarUpload = async () => {
  if (avatarInput.value) {
    await nextTick()
    avatarInput.value.click()
  }
}

// 处理头像上传
const handleAvatarChange = async (event: Event) => {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file || !props.group) return

  // 检查文件大小（最大 2MB）
  if (file.size > 2 * 1024 * 1024) {
    QMessage.error('头像文件大小不能超过 2MB')
    return
  }

  try {
    // 将图片转为 base64
    const reader = new FileReader()
    reader.onload = async (e) => {
      const base64 = e.target?.result as string
      
      // 调用后端 API 更新群头像
      const response = await fetch(`${serverUrl}/api/v1/groups/${props.group!.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({
          avatar: base64
        })
      })

      if (response.ok) {
        QMessage.success('群头像更新成功')
        // 清空 file input
        target.value = ''
      } else {
        QMessage.error('群头像更新失败')
      }
    }
    reader.readAsDataURL(file)
  } catch (error) {
    console.error('更新群头像时出错:', error)
    QMessage.error('群头像更新失败')
  }
}
</script>

<style scoped>
.group-detail-panel {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  background: var(--right-content-bg, #f5f5f5);
}

.group-detail-placeholder {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-color, #f5f5f5);
  border-left: 1px solid var(--border-color, #e8e8e8);
}

.placeholder-content {
  text-align: center;
  color: var(--text-secondary, #666);
}

.placeholder-content i {
  color: var(--text-tertiary, #999);
  margin-bottom: 16px;
}

.placeholder-content h3 {
  margin: 0 0 8px 0;
  color: var(--text-primary, #333);
}

.placeholder-content p {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--text-secondary, #666);
}

/* ---- 头像 + 群信息区 ---- */
.group-info-header {
  padding: 28px 20px 20px;
  text-align: center;
  background: var(--card-bg, #fff);
}

.group-avatar-wrapper {
  position: relative;
  display: inline-block;
  margin-bottom: 14px;
  cursor: default;
}

.group-avatar-wrapper.is-owner {
  cursor: pointer;
}

.group-avatar-wrapper.is-owner:hover .avatar-overlay {
  opacity: 1;
}

.group-avatar {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  border: 3px solid var(--border-color, rgba(0,0,0,0.08));
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.avatar-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  opacity: 0;
  transition: opacity 0.2s;
  font-size: var(--font-size-lg);
}

.group-name-row {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin-bottom: 8px;
}

.group-name-row h2 {
  margin: 0;
  font-size: var(--font-size-lg);
  font-weight: 700;
  color: var(--text-color, #333);
}

.edit-name-btn {
  background: var(--hover-color, rgba(0,0,0,0.04));
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 6px;
  font-size: var(--font-size-xxxs);
  transition: all 0.2s;
}

.edit-name-btn:hover {
  background: var(--border-color, rgba(0,0,0,0.08));
  color: var(--text-color);
}

.group-detail-meta {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.group-badge-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary);
  background: var(--hover-color, rgba(0,0,0,0.04));
  padding: 3px 10px;
  border-radius: 20px;
}

.group-badge-pill.type {
  background: var(--primary-color);
  color: #fff;
  opacity: 0.85;
}

/* ---- 群公告 ---- */
.announcement-bar {
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

.ann-icon {
  font-size: var(--font-size-xxs);
  opacity: 0.4;
  color: var(--primary-color);
  flex-shrink: 0;
}

.ann-text {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.edit-inline-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 6px;
  font-size: var(--font-size-xxxs);
  flex-shrink: 0;
  transition: all 0.2s;
}

.edit-inline-btn:hover {
  background: var(--hover-color);
  color: var(--primary-color);
}

/* ---- 信息列表 ---- */
.info-section {
  padding: 16px 20px;
  background: var(--card-bg, #fff);
  border-bottom: 1px solid var(--border-color, rgba(0,0,0,0.06));
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
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.info-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 8px;
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
  line-height: 1.4;
}

.permission-select {
  padding: 4px 28px 4px 10px;
  border: 1px solid var(--border-color, rgba(0, 0, 0, 0.1));
  border-radius: 6px;
  background: var(--card-bg);
  color: var(--text-color);
  font-size: var(--font-size-xs);
  cursor: pointer;
  appearance: none;
  -webkit-appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6'%3E%3Cpath d='M0 0l5 6 5-6z' fill='%239ca3af'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 8px center;
}

.permission-select:focus {
  outline: none;
  border-color: var(--primary-color);
}

.permission-select option {
  background: var(--card-bg);
  color: var(--text-color);
}

/* ---- 操作按钮 ---- */
.action-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 12px 20px;
  background: var(--card-bg, #fff);
  border-bottom: 1px solid var(--border-color, rgba(0,0,0,0.06));
}

.action-btn {
  flex: 1 0 auto;
  min-width: 0;
  padding: 10px 14px;
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
  white-space: nowrap;
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

/* ---- 群成员 ---- */
.members-section {
  padding: 16px 20px;
  background: var(--card-bg, #fff);
}

.section-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
}

.section-header .section-title {
  margin-bottom: 0;
}

.section-count {
  font-size: var(--font-size-xxs);
  font-weight: 600;
  color: var(--primary-color);
  background: rgba(59, 130, 246, 0.08);
  padding: 1px 8px;
  border-radius: 10px;
}

.members-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(64px, 1fr));
  gap: 8px;
}

.member-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 8px 4px;
  border-radius: 10px;
  cursor: pointer;
  transition: background 0.15s;
}

.member-card:hover {
  background: var(--hover-color, rgba(0,0,0,0.03));
}

.member-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  margin-bottom: 4px;
}

.member-name {
  font-size: var(--font-size-xxxs);
  color: var(--text-color);
  max-width: 56px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.3;
}

.member-role-badge {
  font-size: var(--font-size-tiny);
  padding: 1px 6px;
  border-radius: 8px;
  font-weight: 600;
  margin-top: 2px;
  white-space: nowrap;
}

.member-role-badge.owner {
  background: rgba(255, 215, 0, 0.15);
  color: #d4a017;
}

.member-role-badge.admin {
  background: rgba(79, 172, 254, 0.1);
  color: #2196f3;
}
</style>


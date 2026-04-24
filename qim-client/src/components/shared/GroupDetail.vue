<template>
  <div v-if="group" class="group-detail-panel">
    <div class="group-profile-container">
      <!-- 顶部背景 -->
      <div class="group-profile-header-bg"></div>
      
      <!-- 群聊信息卡片 -->
      <div class="group-profile-card">
        <!-- 头像和基本信息 -->
        <div class="group-profile-avatar-section">
          <div class="group-avatar-container">
            <img
              :src="getGroupAvatar(group)"
              :alt="group.name"
              class="group-avatar"
            />
          </div>
          <div class="group-basic-info">
            <h2 class="group-full-name">{{ group.name }}</h2>
            <p class="group-member-count">{{ group.members ? group.members.length + '人' : '0人' }}</p>
          </div>
        </div>
        
        <!-- 信息分组 -->
        <div class="group-info-sections">
          <!-- 基本信息 -->
          <div class="info-section">
            <div class="section-title">
              <i class="fas fa-info-circle"></i>
              <h3>群聊信息</h3>
            </div>
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">群成员</span>
                <span class="info-value">{{ group.members ? group.members.length : 0 }}人</span>
              </div>
              <div class="info-item">
                <span class="info-label">群主</span>
                <span class="info-value">{{ getGroupOwner(group) || '未知' }}</span>
              </div>
              <div class="info-item full-width">
                <span class="info-label">群公告</span>
                <div class="group-announcement">
                  <span class="announcement-content">{{ group.announcement || '暂无公告' }}</span>
                  <button v-if="isGroupOwner(group)" class="edit-announcement-btn" @click="$emit('editAnnouncement')">
                    <i class="fas fa-edit"></i>
                  </button>
                </div>
              </div>
            </div>
          </div>
          
          <!-- 权限设置 -->
          <div class="info-section" v-if="group.type === 'group'">
            <div class="section-title">
              <i class="fas fa-shield-alt"></i>
              <h3>权限设置</h3>
            </div>
            <div class="info-grid">
              <div class="info-item full-width">
                <span class="info-label">邀请权限</span>
                <div class="permission-setting" v-if="isGroupOwner(group)">
                  <select v-model="invitePermission" @change="updateInvitePermission" class="permission-select">
                    <option value="owner_admin">群主和管理员</option>
                    <option value="all">所有成员</option>
                  </select>
                </div>
                <span v-else class="info-value">{{ getInvitePermissionText(group.invite_permission) }}</span>
              </div>
            </div>
          </div>
        </div>
        
        <!-- 操作按钮 -->
        <div class="group-action-buttons">
          <button class="action-btn primary" @click="$emit('enter', group)">
            <i class="fas fa-comment"></i>
            <span>进入群聊</span>
          </button>
          <button class="action-btn secondary" @click="$emit('invite', group)">
            <i class="fas fa-user-plus"></i>
            <span>邀请成员</span>
          </button>
        </div>
        
        <!-- 群成员列表 -->
        <div class="group-members-section">
          <div class="section-title">
            <i class="fas fa-users"></i>
            <h3>群成员列表</h3>
          </div>
          <div class="members-grid">
            <div v-for="member in group.members" :key="member.id" class="member-item" @click="$emit('startPrivateChat', member)" @contextmenu.prevent="$emit('showMemberContextMenu', $event, member)">
              <img :src="getMemberAvatar(member)" :alt="member.name" class="member-avatar" />
              <span class="member-name">{{ member.name }}</span>
              <span v-if="member.role === 'owner'" class="member-role owner">群主</span>
              <span v-else-if="member.role === 'admin'" class="member-role admin">管理员</span>
            </div>
          </div>
        </div>
      </div>
    </div>
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
import { computed, ref, watch } from 'vue'
import { generateAvatar, getAvatarUrl } from '../../utils/avatar'
import { API_BASE_URL } from '../../config'
import type { Conversation, User } from '../../types'

const serverUrl = localStorage.getItem('serverUrl') || API_BASE_URL

interface Props {
  group: Conversation | null
}

const props = defineProps<Props>()

defineEmits<{
  enter: [conversation: Conversation]
  invite: [conversation: Conversation]
  editAnnouncement: []
  showMemberContextMenu: [event: MouseEvent, member: any]
  startPrivateChat: [user: any]
}>()

// 获取群聊头像
const getGroupAvatar = (group: Conversation) => {
  return getAvatarUrl(group.avatar, group.name || '群聊', serverUrl)
}

// 获取成员头像
const getMemberAvatar = (member: User) => {
  if (!member) return generateAvatar('成员')
  if (member.avatar && member.avatar.startsWith('http')) {
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
  return owner ? owner.name : ''
}

// 检查当前用户是否是群主
const isGroupOwner = (group: Conversation | null) => {
  if (!group || !group.members) return false
  const currentUserId = localStorage.getItem('userId') || ''
  const owner = group.members.find((member: User) => member.role === 'owner')
  return owner ? owner.id === currentUserId : false
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
    const response = await fetch(`${serverUrl}/api/v1/conversations/${props.group.id}`, {
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
      console.log('邀请权限更新成功')
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
</script>

<style scoped>
.group-detail-panel {
  flex: 1;
  overflow-y: auto;
  padding: 0;
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
  font-size: 14px;
  color: var(--text-secondary, #666);
}

/* 群聊详情样式 */
.group-profile-container {
  position: relative;
  padding: 16px;
  margin: 10px 5px;
}

.group-profile-header-bg {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 80px;
  background: linear-gradient(135deg, var(--primary-light), var(--active-color));
  border-radius: 8px 8px 0 0;
  z-index: 1;
}

.group-profile-card {
  position: relative;
  background: var(--card-bg);
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  border: 1px solid var(--border-color);
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

.group-profile-avatar-section {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border-color);
}

.group-avatar-container {
  position: relative;
  margin-right: 16px;
}

.group-avatar {
  width: 64px;
  height: 64px;
  border-radius: 10px;
  object-fit: cover;
  border: 3px solid white;
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
  transition: transform 0.3s ease;
}

.group-avatar:hover {
  transform: scale(1.05);
}

.group-basic-info {
  flex: 1;
}

.group-full-name {
  margin: 0 0 6px 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--text-color);
  letter-spacing: 0.5px;
}

.group-member-count {
  margin: 0;
  font-size: 13px;
  color: var(--text-secondary);
}

.group-info-sections {
  margin-bottom: 20px;
}

.info-section {
  margin-bottom: 16px;
  background: var(--list-bg);
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
  gap: 3px;
}

.info-item.full-width {
  grid-column: 1 / -1;
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

.group-announcement {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 5px 8px;
  background: var(--input-bg);
  border-radius: 4px;
  border: 1px solid var(--border-color);
  transition: all 0.2s ease;
}

.group-announcement:hover {
  border-color: var(--primary-color);
  box-shadow: 0 2px 8px rgba(102, 126, 234, 0.15);
}

.announcement-content {
  flex: 1;
  font-size: 13px;
  color: var(--text-color);
  line-height: 1.4;
}

.edit-announcement-btn {
  background: none;
  border: none;
  color: var(--primary-color);
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  transition: background 0.2s;
  align-self: center;
}

.edit-announcement-btn:hover {
  background: var(--primary-light);
}

.group-action-buttons {
  display: flex;
  gap: 10px;
  justify-content: center;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
  margin-bottom: 20px;
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

.group-members-section {
  margin-top: 20px;
}

.members-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(80px, 1fr));
  gap: 16px;
  margin-top: 12px;
}

.member-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 8px;
  border-radius: 8px;
  transition: background 0.2s;
}

.member-item:hover {
  background: var(--hover-color);
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
}

.member-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  object-fit: cover;
  margin-bottom: 8px;
  border: 2px solid var(--border-color);
  transition: transform 0.3s ease;
}

.member-avatar:hover {
  transform: scale(1.1);
}

.member-name {
  font-size: 12px;
  color: var(--text-color);
  margin-bottom: 4px;
  word-break: break-all;
}

.member-role {
  font-size: 10px;
  padding: 1px 4px;
  border-radius: 3px;
  font-weight: 500;
  white-space: nowrap;
}

.member-role.owner {
  background: linear-gradient(135deg, #ffd700, #ffaa00);
  color: #fff;
  box-shadow: 0 2px 4px rgba(255, 215, 0, 0.3);
}

.member-role.admin {
  background: linear-gradient(135deg, #4facfe, #00f2fe);
  color: #fff;
  box-shadow: 0 2px 4px rgba(79, 172, 254, 0.3);
}

/* 权限设置样式 */
.permission-setting {
  padding: 5px 8px;
  background: var(--input-bg);
  border-radius: 4px;
  border: 1px solid var(--border-color);
  transition: all 0.2s ease;
}

.permission-setting:hover {
  border-color: var(--primary-color);
  box-shadow: 0 2px 8px rgba(102, 126, 234, 0.15);
}

.permission-select {
  width: 100%;
  padding: 6px 10px;
  border: none;
  background: transparent;
  font-size: 13px;
  color: var(--text-color);
  font-weight: 500;
  cursor: pointer;
  outline: none;
}

.permission-select option {
  background: var(--card-bg);
  color: var(--text-color);
  padding: 8px;
}
</style>
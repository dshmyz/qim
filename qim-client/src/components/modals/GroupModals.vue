<template>
  <!-- 群成员模态框 -->
  <div v-if="showGroupMembersModal" class="add-members-modal" @click="$emit('closeGroupMembers')">
    <div class="add-members-content" @click.stop>
      <div class="add-members-header">
        <h3>群成员列表</h3>
        <button class="close-btn" @click="$emit('closeGroupMembers')">×</button>
      </div>
      <div class="add-members-body">
        <div class="group-info">
          <div class="group-avatar">
            <Avatar :src="selectedGroup?.avatar" :name="selectedGroup?.name || '群聊'" :server-url="serverUrl" :alt="selectedGroup?.name" size="md" />
          </div>
          <div class="group-details">
            <div class="group-name">{{ selectedGroup?.name }}</div>
            <div class="group-members-count">{{ groupMembers.length }} 位成员</div>
          </div>
        </div>
        
        <div class="members-section">
          <div class="section-header">
            <span>成员列表</span>
          </div>
          <div class="members-list">
            <div v-for="member in groupMembers" :key="member.id" class="member-item">
              <div class="member-avatar">
                <Avatar :src="member.avatar" :name="member.name" :alt="member.name" size="sm" />
              </div>
              <div class="member-info">
                <div class="member-name">{{ member.name }}</div>
                <div class="member-position">{{ member.position || '无职位信息' }}</div>
              </div>
              <div class="member-actions">
                <button class="remove-member-btn" @click="$emit('removeMember', member)" v-if="member.id !== currentUserId">
                  <i class="fas fa-trash-alt"></i>
                </button>
              </div>
            </div>
            <div v-if="groupMembers.length === 0" class="empty-state">
              <p>暂无成员</p>
            </div>
          </div>
        </div>
      </div>
      <div class="add-members-footer">
        <button class="cancel-btn" @click="$emit('closeGroupMembers')">关闭</button>
      </div>
    </div>
  </div>

  <!-- 群资料模态框 -->
  <div v-if="showGroupInfoModal" class="add-members-modal" @click="$emit('closeGroupInfo')">
    <div class="add-members-content" @click.stop>
      <div class="add-members-header">
        <h3>群聊资料</h3>
        <button class="close-btn" @click="$emit('closeGroupInfo')">×</button>
      </div>
      <div class="add-members-body">
        <div class="group-info">
          <div class="group-avatar" style="width: 80px; height: 80px;">
            <Avatar :src="selectedGroup?.avatar" :name="selectedGroup?.name || '群聊'" :server-url="serverUrl" :alt="selectedGroup?.name" size="xl" />
          </div>
          <div class="group-details">
            <div class="group-name" style="font-size: 20px;">{{ selectedGroup?.name }}</div>
            <div class="group-members-count">{{ selectedGroup?.members?.length || 0 }} 位成员</div>
          </div>
        </div>
        
        <div class="group-details-section">
          <div class="detail-item">
            <div class="detail-label">群聊ID</div>
            <div class="detail-value">{{ selectedGroup?.id }}</div>
          </div>
          <div class="detail-item">
            <div class="detail-label">创建时间</div>
            <div class="detail-value">{{ selectedGroup?.createdAt ? formatTime(selectedGroup.createdAt) : '未知' }}</div>
          </div>
          <div class="detail-item">
            <div class="detail-label">群聊类型</div>
            <div class="detail-value">群聊</div>
          </div>
        </div>
      </div>
      <div class="add-members-footer">
        <button class="cancel-btn" @click="$emit('closeGroupInfo')">关闭</button>
      </div>
    </div>
  </div>

  <!-- 添加成员模态框 -->
  <div v-if="showAddMembersModal" class="add-members-modal" @click="$emit('closeAddMembers')">
    <div class="add-members-content" @click.stop>
      <div class="add-members-header">
        <h3>邀请成员加入群聊</h3>
        <button class="close-btn" @click="$emit('closeAddMembers')">×</button>
      </div>
      <div class="add-members-body">
        <div class="group-info">
          <div class="group-avatar">
            <Avatar :src="selectedGroup?.avatar" :name="selectedGroup?.name || '群聊'" :server-url="serverUrl" :alt="selectedGroup?.name" size="md" />
          </div>
          <div class="group-details">
            <div class="group-name">{{ selectedGroup?.name }}</div>
            <div class="group-members-count">{{ selectedGroup?.members?.length || 0 }} 位成员</div>
          </div>
        </div>
        
        <div class="search-section">
          <div class="search-box">
            <input type="text" v-model="localSearchQuery" placeholder="搜索成员..." class="search-input" />
          </div>
        </div>
        
        <div class="members-section">
          <div class="section-header">
            <span>选择成员</span>
            <span class="selected-count">{{ localSelectedMembers.length }} 已选择</span>
          </div>
          <SelectedMembersBar :members="localSelectedMembers" @remove="removeSelected" />
          <!-- 组织树按部门选人（搜索时本地过滤树，保留部门上下文） -->
          <div class="members-list org-tree-list">
            <OrgTreePicker :org-structure="orgStructure" :search-query="localSearchQuery" :selected-members="localSelectedMembers" :existing-member-ids="existingMemberIds" @update:selected-members="localSelectedMembers = $event" />
          </div>
        </div>
      </div>
      <div class="add-members-footer">
        <button class="cancel-btn" @click="$emit('closeAddMembers')">取消</button>
        <button class="confirm-btn" @click="$emit('confirmAddMembers', [...localSelectedMembers])" :disabled="localSelectedMembers.length === 0">
          邀请 ({{ localSelectedMembers.length }})
        </button>
      </div>
    </div>
  </div>

  <!-- 编辑群名称模态框 -->
  <div v-if="showEditGroupNameModal" class="add-members-modal" @click="$emit('closeEditGroupName')">
    <div class="add-members-content" @click.stop>
      <div class="add-members-header">
        <h3>修改群名称</h3>
        <button class="close-btn" @click="$emit('closeEditGroupName')">×</button>
      </div>
      <div class="add-members-body">
        <div class="group-info">
          <div class="group-avatar">
            <Avatar :src="selectedGroup?.avatar" :name="selectedGroup?.name || '群聊'" :server-url="serverUrl" :alt="selectedGroup?.name" size="md" />
          </div>
          <div class="group-details">
            <div class="group-name">{{ selectedGroup?.name }}</div>
          </div>
        </div>
        
        <div class="group-name-edit-section">
          <input type="text" v-model="localGroupName" placeholder="请输入群名称" class="group-name-input" />
          <p class="group-name-tip">群名称将对所有群成员可见</p>
        </div>
      </div>
      <div class="add-members-footer">
        <button class="cancel-btn" @click="$emit('closeEditGroupName')">取消</button>
        <button class="confirm-btn" @click="$emit('saveGroupName', localGroupName)">保存</button>
      </div>
    </div>
  </div>

  <!-- 编辑群公告模态框 -->
  <div v-if="showEditAnnouncementModal" class="add-members-modal" @click="$emit('closeEditAnnouncement')">
    <div class="add-members-content" @click.stop>
      <div class="add-members-header">
        <h3>编辑群公告</h3>
        <button class="close-btn" @click="$emit('closeEditAnnouncement')">×</button>
      </div>
      <div class="add-members-body">
        <div class="group-info">
          <div class="group-avatar">
            <Avatar :src="selectedGroup?.avatar" :name="selectedGroup?.name || '群聊'" :server-url="serverUrl" :alt="selectedGroup?.name" size="md" />
          </div>
          <div class="group-details">
            <div class="group-name">{{ selectedGroup?.name }}</div>
          </div>
        </div>
        
        <div class="announcement-edit-section">
          <textarea v-model="localAnnouncement" placeholder="输入群公告内容..." class="announcement-textarea" rows="6"></textarea>
          <p class="announcement-tip">群公告将对所有群成员可见</p>
        </div>
      </div>
      <div class="add-members-footer">
        <button class="cancel-btn" @click="$emit('closeEditAnnouncement')">取消</button>
        <button class="confirm-btn" @click="$emit('saveAnnouncement', localAnnouncement)">保存</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Avatar from '../shared/Avatar.vue'
import OrgTreePicker from '../shared/OrgTreePicker.vue'
import SelectedMembersBar from '../shared/SelectedMembersBar.vue'
import { getAvatarUrl } from '../../utils/avatar'
import { useServerUrl } from '../../composables/useServerUrl'
import type { Department } from '../../composables/useOrganizationLogic'

const { serverUrl } = useServerUrl()

interface Member {
  id: string | number
  name: string
  avatar: string
  position?: string
}

interface Group {
  id: string | number
  name: string
  avatar?: string
  members?: Member[]
  createdAt?: string
}

interface Employee extends Member {
  username?: string
}

interface Props {
  showGroupMembersModal: boolean
  showGroupInfoModal: boolean
  showAddMembersModal: boolean
  showEditGroupNameModal: boolean
  showEditAnnouncementModal: boolean
  selectedGroup: Group | null
  groupMembers: Member[]
  allEmployees: Employee[]
  orgStructure: Department[]
  addMembersSearchQuery: string
  selectedAddMembers: Employee[]
  editGroupName: string
  editAnnouncementContent: string
  currentUserId?: string | number
  formatTime: (date: string) => string
}

const props = defineProps<Props>()

const existingMemberIds = computed(() => {
  return (props.selectedGroup?.members ?? []).map(m => m.id)
})

const emit = defineEmits<{
  'closeGroupMembers': []
  'closeGroupInfo': []
  'closeAddMembers': []
  'closeEditGroupName': []
  'closeEditAnnouncement': []
  'removeMember': [member: Member]
  'confirmAddMembers': [members: Employee[]]
  'saveGroupName': [name: string]
  'saveAnnouncement': [content: string]
}>()

const localSearchQuery = ref(props.addMembersSearchQuery)
const localSelectedMembers = ref([...props.selectedAddMembers])
const localGroupName = ref(props.editGroupName)
const localAnnouncement = ref(props.editAnnouncementContent)

watch(() => props.addMembersSearchQuery, (val) => { localSearchQuery.value = val })
watch(() => props.selectedAddMembers, (val) => { localSelectedMembers.value = [...val] })
watch(() => props.editGroupName, (val) => { localGroupName.value = val })
watch(() => props.editAnnouncementContent, (val) => { localAnnouncement.value = val })

const removeSelected = (member: Employee) => {
  localSelectedMembers.value = localSelectedMembers.value.filter(m => String(m.id) !== String(member.id))
}
</script>

<style scoped>
.add-members-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(2px);
}

.add-members-content {
  background: var(--modal-bg, #fff);
  border-radius: 14px;
  width: 500px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.18), 0 2px 8px rgba(0, 0, 0, 0.06);
}

.add-members-header {
  padding: 18px 22px 14px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.add-members-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color, #1a1a1a);
  letter-spacing: -0.01em;
}

.close-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  font-size: 18px;
  cursor: pointer;
  color: var(--text-secondary, #999);
  border-radius: 6px;
  transition: all 0.15s;
}

.close-btn:hover {
  background: var(--hover-color, #f5f5f5);
  color: var(--text-color, #333);
}

.add-members-body {
  padding: 14px 22px 18px;
  overflow-y: auto;
  flex: 1;
}

.group-info {
  display: flex;
  gap: 12px;
  align-items: center;
  margin-bottom: 16px;
}

.group-avatar img {
  width: 42px;
  height: 42px;
  border-radius: 50%;
}

.group-details {
  flex: 1;
}

.group-name {
  font-weight: 500;
  font-size: 14px;
  color: var(--text-color, #1a1a1a);
}

.group-members-count {
  font-size: 12px;
  color: var(--text-secondary, #999);
  margin-top: 2px;
}

.search-section {
  margin-bottom: 12px;
}

.search-input {
  width: 100%;
  padding: 8px 12px 8px 34px;
  border: 1px solid var(--border-color, #e0e0e0);
  border-radius: 8px;
  font-size: 13px;
  background: var(--modal-bg, #fff) url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='14' height='14' viewBox='0 0 24 24' fill='none' stroke='%23999' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Ccircle cx='11' cy='11' r='8'/%3E%3Cline x1='21' y1='21' x2='16.65' y2='16.65'/%3E%3C/svg%3E") no-repeat 10px center;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.search-input:focus {
  outline: none;
  border-color: var(--primary-color, #409eff);
  box-shadow: 0 0 0 3px rgba(64, 158, 255, 0.1);
  background-color: var(--modal-bg, #fff);
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-color, #1a1a1a);
}

.selected-count {
  font-size: 12px;
  font-weight: 400;
  color: var(--text-secondary, #999);
}

.members-list {
  display: flex;
  flex-direction: column;
}

.member-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px;
  border-radius: 8px;
  cursor: pointer;
}

.member-item:hover {
  background: var(--hover-color, #f5f5f5);
}

.member-item.selected {
  background: var(--selected-bg, #e8f4ff);
}

.member-avatar img {
  width: 36px;
  height: 36px;
  border-radius: 50%;
}

.member-info {
  flex: 1;
}

.member-name {
  font-size: 14px;
  color: var(--text-color, #1a1a1a);
}

.member-position {
  font-size: 12px;
  color: var(--text-secondary, #999);
}

.member-checkbox .checkbox {
  cursor: pointer;
}

.remove-member-btn {
  background: none;
  border: none;
  color: #f56c6c;
  cursor: pointer;
  padding: 4px;
}

.empty-state {
  text-align: center;
  padding: 20px 0;
  color: var(--text-secondary, #999);
}

/* ── 编辑群名称 ── */
.group-name-edit-section {
  margin-top: 4px;
}

.group-name-input {
  width: 100%;
  padding: 10px 14px;
  border: 1px solid var(--border-color, #e0e0e0);
  border-radius: 8px;
  font-size: 14px;
  color: var(--text-color, #1a1a1a);
  transition: border-color 0.2s, box-shadow 0.2s;
  box-sizing: border-box;
}

.group-name-input::placeholder {
  color: var(--text-secondary, #bbb);
}

.group-name-input:focus {
  outline: none;
  border-color: var(--primary-color, #409eff);
  box-shadow: 0 0 0 3px rgba(64, 158, 255, 0.1);
}

.group-name-tip {
  font-size: 12px;
  color: var(--text-secondary, #999);
  margin: 8px 0 0;
  line-height: 1.5;
}

/* ── 编辑群公告 ── */
.announcement-edit-section {
  margin-top: 4px;
}

.announcement-textarea {
  width: 100%;
  padding: 10px 14px;
  border: 1px solid var(--border-color, #e0e0e0);
  border-radius: 8px;
  resize: vertical;
  font-family: inherit;
  font-size: 14px;
  color: var(--text-color, #1a1a1a);
  line-height: 1.6;
  transition: border-color 0.2s, box-shadow 0.2s;
  box-sizing: border-box;
}

.announcement-textarea::placeholder {
  color: var(--text-secondary, #bbb);
}

.announcement-textarea:focus {
  outline: none;
  border-color: var(--primary-color, #409eff);
  box-shadow: 0 0 0 3px rgba(64, 158, 255, 0.1);
}

.announcement-tip {
  font-size: 12px;
  color: var(--text-secondary, #999);
  margin: 8px 0 0;
  line-height: 1.5;
}

/* ── 群资料详情 ── */
.group-details-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 16px;
}

.detail-item {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
}

.detail-label {
  color: var(--text-secondary, #999);
  font-size: 14px;
}

.detail-value {
  color: var(--text-color, #1a1a1a);
  font-size: 14px;
}

/* ── 底部按钮 ── */
.add-members-footer {
  padding: 14px 22px;
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.cancel-btn,
.confirm-btn {
  padding: 8px 22px;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.cancel-btn {
  background: var(--btn-bg, #f5f5f5);
  color: var(--text-color, #333);
}

.cancel-btn:hover {
  background: var(--border-color, #e8e8e8);
}

.confirm-btn {
  background: var(--primary-color, #409eff);
  color: white;
}

.confirm-btn:hover {
  opacity: 0.9;
}

.confirm-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>

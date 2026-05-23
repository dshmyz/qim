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
            <img :src="getAvatarUrl(selectedGroup?.avatar, '群聊', serverUrl)" :alt="selectedGroup?.name" />
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
                <img :src="member.avatar" :alt="member.name" />
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
            <img :src="getAvatarUrl(selectedGroup?.avatar, '群聊', serverUrl)" :alt="selectedGroup?.name" style="width: 100%; height: 100%;" />
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
            <img :src="getAvatarUrl(selectedGroup?.avatar, '群聊', serverUrl)" :alt="selectedGroup?.name" />
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
          <div class="members-list">
            <div v-for="employee in filteredEmployees" :key="employee.id" class="member-item" :class="{ selected: localSelectedMembers.some(m => m.id === employee.id) }" @click="toggleMember(employee)">
              <div class="member-avatar">
                <img :src="employee.avatar" :alt="employee.name" />
              </div>
              <div class="member-info">
                <div class="member-name">{{ employee.name }}</div>
                <div class="member-position">{{ employee.position || '无职位信息' }}</div>
              </div>
              <div class="member-checkbox">
                <input type="checkbox" v-model="localSelectedMembers" :value="employee" class="checkbox" />
              </div>
            </div>
            <div v-if="filteredEmployees.length === 0" class="empty-state">
              <p>没有找到匹配的成员</p>
            </div>
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
            <img :src="getAvatarUrl(selectedGroup?.avatar, '群聊', serverUrl)" :alt="selectedGroup?.name" />
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
            <img :src="getAvatarUrl(selectedGroup?.avatar, '群聊', serverUrl)" :alt="selectedGroup?.name" />
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
import { getAvatarUrl } from '../../utils/avatar'
import { useServerUrl } from '../../composables/useServerUrl'

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
  addMembersSearchQuery: string
  selectedAddMembers: Employee[]
  editGroupName: string
  editAnnouncementContent: string
  currentUserId?: string | number
  formatTime: (date: string) => string
}

const props = defineProps<Props>()

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

const filteredEmployees = computed(() => {
  if (!localSearchQuery.value) return props.allEmployees
  const query = localSearchQuery.value.toLowerCase()
  return props.allEmployees.filter(emp => 
    emp.name.toLowerCase().includes(query) || 
    (emp.username && emp.username.toLowerCase().includes(query))
  )
})

const toggleMember = (employee: Employee) => {
  const index = localSelectedMembers.value.findIndex(m => m.id === employee.id)
  if (index === -1) {
    localSelectedMembers.value.push(employee)
  } else {
    localSelectedMembers.value.splice(index, 1)
  }
}
</script>

<style scoped>
.add-members-modal {
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

.add-members-content {
  background: var(--modal-bg, #fff);
  border-radius: 12px;
  width: 500px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
}

.add-members-header {
  padding: 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.add-members-header h3 {
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

.add-members-body {
  padding: 20px;
  overflow-y: auto;
  flex: 1;
}

.group-info {
  display: flex;
  gap: 16px;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 0;
}

.group-avatar img {
  width: 48px;
  height: 48px;
  border-radius: 50%;
}

.group-details {
  flex: 1;
}

.group-name {
  font-weight: 500;
  color: var(--text-color, #333);
}

.group-members-count {
  font-size: 12px;
  color: var(--text-secondary, #999);
}

.search-section {
  margin-bottom: 16px;
}

.search-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border-color, #ddd);
  border-radius: 4px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 0;
}

.selected-count {
  font-size: 12px;
  color: var(--text-secondary, #999);
}

.members-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
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
  color: var(--text-color, #333);
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

.group-name-edit-section {
  margin-top: 16px;
}

.group-name-input {
  width: 100%;
  padding: 12px;
  border: 1px solid var(--border-color, #ddd);
  border-radius: 4px;
  font-size: 14px;
}

.group-name-tip {
  font-size: 12px;
  color: var(--text-secondary, #999);
  margin-top: 8px;
}

.announcement-edit-section {
  margin-top: 16px;
}

.announcement-textarea {
  width: 100%;
  padding: 12px;
  border: 1px solid var(--border-color, #ddd);
  border-radius: 4px;
  resize: vertical;
  font-family: inherit;
  font-size: 14px;
}

.announcement-tip {
  font-size: 12px;
  color: var(--text-secondary, #999);
  margin-top: 8px;
}

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
  color: var(--text-color, #333);
  font-size: 14px;
}

.add-members-footer {
  padding: 16px 20px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.cancel-btn,
.confirm-btn {
  padding: 8px 24px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.cancel-btn {
  background: var(--btn-bg, #f5f5f5);
  color: var(--text-color, #333);
}

.confirm-btn {
  background: var(--primary-color, #409eff);
  color: white;
}

.confirm-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>

<template>
  <div v-if="visible" class="share-modal" @click="close">
    <div class="share-modal-content" @click.stop>
      <div class="share-modal-header">
        <h3>分享{{ shareType === 'file' ? '文件' : shareType === 'note' ? '笔记' : shareType === 'message' ? '消息' : '便签' }}</h3>
        <button class="close-btn" @click="close">×</button>
      </div>
      <div class="share-modal-body">
        <div class="share-search-box">
          <input
            v-model="searchQuery"
            type="text"
            class="share-search-input"
            placeholder="搜索用户或群聊..."
          />
          <i class="fas fa-search share-search-icon"></i>
        </div>
        
        <div class="share-tabs">
          <button 
            class="share-tab" 
            :class="{ active: activeTab === 'users' }"
            @click="activeTab = 'users'"
          >
            用户
          </button>
          <button 
            class="share-tab" 
            :class="{ active: activeTab === 'groups' }"
            @click="activeTab = 'groups'"
          >
            群聊
          </button>
        </div>
        
        <div v-if="activeTab === 'users'" class="share-list">
          <div 
            v-for="user in filteredUsers" 
            :key="user.id"
            class="share-item"
            :class="{ selected: selectedUsers.includes(user.id) }"
            @click="toggleUserSelection(user.id)"
          >
            <img :src="user.avatar" :alt="user.name" class="share-item-avatar" />
            <div class="share-item-info">
              <div class="share-item-name">{{ user.name }}</div>
              <div class="share-item-desc">{{ user.department || '无部门' }}</div>
            </div>
            <div class="share-item-checkbox">
              <i v-if="selectedUsers.includes(user.id)" class="fas fa-check"></i>
            </div>
          </div>
          <div v-if="filteredUsers.length === 0" class="empty-share">
            没有找到匹配的用户
          </div>
        </div>
        
        <div v-else-if="activeTab === 'groups'" class="share-list">
          <div 
            v-for="group in filteredGroups" 
            :key="group.id"
            class="share-item"
            :class="{ selected: selectedGroups.includes(group.id) }"
            @click="toggleGroupSelection(group.id)"
          >
            <img :src="group.avatar" :alt="group.name" class="share-item-avatar" />
            <div class="share-item-info">
              <div class="share-item-name">{{ group.name }}</div>
              <div class="share-item-desc">{{ group.members.length }} 成员</div>
            </div>
            <div class="share-item-checkbox">
              <i v-if="selectedGroups.includes(group.id)" class="fas fa-check"></i>
            </div>
          </div>
          <div v-if="filteredGroups.length === 0" class="empty-share">
            没有找到匹配的群聊
          </div>
        </div>
      </div>
      <div class="share-modal-footer">
        <button class="cancel-btn" @click="close">取消</button>
        <button 
          class="confirm-btn" 
          :disabled="selectedUsers.length === 0 && selectedGroups.length === 0"
          @click="confirm"
        >
          分享
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  shareType: {
    type: String,
    required: true
  },
  users: {
    type: Array,
    default: () => []
  },
  groups: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['close', 'confirm'])

const searchQuery = ref('')
const activeTab = ref('users')
const selectedUsers = ref<string[]>([])
const selectedGroups = ref<string[]>([])

const filteredUsers = computed(() => {
  if (!searchQuery.value) return props.users
  const query = searchQuery.value.toLowerCase()
  return props.users.filter(user => 
    user.name.toLowerCase().includes(query) ||
    (user.department && user.department.toLowerCase().includes(query))
  )
})

const filteredGroups = computed(() => {
  if (!searchQuery.value) return props.groups
  const query = searchQuery.value.toLowerCase()
  return props.groups.filter(group => 
    group.name.toLowerCase().includes(query)
  )
})

const close = () => {
  emit('close')
}

const confirm = () => {
  emit('confirm', {
    users: selectedUsers.value,
    groups: selectedGroups.value
  })
}

const toggleUserSelection = (userId: string) => {
  const index = selectedUsers.value.indexOf(userId)
  if (index > -1) {
    selectedUsers.value.splice(index, 1)
  } else {
    selectedUsers.value.push(userId)
  }
}

const toggleGroupSelection = (groupId: string) => {
  const index = selectedGroups.value.indexOf(groupId)
  if (index > -1) {
    selectedGroups.value.splice(index, 1)
  } else {
    selectedGroups.value.push(groupId)
  }
}

watch(() => props.visible, (newVal) => {
  if (newVal) {
    selectedUsers.value = []
    selectedGroups.value = []
    searchQuery.value = ''
    activeTab.value = 'users'
  }
})
</script>

<style scoped>
.share-modal {
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

.share-modal-content {
  background: var(--card-bg, #ffffff);
  border-radius: 8px;
  width: 480px;
  max-width: 90%;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.share-modal-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color, #e8e8e8);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.share-modal-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color, #333);
}

.close-btn {
  background: none;
  border: none;
  font-size: 20px;
  cursor: pointer;
  color: var(--text-secondary, #999);
  padding: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: all 0.2s;
}

.close-btn:hover {
  background: var(--hover-color, #f5f5f5);
  color: var(--text-color, #333);
}

.share-modal-body {
  padding: 20px;
  flex: 1;
  overflow-y: auto;
}

.share-search-box {
  position: relative;
  margin-bottom: 16px;
}

.share-search-input {
  width: 100%;
  padding: 8px 32px 8px 12px;
  border: 1px solid var(--border-color, #d9d9d9);
  border-radius: 4px;
  font-size: 14px;
  transition: all 0.3s;
  background: var(--input-bg, #ffffff);
  color: var(--text-color, #333);
}

.share-search-input:focus {
  outline: none;
  border-color: var(--primary-color, #1890ff);
  box-shadow: 0 0 0 2px var(--primary-light, rgba(24, 144, 255, 0.2));
}

.share-search-icon {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-secondary, #999);
  font-size: 14px;
}

.share-tabs {
  display: flex;
  margin-bottom: 16px;
  border-bottom: 1px solid var(--border-color, #e8e8e8);
}

.share-tab {
  flex: 1;
  padding: 8px 16px;
  background: none;
  border: none;
  font-size: 14px;
  color: var(--text-secondary, #666);
  cursor: pointer;
  transition: all 0.3s;
  border-bottom: 2px solid transparent;
}

.share-tab:hover {
  color: var(--primary-color, #1890ff);
}

.share-tab.active {
  color: var(--primary-color, #1890ff);
  border-bottom-color: var(--primary-color, #1890ff);
}

.share-list {
  max-height: 300px;
  overflow-y: auto;
}

.share-item {
  display: flex;
  align-items: center;
  padding: 12px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
  margin-bottom: 8px;
}

.share-item:hover {
  background: var(--hover-color, #f5f5f5);
}

.share-item.selected {
  background: var(--primary-light, #e6f7ff);
  border: 1px solid var(--primary-color, #91d5ff);
}

.share-item-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  margin-right: 12px;
  object-fit: cover;
}

.share-item-info {
  flex: 1;
  min-width: 0;
}

.share-item-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color, #333);
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.share-item-desc {
  font-size: 12px;
  color: var(--text-secondary, #999);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.share-item-checkbox {
  width: 20px;
  height: 20px;
  border: 1px solid var(--border-color, #d9d9d9);
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary-color, #1890ff);
  font-size: 12px;
  transition: all 0.2s;
}

.share-item.selected .share-item-checkbox {
  background: var(--primary-color, #1890ff);
  border-color: var(--primary-color, #1890ff);
  color: white;
}

.empty-share {
  text-align: center;
  padding: 40px 0;
  color: var(--text-secondary, #999);
  font-size: 14px;
}

.share-modal-footer {
  padding: 16px 20px;
  border-top: 1px solid var(--border-color, #e8e8e8);
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
}

.cancel-btn {
  padding: 6px 16px;
  border: 1px solid var(--border-color, #d9d9d9);
  border-radius: 4px;
  background: var(--card-bg, white);
  color: var(--text-color, #333);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.3s;
}

.cancel-btn:hover {
  border-color: var(--primary-color, #1890ff);
  color: var(--primary-color, #1890ff);
}

.confirm-btn {
  padding: 6px 16px;
  border: 1px solid var(--primary-color, #1890ff);
  border-radius: 4px;
  background: var(--primary-color, #1890ff);
  color: white;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.3s;
}

.confirm-btn:hover:not(:disabled) {
  background: var(--active-color, #40a9ff);
  border-color: var(--active-color, #40a9ff);
}

.confirm-btn:disabled {
  background: var(--color-gray-100, #f0f0f0);
  border-color: var(--border-color, #d9d9d9);
  color: var(--text-secondary, #999);
  cursor: not-allowed;
}
</style>

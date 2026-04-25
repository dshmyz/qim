<template>
  <div v-if="visible" class="user-profile-modal" @click="$emit('close')">
    <div class="user-profile-content" @click.stop>
      <div class="user-profile-header">
        <h3>个人信息</h3>
        <button class="close-btn" @click="$emit('close')">×</button>
      </div>
      <div class="user-profile-body">
        <div class="profile-avatar">
          <img 
            :src="avatarUrl"
            :alt="currentUser?.username || 'avatar'"
            @click="$emit('avatarClick')"
            class="avatar-clickable"
          />
          <input type="file" accept="image/*" class="avatar-input" @change="$emit('avatarChange', $event)" />
        </div>
        <div class="profile-info">
          <div class="info-item">
            <label>昵称</label>
            <input type="text" v-model="localProfile.nickname" class="profile-input" />
          </div>
          <div class="info-item">
            <label>账号</label>
            <span class="profile-value">{{ localProfile.username }}</span>
          </div>
          <div class="info-item">
            <label>签名</label>
            <textarea v-model="localProfile.signature" class="profile-textarea" placeholder="输入个人签名"></textarea>
          </div>
          <div class="info-item">
            <label>部门</label>
            <span class="profile-value">无</span>
          </div>
          <div class="info-item">
            <label>ID</label>
            <span class="profile-value">{{ localProfile.id }}</span>
          </div>
          <div class="info-item">
            <label>加入时间</label>
            <span class="profile-value">{{ localProfile.joinDate }}</span>
          </div>
        </div>
      </div>
      <div class="user-profile-footer">
        <button class="cancel-btn" @click="$emit('close')">关闭</button>
        <button class="save-btn" @click="$emit('save', { ...localProfile })">保存</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'

interface Props {
  visible: boolean
  currentUser?: { username?: string; avatar?: string; id?: string | number }
  serverUrl: string
  profile: { nickname?: string; signature?: string; username?: string; id?: string | number; joinDate?: string }
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'close': []
  'save': [profile: any]
  'avatarClick': []
  'avatarChange': [event: Event]
}>()

const localProfile = ref({ ...props.profile })

watch(() => props.visible, (val) => {
  if (val) {
    localProfile.value = { ...props.profile }
  }
})

const avatarUrl = computed(() => {
  if (!props.currentUser?.avatar) return 'https://api.dicebear.com/7.x/avataaars/svg?seed=me'
  if (props.currentUser.avatar.startsWith('http')) return props.currentUser.avatar
  return props.serverUrl + props.currentUser.avatar
})
</script>

<style scoped>
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
}

.user-profile-header {
  padding: 20px;
  border-bottom: 1px solid var(--border-color, #eee);
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
  overflow-y: auto;
  flex: 1;
}

.profile-avatar {
  text-align: center;
  margin-bottom: 20px;
  position: relative;
}

.avatar-clickable {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  cursor: pointer;
}

.avatar-input {
  position: absolute;
  top: 0;
  left: 0;
  width: 80px;
  height: 80px;
  opacity: 0;
  cursor: pointer;
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
  min-height: 60px;
  resize: vertical;
}

.profile-value {
  color: var(--text-color, #333);
}

.user-profile-footer {
  padding: 16px 20px;
  border-top: 1px solid var(--border-color, #eee);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.cancel-btn,
.save-btn {
  padding: 8px 24px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.cancel-btn {
  background: var(--btn-bg, #f5f5f5);
  color: var(--text-color, #333);
}

.save-btn {
  background: var(--primary-color, #409eff);
  color: white;
}
</style>

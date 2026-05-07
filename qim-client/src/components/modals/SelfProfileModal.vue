<template>
  <div v-if="visible" class="self-profile-modal" @click="$emit('close')">
    <div class="self-profile-content" @click.stop>
      <div class="modal-header">
        <h3>个人信息</h3>
        <button class="close-btn" @click="$emit('close')">
          <i class="fas fa-times"></i>
        </button>
      </div>
      
      <div class="modal-body">
        <div class="profile-layout">
          <div class="avatar-section">
            <div class="avatar-wrapper">
              <img 
                :src="avatarUrl"
                :alt="currentUser?.username || 'avatar'"
                class="avatar-image"
              />
              <div class="avatar-overlay" @click="triggerAvatarUpload">
                <i class="fas fa-camera"></i>
                <span>更换头像</span>
              </div>
            </div>
            <input 
              ref="avatarInputRef"
              type="file" 
              accept="image/*" 
              class="avatar-input-hidden" 
              @change="handleAvatarSelect" 
            />
            <p class="avatar-hint">点击头像可更换，支持 JPG、PNG 格式</p>
          </div>
          
          <div class="form-section">
            <div class="form-group">
              <label class="form-label">姓名</label>
              <div class="form-value readonly">{{ localProfile.nickname }}</div>
            </div>
            
            <div class="form-group">
              <label class="form-label">账号</label>
              <div class="form-value readonly">{{ localProfile.username }}</div>
            </div>
            
            <div class="form-group">
              <label class="form-label">签名</label>
              <textarea 
                v-model="localProfile.signature" 
                class="form-textarea" 
                placeholder="输入个人签名，让大家更了解你"
                rows="3"
              ></textarea>
            </div>
            
            <div class="form-row">
              <div class="form-group half">
                <label class="form-label">部门</label>
                <div class="form-value readonly">{{ localProfile.department || '未设置' }}</div>
              </div>
              <div class="form-group half">
                <label class="form-label">ID</label>
                <div class="form-value readonly">{{ localProfile.id }}</div>
              </div>
            </div>
            
            <div class="form-group">
              <label class="form-label">加入时间</label>
              <div class="form-value readonly">{{ localProfile.joinDate }}</div>
            </div>
          </div>
        </div>
      </div>
      
      <div class="modal-footer">
        <button class="btn btn-secondary" @click="$emit('close')">取消</button>
        <button class="btn btn-primary" @click="handleSave">保存</button>
      </div>
    </div>
    
    <AvatarCropper
      v-if="showCropper"
      :image-url="pendingImageUrl"
      @confirm="handleCropConfirm"
      @cancel="handleCropCancel"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { generateAvatar, isAbsoluteUrl } from '../../utils/avatar'
import AvatarCropper from './AvatarCropper.vue'

interface Props {
  visible: boolean
  currentUser?: { username?: string; avatar?: string; id?: string | number; department?: string }
  serverUrl: string
  profile: { 
    nickname?: string
    signature?: string
    username?: string
    id?: string | number
    joinDate?: string
    department?: string
  }
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'close': []
  'save': [profile: any]
}>()

const localProfile = ref({ ...props.profile })
const avatarInputRef = ref<HTMLInputElement | null>(null)
const showCropper = ref(false)
const pendingImageUrl = ref('')

watch(() => props.visible, (val) => {
  if (val) {
    localProfile.value = { ...props.profile }
  }
})

const avatarUrl = computed(() => {
  if (!props.currentUser?.avatar) return generateAvatar(props.currentUser?.username || 'me')
  if (isAbsoluteUrl(props.currentUser.avatar)) return props.currentUser.avatar
  return props.serverUrl + props.currentUser.avatar
})

const triggerAvatarUpload = () => {
  avatarInputRef.value?.click()
}

const handleAvatarSelect = (event: Event) => {
  const input = event.target as HTMLInputElement
  if (input.files && input.files.length > 0) {
    const file = input.files[0]
    
    if (!file.type.startsWith('image/')) {
      return
    }
    
    if (file.size > 5 * 1024 * 1024) {
      return
    }
    
    pendingImageUrl.value = URL.createObjectURL(file)
    showCropper.value = true
    input.value = ''
  }
}

const handleCropConfirm = (croppedFile: File) => {
  showCropper.value = false
  if (pendingImageUrl.value) {
    URL.revokeObjectURL(pendingImageUrl.value)
    pendingImageUrl.value = ''
  }
  emit('save', { ...localProfile.value, avatarFile: croppedFile })
}

const handleCropCancel = () => {
  showCropper.value = false
  if (pendingImageUrl.value) {
    URL.revokeObjectURL(pendingImageUrl.value)
    pendingImageUrl.value = ''
  }
}

const handleSave = () => {
  emit('save', { ...localProfile.value })
}
</script>

<style scoped>
.self-profile-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  animation: fadeIn 0.2s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.self-profile-content {
  background: var(--modal-bg, #ffffff);
  border-radius: 16px;
  width: 680px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  animation: slideUp 0.3s ease-out;
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.modal-header {
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-color, #e5e7eb);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.modal-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color, #111827);
}

.close-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: var(--hover-color, #f3f4f6);
  border-radius: 8px;
  cursor: pointer;
  color: var(--text-secondary, #6b7280);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.close-btn:hover {
  background: var(--active-color, #e5e7eb);
  color: var(--text-color, #111827);
}

.modal-body {
  padding: 24px;
  overflow-y: auto;
  flex: 1;
}

.profile-layout {
  display: grid;
  grid-template-columns: 200px 1fr;
  gap: 32px;
}

.avatar-section {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.avatar-wrapper {
  position: relative;
  width: 120px;
  height: 120px;
  border-radius: 50%;
  overflow: hidden;
  cursor: pointer;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.avatar-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.2s;
  color: #ffffff;
}

.avatar-wrapper:hover .avatar-overlay {
  opacity: 1;
}

.avatar-overlay i {
  font-size: 24px;
  margin-bottom: 4px;
}

.avatar-overlay span {
  font-size: 12px;
}

.avatar-input-hidden {
  display: none;
}

.avatar-hint {
  margin-top: 12px;
  font-size: 12px;
  color: var(--text-secondary, #6b7280);
  text-align: center;
}

.form-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.form-group.half {
  flex: 1;
}

.form-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary, #6b7280);
}

.form-input,
.form-textarea {
  padding: 10px 12px;
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 8px;
  font-size: 14px;
  color: var(--text-color, #111827);
  background: var(--input-bg, #ffffff);
  transition: all 0.2s;
}

.form-input:focus,
.form-textarea:focus {
  outline: none;
  border-color: var(--primary-color, #3b82f6);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-textarea {
  resize: vertical;
  min-height: 80px;
  line-height: 1.5;
}

.form-value.readonly {
  padding: 10px 12px;
  background: var(--secondary-color, #f9fafb);
  border-radius: 8px;
  font-size: 14px;
  color: var(--text-color, #111827);
}

.modal-footer {
  padding: 16px 24px;
  border-top: 1px solid var(--border-color, #e5e7eb);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.btn {
  padding: 10px 24px;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-secondary {
  background: var(--secondary-color, #f3f4f6);
  color: var(--text-color, #374151);
}

.btn-secondary:hover {
  background: var(--hover-color, #e5e7eb);
}

.btn-primary {
  background: var(--primary-color, #3b82f6);
  color: #ffffff;
}

.btn-primary:hover {
  background: var(--active-color, #2563eb);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

@media (max-width: 768px) {
  .self-profile-content {
    width: 95%;
    max-height: 90vh;
  }
  
  .profile-layout {
    grid-template-columns: 1fr;
    gap: 24px;
  }
  
  .avatar-section {
    padding-bottom: 16px;
    border-bottom: 1px solid var(--border-color, #e5e7eb);
  }
  
  .form-row {
    grid-template-columns: 1fr;
  }
}

</style>

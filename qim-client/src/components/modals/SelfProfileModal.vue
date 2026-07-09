<template>
  <ModalContainer
    :visible="visible"
    title="个人信息"
    width="680px"
    :overlay-style="{ backdropFilter: 'blur(4px)', webkitBackdropFilter: 'blur(4px)' }"
    @close="$emit('close')"
    @cancel="$emit('close')"
  >
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

    <template #footer>
      <button class="btn btn-secondary" @click="$emit('close')">取消</button>
      <button class="btn btn-primary" @click="handleSave">保存</button>
    </template>

    <!-- AvatarCropper 放在 modal 的 stacking context 内，确保层级正确 -->
    <AvatarCropper
      v-if="showCropper"
      :image-url="pendingImageUrl"
      @confirm="handleCropConfirm"
      @cancel="handleCropCancel"
    />
  </ModalContainer>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { generateAvatar, isAbsoluteUrl } from '../../utils/avatar'
import AvatarCropper from './AvatarCropper.vue'
import ModalContainer from '../shared/ModalContainer.vue'

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

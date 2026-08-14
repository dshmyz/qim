<template>
  <div v-if="visible" class="self-profile-modal" @click="$emit('close')">
    <div class="self-profile-content" @click.stop>
      <!-- 关闭按钮 -->
      <button class="modal-close-btn" @click="$emit('close')">
        <i class="fas fa-times"></i>
      </button>

      <!-- Hero 渐变头部 -->
      <div class="self-hero">
        <div class="self-avatar-wrapper" @click="triggerAvatarUpload">
          <img
            :src="avatarUrl"
            :alt="currentUser?.username || 'avatar'"
            class="self-avatar-img"
          />
          <div class="avatar-overlay">
            <i class="fas fa-camera"></i>
          </div>
        </div>
        <input
          ref="avatarInputRef"
          type="file"
          accept="image/*"
          class="avatar-input-hidden"
          @change="handleAvatarSelect"
        />
        <div class="self-hero-name">{{ localProfile.nickname }}</div>
        <div class="self-hero-account">@{{ localProfile.username }}</div>
        <div class="self-signature-block">
          <textarea
            v-model="localProfile.signature"
            class="self-signature-input"
            placeholder="写一句签名吧…"
            rows="1"
            maxlength="100"
          ></textarea>
        </div>
      </div>

      <!-- 信息区 -->
      <div class="self-info">
        <div class="info-item">
          <span class="info-icon"><i class="fas fa-at"></i></span>
          <div class="info-content">
            <label>账号</label>
            <span class="info-value">{{ localProfile.username || '未设置' }}</span>
          </div>
        </div>
        <div class="info-item">
          <span class="info-icon"><i class="fas fa-building"></i></span>
          <div class="info-content">
            <label>部门</label>
            <span class="info-value" :class="{ 'info-value-empty': !localProfile.department }">{{ localProfile.department || '未设置' }}</span>
          </div>
        </div>
        <div class="info-item">
          <span class="info-icon"><i class="fas fa-mobile-alt"></i></span>
          <div class="info-content">
            <label>手机</label>
            <span class="info-value" :class="{ 'info-value-empty': !localProfile.phone }">{{ localProfile.phone || '未绑定' }}</span>
          </div>
        </div>
        <div class="info-item">
          <span class="info-icon"><i class="fas fa-envelope"></i></span>
          <div class="info-content">
            <label>邮箱</label>
            <span class="info-value" :class="{ 'info-value-empty': !localProfile.email }">{{ localProfile.email || '未绑定' }}</span>
          </div>
        </div>
        <div class="info-item">
          <span class="info-icon"><i class="fas fa-venus-mars"></i></span>
          <div class="info-content">
            <label>性别</label>
            <span class="info-value">{{ formatGender(localProfile.gender) }}</span>
          </div>
        </div>
        <div class="info-item">
          <span class="info-icon"><i class="fas fa-calendar"></i></span>
          <div class="info-content">
            <label>加入时间</label>
            <span class="info-value info-value-muted">{{ localProfile.joinDate || '—' }}</span>
          </div>
        </div>
      </div>

      <!-- 操作栏 -->
      <div class="self-profile-footer">
        <button class="action-btn primary" @click="handleSave">
          <i class="fas fa-check"></i>
          <span>保存</span>
        </button>
        <button class="action-btn secondary" @click="$emit('close')">
          <span>取消</span>
        </button>
      </div>

      <!-- AvatarCropper -->
      <AvatarCropper
        v-if="showCropper"
        :image-url="pendingImageUrl"
        @confirm="handleCropConfirm"
        @cancel="handleCropCancel"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { generateAvatar, isAbsoluteUrl } from '../../utils/avatar'
import AvatarCropper from './AvatarCropper.vue'

interface Props {
  visible: boolean
  currentUser?: { username?: string; avatar?: string; id?: string | number }
  serverUrl: string
  profile: {
    nickname?: string
    signature?: string
    username?: string
    phone?: string
    email?: string
    gender?: string
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

const formatGender = (gender?: string): string => {
  switch (gender) {
    case 'male': return '男'
    case 'female': return '女'
    default: return '保密'
  }
}

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
    if (!file.type.startsWith('image/')) return
    if (file.size > 5 * 1024 * 1024) return
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
/* ---- 模态框容器 ---- */
.self-profile-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.45);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.self-profile-content {
  background-color: var(--card-bg, #fff);
  border-radius: 16px;
  width: 400px;
  max-width: 90%;
  max-height: 85vh;
  overflow: hidden;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15), 0 0 0 1px rgba(0, 0, 0, 0.05);
  display: flex;
  flex-direction: column;
  position: relative;
}

/* ---- 关闭按钮 ---- */
.modal-close-btn {
  position: absolute;
  top: 12px;
  right: 12px;
  z-index: 2;
  width: 30px;
  height: 30px;
  border: none;
  background: rgba(255, 255, 255, 0.85);
  color: #666;
  font-size: var(--font-size-xxs);
  cursor: pointer;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.15);
  transition: all 0.2s;
}

.modal-close-btn:hover {
  background: #fff;
  color: #333;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

/* ---- Hero 渐变头部 ---- */
.self-hero {
  position: relative;
  padding: 32px 24px 20px;
  text-align: center;
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--active-color, #4f6ef7) 100%);
  overflow: hidden;
}

.self-hero::before {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at 30% 20%, rgba(255,255,255,0.15), transparent 60%);
  pointer-events: none;
}

.self-avatar-wrapper {
  position: relative;
  width: 72px;
  height: 72px;
  border-radius: 50%;
  overflow: hidden;
  cursor: pointer;
  display: inline-block;
  margin-bottom: 14px;
  border: 3px solid rgba(255, 255, 255, 0.5);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
  transition: transform 0.3s ease;
}

.self-avatar-wrapper:hover {
  transform: scale(1.06);
}

.self-avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.avatar-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.2s;
  color: #fff;
  font-size: var(--font-size-lg);
}

.self-avatar-wrapper:hover .avatar-overlay {
  opacity: 1;
}

.avatar-input-hidden {
  display: none;
}

.self-hero-name {
  font-size: var(--font-size-lg);
  font-weight: 700;
  color: #fff;
  text-shadow: 0 1px 3px rgba(0,0,0,0.15);
  margin-bottom: 2px;
}

.self-hero-account {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.7);
  margin-bottom: 10px;
}

.self-signature-block {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 0 8px;
}

.self-signature-input {
  flex: 1;
  max-width: 280px;
  padding: 4px 10px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
  font-size: var(--font-size-xs);
  line-height: 1.5;
  resize: none;
  font-family: inherit;
  text-align: center;
  transition: all 0.2s;
  box-sizing: border-box;
}

.self-signature-input::placeholder {
  color: rgba(255, 255, 255, 0.5);
}

.self-signature-input:focus {
  outline: none;
  border-color: rgba(255, 255, 255, 0.4);
  background: rgba(255, 255, 255, 0.15);
}

/* ---- 信息区 ---- */
.self-info {
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow-y: auto;
  flex: 1;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  transition: background-color 0.15s;
}

.info-item:hover {
  background: var(--hover-color, rgba(0,0,0,0.03));
}

.info-icon {
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

.info-content {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
  flex: 1;
}

.info-content label {
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary, #999);
  font-weight: 500;
}

.info-content .info-value {
  font-size: var(--font-size-xs);
  color: var(--text-color, #333);
  font-weight: 500;
  word-break: break-all;
  line-height: 1.4;
}

.info-content .info-value-empty {
  color: var(--text-secondary, #d1d5db);
}

.info-content .info-value-muted {
  color: var(--text-secondary, #999);
  font-weight: 400;
  font-size: var(--font-size-xxs);
}

/* ---- 操作栏 ---- */
.self-profile-footer {
  padding: 16px 20px;
  border-top: 1px solid var(--border-color, rgba(0,0,0,0.06));
  display: flex;
  gap: 8px;
}

.action-btn {
  flex: 1;
  padding: 10px 16px;
  border: none;
  border-radius: 10px;
  font-size: var(--font-size-xs);
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.action-btn i {
  font-size: var(--font-size-xs);
}

.action-btn.primary {
  background: var(--primary-color, #3b82f6);
  color: #fff;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.25);
}

.action-btn.primary:hover {
  background: var(--active-color, #2563eb);
  box-shadow: 0 4px 14px rgba(59, 130, 246, 0.35);
  transform: translateY(-1px);
}

.action-btn.secondary {
  background: var(--hover-color, rgba(0,0,0,0.04));
  color: var(--text-color, #333);
}

.action-btn.secondary:hover {
  background: var(--border-color, rgba(0,0,0,0.08));
  transform: translateY(-1px);
}

/* 高雅紫主题 */
[data-theme="elegant-purple"] .action-btn.primary {
  background: var(--primary-color, #75629a);
}
[data-theme="elegant-purple"] .action-btn.primary:hover {
  background: var(--active-color, #665486);
}

/* 中国红主题 */
[data-theme="chinesered"] .action-btn.primary {
  background: var(--primary-color, #c41e3a);
}
[data-theme="chinesered"] .action-btn.primary:hover {
  background: var(--active-color, #a31d32);
}
</style>

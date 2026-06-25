<template>
  <div class="message-bubble mini-app-message" :class="{ self: isSelf }">
    <div class="mini-app-info" @click="openMiniApp">
      <div class="mini-app-icon-container">
        <img
          v-if="miniAppData?.icon && !iconError"
          :src="miniAppData.icon"
          class="mini-app-icon"
          :alt="miniAppData?.name"
          @error="handleIconError"
        />
        <div v-else class="mini-app-icon mini-app-icon-fallback" :style="{ background: iconBgColor }">
          {{ iconInitial }}
        </div>
        <div class="mini-app-type-label">小程序</div>
      </div>
      <div class="mini-app-details">
        <div class="mini-app-name">{{ miniAppData?.name }}</div>
        <div class="mini-app-description">{{ miniAppData?.description }}</div>
      </div>
      <div class="mini-app-arrow">
        <i class="fas fa-chevron-right"></i>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { getAvatarColor, getInitial } from '../../utils/avatar'

const props = defineProps<{
  miniAppData?: {
    icon: string
    name: string
    description: string
  }
  isSelf?: boolean
}>()

const emit = defineEmits<{
  open: [data: any]
}>()

const iconError = ref(false)

const iconInitial = computed(() => getInitial(props.miniAppData?.name || '小'))
const iconBgColor = computed(() => getAvatarColor(props.miniAppData?.name || '小程序'))

const handleIconError = () => {
  iconError.value = true
}

const openMiniApp = () => {
  emit('open', props.miniAppData)
}
</script>

<style scoped>
.mini-app-message {
  background: var(--sidebar-bg);
  border-radius: 12px;
  padding: 16px;
  width: fit-content;
  min-width: 260px;
  max-width: 100%;
  transition: all 0.2s ease;
  box-sizing: border-box;
  position: relative;
  overflow: hidden;
  cursor: pointer;
}

.mini-app-message::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: linear-gradient(90deg, #4facfe, #00f2fe, #43e97b);
  opacity: 0;
  transition: opacity 0.2s ease;
}

.mini-app-message:hover {
  transform: translateY(-1px);
}

.mini-app-message:hover::before {
  opacity: 1;
}

.mini-app-info {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.mini-app-icon-container {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}

.mini-app-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  object-fit: cover;
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  border: none;
  display: block;
}

.mini-app-icon-fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 600;
  color: #fff;
  user-select: none;
}

.mini-app-type-label {
  font-size: 10px;
  font-weight: 500;
  color: var(--text-secondary);
  background: var(--hover-color);
  padding: 2px 8px;
  border-radius: 4px;
  display: block;
  text-align: center;
  white-space: nowrap;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.mini-app-details {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.mini-app-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  text-align: left;
  letter-spacing: -0.01em;
  line-height: 1.4;
}

.mini-app-description {
  font-size: 12px;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  text-align: left;
  font-weight: 500;
}

.mini-app-arrow {
  color: var(--text-secondary);
  font-size: 12px;
  flex-shrink: 0;
  transition: all 0.2s ease;
  align-self: center;
  margin-left: 4px;
}

.mini-app-message:hover .mini-app-arrow {
  color: var(--primary-color);
  transform: translateX(4px);
}

/* 自己的小程序消息样式：浅色主色背景 + 深色文字 */
.mini-app-message.self {
  background: var(--hover-color);
  background: color-mix(in srgb, var(--primary-color), white 88%);
  border: none;
  color: var(--text-color);
}

.mini-app-message.self::before {
  background: var(--primary-color);
  background: linear-gradient(90deg, var(--primary-color), color-mix(in srgb, var(--primary-color), white 40%), var(--primary-color));
  opacity: 1;
}

.mini-app-message.self .mini-app-name {
  color: var(--text-color);
  font-weight: 600;
}

.mini-app-message.self .mini-app-description {
  color: var(--text-secondary);
}

.mini-app-message.self .mini-app-icon {
  background: rgba(128, 128, 128, 0.1);
}

.mini-app-message.self .mini-app-type-label {
  background: var(--hover-color);
  color: var(--text-secondary);
}

.mini-app-message.self .mini-app-arrow {
  color: var(--text-secondary);
}

.mini-app-message.self:hover .mini-app-arrow {
  color: var(--primary-color);
}

/* 深色主题：纯主色背景 + 白色文字 */
[data-theme="elegant-dark"] .mini-app-message.self {
  background: var(--primary-color);
  color: #ffffff;
}

[data-theme="elegant-dark"] .mini-app-message.self .mini-app-name {
  color: #ffffff;
}

[data-theme="elegant-dark"] .mini-app-message.self .mini-app-description {
  color: rgba(255, 255, 255, 0.85);
}

[data-theme="elegant-dark"] .mini-app-message.self .mini-app-icon {
  background: rgba(255, 255, 255, 0.95);
}

[data-theme="elegant-dark"] .mini-app-message.self .mini-app-type-label {
  background: rgba(255, 255, 255, 0.2);
  color: rgba(255, 255, 255, 0.85);
}

[data-theme="elegant-dark"] .mini-app-message.self .mini-app-arrow {
  color: rgba(255, 255, 255, 0.8);
}

[data-theme="elegant-dark"] .mini-app-message.self:hover .mini-app-arrow {
  color: #ffffff;
}
</style>
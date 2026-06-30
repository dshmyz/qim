<template>
  <div class="message-bubble mini-app-message attachment-card" :class="{ self: isSelf }" @click="openMiniApp">
    <div class="attachment-card__icon mini-app-attachment-icon">
      <img
        v-if="displayIcon && !iconError"
        :src="displayIcon"
        class="mini-app-icon"
        :alt="miniAppData?.name"
        @error="handleIconError"
      />
      <div v-else class="mini-app-icon mini-app-icon-fallback" :style="{ background: iconBgColor }">
        {{ iconInitial }}
      </div>
    </div>
    <div class="attachment-card__content">
      <div class="attachment-card__title">{{ miniAppName }}</div>
      <div class="attachment-card__meta">小程序 · 点击打开</div>
    </div>
    <div class="mini-app-arrow attachment-card__action">
      <i class="fas fa-chevron-right"></i>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { getAvatarColor, getInitial, generateAvatar } from '../../utils/avatar'

const props = defineProps<{
  miniAppData?: {
    icon: string
    name: string
    display_name?: string
    displayName?: string
    title?: string
    appName?: string
    app_name?: string
    description: string
  }
  isSelf?: boolean
}>()

const emit = defineEmits<{
  open: [data: any]
}>()

const iconError = ref(false)

const miniAppName = computed(() =>
  props.miniAppData?.display_name ||
  props.miniAppData?.displayName ||
  props.miniAppData?.title ||
  props.miniAppData?.appName ||
  props.miniAppData?.app_name ||
  props.miniAppData?.name ||
  '小程序'
)

const iconInitial = computed(() => getInitial(miniAppName.value))
const iconBgColor = computed(() => getAvatarColor(miniAppName.value))
const legacyDefaultIcon = generateAvatar('default')
const displayIcon = computed(() => {
  const icon = props.miniAppData?.icon || ''
  if (icon === legacyDefaultIcon) return ''
  return icon
})

const handleIconError = () => {
  iconError.value = true
}

const openMiniApp = () => {
  emit('open', props.miniAppData)
}
</script>

<style scoped>
.attachment-card {
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr) 28px;
  align-items: center;
  gap: 12px;
  width: 280px;
  max-width: min(100%, 320px);
  padding: 12px 12px;
  border-radius: 14px;
  background: color-mix(in srgb, var(--sidebar-bg), transparent 4%);
  border: 1px solid color-mix(in srgb, var(--border-color), transparent 20%);
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.06);
  box-sizing: border-box;
  cursor: pointer;
  transition: border-color 0.16s ease, box-shadow 0.16s ease, transform 0.16s ease;
}

.attachment-card:hover {
  border-color: color-mix(in srgb, var(--primary-color), transparent 58%);
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.08);
  transform: translateY(-1px);
}

.attachment-card__icon {
  width: 42px;
  height: 42px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  background: color-mix(in srgb, var(--primary-color), white 88%);
}

.attachment-card__content {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.attachment-card__title {
  font-size: 14px;
  font-weight: 600;
  line-height: 1.35;
  color: var(--text-color);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  letter-spacing: -0.01em;
}

.attachment-card__meta {
  font-size: 12px;
  line-height: 1.35;
  color: var(--text-secondary);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.attachment-card__action {
  width: 28px;
  height: 28px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  background: transparent;
  transition: background 0.16s ease, color 0.16s ease, transform 0.16s ease;
}

.attachment-card:hover .attachment-card__action {
  color: var(--primary-color);
  background: color-mix(in srgb, var(--primary-color), transparent 90%);
}

.mini-app-icon {
  width: 42px;
  height: 42px;
  border-radius: 12px;
  object-fit: cover;
  display: block;
}

.mini-app-icon-fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 17px;
  font-weight: 700;
  color: #fff;
  user-select: none;
}

.mini-app-arrow {
  font-size: 12px;
  flex-shrink: 0;
}

.attachment-card:hover .mini-app-arrow {
  transform: translateX(2px);
}

.mini-app-message.self {
  background: color-mix(in srgb, var(--sidebar-bg), transparent 4%);
  border-color: transparent;
  color: var(--text-color);
}

:global(.message-item.self) .mini-app-message.self {
  background: color-mix(in srgb, var(--sidebar-bg), transparent 4%);
  border-color: transparent;
  color: var(--text-color);
}

[data-theme="elegant-dark"] .attachment-card {
  background: color-mix(in srgb, var(--panel-bg), white 5%);
  border-color: rgba(255, 255, 255, 0.12);
  box-shadow: none;
}

[data-theme="elegant-dark"] .mini-app-message.self {
  background: color-mix(in srgb, var(--panel-bg), white 5%);
  border-color: transparent;
  color: var(--text-color);
}

:global([data-theme="elegant-dark"] .message-item.self) .mini-app-message.self {
  background: color-mix(in srgb, var(--panel-bg), white 5%);
  border-color: transparent;
  color: var(--text-color);
}
</style>

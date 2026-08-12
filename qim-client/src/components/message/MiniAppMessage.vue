<template>
  <AttachmentCard
    class="mini-app-message"
    :is-self="isSelf"
    @click="openMiniApp"
  >
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
    <template #content>
      <div class="mini-app-title">{{ miniAppName }}</div>
      <div class="mini-app-bottom">
        <div class="mini-app-meta">小程序</div>
        <div class="attachment-card__btn">
          <i class="fas fa-chevron-right"></i>
        </div>
      </div>
    </template>
  </AttachmentCard>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { getAvatarColor, getInitial, generateAvatar } from '../../utils/avatar'
import AttachmentCard from './AttachmentCard.vue'

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
.mini-app-message :deep(.attachment-card__icon) {
  background: color-mix(in srgb, var(--primary-color), white 88%);
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

.mini-app-title {
  font-size: 14px;
  font-weight: 600;
  line-height: 1.35;
  color: var(--text-color);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  letter-spacing: -0.01em;
}

.mini-app-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.mini-app-meta {
  min-height: 16px;
  font-size: 12px;
  line-height: 1.35;
  color: var(--text-secondary);
}
</style>

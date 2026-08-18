<template>
  <span class="sticky-note-mini-card" :class="[colorClass, paperClass]" :style="fontStyle">
    <span v-if="title" class="sticky-note-mini-card__title">{{ title }}</span>
    <span class="sticky-note-mini-card__content">{{ content }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { parseStickyStyle, type StickySharePayload } from '../../utils/stickyShare'

const props = withDefaults(defineProps<{
  payload: StickySharePayload
  maxLength?: number
}>(), {
  maxLength: 60
})

const style = computed(() => parseStickyStyle(props.payload.style))
const colorClass = computed(() => `sticky-mini-${style.value.color}`)
const paperClass = computed(() => `sticky-mini-paper-${style.value.paperStyle}`)
const fontStyle = computed(() => style.value.fontFamily
  ? { fontFamily: style.value.fontFamily }
  : undefined)

const title = computed(() => props.payload.name || '')

// 引用场景单行展示：内容折叠空白并截断
const content = computed(() => {
  const c = props.payload.originalContent || props.payload.content || ''
  const collapsed = c.replace(/\s+/g, ' ').trim()
  return collapsed.length > props.maxLength
    ? `${collapsed.slice(0, props.maxLength)}...`
    : collapsed
})
</script>

<style scoped>
.sticky-note-mini-card {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  max-width: 100%;
  padding: 4px 10px;
  border-radius: 6px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.10);
  overflow: hidden;
  vertical-align: middle;
}

.sticky-note-mini-card__title {
  color: var(--sticky-ink, #6d4c41);
  font-size: var(--font-size-xxs);
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 45%;
  flex-shrink: 0;
}

.sticky-note-mini-card__content {
  color: var(--sticky-ink, #6d4c41);
  opacity: 0.85;
  font-size: var(--font-size-xxs);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

[data-theme="elegant-dark"] .sticky-note-mini-card {
  box-shadow: none;
}
</style>

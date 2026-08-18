<template>
  <div
    class="sticky-note"
    :class="[`sticky-mini-${colorClass}`, `sticky-mini-paper-${paperStyleClass}`]"
    :style="noteStyle"
    :data-note-id="note.id"
    @click="$emit('click', note)"
    draggable="true"
    @dragstart="$emit('dragstart', $event, note.id)"
    @dragover.prevent
    @drop="$emit('drop', $event, index)"
  >
    <div class="sticky-note-pin">
      <div class="pin-head"></div>
      <div class="pin-shadow"></div>
    </div>
    <div class="sticky-note-header">
      <div class="sticky-note-title-container">
        <h3 class="sticky-note-title">{{ note.title }}</h3>
        <span v-if="note.reminder" class="sticky-note-reminder">
          <i class="fas fa-bell"></i>
        </span>
      </div>
      <div class="sticky-note-actions">
        <button class="sticky-note-action" @click.stop="$emit('share', note)" title="分享">
          <i class="fas fa-share-alt"></i>
        </button>
        <button class="sticky-note-delete" @click.stop="$emit('delete', note.id)" title="删除">
          <i class="fas fa-trash-alt"></i>
        </button>
      </div>
    </div>
    <div class="sticky-note-content" :style="contentStyle">{{ note.content }}</div>
    <div v-if="parsedTags.length > 0" class="sticky-note-tags">
      <span
        v-for="(tag, i) in parsedTags"
        :key="i"
        class="sticky-note-tag"
        @click.stop="$emit('filter-tag', tag)"
      >
        {{ tag }}
      </span>
    </div>
    <div class="sticky-note-footer">
      <span class="sticky-note-date">{{ formattedDate }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  note: any
  index: number
}>()

defineEmits<{
  click: [note: any]
  share: [note: any]
  delete: [id: string]
  'filter-tag': [tag: string]
  dragstart: [event: DragEvent, id: string]
  drop: [event: DragEvent, index: number]
}>()

const parseStyle = (styleStr: string | undefined) => {
  if (!styleStr || styleStr === '{}') {
    return { color: 'yellow', paperStyle: 'plain', fontFamily: "Arial, 'Microsoft YaHei', sans-serif" }
  }
  try {
    const style = JSON.parse(styleStr)
    return {
      color: style.color || 'yellow',
      paperStyle: style.paperStyle || 'plain',
      fontFamily: style.fontFamily || "Arial, 'Microsoft YaHei', sans-serif"
    }
  } catch {
    return { color: 'yellow', paperStyle: 'plain', fontFamily: "Arial, 'Microsoft YaHei', sans-serif" }
  }
}

const parsedStyle = computed(() => parseStyle(props.note.style))
const colorClass = computed(() => parsedStyle.value.color)
const paperStyleClass = computed(() => parsedStyle.value.paperStyle)

const contentStyle = computed(() => ({
  fontFamily: parsedStyle.value.fontFamily
}))

const rotationSeed = computed(() => {
  const id = typeof props.note.id === 'string' ? props.note.id : String(props.note.id)
  let hash = 0
  for (let i = 0; i < id.length; i++) {
    hash = ((hash << 5) - hash) + id.charCodeAt(i)
    hash |= 0
  }
  return (Math.abs(hash) % 30) / 10 - 1.5
})

const noteStyle = computed(() => ({
  '--note-rotation': `${rotationSeed.value}deg`,
  // 纸张纹理纵向起点：随卡片顶部留白（标题区）下移，与全局纸张类配合
  '--paper-start': '36px'
}))

const parsedTags = computed(() => {
  const tags = props.note.tags
  if (!tags) return []
  if (Array.isArray(tags)) return tags
  if (typeof tags === 'string') {
    try {
      const parsed = JSON.parse(tags)
      return Array.isArray(parsed) ? parsed : []
    } catch {
      return []
    }
  }
  return []
})

const formattedDate = computed(() => {
  const date = new Date(props.note.created_at)
  if (isNaN(date.getTime())) return ''
  return date.toLocaleString('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
})
</script>

<style scoped>
.sticky-note {
  border-radius: 2px;
  padding: 36px 16px 16px;
  box-shadow:
    2px 3px 12px rgba(0, 0, 0, 0.12),
    0 1px 3px rgba(0, 0, 0, 0.06);
  cursor: pointer;
  transition: transform 0.25s cubic-bezier(0.34, 1.56, 0.64, 1), box-shadow 0.25s ease;
  min-height: 180px;
  position: relative;
  overflow: hidden;
  transform: rotate(var(--note-rotation, -0.5deg));
  display: flex;
  flex-direction: column;
  animation: noteAppear 0.4s cubic-bezier(0.34, 1.56, 0.64, 1) both;
  animation-delay: calc(var(--note-index, 0) * 60ms);
}

@keyframes noteAppear {
  from {
    opacity: 0;
    transform: rotate(var(--note-rotation, -0.5deg)) scale(0.85) translateY(16px);
  }
  to {
    opacity: 1;
    transform: rotate(var(--note-rotation, -0.5deg)) scale(1) translateY(0);
  }
}

.sticky-note.deleting {
  animation: noteDisappear 0.3s ease-in forwards;
}

@keyframes noteDisappear {
  from {
    opacity: 1;
    transform: rotate(var(--note-rotation, -0.5deg)) scale(1);
  }
  to {
    opacity: 0;
    transform: rotate(var(--note-rotation, -0.5deg)) scale(0.85) translateY(-16px);
  }
}

.sticky-note:hover {
  transform: rotate(0deg) translateY(-4px);
  box-shadow:
    4px 6px 20px rgba(0, 0, 0, 0.15),
    0 2px 6px rgba(0, 0, 0, 0.08);
}

.sticky-note.dragging {
  opacity: 0.6;
  transform: rotate(3deg) scale(1.05);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
  z-index: 100;
}

/* 图钉样式 */
.sticky-note-pin {
  position: absolute;
  top: 4px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 2;
}

.pin-head {
  width: 18px;
  height: 18px;
  background: radial-gradient(circle at 35% 35%, #ef5350, #c62828);
  border-radius: 50%;
  box-shadow:
    0 2px 6px rgba(198, 40, 40, 0.45),
    inset 0 -1px 2px rgba(0, 0, 0, 0.2),
    inset 0 1px 1px rgba(255, 255, 255, 0.3);
  position: relative;
  transition: transform 0.25s ease;
}

.pin-head::after {
  content: '';
  position: absolute;
  top: 3px;
  left: 5px;
  width: 5px;
  height: 4px;
  background: rgba(255, 255, 255, 0.45);
  border-radius: 50%;
  transform: rotate(-30deg);
}

.pin-shadow {
  position: absolute;
  bottom: -3px;
  left: 50%;
  transform: translateX(-50%);
  width: 14px;
  height: 4px;
  background: rgba(0, 0, 0, 0.15);
  border-radius: 50%;
  filter: blur(1px);
}

.sticky-note:hover .pin-head {
  transform: rotate(8deg) scale(1.05);
}

/* 颜色系统/纸张纹理/暗色适配见全局 assets/styles/sticky-note-colors.css（单源）。
   模板已挂 sticky-mini-* / sticky-mini-paper-* 全局类，此处不再重复定义 */

/* 头部 */
.sticky-note-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 10px;
}

.sticky-note-title-container {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
}

.sticky-note-reminder {
  color: #ff9800;
  font-size: var(--font-size-xxxs);
  animation: pulse 2s ease-in-out infinite;
  flex-shrink: 0;
}

@keyframes pulse {
  0%, 100% { transform: scale(1); opacity: 1; }
  50% { transform: scale(1.15); opacity: 0.7; }
}

.sticky-note-actions {
  display: flex;
  gap: 4px;
  align-items: center;
  flex-shrink: 0;
}

.sticky-note-action,
.sticky-note-delete {
  width: 24px;
  height: 24px;
  border: none;
  background-color: rgba(0, 0, 0, 0.06);
  color: rgba(0, 0, 0, 0.35);
  border-radius: 50%;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-xxxs);
  opacity: 0;
  transform: scale(0.8);
}

.sticky-note:hover .sticky-note-action,
.sticky-note:hover .sticky-note-delete {
  opacity: 1;
  transform: scale(1);
}

.sticky-note-action:hover {
  background-color: rgba(33, 150, 243, 0.15);
  color: #1976d2;
  transform: scale(1.1);
}

.sticky-note-delete:hover {
  background-color: rgba(244, 67, 54, 0.15);
  color: #d32f2f;
  transform: scale(1.1);
}

/* 标题 */
.sticky-note-title {
  font-size: var(--font-size-sm);
  font-weight: 700;
  margin: 0;
  word-break: break-word;
  line-height: 1.3;
}

/* 内容 */
.sticky-note-content {
  font-size: var(--font-size-xxs);
  line-height: 1.55;
  margin-bottom: 10px;
  flex: 1;
  word-break: break-word;
  white-space: pre-wrap;
  opacity: 0.88;
  /* 墨色随全局色板 --sticky-ink（颜色类已抽到全局） */
  color: var(--sticky-ink, #6d4c41);
}

/* 标签 */
.sticky-note-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 8px;
}

.sticky-note-tag {
  font-size: var(--font-size-tiny);
  padding: 2px 8px;
  border-radius: 10px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.sticky-note-tag:hover {
  filter: brightness(0.92);
  transform: scale(1.05);
}

/* 底部 */
.sticky-note-footer {
  margin-top: auto;
  font-size: var(--font-size-tiny);
  opacity: 0.6;
  transition: opacity 0.2s ease;
}

.sticky-note-date {
}

.sticky-note:hover .sticky-note-footer {
  opacity: 0.85;
}


[data-theme="elegant-dark"] .sticky-note-action,
[data-theme="elegant-dark"] .sticky-note-delete {
  background-color: rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.4);
}

@media (max-width: 768px) {
  .sticky-note {
    min-height: 160px;
    padding: 32px 12px 12px;
  }
}
</style>

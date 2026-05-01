<template>
  <div
    class="sticky-note"
    :class="[colorClass, paperStyleClass]"
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
  '--note-rotation': `${rotationSeed.value}deg`
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

/* 颜色系统 - 柔和渐变 */
.sticky-note.yellow {
  background: linear-gradient(145deg, #fff9c4 0%, #fff59d 100%);
}
.sticky-note.yellow .sticky-note-title { color: #5d4037; }
.sticky-note.yellow .sticky-note-content { color: #6d4c41; }
.sticky-note.yellow .sticky-note-date { color: #a1887f; }
.sticky-note.yellow .sticky-note-tag { background: rgba(93, 64, 55, 0.12); color: #5d4037; }

.sticky-note.blue {
  background: linear-gradient(145deg, #e1f5fe 0%, #b3e5fc 100%);
}
.sticky-note.blue .sticky-note-title { color: #0d47a1; }
.sticky-note.blue .sticky-note-content { color: #1565c0; }
.sticky-note.blue .sticky-note-date { color: #42a5f5; }
.sticky-note.blue .sticky-note-tag { background: rgba(13, 71, 161, 0.1); color: #0d47a1; }

.sticky-note.green {
  background: linear-gradient(145deg, #e8f5e9 0%, #c8e6c9 100%);
}
.sticky-note.green .sticky-note-title { color: #1b5e20; }
.sticky-note.green .sticky-note-content { color: #2e7d32; }
.sticky-note.green .sticky-note-date { color: #66bb6a; }
.sticky-note.green .sticky-note-tag { background: rgba(27, 94, 32, 0.1); color: #1b5e20; }

.sticky-note.red {
  background: linear-gradient(145deg, #fce4ec 0%, #f8bbd0 100%);
}
.sticky-note.red .sticky-note-title { color: #880e4f; }
.sticky-note.red .sticky-note-content { color: #ad1457; }
.sticky-note.red .sticky-note-date { color: #e91e63; }
.sticky-note.red .sticky-note-tag { background: rgba(136, 14, 79, 0.1); color: #880e4f; }

.sticky-note.purple {
  background: linear-gradient(145deg, #f3e5f5 0%, #e1bee7 100%);
}
.sticky-note.purple .sticky-note-title { color: #4a148c; }
.sticky-note.purple .sticky-note-content { color: #6a1b9a; }
.sticky-note.purple .sticky-note-date { color: #ab47bc; }
.sticky-note.purple .sticky-note-tag { background: rgba(74, 20, 140, 0.1); color: #4a148c; }

.sticky-note.pink {
  background: linear-gradient(145deg, #fce4ec 0%, #f8bbd0 100%);
}
.sticky-note.pink .sticky-note-title { color: #880e4f; }
.sticky-note.pink .sticky-note-content { color: #ad1457; }
.sticky-note.pink .sticky-note-date { color: #e91e63; }
.sticky-note.pink .sticky-note-tag { background: rgba(136, 14, 79, 0.1); color: #880e4f; }

/* 纸张样式 */
.sticky-note.lined {
  background-image: repeating-linear-gradient(
    0deg,
    transparent,
    transparent 19px,
    rgba(0, 0, 0, 0.08) 19px,
    rgba(0, 0, 0, 0.08) 20px
  );
  background-size: 100% 20px;
  background-position: 0 36px;
}

.sticky-note.grid {
  background-image:
    repeating-linear-gradient(
      0deg,
      transparent,
      transparent 19px,
      rgba(0, 0, 0, 0.06) 19px,
      rgba(0, 0, 0, 0.06) 20px
    ),
    repeating-linear-gradient(
      90deg,
      transparent,
      transparent 19px,
      rgba(0, 0, 0, 0.06) 19px,
      rgba(0, 0, 0, 0.06) 20px
    );
  background-size: 20px 20px;
  background-position: 0 36px, 0 0;
}

.sticky-note.dotted {
  background-image: radial-gradient(circle, rgba(0, 0, 0, 0.15) 1px, transparent 1px);
  background-size: 18px 18px;
  background-position: 0 44px;
}

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
  font-size: 11px;
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
  font-size: 11px;
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
  font-size: 14px;
  font-weight: 700;
  margin: 0;
  word-break: break-word;
  line-height: 1.3;
}

/* 内容 */
.sticky-note-content {
  font-size: 12px;
  line-height: 1.55;
  margin-bottom: 10px;
  flex: 1;
  word-break: break-word;
  white-space: pre-wrap;
  opacity: 0.88;
}

/* 标签 */
.sticky-note-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 8px;
}

.sticky-note-tag {
  font-size: 10px;
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
  font-size: 10px;
  opacity: 0.6;
  transition: opacity 0.2s ease;
}

.sticky-note-date {
}

.sticky-note:hover .sticky-note-footer {
  opacity: 0.85;
}

/* 暗色主题适配 */
[data-theme="elegant-dark"] .sticky-note.yellow {
  background: linear-gradient(145deg, #4a4520 0%, #3d3a18 100%);
}
[data-theme="elegant-dark"] .sticky-note.yellow .sticky-note-title { color: #fff9c4; }
[data-theme="elegant-dark"] .sticky-note.yellow .sticky-note-content { color: #e6d98a; }
[data-theme="elegant-dark"] .sticky-note.yellow .sticky-note-date { color: #a1887f; }
[data-theme="elegant-dark"] .sticky-note.yellow .sticky-note-tag { background: rgba(255, 249, 196, 0.12); color: #fff9c4; }

[data-theme="elegant-dark"] .sticky-note.blue {
  background: linear-gradient(145deg, #1a3a4a 0%, #15303d 100%);
}
[data-theme="elegant-dark"] .sticky-note.blue .sticky-note-title { color: #b3e5fc; }
[data-theme="elegant-dark"] .sticky-note.blue .sticky-note-content { color: #81d4fa; }
[data-theme="elegant-dark"] .sticky-note.blue .sticky-note-date { color: #42a5f5; }
[data-theme="elegant-dark"] .sticky-note.blue .sticky-note-tag { background: rgba(179, 229, 252, 0.12); color: #b3e5fc; }

[data-theme="elegant-dark"] .sticky-note.green {
  background: linear-gradient(145deg, #1b3a1e 0%, #153018 100%);
}
[data-theme="elegant-dark"] .sticky-note.green .sticky-note-title { color: #c8e6c9; }
[data-theme="elegant-dark"] .sticky-note.green .sticky-note-content { color: #a5d6a7; }
[data-theme="elegant-dark"] .sticky-note.green .sticky-note-date { color: #66bb6a; }
[data-theme="elegant-dark"] .sticky-note.green .sticky-note-tag { background: rgba(200, 230, 201, 0.12); color: #c8e6c9; }

[data-theme="elegant-dark"] .sticky-note.red {
  background: linear-gradient(145deg, #3a1a24 0%, #301520 100%);
}
[data-theme="elegant-dark"] .sticky-note.red .sticky-note-title { color: #f8bbd0; }
[data-theme="elegant-dark"] .sticky-note.red .sticky-note-content { color: #f48fb1; }
[data-theme="elegant-dark"] .sticky-note.red .sticky-note-date { color: #e91e63; }
[data-theme="elegant-dark"] .sticky-note.red .sticky-note-tag { background: rgba(248, 187, 208, 0.12); color: #f8bbd0; }

[data-theme="elegant-dark"] .sticky-note.purple {
  background: linear-gradient(145deg, #2a1a3a 0%, #221530 100%);
}
[data-theme="elegant-dark"] .sticky-note.purple .sticky-note-title { color: #e1bee7; }
[data-theme="elegant-dark"] .sticky-note.purple .sticky-note-content { color: #ce93d8; }
[data-theme="elegant-dark"] .sticky-note.purple .sticky-note-date { color: #ab47bc; }
[data-theme="elegant-dark"] .sticky-note.purple .sticky-note-tag { background: rgba(225, 190, 231, 0.12); color: #e1bee7; }

[data-theme="elegant-dark"] .sticky-note.pink {
  background: linear-gradient(145deg, #3a1a28 0%, #301520 100%);
}
[data-theme="elegant-dark"] .sticky-note.pink .sticky-note-title { color: #f8bbd0; }
[data-theme="elegant-dark"] .sticky-note.pink .sticky-note-content { color: #f48fb1; }
[data-theme="elegant-dark"] .sticky-note.pink .sticky-note-date { color: #e91e63; }
[data-theme="elegant-dark"] .sticky-note.pink .sticky-note-tag { background: rgba(248, 187, 208, 0.12); color: #f8bbd0; }

[data-theme="elegant-dark"] .sticky-note.lined {
  background-image: repeating-linear-gradient(
    0deg,
    transparent,
    transparent 19px,
    rgba(255, 255, 255, 0.06) 19px,
    rgba(255, 255, 255, 0.06) 20px
  );
}

[data-theme="elegant-dark"] .sticky-note.grid {
  background-image:
    repeating-linear-gradient(0deg, transparent, transparent 19px, rgba(255,255,255,0.04) 19px, rgba(255,255,255,0.04) 20px),
    repeating-linear-gradient(90deg, transparent, transparent 19px, rgba(255,255,255,0.04) 19px, rgba(255,255,255,0.04) 20px);
}

[data-theme="elegant-dark"] .sticky-note.dotted {
  background-image: radial-gradient(circle, rgba(255, 255, 255, 0.1) 1px, transparent 1px);
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

<template>
  <div class="message-bubble news-message attachment-card" :class="{ self: isSelf }" @click="openNewsLink">
    <div class="attachment-card__icon news-attachment-icon">
      <img v-if="newsData?.image" :src="newsData.image" class="news-image" :alt="newsData?.title" />
      <i v-else class="fas fa-newspaper"></i>
    </div>
    <div class="attachment-card__content">
      <div class="attachment-card__title">{{ newsData?.title }}</div>
      <div class="attachment-card__meta">资讯 · 查看详情</div>
    </div>
    <div class="attachment-card__action">
      <i class="fas fa-chevron-right"></i>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  newsData?: {
    title: string
    summary: string
    url: string
    image?: string
  }
  isSelf?: boolean
}>()

const emit = defineEmits<{
  open: [url: string]
}>()

const openNewsLink = () => {
  if (props.newsData?.url) {
    emit('open', props.newsData.url)
  }
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
  padding: 12px;
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
  color: #d97706;
  background: color-mix(in srgb, #f59e0b, transparent 88%);
  font-size: 17px;
}

.news-image {
  width: 42px;
  height: 42px;
  object-fit: cover;
  display: block;
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
  font-size: 12px;
  transition: background 0.16s ease, color 0.16s ease, transform 0.16s ease;
}

.attachment-card:hover .attachment-card__action {
  color: var(--primary-color);
  background: color-mix(in srgb, var(--primary-color), transparent 90%);
  transform: translateX(2px);
}

.news-message.self {
  background: color-mix(in srgb, var(--sidebar-bg), transparent 4%);
  border-color: color-mix(in srgb, var(--border-color), transparent 20%);
  color: var(--text-color);
}

:global(.message-item.self) .news-message.self {
  background: color-mix(in srgb, var(--sidebar-bg), transparent 4%);
  border-color: transparent;
  color: var(--text-color);
}

[data-theme="elegant-dark"] .attachment-card {
  background: color-mix(in srgb, var(--panel-bg), white 5%);
  border-color: rgba(255, 255, 255, 0.12);
  box-shadow: none;
}

[data-theme="elegant-dark"] .news-message.self {
  background: color-mix(in srgb, var(--panel-bg), white 5%);
  border-color: rgba(255, 255, 255, 0.12);
  color: var(--text-color);
}

:global([data-theme="elegant-dark"] .message-item.self) .news-message.self {
  background: color-mix(in srgb, var(--panel-bg), white 5%);
  border-color: transparent;
  color: var(--text-color);
}
</style>

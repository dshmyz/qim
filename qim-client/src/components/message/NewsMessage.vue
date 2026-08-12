<template>
  <AttachmentCard class="news-message" :class="{ self: isSelf }" @click="openNewsLink">
    <template #icon>
      <div class="attachment-card__icon" :class="{ 'news-icon': !newsData?.image }">
        <img v-if="newsData?.image" :src="newsData.image" class="news-image" :alt="newsData?.title" />
        <i v-else class="fas fa-newspaper"></i>
      </div>
    </template>
    <template #content>
      <div class="news-title">{{ newsData?.title }}</div>
      <div class="news-bottom">
        <div class="news-meta">资讯 · 查看详情</div>
        <div class="attachment-card__btn">
          <i class="fas fa-chevron-right"></i>
        </div>
      </div>
    </template>
  </AttachmentCard>
</template>

<script setup lang="ts">
import AttachmentCard from './AttachmentCard.vue'

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
.news-message {
  width: 300px;
  max-width: min(100%, 340px);
  min-width: 0;
  box-sizing: border-box;
}

.news-icon {
  color: #ffffff;
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%) !important;
}

.news-title {
  font-size: 14px;
  font-weight: 600;
  line-height: 1.35;
  color: var(--text-color);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  letter-spacing: -0.01em;
}

.news-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.news-meta {
  min-height: 16px;
  font-size: 12px;
  line-height: 1.35;
  color: var(--text-secondary);
}

:deep(.attachment-card__icon) {
  width: 42px;
  height: 42px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  font-size: 17px;
}

.news-image {
  width: 42px;
  height: 42px;
  object-fit: cover;
  display: block;
}
</style>

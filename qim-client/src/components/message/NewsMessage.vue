<template>
  <div class="message-bubble news-message" :class="{ self: isSelf }">
    <div class="news-info" @click="openNewsLink">
      <div class="news-content">
        <div class="news-title">{{ newsData?.title }}</div>
        <div class="news-summary">{{ newsData?.summary }}</div>
      </div>
      <div class="news-image-container" v-if="newsData?.image">
        <img :src="newsData?.image" class="news-image" :alt="newsData?.title" />
      </div>
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
.news-message {
  background: var(--sidebar-bg);
  border-radius: 12px;
  padding: 14px;
  width: fit-content;
  max-width: 100%;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  transition: all 0.2s ease;
  box-sizing: border-box;
}

.news-message:hover {
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.12);
}

.news-info {
  display: flex;
  padding: 12px;
  background: var(--list-bg);
  cursor: pointer;
  transition: background 0.2s;
  border-radius: 8px;
}

.news-info:hover {
  background: var(--hover-color);
}

.news-content {
  flex: 1;
  min-width: 0;
  margin-right: 12px;
}

.news-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color);
  margin-bottom: 6px;
  word-break: break-all;
  line-height: 1.3;
}

.news-summary {
  font-size: 12px;
  color: var(--text-secondary);
  word-break: break-all;
  line-height: 1.3;
}

.news-image-container {
  flex-shrink: 0;
}

.news-image {
  width: 80px;
  height: 60px;
  border-radius: 6px;
  object-fit: cover;
  border: 1px solid var(--border-color);
}

/* 自己的资讯消息样式 */
.news-message.self {
  background: var(--primary-color);
}

.news-message.self .news-info {
  background: rgba(255, 255, 255, 0.1);
}

.news-message.self .news-info:hover {
  background: rgba(255, 255, 255, 0.15);
}

.news-message.self .news-title {
  color: #fff;
}

.news-message.self .news-summary {
  color: rgba(255, 255, 255, 0.8);
}

.news-message.self .news-image {
  border-color: rgba(255, 255, 255, 0.3);
}
</style>
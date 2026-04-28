<template>
  <div class="ai-search-results">
    <div class="results-header">
      <span>找到 {{ results.length }} 条相关消息</span>
      <button class="close-btn" @click="$emit('close')">&times;</button>
    </div>
    <div class="results-list">
      <div
        v-for="result in results"
        :key="result.message_id"
        class="result-item"
        @click="$emit('select', result)"
      >
        <div class="result-sender">{{ result.sender_name }}</div>
        <div class="result-time">{{ result.timestamp }}</div>
        <div class="result-content" v-html="result.highlighted || result.content"></div>
        <div class="result-score">相关度: {{ result.relevance_score }}%</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  results: Array<{
    message_id: number
    content: string
    sender_name: string
    timestamp: string
    relevance_score: number
    highlighted?: string
  }>
  conversationId: number
}>()

defineEmits<{
  select: [result: any]
  close: []
}>()
</script>

<style scoped>
.ai-search-results {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  margin-top: 4px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  max-height: 400px;
  overflow-y: auto;
  z-index: 100;
}

.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 14px;
  border-bottom: 1px solid var(--border-color);
  font-size: 13px;
  color: var(--text-secondary);
}

.close-btn {
  border: none;
  background: transparent;
  font-size: 18px;
  cursor: pointer;
  color: var(--text-secondary);
}

.close-btn:hover {
  color: var(--text-primary);
}

.results-list {
  scrollbar-width: thin;
}

.result-item {
  padding: 12px 14px;
  border-bottom: 1px solid var(--border-color);
  cursor: pointer;
  transition: background 0.2s;
}

.result-item:last-child {
  border-bottom: none;
}

.result-item:hover {
  background: var(--hover-color);
}

.result-sender {
  font-weight: 600;
  font-size: 13px;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.result-time {
  font-size: 11px;
  color: var(--text-secondary);
  margin-bottom: 6px;
}

.result-content {
  font-size: 14px;
  color: var(--text-primary);
  line-height: 1.4;
}

.result-score {
  font-size: 11px;
  color: var(--primary-color);
  margin-top: 4px;
}
</style>

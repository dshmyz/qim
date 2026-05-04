<template>
  <div class="thinking-indicator">
    <span class="dot"></span>
    <span class="dot"></span>
    <span class="dot"></span>
  </div>
</template>

<script setup lang="ts">
// ThinkingIndicator 组件 - 显示 AI 正在思考的动画指示器
</script>

<style scoped>
.thinking-indicator {
  display: flex;
  gap: 4px;
  padding: 12px 14px;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--primary-color);
  animation: pulse 1.5s infinite;
}

.dot:nth-child(2) {
  animation-delay: 0.2s;
}

.dot:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes pulse {
  0%, 100% { opacity: 0.3; }
  50% { opacity: 1; }
}
</style>

<template>
  <div class="thinking-indicator">
    <!-- 阶段文案（"正在看图片…"/"正在处理…"）：让首字前的等待有具体解释；
         无文案时只显示动画点 -->
    <span v-if="label" class="thinking-label">{{ label }}</span>
    <span class="dot"></span>
    <span class="dot"></span>
    <span class="dot"></span>
  </div>
</template>

<script setup lang="ts">
// ThinkingIndicator 组件 - 显示 AI 正在思考的动画指示器
defineProps<{
  /** 可选阶段文案（来自 ai_reply_started.status），为空时只显示动画点 */
  label?: string
}>()
</script>

<style scoped>
.thinking-indicator {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 12px 14px;
}

.thinking-label {
  margin-right: 4px;
  color: var(--text-secondary, #9ca3af);
  font-size: var(--font-size-xs);
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--primary-color);
  animation: pulse 1.5s infinite;
}

/* 用 nth-last-child 定位圆点：有无前一行的状态文案都不影响错峰动画 */
.dot:nth-last-child(2) {
  animation-delay: 0.2s;
}

.dot:nth-last-child(1) {
  animation-delay: 0.4s;
}

@keyframes pulse {
  0%, 100% { opacity: 0.3; }
  50% { opacity: 1; }
}
</style>

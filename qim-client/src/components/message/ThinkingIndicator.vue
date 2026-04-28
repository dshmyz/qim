<template>
  <div class="thinking-indicator">
    <div class="ai-avatar">🤖</div>
    <div class="thinking-content">
      <span class="thinking-text">AI 正在思考</span>
      <span class="thinking-dots">
        <span class="dot">.</span>
        <span class="dot">.</span>
        <span class="dot">.</span>
      </span>
    </div>
  </div>
</template>

<style scoped>
.thinking-indicator {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  max-width: 80%;
  background: var(--color-surface);
  border-radius: 12px;
  border: 1px solid var(--color-border);
}

.ai-avatar {
  font-size: 20px;
  flex-shrink: 0;
}

.thinking-content {
  display: flex;
  align-items: center;
  gap: 4px;
}

.thinking-text {
  font-size: 13px;
  color: var(--color-text-secondary);
}

.thinking-dots {
  display: inline-flex;
}

.dot {
  animation: blink 1.4s infinite;
  font-size: 16px;
  color: var(--color-text-secondary);
}

.dot:nth-child(2) { animation-delay: 0.2s; }
.dot:nth-child(3) { animation-delay: 0.4s; }

@keyframes blink {
  0%, 20% { opacity: 0; }
  50% { opacity: 1; }
  100% { opacity: 0; }
}
</style>

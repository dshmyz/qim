<template>
  <div class="image-placeholder media-preview-placeholder">
    <div class="placeholder-icon">
      <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
        <rect x="3" y="3" width="18" height="18" rx="2" stroke="currentColor" stroke-width="2"/>
        <circle cx="8.5" cy="8.5" r="1.5" fill="currentColor"/>
        <path d="M21 15L16 10L5 21" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
    </div>
    <div class="placeholder-text">{{ text }}</div>
    <div v-if="isLoading" class="loading-spinner"></div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  text?: string
  isLoading?: boolean
}>()
</script>

<style scoped>
.image-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 200px;
  height: 150px;
  background: color-mix(in srgb, var(--sidebar-bg, #ffffff), #f8fafc 45%);
  border-radius: 14px;
  border: 1px solid color-mix(in srgb, var(--border-color, #d9d9d9), transparent 20%);
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.06);
  position: relative;
  overflow: hidden;
  box-sizing: border-box;
}

.placeholder-icon {
  width: 48px;
  height: 48px;
  color: var(--text-secondary, #64748b);
  opacity: 0.55;
}

.placeholder-icon svg {
  width: 100%;
  height: 100%;
}

.placeholder-text {
  font-size: 12px;
  color: var(--text-secondary, #64748b);
  user-select: none;
}

.loading-spinner {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 24px;
  height: 24px;
  border: 2px solid rgba(59, 130, 246, 0.2);
  border-top-color: var(--primary-color, #3b82f6);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: translate(-50%, -50%) rotate(360deg);
  }
}

[data-theme="elegant-dark"] .image-placeholder {
  background: color-mix(in srgb, var(--panel-bg, #1f1f2e), white 5%);
  border-color: rgba(255, 255, 255, 0.12);
  box-shadow: none;
}

[data-theme="elegant-dark"] .placeholder-icon {
  color: #666;
}

[data-theme="elegant-dark"] .placeholder-text {
  color: #666;
}
</style>

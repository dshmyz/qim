<template>
  <div class="attachment-card" :class="{ 'attachment-card--self': isSelf }">
    <div class="attachment-card__icon">
      <slot />
    </div>
    <div class="attachment-card__content">
      <slot name="content" />
    </div>
    <div v-if="$slots.trailing" class="attachment-card__trailing">
      <slot name="trailing" />
    </div>
    <div v-if="$slots.below" class="attachment-card__below">
      <slot name="below" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  isSelf?: boolean
}>()
</script>

<style scoped>
.attachment-card {
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  width: 300px;
  max-width: min(100%, 340px);
  padding: 12px;
  border-radius: 14px;
  background: color-mix(in srgb, var(--sidebar-bg), transparent 4%);
  border: 1px solid color-mix(in srgb, var(--border-color), transparent 60%);
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.06);
  box-sizing: border-box;
  cursor: pointer;
}

.attachment-card:hover {
  border-color: color-mix(in srgb, var(--primary-color), transparent 85%);
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
  color: var(--text-secondary);
  background: color-mix(in srgb, var(--text-secondary), transparent 90%);
}

.attachment-card__content {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.attachment-card__trailing {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.attachment-card__below {
  grid-column: 1 / -1;
}

/* Self (sent by me) */
.attachment-card--self {
  border-color: transparent;
  color: var(--text-color);
}

/* Elegant-dark theme */
[data-theme="elegant-dark"] .attachment-card {
  border-color: rgba(255, 255, 255, 0.12);
  box-shadow: none;
}

[data-theme="elegant-dark"] .attachment-card--self {
  border-color: transparent;
  color: var(--text-color);
}
</style>

<!-- Unscoped: shared button styles for slot content -->
<style>
.attachment-card__btn {
  width: 28px;
  height: 28px;
  border-radius: 9px;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  background: transparent;
  cursor: pointer;
  font-size: 12px;
  flex-shrink: 0;
}

.attachment-card__btn:hover {
  color: var(--primary-color);
  background: color-mix(in srgb, var(--primary-color), transparent 90%);
}

.attachment-card__btn:active {
  transform: scale(0.96);
}

.attachment-card__btn i {
  font-size: 12px;
}
</style>

<template>
  <div class="im-container">
    <div class="loading-overlay" v-if="$slots.loading">
      <slot name="loading"></slot>
    </div>
    <div v-if="$slots.networkError">
      <slot name="networkError"></slot>
    </div>
    <div class="window-controls" v-if="$slots.windowControls">
      <slot name="windowControls"></slot>
    </div>
    <div class="main-content-area">
      <div class="side-options" v-if="$slots.sideOptions">
        <slot name="sideOptions"></slot>
      </div>
      <div class="main-content">
        <slot></slot>
      </div>
    </div>
    <slot name="modals"></slot>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped>
.im-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  width: 100vw;
  position: relative;
  background: var(--bg-color, #f5f5f5);
}

.loading-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--bg-color, #f5f5f5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.window-controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 32px;
  background: var(--window-controls-bg, #e8e8e8);
  -webkit-app-region: drag;
  z-index: 100;
}

.main-content-area {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.side-options {
  width: 60px;
  background: var(--side-options-bg, #2c2c2c);
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px 0;
  gap: 8px;
  flex-shrink: 0;
}

.main-content {
  flex: 1;
  display: flex;
  overflow: hidden;
}
</style>

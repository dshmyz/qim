<template>
  <div class="im-container">
    <!-- 加载过渡动画 -->
    <div class="loading-overlay" v-if="$slots.loading">
      <slot name="loading"></slot>
    </div>

    <!-- 网络错误状态 -->
    <div class="network-error" v-if="$slots.networkError">
      <slot name="networkError"></slot>
    </div>

    <!-- 顶部区域：窗口控制栏 -->
    <div class="top-bar">
      <div class="top-bar-left"></div>
      <WindowControls />
    </div>

    <!-- 主内容区域 -->
    <div class="main-content-area">
      <!-- 左侧选项栏 -->
      <div class="side-options" v-if="$slots.sideOptions">
        <slot name="sideOptions"></slot>
      </div>

      <!-- 主内容 -->
      <div class="main-content">
        <slot></slot>
      </div>
    </div>

    <!-- 模态框插槽 -->
    <slot name="modals"></slot>
  </div>
</template>

<script setup lang="ts">
import WindowControls from './WindowControls.vue'
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

/* 加载状态 */
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

/* 网络错误状态 */
.network-error {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--bg-color, #f5f5f5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 999;
}

/* 顶部区域 */
.top-bar {
  display: flex;
  height: 40px;
  -webkit-app-region: drag;
  flex-shrink: 0;
}

.top-bar-left {
  width: 60px;
  background: var(--sidebar-bg);
  flex-shrink: 0;
  transition: background 0.3s ease;
}

/* 主内容区域 */
.main-content-area {
  display: flex;
  flex: 1;
  overflow: hidden;
}

/* 主内容 */
.main-content {
  flex: 1;
  display: flex;
  overflow: hidden;
}
</style>

<!--
  ContentSkeleton.vue - 内容骨架屏组件

  功能：
  - 根据类型显示不同布局的骨架屏（recent/channels/org/apps/groups/settings）
  - shimmer 动画效果，提升加载体验
  - 支持深色模式，自动适配主题
  - 可配置占位项数量

  使用示例：
  <ContentSkeleton type="recent" />
  <ContentSkeleton type="apps" :count="8" />
  <ContentSkeleton type="org" />
-->
<template>
  <div class="content-skeleton" role="status" aria-label="内容加载中" aria-busy="true">
    <!-- 聊天/会话列表骨架屏 -->
    <div v-if="type === 'recent'" class="skeleton-recent">
      <div v-for="i in count" :key="i" class="skeleton-conversation-item">
        <div class="skeleton-shape skeleton-avatar"></div>
        <div class="skeleton-content">
          <div class="skeleton-shape skeleton-title"></div>
          <div class="skeleton-shape skeleton-subtitle"></div>
        </div>
        <div class="skeleton-meta">
          <div class="skeleton-shape skeleton-time"></div>
          <div class="skeleton-shape skeleton-badge"></div>
        </div>
      </div>
    </div>

    <!-- 频道骨架屏 -->
    <div v-else-if="type === 'channels'" class="skeleton-channels">
      <div class="skeleton-shape skeleton-channel-header"></div>
      <div class="skeleton-message-list">
        <div v-for="i in count" :key="i" class="skeleton-message">
          <div class="skeleton-shape skeleton-avatar"></div>
          <div class="skeleton-message-content">
            <div class="skeleton-shape skeleton-author"></div>
            <div class="skeleton-shape skeleton-text"></div>
            <div class="skeleton-shape skeleton-text short"></div>
          </div>
        </div>
      </div>
    </div>

    <!-- 组织架构骨架屏 -->
    <div v-else-if="type === 'org'" class="skeleton-org">
      <div class="skeleton-dept" v-for="i in 3" :key="i">
        <div class="skeleton-shape skeleton-dept-header"></div>
        <div class="skeleton-employees">
          <div v-for="j in 4" :key="j" class="skeleton-employee">
            <div class="skeleton-shape skeleton-avatar small"></div>
            <div class="skeleton-shape skeleton-info"></div>
          </div>
        </div>
      </div>
    </div>

    <!-- 应用中心骨架屏 -->
    <div v-else-if="type === 'apps'" class="skeleton-apps">
      <div class="skeleton-shape skeleton-section-header"></div>
      <div class="skeleton-grid">
        <div v-for="i in 8" :key="i" class="skeleton-app-item">
          <div class="skeleton-shape skeleton-icon"></div>
          <div class="skeleton-shape skeleton-name"></div>
        </div>
      </div>
    </div>

    <!-- 群组/设置通用列表骨架屏 -->
    <div v-else class="skeleton-generic">
      <div class="skeleton-shape skeleton-header-bar"></div>
      <div v-for="i in count" :key="i" class="skeleton-list-item">
        <div class="skeleton-shape skeleton-line"></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
/**
 * 骨架屏组件 Props
 * @property type - 骨架屏类型，决定显示的布局
 * @property count - 占位项数量，默认 5
 */
interface Props {
  type: 'recent' | 'channels' | 'org' | 'apps' | 'groups' | 'settings'
  count?: number
}

withDefaults(defineProps<Props>(), {
  count: 5
})
</script>

<style scoped>
/* ========================================
   骨架屏主题变量 - 自动适配深色模式
   ======================================== */
.content-skeleton {
  --skeleton-base: #f0f0f0;
  --skeleton-highlight: #e0e0e0;
  --skeleton-radius: var(--radius-md, 8px);
  padding: var(--spacing-3, 12px) var(--spacing-4, 16px);
}

/* 深色模式适配 */
:global([data-theme="elegant-dark"]) .content-skeleton,
:global(.dark-theme) .content-skeleton {
  --skeleton-base: #2a2a2a;
  --skeleton-highlight: #3a3a3a;
}

/* ========================================
   shimmer 动画核心样式
   ======================================== */
.skeleton-shape {
  background: linear-gradient(
    90deg,
    var(--skeleton-base) 25%,
    var(--skeleton-highlight) 50%,
    var(--skeleton-base) 75%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s ease-in-out infinite;
  border-radius: var(--skeleton-radius);
}

@keyframes shimmer {
  0% {
    background-position: -200% 0;
  }
  100% {
    background-position: 200% 0;
  }
}

/* ========================================
   recent 类型 - 聊天/会话列表
   与 ConversationList.vue 布局对齐
   ======================================== */
.skeleton-recent {
  display: flex;
  flex-direction: column;
  width: 100%;
}

.skeleton-conversation-item {
  display: flex;
  align-items: center;
  padding: 12px 20px;
  gap: 12px;
}

.skeleton-conversation-item .skeleton-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  flex-shrink: 0;
}

.skeleton-conversation-item .skeleton-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.skeleton-conversation-item .skeleton-title {
  width: 45%;
  height: 14px;
  border-radius: 4px;
}

.skeleton-conversation-item .skeleton-subtitle {
  width: 70%;
  height: 12px;
  border-radius: 4px;
}

.skeleton-conversation-item .skeleton-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
  flex-shrink: 0;
}

.skeleton-conversation-item .skeleton-time {
  width: 36px;
  height: 11px;
  border-radius: 4px;
}

.skeleton-conversation-item .skeleton-badge {
  width: 20px;
  height: 18px;
  border-radius: 9px;
}

/* ========================================
   channels 类型 - 频道消息列表
   ======================================== */
.skeleton-channels {
  display: flex;
  flex-direction: column;
  width: 100%;
}

.skeleton-channel-header {
  width: 60%;
  height: 24px;
  margin-bottom: var(--spacing-4, 16px);
  border-radius: var(--radius-sm, 4px);
}

.skeleton-message-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-3, 12px);
}

.skeleton-message {
  display: flex;
  gap: 12px;
  padding: 8px 0;
}

.skeleton-message .skeleton-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  flex-shrink: 0;
}

.skeleton-message-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.skeleton-message .skeleton-author {
  width: 30%;
  height: 13px;
  border-radius: 4px;
}

.skeleton-message .skeleton-text {
  width: 85%;
  height: 12px;
  border-radius: 4px;
}

.skeleton-message .skeleton-text.short {
  width: 55%;
  height: 12px;
  border-radius: 4px;
}

/* ========================================
   org 类型 - 组织架构树形结构
   与 OrgTree.vue 布局对齐
   ======================================== */
.skeleton-org {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-4, 16px);
  width: 100%;
}

.skeleton-dept {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2, 8px);
}

.skeleton-dept-header {
  width: 40%;
  height: 16px;
  border-radius: var(--radius-sm, 4px);
  margin-bottom: var(--spacing-2, 8px);
}

.skeleton-employees {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding-left: var(--spacing-6, 24px);
}

.skeleton-employee {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 0;
}

.skeleton-employee .skeleton-avatar.small {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  flex-shrink: 0;
}

.skeleton-employee .skeleton-info {
  width: 50%;
  height: 13px;
  border-radius: 4px;
}

/* ========================================
   apps 类型 - 应用网格布局
   ======================================== */
.skeleton-apps {
  display: flex;
  flex-direction: column;
  width: 100%;
}

.skeleton-section-header {
  width: 35%;
  height: 20px;
  margin-bottom: var(--spacing-4, 16px);
  border-radius: var(--radius-sm, 4px);
}

.skeleton-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--spacing-4, 16px);
}

.skeleton-app-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--spacing-2, 8px);
  padding: var(--spacing-3, 12px);
}

.skeleton-app-item .skeleton-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-lg, 12px);
}

.skeleton-app-item .skeleton-name {
  width: 60%;
  height: 12px;
  border-radius: 4px;
}

/* ========================================
   groups / settings 类型 - 通用列表
   ======================================== */
.skeleton-generic {
  display: flex;
  flex-direction: column;
  width: 100%;
}

.skeleton-header-bar {
  width: 50%;
  height: 22px;
  margin-bottom: var(--spacing-4, 16px);
  border-radius: var(--radius-sm, 4px);
}

.skeleton-list-item {
  padding: var(--spacing-3, 12px) 0;
}

.skeleton-list-item .skeleton-line {
  width: 100%;
  height: 14px;
  border-radius: 4px;
}

/* 让列表行呈现长度变化效果，更接近真实内容 */
.skeleton-list-item:nth-child(odd) .skeleton-line {
  width: 90%;
}

.skeleton-list-item:nth-child(3n) .skeleton-line {
  width: 75%;
}

/* ========================================
   响应式适配
   ======================================== */
@media (max-width: 768px) {
  .skeleton-grid {
    grid-template-columns: repeat(3, 1fr);
    gap: var(--spacing-3, 12px);
  }

  .skeleton-app-item .skeleton-icon {
    width: 40px;
    height: 40px;
  }

  .skeleton-conversation-item {
    padding: 10px 12px;
  }
}

@media (max-width: 480px) {
  .skeleton-grid {
    grid-template-columns: repeat(3, 1fr);
    gap: var(--spacing-2, 8px);
  }

  .skeleton-employees {
    padding-left: var(--spacing-4, 16px);
  }
}

/* ========================================
   无障碍：减少动画偏好
   ======================================== */
@media (prefers-reduced-motion: reduce) {
  .skeleton-shape {
    animation: none;
    background: var(--skeleton-base);
  }
}
</style>

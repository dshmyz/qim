<template>
  <!-- 根节点同时挂 .markdown-message（气泡壳）与 .markdown-content（统一排版单源）。
       排版由全局 markdown-content.css 提供，与 AIAnswerBubble 的 AI 回答正文共用同一份，
       消除复制粘贴的 CSS 漂移。此处仅保留壳与 self 差异。 -->
  <div
    ref="containerRef"
    class="markdown-message markdown-content"
    :class="{ self: isSelf }"
    v-html="html"
    @click="handleLinkClick"
  ></div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useMarkdownRender, handleLinkClick } from '../../composables/useMarkdownRender'

const props = withDefaults(
  defineProps<{
    content: string
    isSelf?: boolean
  }>(),
  {
    isSelf: false,
  }
)

// 统一走 useMarkdownRender（单一渲染管道：解码 mention → marked → 消毒 → emoji/classic）
const { html, containerRef } = useMarkdownRender(
  computed(() => props.content),
  computed(() => ({ decodeMention: true, withEmoji: true }))
)
</script>

<style>
/* 本组件只保留「气泡壳」+ self 差异；markdown 排版统一由 markdown-content.css 提供，
   根节点已挂 .markdown-content，与 AIAnswerBubble 的 AI 回答正文共用同一份排版单源。 */
@import './markdown-content.css';

.markdown-message {
  padding: 10px 14px;
  border-radius: 12px;
  /* 与普通文本消息气泡同源（--message-bubble-bg），避免用 --sidebar-bg 与聊天区背景
     同色而显得没有背景色/线条 */
  background: var(--message-bubble-bg);
  color: var(--text-color);
  font-size: var(--font-size-sm);
  line-height: 1.6;
  word-break: break-word;
}

/* 自身发送的 Markdown 消息：浅色主色背景 + 深色文字。
   排版几何统一由 .markdown-content（markdown-content.css）提供，这里仅覆盖
   self 气泡底色下的文字/border 颜色，与 AIAnswerBubble 的 im.self 观感一致。 */
.markdown-message.self {
  background: var(--hover-color);
  background: color-mix(in srgb, var(--primary-color), white 88%);
  color: var(--text-color);
}

.markdown-message.self h1,
.markdown-message.self h2,
.markdown-message.self h3,
.markdown-message.self strong,
.markdown-message.self em,
.markdown-message.self li,
.markdown-message.self p {
  color: var(--text-color);
}

.markdown-message.self code {
  background-color: var(--hover-color);
  color: var(--text-color);
}

.markdown-message.self pre {
  background-color: var(--hover-color);
  color: var(--text-color);
  border-color: var(--border-color);
}

.markdown-message.self blockquote {
  border-left-color: var(--primary-color);
  color: var(--text-secondary);
}

.markdown-message.self a {
  color: var(--primary-color);
  text-decoration: none;
}

.markdown-message.self a:hover {
  color: var(--primary-hover, var(--primary-color));
}

.markdown-message.self ::selection {
  background: rgba(0, 0, 0, 0.15);
  color: var(--text-color);
}

.markdown-message.self th {
  background: var(--hover-color);
}

.markdown-message.self th,
.markdown-message.self td {
  border-color: var(--border-color);
}

/* 深色主题：纯主色背景 + 白色文字 */
[data-theme="elegant-dark"] .markdown-message.self {
  background: var(--primary-color);
  color: white;
}

[data-theme="elegant-dark"] .markdown-message.self h1,
[data-theme="elegant-dark"] .markdown-message.self h2,
[data-theme="elegant-dark"] .markdown-message.self h3,
[data-theme="elegant-dark"] .markdown-message.self strong,
[data-theme="elegant-dark"] .markdown-message.self em,
[data-theme="elegant-dark"] .markdown-message.self li,
[data-theme="elegant-dark"] .markdown-message.self p {
  color: white;
}

[data-theme="elegant-dark"] .markdown-message.self code {
  background-color: rgba(255, 255, 255, 0.2);
  color: white;
}

[data-theme="elegant-dark"] .markdown-message.self pre {
  background-color: rgba(255, 255, 255, 0.12);
  color: white;
  border-color: rgba(255, 255, 255, 0.2);
}

[data-theme="elegant-dark"] .markdown-message.self blockquote {
  border-left-color: rgba(255, 255, 255, 0.5);
  color: rgba(255, 255, 255, 0.8);
}

[data-theme="elegant-dark"] .markdown-message.self a {
  color: #fef08a;
}

[data-theme="elegant-dark"] .markdown-message.self a:hover {
  color: #fef08a;
}

[data-theme="elegant-dark"] .markdown-message.self ::selection {
  background: rgba(0, 0, 0, 0.25);
  color: white;
}

[data-theme="elegant-dark"] .markdown-message.self th {
  background: rgba(255, 255, 255, 0.15);
}

[data-theme="elegant-dark"] .markdown-message.self th,
[data-theme="elegant-dark"] .markdown-message.self td {
  border-color: rgba(255, 255, 255, 0.2);
}
</style>

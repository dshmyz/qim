<template>
  <!-- AI 渲染能力统一 —— 统一 AI 回答气泡体。
       IM 气泡与 BotChat 独立聊天共用同一组件：markdown 正文（思考占位 + typing dots +
       流式 partial-md）+ 附属反馈行（工具卡片 / 知识来源 / 分身依据）。取代两处各自手写的
       正文/思考/来源/工具渲染。正文排版统一走 useMarkdownRender + markdown-content.css。 -->
  <div :class="['ai-answer-bubble', variant, { self: isSelf }]" @click="handleLinkClick">
    <!-- AI 还没吐第一个字（content 空 + 仍在流）：显示「思考中」+ dots 同行占位，首段到达后自然替换 -->
    <div v-if="showThinking && isStreaming && !content" class="thinking-placeholder">
      <span>思考中</span>
      <span class="typing-indicator-inline"><span class="typing-dot"></span><span class="typing-dot"></span><span class="typing-dot"></span></span>
    </div>
    <div v-else ref="bodyEl" class="markdown-content" v-html="html"></div>
    <div v-if="isStreaming && content" class="typing-indicator">
      <span class="typing-dot"></span>
      <span class="typing-dot"></span>
      <span class="typing-dot"></span>
    </div>

    <!-- 附属反馈行：知识来源 / 工具调用 / 分身依据（统一走 AISources，各组件自身带 v-if 折叠） -->
    <AISources v-if="knowledgeSources && knowledgeSources.length" :sources="knowledgeSources" variant="list" />
    <ToolCallTrace v-if="toolCalls && toolCalls.length" :calls="toolCalls" :open="isStreaming" />
    <AISources v-if="avatarSources && avatarSources.length" :sources="avatarSources" variant="inline" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useMarkdownRender, handleLinkClick } from '../../composables/useMarkdownRender'
import ToolCallTrace from './ToolCallTrace.vue'
import AISources from './AISources.vue'
import type { ToolCallRecord, AISource } from '../../types'
import './markdown-content.css'

const props = withDefaults(defineProps<{
  content: string
  isStreaming: boolean
  isSelf?: boolean
  /** 是否显示「思考中」占位（content 空且 isStreaming 时） */
  showThinking?: boolean
  /** 气泡壳变体：im = IM 消息气泡，botchat = 分身/AI 独立聊天气泡 */
  variant?: 'im' | 'botchat'
  toolCalls?: ToolCallRecord[]
  knowledgeSources?: AISource[]
  avatarSources?: AISource[]
}>(), {
  isSelf: false,
  showThinking: true,
  variant: 'im',
})

// 流式选项随 props 变化：content 变化的 ref 驱动 rerender
const content = computed(() => props.content)
const isStreaming = computed(() => props.isStreaming)
const { html, containerRef: bodyEl } = useMarkdownRender(
  content,
  computed(() => ({ streaming: isStreaming.value, decodeMention: true, withEmoji: true }))
)
</script>

<style>
/* 气泡壳 + 思考/typing UI（bubble-chrome，非 markdown 排版）。
   自人/对侧气泡底色与 StreamingMessage/MarkdownMessage 原有实现对齐。 */
.ai-answer-bubble {
  position: relative;
  width: 100%;
  /* 不再自行限 width 上限：im 变体已嵌套在 .message-content（父级 max-width:80%）
     内，若这里再给 max-width:80% 会与父级 80% 叠加成 64%，比普通文本消息（仅一层 80%）
     明显窄一档；botchat 变体由 BotChatView 的 wrapper 管理宽度。 */
}

/* im 变体：沿用原 StreamingMessage/MarkdownMessage 的气泡壳 */
.ai-answer-bubble.im {
  padding: 10px 14px;
  border-radius: 12px;
  /* 跟随设置字号滑块基准（--font-size-sm），与普通文本/Markdown 气泡统一，
     不再单独用 --font-size-xs（此前导致 AI 消息比普通消息小一档）。 */
  font-size: var(--font-size-sm);
  line-height: 1.6;
}
.ai-answer-bubble.im.self {
  background: color-mix(in srgb, var(--primary-color), white 88%);
  color: var(--text-color);
  border-bottom-right-radius: 4px;
}
[data-theme="elegant-dark"] .ai-answer-bubble.im.self {
  background: var(--primary-color);
  color: white;
}
.ai-answer-bubble.im:not(.self) {
  background: transparent;
  color: var(--text-color);
  border: 1px solid color-mix(in srgb, var(--border-color), transparent 60%);
  border-bottom-left-radius: 4px;
}

/* botchat 变体：气泡壳交给 BotChatView 的 .message-bubble（其内嵌 .ai-answer-bubble），仅保留正文排版 */
.ai-answer-bubble.botchat {
  padding: 0;
  max-width: 100%;
}

.typing-indicator {
  display: flex;
  align-items: center;
  margin-top: 5px;
  gap: 3px;
}

/* 思考中 + dots 同行：v-html 内联渲染不参与气泡，改为模板渲染同样避免局部 min-width 压缩 */
.thinking-placeholder {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: color-mix(in srgb, var(--text-color), transparent 45%);
  font-size: 0.92em;
  letter-spacing: 0.5px;
}
.typing-indicator-inline {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  margin-left: 6px;
  vertical-align: baseline;
}

.typing-indicator-inline .typing-dot,
.typing-indicator .typing-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: currentColor;
  opacity: 0.5;
  animation: aitb-typing 1.4s infinite ease-in-out both;
}
.typing-indicator .typing-dot {
  width: 8px;
  height: 8px;
  opacity: 0.6;
}
.typing-indicator-inline .typing-dot:nth-child(1),
.typing-indicator .typing-dot:nth-child(1) { animation-delay: -0.32s; }
.typing-indicator-inline .typing-dot:nth-child(2),
.typing-indicator .typing-dot:nth-child(2) { animation-delay: -0.16s; }

@keyframes aitb-typing {
  0%, 80%, 100% { transform: scale(0); }
  40% { transform: scale(1); }
}
</style>

<template>
  <div ref="containerRef" class="markdown-content" v-html="html" @click="handleLinkClick"></div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useMarkdownRender, handleLinkClick } from '../../composables/useMarkdownRender'

const props = withDefaults(
  defineProps<{
    content: string
  }>(),
  {
    content: ''
  }
)

// 统一走 useMarkdownRender（单一渲染管道）。BotChatView/NoteEditor 用纯 markdown，不解码 mention、不替换 emoji。
const { html, containerRef } = useMarkdownRender(
  computed(() => props.content),
  computed(() => ({}))
)
</script>

<style>
/* 排版统一由 markdown-content.css 提供（与 MarkdownMessage / AIAnswerBubble 共用同一份单源），
   消除此前自带 scoped CSS 与单源的数值漂移。根节点挂 .markdown-content 命中选择器；
   非 scoped：v-html 注入的元素不带 data-v，scoped 下选择器匹配不上。 */
@import '../message/markdown-content.css';
</style>

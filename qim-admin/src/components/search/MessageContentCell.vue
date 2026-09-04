<template>
  <div class="message-content-cell">
    <!-- text：纯文本保留换行（主客户端 text 消息非 markdown，不做 markdown 解析避免误渲染） -->
    <div v-if="type === 'text'" class="text-content">{{ decodedText }}</div>

    <!-- markdown：解码提及后按 markdown 渲染 -->
    <div v-else-if="type === 'markdown'" class="md-content" v-html="renderedHtml"></div>

    <!-- 图片：JSON {url} 或 URL 形状显示缩略图（点击放大），否则原始文本 -->
    <el-image
      v-else-if="type === 'image' && imageUrl"
      :src="imageUrl"
      :preview-src-list="[imageUrl]"
      preview-teleported
      fit="cover"
      class="msg-image"
    />
    <span v-else-if="type === 'image'">{{ rawText }}</span>

    <!-- 文件/音频/视频：JSON {url,name} 或 URL 形状显示可下载 chip，否则原始文本 -->
    <a
      v-else-if="isFileLike && fileUrl"
      :href="fileUrl"
      target="_blank"
      rel="noopener noreferrer"
      class="file-chip"
    >
      <el-icon class="file-icon"><Document /></el-icon>
      <span class="file-name" :title="fileUrl">{{ fileName }}</span>
    </a>
    <span v-else-if="isFileLike">{{ rawText }}</span>

    <!-- 其它类型：原始文本 -->
    <span v-else class="raw-text">{{ rawText }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Document } from '@element-plus/icons-vue'
import { marked } from 'marked'
import { sanitizeMarkdown } from '@/utils/sanitize'
import { decodeMentionTokens } from '@/utils/mentions'

const props = defineProps<{
  type?: string
  content?: string
}>()

const isFileLike = computed(() => ['file', 'audio', 'video'].includes(props.type || ''))

// 图片/文件消息的 content 是 JSON（{"url":...,"name":...}），解析提取展示字段；
// 解析失败按原始文本兜底。
function parseContent(): { url?: string; name?: string } {
  const s = props.content || ''
  if (!s) return {}
  try {
    const p = JSON.parse(s)
    if (p && typeof p === 'object' && typeof p.url === 'string') {
      return { url: p.url, name: typeof p.name === 'string' ? p.name : undefined }
    }
  } catch { /* 非 JSON（如纯文本/老格式），走原始文本 */ }
  return {}
}

const parsed = computed(() => parseContent())
const imageUrl = computed(() => (props.type === 'image' && isUrlShape(parsed.value.url) ? parsed.value.url! : ''))
const fileUrl = computed(() => (isFileLike.value && isUrlShape(parsed.value.url) ? parsed.value.url! : ''))
const rawText = computed(() => props.content || '-')

const decodedText = computed(() => decodeMentionTokens(props.content || ''))

const renderedHtml = computed(() => {
  const content = decodeMentionTokens(props.content || '')
  if (!content) return ''
  const html = marked.parse(content, { async: false }) as string
  return sanitizeMarkdown(html)
})

function isUrlShape(s?: string): boolean {
  if (!s) return false
  return /^https?:\/\//.test(s) || s.startsWith('/')
}

const fileName = computed(() => {
  if (parsed.value.name) return parsed.value.name
  const c = parsed.value.url || props.content || ''
  return c.split(/[\\/]/).pop() || c || '文件'
})
</script>

<style scoped>
.message-content-cell {
  line-height: 1.5;
}

/* text：保留换行 */
.text-content {
  white-space: pre-wrap;
  word-break: break-word;
}

/* markdown：沿用 admin 已用的 sanitize 渲染，行内样式尽量克制 */
.md-content {
  white-space: normal;
  word-break: break-word;
}
.md-content :deep(p) {
  margin: 0 0 4px;
}
.md-content :deep(p:last-child) {
  margin-bottom: 0;
}
.md-content :deep(code) {
  background: rgba(100, 100, 100, 0.12);
  padding: 1px 5px;
  border-radius: 4px;
  font-size: 0.92em;
}
.md-content :deep(pre) {
  background: rgba(100, 100, 100, 0.08);
  padding: 8px 10px;
  border-radius: 6px;
  overflow-x: auto;
  max-width: 100%;
}

/* 图片缩略图：限高 + 圆角，点击放大 */
.msg-image {
  max-width: 160px;
  max-height: 120px;
  border-radius: 6px;
  cursor: zoom-in;
  display: inline-block;
}
.msg-image :deep(img) {
  border-radius: 6px;
}

/* 文件 chip */
.file-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  padding: 4px 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  background: var(--el-fill-color-light);
  color: var(--el-color-primary);
  text-decoration: none;
  font-size: 13px;
}
.file-chip:hover {
  border-color: var(--el-color-primary);
}
.file-icon {
  flex-shrink: 0;
}
.file-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 220px;
}
</style>

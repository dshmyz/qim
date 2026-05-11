<!--
  TextPreview.vue - 纯文本文件预览组件

  功能：
  - 支持常见文本格式（.txt, .log, .md, .json, .xml, .csv, .yml, .yaml）
  - 使用等宽字体显示
  - 保留格式（换行、空格）
  - 支持语法高亮（JSON）
-->
<template>
  <div class="text-preview">
    <!-- 加载状态 -->
    <div v-if="loading" class="text-loading">
      <LoadingSpinner text="加载文本中..." />
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="text-error">
      <i class="fas fa-exclamation-circle"></i>
      <p>{{ error }}</p>
      <button class="retry-btn" @click="loadTextContent">
        <i class="fas fa-redo"></i> 重试
      </button>
    </div>

    <!-- 文本内容 -->
    <div v-else class="text-content">
      <!-- 工具栏 -->
      <div class="text-toolbar">
        <div class="toolbar-left">
          <span class="file-type-badge">{{ fileTypeLabel }}</span>
          <span class="line-count">{{ lineCount }} 行</span>
        </div>
        <div class="toolbar-right">
          <button class="toolbar-btn" @click="copyContent" title="复制内容">
            <i class="fas fa-copy"></i>
          </button>
          <button class="toolbar-btn" @click="toggleWrap" :title="wrapText ? '取消换行' : '自动换行'">
            <i :class="wrapText ? 'fas fa-align-left' : 'fas fa-align-justify'"></i>
          </button>
        </div>
      </div>

      <!-- 文本显示区域 -->
      <div class="text-container">
        <pre :class="['text-display', { 'wrap-text': wrapText }]">{{ content }}</pre>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import LoadingSpinner from '../../shared/LoadingSpinner.vue'

interface Props {
  url: string
  filename?: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  error: [message: string]
}>()

// 状态
const loading = ref(true)
const error = ref('')
const content = ref('')
const wrapText = ref(true)

// 计算属性
const fileType = computed(() => {
  if (!props.filename) return 'text'
  const ext = props.filename.split('.').pop()?.toLowerCase()
  return ext || 'text'
})

const fileTypeLabel = computed(() => {
  const labels: Record<string, string> = {
    'txt': '纯文本',
    'log': '日志文件',
    'md': 'Markdown',
    'json': 'JSON',
    'xml': 'XML',
    'csv': 'CSV',
    'yml': 'YAML',
    'yaml': 'YAML'
  }
  return labels[fileType.value] || '文本文件'
})

const lineCount = computed(() => {
  if (!content.value) return 0
  return content.value.split('\n').length
})

// 加载文本内容
async function loadTextContent() {
  loading.value = true
  error.value = ''

  try {
    const response = await fetch(props.url)
    if (!response.ok) {
      throw new Error('加载失败')
    }

    const text = await response.text()
    content.value = text
  } catch (err: any) {
    console.error('文本加载失败:', err)
    error.value = '文本加载失败，请重试'
    emit('error', error.value)
  } finally {
    loading.value = false
  }
}

// 复制内容
async function copyContent() {
  try {
    await navigator.clipboard.writeText(content.value)
    // 可以添加提示消息
  } catch (err) {
    console.error('复制失败:', err)
  }
}

// 切换自动换行
function toggleWrap() {
  wrapText.value = !wrapText.value
}

// 监听 URL 变化
watch(() => props.url, () => {
  loadTextContent()
})

// 生命周期
onMounted(() => {
  loadTextContent()
})
</script>

<style scoped>
.text-preview {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.text-loading,
.text-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  gap: 16px;
}

.text-error i {
  font-size: 48px;
  color: var(--error-color);
}

.text-error p {
  font-size: 16px;
  color: var(--text-secondary);
  margin: 0;
}

.retry-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.retry-btn:hover {
  background: var(--primary-hover);
}

.text-content {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.text-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: var(--hover-color);
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}

.toolbar-left,
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.file-type-badge {
  padding: 4px 12px;
  background: var(--primary-color);
  color: white;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.line-count {
  font-size: 13px;
  color: var(--text-secondary);
}

.toolbar-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-color);
  cursor: pointer;
  transition: all 0.2s ease;
}

.toolbar-btn:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.text-container {
  flex: 1;
  overflow: auto;
  background: #f8f9fa;
  padding: 16px;
}

.text-display {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'source-code-pro', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-color);
  margin: 0;
  white-space: pre;
  overflow-x: auto;
}

.text-display.wrap-text {
  white-space: pre-wrap;
  word-wrap: break-word;
}

/* 滚动条样式 */
.text-container::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.text-container::-webkit-scrollbar-track {
  background: transparent;
}

.text-container::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 4px;
}

.text-container::-webkit-scrollbar-thumb:hover {
  background: var(--text-secondary);
}
</style>

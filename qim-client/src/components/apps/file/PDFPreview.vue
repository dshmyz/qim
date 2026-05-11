<!--
  PDFPreview.vue - PDF 文件预览组件

  功能：
  - 使用 pdfjs-dist 渲染 PDF
  - 支持翻页（上一页/下一页）
  - 支持缩放（放大/缩小）
  - 支持全屏查看
  - 显示页码信息
-->
<template>
  <div class="pdf-preview">
    <!-- 加载状态 -->
    <div v-if="loading" class="pdf-loading">
      <LoadingSpinner text="加载 PDF 中..." />
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="pdf-error">
      <i class="fas fa-exclamation-circle"></i>
      <p>{{ error }}</p>
      <button class="retry-btn" @click="loadPDF">
        <i class="fas fa-redo"></i> 重试
      </button>
    </div>

    <!-- PDF 内容 -->
    <div v-else class="pdf-content">
      <!-- 工具栏 -->
      <div class="pdf-toolbar">
        <div class="toolbar-left">
          <button
            class="toolbar-btn"
            :disabled="currentPage <= 1"
            @click="prevPage"
            title="上一页"
          >
            <i class="fas fa-chevron-left"></i>
          </button>
          <span class="page-info">
            {{ currentPage }} / {{ totalPages }}
          </span>
          <button
            class="toolbar-btn"
            :disabled="currentPage >= totalPages"
            @click="nextPage"
            title="下一页"
          >
            <i class="fas fa-chevron-right"></i>
          </button>
        </div>

        <div class="toolbar-center">
          <button
            class="toolbar-btn"
            :disabled="scale <= minScale"
            @click="zoomOut"
            title="缩小"
          >
            <i class="fas fa-search-minus"></i>
          </button>
          <span class="scale-info">{{ Math.round(scale * 100) }}%</span>
          <button
            class="toolbar-btn"
            :disabled="scale >= maxScale"
            @click="zoomIn"
            title="放大"
          >
            <i class="fas fa-search-plus"></i>
          </button>
          <button class="toolbar-btn" @click="resetZoom" title="重置缩放">
            <i class="fas fa-expand"></i>
          </button>
        </div>

        <div class="toolbar-right">
          <button class="toolbar-btn" @click="toggleFullscreen" title="全屏">
            <i :class="isFullscreen ? 'fas fa-compress' : 'fas fa-expand-arrows-alt'"></i>
          </button>
        </div>
      </div>

      <!-- PDF 画布容器 -->
      <div class="pdf-canvas-container" ref="containerRef">
        <canvas ref="canvasRef" class="pdf-canvas"></canvas>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import * as pdfjsLib from 'pdfjs-dist'
import LoadingSpinner from '../../shared/LoadingSpinner.vue'

// 设置 PDF.js worker
pdfjsLib.GlobalWorkerOptions.workerSrc = `//cdnjs.cloudflare.com/ajax/libs/pdf.js/${pdfjsLib.version}/pdf.worker.min.js`

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
const currentPage = ref(1)
const totalPages = ref(0)
const scale = ref(1.5)
const minScale = 0.5
const maxScale = 3.0
const scaleStep = 0.25
const isFullscreen = ref(false)

// PDF 文档和页面
let pdfDoc: pdfjsLib.PDFDocumentProxy | null = null
let pageRendering = false
let pageNumPending: number | null = null

// DOM 引用
const containerRef = ref<HTMLDivElement>()
const canvasRef = ref<HTMLCanvasElement>()

// 加载 PDF
async function loadPDF() {
  loading.value = true
  error.value = ''

  try {
    // 加载 PDF 文档
    const loadingTask = pdfjsLib.getDocument(props.url)
    pdfDoc = await loadingTask.promise
    totalPages.value = pdfDoc.numPages
    currentPage.value = 1

    // 渲染第一页
    await nextTick()
    await renderPage(currentPage.value)
  } catch (err: any) {
    console.error('PDF 加载失败:', err)
    error.value = 'PDF 加载失败，请重试'
    emit('error', error.value)
  } finally {
    loading.value = false
  }
}

// 渲染指定页面
async function renderPage(num: number) {
  if (!pdfDoc || !canvasRef.value) return

  pageRendering = true

  try {
    const page = await pdfDoc.getPage(num)
    const viewport = page.getViewport({ scale: scale.value })

    const canvas = canvasRef.value
    const context = canvas.getContext('2d')

    if (!context) return

    canvas.height = viewport.height
    canvas.width = viewport.width

    const renderContext = {
      canvasContext: context,
      viewport: viewport
    }

    await page.render(renderContext).promise

    pageRendering = false

    // 如果有待渲染的页面，继续渲染
    if (pageNumPending !== null) {
      renderPage(pageNumPending)
      pageNumPending = null
    }
  } catch (err) {
    console.error('页面渲染失败:', err)
    pageRendering = false
  }
}

// 翻页
function prevPage() {
  if (currentPage.value <= 1) return
  currentPage.value--
  queueRenderPage(currentPage.value)
}

function nextPage() {
  if (currentPage.value >= totalPages.value) return
  currentPage.value++
  queueRenderPage(currentPage.value)
}

// 队列渲染（避免同时渲染多个页面）
function queueRenderPage(num: number) {
  if (pageRendering) {
    pageNumPending = num
  } else {
    renderPage(num)
  }
}

// 缩放
function zoomIn() {
  if (scale.value >= maxScale) return
  scale.value = Math.min(scale.value + scaleStep, maxScale)
  queueRenderPage(currentPage.value)
}

function zoomOut() {
  if (scale.value <= minScale) return
  scale.value = Math.max(scale.value - scaleStep, minScale)
  queueRenderPage(currentPage.value)
}

function resetZoom() {
  scale.value = 1.5
  queueRenderPage(currentPage.value)
}

// 全屏
function toggleFullscreen() {
  if (!containerRef.value) return

  if (!isFullscreen.value) {
    if (containerRef.value.requestFullscreen) {
      containerRef.value.requestFullscreen()
    }
  } else {
    if (document.exitFullscreen) {
      document.exitFullscreen()
    }
  }
}

// 监听全屏变化
function handleFullscreenChange() {
  isFullscreen.value = !!document.fullscreenElement
}

// 监听 URL 变化
watch(() => props.url, () => {
  loadPDF()
})

// 监听缩放变化
watch(scale, () => {
  queueRenderPage(currentPage.value)
})

// 生命周期
onMounted(() => {
  loadPDF()
  document.addEventListener('fullscreenchange', handleFullscreenChange)
})

onUnmounted(() => {
  document.removeEventListener('fullscreenchange', handleFullscreenChange)
  if (pdfDoc) {
    pdfDoc.destroy()
    pdfDoc = null
  }
})
</script>

<style scoped>
.pdf-preview {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.pdf-loading,
.pdf-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  gap: 16px;
}

.pdf-error i {
  font-size: 48px;
  color: var(--error-color);
}

.pdf-error p {
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

.pdf-content {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.pdf-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: var(--hover-color);
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}

.toolbar-left,
.toolbar-center,
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
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

.toolbar-btn:hover:not(:disabled) {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.toolbar-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.page-info,
.scale-info {
  font-size: 14px;
  color: var(--text-secondary);
  min-width: 60px;
  text-align: center;
}

.pdf-canvas-container {
  flex: 1;
  overflow: auto;
  display: flex;
  justify-content: center;
  padding: 20px;
  background: #f5f5f5;
}

.pdf-canvas {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  background: white;
}

/* 全屏样式 */
.pdf-canvas-container:fullscreen {
  background: #f5f5f5;
}

.pdf-canvas-container:fullscreen .pdf-canvas {
  max-width: 100%;
  max-height: 100%;
}
</style>

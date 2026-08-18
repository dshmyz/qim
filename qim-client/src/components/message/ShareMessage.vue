<template>
  <!-- 便签消息：直接渲染便签本体（无卡片外壳），限高可滚动，点击弹窗预览完整内容 -->
  <div
    v-if="shareType === 'sticky'"
    ref="stickyNoteRef"
    class="sticky-note-mini sticky-note-mini--message"
    :class="[stickyColorClass, stickyPaperClass]"
    :style="{ ...stickyFontStyle, '--note-rotation': `${rotationSeed}deg` }"
    title="点击预览完整内容"
    @click="showPreview = true"
  >
    <div class="sticky-note-mini-pin">
      <div class="pin-head"></div>
      <div class="pin-shadow"></div>
    </div>
    <div v-if="stickyTitle" class="sticky-note-mini-title">{{ stickyTitle }}</div>
    <div class="sticky-note-mini-content">{{ noteContent }}</div>
    <div v-if="stickyTags.length" class="sticky-note-mini-tags">
      <span v-for="(tag, i) in stickyTags" :key="i" class="sticky-note-mini-tag">{{ tag }}</span>
    </div>
    <div v-if="stickyDate" class="sticky-note-mini-date">{{ stickyDate }}</div>
    <div v-if="stickyScrollable" class="sticky-note-mini-fade"></div>
    <span class="sticky-note-mini-zoom" aria-hidden="true"><i class="fas fa-search-plus"></i></span>
  </div>

  <AttachmentCard
    v-else
    class="share-message"
    :class="{ 'share-message--expanded': isExpanded && shareType === 'note' }"
    :is-self="isSelf"
    @click="toggleContent"
  >
      <i
        class="share-icon"
        :class="shareIconClass"
        :style="{ background: shareIconBg }"
      />
      <template #content>
        <div class="share-title">{{ shareData?.name || content }}</div>
        <div class="share-bottom">
          <div class="share-meta">
            <span class="share-type-label">{{ typeLabel }}</span>
            <span v-if="typeLabel && sizeText"> · </span>
            <span v-if="sizeText">{{ sizeText }}</span>
          </div>
          <div class="share-actions">
            <button v-if="shareType === 'note'" class="attachment-card__btn" @click.stop="showPreview = true" title="预览">
              <i class="fas fa-expand"></i>
            </button>
            <button class="attachment-card__btn" @click.stop="toggleContent" :title="isExpanded ? '收起' : '查看'">
              <i :class="isExpanded ? 'fas fa-chevron-up' : 'fas fa-chevron-down'"></i>
            </button>
          </div>
        </div>
      </template>
      <template v-if="isExpanded && shareType === 'note'" #below>
        <div class="share-expanded-content" @click="handleLinkClick">
          <div ref="noteExpandedRef" class="note-content" v-html="renderedNoteContent"></div>
        </div>
      </template>
    </AttachmentCard>

    <Teleport to="body">
      <div v-if="showPreview" class="note-preview-backdrop" @click.self="showPreview = false">
        <div class="note-preview-dialog" :class="{ 'note-preview-dialog--plain': shareType === 'sticky' }" role="dialog" aria-modal="true">
          <div v-if="shareType === 'note'" class="note-preview-header">
            <h3 class="note-preview-title">{{ shareData?.name || '笔记预览' }}</h3>
            <button class="note-preview-close" @click="showPreview = false">
              <i class="fas fa-times"></i>
            </button>
          </div>
          <div class="note-preview-body" @click="handleLinkClick">
            <div v-if="shareType === 'note'" ref="notePreviewRef" class="note-preview-content" v-html="renderedNoteContent"></div>
            <div v-else-if="shareType === 'sticky'" class="note-preview-content sticky-note-mini sticky-note-mini--preview" :class="[stickyColorClass, stickyPaperClass]" :style="{ ...stickyFontStyle, '--note-rotation': `${rotationSeed}deg` }">
              <div class="sticky-note-mini-pin">
                <div class="pin-head"></div>
                <div class="pin-shadow"></div>
              </div>
              <div v-if="stickyTitle" class="sticky-note-mini-title">{{ stickyTitle }}</div>
              <div class="sticky-note-mini-content">{{ noteContent }}</div>
              <div v-if="stickyTags.length" class="sticky-note-mini-tags">
                <span v-for="(tag, i) in stickyTags" :key="i" class="sticky-note-mini-tag">{{ tag }}</span>
              </div>
              <div v-if="stickyDate || authorName" class="sticky-note-mini-footer">
                <span v-if="stickyDate" class="sticky-note-mini-date">{{ stickyDate }}</span>
                <span v-if="authorName" class="sticky-note-mini-owner">{{ authorName }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { renderMarkdown, handleLinkClick } from '../../composables/useMarkdownRender'
import { useCodeHighlight } from '../../composables/useCodeHighlight'
import { formatFileSize } from '../../utils/fileType'
import { parseStickyShare, parseStickyStyle, parseStickyTags, formatStickyDate } from '../../utils/stickyShare'
import AttachmentCard from './AttachmentCard.vue'

const props = defineProps<{
  content: string
  shareData?: {
    type: string
    name: string
  }
  isSelf?: boolean
  // 便签主人名字：预览弹窗右下角签名展示（旧版本无此 prop 时为空，不显示）
  authorName?: string
}>()

const isExpanded = ref(false)
const showPreview = ref(false)

// 便签消息在聊天里倾斜张贴（基于内容 hash 的确定性角度，同一条消息每次角度一致），
// 旋转中心在顶部图钉处：像便签被图钉钉住后自然歪着；hover 时摆正
const rotationSeed = computed(() => {
  let hash = 0
  for (let i = 0; i < props.content.length; i++) {
    hash = ((hash << 5) - hash) + props.content.charCodeAt(i)
    hash |= 0
  }
  return (Math.abs(hash) % 70) / 10 - 3.5
})

// 便签预览弹窗：点击遮罩或 Esc 关闭（监听器成对注册/移除）
const handlePreviewKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape') showPreview.value = false
}
watch(showPreview, (visible) => {
  if (visible) {
    document.addEventListener('keydown', handlePreviewKeydown)
  } else {
    document.removeEventListener('keydown', handlePreviewKeydown)
  }
})
onBeforeUnmount(() => {
  document.removeEventListener('keydown', handlePreviewKeydown)
})

const shareType = computed(() => {
  return props.shareData?.type || ''
})

const shareIconClass = computed(() => {
  switch (props.shareData?.type) {
    case 'file': return 'fas fa-file'
    case 'note': return 'fas fa-file-alt'
    case 'sticky': return 'fas fa-sticky-note'
    default: return 'fas fa-share-alt'
  }
})

const shareIconBg = computed(() => {
  switch (props.shareData?.type) {
    case 'file': return 'linear-gradient(135deg, #2563eb 0%, #0ea5e9 100%)'
    case 'note': return 'linear-gradient(135deg, #7c3aed 0%, #a855f7 100%)'
    case 'sticky': return 'linear-gradient(135deg, #f59e0b 0%, #f97316 100%)'
    default: return 'linear-gradient(135deg, #7c3aed 0%, #2563eb 100%)'
  }
})

// 便签/笔记分享内容：share 消息 content 为 JSON 字符串，统一解析一次，
// 其余字段由它派生（stickyShare 工具供 ShareMessage 与引用预览共用）
const stickyPayload = computed(() => parseStickyShare(props.content))

const noteContent = computed(() => {
  const payload = stickyPayload.value
  if (payload) return payload.originalContent || payload.content || ''
  // note 类型（分享笔记）不走 sticky 分支，单独兼容
  try {
    const data = JSON.parse(props.content)
    if (data.type === 'note') return data.originalContent || data.content || ''
  } catch {
    /* ignore */
  }
  return ''
})

// 便签样式（颜色/纸张/字体）：旧消息无 style 字段、或数据脏时回退默认黄色
const stickyStyle = computed(() => {
  const payload = stickyPayload.value
  return payload ? parseStickyStyle(payload.style) : null
})

const stickyColorClass = computed(() => `sticky-mini-${stickyStyle.value?.color || 'yellow'}`)
const stickyPaperClass = computed(() => `sticky-mini-paper-${stickyStyle.value?.paperStyle || 'plain'}`)
const stickyFontStyle = computed(() => stickyStyle.value?.fontFamily
  ? { fontFamily: stickyStyle.value.fontFamily }
  : undefined)

const stickyTitle = computed(() => stickyPayload.value?.name || '')

// 便签标签：兼容数组 / JSON 字符串（旧数据）
const stickyTags = computed(() => parseStickyTags(stickyPayload.value?.tags))

// 便签创建日期：像真实便签上的手写日期；旧消息无此字段则不显示
const stickyDate = computed(() => formatStickyDate(stickyPayload.value?.created_at))

// 内容超限可滚动时显示底部渐隐提示（320px 限高 + 内部滚动）
const stickyNoteRef = ref<HTMLElement | null>(null)
const stickyScrollable = ref(false)
const updateStickyScrollable = () => {
  const el = stickyNoteRef.value
  if (!el) return
  stickyScrollable.value = el.scrollHeight > el.clientHeight + 1
}
onMounted(() => {
  updateStickyScrollable()
  nextTick(updateStickyScrollable)
})
watch(noteContent, () => {
  nextTick(updateStickyScrollable)
})

const renderedNoteContent = computed(() => {
  if (!noteContent.value) return ''
  // 统一渲染管道（笔记语义：不解码 mention、不转 Twemoji，与 NoteEditor 预览一致）；
  // 输出已消毒，代码块占位正确恢复，[[title]] 笔记内链带 note-link 保护。
  return renderMarkdown(noteContent.value)
})

// 展开区与预览弹窗两处容器都接代码高亮；useCodeHighlight 同时监听容器挂载，
// 条件挂载（v-if 展开/弹窗）时也能补亮，dataset.highlighted 防重复。
const noteExpandedRef = ref<HTMLElement | null>(null)
const notePreviewRef = ref<HTMLElement | null>(null)
useCodeHighlight(noteExpandedRef, renderedNoteContent)
useCodeHighlight(notePreviewRef, renderedNoteContent)

const toggleContent = () => {
  isExpanded.value = !isExpanded.value
}

const typeLabel = computed(() => {
  switch (props.shareData?.type) {
    case 'file': return '文件'
    case 'note': return '笔记'
    case 'sticky': return '便签'
    case 'message': return '消息'
    default: return '分享'
  }
})

const sizeText = computed(() => {
  try {
    const data = JSON.parse(props.content)
    if (props.shareData?.type === 'file' && typeof data.size === 'number' && data.size > 0) {
      return formatFileSize(data.size)
    }
    if ((props.shareData?.type === 'note' || props.shareData?.type === 'sticky') && data.originalContent) {
      const len = data.originalContent.length
      if (len >= 1024) return `${(len / 1024).toFixed(1)} KB`
      return `${len} 字`
    }
  } catch { /* ignore */ }
  return ''
})
</script>

<style scoped>
.share-message {
  width: 300px;
  max-width: min(100%, 340px);
  min-width: 0;
  box-sizing: border-box;
}

.share-message--expanded {
  width: 100%;
  max-width: 100%;
  align-items: start;
}

.share-icon {
  color: #ffffff;
  font-size: 17px;
}

.share-title {
  font-size: var(--font-size-sm);
  font-weight: 600;
  line-height: 1.35;
  color: var(--text-color);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  letter-spacing: -0.01em;
}

.share-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.share-type-label {
  color: var(--text-secondary);
}

.share-meta {
  min-height: 16px;
  font-size: var(--font-size-xxs);
  line-height: 1.35;
  color: var(--text-secondary);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.share-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

/* Note preview dialog */
.note-preview-backdrop {
  position: fixed;
  inset: 0;
  z-index: 3000;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
}

.note-preview-dialog {
  width: fit-content;
  min-width: 320px;
  max-width: min(680px, 90vw);
  max-height: 80vh;
  background: var(--card-bg);
  border-radius: 14px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* 便签预览弹窗：无 header 标题栏，dialog 透明，便签本体直接浮在遮罩上 */
.note-preview-dialog--plain {
  background: transparent;
  box-shadow: none;
}

.note-preview-dialog--plain .note-preview-body {
  padding: 32px;
  display: flex;
  justify-content: center;
  align-items: flex-start;
  background: transparent;
}

/* 弹窗里的便签：固定纸片宽度，最小高度撑成完整便签比例（内容少时也不是一条窄条），
   内容区弹性撑开、底部行贴底；与消息本体视觉一致 */
.note-preview-dialog--plain .sticky-note-mini {
  width: 300px;
  max-width: 100%;
  min-height: 260px;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.note-preview-dialog--plain .sticky-note-mini-content {
  flex: 1;
}

/* 底部行：左日期、右便签主人签名（签名像真实便签的手写落款） */
.sticky-note-mini-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  padding-right: 22px; /* 避让右下卷角，签名不被盖住 */
  font-size: var(--font-size-tiny);
  opacity: 0.65;
}

.sticky-note-mini-footer .sticky-note-mini-date {
  margin-top: 0;
}

.sticky-note-mini-owner {
  color: var(--sticky-ink, #6d4c41);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 60%;
}

.note-preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}

.note-preview-title {
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--text-color);
  margin: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.note-preview-close {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  background: transparent;
  cursor: pointer;
  font-size: var(--font-size-sm);
  flex-shrink: 0;
}

.note-preview-close:hover {
  background: var(--hover-color);
  color: var(--text-color);
}

.note-preview-body {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.note-preview-content {
  font-size: var(--font-size-xs);
  line-height: 1.7;
  color: var(--text-color);
}

.note-preview-content :deep(h1),
.note-preview-content :deep(h2),
.note-preview-content :deep(h3),
.note-preview-content :deep(h4) {
  margin: 16px 0 8px 0;
  font-weight: 700;
  color: var(--text-color);
  line-height: 1.4;
}

.note-preview-content :deep(h1) { font-size: 1.4em; }
.note-preview-content :deep(h2) { font-size: 1.2em; }
.note-preview-content :deep(h3) { font-size: 1.1em; }
.note-preview-content :deep(h4) { font-size: 1em; }

.note-preview-content :deep(p) {
  margin: 8px 0;
}

.note-preview-content :deep(strong) {
  font-weight: 700;
}

.note-preview-content :deep(em) {
  font-style: italic;
}

.note-preview-content :deep(pre) {
  background: var(--hover-color);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 14px;
  margin: 12px 0;
  overflow-x: auto;
  font-family: 'Fira Code', 'Courier New', monospace;
  font-size: var(--font-size-xs);
  line-height: 1.5;
}

.note-preview-content :deep(code) {
  background: var(--hover-color);
  border-radius: 4px;
  padding: 2px 6px;
  font-family: 'Fira Code', 'Courier New', monospace;
  font-size: var(--font-size-xs);
}

.note-preview-content :deep(pre code) {
  background: none;
  padding: 0;
  border-radius: 0;
}

.note-preview-content :deep(ul),
.note-preview-content :deep(ol) {
  margin: 8px 0;
  padding-left: 24px;
}

.note-preview-content :deep(li) {
  margin: 4px 0;
}

.note-preview-content :deep(a) {
  color: var(--primary-color);
  text-decoration: underline;
}

.note-preview-content :deep(blockquote) {
  margin: 12px 0;
  padding: 10px 16px;
  border-left: 4px solid var(--primary-color);
  background: var(--hover-color);
  border-radius: 0 8px 8px 0;
  color: var(--text-secondary);
}

.note-preview-content :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 12px 0;
  font-size: var(--font-size-sm);
}

.note-preview-content :deep(th),
.note-preview-content :deep(td) {
  border: 1px solid var(--border-color);
  padding: 8px 12px;
  text-align: left;
}

.note-preview-content :deep(th) {
  background: var(--hover-color);
  font-weight: 600;
}

.note-preview-content :deep(hr) {
  border: none;
  border-top: 1px solid var(--border-color);
  margin: 16px 0;
}

.note-preview-content :deep(img) {
  max-width: 100%;
  border-radius: 8px;
  margin: 8px 0;
}

.note-preview-content :deep(input[type="checkbox"]) {
  margin-right: 6px;
}

/* Expanded note/sticky content */
.share-expanded-content {
  margin-top: 8px;
  padding: 12px;
  background: var(--card-bg);
  border-radius: 8px;
  font-size: var(--font-size-xs);
  color: var(--text-color);
  line-height: 1.7;
  border: 1px solid var(--border-color);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  max-height: 400px;
  overflow-y: auto;
}

.share-expanded-content :deep(h1),
.share-expanded-content :deep(h2),
.share-expanded-content :deep(h3),
.share-expanded-content :deep(h4) {
  margin: 14px 0 8px 0;
  font-weight: 700;
  color: var(--text-color);
  line-height: 1.4;
}

.share-expanded-content :deep(h1) { font-size: 1.3em; }
.share-expanded-content :deep(h2) { font-size: 1.15em; }
.share-expanded-content :deep(h3) { font-size: 1.05em; }
.share-expanded-content :deep(h4) { font-size: 1em; }

.share-expanded-content :deep(p) {
  margin: 6px 0;
}

.share-expanded-content :deep(strong) {
  font-weight: 700;
}

.share-expanded-content :deep(em) {
  font-style: italic;
}

.share-expanded-content :deep(pre) {
  background: var(--hover-color);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  padding: 12px;
  margin: 10px 0;
  overflow-x: auto;
  font-family: 'Fira Code', 'Courier New', monospace;
  font-size: var(--font-size-xs);
  line-height: 1.5;
}

.share-expanded-content :deep(code) {
  background: var(--hover-color);
  border-radius: 3px;
  padding: 2px 6px;
  font-family: 'Fira Code', 'Courier New', monospace;
  font-size: var(--font-size-xs);
  color: var(--text-color);
}

.share-expanded-content :deep(pre code) {
  background: none;
  padding: 0;
  border-radius: 0;
}

.share-expanded-content :deep(ul),
.share-expanded-content :deep(ol) {
  margin: 6px 0;
  padding-left: 24px;
}

.share-expanded-content :deep(li) {
  margin: 4px 0;
}

.share-expanded-content :deep(a) {
  color: var(--primary-color);
  text-decoration: underline;
}

.share-expanded-content :deep(blockquote) {
  margin: 10px 0;
  padding: 8px 14px;
  border-left: 4px solid var(--primary-color);
  background: var(--hover-color);
  border-radius: 0 6px 6px 0;
  color: var(--text-secondary);
}

.share-expanded-content :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 10px 0;
  font-size: var(--font-size-xs);
}

.share-expanded-content :deep(th),
.share-expanded-content :deep(td) {
  border: 1px solid var(--border-color);
  padding: 8px 12px;
  text-align: left;
}

.share-expanded-content :deep(th) {
  background: var(--hover-color);
  font-weight: 600;
}

.share-expanded-content :deep(hr) {
  border: none;
  border-top: 1px solid var(--border-color);
  margin: 14px 0;
}

.share-expanded-content :deep(img) {
  max-width: 100%;
  border-radius: 6px;
  margin: 8px 0;
}

.share-expanded-content :deep(input[type="checkbox"]) {
  margin-right: 6px;
}

/* 便签迷你卡片：还原便签颜色/纸张外观。
   背景由全局色板类（sticky-mini-* / sticky-mini-paper-*）提供，此处只管布局。
   三种形态：消息本体（--message 限高滚动，点击弹窗）、预览弹窗（无限制，弹窗 body 滚动） */
.sticky-note-mini {
  border-radius: 2px;
  padding: 30px 14px 14px;
  position: relative;
  box-shadow: 2px 3px 12px rgba(0, 0, 0, 0.12), 0 1px 3px rgba(0, 0, 0, 0.06);
  font-size: var(--font-size-xxs);
  line-height: 1.55;
}

/* 聊天消息里的便签本体：轻微倾斜 + 上下撕纸毛边 + 限高滚动，hover 摆正提示可点击 */
.sticky-note-mini--message {
  width: 300px;
  max-width: min(100%, 340px);
  max-height: 320px;
  overflow-y: auto;
  cursor: pointer;
  transform: rotate(var(--note-rotation, 0deg));
  /* 以顶部图钉为中心旋转：张贴感更真实（底部摆动、顶部固定），hover 时绕同一点摆正 */
  transform-origin: 50% 0;
  transition: transform 0.2s cubic-bezier(0.34, 1.56, 0.64, 1), box-shadow 0.2s ease;
}

.sticky-note-mini--message:hover {
  transform: rotate(0deg) translateY(-2px);
  box-shadow: 4px 6px 20px rgba(0, 0, 0, 0.15), 0 2px 6px rgba(0, 0, 0, 0.08);
}

/* 弹窗里的便签：与消息本体同角度倾斜（同一内容 hash 同一角度），绕顶图钉旋转 */
.sticky-note-mini--preview {
  transform: rotate(var(--note-rotation, 0deg));
  transform-origin: 50% 0;
}

/* 撕纸毛边：顶部一排向下的锯齿阴影带（透出便签本底色），打破规则矩形感。
   消息本体与弹窗预览共用 */
.sticky-note-mini--message::before,
.sticky-note-mini--preview::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 8px;
  background: conic-gradient(from 135deg at 50% 50%, transparent 90deg, rgba(0, 0, 0, 0.08) 0) 0 0 / 16px 8px repeat-x;
  pointer-events: none;
}

/* 纸张卷角：右下角自然卷起（弧线 + 翻起阴影），像便利贴用久了翘起的纸角 */
.sticky-note-mini--message::after,
.sticky-note-mini--preview::after {
  content: '';
  position: absolute;
  bottom: 0;
  right: 0;
  width: 20px;
  height: 20px;
  background: linear-gradient(225deg, transparent 50%, rgba(0, 0, 0, 0.10) 50%);
  border-radius: 0 0 0 12px;
  pointer-events: none;
}

/* 内容可滚动时的底部渐隐提示（避开右下卷角） */
.sticky-note-mini-fade {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 20px;
  height: 26px;
  background: linear-gradient(to bottom, transparent, rgba(0, 0, 0, 0.07));
  pointer-events: none;
}

/* hover 放大镜：右下角（卷角左侧）浮现，提示可点击弹窗预览 */
.sticky-note-mini-zoom {
  position: absolute;
  bottom: 4px;
  right: 26px;
  width: 22px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.18);
  color: #fff;
  font-size: var(--font-size-xxs);
  opacity: 0;
  transform: scale(0.8);
  transition: opacity 0.2s ease, transform 0.2s ease;
  pointer-events: none;
}

.sticky-note-mini--message:hover .sticky-note-mini-zoom {
  opacity: 1;
  transform: scale(1);
}

.sticky-note-mini-pin {
  position: absolute;
  top: 4px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 2;
}

.sticky-note-mini-pin .pin-head {
  width: 18px;
  height: 18px;
  background: radial-gradient(circle at 35% 35%, #ef5350, #c62828);
  border-radius: 50%;
  box-shadow:
    0 2px 6px rgba(198, 40, 40, 0.45),
    inset 0 -1px 2px rgba(0, 0, 0, 0.2),
    inset 0 1px 1px rgba(255, 255, 255, 0.3);
  position: relative;
}

.sticky-note-mini-pin .pin-head::after {
  content: '';
  position: absolute;
  top: 3px;
  left: 5px;
  width: 5px;
  height: 4px;
  background: rgba(255, 255, 255, 0.45);
  border-radius: 50%;
  transform: rotate(-30deg);
}

.sticky-note-mini-pin .pin-shadow {
  position: absolute;
  bottom: -3px;
  left: 50%;
  transform: translateX(-50%);
  width: 14px;
  height: 4px;
  background: rgba(0, 0, 0, 0.15);
  border-radius: 50%;
  filter: blur(1px);
}

.sticky-note-mini-title {
  font-size: var(--font-size-sm);
  font-weight: 700;
  margin: 0 0 8px 0;
  word-break: break-word;
  line-height: 1.3;
}

.sticky-note-mini-content {
  font-size: var(--font-size-xxs);
  line-height: 1.55;
  word-break: break-word;
  white-space: pre-wrap;
  opacity: 0.88;
  color: var(--sticky-ink, #6d4c41);
}

/* 标签：跟随便签色系的半透明小圆点 */
.sticky-note-mini-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 8px;
}

.sticky-note-mini-tag {
  font-size: var(--font-size-tiny);
  padding: 2px 8px;
  border-radius: 10px;
  font-weight: 500;
  color: var(--sticky-ink, #6d4c41);
  background: color-mix(in srgb, var(--sticky-ink, #6d4c41), transparent 88%);
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 便签日期：像手写日期一样低调 */
.sticky-note-mini-date {
  margin-top: 8px;
  font-size: var(--font-size-tiny);
  opacity: 0.6;
}

/* 六色渐变/纸张纹理/暗色色板见 assets/styles/sticky-note-colors.css（全局单源）。
   此处仅保留暗色下需要调整的布局类规则 */
[data-theme="elegant-dark"] .sticky-note-mini {
  box-shadow: none;
}
[data-theme="elegant-dark"] .sticky-note-mini--message:hover {
  box-shadow: none;
}

[data-theme="elegant-dark"] .sticky-note-mini--message::before,
[data-theme="elegant-dark"] .sticky-note-mini--preview::before {
  background: conic-gradient(from 135deg at 50% 50%, transparent 90deg, rgba(255, 255, 255, 0.12) 0) 0 0 / 16px 8px repeat-x;
}

[data-theme="elegant-dark"] .sticky-note-mini--message::after,
[data-theme="elegant-dark"] .sticky-note-mini--preview::after {
  background: linear-gradient(225deg, transparent 50%, rgba(255, 255, 255, 0.10) 50%);
}

[data-theme="elegant-dark"] .sticky-note-mini-fade {
  background: linear-gradient(to bottom, transparent, rgba(255, 255, 255, 0.08));
}

[data-theme="elegant-dark"] .sticky-note-mini-zoom {
  background: rgba(255, 255, 255, 0.25);
}
</style>

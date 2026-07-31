<template>
  <div class="docs-page">
    <div class="docs-header">
      <div class="header-left">
        <el-icon class="header-icon"><Document /></el-icon>
        <div>
          <h1 class="docs-title">{{ currentDoc?.title || '开发文档' }}</h1>
          <p class="docs-desc">CLI 命令行工具与 MCP 协议接入指南</p>
        </div>
      </div>
      <el-radio-group v-model="activeSlug" size="default">
        <el-radio-button
          v-for="doc in docsList"
          :key="doc.slug"
          :value="doc.slug"
        >
          {{ doc.title }}
        </el-radio-button>
      </el-radio-group>
    </div>

    <div v-loading="loading" class="docs-body">
      <div v-if="error" class="docs-error">
        <el-alert :title="error" type="error" show-icon :closable="false" />
      </div>
      <div
        v-else-if="docContent"
        class="markdown-body"
        v-html="renderedContent"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Document } from '@element-plus/icons-vue'
import { getDocContent, listDocs } from '@/api/docs'
import { marked } from 'marked'
import { sanitizeMarkdown } from '@/utils/sanitize'

const route = useRoute()
const router = useRouter()

interface DocItem {
  slug: string
  title: string
}

const docsList = ref<DocItem[]>([
  { slug: 'cli', title: 'CLI 使用指南' },
  { slug: 'mcp', title: 'MCP 接入指南' },
])

const activeSlug = ref(extractSlug(route.path))
const docContent = ref('')
const loading = ref(false)
const error = ref('')

function extractSlug(path: string): string {
  const match = path.match(/\/docs\/(\w+)/)
  return match ? match[1] : 'cli'
}

const currentDoc = computed(() => docsList.value.find(d => d.slug === activeSlug.value))

const renderedContent = computed(() => {
  if (!docContent.value) return ''
  const html = marked.parse(docContent.value, { async: false }) as string
  return sanitizeMarkdown(html)
})

const fetchDoc = async (slug: string) => {
  loading.value = true
  error.value = ''
  try {
    const { data } = await getDocContent(slug)
    docContent.value = data.data?.content || ''
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : '加载文档失败'
    error.value = msg
    docContent.value = ''
  } finally {
    loading.value = false
  }
}

const fetchDocsList = async () => {
  try {
    const { data } = await listDocs()
    if (data.data?.length) {
      docsList.value = data.data
    }
  } catch {
    // 使用默认列表
  }
}

watch(activeSlug, (slug) => {
  router.replace(`/docs/${slug}`)
  fetchDoc(slug)
})

watch(() => route.path, (path) => {
  const slug = extractSlug(path)
  if (slug !== activeSlug.value) {
    activeSlug.value = slug
  }
})

onMounted(() => {
  fetchDocsList()
  fetchDoc(activeSlug.value)
})
</script>

<style scoped>
.docs-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.docs-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-5);
  background: linear-gradient(135deg, #0ea5e9 0%, #6366f1 100%);
  border-radius: var(--radius-xl);
  color: white;
  flex-wrap: wrap;
  gap: var(--space-3);
}

.header-left {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.header-icon {
  font-size: 32px;
  opacity: 0.9;
}

.docs-title {
  margin: 0;
  font-size: 22px;
  font-weight: 800;
  color: white;
  letter-spacing: -0.02em;
}

.docs-desc {
  margin: 0;
  font-size: 13px;
  color: rgba(255, 255, 255, 0.8);
}

:deep(.el-radio-group) {
  --el-radio-button-checked-bg: rgba(255, 255, 255, 0.2);
  --el-radio-button-checked-border-color: rgba(255, 255, 255, 0.4);
  --el-radio-button-checked-text-color: white;
}

:deep(.el-radio-button__inner) {
  background: rgba(255, 255, 255, 0.1) !important;
  border-color: rgba(255, 255, 255, 0.25) !important;
  color: rgba(255, 255, 255, 0.85) !important;
  backdrop-filter: blur(4px);
}

:deep(.el-radio-button__original-radio:checked + .el-radio-button__inner) {
  background: rgba(255, 255, 255, 0.25) !important;
  border-color: rgba(255, 255, 255, 0.5) !important;
  color: white !important;
  box-shadow: -1px 0 0 0 rgba(255, 255, 255, 0.25) !important;
}

.docs-body {
  background: var(--el-bg-color);
  border-radius: var(--radius-xl);
  padding: var(--space-5);
  min-height: 400px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}

.docs-error {
  padding: var(--space-4);
}

/* Markdown 样式 */
:deep(.markdown-body) {
  color: var(--el-text-color-primary);
  font-size: 14px;
  line-height: 1.8;
}

:deep(.markdown-body h1) {
  font-size: 26px;
  font-weight: 800;
  margin: 0 0 16px;
  padding-bottom: 12px;
  border-bottom: 2px solid var(--el-border-color-lighter);
  color: var(--el-text-color-primary);
}

:deep(.markdown-body h2) {
  font-size: 20px;
  font-weight: 700;
  margin: 32px 0 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  color: var(--el-text-color-primary);
}

:deep(.markdown-body h3) {
  font-size: 16px;
  font-weight: 700;
  margin: 24px 0 8px;
  color: var(--el-text-color-primary);
}

:deep(.markdown-body h4) {
  font-size: 14px;
  font-weight: 600;
  margin: 20px 0 8px;
  color: var(--el-text-color-regular);
}

:deep(.markdown-body p) {
  margin: 0 0 12px;
}

:deep(.markdown-body code) {
  background: var(--el-fill-color-light);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
  font-family: 'SF Mono', 'Fira Code', monospace;
  color: #e11d48;
}

:deep(.markdown-body pre) {
  background: var(--el-fill-color-dark);
  border-radius: 8px;
  padding: 16px;
  overflow-x: auto;
  margin: 12px 0;
}

:deep(.markdown-body pre code) {
  background: transparent;
  color: var(--el-text-color-primary);
  padding: 0;
  font-size: 13px;
  line-height: 1.6;
}

:deep(.markdown-body table) {
  width: 100%;
  border-collapse: collapse;
  margin: 12px 0;
  font-size: 13px;
}

:deep(.markdown-body th),
:deep(.markdown-body td) {
  border: 1px solid var(--el-border-color);
  padding: 8px 12px;
  text-align: left;
}

:deep(.markdown-body th) {
  background: var(--el-fill-color-light);
  font-weight: 600;
}

:deep(.markdown-body tr:nth-child(even)) {
  background: var(--el-fill-color-extra-light);
}

:deep(.markdown-body ul),
:deep(.markdown-body ol) {
  padding-left: 24px;
  margin: 8px 0;
}

:deep(.markdown-body li) {
  margin: 4px 0;
}

:deep(.markdown-body blockquote) {
  border-left: 4px solid var(--el-color-primary);
  padding: 8px 16px;
  margin: 12px 0;
  background: var(--el-color-primary-light-9);
  border-radius: 0 8px 8px 0;
  color: var(--el-text-color-regular);
}

:deep(.markdown-body hr) {
  border: none;
  border-top: 1px solid var(--el-border-color-lighter);
  margin: 24px 0;
}

:deep(.markdown-body strong) {
  font-weight: 700;
  color: var(--el-text-color-primary);
}

:deep(.markdown-body a) {
  color: var(--el-color-primary);
  text-decoration: none;
}

:deep(.markdown-body a:hover) {
  text-decoration: underline;
}

@media (max-width: 640px) {
  .docs-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .docs-title {
    font-size: 18px;
  }
}
</style>

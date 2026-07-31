<template>
  <div class="public-docs-page">
    <header class="public-docs-header">
      <div class="header-inner">
        <a href="/admin/landing.html" class="back-link">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M19 12H5"/><polyline points="12 19 5 12 12 5"/>
          </svg>
          返回首页
        </a>
        <div class="header-tabs">
          <a
            v-for="doc in docsList"
            :key="doc.slug"
            :href="`/admin/docs/${doc.slug}`"
            class="tab-item"
            :class="{ active: activeSlug === doc.slug }"
            @click.prevent="switchDoc(doc.slug)"
          >
            {{ doc.title }}
          </a>
        </div>
      </div>
    </header>

    <main class="public-docs-body" v-loading="loading">
      <div v-if="error" class="docs-error">
        <p>{{ error }}</p>
      </div>
      <div
        v-else-if="docContent"
        class="markdown-body"
        v-html="renderedContent"
      />
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getDocContent } from '@/api/docs'
import { marked } from 'marked'
import { sanitizeMarkdown } from '@/utils/sanitize'

const route = useRoute()
const router = useRouter()

const docsList = [
  { slug: 'cli', title: 'CLI 使用指南' },
  { slug: 'mcp', title: 'MCP 接入指南' },
]

const activeSlug = ref(route.params.slug as string || 'cli')
const docContent = ref('')
const loading = ref(false)
const error = ref('')

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

const switchDoc = (slug: string) => {
  activeSlug.value = slug
  router.replace(`/docs/${slug}`)
}

watch(() => route.params.slug, (slug) => {
  if (slug && slug !== activeSlug.value) {
    activeSlug.value = slug as string
    fetchDoc(slug as string)
  }
})

onMounted(() => {
  fetchDoc(activeSlug.value)
})
</script>

<style scoped>
.public-docs-page {
  min-height: 100vh;
  background: #fafbfc;
}

.public-docs-header {
  position: sticky;
  top: 0;
  z-index: 100;
  background: #fff;
  border-bottom: 1px solid #e5e7eb;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.header-inner {
  max-width: 900px;
  margin: 0 auto;
  padding: 0 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
}

.back-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: #6b7280;
  text-decoration: none;
  font-size: 14px;
  transition: color 0.2s;
}

.back-link:hover {
  color: #1a1a2e;
}

.header-tabs {
  display: flex;
  gap: 4px;
}

.tab-item {
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  color: #6b7280;
  text-decoration: none;
  transition: all 0.2s;
}

.tab-item:hover {
  background: #f3f4f6;
  color: #1a1a2e;
}

.tab-item.active {
  background: linear-gradient(135deg, #0ea5e9 0%, #6366f1 100%);
  color: #fff;
}

.public-docs-body {
  max-width: 900px;
  margin: 0 auto;
  padding: 40px 24px 80px;
  min-height: calc(100vh - 56px);
}

.docs-error {
  text-align: center;
  padding: 80px 24px;
  color: #ef4444;
  font-size: 16px;
}

/* Markdown 样式 */
:deep(.markdown-body) {
  color: #1f2937;
  font-size: 15px;
  line-height: 1.8;
}

:deep(.markdown-body h1) {
  font-size: 28px;
  font-weight: 800;
  margin: 0 0 16px;
  padding-bottom: 12px;
  border-bottom: 2px solid #e5e7eb;
  color: #111827;
}

:deep(.markdown-body h2) {
  font-size: 22px;
  font-weight: 700;
  margin: 40px 0 16px;
  padding-bottom: 8px;
  border-bottom: 1px solid #f3f4f6;
  color: #111827;
}

:deep(.markdown-body h3) {
  font-size: 18px;
  font-weight: 700;
  margin: 32px 0 12px;
  color: #111827;
}

:deep(.markdown-body h4) {
  font-size: 15px;
  font-weight: 600;
  margin: 24px 0 8px;
  color: #374151;
}

:deep(.markdown-body p) {
  margin: 0 0 16px;
}

:deep(.markdown-body code) {
  background: #f3f4f6;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  color: #e11d48;
}

:deep(.markdown-body pre) {
  background: #1e293b;
  border-radius: 10px;
  padding: 20px;
  overflow-x: auto;
  margin: 16px 0;
}

:deep(.markdown-body pre code) {
  background: transparent;
  color: #e2e8f0;
  padding: 0;
  font-size: 13px;
  line-height: 1.7;
}

:deep(.markdown-body table) {
  width: 100%;
  border-collapse: collapse;
  margin: 16px 0;
  font-size: 14px;
}

:deep(.markdown-body th),
:deep(.markdown-body td) {
  border: 1px solid #e5e7eb;
  padding: 10px 14px;
  text-align: left;
}

:deep(.markdown-body th) {
  background: #f9fafb;
  font-weight: 600;
}

:deep(.markdown-body tr:nth-child(even)) {
  background: #f9fafb;
}

:deep(.markdown-body ul),
:deep(.markdown-body ol) {
  padding-left: 28px;
  margin: 8px 0 16px;
}

:deep(.markdown-body li) {
  margin: 6px 0;
}

:deep(.markdown-body blockquote) {
  border-left: 4px solid #6366f1;
  padding: 12px 20px;
  margin: 16px 0;
  background: #eef2ff;
  border-radius: 0 10px 10px 0;
  color: #4b5563;
}

:deep(.markdown-body hr) {
  border: none;
  border-top: 1px solid #e5e7eb;
  margin: 32px 0;
}

:deep(.markdown-body strong) {
  font-weight: 700;
  color: #111827;
}

:deep(.markdown-body a) {
  color: #6366f1;
  text-decoration: none;
}

:deep(.markdown-body a:hover) {
  text-decoration: underline;
}

@media (max-width: 640px) {
  .header-inner {
    flex-direction: column;
    height: auto;
    padding: 12px 16px;
    gap: 8px;
  }

  .public-docs-body {
    padding: 24px 16px 60px;
  }
}
</style>

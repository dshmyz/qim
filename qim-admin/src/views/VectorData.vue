<template>
  <div class="vector-data-page">
    <el-card shadow="never">
      <div class="toolbar">
        <div class="toolbar-left">
          <h2 class="page-title">向量数据管理</h2>
          <p class="page-desc">浏览向量数据库中各 collection 的向量内容</p>
        </div>
        <div class="toolbar-right">
          <el-button @click="handleRefresh" :loading="loading">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </div>

      <div class="body-layout">
        <div class="collections-sidebar">
          <div class="collections-header">
            <h3>Collections</h3>
            <el-tag size="small" type="info">{{ collections.length }}</el-tag>
          </div>
          <div class="collection-list" v-loading="loading">
            <div
              v-for="col in collections"
              :key="col.name"
              class="collection-item"
              :class="{ active: selectedCollection === col.name }"
              @click="handleSelect(col.name)"
            >
              <span class="col-name">{{ col.name }}</span>
              <el-tag size="small" round>{{ col.count }}</el-tag>
            </div>
            <el-empty v-if="!loading && collections.length === 0" description="暂无数据" :image-size="60" />
          </div>
        </div>

        <div class="entries-main">
          <template v-if="selectedCollection">
            <div class="entries-header">
              <h3>{{ selectedCollection }}</h3>
              <el-tag size="small" type="info">{{ entries.length }} 条</el-tag>
            </div>

            <div class="entries-list" v-loading="entriesLoading">
              <el-empty v-if="!entriesLoading && entries.length === 0" description="该 Collection 中没有数据" :image-size="60" />

              <div
                v-for="(entry, idx) in entries"
                :key="idx"
                class="entry-card"
              >
                <div class="entry-card-header">
                  <div class="entry-card-title">
                    <span class="entry-card-badge">{{ idx + 1 }}</span>
                    <span class="entry-card-note-id" v-if="entry.doc_id">{{ getNoteIdFromDocId(entry.doc_id) }}</span>
                    <span class="entry-card-chunk-tag" v-if="entry.doc_id">第 {{ getChunkIndex(entry.doc_id) }} 块</span>
                  </div>
                  <el-button size="small" type="primary" text @click="toggleExpand(idx)">
                    {{ isExpanded(idx) ? '收起' : '展开' }}
                  </el-button>
                </div>

                <div class="entry-card-summary">
                  {{ getContentTitle(entry.content) }}
                </div>

                <div v-if="isExpanded(idx)" class="entry-card-full-content">
                  {{ entry.content }}
                </div>

                <div v-if="hasMetadata(entry)" class="entry-meta">
                  <el-tag v-for="(val, key) in entry.metadata" :key="key" size="small" type="info">
                    <span class="meta-key">{{ key }}:</span> {{ val }}
                  </el-tag>
                </div>
              </div>
            </div>
          </template>

          <el-empty v-else description="请从左侧选择一个 Collection 查看数据" :image-size="80" />
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { listCollections, getCollectionData } from '@/api/vectorData'
import type { CollectionInfo, VectorEntry } from '@/api/vectorData'

const loading = ref(false)
const entriesLoading = ref(false)
const collections = ref<CollectionInfo[]>([])
const selectedCollection = ref('')
const entries = ref<VectorEntry[]>([])
const expandedSet = ref<Set<number>>(new Set())

async function handleRefresh() {
  expandedSet.value.clear()
  await fetchCollections()
  if (selectedCollection.value) {
    await fetchEntries(selectedCollection.value)
  }
}

async function fetchCollections() {
  loading.value = true
  try {
    const res = await listCollections()
    collections.value = res.data.data || []
  } catch (e: any) {
    collections.value = []
    ElMessage.error(e?.response?.data?.message || e?.message || '获取集合列表失败')
  } finally {
    loading.value = false
  }
}

async function handleSelect(name: string) {
  selectedCollection.value = name
  expandedSet.value.clear()
  await fetchEntries(name)
}

async function fetchEntries(name: string) {
  entriesLoading.value = true
  try {
    const res = await getCollectionData(name)
    const data = res.data.data
    entries.value = data?.entries || []
  } catch (e: any) {
    entries.value = []
    ElMessage.error(e?.response?.data?.message || e?.message || '获取数据失败')
  } finally {
    entriesLoading.value = false
  }
}

function toggleExpand(idx: number) {
  if (expandedSet.value.has(idx)) {
    expandedSet.value.delete(idx)
  } else {
    expandedSet.value.add(idx)
  }
  expandedSet.value = new Set(expandedSet.value)
}

function isExpanded(idx: number): boolean {
  return expandedSet.value.has(idx)
}

function hasMetadata(entry: VectorEntry): boolean {
  return entry.metadata && Object.keys(entry.metadata).length > 0
}

function getContentTitle(content: string): string {
  const firstLine = content.split('\n')[0].replace(/^#+\s*/, '').trim()
  return firstLine || '（无标题）'
}

function getNoteIdFromDocId(docId: string): string {
  if (docId.startsWith('note_')) {
    return docId.replace(/_chunk_\d+$/, '')
  }
  return docId
}

function getChunkIndex(docId: string): number {
  const match = docId.match(/_chunk_(\d+)$/)
  return match ? parseInt(match[1]) : 0
}

onMounted(() => {
  fetchCollections()
})
</script>

<style scoped>
.vector-data-page {
  height: 100%;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.toolbar-left {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.page-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.page-desc {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.body-layout {
  display: flex;
  gap: 16px;
  height: calc(100vh - 220px);
}

.collections-sidebar {
  width: 260px;
  min-width: 200px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.collections-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--el-border-color-light);
}

.collections-header h3 {
  margin: 0;
  font-size: 14px;
}

.collection-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.collection-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s;
  margin-bottom: 2px;
}

.collection-item:hover {
  background: var(--el-fill-color-light);
}

.collection-item.active {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}

.col-name {
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  margin-right: 8px;
}

.entries-main {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.entries-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.entries-header h3 {
  margin: 0;
  font-size: 15px;
}

.entries-list {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.entry-card {
  border-radius: 8px;
  border: 1px solid #e4e7ed;
  background: #fff;
  padding: 16px;
  margin-bottom: 12px;
  transition: box-shadow 0.2s;
}

.entry-card:hover {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.entry-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.entry-card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.entry-card-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--el-color-primary);
  color: #fff;
  font-size: 12px;
  font-weight: 600;
  flex-shrink: 0;
}

.entry-card-note-id {
  font-size: 13px;
  color: var(--el-text-color-regular);
  font-family: monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.entry-card-chunk-tag {
  font-size: 12px;
  color: var(--el-color-success);
  background: var(--el-color-success-light-9);
  padding: 1px 8px;
  border-radius: 10px;
  flex-shrink: 0;
}

.entry-card-summary {
  font-size: 14px;
  line-height: 1.6;
  color: var(--el-text-color-primary);
  padding: 8px 12px;
  background: #f5f7fa;
  border-radius: 6px;
  border-left: 3px solid var(--el-color-primary);
}

.entry-card-full-content {
  font-size: 13px;
  line-height: 1.8;
  color: var(--el-text-color-regular);
  white-space: pre-wrap;
  padding: 12px;
  margin-top: 8px;
  background: #fafafa;
  border-radius: 6px;
  border: 1px solid #ebeef5;
  max-height: 400px;
  overflow-y: auto;
}

.entry-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 12px;
}

.meta-tag {
  font-size: 11px;
}

.meta-key {
  font-weight: 600;
  margin-right: 2px;
}
</style>
<template>
  <div class="vector-data-app">
    <div class="app-header">
      <h2>向量数据管理</h2>
      <button class="refresh-btn" @click="fetchCollections" :disabled="loading">
        <i class="fas fa-sync-alt" :class="{ spinning: loading }"></i>
        刷新
      </button>
    </div>

    <div v-if="loading && collections.length === 0" class="loading-state">
      <div class="spinner"></div>
      <p>正在加载向量数据...</p>
    </div>

    <div v-else class="layout">
      <div class="collections-panel">
        <h3>Collections ({{ collections.length }})</h3>
        <div class="collection-list">
          <div
            v-for="col in collections"
            :key="col.name"
            class="collection-item"
            :class="{ active: selectedCollection === col.name }"
            @click="selectCollection(col.name)"
          >
            <span class="col-name">{{ col.name }}</span>
            <span class="col-count">{{ col.count }}</span>
          </div>
        </div>
      </div>

      <div class="detail-panel">
        <template v-if="selectedCollection">
          <h3>
            {{ selectedCollection }}
            <span class="entry-count">({{ entries.length }} 条)</span>
          </h3>

          <div v-if="entriesLoading" class="loading-state">
            <div class="spinner"></div>
            <p>加载中...</p>
          </div>

          <div v-else-if="entries.length === 0" class="empty-state">
            该 Collection 中没有数据
          </div>

          <div v-else class="entries-list">
            <div v-for="(entry, idx) in entries" :key="idx" class="entry-card">
              <div class="entry-header">
                <span class="entry-index">#{{ idx + 1 }}</span>
                <span v-if="entry.doc_id" class="entry-doc-id">{{ entry.doc_id }}</span>
              </div>
              <div class="entry-content">{{ entry.content }}</div>
              <div v-if="entry.metadata && Object.keys(entry.metadata).length" class="entry-meta">
                <span v-for="(val, key) in entry.metadata" :key="key" class="meta-tag">
                  <strong>{{ key }}:</strong> {{ val }}
                </span>
              </div>
            </div>
          </div>
        </template>

        <div v-else class="empty-state">
          请从左侧选择一个 Collection 查看数据
        </div>
      </div>
    </div>

    <div v-if="error" class="error-banner">{{ error }}</div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import axios from 'axios'
import { getStoredServerUrl } from '../../composables/useServerUrl'

interface CollectionInfo {
  name: string
  count: number
}

interface VectorEntry {
  doc_id: string
  content: string
  metadata: Record<string, string>
}

const loading = ref(false)
const entriesLoading = ref(false)
const error = ref('')
const collections = ref<CollectionInfo[]>([])
const selectedCollection = ref('')
const entries = ref<VectorEntry[]>([])

const getApi = () => {
  const token = localStorage.getItem('token')
  const serverUrl = getStoredServerUrl()
  return { token, serverUrl, headers: { Authorization: `Bearer ${token}` } }
}

async function fetchCollections() {
  loading.value = true
  error.value = ''
  try {
    const { serverUrl, headers } = getApi()
    const res = await axios.get(`${serverUrl}/api/v1/admin/vector/collections`, { headers })
    if (res.data.code === 200) {
      collections.value = res.data.data || []
    } else {
      error.value = res.data.message || '获取失败'
    }
  } catch (e: any) {
    error.value = e.response?.data?.message || e.message || '请求失败'
  } finally {
    loading.value = false
  }
}

async function selectCollection(name: string) {
  selectedCollection.value = name
  entriesLoading.value = true
  error.value = ''
  try {
    const { serverUrl, headers } = getApi()
    const res = await axios.get(`${serverUrl}/api/v1/admin/vector/collections/${name}`, { headers })
    if (res.data.code === 200) {
      entries.value = res.data.data?.entries || []
    } else {
      error.value = res.data.message || '获取失败'
    }
  } catch (e: any) {
    error.value = e.response?.data?.message || e.message || '请求失败'
  } finally {
    entriesLoading.value = false
  }
}

fetchCollections()
</script>

<style scoped>
.vector-data-app {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  background: var(--card-bg);
}

.app-header h2 {
  margin: 0;
  font-size: 18px;
  color: var(--text-primary);
}

.refresh-btn {
  padding: 6px 14px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--card-bg);
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s;
}

.refresh-btn:hover:not(:disabled) {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.refresh-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.spinning {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px;
  color: var(--text-secondary);
  gap: 12px;
}

.spinner {
  width: 28px;
  height: 28px;
  border: 3px solid var(--border-color);
  border-top-color: var(--primary-color);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.layout {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.collections-panel {
  width: 260px;
  min-width: 200px;
  border-right: 1px solid var(--border-color);
  overflow-y: auto;
  background: var(--card-bg);
}

.collections-panel h3 {
  margin: 0;
  padding: 14px 16px 10px;
  font-size: 13px;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.collection-list {
  padding: 0 8px 8px;
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
  background: var(--hover-color);
}

.collection-item.active {
  background: var(--primary-light);
  color: var(--primary-color);
}

.col-name {
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.col-count {
  font-size: 12px;
  background: var(--border-color);
  color: var(--text-secondary);
  padding: 1px 8px;
  border-radius: 10px;
  min-width: 24px;
  text-align: center;
  margin-left: 8px;
}

.collection-item.active .col-count {
  background: var(--primary-color);
  color: #fff;
}

.detail-panel {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
}

.detail-panel h3 {
  margin: 0 0 16px;
  font-size: 15px;
  color: var(--text-primary);
}

.entry-count {
  font-size: 12px;
  color: var(--text-secondary);
  font-weight: normal;
}

.entries-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.entry-card {
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 14px;
  background: var(--card-bg);
  transition: border-color 0.2s;
}

.entry-card:hover {
  border-color: var(--primary-color);
}

.entry-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}

.entry-index {
  font-size: 12px;
  color: var(--text-secondary);
  background: var(--hover-color);
  padding: 1px 8px;
  border-radius: 4px;
}

.entry-doc-id {
  font-size: 11px;
  color: var(--text-secondary);
  font-family: monospace;
}

.entry-content {
  font-size: 14px;
  line-height: 1.6;
  color: var(--text-primary);
  word-break: break-word;
}

.entry-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
}

.meta-tag {
  font-size: 11px;
  background: var(--hover-color);
  color: var(--text-secondary);
  padding: 2px 8px;
  border-radius: 4px;
}

.meta-tag strong {
  color: var(--text-primary);
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: var(--text-secondary);
  font-size: 14px;
}

.error-banner {
  position: fixed;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  background: #ef4444;
  color: #fff;
  padding: 10px 24px;
  border-radius: 8px;
  font-size: 14px;
  z-index: 3000;
}
</style>
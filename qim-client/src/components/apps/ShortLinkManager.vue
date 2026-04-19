<template>
  <div class="short-link-manager">
    <!-- 短链接管理标题 -->
    <div class="short-link-header">
      <div class="header-left">
        <button class="back-btn" @click="$emit('back')">
          <i class="fas fa-arrow-left"></i>
        </button>
        <div class="short-link-header-info">
          <h2>短链接管理</h2>
          <p class="header-description">生成、管理和跟踪你的短链接</p>
        </div>
      </div>
    </div>
    
    <div class="short-link-content">
      <!-- 生成短链接部分 -->
      <div class="generate-section">
        <h3>生成短链接</h3>
        <div class="generate-form">
          <div class="form-group">
            <label for="original-url">原始URL</label>
            <textarea 
              id="original-url" 
              v-model="originalUrl" 
              class="original-url-input" 
              placeholder="请输入要缩短的URL"
              rows="3"
            ></textarea>
          </div>
          <button 
            class="generate-btn" 
            @click="generateShortLink"
            :disabled="!originalUrl.trim() || isGenerating"
          >
            {{ isGenerating ? '生成中...' : '生成短链接' }}
          </button>
          <div v-if="shortLinkResult" class="short-link-result">
            <label>生成的短链接</label>
            <div class="short-link-output">
              <input type="text" v-model="shortLinkResult" class="short-url-input" readonly />
              <button class="copy-btn" @click="copyShortLink" :disabled="isCopying">
                {{ isCopying ? '已复制' : '复制' }}
              </button>
            </div>
          </div>
        </div>
      </div>
      
      <!-- 短链接列表部分 -->
      <div class="short-link-list-section">
        <h3>我的短链接</h3>
        <div class="short-link-list-header">
          <div class="list-header-original">原始URL</div>
          <div class="list-header-short">短链接</div>
          <div class="list-header-visit">访问次数</div>
          <div class="list-header-created">创建时间</div>
          <div class="list-header-actions">操作</div>
        </div>
        <div class="short-link-list">
          <div 
            v-for="link in shortLinks" 
            :key="link.id" 
            class="short-link-item"
          >
            <div class="short-link-item-original">{{ link.original_url }}</div>
            <div class="short-link-item-short">
              <a :href="link.short_url" target="_blank">{{ link.short_url }}</a>
            </div>
            <div class="short-link-item-visit">{{ link.visit_count }}</div>
            <div class="short-link-item-created">{{ formatDate(link.created_at) }}</div>
            <div class="short-link-item-actions">
              <button class="action-btn copy-btn" @click="copyLink(link.short_url)">
                <i class="fas fa-copy"></i> 复制
              </button>
              <button class="action-btn delete-btn" @click="deleteLink(link.id)">
                <i class="fas fa-trash"></i> 删除
              </button>
            </div>
          </div>
          <div v-if="shortLinks.length === 0 && !isLoading" class="empty-state">
            <div class="empty-icon"><i class="fas fa-link"></i></div>
            <p>暂无短链接</p>
            <p class="empty-hint">生成你的第一个短链接吧</p>
          </div>
          <div v-if="isLoading" class="loading-state">
            <div class="loading-spinner"></div>
            <p>加载中...</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { API_BASE_URL } from '../../config'

// 定义事件
const emit = defineEmits(['back'])

// 服务器URL
const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

// 生成短链接相关状态
const originalUrl = ref('')
const shortLinkResult = ref('')
const isGenerating = ref(false)
const isCopying = ref(false)

// 短链接列表相关状态
const shortLinks = ref<any[]>([])
const isLoading = ref(false)

// 加载短链接列表
const loadShortLinks = async () => {
  try {
    isLoading.value = true
    const token = localStorage.getItem('token')
    const response = await fetch(`${serverUrl.value}/api/v1/shortlinks`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      }
    })

    if (!response.ok) {
      throw new Error('加载短链接列表失败')
    }

    const data = await response.json()
    if (data.code === 0 && Array.isArray(data.data)) {
      shortLinks.value = data.data
    }
  } catch (error) {
    console.error('加载短链接列表失败:', error)
    ElMessage.error('加载短链接列表失败')
  } finally {
    isLoading.value = false
  }
}

// 生成短链接
const generateShortLink = async () => {
  const url = originalUrl.value.trim()
  if (!url) {
    ElMessage.warning('请输入要缩短的URL')
    return
  }

  try {
    isGenerating.value = true
    const token = localStorage.getItem('token')
    const response = await fetch(`${serverUrl.value}/api/v1/shortlinks`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ original_url: url })
    })

    if (!response.ok) {
      throw new Error('生成短链接失败')
    }

    const data = await response.json()
    if (data.code === 0 && data.data) {
      shortLinkResult.value = data.data.short_url
      ElMessage.success('短链接生成成功')
      // 重新加载短链接列表
      await loadShortLinks()
    }
  } catch (error) {
    console.error('生成短链接失败:', error)
    ElMessage.error('生成短链接失败')
  } finally {
    isGenerating.value = false
  }
}

// 复制短链接
const copyShortLink = async () => {
  if (!shortLinkResult.value) return

  try {
    await navigator.clipboard.writeText(shortLinkResult.value)
    isCopying.value = true
    ElMessage.success('短链接已复制到剪贴板')
    setTimeout(() => {
      isCopying.value = false
    }, 2000)
  } catch (error) {
    console.error('复制失败:', error)
    ElMessage.error('复制失败，请手动复制')
  }
}

// 复制指定链接
const copyLink = async (url: string) => {
  try {
    await navigator.clipboard.writeText(url)
    ElMessage.success('短链接已复制到剪贴板')
  } catch (error) {
    console.error('复制失败:', error)
    ElMessage.error('复制失败，请手动复制')
  }
}

// 删除短链接
const deleteLink = async (id: number) => {
  if (!confirm('确定要删除这个短链接吗？')) return

  try {
    const token = localStorage.getItem('token')
    const response = await fetch(`${serverUrl.value}/api/v1/shortlinks/${id}`, {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      }
    })

    if (!response.ok) {
      throw new Error('删除短链接失败')
    }

    const data = await response.json()
    if (data.code === 0) {
      ElMessage.success('短链接删除成功')
      // 重新加载短链接列表
      await loadShortLinks()
    }
  } catch (error) {
    console.error('删除短链接失败:', error)
    ElMessage.error('删除短链接失败')
  }
}

// 格式化日期
const formatDate = (dateString: string) => {
  const date = new Date(dateString)
  return date.toLocaleString()
}

// 组件挂载时加载短链接列表
onMounted(async () => {
  await loadShortLinks()
})
</script>

<style scoped>
.short-link-manager {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.short-link-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  background: var(--card-bg);
  height: 72px;
  box-sizing: border-box;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.short-link-header:hover {
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.back-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: var(--hover-color);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  transition: background 0.2s;
  color: var(--primary-color);
}

.back-btn:hover {
  background: var(--primary-light);
}

.short-link-header-info h2 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 4px 0;
  transition: color 0.3s ease;
}

.header-description {
  margin: 0;
  color: var(--text-secondary);
  font-size: 14px;
}

.short-link-content {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
  background: var(--bg-color);
}

.generate-section {
  background: var(--card-bg);
  border-radius: 6px;
  padding: 20px;
  margin-bottom: 20px;
  border: 1px solid var(--border-color);
}

.generate-section h3 {
  margin: 0 0 16px 0;
  color: var(--text-color);
  font-size: 16px;
  font-weight: 500;
}

.generate-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-group label {
  font-size: 14px;
  color: var(--text-color);
  font-weight: 500;
}

.original-url-input {
  width: 100%;
  padding: 12px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-size: 14px;
  background: var(--bg-color);
  color: var(--text-color);
  resize: vertical;
  min-height: 80px;
}

.original-url-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
}

.generate-btn {
  align-self: flex-start;
  padding: 10px 24px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 4px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.generate-btn:hover:not(:disabled) {
  opacity: 0.9;
  transform: translateY(-1px);
}

.generate-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

.short-link-result {
  margin-top: 8px;
  padding: 16px;
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 4px;
}

.short-link-result label {
  display: block;
  font-size: 14px;
  color: var(--text-color);
  font-weight: 500;
  margin-bottom: 8px;
}

.short-link-output {
  display: flex;
  gap: 12px;
  align-items: center;
}

.short-url-input {
  flex: 1;
  padding: 10px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-size: 14px;
  background: var(--bg-color);
  color: var(--text-color);
}

.copy-btn {
  padding: 8px 16px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 4px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.copy-btn:hover:not(:disabled) {
  opacity: 0.9;
}

.copy-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.short-link-list-section {
  background: var(--card-bg);
  border-radius: 6px;
  padding: 20px;
  border: 1px solid var(--border-color);
}

.short-link-list-section h3 {
  margin: 0 0 16px 0;
  color: var(--text-color);
  font-size: 16px;
  font-weight: 500;
}

.short-link-list-header {
  display: grid;
  grid-template-columns: 3fr 2fr 1fr 1fr 1fr;
  gap: 16px;
  padding: 12px 16px;
  background: var(--bg-color);
  border-radius: 4px 4px 0 0;
  font-weight: 600;
  font-size: 14px;
  color: var(--text-color);
  border: 1px solid var(--border-color);
  border-bottom: none;
}

.short-link-list {
  max-height: 400px;
  overflow-y: auto;
  border: 1px solid var(--border-color);
  border-top: none;
  border-radius: 0 0 4px 4px;
}

.short-link-item {
  display: grid;
  grid-template-columns: 3fr 2fr 1fr 1fr 1fr;
  gap: 16px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  align-items: center;
  transition: background-color 0.2s ease;
}

.short-link-item:hover {
  background: var(--hover-color);
}

.short-link-item:last-child {
  border-bottom: none;
}

.short-link-item-original {
  font-size: 14px;
  color: var(--text-color);
  word-break: break-all;
}

.short-link-item-short a {
  font-size: 14px;
  color: var(--primary-color);
  text-decoration: none;
  word-break: break-all;
}

.short-link-item-short a:hover {
  text-decoration: underline;
}

.short-link-item-visit {
  font-size: 14px;
  color: var(--text-color);
  text-align: center;
}

.short-link-item-created {
  font-size: 14px;
  color: var(--text-secondary);
  text-align: center;
}

.short-link-item-actions {
  display: flex;
  gap: 8px;
  justify-content: center;
}

.action-btn {
  padding: 6px 12px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 4px;
  background: var(--card-bg);
}

.action-btn.copy-btn {
  background: rgba(59, 130, 246, 0.1);
  color: var(--primary-color);
  border-color: rgba(59, 130, 246, 0.3);
}

.action-btn.copy-btn:hover {
  background: rgba(59, 130, 246, 0.2);
  border-color: var(--primary-color);
}

.action-btn.delete-btn {
  background: rgba(245, 108, 108, 0.1);
  color: #f56c6c;
  border-color: rgba(245, 108, 108, 0.3);
}

.action-btn.delete-btn:hover {
  background: rgba(245, 108, 108, 0.2);
  border-color: #f56c6c;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-top: none;
  border-radius: 0 0 4px 4px;
}

.empty-icon {
  font-size: 48px;
  color: var(--text-secondary);
  margin-bottom: 16px;
}

.empty-state p {
  margin: 8px 0;
  color: var(--text-secondary);
  font-size: 14px;
}

.empty-hint {
  font-size: 12px !important;
  opacity: 0.8;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-top: none;
  border-radius: 0 0 4px 4px;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--border-color);
  border-top: 3px solid var(--primary-color);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 16px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.loading-state p {
  color: var(--text-secondary);
  font-size: 14px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .short-link-list-header,
  .short-link-item {
    grid-template-columns: 2fr 1fr 1fr;
    grid-template-areas:
      "original original original"
      "short visit created"
      "actions actions actions";
  }
  
  .short-link-list-header .list-header-original,
  .short-link-item .short-link-item-original {
    grid-area: original;
  }
  
  .short-link-list-header .list-header-short,
  .short-link-item .short-link-item-short {
    grid-area: short;
  }
  
  .short-link-list-header .list-header-visit,
  .short-link-item .short-link-item-visit {
    grid-area: visit;
  }
  
  .short-link-list-header .list-header-created,
  .short-link-item .short-link-item-created {
    grid-area: created;
  }
  
  .short-link-list-header .list-header-actions,
  .short-link-item .short-link-item-actions {
    grid-area: actions;
  }
}
</style>
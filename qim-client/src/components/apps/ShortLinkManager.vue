<template>
  <div class="short-link-manager">
    <!-- 短链接管理标题 -->
    <div class="short-link-header">
      <div class="header-left">
        <button class="back-btn" @click="$emit('back')">
          <i class="fas fa-chevron-left"></i>
        </button>
        <button class="toggle-sidebar-btn" @click="$emit('toggleSidebar')">
          <i class="fas fa-compress"></i>
        </button>
        <div class="short-link-header-info">
          <h2>短链接管理</h2>
          <p class="header-description">生成、管理和跟踪你的短链接</p>
        </div>
      </div>
    </div>
    
    <div class="short-link-content">
      <!-- 快速生成区 -->
      <QuickGenerateSection
        ref="quickGenerateRef"
        :generated-url="generatedUrl"
        :is-generating="isGenerating"
        :is-copying="isCopying"
        @generate="handleGenerate"
        @copy="handleCopyGenerated"
        @batch="handleBatch"
        @advanced="handleAdvanced"
      />
      
      <!-- 统计卡片 -->
      <StatsCards :stats="stats" />
      
      <!-- 列表管理 -->
      <ShortLinkList
        :links="shortLinks"
        @copy="handleCopy"
        @delete="handleDelete"
        @export="handleExport"
        @batch-delete="handleBatchDelete"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import QMessage from '../../utils/qmessage'
import { API_BASE_URL } from '../../config'
import QuickGenerateSection from './shortlink/QuickGenerateSection.vue'
import StatsCards from './shortlink/StatsCards.vue'
import ShortLinkList from './shortlink/ShortLinkList.vue'
import type { ShortLink } from './shortlink/ShortLinkItem.vue'

// 定义事件
const emit = defineEmits(['back', 'toggleSidebar'])

// 服务器URL
const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

// 快速生成相关状态
const generatedUrl = ref('')
const isGenerating = ref(false)
const isCopying = ref(false)
const quickGenerateRef = ref<InstanceType<typeof QuickGenerateSection> | null>(null)

// 短链接列表相关状态
const shortLinks = ref<ShortLink[]>([])
const isLoading = ref(false)

// 统计数据
const stats = computed(() => {
  const totalLinks = shortLinks.value.length
  const totalVisits = shortLinks.value.reduce((sum, link) => sum + link.visit_count, 0)
  const activeLinks = shortLinks.value.filter(link => link.visit_count > 0).length
  
  // 计算今日访问量
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const todayVisits = shortLinks.value.reduce((sum, link) => {
    const linkDate = new Date(link.created_at)
    linkDate.setHours(0, 0, 0, 0)
    if (linkDate.getTime() === today.getTime()) {
      return sum + link.visit_count
    }
    return sum
  }, 0)
  
  return {
    totalLinks,
    totalLinksTrend: 0, // 可以从后端获取趋势数据
    totalVisits,
    totalVisitsTrend: 0,
    todayVisits,
    todayVisitsTrend: 0,
    activeLinks,
    activeRate: totalLinks > 0 ? Math.round((activeLinks / totalLinks) * 100) : 0
  }
})

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
    QMessage.error('加载短链接列表失败')
  } finally {
    isLoading.value = false
  }
}

// 生成短链接
const handleGenerate = async (url: string) => {
  if (!url.trim()) {
    QMessage.warning('请输入要缩短的URL')
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
      body: JSON.stringify({ original_url: url.trim() })
    })

    if (!response.ok) {
      throw new Error('生成短链接失败')
    }

    const data = await response.json()
    if (data.code === 0 && data.data) {
      generatedUrl.value = data.data.short_url
      QMessage.success('短链接生成成功')
      // 重新加载短链接列表
      await loadShortLinks()
      // 清空输入框
      quickGenerateRef.value?.clear()
    }
  } catch (error) {
    console.error('生成短链接失败:', error)
    QMessage.error('生成短链接失败')
  } finally {
    isGenerating.value = false
  }
}

// 复制生成的短链接
const handleCopyGenerated = async () => {
  if (!generatedUrl.value) return

  try {
    await navigator.clipboard.writeText(generatedUrl.value)
    isCopying.value = true
    QMessage.success('短链接已复制到剪贴板')
    setTimeout(() => {
      isCopying.value = false
    }, 2000)
  } catch (error) {
    console.error('复制失败:', error)
    QMessage.error('复制失败，请手动复制')
  }
}

// 复制指定链接
const handleCopy = async (url: string) => {
  try {
    await navigator.clipboard.writeText(url)
    QMessage.success('短链接已复制到剪贴板')
  } catch (error) {
    console.error('复制失败:', error)
    QMessage.error('复制失败，请手动复制')
  }
}

// 删除短链接
const handleDelete = async (id: number) => {
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
      QMessage.success('短链接删除成功')
      // 重新加载短链接列表
      await loadShortLinks()
    }
  } catch (error) {
    console.error('删除短链接失败:', error)
    QMessage.error('删除短链接失败')
  }
}

// 批量生成
const handleBatch = () => {
  QMessage.info('批量生成功能开发中')
}

// 高级选项
const handleAdvanced = () => {
  QMessage.info('高级选项功能开发中')
}

// 导出
const handleExport = () => {
  QMessage.info('导出功能开发中')
}

// 批量删除
const handleBatchDelete = () => {
  QMessage.info('批量操作功能开发中')
}

// ⌘+V 快捷键支持
const handlePaste = async (e: ClipboardEvent) => {
  const clipboardData = e.clipboardData
  if (!clipboardData) return
  
  const pastedText = clipboardData.getData('text')
  
  // 检查是否是URL
  if (pastedText && (pastedText.startsWith('http://') || pastedText.startsWith('https://'))) {
    // 如果输入框没有焦点，自动粘贴并生成
    const activeElement = document.activeElement
    const isInputFocused = activeElement?.tagName === 'INPUT' || activeElement?.tagName === 'TEXTAREA'
    
    if (!isInputFocused) {
      e.preventDefault()
      quickGenerateRef.value?.focus()
      // 延迟触发生成，等待输入框获得焦点
      setTimeout(() => {
        handleGenerate(pastedText)
      }, 100)
    }
  }
}

// 组件挂载时加载短链接列表并注册快捷键
onMounted(async () => {
  await loadShortLinks()
  document.addEventListener('paste', handlePaste)
})

// 组件卸载时移除快捷键监听
onUnmounted(() => {
  document.removeEventListener('paste', handlePaste)
})
</script>

<style scoped>
.short-link-manager {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--bg-color, #f5f7fa);
}

.short-link-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: var(--card-bg, white);
  height: 72px;
  box-sizing: border-box;
  box-shadow: 0 2px 4px var(--shadow-color, rgba(0, 0, 0, 0.05));
  transition: all 0.3s ease;
  border-bottom: 1px solid var(--border-color, #e5e7eb);
}

.short-link-header:hover {
  box-shadow: 0 2px 6px var(--shadow-color-hover, rgba(0, 0, 0, 0.1));
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
  background: var(--hover-color, #f3f4f6);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  transition: background 0.2s;
  color: var(--primary-color, #667eea);
}

.back-btn:hover {
  background: var(--primary-light, rgba(102, 126, 234, 0.1));
}

.toggle-sidebar-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: var(--hover-color, #f3f4f6);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  transition: all 0.3s ease;
  color: var(--primary-color, #667eea);
}

.toggle-sidebar-btn:hover {
  background: var(--primary-light, rgba(102, 126, 234, 0.1));
}

.short-link-header-info h2 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary, #1f2937);
  margin: 0 0 4px 0;
  transition: color 0.3s ease;
}

.header-description {
  margin: 0;
  color: var(--text-secondary, #6b7280);
  font-size: 14px;
}

.short-link-content {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
  background: var(--bg-color, #f5f7fa);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .short-link-header {
    padding: 12px 16px;
    height: auto;
  }
  
  .short-link-content {
    padding: 16px;
  }
}
</style>

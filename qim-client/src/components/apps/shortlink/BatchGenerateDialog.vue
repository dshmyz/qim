<template>
  <QDialog
    :visible="visible"
    title="批量生成短链接"
    width="600px"
    :close-on-click-mask="false"
    @update:visible="$emit('update:visible', $event)"
    @close="handleClose"
  >
    <div class="batch-generate-content">
      <div class="form-group">
        <label class="form-label">输入URL列表（每行一个）</label>
        <textarea
          v-model="urlList"
          class="url-textarea"
          placeholder="https://example1.com&#10;https://example2.com&#10;https://example3.com"
          rows="8"
        ></textarea>
        <div class="url-count">
          已输入 {{ urlCount }} 个URL
        </div>
      </div>

      <!-- 生成结果 -->
      <div v-if="results.length > 0" class="results-section">
        <div class="results-header">
          <span class="results-title">生成结果</span>
          <span class="results-stats">
            成功: {{ successCount }} / 失败: {{ failCount }}
          </span>
        </div>
        <div class="results-list">
          <div
            v-for="(result, index) in results"
            :key="index"
            :class="['result-item', result.success ? 'success' : 'error']"
          >
            <div class="result-original">{{ result.originalUrl }}</div>
            <div v-if="result.success" class="result-short">
              <a :href="result.shortUrl" target="_blank">{{ result.shortUrl }}</a>
              <button class="copy-link-btn" @click="copyResult(result.shortUrl)">
                <i class="fas fa-copy"></i>
              </button>
            </div>
            <div v-else class="result-error">
              <i class="fas fa-exclamation-circle"></i> {{ result.error }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <button class="q-btn q-btn--default" @click="handleClose">取消</button>
      <button
        class="q-btn q-btn--primary"
        :disabled="!hasValidUrls || isGenerating"
        @click="handleGenerate"
      >
        {{ isGenerating ? '生成中...' : '开始生成' }}
      </button>
    </template>
  </QDialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import QDialog from '../../shared/QDialog.vue'
import QMessage from '../../../utils/qmessage'
import { useServerUrl } from '../../../composables/useServerUrl'

interface BatchResult {
  originalUrl: string
  success: boolean
  shortUrl?: string
  error?: string
}

interface Props {
  visible: boolean
}

interface Emits {
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { serverUrl } = useServerUrl()
const urlList = ref('')
const isGenerating = ref(false)
const results = ref<BatchResult[]>([])

// 计算有效的URL数量
const urlCount = computed(() => {
  return urlList.value
    .split('\n')
    .map(url => url.trim())
    .filter(url => url.length > 0 && isValidUrl(url)).length
})

// 是否有有效的URL
const hasValidUrls = computed(() => urlCount.value > 0)

// 成功数量
const successCount = computed(() => results.value.filter(r => r.success).length)

// 失败数量
const failCount = computed(() => results.value.filter(r => !r.success).length)

// 验证URL格式
const isValidUrl = (url: string): boolean => {
  try {
    new URL(url)
    return url.startsWith('http://') || url.startsWith('https://')
  } catch {
    return false
  }
}

// 批量生成
const handleGenerate = async () => {
  const urls = urlList.value
    .split('\n')
    .map(url => url.trim())
    .filter(url => url.length > 0)

  if (urls.length === 0) {
    QMessage.warning('请输入至少一个有效的URL')
    return
  }

  isGenerating.value = true
  results.value = []
  const token = localStorage.getItem('token')

  for (const url of urls) {
    if (!isValidUrl(url)) {
      results.value.push({
        originalUrl: url,
        success: false,
        error: '无效的URL格式'
      })
      continue
    }

    try {
      const response = await fetch(`${serverUrl.value}/api/v1/shortlinks`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ original_url: url })
      })

      const data = await response.json()
      if (data.code === 0 && data.data) {
        results.value.push({
          originalUrl: url,
          success: true,
          shortUrl: data.data.short_url
        })
      } else {
        results.value.push({
          originalUrl: url,
          success: false,
          error: data.message || '生成失败'
        })
      }
    } catch (error) {
      results.value.push({
        originalUrl: url,
        success: false,
        error: '网络请求失败'
      })
    }
  }

  isGenerating.value = false

  if (successCount.value > 0) {
    QMessage.success(`成功生成 ${successCount.value} 个短链接`)
    emit('success')
  }

  if (failCount.value > 0) {
    QMessage.warning(`${failCount.value} 个URL生成失败`)
  }
}

// 复制结果
const copyResult = async (url: string) => {
  try {
    await navigator.clipboard.writeText(url)
    QMessage.success('已复制到剪贴板')
  } catch (error) {
    QMessage.error('复制失败')
  }
}

// 关闭对话框
const handleClose = () => {
  emit('update:visible', false)
  // 重置状态
  urlList.value = ''
  results.value = []
}
</script>

<style scoped>
.batch-generate-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.url-textarea {
  width: 100%;
  padding: 12px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  font-size: 14px;
  font-family: inherit;
  resize: vertical;
  background: var(--input-bg, var(--right-content-bg));
  color: var(--text-color);
  transition: border-color var(--transition-fast);
}

.url-textarea:focus {
  outline: none;
  border-color: var(--primary-color);
}

.url-count {
  font-size: 12px;
  color: var(--text-secondary);
}

.results-section {
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: var(--color-gray-50, #f9fafb);
  border-bottom: 1px solid var(--border-color);
}

.results-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.results-stats {
  font-size: 12px;
  color: var(--text-secondary);
}

.results-list {
  max-height: 200px;
  overflow-y: auto;
}

.result-item {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  font-size: 13px;
}

.result-item:last-child {
  border-bottom: none;
}

.result-item.success {
  background: var(--color-green-50, rgba(34, 197, 94, 0.05));
}

.result-item.error {
  background: var(--color-red-50, rgba(239, 68, 68, 0.05));
}

.result-original {
  color: var(--text-secondary);
  margin-bottom: 4px;
  word-break: break-all;
}

.result-short {
  display: flex;
  align-items: center;
  gap: 8px;
}

.result-short a {
  color: var(--primary-color);
  text-decoration: none;
}

.result-short a:hover {
  text-decoration: underline;
}

.copy-link-btn {
  padding: 4px 8px;
  background: var(--primary-bg, rgba(59, 130, 246, 0.1));
  border: none;
  border-radius: var(--radius-sm);
  color: var(--primary-color);
  cursor: pointer;
  font-size: 12px;
  transition: all var(--transition-fast);
}

.copy-link-btn:hover {
  background: var(--primary-bg-hover, rgba(59, 130, 246, 0.2));
}

.result-error {
  color: var(--danger-color, #ef4444);
  display: flex;
  align-items: center;
  gap: 4px;
}

.q-btn {
  padding: 8px 20px;
  border-radius: var(--radius-md);
  font-size: var(--font-size-base);
  font-weight: var(--font-weight-medium);
  cursor: pointer;
  transition: all var(--transition-fast);
  min-width: 80px;
  border: 1px solid var(--border-color);
}

.q-btn--default {
  background: var(--right-content-bg);
  color: var(--text-color);
}

.q-btn--default:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.q-btn--primary {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.q-btn--primary:hover:not(:disabled) {
  background: var(--primary-dark);
  border-color: var(--primary-dark);
}

.q-btn--primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>

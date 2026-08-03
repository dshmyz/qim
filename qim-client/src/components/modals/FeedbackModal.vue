<template>
  <ModalContainer
    :visible="visible"
    :title="currentView === 'submit' ? '问题反馈' : '我的反馈'"
    width="500px"
    @close="$emit('close')"
    @cancel="$emit('close')"
  >
    <!-- 提交视图 -->
    <div v-if="currentView === 'submit'">
      <div class="feedback-type-section">
        <label class="feedback-label">反馈类型</label>
        <div class="feedback-type-options">
          <button
            v-for="type in feedbackTypes"
            :key="type.value"
            class="feedback-type-btn"
            :class="{ active: selectedType === type.value }"
            @click="selectedType = type.value"
          >
            <i :class="type.icon"></i>
            {{ type.label }}
          </button>
        </div>
      </div>

      <div class="feedback-content-section">
        <label class="feedback-label">反馈内容</label>
        <textarea
          v-model="content"
          class="feedback-textarea"
          :placeholder="placeholderText"
          rows="6"
        ></textarea>
        <div class="feedback-hint">请详细描述您遇到的问题或建议，以便我们更好地改进（至少 10 个字）</div>
      </div>

      <div class="feedback-screenshot-section">
        <label class="feedback-label">截图（可选）</label>
        <div class="feedback-screenshot-upload" @click="triggerScreenshotUpload">
          <i class="fas fa-image"></i>
          <span v-if="!screenshotFile">点击上传截图</span>
          <span v-else>{{ screenshotFile.name }}</span>
        </div>
        <div v-if="screenshotFile" class="feedback-screenshot-preview">
          <img :src="screenshotPreview" alt="截图预览" />
          <button class="feedback-screenshot-remove" @click="removeScreenshot">
            <i class="fas fa-times"></i>
          </button>
        </div>
        <input
          ref="screenshotInput"
          type="file"
          accept="image/*"
          style="display: none"
          @change="handleScreenshotChange"
        />
      </div>
    </div>

    <!-- 列表视图 -->
    <div v-else class="feedback-list-view">
      <div v-if="listLoading" class="feedback-list-loading">
        <i class="fas fa-spinner fa-spin"></i>
        <span>加载中...</span>
      </div>
      <div v-else-if="myFeedbacks.length === 0" class="feedback-list-empty">
        <i class="fas fa-inbox"></i>
        <p>暂无反馈记录</p>
      </div>
      <div v-else class="feedback-list">
        <div
          v-for="item in myFeedbacks"
          :key="item.id"
          class="feedback-item"
          :class="{ expanded: expandedId === item.id }"
          @click="toggleExpand(item.id)"
        >
          <div class="feedback-item-header">
            <div class="feedback-item-meta">
              <i :class="getTypeIcon(item.type)" class="feedback-item-type-icon"></i>
              <span class="feedback-item-time">{{ formatTime(item.createdAt) }}</span>
              <span class="feedback-status-tag" :class="item.status">{{ statusLabel(item.status) }}</span>
            </div>
            <i class="fas fa-chevron-down feedback-item-arrow"></i>
          </div>
          <div class="feedback-item-content">{{ item.content }}</div>
          <div v-if="expandedId === item.id" class="feedback-item-detail">
            <div v-if="item.screenshot" class="feedback-item-screenshot">
              <img :src="getScreenshotUrl(item.screenshot)" alt="截图" @click.stop />
            </div>
            <div v-if="item.reply" class="feedback-item-reply">
              <div class="feedback-reply-label">
                <i class="fas fa-reply"></i>
                管理员回复{{ item.handlerName ? `（${item.handlerName}）` : '' }}
              </div>
              <div class="feedback-reply-content">{{ item.reply }}</div>
            </div>
            <div v-if="!item.reply && item.status === 'pending'" class="feedback-item-no-reply">
              等待处理中...
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <template v-if="currentView === 'submit'">
        <button class="feedback-btn link" @click="switchToList">
          <i class="fas fa-list"></i>
          我的反馈
        </button>
        <div class="footer-right">
          <button class="feedback-btn cancel" @click="$emit('close')">取消</button>
          <button
            class="feedback-btn submit"
            :disabled="!canSubmit || submitting"
            @click="submitFeedback"
          >
            <span v-if="submitting">提交中...</span>
            <span v-else>提交反馈</span>
          </button>
        </div>
      </template>
      <template v-else>
        <button class="feedback-btn link" @click="currentView = 'submit'">
          <i class="fas fa-plus"></i>
          提交新反馈
        </button>
        <button class="feedback-btn cancel" @click="$emit('close')">关闭</button>
      </template>
    </template>
  </ModalContainer>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { request } from '../../composables/useRequest'
import ModalContainer from '../shared/ModalContainer.vue'

const QMessage = (window as any).$QMessage

interface FeedbackItem {
  id: number
  type: string
  content: string
  status: string
  priority: string
  screenshot?: string
  reply?: string
  handlerName?: string
  createdAt: string
  updatedAt: string
}

interface Props {
  visible: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'close': []
  'success': []
}>()

const feedbackTypes = [
  { value: 'bug', label: 'Bug反馈', icon: 'fas fa-bug' },
  { value: 'feature', label: '功能建议', icon: 'fas fa-lightbulb' },
  { value: 'other', label: '其他', icon: 'fas fa-ellipsis-h' }
]

const currentView = ref<'submit' | 'list'>('submit')
const selectedType = ref('feature')
const content = ref('')
const screenshotFile = ref<File | null>(null)
const screenshotPreview = ref('')
const screenshotInput = ref<HTMLInputElement | null>(null)
const submitting = ref(false)

// 列表相关
const myFeedbacks = ref<FeedbackItem[]>([])
const listLoading = ref(false)
const expandedId = ref<number | null>(null)

const placeholderText = computed(() => {
  switch (selectedType.value) {
    case 'bug':
      return '请描述您遇到的bug问题，包括操作步骤、预期结果和实际结果...'
    case 'feature':
      return '请描述您希望增加的功能或改进建议...'
    default:
      return '请详细描述您的问题或建议...'
  }
})

const canSubmit = computed(() => {
  return content.value.trim().length >= 10
})

watch(() => props.visible, (val) => {
  if (val && currentView.value === 'list') {
    loadMyFeedbacks()
  }
})

const switchToList = async () => {
  currentView.value = 'list'
  expandedId.value = null
  await loadMyFeedbacks()
}

const loadMyFeedbacks = async () => {
  listLoading.value = true
  try {
    const res = await request('/api/v1/my-feedbacks?pageSize=50')
    if (res.code === 0) {
      myFeedbacks.value = res.data.list || []
    }
  } catch (error) {
    console.error('获取反馈列表失败:', error)
  }
  listLoading.value = false
}

const toggleExpand = (id: number) => {
  expandedId.value = expandedId.value === id ? null : id
}

const getTypeIcon = (type: string) => {
  switch (type) {
    case 'bug': return 'fas fa-bug'
    case 'feature': return 'fas fa-lightbulb'
    default: return 'fas fa-ellipsis-h'
  }
}

const statusLabel = (status: string) => {
  switch (status) {
    case 'pending': return '待处理'
    case 'processing': return '处理中'
    case 'resolved': return '已解决'
    case 'rejected': return '已拒绝'
    default: return status
  }
}

const formatTime = (dateStr: string) => {
  const d = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时前`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}天前`
  return `${d.getMonth() + 1}/${d.getDate()}`
}

const getScreenshotUrl = (path: string) => {
  if (path.startsWith('http')) return path
  return `/static/${path}`
}

const triggerScreenshotUpload = () => {
  screenshotInput.value?.click()
}

const handleScreenshotChange = (event: Event) => {
  const target = event.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    const file = target.files[0]
    if (file.size > 5 * 1024 * 1024) {
      QMessage.warning('截图大小不能超过5MB')
      return
    }
    screenshotFile.value = file
    const reader = new FileReader()
    reader.onload = (e) => {
      screenshotPreview.value = e.target?.result as string
    }
    reader.readAsDataURL(file)
  }
}

const removeScreenshot = () => {
  screenshotFile.value = null
  screenshotPreview.value = ''
  if (screenshotInput.value) {
    screenshotInput.value.value = ''
  }
}

const submitFeedback = async () => {
  if (!canSubmit.value || submitting.value) return

  submitting.value = true

  try {
    const formData = new FormData()
    formData.append('type', selectedType.value)
    formData.append('content', content.value.trim())
    if (screenshotFile.value) {
      formData.append('screenshot', screenshotFile.value)
    }

    const res = await request('/api/v1/feedbacks', {
      method: 'POST',
      body: formData
    })

    if (res.code === 0) {
      content.value = ''
      screenshotFile.value = null
      screenshotPreview.value = ''
      selectedType.value = 'feature'
      emit('success')
      // 切换到列表视图让用户看到刚提交的反馈
      await switchToList()
    }
  } catch (error) {
    console.error('提交反馈失败:', error)
  }

  submitting.value = false
}
</script>

<style scoped>
.feedback-label {
  display: block;
  margin-bottom: 12px;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.feedback-type-options {
  display: flex;
  gap: 12px;
  margin-bottom: 24px;
}

.feedback-type-btn {
  flex: 1;
  padding: 12px 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--bg-color);
  color: var(--text-color);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  transition: all 0.2s;
}

.feedback-type-btn:hover {
  border-color: var(--primary-color);
}

.feedback-type-btn.active {
  border-color: var(--primary-color);
  background: var(--primary-light);
  color: var(--primary-color);
}

.feedback-type-btn i {
  font-size: 20px;
}

.feedback-textarea {
  width: 100%;
  padding: 12px 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--input-bg);
  color: var(--text-color);
  font-size: 14px;
  font-family: inherit;
  resize: none;
  box-sizing: border-box;
  transition: border-color 0.2s;
}

.feedback-textarea:focus {
  outline: none;
  border-color: var(--primary-color);
}

.feedback-textarea::placeholder {
  color: var(--text-secondary);
}

.feedback-hint {
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-secondary);
}

.feedback-screenshot-upload {
  padding: 24px;
  border: 2px dashed var(--border-color);
  border-radius: 8px;
  text-align: center;
  cursor: pointer;
  color: var(--text-secondary);
  transition: all 0.2s;
}

.feedback-screenshot-upload:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.feedback-screenshot-upload i {
  font-size: 32px;
  margin-bottom: 8px;
  display: block;
}

.feedback-screenshot-preview {
  position: relative;
  margin-top: 12px;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--border-color);
}

.feedback-screenshot-preview img {
  display: block;
  width: 100%;
  max-height: 200px;
  object-fit: contain;
  background: #f5f5f5;
}

.feedback-screenshot-remove {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.6);
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  transition: background 0.2s;
}

.feedback-screenshot-remove:hover {
  background: rgba(0, 0, 0, 0.8);
}

/* 列表视图 */
.feedback-list-view {
  min-height: 300px;
  max-height: 450px;
  overflow-y: auto;
}

.feedback-list-loading,
.feedback-list-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: var(--text-secondary);
  gap: 12px;
}

.feedback-list-loading i,
.feedback-list-empty i {
  font-size: 32px;
  opacity: 0.5;
}

.feedback-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.feedback-item {
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 12px 16px;
  cursor: pointer;
  transition: all 0.2s;
}

.feedback-item:hover {
  border-color: var(--primary-color);
  background: var(--hover-color);
}

.feedback-item-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.feedback-item-meta {
  display: flex;
  align-items: center;
  gap: 8px;
}

.feedback-item-type-icon {
  font-size: 14px;
  color: var(--text-secondary);
}

.feedback-item-time {
  font-size: 12px;
  color: var(--text-secondary);
}

.feedback-status-tag {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 10px;
  font-weight: 500;
}

.feedback-status-tag.pending {
  background: #f0f0f0;
  color: #888;
}

.feedback-status-tag.processing {
  background: #e6f7ff;
  color: #1890ff;
}

.feedback-status-tag.resolved {
  background: #f6ffed;
  color: #52c41a;
}

.feedback-status-tag.rejected {
  background: #fff1f0;
  color: #ff4d4f;
}

.feedback-item-arrow {
  font-size: 12px;
  color: var(--text-secondary);
  transition: transform 0.2s;
}

.feedback-item.expanded .feedback-item-arrow {
  transform: rotate(180deg);
}

.feedback-item-content {
  margin-top: 8px;
  font-size: 14px;
  color: var(--text-color);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.feedback-item.expanded .feedback-item-content {
  -webkit-line-clamp: unset;
}

.feedback-item-detail {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border-color);
}

.feedback-item-screenshot {
  margin-bottom: 12px;
}

.feedback-item-screenshot img {
  max-width: 100%;
  max-height: 200px;
  border-radius: 6px;
  border: 1px solid var(--border-color);
  cursor: pointer;
}

.feedback-item-reply {
  background: var(--hover-color);
  border-radius: 8px;
  padding: 12px;
}

.feedback-reply-label {
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 6px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.feedback-reply-content {
  font-size: 14px;
  color: var(--text-color);
  line-height: 1.6;
  white-space: pre-wrap;
}

.feedback-item-no-reply {
  font-size: 13px;
  color: var(--text-secondary);
  text-align: center;
  padding: 8px;
}

/* Footer */
.feedback-btn {
  padding: 10px 24px;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.2s;
}

.feedback-btn.link {
  background: transparent;
  color: var(--primary-color);
  padding: 10px 12px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.feedback-btn.link:hover {
  opacity: 0.8;
}

.feedback-btn.cancel {
  background: var(--btn-bg);
  color: var(--text-color);
}

.feedback-btn.cancel:hover {
  background: var(--hover-color);
}

.feedback-btn.submit {
  background: var(--primary-color);
  color: white;
  display: flex;
  align-items: center;
  gap: 8px;
}

.feedback-btn.submit:hover:not(:disabled) {
  background: var(--primary-dark);
}

.feedback-btn.submit:disabled {
  background: var(--border-color);
  cursor: not-allowed;
}

.footer-right {
  display: flex;
  gap: 8px;
  margin-left: auto;
}
</style>

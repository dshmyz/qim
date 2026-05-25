<template>
  <div v-if="visible" class="feedback-modal-overlay" @click="$emit('close')">
    <div class="feedback-modal" @click.stop>
      <div class="feedback-modal-header">
        <h3>意见反馈</h3>
        <button class="feedback-modal-close" @click="$emit('close')">×</button>
      </div>
      <div class="feedback-modal-content">
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
      <div class="feedback-modal-footer">
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
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { request } from '../../composables/useRequest'

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

const selectedType = ref('feature')
const content = ref('')
const screenshotFile = ref<File | null>(null)
const screenshotPreview = ref('')
const screenshotInput = ref<HTMLInputElement | null>(null)
const submitting = ref(false)

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

const triggerScreenshotUpload = () => {
  screenshotInput.value?.click()
}

const handleScreenshotChange = (event: Event) => {
  const target = event.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    const file = target.files[0]
    if (file.size > 5 * 1024 * 1024) {
      alert('截图大小不能超过5MB')
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
      emit('close')
    }
  } catch (error) {
    console.error('提交反馈失败:', error)
  }

  submitting.value = false
}
</script>

<style scoped>
.feedback-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.feedback-modal {
  background: var(--modal-bg);
  border-radius: 12px;
  width: 500px;
  max-width: calc(100vw - 40px);
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  overflow: hidden;
}

.feedback-modal-header {
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.feedback-modal-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
}

.feedback-modal-close {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 8px;
  background: none;
  font-size: 24px;
  cursor: pointer;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.feedback-modal-close:hover {
  background: var(--hover-color);
  color: var(--text-color);
}

.feedback-modal-content {
  padding: 24px;
  overflow-y: auto;
  flex: 1;
}

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

.feedback-screenshot-upload.has-image {
  display: none;
}

.feedback-modal-footer {
  padding: 16px 24px;
  border-top: 1px solid var(--border-color);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  flex-shrink: 0;
}

.feedback-btn {
  padding: 10px 24px;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.2s;
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
</style>

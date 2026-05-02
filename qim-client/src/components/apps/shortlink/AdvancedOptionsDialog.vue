<template>
  <QDialog
    :visible="visible"
    title="高级选项"
    width="500px"
    :close-on-click-mask="false"
    @update:visible="$emit('update:visible', $event)"
    @close="handleClose"
  >
    <div class="advanced-options-content">
      <!-- 原始URL -->
      <div class="form-group">
        <label class="form-label">原始URL</label>
        <input
          v-model="formData.originalUrl"
          type="text"
          class="form-input"
          placeholder="https://example.com"
        />
      </div>

      <!-- 自定义后缀 -->
      <div class="form-group">
        <label class="form-label">
          自定义短链接后缀
          <span class="optional-tag">可选</span>
        </label>
        <div class="suffix-input-group">
          <span class="suffix-prefix">{{ baseUrl }}/</span>
          <input
            v-model="formData.customSuffix"
            type="text"
            class="form-input suffix-input"
            placeholder="my-custom-link"
            maxlength="20"
          />
        </div>
        <div class="form-hint">
          只能包含字母、数字、下划线和连字符，最多20个字符
        </div>
      </div>

      <!-- 过期时间 -->
      <div class="form-group">
        <label class="form-label">
          过期时间
          <span class="optional-tag">可选</span>
        </label>
        <div class="expiry-options">
          <label class="radio-option">
            <input
              v-model="formData.expiryType"
              type="radio"
              value="never"
            />
            <span class="radio-label">永不过期</span>
          </label>
          <label class="radio-option">
            <input
              v-model="formData.expiryType"
              type="radio"
              value="custom"
            />
            <span class="radio-label">自定义时间</span>
          </label>
        </div>
        <input
          v-if="formData.expiryType === 'custom'"
          v-model="formData.expiryDate"
          type="datetime-local"
          class="form-input"
          :min="minDateTime"
        />
      </div>

      <!-- 访问密码 -->
      <div class="form-group">
        <label class="form-label">
          访问密码
          <span class="optional-tag">可选</span>
        </label>
        <div class="password-input-group">
          <input
            v-model="formData.password"
            :type="showPassword ? 'text' : 'password'"
            class="form-input"
            placeholder="设置访问密码"
            maxlength="20"
          />
          <button
            class="toggle-password-btn"
            type="button"
            @click="showPassword = !showPassword"
          >
            <i :class="showPassword ? 'fas fa-eye-slash' : 'fas fa-eye'"></i>
          </button>
        </div>
        <div class="form-hint">
          设置后，访问短链接需要输入密码
        </div>
      </div>
    </div>

    <template #footer>
      <button class="q-btn q-btn--default" @click="handleClose">取消</button>
      <button
        class="q-btn q-btn--primary"
        :disabled="!isValid || isGenerating"
        @click="handleGenerate"
      >
        {{ isGenerating ? '生成中...' : '生成短链接' }}
      </button>
    </template>
  </QDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import QDialog from '../../shared/QDialog.vue'
import QMessage from '../../../utils/qmessage'
import { API_BASE_URL } from '../../../config'

interface AdvancedOptions {
  originalUrl: string
  customSuffix: string
  expiryType: 'never' | 'custom'
  expiryDate: string
  password: string
}

interface Props {
  visible: boolean
  initialUrl?: string
}

interface Emits {
  (e: 'update:visible', value: boolean): void
  (e: 'success', shortUrl: string): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)
const baseUrl = computed(() => {
  // 从 serverUrl 提取基础URL用于显示
  try {
    const url = new URL(serverUrl.value)
    return `${url.origin}/s`
  } catch {
    return `${serverUrl.value}/s`
  }
})

const isGenerating = ref(false)
const showPassword = ref(false)

const formData = ref<AdvancedOptions>({
  originalUrl: '',
  customSuffix: '',
  expiryType: 'never',
  expiryDate: '',
  password: ''
})

// 最小日期时间（当前时间）
const minDateTime = computed(() => {
  const now = new Date()
  now.setMinutes(now.getMinutes() - now.getTimezoneOffset())
  return now.toISOString().slice(0, 16)
})

// 验证表单
const isValid = computed(() => {
  if (!formData.value.originalUrl.trim()) return false

  // 验证URL格式
  try {
    const url = new URL(formData.value.originalUrl)
    if (!url.protocol.startsWith('http')) return false
  } catch {
    return false
  }

  // 验证自定义后缀格式
  if (formData.value.customSuffix) {
    const suffixRegex = /^[a-zA-Z0-9_-]+$/
    if (!suffixRegex.test(formData.value.customSuffix)) return false
  }

  // 如果选择了自定义过期时间，验证日期
  if (formData.value.expiryType === 'custom' && formData.value.expiryDate) {
    const expiryDate = new Date(formData.value.expiryDate)
    if (expiryDate <= new Date()) return false
  }

  return true
})

// 监听 visible 变化，初始化表单
watch(() => props.visible, (newVal) => {
  if (newVal && props.initialUrl) {
    formData.value.originalUrl = props.initialUrl
  }
})

// 生成短链接
const handleGenerate = async () => {
  if (!isValid.value) {
    QMessage.warning('请填写正确的信息')
    return
  }

  isGenerating.value = true
  const token = localStorage.getItem('token')

  try {
    const requestBody: Record<string, string | number> = {
      original_url: formData.value.originalUrl.trim()
    }

    // 添加自定义后缀
    if (formData.value.customSuffix) {
      requestBody.custom_suffix = formData.value.customSuffix
    }

    // 添加过期时间
    if (formData.value.expiryType === 'custom' && formData.value.expiryDate) {
      requestBody.expires_at = new Date(formData.value.expiryDate).toISOString()
    }

    // 添加密码
    if (formData.value.password) {
      requestBody.password = formData.value.password
    }

    const response = await fetch(`${serverUrl.value}/api/v1/shortlinks`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(requestBody)
    })

    const data = await response.json()
    if (data.code === 0 && data.data) {
      QMessage.success('短链接生成成功')
      emit('success', data.data.short_url)
      handleClose()
    } else {
      QMessage.error(data.message || '生成失败')
    }
  } catch (error) {
    console.error('生成短链接失败:', error)
    QMessage.error('生成短链接失败')
  } finally {
    isGenerating.value = false
  }
}

// 关闭对话框
const handleClose = () => {
  emit('update:visible', false)
  // 重置表单
  formData.value = {
    originalUrl: '',
    customSuffix: '',
    expiryType: 'never',
    expiryDate: '',
    password: ''
  }
  showPassword.value = false
}
</script>

<style scoped>
.advanced-options-content {
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
  display: flex;
  align-items: center;
  gap: 8px;
}

.optional-tag {
  font-size: 12px;
  font-weight: normal;
  color: var(--text-secondary);
  background: var(--color-gray-100, #f3f4f6);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
}

.form-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  font-size: 14px;
  background: var(--input-bg, var(--right-content-bg));
  color: var(--text-color);
  transition: border-color var(--transition-fast);
}

.form-input:focus {
  outline: none;
  border-color: var(--primary-color);
}

.form-hint {
  font-size: 12px;
  color: var(--text-secondary);
}

.suffix-input-group {
  display: flex;
  align-items: stretch;
}

.suffix-prefix {
  padding: 10px 12px;
  background: var(--color-gray-100, #f3f4f6);
  border: 1px solid var(--border-color);
  border-right: none;
  border-radius: var(--radius-md) 0 0 var(--radius-md);
  font-size: 14px;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  white-space: nowrap;
}

.suffix-input {
  border-radius: 0 var(--radius-md) var(--radius-md) 0;
}

.expiry-options {
  display: flex;
  gap: 20px;
  margin-bottom: 8px;
}

.radio-option {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
}

.radio-option input[type="radio"] {
  cursor: pointer;
}

.radio-label {
  font-size: 14px;
  color: var(--text-color);
}

.password-input-group {
  display: flex;
  align-items: stretch;
}

.password-input-group .form-input {
  border-radius: var(--radius-md) 0 0 var(--radius-md);
  border-right: none;
}

.toggle-password-btn {
  padding: 10px 12px;
  background: var(--color-gray-100, #f3f4f6);
  border: 1px solid var(--border-color);
  border-radius: 0 var(--radius-md) var(--radius-md) 0;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.toggle-password-btn:hover {
  background: var(--color-gray-200, #e5e7eb);
  color: var(--text-color);
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

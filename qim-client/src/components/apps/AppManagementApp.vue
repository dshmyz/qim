<template>
  <div class="app-management-app">
    <AppHeader title="应用管理" @back="$emit('back')" />

    <!-- 应用卡片网格 -->
    <div class="app-card-grid">
      <div
        v-for="app in userApps"
        :key="app.id"
        class="app-card"
        @click="openApp(app)"
      >
        <div class="app-card-actions">
          <button class="app-card-action edit" @click.stop="showEditAppModal(app)" title="编辑">
            <i class="fas fa-pen"></i>
          </button>
          <button class="app-card-action delete" @click.stop="deleteApp(app.id)" title="删除">
            <i class="fas fa-trash"></i>
          </button>
        </div>
        <div class="app-card-icon"><i :class="app.icon || 'fas fa-puzzle-piece'"></i></div>
        <div class="app-card-name">{{ app.name }}</div>
      </div>

      <!-- 创建应用卡片 -->
      <div class="app-card app-card-create" @click="showCreateAppModal">
        <div class="app-card-icon create-icon"><i class="fas fa-plus"></i></div>
        <div class="app-card-name">创建应用</div>
      </div>
    </div>

    <!-- 创建/编辑应用模态框 -->
    <ModalContainer
      :visible="showAppModal"
      :title="selectedApp ? '编辑应用' : '创建应用'"
      @close="closeAppModal"
      :content-style="{ width: '480px', minWidth: '480px' }"
    >
      <div class="app-form">
        <!-- 应用名称 -->
        <div class="app-form-group">
          <label>应用名称 <span class="required-mark">*</span></label>
          <input
            v-model="formData.name"
            type="text"
            class="app-form-input"
            :class="{ 'input-error': formErrors.name }"
            placeholder="给应用起个名字"
            @input="clearError('name')"
          />
          <span v-if="formErrors.name" class="field-error">{{ formErrors.name }}</span>
        </div>

        <!-- 应用链接 -->
        <div class="app-form-group">
          <label>应用链接 <span class="required-mark">*</span></label>
          <input
            v-model="formData.url"
            type="url"
            class="app-form-input"
            :class="{ 'input-error': formErrors.url }"
            placeholder="https://example.com"
            @input="clearError('url')"
          />
          <span v-if="formErrors.url" class="field-error">{{ formErrors.url }}</span>
        </div>

        <!-- 应用图标（双模式：网格选择 / 自定义输入） -->
        <div class="app-form-group">
          <label>
            应用图标
            <span class="optional-tag">可选</span>
          </label>

          <!-- 图标网格模式 -->
          <div v-if="!iconCustomMode" class="icon-picker">
            <div class="icon-grid">
              <button
                v-for="icon in presetIcons"
                :key="icon"
                class="icon-item"
                :class="{ selected: formData.icon === icon }"
                @click="selectIcon(icon)"
                :title="icon"
              >
                <i :class="icon"></i>
              </button>
              <button
                class="icon-item icon-custom-trigger"
                @click="iconCustomMode = true"
                title="自定义图标"
              >
                <i class="fas fa-pen"></i>
                <span class="icon-custom-label">自定义</span>
              </button>
            </div>
            <div v-if="formData.icon" class="icon-selected-hint">
              <i :class="formData.icon"></i>
              <code>{{ formData.icon }}</code>
            </div>
          </div>

          <!-- 自定义输入模式 -->
          <div v-else class="icon-custom-input">
            <div class="icon-custom-row">
              <input
                v-model="formData.icon"
                type="text"
                class="app-form-input"
                placeholder="fas fa-xxx"
              />
              <div class="icon-custom-preview">
                <i v-if="formData.icon" :class="formData.icon"></i>
                <i v-else class="fas fa-question" style="opacity: 0.3"></i>
              </div>
            </div>
            <button class="icon-back-btn" @click="iconCustomMode = false">
              <i class="fas fa-arrow-left"></i> 返回图标列表
            </button>
          </div>
        </div>

        <!-- 打开方式 -->
        <div class="app-form-group">
          <label>打开方式</label>
          <div class="open-type-options">
            <label
              class="open-type-option"
              :class="{ active: formData.openType === 'in-app' }"
            >
              <input
                v-model="formData.openType"
                type="radio"
                value="in-app"
              />
              <i class="fas fa-window-maximize"></i>
              <span>应用内打开</span>
            </label>
            <label
              class="open-type-option"
              :class="{ active: formData.openType === 'external' }"
            >
              <input
                v-model="formData.openType"
                type="radio"
                value="external"
              />
              <i class="fas fa-external-link-alt"></i>
              <span>浏览器打开</span>
            </label>
          </div>
        </div>

        <!-- 实时预览 -->
        <div class="app-preview-section">
          <div class="preview-label">预览</div>
          <div class="app-preview-card">
            <div class="preview-icon">
              <i :class="formData.icon || 'fas fa-puzzle-piece'"></i>
            </div>
            <div class="preview-info">
              <div class="preview-name">{{ formData.name || '应用名称' }}</div>
              <div class="preview-meta">
                <span class="preview-tag">{{ formData.openType === 'in-app' ? '应用内' : '浏览器' }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <button class="app-modal-btn app-cancel-btn" @click="closeAppModal" :disabled="saving">取消</button>
        <button
          class="app-modal-btn app-confirm-btn"
          :class="{ 'btn-disabled': !isValid }"
          :disabled="!isValid || saving"
          @click="saveApp"
        >
          <i v-if="saving" class="fas fa-spinner fa-spin"></i>
          {{ saving ? (selectedApp ? '保存中…' : '创建中…') : (selectedApp ? '保存' : '创建') }}
        </button>
      </template>
    </ModalContainer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { appsApi } from '../../api'
import { logger } from '../../utils/logger';
import QMessageBox from '../../utils/qmessagebox'
import AppHeader from './AppHeader.vue'
import ModalContainer from '../../components/shared/ModalContainer.vue'

const QMessage = (window as any).$QMessage

// 定义事件
const emit = defineEmits(['back'])

// 应用列表
const userApps = ref<any[]>([])

// 预置图标列表
const presetIcons = [
  'fas fa-star',
  'fas fa-bolt',
  'fas fa-robot',
  'fas fa-brain',
  'fas fa-comments',
  'fas fa-envelope',
  'fas fa-phone',
  'fas fa-video',
  'fas fa-calendar',
  'fas fa-tasks',
  'fas fa-folder',
  'fas fa-file-alt',
  'fas fa-chart-bar',
  'fas fa-cog',
  'fas fa-plug',
  'fas fa-puzzle-piece',
  'fas fa-shopping-cart',
  'fas fa-book',
  'fas fa-graduation-cap',
  'fas fa-music',
  'fas fa-camera',
  'fas fa-map-marker-alt',
  'fas fa-globe',
  'fas fa-lock',
  'fas fa-key',
  'fas fa-search',
  'fas fa-bell',
  'fas fa-heart',
  'fas fa-clipboard',
  'fas fa-calculator',
]

// 模态框状态
const showAppModal = ref(false)
const selectedApp = ref<any>(null)
const saving = ref(false)
const iconCustomMode = ref(false)

const formData = ref({
  name: '',
  icon: 'fas fa-star',
  url: '',
  status: 'active',
  openType: 'in-app' // in-app: 在应用内打开, external: 使用默认浏览器打开
})

const formErrors = ref<Record<string, string>>({})

// 表单校验
const isValid = computed(() => {
  const name = formData.value.name.trim()
  const url = formData.value.url.trim()
  if (!name) return false
  if (!url) return false
  // URL 格式校验
  try {
    new URL(url)
  } catch {
    return false
  }
  return true
})

const clearError = (field: string) => {
  delete formErrors.value[field]
}

const validateForm = (): boolean => {
  const errors: Record<string, string> = {}
  const name = formData.value.name.trim()
  const url = formData.value.url.trim()

  if (!name) {
    errors.name = '请输入应用名称'
  }
  if (!url) {
    errors.url = '请输入应用链接'
  } else {
    try {
      new URL(url)
    } catch {
      errors.url = '请输入有效的 URL（以 https:// 开头）'
    }
  }

  formErrors.value = errors
  return Object.keys(errors).length === 0
}

// 加载用户应用
const loadApps = async () => {
  try {
    const appsArray = await appsApi.list()
    userApps.value = appsArray.map((app: any) => ({
      ...app,
      openType: app.open_type || app.openType || 'in-app' // 默认为在应用内打开
    }))
    logger.log('应用列表加载成功:', userApps.value)
  } catch (error) {
    console.error('加载应用列表异常:', error)
  }
}

// 打开应用
const openApp = (app: any) => {
  logger.log('打开应用:', app.name)

  // 触发自定义事件，通知父组件（Main.vue）打开应用
  const event = new CustomEvent('open-user-app', {
    detail: app
  })
  window.dispatchEvent(event)
  logger.log('已发送打开应用事件:', app)
}

// 选择图标
const selectIcon = (icon: string) => {
  formData.value.icon = icon
}

// 判断图标是否在预置列表中
const isPresetIcon = (icon: string) => presetIcons.includes(icon)

// 显示创建应用模态框
const showCreateAppModal = () => {
  formData.value = {
    name: '',
    icon: 'fas fa-star',
    url: '',
    status: 'active',
    openType: 'in-app'
  }
  selectedApp.value = null
  formErrors.value = {}
  iconCustomMode.value = false
  showAppModal.value = true
}

// 显示编辑应用模态框
const showEditAppModal = (app: any) => {
  selectedApp.value = { ...app }
  formData.value = {
    name: app.name,
    icon: app.icon,
    url: app.url,
    status: app.status,
    openType: app.openType || 'in-app' // 默认为在应用内打开
  }
  formErrors.value = {}
  // 如果图标不在预置列表中，自动切换到自定义输入模式
  iconCustomMode.value = !isPresetIcon(app.icon)
  showAppModal.value = true
}

// 关闭应用模态框
const closeAppModal = () => {
  if (saving.value) return
  showAppModal.value = false
  selectedApp.value = null
}

// 保存应用
const saveApp = async () => {
  if (saving.value) return
  if (!validateForm()) return

  saving.value = true
  // 编辑态须在 closeAppModal 清空 selectedApp 之前记录，用于保存后的成功提示文案
  const isEdit = !!selectedApp.value
  try {
    // 转换 openType 为后端期望的 open_type 字段
    const { openType, ...restFormData } = formData.value
    const payload = {
      ...restFormData,
      open_type: openType
    }

    if (selectedApp.value) {
      logger.log('编辑应用:', payload)
      await appsApi.update(selectedApp.value.id, payload)
    } else {
      logger.log('创建应用:', payload)
      await appsApi.create(payload)
    }

    closeAppModal()
    await loadApps()
    window.dispatchEvent(new CustomEvent('refresh-user-apps'))
    logger.log('应用保存成功')
    // closeAppModal 已清空 selectedApp，须先于关闭记录编辑态
    QMessage.success(isEdit ? '应用已更新' : '应用已创建')
  } catch (error: any) {
    const msg = error?.message || ''
    if (msg.includes('权限')) {
      QMessage.error('权限不足，无法执行此操作')
    } else {
      QMessage.error('保存应用失败，请稍后重试')
    }
    console.error('应用保存异常:', error)
  } finally {
    saving.value = false
  }
}

// 删除应用
const deleteApp = async (appId: number) => {
  const result = await QMessageBox.confirm('确定要删除这个应用吗？', '删除应用', { confirmButtonText: '删除', type: 'warning' })
  if (result.action !== 'confirm') return
  try {
    logger.log('删除应用:', appId)
    await appsApi.remove(appId)
    await loadApps()
    // 通知父组件重新加载用户应用
    window.dispatchEvent(new CustomEvent('refresh-user-apps'))
    logger.log('应用删除成功')
    QMessage.success('应用已删除')
  } catch (error: any) {
    const msg = error?.message || ''
    if (msg.includes('权限')) {
      QMessage.error('权限不足，无法执行此操作')
    } else {
      QMessage.error('删除应用失败，请稍后重试')
    }
    console.error('应用删除异常:', error)
  }
}

// 组件挂载时加载应用
onMounted(() => {
  loadApps()
})
</script>

<style scoped>
.app-management-app {
  height: 100%;
  overflow-y: auto;
}

/* ===== 卡片网格 ===== */
.app-card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 16px;
  margin: 20px;
}

.app-card {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 20px 12px;
  background: var(--content-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.app-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  border-color: var(--primary-color);
}

/* 操作按钮（hover 显示） */
.app-card-actions {
  position: absolute;
  top: 6px;
  right: 6px;
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.15s ease;
}

.app-card:hover .app-card-actions {
  opacity: 1;
}

.app-card-action {
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-xxxs);
  transition: all 0.15s ease;
  backdrop-filter: blur(4px);
}

.app-card-action.edit {
  background: rgba(59, 130, 246, 0.15);
  color: var(--primary-color);
}

.app-card-action.edit:hover {
  background: rgba(59, 130, 246, 0.3);
}

.app-card-action.delete {
  background: rgba(239, 68, 68, 0.12);
  color: #ef4444;
}

.app-card-action.delete:hover {
  background: rgba(239, 68, 68, 0.25);
}

/* 图标 */
.app-card-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-2xl);
  color: var(--primary-color);
  background: rgba(59, 130, 246, 0.08);
}

.app-card-name {
  font-size: var(--font-size-xs);
  color: var(--text-primary);
  text-align: center;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 创建卡片 */
.app-card-create {
  border-style: dashed;
  border-color: var(--border-color);
}

.app-card-create:hover {
  border-color: var(--primary-color);
  background: rgba(59, 130, 246, 0.03);
}

.app-card-create .create-icon {
  background: transparent;
  color: var(--text-secondary);
  font-size: var(--font-size-xl);
}

.app-card-create:hover .create-icon {
  color: var(--primary-color);
}

/* ===== 表单样式 ===== */
.app-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.app-form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.app-form-group label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--text-primary);
}

.required-mark {
  color: #ef4444;
  font-size: var(--font-size-sm);
}

.optional-tag {
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary);
  background: var(--hover-color);
  padding: 1px 6px;
  border-radius: 4px;
  font-weight: 400;
}

.app-form-input,
.app-form-select {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: var(--font-size-sm);
  color: var(--text-primary);
  background-color: var(--bg-color);
  transition: all 0.2s ease;
  box-sizing: border-box;
}

.app-form-input:focus,
.app-form-select:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.app-form-input.input-error {
  border-color: #ef4444;
}

.app-form-input.input-error:focus {
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.1);
  border-color: #ef4444;
}

.field-error {
  font-size: var(--font-size-xxs);
  color: #ef4444;
  margin-top: 2px;
}

/* ===== 图标选择器 ===== */
.icon-picker {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.icon-grid {
  display: grid;
  grid-template-columns: repeat(8, 1fr) !important;
  gap: 6px;
}

.icon-item {
  width: 100%;
  height: 44px;
  min-width: 0;
  border: 2px solid transparent;
  border-radius: 8px;
  background: var(--bg-color);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-lg);
  color: var(--text-primary);
  transition: all 0.15s ease;
}

.icon-item:hover {
  border-color: var(--primary-color);
  background: rgba(59, 130, 246, 0.05);
}

.icon-item.selected {
  border-color: var(--primary-color);
  background: rgba(59, 130, 246, 0.1);
  color: var(--primary-color);
}

.icon-custom-trigger {
  flex-direction: column;
  gap: 1px;
  height: 44px;
  border: 2px dashed var(--border-color);
  color: var(--text-secondary);
  font-size: var(--font-size-xxs);
}

.icon-custom-trigger:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.icon-custom-trigger i {
  font-size: var(--font-size-xs);
}

.icon-custom-label {
  font-size: var(--font-size-tiny);
  line-height: 1;
  white-space: nowrap;
}

.icon-selected-hint {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
  padding: 4px 0;
}

.icon-selected-hint i {
  font-size: var(--font-size-base);
  color: var(--primary-color);
}

.icon-selected-hint code {
  font-size: var(--font-size-xxs);
  background: var(--hover-color);
  padding: 2px 6px;
  border-radius: 4px;
}

/* 自定义输入模式 */
.icon-custom-input {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.icon-custom-row {
  display: flex;
  gap: 10px;
  align-items: center;
}

.icon-custom-row .app-form-input {
  flex: 1;
}

.icon-custom-preview {
  width: 40px;
  height: 40px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-xl);
  color: var(--primary-color);
  background: var(--bg-color);
  flex-shrink: 0;
}

.icon-back-btn {
  align-self: flex-start;
  border: none;
  background: none;
  font-size: var(--font-size-xxs);
  color: var(--primary-color);
  cursor: pointer;
  padding: 2px 0;
  display: flex;
  align-items: center;
  gap: 4px;
}

.icon-back-btn:hover {
  text-decoration: underline;
}

/* ===== 打开方式（radio 卡片） ===== */
.open-type-options {
  display: flex;
  gap: 10px;
}

.open-type-option {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border: 2px solid var(--border-color);
  border-radius: 8px;
  cursor: pointer;
  font-size: var(--font-size-xs);
  color: var(--text-primary);
  transition: all 0.15s ease;
  background: var(--bg-color);
}

.open-type-option input[type="radio"] {
  display: none;
}

.open-type-option i {
  font-size: var(--font-size-base);
  color: var(--text-secondary);
}

.open-type-option.active {
  border-color: var(--primary-color);
  background: rgba(59, 130, 246, 0.05);
}

.open-type-option.active i {
  color: var(--primary-color);
}

.open-type-option:hover {
  border-color: var(--primary-color);
}

/* ===== 预览区域 ===== */
.app-preview-section {
  border-top: 1px solid var(--border-color);
  padding-top: 16px;
  margin-top: 4px;
}

.preview-label {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  margin-bottom: 8px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.app-preview-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 8px;
}

.preview-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  background: rgba(59, 130, 246, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-lg);
  color: var(--primary-color);
  flex-shrink: 0;
}

.preview-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.preview-name {
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.preview-meta {
  display: flex;
  gap: 6px;
}

.preview-tag {
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary);
  background: var(--hover-color);
  padding: 1px 6px;
  border-radius: 4px;
}

/* ===== 按钮 ===== */
.app-modal-btn {
  padding: 8px 24px;
  border: none;
  border-radius: 8px;
  font-size: var(--font-size-sm);
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.app-cancel-btn {
  background-color: var(--bg-color);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.app-cancel-btn:hover:not(:disabled) {
  background-color: var(--hover-color);
  color: var(--text-primary);
  border-color: var(--primary-color);
}

.app-confirm-btn {
  background-color: var(--primary-color);
  color: white;
}

.app-confirm-btn:hover:not(:disabled):not(.btn-disabled) {
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
  opacity: 0.95;
}

.app-confirm-btn.btn-disabled,
.app-confirm-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.app-confirm-btn .fa-spin {
  animation: fa-spin 1s infinite linear;
}

@keyframes fa-spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* ===== 响应式设计 ===== */
@media (max-width: 768px) {
  .app-card-grid {
    grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
    gap: 12px;
  }

  .icon-grid {
    grid-template-columns: repeat(6, 1fr);
  }

  .open-type-options {
    flex-direction: column;
  }
}
</style>

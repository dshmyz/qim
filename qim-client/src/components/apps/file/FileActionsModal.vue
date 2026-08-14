<template>
  <ModalContainer
    :visible="visible"
    :title="modalTitle"
    width="480px"
    @close="handleClose"
    @cancel="handleClose"
  >
    <!-- 操作类型选择 -->
    <div class="action-tabs">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        :class="['tab-btn', { active: activeTab === tab.key }]"
        @click="activeTab = tab.key"
      >
        <i :class="tab.icon"></i>
        {{ tab.label }}
      </button>
    </div>

    <!-- 重命名表单 -->
    <div v-if="activeTab === 'rename'" class="form-section">
      <div class="form-group">
        <label for="rename-input">新名称</label>
        <input
          id="rename-input"
          ref="renameInputRef"
          type="text"
          v-model="newName"
          class="form-input"
          placeholder="请输入新名称"
          @keyup.enter="handleSubmit"
        />
        <p v-if="file" class="original-name">原名称：{{ file.name }}</p>
      </div>
    </div>

    <!-- 移动表单 -->
    <div v-if="activeTab === 'move'" class="form-section">
      <div class="form-group">
        <label for="move-folder-select">目标文件夹</label>
        <select
          id="move-folder-select"
          v-model="targetFolderId"
          class="form-input"
        >
          <option :value="null">根目录</option>
          <option v-for="folder in folders" :key="folder.id" :value="folder.id">
            {{ folder.name }}
          </option>
        </select>
        <p v-if="file" class="current-location">
          当前位置：{{ currentFolderName }}
        </p>
      </div>
    </div>

    <template #footer>
      <button class="modal-btn cancel-btn" @click="handleClose">取消</button>
      <button
        class="modal-btn confirm-btn"
        @click="handleSubmit"
        :disabled="submitting || !canSubmit"
      >
        {{ submitting ? '提交中...' : submitButtonText }}
      </button>
    </template>
  </ModalContainer>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { fileApi, type FileItem, type FolderItem } from '../../../api/file'
import QMessage from '../../../utils/qmessage'
import ModalContainer from '../../shared/ModalContainer.vue'

interface Props {
  visible: boolean
  file?: FileItem | null
  folders?: FolderItem[]
}

const props = withDefaults(defineProps<Props>(), {
  file: null,
  folders: () => []
})

const emit = defineEmits<{
  close: []
  success: []
}>()

const tabs = [
  { key: 'rename', label: '重命名', icon: 'fas fa-edit' },
  { key: 'move', label: '移动', icon: 'fas fa-arrows-alt' }
] as const

const activeTab = ref<'rename' | 'move'>('rename')
const newName = ref('')
const targetFolderId = ref<number | null>(null)
const submitting = ref(false)
const renameInputRef = ref<HTMLInputElement | null>(null)

// 根据当前文件信息计算原文件夹名称
const currentFolderName = computed(() => {
  if (!props.file) return '根目录'
  if (props.file.folder_id === null) return '根目录'
  const folder = props.folders.find(f => f.id === props.file?.folder_id)
  return folder?.name ?? '根目录'
})

// 模态框标题根据当前tab动态变化
const modalTitle = computed(() => {
  if (activeTab.value === 'rename') return '重命名文件'
  return '移动文件'
})

// 提交按钮文字
const submitButtonText = computed(() => {
  if (activeTab.value === 'rename') return '保存'
  return '移动'
})

// 验证是否可以提交
const canSubmit = computed(() => {
  if (activeTab.value === 'rename') {
    return newName.value.trim().length > 0
  }
  // 移动操作始终可提交（允许移动到根目录）
  return true
})

// 监听 visible 变化，重置表单
watch(() => props.visible, async (newVal) => {
  if (newVal) {
    activeTab.value = 'rename'
    newName.value = props.file?.name ?? ''
    targetFolderId.value = props.file?.folder_id ?? null
    await nextTick()
    renameInputRef.value?.focus()
  }
})

function handleClose() {
  emit('close')
}

async function handleSubmit() {
  if (!props.file) return

  submitting.value = true
  try {
    if (activeTab.value === 'rename') {
      const trimmedName = newName.value.trim()
      if (!trimmedName) {
        QMessage.error('请输入文件名称')
        return
      }
      await fileApi.updateFile(props.file.id, { name: trimmedName })
      QMessage.success('文件已重命名')
    } else {
      await fileApi.updateFile(props.file.id, { folder_id: targetFolderId.value })
      QMessage.success('文件已移动')
    }
    emit('success')
    emit('close')
  } catch (error: any) {
    const message = error?.response?.data?.message || '操作失败，请稍后重试'
    QMessage.error(message)
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
/* 操作 tabs */
.action-tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 20px;
  border-bottom: 1px solid var(--border-color);
  padding-bottom: 12px;
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: none;
  border-radius: 6px;
  font-size: var(--font-size-sm);
  cursor: pointer;
  transition: all 0.2s ease;
  background: var(--hover-color);
  color: var(--text-color);
}

.tab-btn i {
  font-size: var(--font-size-xxs);
}

.tab-btn:hover {
  background: var(--primary-light);
  color: var(--primary-color);
}

.tab-btn.active {
  background: var(--primary-color);
  color: white;
}

.tab-btn.active:hover {
  background: var(--primary-hover);
  color: white;
}

.form-section {
  min-height: 80px;
}

.form-group {
  margin-bottom: 12px;
}

.form-group label {
  display: block;
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--text-color);
  margin-bottom: 8px;
}

.form-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: var(--font-size-sm);
  background: var(--input-bg);
  color: var(--text-color);
  transition: all 0.2s ease;
  box-sizing: border-box;
}

.form-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.1);
}

.form-input::placeholder {
  color: var(--text-secondary);
}

select.form-input {
  cursor: pointer;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%23999' d='M6 8L1 3h10z'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
  padding-right: 36px;
}

.original-name,
.current-location {
  margin: 8px 0 0;
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}

.modal-btn {
  padding: 8px 24px;
  border-radius: 8px;
  font-size: var(--font-size-sm);
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  border: 1px solid transparent;
}

.modal-btn.cancel-btn {
  background: var(--card-bg);
  color: var(--text-color);
  border-color: var(--border-color);
}

.modal-btn.cancel-btn:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.modal-btn.confirm-btn {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.modal-btn.confirm-btn:hover:not(:disabled) {
  background: var(--primary-hover);
}

.modal-btn.confirm-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

@media (max-width: 480px) {
  .modal-btn {
    width: 100%;
  }
}
</style>

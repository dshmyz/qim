<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click="handleClose">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>{{ isEditing ? '编辑文件夹' : '创建文件夹' }}</h3>
          <button class="modal-close" @click="handleClose">&times;</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label for="folder-name-input">文件夹名称</label>
            <input
              id="folder-name-input"
              ref="nameInputRef"
              type="text"
              v-model="folderName"
              placeholder="请输入文件夹名称"
              class="form-input"
              @keyup.enter="handleSubmit"
            />
          </div>
          <div v-if="!isEditing" class="form-group">
            <label for="parent-folder-select">上级文件夹</label>
            <select
              id="parent-folder-select"
              v-model="parentFolderId"
              class="form-input"
            >
              <option :value="null">根目录</option>
              <option v-for="folder in folders" :key="folder.id" :value="folder.id">
                {{ folder.name }}
              </option>
            </select>
          </div>
        </div>
        <div class="modal-footer">
          <button class="modal-btn cancel-btn" @click="handleClose">取消</button>
          <button class="modal-btn confirm-btn" @click="handleSubmit" :disabled="submitting">
            {{ submitting ? '提交中...' : (isEditing ? '保存' : '创建') }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { folderApi, type FolderItem } from '../../../api/file'
import QMessage from '../../../utils/qmessage'

interface Props {
  visible: boolean
  isEditing?: boolean
  folder?: FolderItem | null
  folders?: FolderItem[]
}

const props = withDefaults(defineProps<Props>(), {
  isEditing: false,
  folder: null,
  folders: () => []
})

const emit = defineEmits<{
  close: []
  success: []
}>()

const folderName = ref('')
const parentFolderId = ref<number | null>(null)
const submitting = ref(false)
const nameInputRef = ref<HTMLInputElement | null>(null)

// 监听 visible 变化，初始化表单
watch(() => props.visible, async (newVal) => {
  if (newVal) {
    if (props.isEditing && props.folder) {
      folderName.value = props.folder.name
      parentFolderId.value = props.folder.parent_id
    } else {
      folderName.value = ''
      parentFolderId.value = null
    }
    await nextTick()
    nameInputRef.value?.focus()
  }
})

function handleClose() {
  emit('close')
}

async function handleSubmit() {
  const trimmedName = folderName.value.trim()
  if (!trimmedName) {
    QMessage.error('请输入文件夹名称')
    return
  }

  submitting.value = true
  try {
    if (props.isEditing && props.folder) {
      await folderApi.updateFolder(props.folder.id, {
        name: trimmedName
      })
      QMessage.success('文件夹名称已更新')
    } else {
      await folderApi.createFolder(trimmedName, parentFolderId.value)
      QMessage.success('文件夹创建成功')
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
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
  animation: fadeIn 0.3s ease;
}

.modal-content {
  background: var(--card-bg);
  border-radius: 12px;
  width: 90%;
  max-width: 480px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
  animation: slideIn 0.3s ease;
  overflow: hidden;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
  margin: 0;
}

.modal-close {
  background: none;
  border: none;
  font-size: 24px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
}

.modal-close:hover {
  color: var(--text-color);
  background: var(--hover-color);
}

.modal-body {
  padding: 24px;
}

.form-group {
  margin-bottom: 16px;
}

.form-group:last-child {
  margin-bottom: 0;
}

.form-group label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
  margin-bottom: 8px;
}

.form-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: 14px;
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

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 24px;
  border-top: 1px solid var(--border-color);
}

.modal-btn {
  padding: 8px 24px;
  border-radius: 8px;
  font-size: 14px;
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

.modal-btn.confirm-btn:hover {
  background: var(--primary-hover);
}

.modal-btn.confirm-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slideIn {
  from {
    transform: translateY(-20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

@media (max-width: 480px) {
  .modal-content {
    width: 95%;
    margin: 16px;
  }

  .modal-header,
  .modal-body,
  .modal-footer {
    padding-left: 16px;
    padding-right: 16px;
  }

  .modal-footer {
    flex-direction: column;
  }

  .modal-btn {
    width: 100%;
  }
}
</style>

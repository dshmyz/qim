<template>
  <ModalContainer
    :visible="visible"
    :title="isEditing ? '重命名文件夹' : '创建文件夹'"
    width="480px"
    @close="handleClose"
    @cancel="handleClose"
  >
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
    <div v-if="!isEditing && fixedParentId === undefined" class="form-group">
      <label for="parent-folder-select">上级文件夹</label>
      <FolderTreeSelect
        id="parent-folder-select"
        v-model="parentFolderId"
        :options="options"
      />
    </div>

    <template #footer>
      <button class="modal-btn cancel-btn" @click="handleClose">取消</button>
      <button class="modal-btn confirm-btn" @click="handleSubmit" :disabled="submitting">
        {{ submitting ? '提交中...' : (isEditing ? '保存' : '创建') }}
      </button>
    </template>
  </ModalContainer>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import QMessage from '../../../utils/qmessage'
import ModalContainer from '../../shared/ModalContainer.vue'
import FolderTreeSelect from './FolderTreeSelect.vue'
import { useFolderTree, type FolderNode, type FolderOption } from '../../../composables/useFolderTree'

interface Props {
  visible: boolean
  isEditing?: boolean
  folder?: FolderNode | null
  options?: FolderOption[]
  /** 固定父级（侧栏新建：跟随选中节点），有值时隐藏父级选择器 */
  fixedParentId?: number | null
  /** 父级选择器初始值（文件列表新建：默认当前所在文件夹，可改） */
  initialParentId?: number | null
}

const props = withDefaults(defineProps<Props>(), {
  isEditing: false,
  folder: null,
  options: () => [],
  initialParentId: null
})

const emit = defineEmits<{
  close: []
  success: []
}>()

const { createFolder, renameFolder, error } = useFolderTree()

const folderName = ref('')
const parentFolderId = ref<number | null>(null)
const submitting = ref(false)
const nameInputRef = ref<HTMLInputElement | null>(null)

// 监听 visible 变化，初始化表单
watch(() => props.visible, async (newVal) => {
  if (newVal) {
    if (props.isEditing && props.folder) {
      folderName.value = props.folder.name
    } else {
      folderName.value = ''
      parentFolderId.value = props.fixedParentId !== undefined ? props.fixedParentId : props.initialParentId
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
      const ok = await renameFolder(props.folder.id, trimmedName)
      if (!ok) throw new Error(error.value || '重命名失败')
      QMessage.success('文件夹名称已更新')
    } else {
      const parentId = props.fixedParentId !== undefined ? props.fixedParentId : parentFolderId.value
      const ok = await createFolder(trimmedName, parentId)
      if (!ok) throw new Error(error.value || '创建失败')
      QMessage.success('文件夹创建成功')
    }
    emit('success')
    emit('close')
  } catch (e) {
    QMessage.error(e instanceof Error ? e.message : '操作失败，请稍后重试')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.form-group {
  margin-bottom: 16px;
}

.form-group:last-child {
  margin-bottom: 0;
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

.modal-btn.confirm-btn:hover {
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

<template>
  <ModalContainer
    :visible="visible"
    title="管理快速回复"
    :show-footer="false"
    @close="$emit('update:visible', false)"
  >
    <div class="quick-reply-manager">
      <p class="quick-reply-manager__tip">
        管理你的常用短语，保存后立即生效。空内容自动过滤，重复自动去重。
      </p>

      <div class="quick-reply-manager__list">
        <div
          v-for="(item, idx) in localReplies"
          :key="idx"
          class="quick-reply-manager__row"
        >
          <input
            v-model="localReplies[idx]"
            type="text"
            class="quick-reply-manager__input"
            placeholder="输入短语..."
            maxlength="50"
          />
          <button
            class="quick-reply-manager__del-btn"
            type="button"
            title="删除"
            @click="removeAt(idx)"
          >
            <i class="fas fa-times"></i>
          </button>
        </div>
        <div v-if="localReplies.length === 0" class="quick-reply-manager__empty">
          暂无短语，点击下方"添加"创建
        </div>
      </div>

      <button
        class="quick-reply-manager__add-btn"
        type="button"
        @click="addNew"
      >
        <i class="fas fa-plus"></i> 添加短语
      </button>

      <div v-if="errorMsg" class="quick-reply-manager__error">{{ errorMsg }}</div>

      <div class="quick-reply-manager__footer">
        <button
          class="quick-reply-manager__btn quick-reply-manager__btn--cancel"
          type="button"
          @click="$emit('update:visible', false)"
        >
          取消
        </button>
        <button
          class="quick-reply-manager__btn quick-reply-manager__btn--save"
          type="button"
          :disabled="saving"
          @click="handleSave"
        >
          <span v-if="saving" class="quick-reply-manager__spinner"></span>
          {{ saving ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>
  </ModalContainer>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import ModalContainer from '../shared/ModalContainer.vue'
import QMessage from '../../utils/qmessage'
import { fetchQuickReplies, saveQuickReplies } from '../../api/quickReplies'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{
  'update:visible': [value: boolean]
  'saved': []
}>()

const localReplies = ref<string[]>([])
const saving = ref(false)
const loading = ref(false)
const errorMsg = ref('')

// 弹窗打开时拉取最新数据
watch(() => props.visible, async (v) => {
  if (v) {
    errorMsg.value = ''
    loading.value = true
    try {
      localReplies.value = await fetchQuickReplies()
    } catch (e) {
      errorMsg.value = '加载失败，请重试'
    } finally {
      loading.value = false
    }
  }
}, { immediate: true })

const addNew = () => {
  localReplies.value.push('')
}

const removeAt = (idx: number) => {
  localReplies.value.splice(idx, 1)
}

const handleSave = async () => {
  errorMsg.value = ''
  saving.value = true
  try {
    await saveQuickReplies(localReplies.value)
    QMessage.success('保存成功')
    emit('saved')
    emit('update:visible', false)
  } catch (e: any) {
    errorMsg.value = e?.message || '保存失败'
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.quick-reply-manager__tip {
  margin: 0 0 12px;
  font-size: 12px;
  color: var(--text-secondary, #909399);
  line-height: 1.5;
}

.quick-reply-manager__list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 360px;
  overflow-y: auto;
  padding-right: 4px;
}

.quick-reply-manager__row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.quick-reply-manager__input {
  flex: 1;
  padding: 8px 10px;
  border: 1px solid var(--border-color, #dcdfe6);
  border-radius: 6px;
  font-size: 13px;
  color: var(--text-color, #303133);
  background: var(--input-bg, #fff);
  outline: none;
  transition: border-color 0.15s;
}

.quick-reply-manager__input:focus {
  border-color: var(--primary-color, #3385ff);
}

.quick-reply-manager__del-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  color: var(--text-secondary, #909399);
  cursor: pointer;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all 0.15s;
}

.quick-reply-manager__del-btn:hover {
  background: var(--hover-color, #f5f7fa);
  color: var(--color-danger, #f56c6c);
}

.quick-reply-manager__empty {
  padding: 16px;
  text-align: center;
  color: var(--text-secondary, #909399);
  font-size: 13px;
}

.quick-reply-manager__add-btn {
  margin-top: 12px;
  padding: 8px 12px;
  border: 1px dashed var(--border-color, #dcdfe6);
  background: transparent;
  color: var(--primary-color, #3385ff);
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  width: 100%;
  transition: all 0.15s;
}

.quick-reply-manager__add-btn:hover {
  border-color: var(--primary-color, #3385ff);
  background: var(--primary-light, rgba(51, 133, 255, 0.08));
}

.quick-reply-manager__error {
  margin-top: 12px;
  padding: 8px 10px;
  background: rgba(245, 108, 108, 0.1);
  color: var(--color-danger, #f56c6c);
  border-radius: 6px;
  font-size: 12px;
}

.quick-reply-manager__footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--border-color, #eee);
}

.quick-reply-manager__btn {
  padding: 8px 20px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  transition: all 0.15s;
}

.quick-reply-manager__btn--cancel {
  background: var(--bg-color, #fff);
  color: var(--text-secondary, #666);
  border: 1px solid var(--border-color, #ddd);
}

.quick-reply-manager__btn--cancel:hover {
  border-color: var(--primary-color, #3385ff);
  color: var(--primary-color, #3385ff);
}

.quick-reply-manager__btn--save {
  background: var(--primary-color, #3385ff);
  color: #fff;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.quick-reply-manager__btn--save:hover:not(:disabled) {
  background: var(--active-color, #66b1ff);
}

.quick-reply-manager__btn--save:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.quick-reply-manager__spinner {
  width: 12px;
  height: 12px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: quick-spin 0.8s linear infinite;
}

@keyframes quick-spin {
  to { transform: rotate(360deg); }
}
</style>

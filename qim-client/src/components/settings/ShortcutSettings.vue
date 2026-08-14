<template>
  <div class="shortcut-settings">
    <div v-if="loading" class="shortcut-loading">加载中...</div>
    <template v-else>
      <div class="shortcut-group">
        <div class="shortcut-group-header">
          <h4 class="shortcut-group-title">全局快捷键</h4>
          <button class="shortcut-reset-btn" @click="handleReset">恢复默认</button>
        </div>
        <ShortcutInput
          v-for="key in globalKeys"
          :key="key"
          :label="SHORTCUT_LABELS.global[key]"
          :modelValue="localShortcuts.global[key]"
          scope="global"
          @update:modelValue="updateItem('global', key, $event)"
        />
      </div>

      <div class="shortcut-group">
        <h4 class="shortcut-group-title">编辑器快捷键</h4>
        <ShortcutInput
          v-for="key in editorKeys"
          :key="key"
          :label="SHORTCUT_LABELS.editor[key]"
          :modelValue="localShortcuts.editor[key]"
          scope="editor"
          @update:modelValue="updateItem('editor', key, $event)"
        />
      </div>

      <div class="shortcut-group">
        <h4 class="shortcut-group-title">AI 快捷键</h4>
        <ShortcutInput
          v-for="key in aiKeys"
          :key="key"
          :label="SHORTCUT_LABELS.ai[key]"
          :modelValue="localShortcuts.ai[key]"
          scope="global"
          @update:modelValue="updateItem('ai', key, $event)"
        />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import ShortcutInput from './ShortcutInput.vue'
import {
  useShortcuts, checkConflicts, SHORTCUT_LABELS,
  DEFAULT_SHORTCUTS, type ShortcutsConfig, type ShortcutItem
} from '../../composables/useShortcuts'

const props = defineProps<{
  modelValue?: ShortcutsConfig
}>()

const emit = defineEmits<{
  'update:modelValue': [value: ShortcutsConfig]
}>()

const { loadShortcuts } = useShortcuts()

const loading = ref(true)
const localShortcuts = ref<ShortcutsConfig>(JSON.parse(JSON.stringify(DEFAULT_SHORTCUTS)))

const globalKeys = Object.keys(SHORTCUT_LABELS.global)
const editorKeys = Object.keys(SHORTCUT_LABELS.editor)
const aiKeys = Object.keys(SHORTCUT_LABELS.ai || {})

onMounted(async () => {
  try {
    localShortcuts.value = await loadShortcuts()
    emitChange()
  } catch (e) {
    console.error('加载快捷键配置失败:', e)
  } finally {
    loading.value = false
  }
})

function updateItem(scope: 'global' | 'editor' | 'ai', name: string, value: ShortcutItem) {
  localShortcuts.value[scope][name] = value
  emitChange()
}

function emitChange() {
  // 深拷贝为普通对象，避免 Vue Proxy 传给父组件后 IPC 克隆失败
  emit('update:modelValue', JSON.parse(JSON.stringify(localShortcuts.value)))
}

function handleReset() {
  localShortcuts.value = JSON.parse(JSON.stringify(DEFAULT_SHORTCUTS))
  emitChange()
  window.$QMessage?.success('已恢复默认快捷键')
}

// 暴露冲突检测，供父组件保存前调用
defineExpose({
  checkConflicts: () => checkConflicts(localShortcuts.value),
  getShortcuts: () => JSON.parse(JSON.stringify(localShortcuts.value))
})
</script>

<style scoped>
.shortcut-settings {
  padding: 10px 0;
}

.shortcut-loading {
  text-align: center;
  color: var(--text-secondary, #999);
  padding: 40px 0;
}

.shortcut-group {
  margin-bottom: 24px;
}

.shortcut-group-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.shortcut-group-title {
  margin: 0;
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--text-color);
}

.shortcut-reset-btn {
  padding: 4px 12px;
  border-radius: var(--radius-md, 6px);
  font-size: var(--font-size-xs);
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid var(--border-color);
  background: transparent;
  color: var(--text-color);
}

.shortcut-reset-btn:hover {
  background: var(--hover-bg, rgba(0, 0, 0, 0.05));
}
</style>

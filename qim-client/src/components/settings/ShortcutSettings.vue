<template>
  <div class="shortcut-settings">
    <div v-if="loading" class="shortcut-loading">加载中...</div>
    <template v-else>
      <div class="shortcut-group">
        <h4 class="shortcut-group-title">全局快捷键</h4>
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

      <div class="shortcut-actions">
        <button class="shortcut-reset-btn" @click="handleReset">恢复默认</button>
        <button class="shortcut-save-btn" @click="handleSave">保存</button>
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

const { loadShortcuts, saveShortcuts, resetShortcuts } = useShortcuts()

const loading = ref(true)
const localShortcuts = ref<ShortcutsConfig>(JSON.parse(JSON.stringify(DEFAULT_SHORTCUTS)))

const globalKeys = Object.keys(SHORTCUT_LABELS.global)
const editorKeys = Object.keys(SHORTCUT_LABELS.editor)

onMounted(async () => {
  try {
    localShortcuts.value = await loadShortcuts()
  } catch (e) {
    console.error('加载快捷键配置失败:', e)
  } finally {
    loading.value = false
  }
})

function updateItem(scope: 'global' | 'editor', name: string, value: ShortcutItem) {
  localShortcuts.value[scope][name] = value
}

async function handleSave() {
  const conflicts = checkConflicts(localShortcuts.value)
  if (conflicts.length > 0) {
    const c = conflicts[0]
    window.$QMessage?.error(`快捷键冲突：「${c.a.label}」与「${c.b.label}」使用了相同的组合`, 5000)
    return
  }
  try {
    await saveShortcuts(localShortcuts.value)
    window.$QMessage?.success('快捷键设置已保存')
  } catch (e) {
    console.error('保存快捷键配置失败:', e)
    window.$QMessage?.error('保存失败，请重试')
  }
}

async function handleReset() {
  try {
    localShortcuts.value = await resetShortcuts()
    window.$QMessage?.success('已恢复默认快捷键')
  } catch (e) {
    console.error('重置快捷键失败:', e)
    window.$QMessage?.error('重置失败，请重试')
  }
}
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

.shortcut-group-title {
  margin: 0 0 12px 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color);
}

.shortcut-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
}

.shortcut-reset-btn,
.shortcut-save-btn {
  padding: 6px 18px;
  border-radius: var(--radius-md, 6px);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.shortcut-reset-btn {
  border: 1px solid var(--border-color);
  background: transparent;
  color: var(--text-color);
}

.shortcut-reset-btn:hover {
  background: var(--hover-bg, rgba(0, 0, 0, 0.05));
}

.shortcut-save-btn {
  border: none;
  background: var(--primary-color);
  color: white;
}

.shortcut-save-btn:hover {
  opacity: 0.9;
}
</style>

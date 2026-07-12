<template>
  <div class="data-storage-settings">
    <!-- 默认保存目录（A7 迁移） -->
    <div class="settings-section-header"><h4>存储设置</h4></div>
    <div class="settings-item">
      <label>默认保存目录</label>
      <div class="settings-item-content">
        <div class="input-with-btn">
          <input
            type="text"
            :value="defaultSaveDirectory"
            class="settings-input"
            data-testid="save-directory-input"
            placeholder="选择默认保存目录"
            readonly
          />
          <button
            class="browse-btn"
            data-testid="browse-directory-btn"
            @click="$emit('browseDirectory', (path: string) => $emit('update:defaultSaveDirectory', path))"
          >
            <i class="fas fa-folder-open"></i>
            <span>浏览</span>
          </button>
        </div>
        <div class="settings-hint">设置接收文件的默认保存位置</div>
      </div>
    </div>

    <!-- 缓存管理（A2 改造） -->
    <div class="settings-section-header"><h4>缓存管理</h4></div>
    <div class="cache-overview">
      <div class="cache-total-row">
        <span class="cache-label">当前缓存占用</span>
        <span class="cache-size" data-testid="cache-total">{{ formatSize(cacheTotal) }}</span>
      </div>
    </div>
    <div
      v-for="cat in cacheCategories"
      :key="cat.id"
      class="settings-item"
    >
      <label>{{ cat.label }}</label>
      <div class="settings-item-content">
        <div class="cache-row">
          <span class="cache-size" :data-testid="`cache-category-${cat.id}`">{{ formatSize(getCategorySize(cat.id)) }}</span>
          <button
            class="action-btn"
            :data-testid="`clear-category-${cat.id}`"
            @click="clearCategory(cat.id)"
          >清理</button>
        </div>
      </div>
    </div>
    <div class="settings-item">
      <label>全部清理</label>
      <div class="settings-item-content">
        <button class="action-btn danger" data-testid="clear-all-btn" @click="clearAll">清除全部缓存</button>
        <div class="settings-hint">清除所有缓存数据（登录凭证除外）</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import QMessage from '../../utils/qmessage'

interface Props {
  defaultSaveDirectory: string
}

defineProps<Props>()

const emit = defineEmits<{
  'update:defaultSaveDirectory': [value: string]
  'browseDirectory': [callback: (path: string) => void]
  'cacheCleared': []
}>()

// 缓存分类定义
interface CacheCategory {
  id: string
  label: string
  match: (key: string) => boolean
}

const cacheCategories: CacheCategory[] = [
  {
    id: 'settings',
    label: '设置数据',
    match: (key) => ['messageSettings', 'appearanceSettings', 'theme', 'fontSize'].includes(key),
  },
]

// 受保护的 key，清理时不删除
const PROTECTED_KEYS = ['token']

const cacheTotal = ref(0)

// 计算缓存总大小（近似字节数）
const getCacheTotal = (): number => {
  let total = 0
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i)
    if (key) {
      const value = localStorage.getItem(key) || ''
      total += key.length + value.length
    }
  }
  return total
}

// 计算指定分类的缓存大小
const getCategorySize = (categoryId: string): number => {
  const category = cacheCategories.find(c => c.id === categoryId)
  if (!category) return 0
  let size = 0
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i)
    if (key && category.match(key)) {
      const value = localStorage.getItem(key) || ''
      size += key.length + value.length
    }
  }
  return size
}

// 格式化大小为人类可读字符串
const formatSize = (bytes: number): string => {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

// 清理指定分类的缓存
const clearCategory = (categoryId: string) => {
  const category = cacheCategories.find(c => c.id === categoryId)
  if (!category) return
  const keysToRemove: string[] = []
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i)
    if (key && category.match(key) && !PROTECTED_KEYS.includes(key)) {
      keysToRemove.push(key)
    }
  }
  keysToRemove.forEach(key => localStorage.removeItem(key))
  cacheTotal.value = getCacheTotal()
  QMessage.success(`已清理${category.label}`)
  emit('cacheCleared')
}

// 清理全部缓存（保留受保护数据）
const clearAll = () => {
  if (!confirm('确定要清除全部缓存吗？这将重置所有设置数据为默认值（登录凭证除外）。')) return
  const keysToRemove: string[] = []
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i)
    if (key && !PROTECTED_KEYS.includes(key)) {
      keysToRemove.push(key)
    }
  }
  keysToRemove.forEach(key => localStorage.removeItem(key))
  cacheTotal.value = getCacheTotal()
  QMessage.success('缓存已全部清除')
  emit('cacheCleared')
}

onMounted(() => {
  cacheTotal.value = getCacheTotal()
})
</script>

<style scoped>
.data-storage-settings {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.settings-section-header {
  margin-bottom: 8px;
}

.settings-section-header h4 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
}

.settings-item {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  min-height: 40px;
}

.settings-item label {
  min-width: 100px;
  flex-shrink: 0;
  color: var(--text-color);
  font-size: 14px;
  padding-top: 10px;
  font-weight: 500;
}

.settings-item-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-width: 100%;
}

.settings-input {
  padding: 10px 14px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  flex: 1;
  font-size: 14px;
  background: var(--input-bg);
  color: var(--text-color);
  transition: border-color 0.2s;
}

.input-with-btn {
  display: flex;
  gap: 12px;
}

.input-with-btn .settings-input {
  flex: 1;
}

.settings-hint {
  font-size: 12px;
  color: var(--text-secondary);
  width: 100%;
  margin-top: 6px;
  line-height: 1.4;
}

.browse-btn {
  padding: 10px 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--btn-bg);
  cursor: pointer;
  color: var(--text-color);
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 8px;
}

.browse-btn:hover {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.cache-overview {
  padding: 16px;
  background: var(--sidebar-bg);
  border-radius: 12px;
}

.cache-total-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.cache-label {
  font-size: 14px;
  color: var(--text-color);
  font-weight: 500;
}

.cache-size {
  font-size: 14px;
  color: var(--text-secondary);
}

.cache-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.action-btn {
  padding: 8px 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--btn-bg);
  cursor: pointer;
  color: var(--text-color);
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.action-btn:hover {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.action-btn.danger:hover {
  background: #e74c3c;
  border-color: #e74c3c;
}
</style>

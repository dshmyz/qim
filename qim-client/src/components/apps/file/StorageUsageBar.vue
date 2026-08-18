<template>
  <div class="storage-usage-bar" :title="`已用 ${formatFileSize(used)} / ${formatFileSize(quota)}`">
    <div class="storage-usage-bar__label">
      <i class="fas fa-database"></i>
      <span>已用 {{ formatFileSize(used) }} / {{ formatFileSize(quota) }}</span>
    </div>
    <div class="storage-usage-bar__track">
      <div
        class="storage-usage-bar__fill"
        :class="{ 'storage-usage-bar__fill--danger': percent >= 90 }"
        :style="{ width: percent + '%' }"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { fileApi } from '../../../api/file'
import { formatFileSize } from '../../../utils/fileType'

const used = ref(0)
const quota = ref(0)

const percent = computed(() => {
  if (quota.value <= 0) return 0
  return Math.min(100, Math.round((used.value / quota.value) * 100))
})

async function load() {
  try {
    const res = await fileApi.getStorageUsage()
    if (res.data.code === 0 && res.data.data) {
      used.value = res.data.data.used
      quota.value = res.data.data.quota
    }
  } catch {
    // 静默失败：容量条不阻塞文件箱使用
  }
}

onMounted(load)

defineExpose({ reload: load })
</script>

<style scoped>
.storage-usage-bar {
  padding: var(--spacing-2) var(--spacing-4);
  border-top: 1px solid var(--border-color);
  background: var(--card-bg);
  flex-shrink: 0;
}

.storage-usage-bar__label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  margin-bottom: 6px;
  white-space: nowrap;
}

.storage-usage-bar__label i {
  font-size: 11px;
}

.storage-usage-bar__track {
  height: 4px;
  border-radius: 2px;
  background: var(--border-color);
  overflow: hidden;
}

.storage-usage-bar__fill {
  height: 100%;
  border-radius: 2px;
  background: var(--primary-color);
  transition: width 0.3s ease;
}

.storage-usage-bar__fill--danger {
  background: var(--error-color);
}
</style>

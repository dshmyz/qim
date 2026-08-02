<template>
  <!-- 斜杠命令首次引导：只在未展示过时出现，关闭后 localStorage 记忆不再出现 -->
  <div v-if="visible" class="slash-hint">
    <i class="fas fa-bolt slash-hint__icon"></i>
    <span class="slash-hint__text">
      在输入框输入 <strong class="slash-hint__key">/</strong> 可快速插入文件、笔记、任务、短语
    </span>
    <button class="slash-hint__try" type="button" @click="handleTry">试一下</button>
    <button class="slash-hint__close" type="button" :aria-label="'关闭'" @click="handleClose">
      <i class="fas fa-times"></i>
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const STORAGE_KEY = 'qim:slash_hint_dismissed'

const visible = ref(false)

onMounted(() => {
  // 已关闭过的用户不再展示
  try {
    if (localStorage.getItem(STORAGE_KEY) === '1') return
  } catch {
    // localStorage 不可用时降级为每次展示（不影响功能）
  }
  visible.value = true
})

const emit = defineEmits<{
  /** 用户点击"试一下"：父组件在输入框插入 / 触发命令列表 */
  try: []
}>()

/** 关闭并记忆，刷新后不再出现 */
const handleClose = () => {
  visible.value = false
  try {
    localStorage.setItem(STORAGE_KEY, '1')
  } catch {
    // 忽略写入失败
  }
}

/** 点击"试一下"：关闭气泡并通知父组件插入 / */
const handleTry = () => {
  handleClose()
  emit('try')
}
</script>

<style scoped>
.slash-hint {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 7px 12px;
  margin-bottom: 6px;
  background: var(--primary-light, rgba(59, 130, 246, 0.08));
  border: 1px solid rgba(59, 130, 246, 0.2);
  border-radius: 8px;
  font-size: 12px;
  color: var(--text-color, #303133);
  animation: slash-hint-in 0.25s ease;
}

@keyframes slash-hint-in {
  from { opacity: 0; transform: translateY(-4px); }
  to { opacity: 1; transform: translateY(0); }
}

.slash-hint__icon {
  color: var(--primary-color, #3b82f6);
  flex-shrink: 0;
}

.slash-hint__text {
  flex: 1;
  color: var(--text-secondary, #606266);
}

.slash-hint__key {
  display: inline-block;
  padding: 1px 6px;
  margin: 0 2px;
  background: var(--card-bg, #fff);
  border: 1px solid var(--border-color, #dcdfe6);
  border-radius: 4px;
  font-family: monospace;
  font-weight: 600;
  color: var(--text-color, #303133);
  box-shadow: 0 1px 2px rgba(0, 0,0,0.04);
}

.slash-hint__try {
  flex-shrink: 0;
  padding: 3px 10px;
  background: var(--primary-color, #3b82f6);
  color: #fff;
  border: none;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  transition: background 0.15s;
}

.slash-hint__try:hover {
  background: var(--primary-hover, #2563eb);
}

.slash-hint__close {
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  color: var(--text-secondary, #909399);
  cursor: pointer;
  border-radius: 4px;
  font-size: 12px;
  transition: background 0.15s;
}

.slash-hint__close:hover {
  background: var(--hover-color, rgba(0,0,0,0.06));
  color: var(--text-color, #303133);
}
</style>

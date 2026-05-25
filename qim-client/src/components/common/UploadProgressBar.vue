<template>
  <Teleport to="body">
    <Transition name="upload-progress-slide">
      <div v-if="visible && hasTasks" class="upload-progress-bar">
        <!-- 总进度区域 -->
        <div class="progress-header" @click="toggleExpand">
          <div class="progress-info">
            <div class="progress-icon">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" stroke-linecap="round" stroke-linejoin="round"/>
                <polyline points="17,8 12,3 7,8" stroke-linecap="round" stroke-linejoin="round"/>
                <line x1="12" y1="3" x2="12" y2="15" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </div>
            <div class="progress-text">
              <span class="progress-title">
                {{ uploadingText }}
              </span>
              <span class="progress-percentage">{{ totalProgress }}%</span>
            </div>
          </div>

          <div class="progress-bar-container">
            <div class="progress-bar" :style="{ width: `${totalProgress}%` }"></div>
          </div>

          <div class="progress-actions">
            <button
              v-if="hasCompletedTasks"
              class="action-btn clear-btn"
              @click.stop="handleClearCompleted"
              title="清空已完成"
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="3,6 5,6 21,6" stroke-linecap="round" stroke-linejoin="round"/>
                <path d="M19,6v14a2,2,0,0,1-2,2H7a2,2,0,0,1-2-2V6m3,0V4a2,2,0,0,1,2-2h4a2,2,0,0,1,2,2v2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </button>
            <button class="action-btn toggle-btn" @click.stop="toggleExpand">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                :class="{ 'rotated': isExpanded }"
              >
                <polyline points="6,9 12,15 18,9" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </button>
          </div>
        </div>

        <!-- 任务列表 -->
        <Transition name="task-list-expand">
          <div v-show="isExpanded" class="task-list">
            <div
              v-for="task in sortedTasks"
              :key="task.uploadId"
              class="task-item"
              :class="`task-item--${task.status}`"
            >
              <div class="task-info">
                <div class="task-file-icon">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" stroke-linecap="round" stroke-linejoin="round"/>
                    <polyline points="14,2 14,8 20,8" stroke-linecap="round" stroke-linejoin="round"/>
                  </svg>
                </div>
                <div class="task-details">
                  <div class="task-name" :title="task.file.name">
                    {{ task.file.name }}
                  </div>
                  <div class="task-meta">
                    <span class="task-size">
                      {{ formatSize(task.uploadedSize) }} / {{ formatSize(task.totalSize) }}
                    </span>
                    <span class="task-status" :class="`status--${task.status}`">
                      {{ statusText(task.status) }}
                    </span>
                  </div>
                  <div v-if="task.status === 'uploading' || task.status === 'pending'" class="task-progress-bar">
                    <div class="task-progress" :style="{ width: `${task.progress}%` }"></div>
                  </div>
                  <div v-if="task.error" class="task-error">
                    {{ task.error }}
                  </div>
                </div>
              </div>

              <div class="task-actions">
                <button
                  v-if="task.status === 'uploading' || task.status === 'pending'"
                  class="task-action-btn cancel-btn"
                  @click="handleCancel(task.uploadId)"
                  title="取消上传"
                >
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="18" y1="6" x2="6" y2="18" stroke-linecap="round" stroke-linejoin="round"/>
                    <line x1="6" y1="6" x2="18" y2="18" stroke-linecap="round" stroke-linejoin="round"/>
                  </svg>
                </button>
                <button
                  v-if="task.status === 'failed'"
                  class="task-action-btn retry-btn"
                  @click="handleRetry(task.uploadId)"
                  title="重试"
                >
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="23,4 23,10 17,10" stroke-linecap="round" stroke-linejoin="round"/>
                    <path d="M20.49,15a9,9,0,1,1-2.12-9.36L23,10" stroke-linecap="round" stroke-linejoin="round"/>
                  </svg>
                </button>
                <button
                  v-if="task.status === 'completed' || task.status === 'failed' || task.status === 'cancelled'"
                  class="task-action-btn delete-btn"
                  @click="handleRemove(task.uploadId)"
                  title="删除"
                >
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3,6 5,6 21,6" stroke-linecap="round" stroke-linejoin="round"/>
                    <path d="M19,6v14a2,2,0,0,1-2,2H7a2,2,0,0,1-2-2V6m3,0V4a2,2,0,0,1,2-2h4a2,2,0,0,1,2,2v2" stroke-linecap="round" stroke-linejoin="round"/>
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useUploadStore, type UploadTask } from '../../stores/upload'
import { uploadFile, cancelUpload } from '../../composables/useFileUpload'

interface Props {
  visible: boolean
}

defineProps<Props>()

const uploadStore = useUploadStore()

// 计算属性
const hasTasks = computed(() => uploadStore.tasks.size > 0)

const hasCompletedTasks = computed(() => uploadStore.completedTasks.length > 0)

const totalProgress = computed(() => uploadStore.totalProgress)

const isExpanded = computed(() => uploadStore.isExpanded)

const uploadingText = computed(() => {
  const activeCount = uploadStore.activeTasks.length
  const failedCount = uploadStore.failedTasks.length
  const completedCount = uploadStore.completedTasks.length

  if (activeCount > 0) {
    return `正在上传 ${activeCount} 个文件`
  } else if (failedCount > 0) {
    return `${failedCount} 个文件上传失败`
  } else if (completedCount > 0) {
    return `已完成 ${completedCount} 个文件上传`
  }
  return '上传任务'
})

const sortedTasks = computed(() => {
  const tasks = Array.from(uploadStore.tasks.values())
  // 按状态排序：uploading > pending > failed > completed > cancelled
  const statusOrder: Record<string, number> = {
    uploading: 0,
    pending: 1,
    failed: 2,
    completed: 3,
    cancelled: 4
  }
  return tasks.sort((a, b) => statusOrder[a.status] - statusOrder[b.status])
})

// 方法
function toggleExpand() {
  uploadStore.toggleExpanded()
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

function statusText(status: UploadTask['status']): string {
  const statusMap: Record<string, string> = {
    pending: '等待中',
    uploading: '上传中',
    completed: '已完成',
    failed: '失败',
    cancelled: '已取消'
  }
  return statusMap[status] || status
}

async function handleCancel(uploadId: string) {
  try {
    await cancelUpload(uploadId)
    uploadStore.cancelTask(uploadId)
  } catch (error) {
    console.error('取消上传失败:', error)
  }
}

async function handleRetry(uploadId: string) {
  const task = uploadStore.tasks.get(uploadId)
  if (!task) return

  // 移除旧任务
  uploadStore.removeTask(uploadId)

  // 重新上传
  try {
    await uploadFile(task.file, task.folderId ?? undefined)
  } catch (error) {
    console.error('重试上传失败:', error)
  }
}

function handleRemove(uploadId: string) {
  uploadStore.removeTask(uploadId)
}

function handleClearCompleted() {
  uploadStore.clearCompleted()
}
</script>

<style scoped>
.upload-progress-bar {
  position: fixed;
  bottom: 24px;
  right: 24px;
  width: 380px;
  background: var(--card-bg, #ffffff);
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 16px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12), 0 2px 8px rgba(0, 0, 0, 0.06);
  z-index: var(--z-sticky, 1020);
  max-height: 480px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  backdrop-filter: blur(12px);
}

/* 进度头部 */
.progress-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  cursor: pointer;
  user-select: none;
  background: linear-gradient(135deg, var(--card-bg, #fff) 0%, var(--color-primary-50, #f0f5ff) 100%);
}

.progress-header:hover {
  background: linear-gradient(135deg, var(--hover-color, #f5f6f8) 0%, var(--color-primary-50, #e8f0fe) 100%);
}

.progress-info {
  display: flex;
  align-items: center;
  gap: var(--spacing-2, 8px);
  flex-shrink: 0;
}

.progress-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--primary-color, #3385ff), #6366f1);
  border-radius: 12px;
  color: #fff;
  flex-shrink: 0;
}

.progress-icon svg {
  width: 18px;
  height: 18px;
}

.progress-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.progress-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-color, #1a1a2e);
}

.progress-percentage {
  font-size: 20px;
  font-weight: 700;
  color: var(--primary-color, #3385ff);
  line-height: 1;
}

/* 进度条 */
.progress-bar-container {
  flex: 1;
  height: 6px;
  background: var(--color-gray-100, #e5e7eb);
  border-radius: 3px;
  overflow: hidden;
}

.progress-bar {
  height: 100%;
  background: linear-gradient(90deg, #3385ff, #6366f1, #3385ff);
  background-size: 200% 100%;
  border-radius: 3px;
  transition: width 0.3s ease;
  animation: progress-shimmer 2s linear infinite;
}

@keyframes progress-shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: 0 0; }
}

/* 操作按钮 */
.progress-actions {
  display: flex;
  align-items: center;
  gap: var(--spacing-2, 8px);
  flex-shrink: 0;
}

.action-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-secondary, #9ca3af);
  cursor: pointer;
  border-radius: 8px;
  transition: all 0.15s ease;
}

.action-btn:hover {
  background: var(--hover-color, #f3f4f6);
  border-color: var(--border-color, #e5e7eb);
  color: var(--text-color, #1a1a2e);
}

.action-btn svg {
  width: 16px;
  height: 16px;
}

.toggle-btn svg {
  transition: transform var(--transition-base, 200ms ease);
}

.toggle-btn svg.rotated {
  transform: rotate(180deg);
}

.clear-btn:hover {
  color: var(--error-color, #f34040);
}

/* 任务列表 */
.task-list {
  max-height: 320px;
  overflow-y: auto;
  border-top: 1px solid var(--border-color, #e5e7eb);
  background: var(--card-bg, #fff);
}

.task-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border-color, #f0f0f0);
  transition: background 0.15s ease;
}

.task-item:last-child {
  border-bottom: none;
}

.task-item:hover {
  background: var(--hover-color, #f8f9fb);
}

.task-item--completed {
  background: linear-gradient(135deg, #f6fff9, #f0fdf4);
}

.task-item--failed {
  background: linear-gradient(135deg, #fff6f6, #fef2f2);
}

.task-item--cancelled {
  opacity: 0.5;
}

/* 任务信息 */
.task-info {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-3, 12px);
  flex: 1;
  min-width: 0;
}

.task-file-icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-gray-100, #f8f8f8);
  border-radius: var(--radius-md, 8px);
  color: var(--text-secondary, #a0a0a0);
  flex-shrink: 0;
}

.task-file-icon svg {
  width: 18px;
  height: 18px;
}

.task-details {
  flex: 1;
  min-width: 0;
}

.task-name {
  font-size: var(--font-size-sm, 14px);
  font-weight: var(--font-weight-medium, 500);
  color: var(--text-color, #404040);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-meta {
  display: flex;
  align-items: center;
  gap: var(--spacing-2, 8px);
  margin-top: 4px;
  font-size: var(--font-size-xs, 12px);
}

.task-size {
  color: var(--text-secondary, #a0a0a0);
}

.task-status {
  padding: 2px 6px;
  border-radius: var(--radius-sm, 4px);
  font-size: 11px;
  font-weight: var(--font-weight-medium, 500);
}

.status--pending {
  background: var(--color-gray-100, #f8f8f8);
  color: var(--text-secondary, #a0a0a0);
}

.status--uploading {
  background: var(--color-primary-50, #f8faff);
  color: var(--primary-color, #3385ff);
}

.status--completed {
  background: var(--color-success-50, #f6fff9);
  color: var(--success-color, #26b361);
}

.status--failed {
  background: var(--color-error-50, #fff6f6);
  color: var(--error-color, #f34040);
}

.status--cancelled {
  background: var(--color-gray-100, #f8f8f8);
  color: var(--text-secondary, #a0a0a0);
}

/* 任务进度条 */
.task-progress-bar {
  height: 4px;
  background: var(--color-gray-100, #e5e7eb);
  border-radius: 2px;
  overflow: hidden;
  margin-top: 6px;
}

.task-progress {
  height: 100%;
  background: linear-gradient(90deg, var(--primary-color, #3385ff), #6366f1);
  border-radius: 2px;
  transition: width 0.3s ease;
}

/* 任务错误信息 */
.task-error {
  margin-top: 4px;
  font-size: var(--font-size-xs, 12px);
  color: var(--error-color, #f34040);
}

/* 任务操作按钮 */
.task-actions {
  display: flex;
  align-items: center;
  gap: var(--spacing-1, 4px);
  flex-shrink: 0;
}

.task-action-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-secondary, #9ca3af);
  cursor: pointer;
  border-radius: 8px;
  transition: all 0.15s ease;
}

.task-action-btn:hover {
  background: var(--hover-color, #f3f4f6);
  border-color: var(--border-color, #e5e7eb);
  color: var(--text-color, #1a1a2e);
}

.task-action-btn svg {
  width: 15px;
  height: 15px;
}

.cancel-btn:hover {
  color: var(--error-color, #f34040);
}

.retry-btn:hover {
  color: var(--primary-color, #3385ff);
}

.delete-btn:hover {
  color: var(--error-color, #f34040);
}

/* 动画 */
.upload-progress-slide-enter-active,
.upload-progress-slide-leave-active {
  transition: all var(--transition-slow, 300ms ease);
}

.upload-progress-slide-enter-from,
.upload-progress-slide-leave-to {
  transform: translateY(-100%);
  opacity: 0;
}

.task-list-expand-enter-active,
.task-list-expand-leave-active {
  transition: all var(--transition-base, 200ms ease);
}

.task-list-expand-enter-from,
.task-list-expand-leave-to {
  max-height: 0;
  opacity: 0;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .progress-header {
    padding: var(--spacing-2, 8px) var(--spacing-3, 12px);
    gap: var(--spacing-2, 8px);
  }

  .progress-title {
    font-size: var(--font-size-xs, 12px);
  }

  .progress-percentage {
    font-size: 11px;
  }

  .task-item {
    padding: var(--spacing-2, 8px) var(--spacing-3, 12px);
    gap: var(--spacing-2, 8px);
  }

  .task-file-icon {
    width: 28px;
    height: 28px;
  }

  .task-file-icon svg {
    width: 16px;
    height: 16px;
  }

  .task-name {
    font-size: var(--font-size-xs, 12px);
  }

  .task-meta {
    font-size: 11px;
  }
}

@media (max-width: 480px) {
  .progress-info {
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
  }

  .progress-text {
    flex-direction: row;
    align-items: center;
    gap: var(--spacing-2, 8px);
  }

  .task-info {
    flex-direction: column;
    align-items: stretch;
    gap: var(--spacing-2, 8px);
  }

  .task-file-icon {
    display: none;
  }

  .task-actions {
    align-self: flex-start;
  }
}
</style>

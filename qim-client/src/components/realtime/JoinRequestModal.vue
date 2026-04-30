<template>
  <Teleport to="body">
    <Transition name="modal-fade">
      <div v-if="visible" class="join-request-modal-mask" @click.self="handleClose">
        <div class="join-request-modal">
          <div class="modal-header">
            <h3 class="modal-title">加入请求</h3>
            <button class="close-btn" @click="handleClose">
              <i class="fas fa-times"></i>
            </button>
          </div>

          <div class="modal-body">
            <div v-if="requests.length === 0" class="empty-state">
              <i class="fas fa-inbox"></i>
              <span>暂无加入请求</span>
            </div>

            <div v-else class="request-list">
              <div
                v-for="request in requests"
                :key="`${request.session_id}-${request.user_id}`"
                class="request-item"
              >
                <div class="requester-info">
                  <img
                    v-if="request.user?.avatar"
                    :src="request.user.avatar"
                    :alt="request.user.nickname"
                    class="requester-avatar"
                  />
                  <div v-else class="requester-avatar-placeholder">
                    <i class="fas fa-user"></i>
                  </div>
                  <div class="requester-detail">
                    <div class="requester-name">
                      {{ request.user?.nickname || `用户${request.user_id}` }}
                    </div>
                    <div class="request-time">
                      {{ formatTime(request.timestamp) }}
                    </div>
                  </div>
                </div>

                <div class="request-actions">
                  <button
                    class="approve-btn"
                    @click="handleApprove(request.session_id, request.user_id)"
                  >
                    <i class="fas fa-check"></i>
                    <span>同意</span>
                  </button>
                  <button
                    class="reject-btn"
                    @click="handleReject(request.session_id, request.user_id)"
                  >
                    <i class="fas fa-times"></i>
                    <span>拒绝</span>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import type { JoinRequest } from '../../types/realtime'

interface Props {
  visible: boolean
  requests: JoinRequest[]
}

defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  approve: [sessionId: string, userId: number]
  reject: [sessionId: string, userId: number]
}>()

const handleClose = () => {
  emit('update:visible', false)
}

const handleApprove = (sessionId: string, userId: number) => {
  emit('approve', sessionId, userId)
}

const handleReject = (sessionId: string, userId: number) => {
  emit('reject', sessionId, userId)
}

const formatTime = (timestamp: number): string => {
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  if (diff < 60000) {
    return '刚刚'
  } else if (diff < 3600000) {
    return `${Math.floor(diff / 60000)}分钟前`
  } else if (diff < 86400000) {
    return `${Math.floor(diff / 3600000)}小时前`
  } else {
    return date.toLocaleDateString()
  }
}
</script>

<style scoped>
.join-request-modal-mask {
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
}

.join-request-modal {
  background: var(--panel-bg, #fff);
  border-radius: 16px;
  width: 400px;
  max-width: 90vw;
  max-height: 80vh;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
}

.modal-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 16px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.modal-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #fff;
}

.close-btn {
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.2);
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.close-btn:hover {
  background: rgba(255, 255, 255, 0.3);
}

.modal-body {
  padding: 16px;
  max-height: 400px;
  overflow-y: auto;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: var(--text-secondary, #6b7280);
}

.empty-state i {
  font-size: 32px;
  margin-bottom: 12px;
  opacity: 0.5;
}

.empty-state span {
  font-size: 14px;
}

.request-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.request-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
  background: var(--color-gray-50, #f9fafb);
  border-radius: 12px;
  transition: all 0.2s;
}

.request-item:hover {
  background: var(--color-gray-100, #f3f4f6);
}

.requester-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.requester-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
}

.requester-avatar-placeholder {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 16px;
}

.requester-detail {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.requester-name {
  font-weight: 500;
  font-size: 14px;
  color: var(--text-color, #1f2937);
}

.request-time {
  font-size: 12px;
  color: var(--text-secondary, #6b7280);
}

.request-actions {
  display: flex;
  gap: 8px;
}

.approve-btn,
.reject-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 12px;
  border: none;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.approve-btn {
  background: linear-gradient(135deg, #22c55e 0%, #16a34a 100%);
  color: #fff;
}

.approve-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(34, 197, 94, 0.3);
}

.reject-btn {
  background: var(--color-gray-200, #e5e7eb);
  color: var(--text-secondary, #6b7280);
}

.reject-btn:hover {
  background: var(--color-gray-300, #d1d5db);
}

.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: all 0.3s ease;
}

.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}

.modal-fade-enter-from .join-request-modal,
.modal-fade-leave-to .join-request-modal {
  transform: scale(0.95);
}
</style>

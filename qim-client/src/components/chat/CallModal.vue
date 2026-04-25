<template>
  <div v-if="visible" class="call-modal" @click="$emit('end-call')">
    <div class="call-modal-content" @click.stop>
      <div class="call-modal-header">
        <h3>{{ callType === 'voice' ? '语音通话' : '视频通话' }}</h3>
      </div>
      <div class="call-modal-body">
        <div class="call-info">
          <div class="call-avatar">
            <img :src="avatar" :alt="name || '未知'" />
          </div>
          <div class="call-name">{{ name || '未知' }}</div>
          <div class="call-status">
            <span v-if="status === 'ringing'" class="status-ringing">正在呼叫...</span>
            <span v-else-if="status === 'answered'" class="status-answered">通话中</span>
            <span v-else-if="status === 'ended'" class="status-ended">通话结束</span>
          </div>
        </div>

        <!-- 视频通话区域 -->
        <div v-if="callType === 'video' && status === 'answered'" class="video-container">
          <div class="local-video">
            <div class="video-placeholder">
              <i class="fas fa-user"></i>
              <span>您</span>
            </div>
          </div>
          <div class="remote-video">
            <div class="video-placeholder">
              <i class="fas fa-user"></i>
              <span>{{ name || '对方' }}</span>
            </div>
          </div>
        </div>
      </div>
      <div class="call-modal-footer">
        <button v-if="status === 'ringing'" class="call-btn reject-btn" @click.stop="$emit('reject-call')">
          <i class="fas fa-phone-slash"></i>
          <span>拒绝</span>
        </button>
        <button v-if="status === 'ringing'" class="call-btn answer-btn" @click.stop="$emit('answer-call')">
          <i class="fas fa-phone"></i>
          <span>接听</span>
        </button>
        <button v-else class="call-btn end-btn" @click.stop="$emit('end-call')">
          <i class="fas fa-phone-slash"></i>
          <span>结束通话</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  visible: boolean
  callType: 'voice' | 'video' | ''
  status: 'ringing' | 'answered' | 'ended' | ''
  avatar: string
  name: string
}

defineProps<Props>()
defineEmits<{
  (e: 'reject-call'): void
  (e: 'answer-call'): void
  (e: 'end-call'): void
}>()
</script>

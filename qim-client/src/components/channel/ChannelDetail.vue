<template>
  <div class="channel-detail-content">
    <div class="right-content-header">
      <div class="header-left-group">
        <button class="toggle-sidebar-btn" @click="$emit('toggleSidebar')">
          <i class="fas fa-compress"></i>
        </button>
        <h2>{{ channel.name }}</h2>
      </div>
    </div>
    <div class="channel-detail-info">
      <div class="channel-header-info">
        <img :src="getAvatarUrl(channel.avatar, channel.name, serverUrl)" :alt="channel.name" class="channel-header-avatar" />
        <div class="channel-header-text">
          <p class="channel-description">{{ channel.description }}</p>
          <div class="channel-meta">
            <span>创建者: {{ channel.creator?.name || '未知' }}</span>
            <span v-if="channel.created_at">创建时间: {{ formatTime(channel.created_at) }}</span>
          </div>
        </div>
      </div>
      <div class="channel-header-actions">
        <button 
          v-if="channel.is_subscribed" 
          class="btn btn-secondary subscribed" 
          @click="$emit('unsubscribe', channel)"
        >
          <i class="fas fa-check"></i> 已订阅
        </button>
        <button 
          v-else 
          class="btn btn-primary" 
          @click="$emit('subscribe', channel)"
        >
          <i class="fas fa-plus"></i> 订阅
        </button>
      </div>
    </div>
    
    <div class="channel-messages">
      <h3>最新消息</h3>
      <div v-if="!channel.messages || channel.messages.length === 0" class="empty-messages">
        <i class="fas fa-comment-alt"></i>
        <p>暂无消息</p>
      </div>
      <div v-else class="message-list">
        <div 
          v-for="message in channel.messages" 
          :key="message.id" 
          class="message-item"
        >
          <img :src="getAvatarUrl(message.sender?.avatar, message.sender?.name || 'user', serverUrl)" :alt="message.sender?.name" class="message-avatar" />
          <div class="message-content">
            <div class="message-header">
              <span class="message-sender">{{ message.sender?.name || '未知' }}</span>
              <span class="message-time">{{ formatTime(message.created_at) }}</span>
            </div>
            <div class="message-text">{{ message.content }}</div>
          </div>
        </div>
      </div>
    </div>
    
    <div v-if="isCreator" class="message-input-area">
      <textarea 
        v-model="localMessage" 
        placeholder="输入消息..." 
        rows="2"
        class="message-textarea"
      ></textarea>
      <button class="btn btn-primary send-btn" @click="$emit('sendMessage', channel, localMessage)" :disabled="!localMessage.trim()">发送</button>
    </div>
    <div v-else-if="channel.is_subscribed" class="message-readonly-hint">
      <i class="fas fa-bullhorn"></i>
      <span>频道为广播模式，仅创建者可发布消息</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { getAvatarUrl } from '../../utils/avatar'
import { API_BASE_URL } from '../../config'

const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

interface ChannelMessage {
  id: string | number
  sender?: { name?: string; avatar?: string }
  content: string
  created_at: string
}

interface Channel {
  id: string | number
  name: string
  avatar?: string
  description: string
  creator?: { name?: string }
  created_at?: string
  is_subscribed?: boolean
  messages?: ChannelMessage[]
}

interface Props {
  channel: Channel
  isCreator: boolean
  formatTime: (date: string) => string
  initialMessage?: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'toggleSidebar': []
  'subscribe': [channel: Channel]
  'unsubscribe': [channel: Channel]
  'sendMessage': [channel: Channel, message: string]
}>()

const localMessage = ref(props.initialMessage || '')
</script>

<style scoped>
.channel-detail-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.right-content-header {
  padding: 16px 20px;
  background: var(--right-content-header-bg, #fff);
  height: 72px;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.right-content-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 500;
  color: var(--text-color, #333);
}

.header-left-group {
  display: flex;
  align-items: center;
  gap: 12px;
}

.toggle-sidebar-btn {
  background: none;
  border: none;
  cursor: pointer;
  padding: 8px;
  color: var(--text-color, #333);
}

.channel-detail-info {
  padding: 20px;
  border-bottom: 1px solid var(--border-color, #eee);
}

.channel-header-info {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}

.channel-header-avatar {
  width: 64px;
  height: 64px;
  border-radius: 50%;
}

.channel-header-text {
  flex: 1;
}

.channel-description {
  margin: 0 0 8px 0;
  color: var(--text-color, #333);
}

.channel-meta {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: var(--text-secondary, #999);
}

.channel-header-actions {
  display: flex;
  gap: 12px;
}

.btn {
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
}

.btn-primary {
  background: var(--primary-color, #409eff);
  color: white;
}

.btn-secondary.subscribed {
  background: var(--success-color, #67c23a);
  color: white;
}

.channel-messages {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
}

.channel-messages h3 {
  margin: 0 0 16px 0;
  font-size: 16px;
  color: var(--text-color, #333);
}

.empty-messages {
  text-align: center;
  padding: 40px 0;
  color: var(--text-secondary, #999);
}

.empty-messages i {
  font-size: 48px;
  margin-bottom: 12px;
}

.message-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.message-item {
  display: flex;
  gap: 12px;
}

.message-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
}

.message-content {
  flex: 1;
}

.message-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 4px;
}

.message-sender {
  font-weight: 500;
  color: var(--text-color, #333);
}

.message-time {
  font-size: 12px;
  color: var(--text-secondary, #999);
}

.message-text {
  color: var(--text-color, #333);
  line-height: 1.5;
}

.message-input-area {
  padding: 16px 20px;
  border-top: 1px solid var(--border-color, #eee);
  display: flex;
  gap: 12px;
  align-items: flex-end;
}

.message-textarea {
  flex: 1;
  padding: 12px;
  border: 1px solid var(--border-color, #ddd);
  border-radius: 4px;
  resize: none;
  font-family: inherit;
  font-size: 14px;
}

.send-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.message-readonly-hint {
  padding: 12px 20px;
  border-top: 1px solid var(--border-color, #eee);
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-secondary, #999);
  font-size: 13px;
  background: var(--hover-color, #f5f5f5);
}

.message-readonly-hint i {
  font-size: 14px;
}
</style>

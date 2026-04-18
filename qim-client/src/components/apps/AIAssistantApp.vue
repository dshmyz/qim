<template>
  <div class="ai-assistant-app">
    <div class="ai-assistant-header">
      <div class="header-left">
        <button class="back-btn" @click="$emit('back')">
          <i class="fas fa-arrow-left"></i>
        </button>
        <h2>AI 助手</h2>
      </div>
    </div>
    <div class="ai-assistant-content">
      <div v-if="!showBotChat" class="bot-selection">
        <h3>选择机器人</h3>
        <div class="bot-list">
          <div 
            v-for="bot in bots" 
            :key="bot.id" 
            class="bot-item"
            @click="selectBot(bot.id)"
          >
            <div class="bot-avatar">
              <img :src="bot.avatar" :alt="bot.name" v-if="bot.avatar">
              <i class="fas fa-robot" v-else></i>
            </div>
            <div class="bot-info">
              <h4>{{ bot.name }}</h4>
              <p>{{ bot.description }}</p>
              <span class="bot-type" :class="bot.type">{{ bot.type === 'ai' ? 'AI 机器人' : '系统机器人' }}</span>
            </div>
          </div>
          <div v-if="bots.length === 0" class="empty-bots">
            <i class="fas fa-robot"></i>
            <p>暂无可用的机器人</p>
          </div>
        </div>
      </div>
      <div v-else class="bot-chat">
        <div class="chat-header">
          <button class="back-button" @click="exitBotChat">
            <i class="fas fa-arrow-left"></i>
          </button>
          <h3>{{ currentBot?.name || 'AI 助手' }}</h3>
        </div>
        <div class="chat-messages" ref="chatMessagesRef">
          <div 
            v-for="message in botMessages" 
            :key="message.id" 
            :class="['message', message.sender === 'user' ? 'user-message' : 'bot-message']"
          >
            <div class="message-content">{{ message.content }}</div>
            <div class="message-time">{{ formatTime(message.timestamp) }}</div>
          </div>
        </div>
        <div class="chat-input">
          <input 
            v-model="botMessageInput" 
            type="text" 
            placeholder="输入消息..." 
            @keyup.enter="sendBotMessage"
          >
          <button @click="sendBotMessage" class="send-button">
            <i class="fas fa-paper-plane"></i>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import axios from 'axios'
import { API_BASE_URL } from '../../config'

// 服务器URL
const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

// 定义事件
const emit = defineEmits(['back'])

// AI 助手相关状态
const selectedBotId = ref('')
const bots = ref<any[]>([])
const botMessages = ref<any[]>([])
const botMessageInput = ref('')
const showBotChat = ref(false)
const chatMessagesRef = ref<HTMLDivElement | null>(null)

// 获取当前选中的机器人
const currentBot = computed(() => {
  return bots.value.find(bot => bot.id === selectedBotId.value)
})

// 获取token
const getToken = () => {
  return localStorage.getItem('token')
}

// 加载机器人列表
const loadBots = async () => {
  try {
    const token = getToken()
    const response = await axios.get(`${serverUrl.value}/api/v1/bots`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    if (response.data.code === 0) {
      bots.value = response.data.data
    }
  } catch (error) {
    console.error('加载机器人列表失败:', error)
  }
}

// 选择机器人
const selectBot = (botId: number) => {
  selectedBotId.value = botId.toString()
  botMessages.value = []
  showBotChat.value = true
}

// 退出机器人聊天
const exitBotChat = () => {
  showBotChat.value = false
  selectedBotId.value = ''
  botMessages.value = []
}

// 发送消息给机器人
const sendBotMessage = async () => {
  if (!botMessageInput.value.trim() || !selectedBotId.value) return
  
  const message = botMessageInput.value.trim()
  botMessageInput.value = ''
  
  // 添加用户消息
  const userMessage = {
    id: Date.now(),
    content: message,
    sender: 'user',
    timestamp: new Date()
  }
  botMessages.value.push(userMessage)
  
  // 滚动到底部
  await scrollToBottom()
  
  try {
    const token = getToken()
    // 创建单一会话
    const convResponse = await axios.post(`${serverUrl.value}/api/v1/conversations/single`, {
      recipient_id: 0, // 0 表示机器人
      bot_id: selectedBotId.value
    }, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    
    if (convResponse.data.code === 0) {
      const convId = convResponse.data.data.id
      // 发送消息
      await axios.post(`${serverUrl.value}/api/v1/conversations/${convId}/messages`, {
        content: message,
        type: 'text'
      }, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })
      
      // 模拟机器人回复
      setTimeout(() => {
        const replies = [
          "这是一个有趣的问题！让我想想...",
          "根据我的理解，你是在问关于...",
          "好的，我来帮你解答这个问题。",
          "这个问题很有意思，我认为...",
          "让我分析一下这个问题..."
        ]
        const botReply = {
          id: Date.now() + 1,
          content: replies[Math.floor(Math.random() * replies.length)] + "\n\n你刚才说：" + message,
          sender: 'bot',
          timestamp: new Date()
        }
        botMessages.value.push(botReply)
        scrollToBottom()
      }, 1000)
    }
  } catch (error) {
    console.error('发送消息失败:', error)
  }
}

// 滚动到底部
const scrollToBottom = async () => {
  await nextTick()
  if (chatMessagesRef.value) {
    chatMessagesRef.value.scrollTop = chatMessagesRef.value.scrollHeight
  }
}

// 格式化时间
const formatTime = (date: Date) => {
  return date.toLocaleTimeString()
}

// 组件挂载时加载机器人列表
onMounted(async () => {
  await loadBots()
})
</script>

<style scoped>
.ai-assistant-app {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--content-bg);
  overflow: hidden;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.ai-assistant-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  background-color: var(--card-bg);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
  height: 72px;
  box-sizing: border-box;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.back-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: var(--hover-color);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  transition: background 0.2s;
  color: var(--primary-color);
}

.back-btn:hover {
  background: var(--primary-light);
}

.ai-assistant-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
}

.ai-assistant-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--content-bg);
}

/* 机器人选择 */
.bot-selection {
  padding: 20px;
  flex: 1;
  overflow-y: auto;
}

.bot-selection h3 {
  margin: 0 0 20px 0;
  font-size: 16px;
  color: var(--text-primary);
}

.bot-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
}

.bot-item {
  background: var(--card-bg);
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
  display: flex;
  align-items: center;
  border: 1px solid var(--border-color);
}

.bot-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
}

.bot-avatar {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  background: var(--bg-color);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 15px;
  overflow: hidden;
}

.bot-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.bot-avatar i {
  font-size: 32px;
  color: var(--primary-color);
}

.bot-info {
  flex: 1;
}

.bot-info h4 {
  margin: 0 0 5px 0;
  font-size: 16px;
  color: var(--text-primary);
}

.bot-info p {
  margin: 0 0 10px 0;
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.4;
}

.bot-type {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.bot-type.ai {
  background: #E3F2FD;
  color: #1976D2;
}

.bot-type.system {
  background: #E8F5E8;
  color: #388E3C;
}

.empty-bots {
  text-align: center;
  padding: 60px 20px;
  color: var(--text-secondary);
}

.empty-bots i {
  font-size: 48px;
  margin-bottom: 10px;
  color: var(--text-tertiary);
}

/* 机器人聊天 */
.bot-chat {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--card-bg);
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.chat-header {
  padding: 15px 20px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  background: var(--bg-color);
}

.chat-header h3 {
  margin: 0 0 0 10px;
  font-size: 16px;
  color: var(--text-primary);
}

.back-button {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  color: var(--text-primary);
  cursor: pointer;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s;
}

.back-button:hover {
  background: var(--hover-color);
}

.chat-messages {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 15px;
  background: var(--content-bg);
}

.message {
  max-width: 80%;
  padding: 10px 15px;
  border-radius: 18px;
  position: relative;
}

.user-message {
  align-self: flex-end;
  background: var(--primary-color);
  color: white;
  border-bottom-right-radius: 4px;
}

.bot-message {
  align-self: flex-start;
  background: var(--list-bg);
  color: var(--text-primary);
  border-bottom-left-radius: 4px;
}

.message-content {
  font-size: 14px;
  line-height: 1.4;
}

.message-time {
  font-size: 12px;
  opacity: 0.7;
  margin-top: 5px;
  text-align: right;
}

.chat-input {
  padding: 15px 20px;
  border-top: 1px solid var(--border-color);
  display: flex;
  gap: 10px;
  background: var(--card-bg);
}

.chat-input input {
  flex: 1;
  padding: 10px 15px;
  border: 1px solid var(--border-color);
  border-radius: 20px;
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
  background: var(--bg-color);
  color: var(--text-primary);
}

.chat-input input:focus {
  border-color: var(--primary-color);
}

.send-button {
  width: 40px;
  height: 40px;
  border: none;
  border-radius: 50%;
  background: var(--primary-color);
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background-color 0.2s;
}

.send-button:hover {
  background: var(--primary-hover);
}

.send-button i {
  font-size: 16px;
}
</style>
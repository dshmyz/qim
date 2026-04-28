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
        <!-- Tab 导航 -->
        <div class="bot-tabs">
          <button :class="['bot-tab', { active: activeTab === 'available' }]" @click="activeTab = 'available'">可用机器人</button>
          <button :class="['bot-tab', { active: activeTab === 'my-bots' }]" @click="activeTab = 'my-bots'">我的机器人</button>
          <button class="bot-tab" @click="openCreateModal"><i class="fas fa-plus"></i> 创建机器人</button>
        </div>

        <!-- 可用机器人 Tab 内容 -->
        <div v-show="activeTab === 'available'" class="tab-content">
        <h3>选择模式</h3>
        <div class="mode-list">
          <div class="mode-item" @click="selectMode('chat')">
            <div class="mode-icon">
              <i class="fas fa-comments"></i>
            </div>
            <div class="mode-info">
              <h4>聊天模式</h4>
              <p>与AI进行日常对话，获取信息和建议</p>
            </div>
          </div>
          <div class="mode-item" @click="selectMode('ops')">
            <div class="mode-icon ops">
              <i class="fas fa-server"></i>
            </div>
            <div class="mode-info">
              <h4>运维模式</h4>
              <p>智能故障排查、命令生成、日志分析等</p>
            </div>
          </div>
        </div>

        <div class="bot-selection-header">
          <h3 class="mt-4">选择机器人</h3>
          <button class="create-bot-btn" @click="openCreateModal">
            <i class="fas fa-plus"></i>
            <span>创建机器人</span>
          </button>
        </div>
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
              <span v-if="bot.status === 'pending'" class="bot-status pending">待审批</span>
            </div>
          </div>
          <div v-if="bots.length === 0" class="empty-bots">
            <i class="fas fa-robot"></i>
            <p>暂无可用的机器人</p>
            <button class="create-first-bot-btn" @click="openCreateModal">
              <i class="fas fa-plus"></i>
              <span>创建第一个机器人</span>
            </button>
          </div>
        </div>
        </div>

        <!-- 我的机器人 Tab -->
        <div v-show="activeTab === 'my-bots'" class="tab-content">
          <MyBotsPanel @use-bot="handleUseBot" @edit-bot="handleEditBot" @open-create="openCreateModal" />
        </div>

        <!-- 创建机器人模态框 -->
        <div v-if="showCreateBotModal" class="tool-modal">
          <div class="modal-content">
            <div class="modal-header">
              <h3>{{ !showCreateForm && !createMethod ? '创建机器人' : createMethod === 'template' && !showCreateForm ? '选择模板' : createMethod === 'template' ? '创建机器人（模板）' : '创建自定义机器人' }}</h3>
              <button class="close-button" @click="showCreateBotModal = false">
                <i class="fas fa-times"></i>
              </button>
            </div>
            <div class="modal-body">
              <!-- 步骤1：选择创建方式 -->
              <div v-if="!showCreateForm && !createMethod" class="create-method-selector">
                <h3>选择创建方式</h3>
                <div class="method-options">
                  <div class="method-option recommended" @click="selectCreateMethod('template')">
                    <div class="method-icon"><i class="fas fa-layer-group"></i></div>
                    <h4>使用模板</h4>
                    <p>从预设模板快速创建，推荐新手使用</p>
                  </div>
                  <div class="method-option" @click="selectCreateMethod('custom')">
                    <div class="method-icon"><i class="fas fa-edit"></i></div>
                    <h4>自定义</h4>
                    <p>完全自定义配置，需管理员审批</p>
                  </div>
                </div>
              </div>

              <!-- 步骤2：选择模板 -->
              <div v-else-if="createMethod === 'template' && !showCreateForm" class="template-selector">
                <div v-if="templates.length > 0" class="template-list">
                  <div v-for="tpl in templates" :key="tpl.id" class="template-item" @click="createFromTemplate(tpl)">
                    <div class="template-avatar">
                      <img :src="tpl.avatar" :alt="tpl.name" v-if="tpl.avatar">
                      <i class="fas fa-robot" v-else></i>
                    </div>
                    <div class="template-info">
                      <h4>{{ tpl.name }}</h4>
                      <p>{{ tpl.description }}</p>
                    </div>
                  </div>
                </div>
                <div v-else class="empty-templates">
                  <i class="fas fa-inbox"></i>
                  <p>暂无可用模板，请选择自定义创建</p>
                  <button class="cancel-button" style="margin-top: 12px;" @click="selectCreateMethod('custom')">切换到自定义</button>
                </div>
              </div>

              <!-- 步骤3：填写表单 -->
              <div v-else>
              <div class="form-group">
                <label>机器人名称</label>
                <input v-model="createBotForm.name" type="text" placeholder="请输入机器人名称">
              </div>
              <div class="form-group">
                <label>描述</label>
                <textarea v-model="createBotForm.description" placeholder="请输入机器人描述" rows="3"></textarea>
              </div>
              <div class="form-group">
                <label>机器人类型</label>
                <select v-model="createBotForm.type">
                  <option value="ai">AI 机器人</option>
                  <option value="custom">自定义机器人</option>
                </select>
              </div>
              <div class="form-group" v-if="createBotForm.type === 'ai'">
                <label>AI 提供商</label>
                <select v-model="createBotForm.provider">
                  <option value="openai">OpenAI</option>
                  <option value="baidu">百度文心一言</option>
                  <option value="alibaba">阿里通义千问</option>
                  <option value="tencent">腾讯混元大模型</option>
                  <option value="bytedance">字节跳动豆包</option>
                  <option value="anthropic">Anthropic Claude</option>
                  <option value="custom">自定义模型</option>
                </select>
              </div>
              <div class="form-group" v-if="createBotForm.provider === 'custom'">
                <label>自定义模型地址</label>
                <input v-model="createBotForm.custom_model_url" type="text" placeholder="请输入模型 API 地址">
              </div>
              <div class="form-group" v-if="createBotForm.type === 'custom'">
                <label>龙虾地址</label>
                <input v-model="createBotForm.lobster_url" type="text" placeholder="请输入龙虾地址">
              </div>
              <div class="form-group">
                <label>头像 URL</label>
                <input v-model="createBotForm.avatar" type="text" placeholder="请输入头像 URL（可选）">
              </div>
              </div>
            </div>
            <div class="modal-footer">
              <button class="cancel-button" @click="showCreateBotModal = false">取消</button>
              <button class="submit-button" @click="createBot" :disabled="creatingBot">
                {{ creatingBot ? '创建中...' : (createMethod === 'template' ? '创建' : '提交审批') }}
              </button>
            </div>
          </div>
        </div>
      </div>
      <div v-else class="bot-chat">
        <div class="chat-header">
          <button class="back-button" @click="exitBotChat">
            <i class="fas fa-arrow-left"></i>
          </button>
          <h3>{{ currentBot?.name || (currentMode === 'ops' ? '运维助手' : 'AI 助手') }}</h3>
        </div>
        <div class="chat-messages" ref="chatMessagesRef">
          <div 
            v-for="message in botMessages" 
            :key="message.id" 
            :class="['message', message.sender === 'user' ? 'user-message' : 'bot-message']"
          >
            <div v-if="message.sender === 'bot'" class="message-content" v-html="renderMarkdown(message.content)"></div>
            <div v-else class="message-content">{{ message.content }}</div>
            <div class="message-time">{{ formatTime(message.timestamp) }}</div>
          </div>
          <div v-if="isBotThinking" class="message bot-message thinking">
            <div class="message-content">
              <div class="thinking-indicator">
                <span class="dot"></span>
                <span class="dot"></span>
                <span class="dot"></span>
              </div>
              <span>思考中...</span>
            </div>
          </div>
        </div>
        <div class="chat-input">
          <input 
            v-model="botMessageInput" 
            type="text" 
            :placeholder="currentMode === 'ops' ? '输入运维问题...' : '输入消息...'" 
            @keyup.enter="sendBotMessage"
          >
          <button @click="sendBotMessage" class="send-button">
            <i class="fas fa-paper-plane"></i>
          </button>
        </div>
        <div v-if="currentMode === 'ops'" class="ops-tools">
          <h4>运维工具</h4>
          <div class="tool-buttons">
            <button class="tool-button" @click="openTroubleshootingTool">
              <i class="fas fa-bug"></i>
              <span>故障排查</span>
            </button>
            <button class="tool-button" @click="openCommandTool">
              <i class="fas fa-terminal"></i>
              <span>命令生成</span>
            </button>
            <button class="tool-button" @click="openLogAnalysisTool">
              <i class="fas fa-file-alt"></i>
              <span>日志分析</span>
            </button>
            <button class="tool-button" @click="openAlertTool">
              <i class="fas fa-bell"></i>
              <span>告警处理</span>
            </button>
            <button class="tool-button" @click="openKnowledgeTool">
              <i class="fas fa-book"></i>
              <span>知识问答</span>
            </button>
          </div>
        </div>
      </div>
      
      <!-- 运维工具模态框 -->
      <div v-if="showToolModal" class="tool-modal">
        <div class="modal-content">
          <div class="modal-header">
            <h3>{{ currentToolTitle }}</h3>
            <button class="close-button" @click="closeToolModal">
              <i class="fas fa-times"></i>
            </button>
          </div>
          <div class="modal-body">
            <!-- 故障排查工具 -->
            <div v-if="currentTool === 'troubleshooting'">
              <div class="form-group">
                <label>故障症状</label>
                <textarea v-model="toolForm.symptom" placeholder="描述故障症状..." rows="3"></textarea>
              </div>
              <div class="form-group">
                <label>服务器信息</label>
                <input v-model="toolForm.server" type="text" placeholder="服务器IP或主机名">
              </div>
              <div class="form-group">
                <label>相关日志</label>
                <textarea v-model="toolForm.logs" placeholder="粘贴相关日志..." rows="4"></textarea>
              </div>
            </div>
            
            <!-- 命令生成工具 -->
            <div v-if="currentTool === 'command'">
              <div class="form-group">
                <label>命令描述</label>
                <input v-model="toolForm.description" type="text" placeholder="描述要执行的操作...">
              </div>
              <div class="form-group">
                <label>目标平台</label>
                <select v-model="toolForm.platform">
                  <option value="linux">Linux</option>
                  <option value="windows">Windows</option>
                  <option value="macos">macOS</option>
                </select>
              </div>
              <div class="form-group">
                <label>输出格式</label>
                <select v-model="toolForm.format">
                  <option value="single">单个命令</option>
                  <option value="script">脚本</option>
                </select>
              </div>
            </div>
            
            <!-- 日志分析工具 -->
            <div v-if="currentTool === 'log'">
              <div class="form-group">
                <label>日志内容</label>
                <textarea v-model="toolForm.logContent" placeholder="粘贴日志内容..." rows="6"></textarea>
              </div>
              <div class="form-group">
                <label>服务名称</label>
                <input v-model="toolForm.service" type="text" placeholder="服务名称">
              </div>
              <div class="form-group">
                <label>严重程度</label>
                <select v-model="toolForm.severity">
                  <option value="error">Error</option>
                  <option value="warning">Warning</option>
                  <option value="info">Info</option>
                </select>
              </div>
            </div>
            
            <!-- 告警处理工具 -->
            <div v-if="currentTool === 'alert'">
              <div class="form-group">
                <label>告警内容</label>
                <textarea v-model="toolForm.alertContent" placeholder="粘贴告警内容..." rows="4"></textarea>
              </div>
              <div class="form-group">
                <label>告警级别</label>
                <select v-model="toolForm.severity">
                  <option value="critical">Critical</option>
                  <option value="warning">Warning</option>
                  <option value="info">Info</option>
                </select>
              </div>
              <div class="form-group">
                <label>相关服务</label>
                <input v-model="toolForm.service" type="text" placeholder="相关服务名称">
              </div>
            </div>
            
            <!-- 知识问答工具 -->
            <div v-if="currentTool === 'knowledge'">
              <div class="form-group">
                <label>问题内容</label>
                <textarea v-model="toolForm.question" placeholder="输入运维相关问题..." rows="3"></textarea>
              </div>
              <div class="form-group">
                <label>问题类别</label>
                <select v-model="toolForm.category">
                  <option value="linux">Linux</option>
                  <option value="network">网络</option>
                  <option value="database">数据库</option>
                  <option value="security">安全</option>
                </select>
              </div>
            </div>
          </div>
          <div class="modal-footer">
            <button class="cancel-button" @click="closeToolModal">取消</button>
            <button class="submit-button" @click="executeTool">执行</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick, reactive } from 'vue'
import axios from 'axios'
import { API_BASE_URL } from '../../config'
import { marked } from 'marked'
import { sanitizeMarkdown } from '../../utils/sanitize'
import { logger } from '../../utils/logger';
import MyBotsPanel from './MyBotsPanel.vue'
import { useBots } from '../../composables/useBots'

// 服务器URL
const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

// useBots composable
const { fetchBots, fetchTemplates, fetchMyBotCount, createBot: submitCreateBot } = useBots()

// 定义事件
const emit = defineEmits(['back'])

// AI 助手相关状态
const selectedBotId = ref('')
const bots = ref<any[]>([])
const botMessages = ref<any[]>([])
const botMessageInput = ref('')
const showBotChat = ref(false)
const chatMessagesRef = ref<HTMLDivElement | null>(null)
const isBotThinking = ref(false)

// 模式和工具相关状态
const currentMode = ref<'chat' | 'ops'>('chat')
const showToolModal = ref(false)
const currentTool = ref<string>('')
const currentToolTitle = ref('')

// 运维模式对话历史
const opsMessagesHistory = ref<{role: 'user' | 'assistant'; content: string}[]>([])

// 工具表单数据
const toolForm = ref({
  symptom: '',
  server: '',
  logs: '',
  description: '',
  platform: 'linux',
  format: 'single',
  logContent: '',
  service: '',
  severity: 'error',
  alertContent: '',
  question: '',
  category: 'linux'
})

// 创建机器人相关状态
const showCreateBotModal = ref(false)
const creatingBot = ref(false)
const createBotForm = ref({
  name: '',
  description: '',
  type: 'ai',
  provider: 'openai',
  custom_model_url: '',
  lobster_url: '',
  avatar: ''
})

// Tab 相关
const activeTab = ref<'available' | 'my-bots'>('available')

// 创建方式
const createMethod = ref<'template' | 'custom' | null>(null)
const showCreateForm = ref(false)
const templates = ref<any[]>([])

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
    const response = await fetchBots()
    if (Array.isArray(response)) {
      bots.value = response
    }
  } catch (error) {
    console.error('加载机器人列表失败:', error)
  }
}

// 打开创建模态框
const openCreateModal = async () => {
  showCreateBotModal.value = true
  createMethod.value = null
  showCreateForm.value = false
  // 预加载模板列表
  templates.value = await fetchTemplates()
  // 检查数量限制
  const count = await fetchMyBotCount()
  if (count >= 5) {
    alert('已达到创建上限（5个），如需更多请联系管理员')
    showCreateBotModal.value = false
  }
}

// 选择创建方式
const selectCreateMethod = (method: 'template' | 'custom' | null) => {
  createMethod.value = method
  if (method === 'template') {
    showCreateForm.value = false
  }
}

// 从模板创建
const createFromTemplate = async (tpl: any) => {
  let config = {}
  try { config = JSON.parse(tpl.config || '{}') } catch { config = {} }
  createBotForm.value = {
    name: tpl.name,
    description: tpl.description,
    type: tpl.type || 'ai',
    provider: 'openai',
    custom_model_url: '',
    lobster_url: '',
    avatar: tpl.avatar,
  }
  showCreateForm.value = true
}

// 选择模式
const selectMode = (mode: 'chat' | 'ops') => {
  currentMode.value = mode
  showBotChat.value = true
  botMessages.value = []
}

// 选择机器人
const selectBot = async (botId: number) => {
  selectedBotId.value = botId.toString()
  currentMode.value = 'chat'
  botMessages.value = []
  showBotChat.value = true
  
  // 创建会话并加载历史消息
  try {
    const token = getToken()
    const convResponse = await axios.post(`${serverUrl.value}/api/v1/conversations/single`, {
      recipient_id: 0, // 0 表示机器人
      bot_id: botId
    }, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    
    if (convResponse.data.code === 0) {
      const convId = convResponse.data.data.id
      // 加载历史消息
      await loadBotMessages(convId)
    }
  } catch (error) {
    console.error('创建会话失败:', error)
  }
}

// 退出机器人聊天
const exitBotChat = () => {
  showBotChat.value = false
  selectedBotId.value = ''
  botMessages.value = []
  currentMode.value = 'chat'
  opsMessagesHistory.value = []
}

// 发送消息给机器人
const sendBotMessage = async () => {
  if (!botMessageInput.value.trim()) return
  
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
  
  // 设置思考中状态
  isBotThinking.value = true
  
  try {
    const token = getToken()
    
    if (currentMode.value === 'ops') {
      // 运维模式：使用流式API
      // 将用户消息添加到历史记录
      opsMessagesHistory.value.push({ role: 'user', content: message })
      await sendStreamingRequest(message, token)
    } else {
      // 聊天模式：通过会话发送消息
      if (!selectedBotId.value) return
      
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
        
        // 监听WebSocket消息或轮询获取回复
        // 这里暂时使用轮询方式
        setTimeout(() => {
          loadBotMessages(convId)
        }, 2000)
      }
    }
  } catch (error) {
    console.error('发送消息失败:', error)
    // 显示错误消息
    const errorMessage = {
      id: Date.now() + 1,
      content: '发送消息失败，请稍后再试。',
      sender: 'system',
      timestamp: new Date()
    }
    botMessages.value.push(errorMessage)
    scrollToBottom()
    // 取消思考中状态
    isBotThinking.value = false
  }
}

// 处理 SSE 事件数据的辅助函数
const processEventData = (eventData: string, streamMessage: any) => {
  if (eventData.startsWith("data: ")) {
    eventData = eventData.substring(eventData.indexOf('data: ') + 6)
  }
  logger.log('处理事件数据:', eventData)
  
  try {
    const chunk = JSON.parse(eventData)
    
    if (chunk.finish === 'stop') {
      isBotThinking.value = false
      return true
    }
    
    if (chunk.content) {
      streamMessage.content += chunk.content
      scrollToBottom()
    }
    
    return false
  } catch (parseError) {
    if (eventData === '[DONE]') {
      isBotThinking.value = false
      return true
    }
    if (eventData.startsWith('{"error"')) {
      const errorObj = JSON.parse(eventData)
      throw new Error(errorObj.error)
    }
    logger.log('解析JSON失败，直接使用原始数据:', eventData)
    streamMessage.content += eventData
    scrollToBottom()
    return false
  }
}

// 发送流式请求（带重试机制）
const sendStreamingRequest = async (message: string, token: string, maxRetries = 3) => {
  let lastError: Error | null = null

  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    try {
      const streamMessageId = Date.now() + 1
      const streamMessage = reactive({
        id: streamMessageId,
        content: '',
        sender: 'bot',
        timestamp: new Date()
      })
      botMessages.value.push(streamMessage)
      logger.log('开始流式请求 (尝试 ' + attempt + '/' + maxRetries + '):', message)

      const response = await fetch(`${serverUrl.value}/api/v1/ai/completion/stream`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({
          messages: opsMessagesHistory.value
        })
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error('No response body')
      }

      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { value, done } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })

        while (buffer.includes('\n\n')) {
          const eventEnd = buffer.indexOf('\n\n')
          const event = buffer.substring(0, eventEnd)
          buffer = buffer.substring(eventEnd + 2)

          if (event.startsWith('data: ')) {
            const data = event.slice(6)
            // if (data === '[DONE]') continue
            if (processEventData(data, streamMessage)) break
          }
        }
      }

      opsMessagesHistory.value.push({ role: 'assistant', content: streamMessage.content })
      return
    } catch (error) {
      lastError = error instanceof Error ? error : new Error(String(error))
      console.error('流式请求失败 (尝试 ' + attempt + '/' + maxRetries + '):', lastError.message)

      if (attempt < maxRetries) {
        const delay = Math.pow(2, attempt - 1) * 1000
        logger.log('等待 ' + delay + 'ms 后重试...')
        await new Promise(resolve => setTimeout(resolve, delay))
      }
    }
  }

  console.error('流式请求失败，已达到最大重试次数')
  const errorMessageId = Date.now() + 1
  botMessages.value.push({
    id: errorMessageId,
    content: `流式请求失败: ${lastError?.message || '未知错误'}（已重试 ${maxRetries} 次）`,
    sender: 'system',
    timestamp: new Date()
  })
  await scrollToBottom()
  // 取消思考中状态
  isBotThinking.value = false
}

// 加载机器人消息
const loadBotMessages = async (convId: number) => {
  try {
    const token = getToken()
    const response = await axios.get(`${serverUrl.value}/api/v1/conversations/${convId}/messages`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    
    if (response.data.code === 0) {
      const messages = response.data.data
      // 过滤出机器人消息
      const botMessagesOnly = messages.filter((msg: any) => msg.sender_id === 0)
      // 只添加新的机器人消息
      botMessagesOnly.forEach((msg: any) => {
        const exists = botMessages.value.some(m => m.id === msg.id)
        if (!exists) {
          botMessages.value.push({
            id: msg.id,
            content: msg.content,
            sender: 'bot',
            timestamp: new Date(msg.created_at)
          })
        }
      })
      scrollToBottom()
    }
  } catch (error) {
    console.error('加载消息失败:', error)
  } finally {
    // 取消思考中状态
    isBotThinking.value = false
  }
}

// 打开故障排查工具
const openTroubleshootingTool = () => {
  currentTool.value = 'troubleshooting'
  currentToolTitle.value = '智能故障排查'
  showToolModal.value = true
}

// 打开命令生成工具
const openCommandTool = () => {
  currentTool.value = 'command'
  currentToolTitle.value = '命令生成'
  showToolModal.value = true
}

// 打开日志分析工具
const openLogAnalysisTool = () => {
  currentTool.value = 'log'
  currentToolTitle.value = '日志分析'
  showToolModal.value = true
}

// 打开告警处理工具
const openAlertTool = () => {
  currentTool.value = 'alert'
  currentToolTitle.value = '智能告警处理'
  showToolModal.value = true
}

// 打开知识问答工具
const openKnowledgeTool = () => {
  currentTool.value = 'knowledge'
  currentToolTitle.value = '运维知识问答'
  showToolModal.value = true
}

// 关闭工具模态框
const closeToolModal = () => {
  showToolModal.value = false
  currentTool.value = ''
  currentToolTitle.value = ''
  // 重置表单
  Object.keys(toolForm.value).forEach(key => {
    toolForm.value[key as keyof typeof toolForm.value] = ''
  })
  toolForm.value.platform = 'linux'
  toolForm.value.format = 'single'
  toolForm.value.severity = 'error'
  toolForm.value.category = 'linux'
}

// 创建机器人
const createBot = async () => {
  if (!createBotForm.value.name.trim()) {
    alert('请输入机器人名称')
    return
  }

  creatingBot.value = true
  try {
    const isTemplate = createMethod.value === 'template'
    const response = await submitCreateBot({
      ...createBotForm.value,
      is_template: isTemplate,
    })

    if (response.code === 0) {
      await loadBots()
      showCreateBotModal.value = false
      Object.keys(createBotForm.value).forEach(key => {
        createBotForm.value[key as keyof typeof createBotForm.value] = ''
      })
      createBotForm.value.type = 'ai'
      createBotForm.value.provider = 'openai'
      createMethod.value = null
      showCreateForm.value = false
      alert(isTemplate ? '机器人创建成功' : '已提交审批，等待管理员审核')
    } else {
      alert('创建失败，请稍后再试')
    }
  } catch (error: any) {
    if (error.response?.data?.code === 400 && error.response?.data?.message?.includes('上限')) {
      alert('已达到创建上限，请联系管理员')
    } else {
      alert('创建失败，请稍后再试')
    }
  } finally {
    creatingBot.value = false
  }
}

const handleEditBot = (bot: any) => {
  createBotForm.value = {
    name: bot.name,
    description: bot.description,
    type: bot.type,
    provider: bot.config ? JSON.parse(bot.config).provider || 'openai' : 'openai',
    custom_model_url: '',
    lobster_url: '',
    avatar: bot.avatar,
  }
  showCreateBotModal.value = true
  createMethod.value = 'custom'
  showCreateForm.value = true
}

const handleUseBot = async (bot: any) => {
  activeTab.value = 'available'
  await selectBot(bot.id)
}

// 执行工具
const executeTool = async () => {
  try {
    const token = getToken()
    let response
    
    switch (currentTool.value) {
      case 'troubleshooting':
        response = await axios.post(`${serverUrl.value}/api/v1/ai/ops/troubleshooting`, {
          symptom: toolForm.value.symptom,
          server: toolForm.value.server,
          logs: toolForm.value.logs
        }, {
          headers: {
            'Authorization': `Bearer ${token}`
          }
        })
        break
      case 'command':
        response = await axios.post(`${serverUrl.value}/api/v1/ai/ops/command`, {
          description: toolForm.value.description,
          platform: toolForm.value.platform,
          format: toolForm.value.format
        }, {
          headers: {
            'Authorization': `Bearer ${token}`
          }
        })
        break
      case 'log':
        response = await axios.post(`${serverUrl.value}/api/v1/ai/ops/logs`, {
          log_content: toolForm.value.logContent,
          service: toolForm.value.service,
          severity: toolForm.value.severity
        }, {
          headers: {
            'Authorization': `Bearer ${token}`
          }
        })
        break
      case 'alert':
        response = await axios.post(`${serverUrl.value}/api/v1/ai/ops/alert`, {
          alert_content: toolForm.value.alertContent,
          severity: toolForm.value.severity,
          service: toolForm.value.service
        }, {
          headers: {
            'Authorization': `Bearer ${token}`
          }
        })
        break
      case 'knowledge':
        response = await axios.post(`${serverUrl.value}/api/v1/ai/ops/knowledge`, {
          question: toolForm.value.question,
          category: toolForm.value.category
        }, {
          headers: {
            'Authorization': `Bearer ${token}`
          }
        })
        break
      default:
        return
    }
    
    if (response.data.code === 200) {
      // 添加工具执行结果到聊天消息
      const resultMessage = {
        id: Date.now(),
        content: formatToolResult(response.data.data, currentTool.value),
        sender: 'bot',
        timestamp: new Date()
      }
      botMessages.value.push(resultMessage)
      scrollToBottom()
      closeToolModal()
    }
  } catch (error) {
    console.error('执行工具失败:', error)
    // 显示错误消息
    const errorMessage = {
      id: Date.now(),
      content: '执行工具失败，请稍后再试。',
      sender: 'system',
      timestamp: new Date()
    }
    botMessages.value.push(errorMessage)
    scrollToBottom()
  }
}

// 格式化工具执行结果
const formatToolResult = (result: any, tool: string): string => {
  switch (tool) {
    case 'troubleshooting':
      return `**故障分析**：${result.analysis}\n\n**解决方案**：\n${result.solutions.map((sol: string) => `- ${sol}`).join('\n')}\n\n**推荐操作**：${result.recommended}`
    case 'command':
      return `**生成命令**：\n\`\`\`${result.command}\`\`\`\n\n**命令说明**：${result.explanation}`
    case 'log':
      return `**日志分析**：\n- 错误数量：${result.error_count}\n- 警告数量：${result.warning_count}\n- 发现模式：${result.patterns.join(', ')}\n\n**建议**：\n${result.recommendations.map((rec: string) => `- ${rec}`).join('\n')}`
    case 'alert':
      return `**告警分析**：\n- 类别：${result.category}\n- 优先级：${result.priority}\n- 自动解决：${result.auto_resolve ? '是' : '否'}\n\n**建议操作**：\n${result.actions.map((action: string) => `- ${action}`).join('\n')}`
    case 'knowledge':
      return `**回答**：${result.answer}\n\n**参考资料**：\n${result.references.map((ref: string) => `- ${ref}`).join('\n')}\n\n**推荐操作**：\n${result.recommended.map((rec: string) => `- ${rec}`).join('\n')}`
    default:
      return JSON.stringify(result, null, 2)
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

// 渲染Markdown内容
const renderMarkdown = (content: string): string => {
  if (!content) return ''

  try {
    const result = marked.parse(content)
    if (result instanceof Promise) {
      // 流式内容不完整时，返回原始内容
      return sanitizeMarkdown(content.replace(/\r\n|\n|\r/g, '<br>'))
    }
    // marked解析后的内容可能还包含换行符，需要再次处理
    // 然后使用 DOMPurify 进行消毒，防止 XSS 攻击
    const resultWithBreaks = (result as string).replace(/\r\n|\n|\r/g, '<br>')
    return sanitizeMarkdown(resultWithBreaks)
  } catch (parseError) {
    // 解析失败时返回已处理换行符的原始内容（已消毒）
    return sanitizeMarkdown(content.replace(/\r\n|\n|\r/g, '<br>'))
  }
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
  position: relative;
}

/* 模式选择 */
.mode-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
  margin-bottom: 30px;
}

.mode-item {
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

.mode-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
}

.mode-icon {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  background: var(--bg-color);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 15px;
  font-size: 24px;
  color: var(--primary-color);
}

.mode-icon.ops {
  background: #E3F2FD;
  color: #1976D2;
}

.mode-info h4 {
  margin: 0 0 5px 0;
  font-size: 16px;
  color: var(--text-primary);
}

.mode-info p {
  margin: 0;
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.4;
}

.mt-4 {
  margin-top: 24px;
}

/* 机器人选择 */
.bot-selection {
  padding: 20px;
  flex: 1;
  overflow-y: auto;
}

.bot-selection-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.create-bot-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border: 1px solid var(--primary-color);
  border-radius: 6px;
  background: var(--primary-color);
  color: white;
  cursor: pointer;
  transition: all 0.2s;
  font-size: 14px;
}

.create-bot-btn:hover {
  background: var(--primary-hover);
  transform: translateY(-1px);
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

.bot-status {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
  margin-left: 8px;
}

.bot-status.pending {
  background: #FFF8E1;
  color: #FF9800;
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

.create-first-bot-btn {
  margin-top: 20px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  border: 1px solid var(--primary-color);
  border-radius: 6px;
  background: var(--primary-color);
  color: white;
  cursor: pointer;
  transition: all 0.2s;
  font-size: 14px;
}

.create-first-bot-btn:hover {
  background: var(--primary-hover);
  transform: translateY(-1px);
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
  padding: 10px 14px;
  border-radius: 12px;
  position: relative;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  word-break: break-word;
  white-space: pre-wrap;
}

.user-message {
  align-self: flex-end;
  background: var(--primary-color);
  color: white;
  border-bottom-right-radius: 4px;
}

.bot-message {
  align-self: flex-start;
  background: var(--sidebar-bg);
  color: var(--text-primary);
  border-bottom-left-radius: 4px;
}

.message-content {
  font-size: 14px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-wrap: break-word;
  word-break: break-word;
}

.message-time {
  font-size: 11px;
  opacity: 0.6;
  margin-top: 4px;
  text-align: right;
  transition: opacity 0.3s ease;
}

.message:hover .message-time {
  opacity: 0.8;
}

/* 思考中状态 */
.thinking .message-content {
  display: flex;
  align-items: center;
  gap: 10px;
}

.thinking-indicator {
  display: flex;
  gap: 4px;
}

.thinking-indicator .dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: var(--primary-color);
  animation: pulse 1.5s infinite ease-in-out;
}

.thinking-indicator .dot:nth-child(2) {
  animation-delay: 0.2s;
}

.thinking-indicator .dot:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes pulse {
  0%, 100% {
    opacity: 0.3;
    transform: scale(0.8);
  }
  50% {
    opacity: 1;
    transform: scale(1.2);
  }
}

/* Markdown样式 */
.message-content h1,
.message-content h2,
.message-content h3,
.message-content h4,
.message-content h5,
.message-content h6 {
  margin-top: 16px;
  margin-bottom: 8px;
  font-weight: 600;
  line-height: 1.25;
}

.message-content h1 {
  font-size: 18px;
}

.message-content h2 {
  font-size: 16px;
}

.message-content h3 {
  font-size: 14px;
}

.message-content p {
  margin-bottom: 10px;
  line-height: 1.5;
}

.message-content ul,
.message-content ol {
  margin-bottom: 10px;
  padding-left: 20px;
}

.message-content li {
  margin-bottom: 4px;
  line-height: 1.4;
}

.message-content blockquote {
  border-left: 4px solid var(--primary-color);
  padding-left: 12px;
  margin: 10px 0;
  color: var(--text-secondary);
  font-style: italic;
}

.message-content strong {
  font-weight: 600;
}

.message-content em {
  font-style: italic;
}

.message-content a {
  color: #3b82f6;
  text-decoration: none;
  font-weight: 500;
  transition: all 0.3s ease;
}

.message-content a:hover {
  color: #2563eb;
  text-decoration: underline;
  transform: translateY(-1px);
}

.user-message .message-content a {
  color: #e3f2fd;
}

.user-message .message-content a:hover {
  color: white;
  text-decoration: underline;
}

/* 代码块样式 */
.message-content pre {
  background-color: #f5f5f5;
  border-radius: 6px;
  padding: 12px;
  margin: 10px 0;
  overflow-x: auto;
  font-family: 'Courier New', Courier, monospace;
  font-size: 13px;
  line-height: 1.4;
  border: 1px solid #e0e0e0;
}

.message-content code {
  background-color: #f0f0f0;
  padding: 2px 4px;
  border-radius: 3px;
  font-family: 'Courier New', Courier, monospace;
  font-size: 13px;
  color: #e96900;
}

.message-content pre code {
  background-color: transparent;
  padding: 0;
  color: inherit;
}

/* 深色模式下的代码块样式 */
[data-theme="elegant-dark"] .message-content pre {
  background-color: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.1);
}

[data-theme="elegant-dark"] .message-content code {
  background-color: rgba(255, 255, 255, 0.1);
  color: #ff9800;
}

/* 用户消息中的代码块样式 */
.user-message .message-content pre {
  background-color: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.user-message .message-content code {
  background-color: rgba(255, 255, 255, 0.15);
  color: #ffcc80;
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

/* 运维工具 */
.ops-tools {
  padding: 15px 20px;
  border-top: 1px solid var(--border-color);
  background: var(--bg-color);
}

.ops-tools h4 {
  margin: 0 0 15px 0;
  font-size: 14px;
  color: var(--text-primary);
  font-weight: 600;
}

.tool-buttons {
  display: flex;
  gap: 10px;
  overflow-x: auto;
  padding-bottom: 5px;
}

.tool-button {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 12px 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--card-bg);
  cursor: pointer;
  transition: all 0.2s;
  min-width: 80px;
}

.tool-button:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
  transform: translateY(-2px);
}

.tool-button i {
  font-size: 20px;
  color: var(--primary-color);
  margin-bottom: 5px;
}

.tool-button span {
  font-size: 12px;
  color: var(--text-primary);
  text-align: center;
}

/* 工具模态框 */
.tool-modal {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: var(--card-bg);
  border-radius: 8px;
  width: 90%;
  max-width: 600px;
  max-height: 80vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.modal-header {
  padding: 15px 20px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--bg-color);
}

.modal-header h3 {
  margin: 0;
  font-size: 16px;
  color: var(--text-primary);
  font-weight: 600;
}

.close-button {
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

.close-button:hover {
  background: var(--hover-color);
}

.modal-body {
  padding: 20px;
  flex: 1;
  overflow-y: auto;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

.form-group input,
.form-group select,
.form-group textarea {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
  background: var(--bg-color);
  color: var(--text-primary);
  box-sizing: border-box;
}

.form-group input:focus,
.form-group select:focus,
.form-group textarea:focus {
  border-color: var(--primary-color);
}

.form-group textarea {
  resize: vertical;
  min-height: 100px;
}

.modal-footer {
  padding: 15px 20px;
  border-top: 1px solid var(--border-color);
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  background: var(--bg-color);
}

.cancel-button {
  padding: 8px 16px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--card-bg);
  color: var(--text-primary);
  cursor: pointer;
  transition: all 0.2s;
  font-size: 14px;
}

.cancel-button:hover {
  background: var(--hover-color);
}

.submit-button {
  padding: 8px 16px;
  border: none;
  border-radius: 6px;
  background: var(--primary-color);
  color: white;
  cursor: pointer;
  transition: background-color 0.2s;
  font-size: 14px;
}

.submit-button:hover {
  background: var(--primary-hover);
}

/* 响应式设计 */
/* Tab 导航 */
.bot-tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 24px;
  border-bottom: 2px solid var(--border-color);
  padding-bottom: 0;
}

.bot-tab {
  padding: 10px 20px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  border-bottom: 2px solid transparent;
  margin-bottom: -2px;
  transition: all 0.2s;
}

.bot-tab:hover { color: var(--text-primary); }
.bot-tab.active { color: var(--primary-color); border-bottom-color: var(--primary-color); }

.tab-content { padding-top: 4px; }

/* 创建方式选择器 */
.create-method-selector { padding: 30px; text-align: center; }
.create-method-selector h3 { margin-bottom: 24px; color: var(--text-primary); }
.method-options { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin-bottom: 24px; }
.method-option { padding: 24px; border: 2px solid var(--border-color); border-radius: 12px; cursor: pointer; transition: all 0.2s; }
.method-option:hover { border-color: var(--primary-color); transform: translateY(-2px); }
.method-option.recommended { border-color: var(--primary-color); background: rgba(59, 130, 246, 0.05); }
.method-icon { width: 50px; height: 50px; border-radius: 50%; background: var(--bg-color); display: flex; align-items: center; justify-content: center; margin: 0 auto 12px; font-size: 22px; color: var(--primary-color); }
.method-option h4 { margin: 0 0 8px; color: var(--text-primary); }
.method-option p { margin: 0; font-size: 13px; color: var(--text-secondary); }

/* 模板选择器 */
.template-selector { padding: 20px; }
.template-list { display: flex; flex-direction: column; gap: 12px; }
.template-item { display: flex; align-items: center; gap: 12px; padding: 14px; border: 1px solid var(--border-color); border-radius: 8px; cursor: pointer; transition: all 0.2s; }
.template-item:hover { background: var(--hover-color); border-color: var(--primary-color); }
.template-avatar { width: 44px; height: 44px; border-radius: 50%; background: var(--bg-color); display: flex; align-items: center; justify-content: center; overflow: hidden; }
.template-avatar img { width: 100%; height: 100%; object-fit: cover; }
.template-info h4 { margin: 0 0 4px; font-size: 15px; color: var(--text-primary); }
.template-info p { margin: 0; font-size: 13px; color: var(--text-secondary); }
.empty-templates { text-align: center; padding: 40px; color: var(--text-secondary); }

@media (max-width: 768px) {
  .mode-list {
    grid-template-columns: 1fr;
  }

  .bot-list {
    grid-template-columns: 1fr;
  }

  .tool-buttons {
    flex-wrap: wrap;
  }

  .tool-button {
    flex: 1;
    min-width: calc(20% - 8px);
  }

  .modal-content {
    width: 95%;
    max-height: 90vh;
  }

  .method-options { grid-template-columns: 1fr; }
}
</style>
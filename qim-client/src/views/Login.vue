<template>
  <div class="login-container">
    <!-- 窗口控制按钮 -->
    <div class="window-controls">
      <button class="window-control-btn minimize-btn" @click="minimizeWindow">—</button>
      <button class="window-control-btn maximize-btn" @click="maximizeWindow">☐</button>
      <button class="window-control-btn close-btn" @click="closeWindow">×</button>
    </div>
    
    <!-- 背景装饰 -->
    <div class="background-decoration">
      <div class="circle circle-1"></div>
      <div class="circle circle-2"></div>
      <div class="circle circle-3"></div>
    </div>
    
    <div class="login-form">
      
      <div class="login-header">
        <div class="app-logo">
          <i class="fas fa-comments fa-4x"></i>
        </div>
        <h2>QIM</h2>
        <p>即时通讯应用</p>
      </div>
      
      <!-- 登录表单 -->
      <el-form v-if="!show2FAForm" :model="loginForm" :rules="rules" ref="loginFormRef" label-position="top">
        <el-form-item prop="username" class="form-item">
          <div class="input-wrapper">
            <i class="fas fa-user input-icon"></i>
            <el-input 
              v-model="loginForm.username" 
              placeholder="请输入用户名" 
              class="login-input"
              @focus="handleInputFocus('username')"
              @blur="handleInputBlur('username')"
              @change="checkTwoFactorStatus"
            />
          </div>
        </el-form-item>
        
        <el-form-item prop="password" class="form-item">
          <div class="input-wrapper">
            <i class="fas fa-lock input-icon"></i>
            <el-input 
              v-model="loginForm.password" 
              type="password" 
              placeholder="请输入密码" 
              class="login-input"
              @focus="handleInputFocus('password')"
              @blur="handleInputBlur('password')"
            />
          </div>
        </el-form-item>
        
        <div class="form-options">
          <el-checkbox v-model="loginForm.remember" class="remember-checkbox">记住密码</el-checkbox>
          <el-button link @click="showServerSettings = true" class="server-settings-btn">设置服务器地址</el-button>
        </div>
        
        <el-form-item class="form-item">
          <el-button 
            type="primary" 
            @click="login" 
            class="login-button"
            :loading="isLoading"
          >
            {{ isLoading ? '登录中...' : '登录' }}
          </el-button>
        </el-form-item>
      </el-form>
      
      <!-- 双因素认证表单 -->
      <el-form v-else :model="twoFAForm" :rules="twoFARules" ref="twoFAFormRef" label-position="top" class="twofa-form">
        <h3 class="twofa-title">双因素认证</h3>
        
        <p class="twofa-message">你已开启双因素，请输入动态口令</p>
        
        <el-form-item prop="code" class="twofa-input-item">
          <el-input 
            v-model="twoFAForm.code" 
            placeholder="请输入6位验证码" 
            class="twofa-input"
            @keyup.enter="verifyTwoFA"
            maxlength="6"
          />
        </el-form-item>
        
        <el-form-item class="twofa-button-item">
          <el-button 
            type="primary" 
            @click="verifyTwoFA" 
            class="twofa-button"
            :loading="isLoading"
          >
            {{ isLoading ? '验证中...' : '验证' }}
          </el-button>
        </el-form-item>
        
        <div class="twofa-actions">
          <el-button 
            link 
            @click="resendCode"
            class="twofa-action-btn"
            :disabled="isResending"
          >
            {{ isResending ? '重新发送中...' : '重新发送' }}
          </el-button>
          <el-button 
            link 
            @click="show2FAForm = false"
            class="twofa-action-btn"
          >
            返回登录
          </el-button>
        </div>
      </el-form>
      
    </div>
    
    <!-- 版本号显示 -->
    <div class="version-info">
      <div class="info-row">
        <span class="info-item">版本: {{ packageJson.version }}</span>
        <span class="info-separator">|</span>
        <span class="info-item">© 2026 QIM</span>
      </div>
    </div>
    
    <!-- 服务器设置弹窗 -->
    <el-dialog
      v-model="showServerSettings"
      title="服务器设置"
      width="400px"
    >
      <el-form :model="serverSettings" label-width="100px">
        <el-form-item label="服务器地址">
          <el-input v-model="serverSettings.url" placeholder="请输入服务器地址" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showServerSettings = false">取消</el-button>
          <el-button type="primary" @click="saveServerSettings">保存</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import packageJson from '../../package.json'
import { API_BASE_URL } from '../config'

const emit = defineEmits<{
  (e: 'login-success', user: { username: string }): void
}>()

const loginFormRef = ref<FormInstance>()
const loginForm = reactive({
  username: '',
  password: '',
  remember: false,
  twoFactorEnabled: false
})

const rules = reactive<FormRules>({
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' }
  ]
})

// 双因素认证相关
const show2FAForm = ref(false)
const twoFAFormRef = ref<FormInstance>()
const twoFAForm = reactive({
  code: ''
})

const twoFARules = reactive<FormRules>({
  code: [
    { required: true, message: '请输入验证码', trigger: 'blur' },
    { pattern: /^\d{6}$/, message: '请输入6位数字验证码', trigger: 'blur' }
  ]
})

// 登录会话信息，用于双因素认证
const loginSession = ref<string>('')

// 检查用户双因素认证状态
const checkTwoFactorStatus = async () => {
  if (!loginForm.username) return
  
  try {
    const response = await fetch(`${serverSettings.url}/api/v1/auth/check-2fa`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ username: loginForm.username })
    })
    
    const data = await response.json()
    if (data.code === 0) {
      // 保存用户双因素认证状态
      loginForm.twoFactorEnabled = data.data.twoFactorEnabled
    }
  } catch (error) {
    console.error('检查双因素认证状态失败:', error)
  }
}

const isLoading = ref(false)
const focusedInput = ref<string | null>(null)
const showServerSettings = ref(false)
const isResending = ref(false)

const serverSettings = reactive({
  url: API_BASE_URL
})

const handleInputFocus = (input: string) => {
  focusedInput.value = input
}

const handleInputBlur = (input: string) => {
  if (focusedInput.value === input) {
    focusedInput.value = null
  }
}

const saveServerSettings = () => {
  // 保存服务器地址到本地存储
  localStorage.setItem('serverUrl', serverSettings.url)
  showServerSettings.value = false
  ElMessage.success('服务器地址保存成功')
}

const login = async () => {
  if (!loginFormRef.value) return
  
  await loginFormRef.value.validate(async (valid, fields) => {
    if (valid) {
      isLoading.value = true
      try {
        const response = await fetch(`${serverSettings.url}/api/v1/auth/login`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            username: loginForm.username,
            password: loginForm.password,
            version: packageJson.version
          })
        })
        
        // 无论响应状态码如何，都尝试解析响应数据
        const data = await response.json()
        if (data.code === 0) {
          // 保存token和用户信息
          localStorage.setItem('token', data.data.token)
          localStorage.setItem('user', JSON.stringify(data.data.user))
          
          // 如果勾选了记住密码，保存到本地存储
          if (loginForm.remember) {
            localStorage.setItem('username', loginForm.username)
            localStorage.setItem('password', loginForm.password)
            localStorage.setItem('remember', 'true')
          } else {
            localStorage.removeItem('username')
            localStorage.removeItem('password')
            localStorage.removeItem('remember')
          }
          
          emit('login-success', data.data.user)
        } else if (data.code === 401 && data.message === '需要双因素认证') {
          // 需要双因素认证
          loginSession.value = data.data.session
          show2FAForm.value = true
        } else {
          ElMessage.error(data.message || '登录失败')
        }
      } catch (error) {
        console.error('登录错误:', error)
        ElMessage.error('网络错误，请检查服务器连接')
      } finally {
        isLoading.value = false
      }
    } else {
      console.log('验证失败:', fields)
    }
  })
}

const verifyTwoFA = async () => {
  if (!twoFAFormRef.value) return
  
  await twoFAFormRef.value.validate(async (valid, fields) => {
    if (valid) {
      isLoading.value = true
      try {
        const response = await fetch(`${serverSettings.url}/api/v1/auth/2fa/verify`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            session: loginSession.value,
            code: twoFAForm.code,
            username: loginForm.username
          })
        })
        
        if (response.ok) {
          const data = await response.json()
          if (data.code === 0) {
            // 保存token和用户信息
            localStorage.setItem('token', data.data.token)
            localStorage.setItem('user', JSON.stringify(data.data.user))
            
            // 如果勾选了记住密码，保存到本地存储
            if (loginForm.remember) {
              localStorage.setItem('username', loginForm.username)
              localStorage.setItem('password', loginForm.password)
              localStorage.setItem('remember', 'true')
            } else {
              localStorage.removeItem('username')
              localStorage.removeItem('password')
              localStorage.removeItem('remember')
            }
            
            emit('login-success', data.data.user)
          } else {
            ElMessage.error(data.message || '验证失败')
          }
        } else {
          const errorData = await response.json()
          ElMessage.error(errorData.message || '验证失败')
        }
      } catch (error) {
        console.error('验证错误:', error)
        ElMessage.error('网络错误，请检查服务器连接')
      } finally {
        isLoading.value = false
      }
    } else {
      console.log('验证失败:', fields)
    }
  })
}

// 初始化时加载保存的设置
const loadSavedSettings = () => {
  const savedUsername = localStorage.getItem('username')
  const savedPassword = localStorage.getItem('password')
  const savedRemember = localStorage.getItem('remember')
  const savedServerUrl = localStorage.getItem('serverUrl')
  
  if (savedUsername) {
    loginForm.username = savedUsername
  }
  
  if (savedPassword) {
    loginForm.password = savedPassword
  }
  
  if (savedRemember === 'true') {
    loginForm.remember = true
  }
  
  if (savedServerUrl) {
    serverSettings.url = savedServerUrl
  }
}

// 调用加载保存的设置
loadSavedSettings()

// 重新发送验证码
const resendCode = async () => {
  if (isResending.value) return
  
  isResending.value = true
  try {
    const response = await fetch(`${serverSettings.url}/api/v1/auth/2fa/resend`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        session: loginSession.value,
        username: loginForm.username
      })
    })
    
    const data = await response.json()
    if (data.code === 0) {
      ElMessage.success('验证码已重新发送')
    } else {
      ElMessage.error(data.message || '重新发送验证码失败')
    }
  } catch (error) {
    console.error('重新发送验证码错误:', error)
    ElMessage.error('网络错误，请检查服务器连接')
  } finally {
    isResending.value = false
  }
}

// 窗口控制函数
const minimizeWindow = () => {
  // 发送消息到主进程，请求最小化窗口
  if (window.electron) {
    window.electron.ipcRenderer.send('minimize-window')
  } else {
    console.log('最小化窗口')
  }
}

const maximizeWindow = () => {
  // 发送消息到主进程，请求最大化窗口
  if (window.electron) {
    window.electron.ipcRenderer.send('maximize-window')
  } else {
    console.log('最大化窗口')
  }
}

const closeWindow = () => {
  // 发送消息到主进程，请求关闭窗口
  if (window.electron) {
    window.electron.ipcRenderer.send('close-window')
  } else {
    console.log('关闭窗口')
  }
}
</script>

<style scoped>
.login-container {
  width: 100vw;
  height: 100vh;
  background: linear-gradient(135deg, #64b5f6 0%, #4fc3f7 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  -webkit-app-region: drag;
}

/* 背景装饰 */
.background-decoration {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 0;
}

.circle {
  position: absolute;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.1);
  animation: float 6s ease-in-out infinite;
}

.circle-1 {
  width: 300px;
  height: 300px;
  top: -100px;
  left: -100px;
  animation-delay: 0s;
}

.circle-2 {
  width: 200px;
  height: 200px;
  bottom: -50px;
  right: -50px;
  animation-delay: 2s;
}

.circle-3 {
  width: 150px;
  height: 150px;
  top: 50%;
  right: 10%;
  animation-delay: 4s;
}

@keyframes float {
  0%, 100% {
    transform: translateY(0) translateX(0);
  }
  50% {
    transform: translateY(-20px) translateX(20px);
  }
}

.login-form {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  padding: 56px 48px;
  width: 560px;
  min-height: 500px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.3);
  z-index: 1;
  animation: slideIn 0.5s ease-out;
  -webkit-app-region: no-drag;
}

/* 窗口控制按钮 */
.window-controls {
  position: fixed;
  top: 16px;
  right: 16px;
  display: flex;
  gap: 8px;
  z-index: 1000;
  -webkit-app-region: no-drag;
}

.window-control-btn {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  background: rgba(240, 242, 245, 0.8);
  color: #666;
}

.window-control-btn:hover {
  background: rgba(220, 222, 225, 0.8);
  color: #333;
}

.minimize-btn:hover {
  background: #ffbd2e;
  color: white;
}

.maximize-btn:hover {
  background: #1890ff;
  color: white;
}

.close-btn:hover {
  background: #f5222d;
  color: white;
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(30px) scale(0.95);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.app-logo {
  width: 80px;
  height: 80px;
  background: linear-gradient(135deg, #64b5f6 0%, #4fc3f7 100%);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 24px;
  box-shadow: 0 4px 12px rgba(100, 181, 246, 0.3);
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.05);
  }
}

.app-logo i {
  color: white;
  font-size: 32px;
}

.login-header h2 {
  margin: 0 0 8px 0;
  font-size: 28px;
  font-weight: 700;
  color: #333;
  letter-spacing: 2px;
}

.login-header p {
  margin: 0;
  font-size: 16px;
  color: #666;
}

.form-item {
  margin-bottom: 24px;
}

.input-wrapper {
  position: relative;
  border-radius: 8px;
  background: rgba(240, 242, 245, 0.8);
  transition: all 0.3s ease;
  width: 100%;
  box-sizing: border-box;
}

.input-wrapper:focus-within {
  background: white;
  box-shadow: 0 0 0 3px rgba(100, 181, 246, 0.2);
  transform: translateY(-2px);
}

.input-icon {
  position: absolute;
  left: 16px;
  top: 50%;
  transform: translateY(-50%);
  color: #999;
  transition: all 0.3s ease;
}

.input-wrapper:focus-within .input-icon {
  color: #64b5f6;
}

.login-input {
  padding-left: 48px !important;
  border: none !important;
  background: transparent !important;
  border-radius: 8px !important;
  height: 56px !important;
  font-size: 16px !important;
  transition: all 0.3s ease !important;
}

.login-input:focus {
  box-shadow: none !important;
}

.form-options {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.remember-checkbox {
  font-size: 14px;
  color: #666;
}

.server-settings-btn {
  font-size: 14px;
  color: #64b5f6;
  transition: all 0.3s ease;
}

.server-settings-btn:hover {
  color: #42a5f5;
}

.back-to-login {
  font-size: 14px;
  color: #64b5f6;
  transition: all 0.3s ease;
}

.back-to-login:hover {
  color: #42a5f5;
}

.twofa-form {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 32px 24px;
  text-align: center;
}

.twofa-title {
  margin: 0 0 16px 0;
  font-size: 18px;
  font-weight: 600;
  color: #333;
}

.twofa-message {
  margin: 0 0 32px 0;
  font-size: 14px;
  color: #666;
  line-height: 1.4;
}

.twofa-input-item {
  width: 100%;
  max-width: 280px;
  margin-bottom: 24px !important;
}

.twofa-input {
  width: 100%;
  height: 48px;
  font-size: 16px;
  letter-spacing: 8px;
  text-align: center;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  transition: all 0.3s ease;
}

.twofa-input:focus {
  border-color: #64b5f6;
  box-shadow: 0 0 0 2px rgba(100, 181, 246, 0.2);
}

.twofa-button-item {
  width: 100%;
  max-width: 280px;
  margin-bottom: 24px !important;
}

.twofa-button {
  width: 100%;
  height: 48px;
  font-size: 14px;
  font-weight: 500;
  border-radius: 8px;
  background: linear-gradient(135deg, #64b5f6 0%, #4fc3f7 100%);
  border: none;
  transition: all 0.3s ease;
}

.twofa-button:hover {
  background: linear-gradient(135deg, #42a5f5 0%, #29b6f6 100%);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(100, 181, 246, 0.3);
}

.twofa-actions {
  display: flex;
  justify-content: space-between;
  width: 100%;
  max-width: 280px;
}

.twofa-action-btn {
  font-size: 14px;
  color: #64b5f6;
  padding: 0;
  transition: color 0.3s ease;
}

.twofa-action-btn:hover {
  color: #42a5f5;
}

.twofa-action-btn:disabled {
  color: #999;
  cursor: not-allowed;
}

.login-button {
  width: 100%;
  height: 56px;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 600;
  background: linear-gradient(135deg, #64b5f6 0%, #4fc3f7 100%);
  border: none;
  transition: all 0.3s ease;
}

.login-button:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(100, 181, 246, 0.4);
  background: linear-gradient(135deg, #42a5f5 0%, #29b6f6 100%);
}

.login-button:active {
  transform: translateY(0);
}

/* 版本号样式 */
.version-info {
  position: absolute;
  bottom: 16px;
  right: 16px;
  text-align: right;
  z-index: 10;
  -webkit-app-region: no-drag;
}

.info-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 10px;
  color: rgba(102, 102, 102, 0.6);
}

.info-item {
  white-space: nowrap;
}

.info-separator {
  color: rgba(102, 102, 102, 0.4);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .login-form {
    width: 90%;
    padding: 32px;
  }
  
  .circle-1 {
    width: 200px;
    height: 200px;
  }
  
  .circle-2 {
    width: 150px;
    height: 150px;
  }
  
  .circle-3 {
    width: 100px;
    height: 100px;
  }
  
  .form-options {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
  
  .server-settings-btn {
    align-self: flex-start;
  }
}
</style>
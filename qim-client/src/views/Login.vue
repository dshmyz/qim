<template>
  <div class="login-container">
    <div class="window-controls">
      <button class="window-control-btn minimize-btn" @click="minimizeWindow">—</button>
      <button class="window-control-btn maximize-btn" @click="maximizeWindow">☐</button>
      <button class="window-control-btn close-btn" @click="closeWindow">×</button>
    </div>
    
    <div class="background-decoration">
      <div class="circle circle-1"></div>
      <div class="circle circle-2"></div>
      <div class="circle circle-3"></div>
    </div>
    
    <div class="login-form">
      
      <div class="login-header">
        <div class="app-logo">
          <AppLogo size="extraLarge" />
        </div>
        <h2>QIM 青雀</h2>
        <p>简洁 · 高效 · 智能</p>
      </div>
      
      <form v-if="!show2FAForm" @submit.prevent="login" class="login-form-content">
        <div class="form-item">
          <div class="input-wrapper" :class="{ 'input-error': errors.username }">
            <i class="fas fa-user input-icon"></i>
            <input 
              v-model="loginForm.username"
              type="text"
              placeholder="请输入用户名"
              class="login-input"
              @focus="handleInputFocus('username')"
              @blur="handleInputBlur('username')"
              @change="checkTwoFactorStatus"
            />
          </div>
          <div v-if="errors.username" class="error-message">{{ errors.username }}</div>
        </div>
        
        <div class="form-item">
          <div class="input-wrapper" :class="{ 'input-error': errors.password }">
            <i class="fas fa-lock input-icon"></i>
            <input 
              v-model="loginForm.password"
              type="password"
              placeholder="请输入密码"
              class="login-input"
              @focus="handleInputFocus('password')"
              @blur="handleInputBlur('password')"
            />
          </div>
          <div v-if="errors.password" class="error-message">{{ errors.password }}</div>
        </div>
        
        <div class="form-options">
          <label class="remember-checkbox">
            <input type="checkbox" v-model="loginForm.remember" />
            <span>记住密码</span>
          </label>
          <button type="button" @click="showServerSettings = true" class="server-settings-btn">
            设置服务器地址
          </button>
        </div>
        
        <div class="form-item">
          <button 
            type="submit"
            class="login-button"
            :disabled="isLoading"
          >
            {{ isLoading ? '登录中...' : '登录' }}
          </button>
        </div>
      </form>
      
      <form v-else @submit.prevent="verifyTwoFA" class="twofa-form">
        <h3 class="twofa-title">双因素认证</h3>
        
        <p class="twofa-message">你已开启双因素，请输入动态口令</p>
        
        <div class="twofa-input-item">
          <div class="input-wrapper" :class="{ 'input-error': errors.code }">
            <input 
              v-model="twoFAForm.code"
              type="text"
              placeholder="请输入6位验证码"
              class="twofa-input"
              @keyup.enter="verifyTwoFA"
              maxlength="6"
            />
          </div>
          <div v-if="errors.code" class="error-message">{{ errors.code }}</div>
        </div>
        
        <div class="twofa-button-item">
          <button 
            type="submit"
            class="twofa-button"
            :disabled="isLoading"
          >
            {{ isLoading ? '验证中...' : '验证' }}
          </button>
        </div>
        
        <div class="twofa-actions">
          <button 
            type="button"
            @click="resendCode"
            class="twofa-action-btn"
            :disabled="isResending"
          >
            {{ isResending ? '重新发送中...' : '重新发送' }}
          </button>
          <button 
            type="button"
            @click="show2FAForm = false"
            class="twofa-action-btn"
          >
            返回登录
          </button>
        </div>
      </form>
      
    </div>
    
    <div class="version-info">
      <div class="info-row">
        <span class="info-item">版本: {{ packageJson.version }}</span>
        <span class="info-separator">|</span>
        <span class="info-item">© 2026 QIM</span>
      </div>
    </div>
    
    <div v-if="showServerSettings" class="dialog-overlay" @click.self="showServerSettings = false">
      <div class="dialog-content">
        <div class="dialog-header">
          <h3>服务器设置</h3>
          <button class="dialog-close" @click="showServerSettings = false">×</button>
        </div>
        <div class="dialog-body">
          <div class="form-group">
            <label>服务器地址</label>
            <input v-model="serverSettings.url" type="text" placeholder="请输入服务器地址" />
          </div>
        </div>
        <div class="dialog-footer">
          <button @click="showServerSettings = false">取消</button>
          <button type="button" class="btn-primary" @click="saveServerSettings">保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import packageJson from '../../package.json'
import { API_BASE_URL } from '../config'
import QMessage from '../utils/qmessage'
import AppLogo from '../components/shared/AppLogo.vue'

declare global {
  interface Window {
    electron?: {
      ipcRenderer: {
        send: (channel: string, ...args: any[]) => void
      }
    }
  }
}

interface FormErrors {
  username?: string
  password?: string
  code?: string
}

const emit = defineEmits<{
  (e: 'login-success', user: { username: string }): void
}>()

const loginForm = reactive({
  username: '',
  password: '',
  remember: false,
  twoFactorEnabled: false
})

const twoFAForm = reactive({
  code: ''
})

const errors = ref<FormErrors>({})
const show2FAForm = ref(false)
const loginSession = ref<string>('')
const isLoading = ref(false)
const focusedInput = ref<string | null>(null)
const showServerSettings = ref(false)
const isResending = ref(false)

const serverSettings = reactive({
  url: API_BASE_URL
})

const validateField = (field: keyof FormErrors, value: string): boolean => {
  switch (field) {
    case 'username':
      if (!value.trim()) {
        errors.value.username = '请输入用户名'
        return false
      }
      errors.value.username = undefined
      return true
    case 'password':
      if (!value) {
        errors.value.password = '请输入密码'
        return false
      }
      errors.value.password = undefined
      return true
    case 'code':
      if (!value) {
        errors.value.code = '请输入验证码'
        return false
      }
      if (!/^\d{6}$/.test(value)) {
        errors.value.code = '请输入6位数字验证码'
        return false
      }
      errors.value.code = undefined
      return true
    default:
      return true
  }
}

const validateForm = (formType: 'login' | 'twofa'): boolean => {
  let valid = true
  errors.value = {}

  if (formType === 'login') {
    valid = validateField('username', loginForm.username) && valid
    valid = validateField('password', loginForm.password) && valid
  } else if (formType === 'twofa') {
    valid = validateField('code', twoFAForm.code) && valid
  }

  return valid
}

const checkTwoFactorStatus = async () => {
  if (!loginForm.username) return
  
  try {
    const response = await fetch(`${serverSettings.url}/api/v1/auth/check-2fa`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: loginForm.username })
    })
    
    const data = await response.json()
    if (data.code === 0) {
      loginForm.twoFactorEnabled = data.data.twoFactorEnabled
    }
  } catch (error) {
    console.error('检查双因素认证状态失败:', error)
  }
}

const handleInputFocus = (input: string) => {
  focusedInput.value = input
  errors.value[input as keyof FormErrors] = undefined
}

const handleInputBlur = (input: string) => {
  if (focusedInput.value === input) {
    focusedInput.value = null
    validateField(input as keyof FormErrors, loginForm[input as keyof typeof loginForm] as string)
  }
}

const saveServerSettings = () => {
  localStorage.setItem('serverUrl', serverSettings.url)
  showServerSettings.value = false
  QMessage.success('服务器地址保存成功')

  if (window.electron) {
    window.electron.ipcRenderer.send('set-server-url', serverSettings.url)
  }
}

const login = async () => {
  if (!validateForm('login')) return
  
  isLoading.value = true
  try {
    const response = await fetch(`${serverSettings.url}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: loginForm.username,
        password: loginForm.password,
        version: packageJson.version
      })
    })
    
    const data = await response.json()
    if (data.code === 0) {
      localStorage.setItem('token', data.data.token)
      localStorage.setItem('user', JSON.stringify(data.data.user))
      
      if (loginForm.remember) {
        localStorage.setItem('username', loginForm.username)
        localStorage.setItem('password', btoa(encodeURIComponent(loginForm.password)))
        localStorage.setItem('remember', 'true')
      } else {
        localStorage.removeItem('username')
        localStorage.removeItem('password')
        localStorage.removeItem('remember')
      }
      
      emit('login-success', data.data.user)
    } else if (data.code === 401 && data.message === '需要双因素认证') {
      loginSession.value = data.data.session
      show2FAForm.value = true
    } else {
      QMessage.error(data.message || '登录失败')
    }
  } catch (error) {
    console.error('登录错误:', error)
    QMessage.error('网络错误，请检查服务器连接')
  } finally {
    isLoading.value = false
  }
}

const verifyTwoFA = async () => {
  if (!validateForm('twofa')) return
  
  isLoading.value = true
  try {
    const response = await fetch(`${serverSettings.url}/api/v1/auth/2fa/verify`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        session: loginSession.value,
        code: twoFAForm.code,
        username: loginForm.username
      })
    })
    
    if (response.ok) {
      const data = await response.json()
      if (data.code === 0) {
        localStorage.setItem('token', data.data.token)
        localStorage.setItem('user', JSON.stringify(data.data.user))
        
        if (loginForm.remember) {
          localStorage.setItem('username', loginForm.username)
          localStorage.setItem('password', btoa(encodeURIComponent(loginForm.password)))
          localStorage.setItem('remember', 'true')
        } else {
          localStorage.removeItem('username')
          localStorage.removeItem('password')
          localStorage.removeItem('remember')
        }
        
        emit('login-success', data.data.user)
      } else {
        QMessage.error(data.message || '验证失败')
      }
    } else {
      const errorData = await response.json()
      QMessage.error(errorData.message || '验证失败')
    }
  } catch (error) {
    console.error('验证错误:', error)
    QMessage.error('网络错误，请检查服务器连接')
  } finally {
    isLoading.value = false
  }
}

const loadSavedSettings = () => {
  const savedUsername = localStorage.getItem('username')
  const savedPassword = localStorage.getItem('password')
  const savedRemember = localStorage.getItem('remember')
  const savedServerUrl = localStorage.getItem('serverUrl')
  
  if (savedRemember === 'true' && savedUsername && savedPassword) {
    loginForm.username = savedUsername
    loginForm.password = decodeURIComponent(atob(savedPassword))
    loginForm.remember = true
  }
  if (savedServerUrl) serverSettings.url = savedServerUrl
}

loadSavedSettings()

const resendCode = async () => {
  if (isResending.value) return
  
  isResending.value = true
  try {
    const response = await fetch(`${serverSettings.url}/api/v1/auth/2fa/resend`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        session: loginSession.value,
        username: loginForm.username
      })
    })
    
    const data = await response.json()
    if (data.code === 0) {
      QMessage.success('验证码已重新发送')
    } else {
      QMessage.error(data.message || '重新发送验证码失败')
    }
  } catch (error) {
    console.error('重新发送验证码错误:', error)
    QMessage.error('网络错误，请检查服务器连接')
  } finally {
    isResending.value = false
  }
}

const minimizeWindow = () => {
  if (window.electron) {
    window.electron.ipcRenderer.send('minimize-window')
  } else {
    console.log('最小化窗口')
  }
}

const maximizeWindow = () => {
  if (window.electron) {
    window.electron.ipcRenderer.send('maximize-window')
  } else {
    console.log('最大化窗口')
  }
}

const closeWindow = () => {
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
  background: linear-gradient(135deg, #e8ecf1 0%, #d5dde5 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  -webkit-app-region: drag;
}

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
  background: rgba(100, 181, 246, 0.08);
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
  background: #ffffff;
  border-radius: 16px;
  padding: 56px 48px;
  width: 420px;
  min-height: 500px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06), 0 4px 16px rgba(0, 0, 0, 0.04);
  z-index: 1;
  animation: slideIn 0.5s ease-out;
  -webkit-app-region: no-drag;
}

.window-controls {
  position: fixed;
  top: 16px;
  right: 16px;
  display: flex;
  gap: 8px;
  z-index: 1000;
  -webkit-app-region: no-drag;
  box-shadow: none !important;
  padding: 0;
  height: auto;
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
  background: transparent;
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
  margin-bottom: 28px;
}

.app-logo {
  width: 56px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 16px;
}

.app-logo img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.login-header h2 {
  margin: 0 0 6px 0;
  font-size: 22px;
  font-weight: 600;
  color: #333;
  letter-spacing: 1px;
}

.login-header p {
  margin: 0;
  font-size: 13px;
  color: #888;
}

.login-form-content {
  width: 100%;
}

.form-item {
  margin-bottom: 18px;
}

.input-wrapper {
  position: relative;
  border-radius: 8px;
  background: #f5f7fa;
  transition: all 0.3s ease;
  width: 100%;
  box-sizing: border-box;
  border: 1px solid #e8ecf1;
}

.input-wrapper:focus-within {
  background: #ffffff;
  border-color: #64b5f6;
  box-shadow: 0 0 0 3px rgba(100, 181, 246, 0.1);
}

.input-wrapper.input-error {
  border-color: #f5222d;
  background: #fff2f0;
}

.input-wrapper.input-error:focus-within {
  box-shadow: 0 0 0 3px rgba(245, 34, 45, 0.1);
}

.input-icon {
  position: absolute;
  left: 16px;
  top: 50%;
  transform: translateY(-50%);
  color: #999;
  transition: all 0.3s ease;
  z-index: 1;
}

.input-wrapper:focus-within .input-icon {
  color: #64b5f6;
}

.input-wrapper.input-error .input-icon {
  color: #f5222d;
}

.login-input {
  width: 100%;
  padding: 14px 14px 14px 48px;
  border: none;
  background: transparent;
  border-radius: 8px;
  font-size: 15px;
  transition: all 0.3s ease;
  outline: none;
  box-sizing: border-box;
}

.error-message {
  margin-top: 8px;
  font-size: 12px;
  color: #f5222d;
  padding-left: 4px;
}

.form-options {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 18px;
}

.remember-checkbox {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #666;
  cursor: pointer;
}

.remember-checkbox input[type="checkbox"] {
  width: 16px;
  height: 16px;
  cursor: pointer;
}

.server-settings-btn {
  font-size: 14px;
  color: #666666;
  background: none;
  border: none;
  cursor: pointer;
  transition: all 0.2s ease;
  padding: 0;
}

.server-settings-btn:hover {
  color: #333333;
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
  margin-bottom: 24px;
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
  outline: none;
  background: transparent;
}

.twofa-input:focus {
  border-color: #64b5f6;
  box-shadow: 0 0 0 2px rgba(100, 181, 246, 0.2);
}

.twofa-button-item {
  width: 100%;
  max-width: 280px;
  margin-bottom: 24px;
}

.twofa-button {
  width: 100%;
  height: 48px;
  font-size: 14px;
  font-weight: 500;
  border-radius: 8px;
  background: #5b8def;
  border: none;
  cursor: pointer;
  transition: all 0.2s ease;
  color: white;
}

.twofa-button:hover:not(:disabled) {
  background: #4a7de0;
}

.twofa-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
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
  background: none;
  border: none;
  cursor: pointer;
  transition: color 0.3s ease;
}

.twofa-action-btn:hover:not(:disabled) {
  color: #42a5f5;
}

.twofa-action-btn:disabled {
  color: #999;
  cursor: not-allowed;
}

.login-button {
  width: 100%;
  height: 48px;
  border-radius: 8px;
  font-size: 15px;
  font-weight: 500;
  background: #5b8def;
  border: none;
  cursor: pointer;
  transition: all 0.2s ease;
  color: white;
}

.login-button:hover:not(:disabled) {
  background: #4a7de0;
}

.login-button:active:not(:disabled) {
  transform: translateY(0);
}

.login-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.dialog-overlay {
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

.dialog-content {
  background: white;
  border-radius: 8px;
  width: 400px;
  max-width: 90%;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
}

.dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
}

.dialog-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #333;
}

.dialog-close {
  width: 28px;
  height: 28px;
  border: none;
  background: none;
  font-size: 24px;
  cursor: pointer;
  color: #666;
  border-radius: 4px;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
}

.dialog-close:hover {
  background: #f0f0f0;
  color: #333;
}

.dialog-body {
  padding: 24px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-group label {
  font-size: 14px;
  color: #666;
}

.form-group input {
  padding: 10px 12px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s ease;
}

.form-group input:focus {
  border-color: #64b5f6;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 24px;
}

.dialog-footer button {
  padding: 8px 20px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  background: white;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.dialog-footer button:hover {
  border-color: #c0c4cc;
  background: #f5f7fa;
}

.dialog-footer .btn-primary {
  background: #5b8def;
  border: none;
  color: white;
}

.dialog-footer .btn-primary:hover {
  background: #4a7de0;
}

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
  color: rgba(102, 102, 102, 0.5);
}

.info-item {
  white-space: nowrap;
}

.info-separator {
  color: rgba(102, 102, 102, 0.4);
}

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

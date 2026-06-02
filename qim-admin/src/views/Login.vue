<template>
  <div class="login-container">
    <!-- 背景装饰 -->
    <div class="login-bg">
      <div class="bg-shape bg-shape-1"></div>
      <div class="bg-shape bg-shape-2"></div>
      <div class="bg-shape bg-shape-3"></div>
      <div class="bg-grid"></div>
    </div>

    <!-- 登录卡片 -->
    <div class="login-card-wrapper">
      <div class="login-card">
        <div class="login-header">
          <img src="/app-logo-v1.png" alt="QIM Logo" class="logo-img" />
          <h1 class="login-title">QIM Admin</h1>
          <p class="login-subtitle">企业级即时通讯管理后台</p>
        </div>

        <el-form
          :model="loginForm"
          :rules="rules"
          ref="formRef"
          @submit.prevent="handleLogin"
          class="login-form"
        >
          <el-form-item prop="username" class="form-item">
            <el-input
              v-model="loginForm.username"
              placeholder="请输入用户名"
              size="large"
              :prefix-icon="User"
            />
          </el-form-item>

          <el-form-item prop="password" class="form-item">
            <el-input
              v-model="loginForm.password"
              type="password"
              placeholder="请输入密码"
              size="large"
              show-password
              :prefix-icon="Lock"
              @keyup.enter="handleLogin"
            />
          </el-form-item>

          <div class="form-actions">
            <el-button
              type="primary"
              @click="handleLogin"
              :loading="loading"
              size="large"
              class="login-btn"
            >
              <span>登 录</span>
              <el-icon :size="18"><ArrowRight /></el-icon>
            </el-button>
          </div>
        </el-form>

        <div class="login-footer">
          <span>{{ copyrightShort }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock, ArrowRight } from '@element-plus/icons-vue'
import type { FormInstance } from 'element-plus'
import { login } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import { usePermissionStore } from '@/stores/permission'
import { getCopyrightShort } from '@/config/appConfig'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const permissionStore = usePermissionStore()
const formRef = ref<FormInstance>()
const loading = ref(false)
const copyrightShort = getCopyrightShort()

const loginForm = reactive({
  username: '',
  password: '',
})

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

const handleLogin = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  loading.value = true
  try {
    const { data } = await login(loginForm)
    if (data.data) {
      authStore.setToken(data.data.token)
      authStore.setUser(data.data.user)

      // 初始化权限
      if (data.data.user.roles?.length) {
        permissionStore.setRoles(data.data.user.roles.map((r: string) => ({ id: 0, name: r, code: r, description: '', permissions: [], userCount: 0, createdAt: '' })))
      }
      permissionStore.markInitialized()

      // 跳转到 redirect 页面或后台首页
      const redirect = route.query.redirect as string
      router.push(redirect || '/')
    }
  } catch (error: any) {
    const message = error?.response?.data?.message || '登录失败，请检查用户名和密码'
    ElMessage.error(message)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%);
  position: relative;
  overflow: hidden;
}

/* ==========================================
   背景装饰
   ========================================== */
.login-bg {
  position: absolute;
  inset: 0;
  overflow: hidden;
}

.bg-shape {
  position: absolute;
  border-radius: 50%;
  opacity: 0.08;
}

.bg-shape-1 {
  width: 600px;
  height: 600px;
  background: linear-gradient(135deg, #0ea5e9, #6366f1);
  top: -200px;
  right: -100px;
  animation: float 20s ease-in-out infinite;
}

.bg-shape-2 {
  width: 400px;
  height: 400px;
  background: linear-gradient(135deg, #10b981, #0ea5e9);
  bottom: -100px;
  left: -100px;
  animation: float 15s ease-in-out infinite reverse;
}

.bg-shape-3 {
  width: 300px;
  height: 300px;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  top: 50%;
  left: 60%;
  animation: float 18s ease-in-out infinite;
}

.bg-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.02) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.02) 1px, transparent 1px);
  background-size: 60px 60px;
}

@keyframes float {
  0%, 100% {
    transform: translate(0, 0) scale(1);
  }
  50% {
    transform: translate(30px, -30px) scale(1.1);
  }
}

/* ==========================================
   登录卡片
   ========================================== */
.login-card-wrapper {
  position: relative;
  z-index: 10;
  width: 100%;
  max-width: 440px;
  padding: 0 var(--space-4);
}

.login-card {
  background: rgba(255, 255, 255, 0.97);
  backdrop-filter: blur(24px);
  border-radius: var(--radius-2xl);
  box-shadow: 0 32px 64px -12px rgba(0, 0, 0, 0.2);
  padding: var(--space-10) var(--space-8);
  animation: slideUp 0.6s var(--ease-out);
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(30px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 品牌头部 */
.login-header {
  text-align: center;
  margin-bottom: var(--space-8);
}

.logo-img {
  width: 120px;
  height: 120px;
  object-fit: contain;
  margin: 0 auto var(--space-4);
  display: block;
}

.login-title {
  font-size: 28px;
  font-weight: 800;
  color: var(--color-text-primary);
  margin: 0 0 var(--space-2);
  letter-spacing: -0.02em;
}

.login-subtitle {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin: 0;
}

/* 表单 */
.login-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.form-item {
  margin-bottom: 0;
}

:deep(.el-input__wrapper) {
  border-radius: var(--radius-lg) !important;
  height: 48px !important;
  box-shadow: 0 0 0 1px var(--color-border) inset !important;
  transition: all var(--duration-normal) var(--ease-out) !important;
}

:deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px var(--color-primary-light) inset !important;
}

:deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 2px var(--color-primary) inset, var(--shadow-input-focus) !important;
}

:deep(.el-input__inner) {
  font-size: 15px !important;
}

/* 按钮 */
.form-actions {
  margin-top: var(--space-6);
}

.login-btn {
  width: 100%;
  height: 52px;
  border-radius: var(--radius-lg);
  font-size: 16px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: var(--gradient-primary);
  border: none;
  box-shadow: 0 4px 14px -2px rgba(14, 165, 233, 0.35);
  transition: all var(--duration-normal) var(--ease-out);
  letter-spacing: 0.04em;
}

.login-btn:hover {
  transform: translateY(-2px) scale(1.02);
  box-shadow: 0 8px 24px -4px rgba(14, 165, 233, 0.45);
}

.login-btn:active {
  transform: translateY(0) scale(0.98);
}

/* 页脚 */
.login-footer {
  text-align: center;
  margin-top: var(--space-8);
  padding-top: var(--space-6);
  border-top: 1px solid var(--color-border);
  font-size: 13px;
  color: var(--color-text-muted);
}

/* 响应式 */
@media (max-width: 480px) {
  .login-card {
    padding: var(--space-8) var(--space-6);
  }

  .login-title {
    font-size: 24px;
  }

  .bg-shape-1 {
    width: 300px;
    height: 300px;
  }

  .bg-shape-2,
  .bg-shape-3 {
    width: 200px;
    height: 200px;
  }
}
</style>

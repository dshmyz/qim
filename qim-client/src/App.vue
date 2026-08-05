<template>
  <div class="im-app">
    <Login v-if="!isLoggedIn" @login-success="handleLoginSuccess" />
    <Main v-else @logout="handleLogout" />
    <QMessage />
    <QMessageBox />
  </div>
</template>

<script setup lang="ts">
import { ref, onUnmounted } from 'vue'
import Main from './views/Main.vue'
import Login from './views/Login.vue'
import { logger } from './utils/logger'
import { useRenderRulesStore } from './stores/renderRules'

const isLoggedIn = ref(false)
const renderRulesStore = useRenderRulesStore()
let renderRulesTimer: number | undefined

const handleLoginSuccess = (user: any) => {
  logger.log('登录成功:', user)
  isLoggedIn.value = true
  // 登录后拉取渲染规则，启动 5 分钟轮询
  renderRulesStore.fetchRules()
  renderRulesTimer = window.setInterval(() => renderRulesStore.fetchRules(), 5 * 60 * 1000)
}

const handleLogout = () => {
  logger.log('退出登录')
  isLoggedIn.value = false
  if (renderRulesTimer) {
    clearInterval(renderRulesTimer)
    renderRulesTimer = undefined
  }
  localStorage.removeItem('user')
  localStorage.removeItem('token')
  localStorage.removeItem('refresh_token')
}

// 不做启动自动登录：每次启动都停留在登录页，由用户手动登录。
// 曾经的实现会用 localStorage 里残留的 token+user 直接恢复会话（isLoggedIn=true），
// 该行为已按产品决策移除——应用重启后必须重新登录。
onUnmounted(() => {
  if (renderRulesTimer) clearInterval(renderRulesTimer)
})
</script>

<style scoped>
.im-app {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
}
</style>

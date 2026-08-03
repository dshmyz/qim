<template>
  <div class="im-app">
    <Login v-if="!isLoggedIn" @login-success="handleLoginSuccess" />
    <Main v-else @logout="handleLogout" />
    <QMessage />
    <QMessageBox />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
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

onMounted(() => {
  const token = localStorage.getItem('token')
  const user = localStorage.getItem('user')
  if (token && user) {
    isLoggedIn.value = true
    // 已登录态恢复：拉取渲染规则，启动 5 分钟轮询
    renderRulesStore.fetchRules()
    renderRulesTimer = window.setInterval(() => renderRulesStore.fetchRules(), 5 * 60 * 1000)
  }
})

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

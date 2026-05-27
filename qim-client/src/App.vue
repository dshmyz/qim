<template>
  <div class="im-app">
    <Login v-if="!isLoggedIn" @login-success="handleLoginSuccess" />
    <Main v-else @logout="handleLogout" />
    <QMessage />
    <QMessageBox />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Main from './views/Main.vue'
import Login from './views/Login.vue'
import { logger } from './utils/logger'

const isLoggedIn = ref(false)

const handleLoginSuccess = (user: any) => {
  logger.log('登录成功:', user)
  isLoggedIn.value = true
}

const handleLogout = () => {
  logger.log('退出登录')
  isLoggedIn.value = false
  localStorage.removeItem('user')
  localStorage.removeItem('token')
}

onMounted(() => {
  isLoggedIn.value = false
  localStorage.removeItem('user')
  localStorage.removeItem('token')
})
</script>

<style scoped>
.im-app {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
}
</style>

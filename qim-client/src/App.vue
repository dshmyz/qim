<template>
  <div class="im-app">
    <Login v-if="!isLoggedIn" @login-success="handleLoginSuccess" />
    <Main v-else @logout="handleLogout" />
    <VersionUpdateDialog
      v-if="showVersionUpdate"
      :visible="showVersionUpdate"
      :current-version="currentVersion"
      :latest-version="latestVersion"
      :release-notes="releaseNotes"
      :update-url="updateUrl"
      :force-update="forceUpdate"
      @close="showVersionUpdate = false"
      @update="handleUpdate"
    />
    <QMessage />
    <QMessageBox />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Main from './views/Main.vue'
import Login from './views/Login.vue'
import VersionUpdateDialog from './components/modals/VersionUpdateDialog.vue'
import { checkVersionUpdate, getCurrentVersion } from './utils/version'
import { logger } from './utils/logger';

const isLoggedIn = ref(false)

// 版本更新相关状态
const showVersionUpdate = ref(false)
const currentVersion = ref(getCurrentVersion())
const latestVersion = ref('')
const releaseNotes = ref('')
const updateUrl = ref('')
const forceUpdate = ref(false)

const handleLoginSuccess = (user: any) => {
  logger.log('登录成功:', user)
  isLoggedIn.value = true
}

const handleLogout = () => {
  logger.log('退出登录')
  isLoggedIn.value = false
  // 清除本地存储的用户信息和token
  localStorage.removeItem('user')
  localStorage.removeItem('token')
}

const handleUpdate = () => {
  logger.log('用户点击了更新')
  // 可以在这里添加更新相关的逻辑
}

// 检查版本更新
const checkForUpdates = async () => {
  try {
    const updateInfo = await checkVersionUpdate()
    if (updateInfo && updateInfo.needUpdate) {
      latestVersion.value = updateInfo.latestVersion
      releaseNotes.value = updateInfo.releaseNotes
      updateUrl.value = updateInfo.updateUrl
      forceUpdate.value = updateInfo.forceUpdate
      showVersionUpdate.value = true
    }
  } catch (error) {
    console.error('版本检查失败:', error)
  }
}

// 初始化时检查本地存储，保持登录状态
onMounted(() => {
  // 强制跳转到登录页面，以便重新登录
  isLoggedIn.value = false
  localStorage.removeItem('user')
  localStorage.removeItem('token')
  
  // 检查版本更新
  checkForUpdates()
})
</script>

<style scoped>
.im-app {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
}
</style>

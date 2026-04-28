<!-- src/layouts/AdminLayout.vue -->
<template>
  <el-container class="admin-layout">
    <Sidebar :collapsed="isCollapsed" @toggle="isCollapsed = !isCollapsed" />
    <el-container class="main-container">
      <Header>
        <template #breadcrumb>
          <Breadcrumb :title="currentTitle" />
        </template>
      </Header>
      <el-main class="admin-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import Sidebar from '@/components/layout/Sidebar/index.vue'
import Header from '@/components/layout/Header/index.vue'
import Breadcrumb from '@/components/layout/Breadcrumb/index.vue'

const route = useRoute()
const isCollapsed = ref(false)

const titleMap: Record<string, string> = {
  '/': '仪表盘',
  '/statistics': '数据统计',
  '/users': '用户管理',
  '/organization': '组织架构',
  '/roles': '角色权限',
  '/groups': '群组管理',
  '/conversations': '会话管理',
  '/channels': '频道管理',
  '/apps': '应用管理',
  '/mini-apps': '小程序管理',
  '/ai-assistant': 'AI 助手',
  '/ai-ops': 'AI 运维面板',
  '/messages': '系统消息',
  '/notifications': '通知管理',
  '/blacklist': '黑名单管理',
  '/sensitive-words': '敏感词管理',
  '/operation-logs': '操作日志',
  '/system-config': '系统配置',
  '/version-management': '版本管理',
}

const currentTitle = computed(() => titleMap[route.path] || '仪表盘')
</script>

<style scoped>
.admin-layout {
  height: 100vh;
  background-color: var(--color-bg-page);
}

.main-container {
  background-color: var(--color-bg-page);
}

.admin-main {
  background-color: var(--color-bg-page);
  padding: var(--space-6);
  overflow-y: auto;
}
</style>

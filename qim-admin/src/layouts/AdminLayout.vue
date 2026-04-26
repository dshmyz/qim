<template>
  <el-container class="admin-layout">
    <el-aside :width="asideWidth" class="admin-aside" :class="{ 'is-collapsed': isCollapsed }">
      <div class="logo-container">
        <div class="logo-icon">
          <el-icon :size="28"><ChatDotRound /></el-icon>
        </div>
        <h2 class="logo-text" v-show="!isCollapsed">QIM Admin</h2>
      </div>

      <div class="menu-wrapper">
        <el-menu
          :default-active="currentRoute"
          :collapse="isCollapsed"
          router
          class="sidebar-menu"
        >
          <!-- 数据概览 -->
          <el-sub-menu index="dashboard-group">
            <template #title>
              <el-icon><DataAnalysis /></el-icon>
              <span>数据概览</span>
            </template>
            <el-menu-item index="/">
              <el-icon><HomeFilled /></el-icon>
              <template #title>仪表盘</template>
            </el-menu-item>
            <el-menu-item index="/statistics">
              <el-icon><TrendCharts /></el-icon>
              <template #title>数据统计</template>
            </el-menu-item>
          </el-sub-menu>

          <!-- 用户与组织 -->
          <el-sub-menu index="user-group">
            <template #title>
              <el-icon><User /></el-icon>
              <span>用户与组织</span>
            </template>
            <el-menu-item index="/users">
              <el-icon><User /></el-icon>
              <template #title>用户管理</template>
            </el-menu-item>
            <el-menu-item index="/organization">
              <el-icon><School /></el-icon>
              <template #title>组织架构</template>
            </el-menu-item>
            <el-menu-item index="/roles">
              <el-icon><Key /></el-icon>
              <template #title>角色权限</template>
            </el-menu-item>
          </el-sub-menu>

          <!-- 会话与群组 -->
          <el-sub-menu index="chat-group">
            <template #title>
              <el-icon><ChatDotRound /></el-icon>
              <span>会话与群组</span>
            </template>
            <el-menu-item index="/groups">
              <el-icon><UserFilled /></el-icon>
              <template #title>群组管理</template>
            </el-menu-item>
            <el-menu-item index="/conversations">
              <el-icon><ChatLineSquare /></el-icon>
              <template #title>会话管理</template>
            </el-menu-item>
            <el-menu-item index="/channels">
              <el-icon><Connection /></el-icon>
              <template #title>频道管理</template>
            </el-menu-item>
          </el-sub-menu>

          <!-- 应用生态 -->
          <el-sub-menu index="app-group">
            <template #title>
              <el-icon><Grid /></el-icon>
              <span>应用生态</span>
            </template>
            <el-menu-item index="/apps">
              <el-icon><Monitor /></el-icon>
              <template #title>应用管理</template>
            </el-menu-item>
            <el-menu-item index="/mini-apps">
              <el-icon><Cellphone /></el-icon>
              <template #title>小程序管理</template>
            </el-menu-item>
            <el-menu-item index="/ai-assistant">
              <el-icon><Cpu /></el-icon>
              <template #title>AI 助手</template>
            </el-menu-item>
          </el-sub-menu>

          <!-- 消息与通知 -->
          <el-sub-menu index="msg-group">
            <template #title>
              <el-icon><Bell /></el-icon>
              <span>消息与通知</span>
            </template>
            <el-menu-item index="/messages">
              <el-icon><ChatDotRound /></el-icon>
              <template #title>系统消息</template>
            </el-menu-item>
            <el-menu-item index="/notifications">
              <el-icon><BellFilled /></el-icon>
              <template #title>通知管理</template>
            </el-menu-item>
          </el-sub-menu>

          <!-- 安全与合规 -->
          <el-sub-menu index="security-group">
            <template #title>
              <el-icon><Lock /></el-icon>
              <span>安全与合规</span>
            </template>
            <el-menu-item index="/blacklist">
              <el-icon><CircleCloseFilled /></el-icon>
              <template #title>黑名单管理</template>
            </el-menu-item>
            <el-menu-item index="/sensitive-words">
              <el-icon><Warning /></el-icon>
              <template #title>敏感词管理</template>
            </el-menu-item>
            <el-menu-item index="/operation-logs">
              <el-icon><Document /></el-icon>
              <template #title>操作日志</template>
            </el-menu-item>
          </el-sub-menu>

          <!-- 系统设置 -->
          <el-sub-menu index="system-group">
            <template #title>
              <el-icon><Setting /></el-icon>
              <span>系统设置</span>
            </template>
            <el-menu-item index="/system-config">
              <el-icon><Tools /></el-icon>
              <template #title>系统配置</template>
            </el-menu-item>
            <el-menu-item index="/version-management">
              <el-icon><Upload /></el-icon>
              <template #title>版本管理</template>
            </el-menu-item>
          </el-sub-menu>
        </el-menu>
      </div>

      <button class="collapse-btn" @click="toggleSidebar" :title="isCollapsed ? '展开侧边栏' : '收起侧边栏'">
        <el-icon :size="18">
          <Fold v-if="!isCollapsed" />
          <Expand v-else />
        </el-icon>
      </button>
    </el-aside>

    <el-container class="main-container">
      <el-header class="admin-header">
        <div class="header-left">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item>{{ currentTitle }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="header-right">
          <button class="theme-toggle" @click="toggleTheme" :title="isDark ? '切换到亮色主题' : '切换到暗色主题'">
            <el-icon :size="20">
              <Sunny v-if="isDark" />
              <Moon v-else />
            </el-icon>
          </button>

          <el-dropdown trigger="click">
            <span class="user-dropdown">
              <el-avatar :size="34">
                {{ authStore.user?.username?.charAt(0) || 'A' }}
              </el-avatar>
              <span class="username">{{ authStore.user?.username || 'Admin' }}</span>
              <el-icon :size="14"><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="handleLogout">
                  <el-icon><SwitchButton /></el-icon>
                  <span>退出登录</span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <el-main class="admin-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  HomeFilled, User, UserFilled, ChatDotRound, Bell,
  CircleCloseFilled, TrendCharts, School, ChatLineSquare,
  Connection, Grid, Monitor, Cellphone, BellFilled,
  Fold, Expand, Sunny, Moon, ArrowDown, SwitchButton,
  DataAnalysis, Key, Cpu, Warning, Document,
  Lock, Setting, Tools, Upload,
} from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const isCollapsed = ref(false)
const isDark = ref(false)

const asideWidth = computed(() => isCollapsed.value ? '64px' : '240px')

const currentRoute = computed(() => route.path)
const currentTitles: Record<string, string> = {
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
  '/messages': '系统消息',
  '/notifications': '通知管理',
  '/blacklist': '黑名单管理',
  '/sensitive-words': '敏感词管理',
  '/operation-logs': '操作日志',
  '/system-config': '系统配置',
  '/version-management': '版本管理',
}
const currentTitle = computed(() => currentTitles[route.path] || '仪表盘')

const toggleSidebar = () => {
  isCollapsed.value = !isCollapsed.value
}

const toggleTheme = () => {
  isDark.value = !isDark.value
  document.documentElement.setAttribute('data-theme', isDark.value ? 'dark' : 'light')
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

onMounted(() => {
  const savedTheme = localStorage.getItem('theme')
  if (savedTheme === 'dark') {
    isDark.value = true
    document.documentElement.setAttribute('data-theme', 'dark')
  }
})

const handleLogout = () => {
  authStore.logout()
  router.push('/login')
}
</script>

<style scoped>
.admin-layout {
  height: 100vh;
  background-color: var(--color-bg-page);
}

/* ==========================================
   侧边栏 - 扁平深色风格
   ========================================== */
.admin-aside {
  background: var(--sidebar-bg);
  overflow: hidden;
  position: relative;
  transition: width var(--duration-normal) var(--ease-out);
  box-shadow: 4px 0 16px rgba(0, 0, 0, 0.08);
  z-index: 10;
}

.logo-container {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 0 var(--space-4);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.logo-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--gradient-primary);
  border-radius: var(--radius-md);
  color: white;
  flex-shrink: 0;
  box-shadow: 0 2px 8px rgba(14, 165, 233, 0.25);
}

.logo-text {
  color: white;
  font-size: 18px;
  font-weight: 800;
  margin: 0;
  white-space: nowrap;
  letter-spacing: -0.02em;
}

.menu-wrapper {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: var(--space-2) 0;
}

.sidebar-menu {
  background: transparent !important;
  border-right: none !important;
}

/* 侧边栏菜单样式 */
:deep(.el-menu-item),
:deep(.el-sub-menu__title) {
  color: var(--sidebar-text) !important;
  height: 44px !important;
  line-height: 44px !important;
  font-weight: 500 !important;
}

:deep(.el-menu-item:hover),
:deep(.el-sub-menu__title:hover) {
  background-color: rgba(255, 255, 255, 0.06) !important;
  color: var(--sidebar-text-active) !important;
}

:deep(.el-menu-item.is-active) {
  background: rgba(255, 255, 255, 0.1) !important;
  color: var(--sidebar-text-active) !important;
  font-weight: 700 !important;
}

:deep(.el-sub-menu .el-menu-item) {
  min-width: auto !important;
  margin: 2px 8px !important;
  background: rgba(255, 255, 255, 0.03) !important;
  border-radius: var(--radius-sm) !important;
}

:deep(.el-sub-menu .el-menu-item:hover) {
  background: rgba(255, 255, 255, 0.08) !important;
}

:deep(.el-sub-menu .el-menu-item.is-active) {
  background: rgba(255, 255, 255, 0.15) !important;
  color: white !important;
}

:deep(.el-sub-menu .el-menu) {
  background: rgba(0, 0, 0, 0.12) !important;
  border-radius: var(--radius-lg);
  margin: 4px 8px;
}

.collapse-btn {
  position: absolute;
  bottom: var(--space-4);
  left: 50%;
  transform: translateX(-50%);
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: var(--radius-md);
  color: rgba(255, 255, 255, 0.7);
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-out);
}

.collapse-btn:hover {
  background: rgba(255, 255, 255, 0.15);
  color: white;
  transform: translateX(-50%) scale(1.05);
}

/* ==========================================
   主内容区域
   ========================================== */
.main-container {
  background-color: var(--color-bg-page);
}

.admin-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 64px;
  background-color: var(--color-surface);
  padding: 0 var(--space-6);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.header-left {
  display: flex;
  align-items: center;
}

.header-right {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

/* 主题切换按钮 */
.theme-toggle {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-out);
}

.theme-toggle:hover {
  background-color: var(--color-primary-lighter);
  color: var(--color-primary);
  border-color: var(--color-primary);
  transform: scale(1.05);
}

/* 用户下拉 */
.user-dropdown {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-lg);
  transition: background-color var(--duration-fast) var(--ease-out);
}

.user-dropdown:hover {
  background-color: var(--color-surface-hover);
}

.username {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}

/* 主内容区 */
.admin-main {
  background-color: var(--color-bg-page);
  padding: var(--space-6);
  overflow-y: auto;
}

/* 响应式 */
@media (max-width: 768px) {
  .admin-aside {
    position: fixed;
    left: 0;
    top: 0;
    bottom: 0;
    z-index: 1000;
    width: 240px;
    transform: translateX(-100%);
    transition: transform var(--duration-normal) var(--ease-out);
  }

  .admin-aside.is-collapsed {
    transform: translateX(0);
    width: 240px;
  }

  .admin-header {
    padding: 0 var(--space-4);
  }

  .username {
    display: none;
  }
}
</style>

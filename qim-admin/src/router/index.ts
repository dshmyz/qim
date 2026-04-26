import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/',
    component: () => import('@/layouts/AdminLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'Dashboard',
        component: () => import('@/views/Dashboard.vue'),
        meta: { title: '仪表盘' },
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('@/views/Users.vue'),
        meta: { title: '用户管理' },
      },
      {
        path: 'organization',
        name: 'Organization',
        component: () => import('@/views/Organization.vue'),
        meta: { title: '组织架构' },
      },
      {
        path: 'groups',
        name: 'Groups',
        component: () => import('@/views/Groups.vue'),
        meta: { title: '群组管理' },
      },
      {
        path: 'conversations',
        name: 'Conversations',
        component: () => import('@/views/Conversations.vue'),
        meta: { title: '会话管理' },
      },
      {
        path: 'channels',
        name: 'Channels',
        component: () => import('@/views/Channels.vue'),
        meta: { title: '频道管理' },
      },
      {
        path: 'apps',
        name: 'Apps',
        component: () => import('@/views/Apps.vue'),
        meta: { title: '应用管理' },
      },
      {
        path: 'mini-apps',
        name: 'MiniApps',
        component: () => import('@/views/MiniApps.vue'),
        meta: { title: '小程序管理' },
      },
      {
        path: 'messages',
        name: 'Messages',
        component: () => import('@/views/SystemMessages.vue'),
        meta: { title: '系统消息' },
      },
      {
        path: 'notifications',
        name: 'Notifications',
        component: () => import('@/views/Notifications.vue'),
        meta: { title: '通知管理' },
      },
      {
        path: 'blacklist',
        name: 'Blacklist',
        component: () => import('@/views/Blacklist.vue'),
        meta: { title: '黑名单管理' },
      },
      {
        path: 'statistics',
        name: 'Statistics',
        component: () => import('@/views/Statistics.vue'),
        meta: { title: '数据统计' },
      },
      {
        path: 'roles',
        name: 'Roles',
        component: () => import('@/views/Roles.vue'),
        meta: { title: '角色权限' },
      },
      {
        path: 'ai-assistant',
        name: 'AIAssistant',
        component: () => import('@/views/AIAssistant.vue'),
        meta: { title: 'AI 助手' },
      },
      {
        path: 'sensitive-words',
        name: 'SensitiveWords',
        component: () => import('@/views/SensitiveWords.vue'),
        meta: { title: '敏感词管理' },
      },
      {
        path: 'operation-logs',
        name: 'OperationLogs',
        component: () => import('@/views/OperationLogs.vue'),
        meta: { title: '操作日志' },
      },
      {
        path: 'system-config',
        name: 'SystemConfig',
        component: () => import('@/views/SystemConfig.vue'),
        meta: { title: '系统配置' },
      },
      {
        path: 'version-management',
        name: 'VersionManagement',
        component: () => import('@/views/VersionManagement.vue'),
        meta: { title: '版本管理' },
      },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to, _from, next) => {
  const requiresAuth = to.meta.requiresAuth !== false
  const token = localStorage.getItem('token')

  if (requiresAuth && !token) {
    next('/login')
  } else if (to.path === '/login' && token) {
    next('/')
  } else {
    next()
  }
})

export default router

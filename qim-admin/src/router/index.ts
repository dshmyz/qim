import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'Home',
    component: () => import('@/views/Home.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/admin',
    component: () => import('@/layouts/AdminLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'Dashboard',
        component: () => import('@/views/Dashboard/index.vue'),
        meta: { title: '仪表盘' },
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('@/views/UserManagement/index.vue'),
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
        component: () => import('@/views/GroupManagement/index.vue'),
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
        path: 'message-search',
        name: 'MessageSearch',
        component: () => import('@/views/MessageSearch/index.vue'),
        meta: { title: '消息搜索' },
      },
      {
        path: 'file-storage',
        name: 'FileStorage',
        component: () => import('@/views/FileManagement/Storage.vue'),
        meta: { title: '文件存储管理' },
      },
      {
        path: 'server-monitor',
        name: 'ServerMonitor',
        component: () => import('@/views/SystemMonitor/Server.vue'),
        meta: { title: '服务器监控' },
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
        component: () => import('@/views/RoleManagement/index.vue'),
        meta: { title: '角色权限' },
      },
      {
        path: 'ai-assistant',
        name: 'AIAssistant',
        component: () => import('@/views/AIAssistant.vue'),
        meta: { title: 'AI 助手' },
      },
      {
        path: 'ai-ops',
        name: 'AIOps',
        component: () => import('@/views/AIOps.vue'),
        meta: { title: 'AI 运维面板' },
      },
      {
        path: 'ai-config',
        name: 'AIConfig',
        component: () => import('@/views/AIConfig/Providers.vue'),
        meta: { title: 'AI 模型配置' },
      },
      {
        path: 'approvals',
        name: 'Approvals',
        component: () => import('@/views/UnifiedApprovalPanel.vue'),
        meta: { title: '审批管理' },
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
  {
    path: '/403',
    name: 'Forbidden',
    component: () => import('@/views/Forbidden.vue'),
    meta: { requiresAuth: false },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router

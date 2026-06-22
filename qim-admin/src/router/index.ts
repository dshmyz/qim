import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

// 路由 meta.roles 约定：
// - 不配置 meta.roles：所有登录用户可访问（如仪表盘）
// - 配置 meta.roles：用户需拥有其中任一角色才能访问
// 角色码由后端定义，当前有 system_admin / system_publisher 等
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
        component: () => import('@/views/Dashboard/index.vue'),
        meta: { title: '仪表盘' },
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('@/views/UserManagement/index.vue'),
        meta: { title: '用户管理', roles: ['system_admin'] },
      },
      {
        path: 'organization',
        name: 'Organization',
        component: () => import('@/views/Organization.vue'),
        meta: { title: '组织架构', roles: ['system_admin'] },
      },
      {
        path: 'groups',
        name: 'Groups',
        component: () => import('@/views/GroupManagement/index.vue'),
        meta: { title: '群组管理', roles: ['system_admin'] },
      },
      {
        path: 'conversations',
        name: 'Conversations',
        component: () => import('@/views/Conversations.vue'),
        meta: { title: '会话管理', roles: ['system_admin'] },
      },
      {
        path: 'channels',
        name: 'Channels',
        component: () => import('@/views/Channels.vue'),
        meta: { title: '频道管理', roles: ['system_admin'] },
      },
      {
        path: 'apps',
        name: 'Apps',
        component: () => import('@/views/Apps.vue'),
        meta: { title: '应用管理', roles: ['system_admin'] },
      },
      {
        path: 'mini-apps',
        name: 'MiniApps',
        component: () => import('@/views/MiniApps.vue'),
        meta: { title: '小程序管理', roles: ['system_admin'] },
      },
      {
        path: 'messages',
        name: 'Messages',
        component: () => import('@/views/SystemMessages.vue'),
        meta: { title: '系统消息', roles: ['system_admin', 'system_publisher'] },
      },
      {
        path: 'message-search',
        name: 'MessageSearch',
        component: () => import('@/views/MessageSearch/index.vue'),
        meta: { title: '消息搜索', roles: ['system_admin'] },
      },
      {
        path: 'file-storage',
        name: 'FileStorage',
        component: () => import('@/views/FileManagement/Storage.vue'),
        meta: { title: '文件存储管理', roles: ['system_admin'] },
      },
      {
        path: 'server-monitor',
        name: 'ServerMonitor',
        component: () => import('@/views/SystemMonitor/Server.vue'),
        meta: { title: '服务器监控', roles: ['system_admin'] },
      },
      {
        path: 'notifications',
        name: 'Notifications',
        component: () => import('@/views/Notifications.vue'),
        meta: { title: '通知管理', roles: ['system_admin'] },
      },
      {
        path: 'blacklist',
        name: 'Blacklist',
        component: () => import('@/views/Blacklist.vue'),
        meta: { title: '黑名单管理', roles: ['system_admin'] },
      },
      {
        path: 'statistics',
        name: 'Statistics',
        component: () => import('@/views/Statistics.vue'),
        meta: { title: '数据统计', roles: ['system_admin'] },
      },
      {
        path: 'roles',
        name: 'Roles',
        component: () => import('@/views/RoleManagement/index.vue'),
        meta: { title: '角色权限', roles: ['system_admin'] },
      },
      {
        path: 'ai-assistant',
        name: 'AIAssistant',
        component: () => import('@/views/AIAssistant.vue'),
        meta: { title: 'AI 助手', roles: ['system_admin', 'system_publisher'] },
      },
      {
        path: 'ai-ops',
        name: 'AIOps',
        component: () => import('@/views/AIOps.vue'),
        meta: { title: 'AI 运维面板', roles: ['system_admin'] },
      },
      {
        path: 'ai-config',
        name: 'AIConfig',
        component: () => import('@/views/AIConfig/Providers.vue'),
        meta: { title: 'AI 模型配置', roles: ['system_admin'] },
      },
      {
        path: 'mcp-tools',
        name: 'MCPTools',
        component: () => import('@/views/MCPTools.vue'),
        meta: { title: 'MCP 工具管理', roles: ['system_admin'] },
      },
      {
        path: 'knowledge-graph',
        name: 'KnowledgeGraph',
        component: () => import('@/views/KnowledgeGraph.vue'),
        meta: { title: '知识图谱', roles: ['system_admin'] },
      },
      {
        path: 'vector-data',
        name: 'VectorData',
        component: () => import('@/views/VectorData.vue'),
        meta: { title: '向量数据', roles: ['system_admin'] },
      },
      {
        path: 'approvals',
        name: 'Approvals',
        component: () => import('@/views/UnifiedApprovalPanel.vue'),
        meta: { title: '审批管理', roles: ['system_admin'] },
      },
      {
        path: 'auth-config',
        name: 'AuthConfig',
        component: () => import('@/views/AuthConfig/index.vue'),
        meta: { title: '认证配置', roles: ['system_admin'] },
      },
      {
        path: 'org-sync',
        name: 'OrgSync',
        component: () => import('@/views/OrgSync/index.vue'),
        meta: { title: '组织架构同步', roles: ['system_admin'] },
      },
      {
        path: 'sensitive-words',
        name: 'SensitiveWords',
        component: () => import('@/views/SensitiveWords.vue'),
        meta: { title: '敏感词管理', roles: ['system_admin'] },
      },
      {
        path: 'operation-logs',
        name: 'OperationLogs',
        component: () => import('@/views/OperationLogs.vue'),
        meta: { title: '操作日志', roles: ['system_admin'] },
      },
      {
        path: 'feedbacks',
        name: 'Feedbacks',
        component: () => import('@/views/FeedbackManagement.vue'),
        meta: { title: '意见反馈', roles: ['system_admin'] },
      },
      {
        path: 'crash-logs',
        name: 'CrashLogs',
        component: () => import('@/views/CrashLogs/index.vue'),
        meta: { title: '崩溃日志', roles: ['system_admin'] },
      },
      {
        path: 'system-config',
        name: 'SystemConfig',
        component: () => import('@/views/SystemConfig.vue'),
        meta: { title: '系统配置', roles: ['system_admin'] },
      },
      {
        path: 'version-management',
        name: 'VersionManagement',
        component: () => import('@/views/ClientManagement/Versions.vue'),
        meta: { title: '版本管理', roles: ['system_admin'] },
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
  history: createWebHistory('/admin/'),
  routes,
})

export default router

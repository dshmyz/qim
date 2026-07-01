import type { Component } from 'vue'
import type { RouteRecordRaw } from 'vue-router'
import {
  BellFilled,
  Cellphone,
  ChatDotRound,
  ChatLineSquare,
  Checked,
  CircleCloseFilled,
  Connection,
  Cpu,
  DataAnalysis,
  Document,
  Folder,
  Grid,
  HomeFilled,
  Key,
  Lock,
  Message,
  Monitor,
  School,
  Setting,
  Tools,
  TrendCharts,
  Upload,
  User,
  UserFilled,
  Warning,
  WarningFilled,
  Search,
} from '@element-plus/icons-vue'

export interface AdminModuleItem {
  path: string
  name: string
  title: string
  icon: Component
  component: RouteRecordRaw['component']
  roles?: string[]
  permission?: string
  role?: string | string[]
  hidden?: boolean
}

export interface AdminModuleGroup {
  key: string
  title: string
  icon: Component
  items: AdminModuleItem[]
}

export const adminModuleGroups: AdminModuleGroup[] = [
  {
    key: 'overview',
    title: '概览',
    icon: DataAnalysis,
    items: [
      {
        path: '',
        name: 'Dashboard',
        title: '仪表盘',
        icon: HomeFilled,
        component: () => import('@/views/Dashboard/index.vue'),
      },
      {
        path: 'statistics',
        name: 'Statistics',
        title: '数据统计',
        icon: TrendCharts,
        component: () => import('@/views/Statistics.vue'),
        roles: ['system_admin'],
        role: 'system_admin',
      },
    ],
  },
  {
    key: 'identity',
    title: '身份与组织',
    icon: User,
    items: [
      {
        path: 'users',
        name: 'Users',
        title: '用户管理',
        icon: User,
        component: () => import('@/views/UserManagement/index.vue'),
        roles: ['system_admin'],
        permission: 'user:read',
      },
      {
        path: 'organization',
        name: 'Organization',
        title: '组织架构',
        icon: School,
        component: () => import('@/views/Organization.vue'),
        roles: ['system_admin'],
        permission: 'organization:read',
      },
      {
        path: 'roles',
        name: 'Roles',
        title: '角色权限',
        icon: Key,
        component: () => import('@/views/RoleManagement/index.vue'),
        roles: ['system_admin'],
        permission: 'role:read',
      },
      {
        path: 'auth-config',
        name: 'AuthConfig',
        title: '认证配置',
        icon: Key,
        component: () => import('@/views/AuthConfig/index.vue'),
        roles: ['system_admin'],
        permission: 'auth:read',
      },
      {
        path: 'org-sync',
        name: 'OrgSync',
        title: '组织架构同步',
        icon: Connection,
        component: () => import('@/views/OrgSync/index.vue'),
        roles: ['system_admin'],
        permission: 'org:read',
      },
    ],
  },
  {
    key: 'communication',
    title: '沟通治理',
    icon: ChatDotRound,
    items: [
      {
        path: 'groups',
        name: 'Groups',
        title: '群组管理',
        icon: UserFilled,
        component: () => import('@/views/GroupManagement/index.vue'),
        roles: ['system_admin'],
        permission: 'group:read',
      },
      {
        path: 'conversations',
        name: 'Conversations',
        title: '会话管理',
        icon: ChatLineSquare,
        component: () => import('@/views/Conversations.vue'),
        roles: ['system_admin'],
        permission: 'conversation:read',
      },
      {
        path: 'channels',
        name: 'Channels',
        title: '频道管理',
        icon: Connection,
        component: () => import('@/views/Channels.vue'),
        roles: ['system_admin'],
        permission: 'channel:read',
      },
      {
        path: 'messages',
        name: 'Messages',
        title: '系统消息',
        icon: ChatDotRound,
        component: () => import('@/views/SystemMessages.vue'),
        roles: ['system_admin', 'system_publisher'],
        permission: 'message:read',
      },
      {
        path: 'message-search',
        name: 'MessageSearch',
        title: '消息搜索',
        icon: Search,
        component: () => import('@/views/MessageSearch/index.vue'),
        roles: ['system_admin'],
        permission: 'message:read',
      },
      {
        path: 'notifications',
        name: 'Notifications',
        title: '通知管理',
        icon: BellFilled,
        component: () => import('@/views/Notifications.vue'),
        roles: ['system_admin'],
        permission: 'notification:read',
      },
    ],
  },
  {
    key: 'apps',
    title: '应用与集成',
    icon: Grid,
    items: [
      {
        path: 'apps',
        name: 'Apps',
        title: '应用管理',
        icon: Monitor,
        component: () => import('@/views/Apps.vue'),
        roles: ['system_admin'],
        permission: 'app:read',
      },
      {
        path: 'mini-apps',
        name: 'MiniApps',
        title: '小程序管理',
        icon: Cellphone,
        component: () => import('@/views/MiniApps.vue'),
        roles: ['system_admin'],
        permission: 'miniapp:read',
      },
      {
        path: 'mcp-tools',
        name: 'MCPTools',
        title: 'MCP 工具管理',
        icon: Tools,
        component: () => import('@/views/MCPTools.vue'),
        roles: ['system_admin'],
        permission: 'ai:read',
      },
    ],
  },
  {
    key: 'ai',
    title: 'AI 与知识',
    icon: Cpu,
    items: [
      {
        path: 'ai-assistant',
        name: 'AIAssistant',
        title: 'AI 助手',
        icon: Cpu,
        component: () => import('@/views/AIAssistant.vue'),
        roles: ['system_admin', 'system_publisher'],
        permission: 'ai:read',
      },
      {
        path: 'ai-ops',
        name: 'AIOps',
        title: 'AI 运维面板',
        icon: Monitor,
        component: () => import('@/views/AIOps.vue'),
        roles: ['system_admin'],
        permission: 'ai:read',
      },
      {
        path: 'ai-config',
        name: 'AIConfig',
        title: 'AI 模型配置',
        icon: Setting,
        component: () => import('@/views/AIConfig/Providers.vue'),
        roles: ['system_admin'],
        permission: 'ai:read',
      },
      {
        path: 'knowledge-graph',
        name: 'KnowledgeGraph',
        title: '知识图谱',
        icon: Connection,
        component: () => import('@/views/KnowledgeGraph.vue'),
        roles: ['system_admin'],
        permission: 'ai:read',
      },
      {
        path: 'vector-data',
        name: 'VectorData',
        title: '向量数据',
        icon: DataAnalysis,
        component: () => import('@/views/VectorData.vue'),
        roles: ['system_admin'],
        permission: 'ai:read',
      },
    ],
  },
  {
    key: 'security',
    title: '安全审计',
    icon: Lock,
    items: [
      {
        path: 'approvals',
        name: 'Approvals',
        title: '审批管理',
        icon: Checked,
        component: () => import('@/views/UnifiedApprovalPanel.vue'),
        roles: ['system_admin'],
        role: 'system_admin',
      },
      {
        path: 'blacklist',
        name: 'Blacklist',
        title: '黑名单管理',
        icon: CircleCloseFilled,
        component: () => import('@/views/Blacklist.vue'),
        roles: ['system_admin'],
        permission: 'blacklist:read',
      },
      {
        path: 'sensitive-words',
        name: 'SensitiveWords',
        title: '敏感词管理',
        icon: Warning,
        component: () => import('@/views/SensitiveWords.vue'),
        roles: ['system_admin'],
        permission: 'sensitive:read',
      },
      {
        path: 'operation-logs',
        name: 'OperationLogs',
        title: '操作日志',
        icon: Document,
        component: () => import('@/views/OperationLogs.vue'),
        roles: ['system_admin'],
        permission: 'log:read',
      },
      {
        path: 'feedbacks',
        name: 'Feedbacks',
        title: '意见反馈',
        icon: Message,
        component: () => import('@/views/FeedbackManagement.vue'),
        roles: ['system_admin'],
        permission: 'feedback:read',
      },
      {
        path: 'crash-logs',
        name: 'CrashLogs',
        title: '崩溃日志',
        icon: WarningFilled,
        component: () => import('@/views/CrashLogs/index.vue'),
        roles: ['system_admin'],
        permission: 'log:read',
      },
    ],
  },
  {
    key: 'system',
    title: '系统运维',
    icon: Setting,
    items: [
      {
        path: 'system-config',
        name: 'SystemConfig',
        title: '系统配置',
        icon: Tools,
        component: () => import('@/views/SystemConfig.vue'),
        roles: ['system_admin'],
        permission: 'config:read',
      },
      {
        path: 'version-management',
        name: 'VersionManagement',
        title: '版本管理',
        icon: Upload,
        component: () => import('@/views/ClientManagement/Versions.vue'),
        roles: ['system_admin'],
        permission: 'version:read',
      },
      {
        path: 'file-storage',
        name: 'FileStorage',
        title: '文件存储管理',
        icon: Folder,
        component: () => import('@/views/FileManagement/Storage.vue'),
        roles: ['system_admin'],
        permission: 'file:read',
      },
      {
        path: 'server-monitor',
        name: 'ServerMonitor',
        title: '服务器监控',
        icon: Monitor,
        component: () => import('@/views/SystemMonitor/Server.vue'),
        roles: ['system_admin'],
        permission: 'monitor:read',
      },
    ],
  },
]

export const adminModules = adminModuleGroups.flatMap(group => group.items)

export const adminRouteRecords: RouteRecordRaw[] = adminModules.map(module => ({
  path: module.path,
  name: module.name,
  component: module.component,
  meta: {
    title: module.title,
    roles: module.roles,
  },
} as RouteRecordRaw))

export const sidebarModuleGroups = adminModuleGroups
  .map(group => ({
    ...group,
    items: group.items
      .filter(item => !item.hidden)
      .map(item => ({
        ...item,
        path: item.path ? `/${item.path}` : '/',
      })),
  }))
  .filter(group => group.items.length > 0)

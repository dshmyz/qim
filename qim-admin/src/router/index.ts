import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { adminRouteRecords } from '@/config/adminModules'

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
    children: adminRouteRecords,
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

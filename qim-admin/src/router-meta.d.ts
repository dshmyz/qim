// 扩展 vue-router 的 RouteMeta，支持路由级角色配置
// 单独成文件避免 env.d.ts 因 import 变成 module 导致其 ambient 声明失效
import 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    title?: string
    roles?: string[]
  }
}

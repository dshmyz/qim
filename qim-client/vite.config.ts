import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'
import pkg from './package.json'

export default defineConfig(({ mode }) => {
  const isProd = mode === 'production'
  const extra = (pkg as any).build?.extraMetadata || {}

  return {
    define: {
      __APP_NAME__: JSON.stringify(pkg.name),
      __APP_VERSION__: JSON.stringify(pkg.version),
      __APP_PRODUCT_NAME_CN__: JSON.stringify(extra.productNameCN),
      __APP_COPYRIGHT_YEAR__: JSON.stringify(extra.copyrightYear),
    },
    base: './',
    plugins: [vue()],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src')
      }
    },
    server: {
      port: 3000,
      host: true
    },
    build: {
      chunkSizeWarningLimit: 500,
      minify: isProd ? 'terser' : false,
      terserOptions: isProd ? {
        compress: {
          drop_debugger: true,
        }
      } : undefined,
      rollupOptions: {
        output: {
          manualChunks(id) {
            // Vue 核心框架
            if (id.includes('node_modules/vue') || id.includes('node_modules/@vue')) {
              return 'vue'
            }
            // Pinia 状态管理
            if (id.includes('node_modules/pinia')) {
              return 'pinia'
            }
            // axios
            if (id.includes('node_modules/axios')) {
              return 'axios'
            }
            // markdown 解析 (marked) — 较大，独立分包
            if (id.includes('node_modules/marked')) {
              return 'marked'
            }
            // PDF 相关 — 最大依赖，完全按需
            if (id.includes('node_modules/pdfjs-dist')) {
              return 'pdfjs'
            }
            // 其他 node_modules 统一 vendor
            if (id.includes('node_modules')) {
              return 'vendor'
            }
            // 聊天主模块
            if (id.includes('/components/chat/')) {
              return 'chat'
            }
            // AI 相关
            if (id.includes('/components/apps/ai/') || id.includes('/components/apps/AIAssistantApp')) {
              return 'ai'
            }
            // 日历应用
            if (id.includes('/components/apps/CalendarApp')) {
              return 'calendar'
            }
            // 文件管理
            if (id.includes('/components/apps/FileManagementApp')) {
              return 'file'
            }
            // 便签 + 笔记
            if (id.includes('/components/apps/StickyNotesApp') || id.includes('/components/apps/NotesApp')) {
              return 'notes'
            }
            // 任务管理
            if (id.includes('/components/apps/task/')) {
              return 'task'
            }
            // 设置相关
            if (id.includes('/components/settings/') || id.includes('/components/avatar/')) {
              return 'settings'
            }
            // 群聊相关
            if (id.includes('/components/shared/Group') || id.includes('/components/modals/CreateGroup') || id.includes('/components/modals/GroupModals')) {
              return 'group'
            }
          }
        }
      }
    }
  }
})

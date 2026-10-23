import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'
import pkg from './package.json'

export default defineConfig(({ mode }) => {
  const isProd = mode === 'production'
  const extra = (pkg as any).build?.extraMetadata || {}
  const productName = (pkg as any).build?.productName || pkg.name

  return {
    define: {
      __APP_NAME__: JSON.stringify(pkg.name),
      __APP_VERSION__: JSON.stringify(pkg.version),
      __APP_PRODUCT_NAME__: JSON.stringify(productName),
      __APP_PRODUCT_NAME_CN__: JSON.stringify(extra.productNameCN),
      __APP_COPYRIGHT_YEAR__: JSON.stringify(extra.copyrightYear),
      __APP_RELEASE_DATE__: JSON.stringify(extra.releaseDate),
      __APP_CONTACT_EMAIL__: JSON.stringify(extra.contactEmail),
    },
    base: './',
    plugins: [
      vue(),
      // 页面 <title> 中的产品名与 package.json build.productName 同源，改名不用碰 HTML
      {
        transformIndexHtml: {
          order: 'pre',
          handler: (html) => html.replace(
            /<title>.*?<\/title>/,
            // 函数替换：产品名含 $&/$' 等 replace 特殊序列时不会被误解释
            () => `<title>${productName} - 即时通讯组件库</title>`
          ),
        },
      },
    ],
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
            // ECharts 关系图引擎（echarts + zrender）— 完全按需，只在点开图谱页时才拉取。
            // 命名成独立 chunk：echarts 仅被 AvatarGraph 动态 import，渲染时才加载，不进首屏。
            if (id.includes('node_modules/echarts') || id.includes('node_modules/zrender')) {
              return 'echarts'
            }
            // 其他 node_modules 统一 vendor
            if (id.includes('node_modules')) {
              return 'vendor'
            }
            // 以下业务模块互相引用密集（chat↔group↔settings↔ai↔calendar↔file），
            // 且入口对它们全部是静态引入（首屏必然一起加载）。拆开会产生 chunk 级
            // 循环依赖：环内任一模块顶层跨 chunk 取值（如 logger 顶层 new 时调
            // getProductName、SFC 的 _export_sfc 落在别的 chunk）就会在打包后报
            // "Cannot access 't' before initialization"，dev 不分包所以只在包里炸。
            // 因此合并为单一 main chunk，根治 12 条 chunk 环；notes/task 等
            // 真正按需动态加载的模块仍保持独立分包。
            // 聊天主模块 + 群聊/设置/AI/日历/文件（互相成环的六个合并）
            if (
              id.includes('/components/chat/') ||
              id.includes('/components/apps/ai/') ||
              id.includes('/components/apps/AIAssistantApp') ||
              id.includes('/components/apps/CalendarApp') ||
              id.includes('/components/apps/FileManagementApp') ||
              id.includes('/components/settings/') ||
              id.includes('/components/avatar/') ||
              id.includes('/components/shared/Group') ||
              id.includes('/components/modals/CreateGroup') ||
              id.includes('/components/modals/GroupModals')
            ) {
              return 'main'
            }
            // 便签 + 笔记（入口动态引入，保持独立按需加载）
            if (id.includes('/components/apps/StickyNotesApp') || id.includes('/components/apps/NotesApp')) {
              return 'notes'
            }
            // 任务管理（入口动态引入，保持独立按需加载）
            if (id.includes('/components/apps/task/')) {
              return 'task'
            }
          }
        }
      }
    }
  }
})

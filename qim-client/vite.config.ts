import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

export default defineConfig(({ mode }) => {
  const isProd = mode === 'production'

  return {
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
          drop_console: true,
          drop_debugger: true,
          passes: 2,
          pure_funcs: ['console.log', 'console.debug', 'console.info', 'console.warn']
        }
      } : undefined,
      rollupOptions: {
        output: {
        manualChunks: {
          // Vue 核心框架
          vue: ['vue'],
          // 聊天主模块
          chat: [
            './src/components/chat/ChatWindow.vue',
            './src/components/chat/ChatBody.vue',
            './src/components/chat/ChatHeader.vue',
            './src/components/chat/ChatInputArea.vue',
          ],
          // AI 相关功能
          ai: [
            './src/components/apps/AIAssistantApp.vue',
            './src/components/apps/ai/ChatCenter.vue',
          ],
          // 便签应用
          sticky: ['./src/components/apps/StickyNotesApp.vue'],
          // 笔记应用
          notes: ['./src/components/apps/NotesApp.vue'],
          // 任务管理
          task: ['./src/components/apps/task/TaskManagementApp.vue'],
          // 日历应用
          calendar: ['./src/components/apps/CalendarApp.vue'],
          // 文件管理
          file: ['./src/components/apps/FileManagementApp.vue'],
          // 短链接
          shortlink: ['./src/components/apps/ShortLinkManager.vue'],
          // 设置相关
          settings: [
            './src/components/settings/SettingsPanel.vue',
            './src/components/avatar/AvatarSettingsPanel.vue',
          ],
          // 群聊相关
          group: [
            './src/components/shared/GroupDetail.vue',
            './src/components/modals/CreateGroupModal.vue',
            './src/components/modals/GroupModals.vue',
          ],
        }
      }
    }
  }
})

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig(({ mode }) => {
  const isProd = mode === 'production'

  return {
    plugins: [vue()],
    resolve: {
      alias: {
        '@': resolve(__dirname, 'src'),
      },
    },
    server: {
      port: 3008,
      proxy: {
        '/api': {
          target: 'http://localhost:8080',
          changeOrigin: true,
        },
      },
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
          manualChunks(id) {
            // Vue 生态核心
            if (id.includes('node_modules/vue') || id.includes('node_modules/@vue') ||
                id.includes('node_modules/vue-router') || id.includes('node_modules/pinia')) {
              return 'vue'
            }
            // Element Plus — 最大的 UI 依赖，独立分包
            if (id.includes('node_modules/element-plus') || id.includes('node_modules/@element-plus')) {
              return 'element-plus'
            }
            // ECharts — 图表库，独立分包
            if (id.includes('node_modules/echarts') || id.includes('node_modules/zrender')) {
              return 'echarts'
            }
            // axios
            if (id.includes('node_modules/axios')) {
              return 'axios'
            }
            // 其余依赖
            if (id.includes('node_modules')) {
              return 'vendor'
            }
          },
        },
      },
    },
  }
})

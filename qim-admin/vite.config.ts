import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'
import pkg from './package.json'

export default defineConfig(({ mode }) => {
  const isProd = mode === 'production'
  const extra = (pkg as any).build?.extraMetadata || {}

  return {
    define: {
      __APP_PRODUCT_NAME__: JSON.stringify(extra.productName || pkg.name),
      __APP_VERSION__: JSON.stringify(pkg.version),
      __APP_PRODUCT_NAME_CN__: JSON.stringify(extra.productNameCN || '青雀'),
      __APP_COPYRIGHT_YEAR__: JSON.stringify(extra.copyrightYear || '2026'),
    },
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
    base: '/admin/',
    build: {
      // 直接输出到 Go embed 源目录，省去手动拷贝，避免 embed 进二进制的产物与源码脱节
      outDir: '../qim-server/web/webroot/admin',
      emptyOutDir: true,
      chunkSizeWarningLimit: 500,
      minify: isProd ? 'esbuild' : false,
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

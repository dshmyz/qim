import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'
import pkg from './package.json'

export default defineConfig(({ mode }) => {
  const isProd = mode === 'production'
  const extra = (pkg as any).build?.extraMetadata || {}

  return {
    define: {
      __APP_NAME__: JSON.stringify(extra.productName || 'QIM'),
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
    base: '/',
    build: {
      outDir: 'dist-landing',
      minify: isProd ? 'esbuild' : false,
      rollupOptions: {
        input: {
          main: resolve(__dirname, 'landing.html'),
        },
      },
    },
  }
})

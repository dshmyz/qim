/// <reference types="vitest" />
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

import pkg from './package.json'

const extra = (pkg as any).build?.extraMetadata || {}

export default defineConfig({
  plugins: [vue()],
  define: {
    __APP_NAME__: JSON.stringify(pkg.name),
    __APP_VERSION__: JSON.stringify(pkg.version),
    __APP_PRODUCT_NAME__: JSON.stringify((pkg as any).build?.productName),
    __APP_PRODUCT_NAME_CN__: JSON.stringify(extra.productNameCN),
    __APP_COPYRIGHT_YEAR__: JSON.stringify(extra.copyrightYear),
    __APP_RELEASE_DATE__: JSON.stringify(extra.releaseDate),
    __APP_CONTACT_EMAIL__: JSON.stringify(extra.contactEmail),
  },
  test: {
    environment: 'happy-dom',
    globals: true,
    setupFiles: ['./tests/setup.ts'],
    include: ['tests/unit/**/*.test.ts', 'tests/unit/**/*.spec.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'node_modules/',
        'tests/',
        '**/*.d.ts',
        '**/*.config.*',
        'electron/',
      ],
    },
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
})

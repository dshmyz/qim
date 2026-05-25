import { defineConfig } from '@rsbuild/core';
import { pluginReact } from '@rsbuild/plugin-react';
import { pluginLess } from '@rsbuild/plugin-less';

export default defineConfig({
  plugins: [pluginReact(), pluginLess()],
  source: {
    entry: {
      electron: './electron/index.tsx',
    },
  },
  output: {
    assetPrefix: './',
    distPath: {
      root: 'dist',
    },
    cleanDistPath: false,
  },
  html: {
    template: './electron.html',
  },
});

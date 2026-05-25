import { defineConfig } from '@rslib/core';
import { pluginLess } from '@rsbuild/plugin-less';

export default defineConfig({
  plugins: [pluginLess()],
  source: {
    entry: {
      Screenshots: './Screenshots/exports.ts',
    },
  },
  lib: [
    {
      format: 'esm',
      dts: true,
    },
  ],
  output: {
    target: 'web',
    cleanDistPath: false,
    distPath: {
      root: 'lib',
    },
  },
});

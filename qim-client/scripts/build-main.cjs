// 把 electron/main.js 及其 ESM 依赖打成单个 CJS 文件 electron/main.bundle.cjs，
// 供所有平台的安装包使用（package.json 的 build.extraMetadata.main 指向 electron/main.cjs，
// 由它 require 本产物）。源码 ESM 仅在开发模式（electron:dev，Electron ≥28）直接运行。
//
// 为什么统一走 bundle：
// - 同一份主进程代码在所有平台上是同一个产物，避免"win7 跑 bundle、其他平台跑源码"的执行路径分叉；
// - Electron 22（win7）主进程不支持 ESM，bundle 是唯一通行的方式，统一后 win7 不再是特例。
//
// external 只留 electron 与 electron-updater：
// - electron 运行时由运行体注入，npm 包里的 electron/index.js 只是路径字符串；
// - electron-updater 是运行时依赖（会进 asar），保持原目录布局以保证 app-update.yml 路径解析不变。
const path = require('path')
const root = path.resolve(__dirname, '..')

let esbuild
try {
  // esbuild 未提升到顶层 node_modules（嵌套在 vite 之下），两种位置都尝试
  esbuild = require('esbuild')
} catch {
  try {
    esbuild = require(path.join(root, 'node_modules', 'vite', 'node_modules', 'esbuild'))
  } catch {
    console.error('未找到 esbuild。请先执行: npm install')
    process.exit(1)
  }
}

const result = esbuild.buildSync({
  entryPoints: [path.join(root, 'electron', 'main.js')],
  bundle: true,
  platform: 'node',
  format: 'cjs',
  target: 'node16',
  external: ['electron', 'electron-updater'],
  outfile: path.join(root, 'electron', 'main.bundle.cjs'),
  sourcemap: false,
  logLevel: 'warning',
})

if (result.errors.length > 0) {
  process.exit(1)
}

console.log('主进程 bundle 生成完成: electron/main.bundle.cjs')

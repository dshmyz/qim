// 所有平台安装包的统一入口（开发模式不走这里，直接运行 ESM 的 electron/main.js）。
// electron-builder 通过各构建配置的 extraMetadata.main 选中本文件，
// 由它加载 scripts/build-main.cjs 产出的 CJS bundle。
// 启动失败写 %TEMP%\qim-main-error.log（与 main.js 的日志约定一致），便于离线定位。
try {
  require('./main.bundle.cjs')
} catch (error) {
  try {
    const os = require('os')
    const fs = require('fs')
    const path = require('path')
    fs.appendFileSync(
      path.join(os.tmpdir(), 'qim-main-error.log'),
      `[${new Date().toISOString()}] 主进程入口加载失败: ${(error && error.stack) || error}\n`
    )
  } catch { /* 日志写失败忽略 */ }
  console.error('Failed to load main bundle:', error)
  throw error
}

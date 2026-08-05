import { safeStorage } from 'electron'

// 「记住密码」不再明文落 localStorage：由主进程用系统安全存储（Keychain/DPAPI/libsecret）
// 加解密后仅落密文（base64）。加密不可用/失败/损坏时返回 ''，由渲染层优雅回退不落盘、不回填。
// 与 auto-update 同样依赖主进程 Electron 能力，但业务域无关，故独立成模块由 main.js 直接注册。
export function registerSecurePasswordIpc(ipcMain) {
  ipcMain.handle('password:encrypt', (_event, plaintext) => {
    try {
      if (!safeStorage.isEncryptionAvailable() || !plaintext) return ''
      return safeStorage.encryptString(plaintext).toString('base64')
    } catch (error) {
      console.error('[safe-password] 加密失败:', error)
      return ''
    }
  })

  ipcMain.handle('password:decrypt', (_event, base64) => {
    try {
      if (!base64 || !safeStorage.isEncryptionAvailable()) return ''
      return safeStorage.decryptString(Buffer.from(base64, 'base64'))
    } catch (error) {
      // 解密失败（密钥库锁定/换用户/损坏）不抛错，返回 '' 让渲染层留空输入框
      console.error('[safe-password] 解密失败:', error)
      return ''
    }
  })
}

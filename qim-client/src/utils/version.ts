import packageJson from '../../package.json'

// 获取当前版本
export const getCurrentVersion = (): string => {
  return packageJson.version
}

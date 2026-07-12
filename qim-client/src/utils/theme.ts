/**
 * app 主题工具函数。
 *
 * 项目通过 document.documentElement.dataset.theme 存储当前主题
 * （如 modern-light / elegant-dark / ocean-blue 等），见 useSettings.ts。
 *
 * 本模块提供与 app 主题系统一致的判断与监听能力，
 * 替代 window.matchMedia('(prefers-color-scheme: dark)')，
 * 后者只反映系统主题，与 app 内主题切换相互独立。
 */

/** 暗色主题 id 列表（与 SettingsPanel.vue 中的 themes 定义保持一致） */
export const DARK_THEMES: readonly string[] = ['elegant-dark']

/**
 * 判断 app 当前主题是否为暗色。
 * 依据：document.documentElement 上的 data-theme 属性是否命中 DARK_THEMES。
 */
export function isAppDarkTheme(): boolean {
  const theme = document.documentElement.dataset.theme
  return theme != null && DARK_THEMES.includes(theme)
}

/**
 * 监听 app 主题变化。当 document.documentElement 的 data-theme 属性变化时触发回调。
 * @returns 取消监听的函数
 */
export function onAppThemeChange(callback: () => void): () => void {
  const observer = new MutationObserver(callback)
  observer.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['data-theme'],
  })
  return () => observer.disconnect()
}

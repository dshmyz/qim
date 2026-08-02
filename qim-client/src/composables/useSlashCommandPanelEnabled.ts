/**
 * 斜杠命令面板全局开关。
 *
 * 控制输入 / 时是否弹出命令列表/搜索面板（含 /task、/note、/quick 等所有斜杠命令）。
 * 数据源：后端 user_settings 表，key=slash_command_panel_enabled，value=boolean。
 * 默认开启（true）；用户在「设置 → 消息设置」里可完全关闭整个斜杠命令面板。
 * 后端请求失败时兜底 true，保证核心功能可用。
 *
 * 拦截点：ChatWindow.vue 的 handleInputChange，在斜杠命令检测入口统一判断，
 * 关闭时所有斜杠命令（含命令列表模式）都不触发。
 */

import { ref } from 'vue'
import { request, type ApiResponse } from './useRequest'

const SETTING_KEY = 'slash_command_panel_enabled'

/** 全局开关状态。默认 true（开启）。 */
const slashCommandPanelEnabled = ref(true)

/** 已加载标志，避免重复加载 */
let loaded = false

interface UserSettingValue<T> {
  value: T
  has_value: boolean
}

/** 从后端加载开关状态。应用启动时调用一次即可。 */
export async function loadSlashCommandPanelEnabled(): Promise<void> {
  if (loaded) return
  loaded = true
  try {
    const res = await request<ApiResponse<UserSettingValue<boolean>>>(
      `/api/v1/user-settings/${SETTING_KEY}`,
      { method: 'GET' }
    )
    const { value, has_value } = res.data ?? { value: null, has_value: false }
    // 未设置时默认 true；已设置时取布尔值（非布尔也兜底 true）
    slashCommandPanelEnabled.value = !has_value || typeof value !== 'boolean' ? true : value
  } catch {
    // 网络异常兜底 true
    slashCommandPanelEnabled.value = true
  }
}

/** 保存开关状态到后端并更新本地状态。 */
export async function setSlashCommandPanelEnabled(value: boolean): Promise<void> {
  // 先更新本地，保证 UI 即时响应
  slashCommandPanelEnabled.value = value
  try {
    await request<ApiResponse<null>>(
      `/api/v1/user-settings/${SETTING_KEY}`,
      { method: 'PUT', body: JSON.stringify({ value }) }
    )
  } catch {
    // 保存失败时回滚本地状态，保持与后端一致
    slashCommandPanelEnabled.value = !value
    throw new Error('保存失败，请重试')
  }
}

export function useSlashCommandPanelEnabled() {
  return {
    slashCommandPanelEnabled,
    loadSlashCommandPanelEnabled,
    setSlashCommandPanelEnabled,
  }
}

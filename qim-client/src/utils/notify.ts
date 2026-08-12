// 统一通知入口：Electron 下走主进程 IPC（带 app 图标，Linux libnotify 稳定渲染），
// 非 Electron（浏览器/dev 回退）用 Web Notification。
// payload 可选：携带深链目标（如 { type, ...target }），桌面通知点击后回传渲染进程做跳转。
// icon 可选：通知图标 URL（浏览器回退时生效；Electron 路径始终使用 app 图标）。
export async function showReminder(title: string, body: string, payload?: any, icon?: string): Promise<void> {
  const api = (window as any).electron?.notifications
  if (api?.show) {
    try {
      await api.show(title, body, payload)
      return
    } catch {
      // 主进程失败则回退 Web Notification
    }
  }
  if ('Notification' in window) {
    const options: NotificationOptions = { body }
    if (icon) options.icon = icon
    if (Notification.permission === 'granted') {
      const n = new Notification(title, options)
      if (payload) {
        n.onclick = () => window.dispatchEvent(new CustomEvent('notification-click', { detail: payload }))
      }
    } else if (Notification.permission !== 'denied') {
      const perm = await Notification.requestPermission()
      if (perm === 'granted') {
        const n = new Notification(title, options)
        if (payload) {
          n.onclick = () => window.dispatchEvent(new CustomEvent('notification-click', { detail: payload }))
        }
      }
    }
  }
}

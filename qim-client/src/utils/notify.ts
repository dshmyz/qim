// 统一通知入口：Electron 下走主进程 IPC（带 app 图标，Linux libnotify 稳定渲染），
// 非 Electron（浏览器/dev 回退）用 Web Notification。
export async function showReminder(title: string, body: string): Promise<void> {
  const api = (window as any).electron?.notifications
  if (api?.show) {
    try {
      await api.show(title, body)
      return
    } catch {
      // 主进程失败则回退 Web Notification
    }
  }
  if ('Notification' in window) {
    if (Notification.permission === 'granted') {
      new Notification(title, { body })
    } else if (Notification.permission !== 'denied') {
      const perm = await Notification.requestPermission()
      if (perm === 'granted') new Notification(title, { body })
    }
  }
}

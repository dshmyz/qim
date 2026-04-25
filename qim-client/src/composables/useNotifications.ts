import { ref, computed } from 'vue'

/**
 * 通知管理 composable
 * 管理通知中心显示状态、未读计数等核心状态
 */
export function useNotifications() {
  // 通知列表
  const notifications = ref<any[]>([])
  
  // 通知中心显示状态
  const showNotificationCenter = ref(false)
  const notificationCenterPosition = ref({ x: 0, y: 0 })

  // 未读通知计数
  const unreadNotificationCount = ref(0)

  // 过滤未读通知
  const filteredNotifications = computed(() => {
    return notifications.value.filter(n => !n.read)
  })

  // 计算通知中心弹窗位置
  const calculatePosition = (event: MouseEvent) => {
    const notificationButton = event.currentTarget as HTMLElement
    if (!notificationButton) return

    const rect = notificationButton.getBoundingClientRect()

    const menuWidth = 380
    const menuHeight = 480
    const windowWidth = window.innerWidth
    const windowHeight = window.innerHeight

    let x = rect.right + 2
    let y = rect.top

    if (x + menuWidth > windowWidth) {
      x = rect.left - menuWidth - 10
    }

    if (y + menuHeight > windowHeight) {
      y = windowHeight - menuHeight - 10
    }

    if (y < 0) {
      y = 10
    }

    notificationCenterPosition.value = { x, y }
  }

  // 打开/关闭通知中心
  const handleNotificationCenter = (event: MouseEvent) => {
    event.stopPropagation()

    // 切换通知中心显示状态
    if (showNotificationCenter.value) {
      showNotificationCenter.value = false
      return
    }

    calculatePosition(event)
    showNotificationCenter.value = true
    unreadNotificationCount.value = 0  // 打开时重置未读计数
  }

  // 关闭通知中心
  const closeNotificationCenter = () => {
    showNotificationCenter.value = false
  }

  // 处理通知点击 - 根据类型路由到不同页面
  const handleNotificationClick = (notification: any) => {
    if (notification.type === 'message' && notification.data?.conversationId) {
      // 切换到消息会话
      console.log('Navigate to conversation:', notification.data.conversationId)
    } else if (notification.type === 'group' && notification.data?.groupId) {
      // 切换到群组
      console.log('Navigate to group:', notification.data.groupId)
    }
  }

  // 处理新通知 - 创建完整通知对象并添加到列表
  const handleNewNotification = (notification: any) => {
    const newNotification = {
      id: notification.id || Date.now().toString(),
      title: notification.title || '新通知',
      content: notification.content || '',
      timestamp: notification.timestamp || Date.now(),
      read: false,
      type: notification.type || 'system',
      data: notification.data || {}
    }
    notifications.value = [newNotification, ...notifications.value]
    unreadNotificationCount.value++
  }

  // 标记所有通知为已读
  const markAllNotificationsAsRead = () => {
    notifications.value.forEach(n => n.read = true)
    unreadNotificationCount.value = 0
  }

  return {
    notifications,
    unreadNotificationCount,
    showNotificationCenter,
    notificationCenterPosition,
    filteredNotifications,
    handleNotificationCenter,
    closeNotificationCenter,
    handleNotificationClick,
    handleNewNotification,
    markAllNotificationsAsRead
  }
}

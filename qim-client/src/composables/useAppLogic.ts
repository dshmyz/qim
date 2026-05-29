import { ref, computed, type Ref } from 'vue'
import { request } from '../composables/useRequest'
import { logger } from '../utils/logger'
import QMessage from '../utils/qmessage'

export interface App {
  id: string
  name: string
  code?: string
  icon: string
  description?: string
  category?: string
}

export function useAppLogic(externalRefs?: {
  selectedAppId?: Ref<string>
  recentApps?: Ref<App[]>
  currentUserApp?: Ref<any>
  showMiniAppList?: Ref<boolean>
  externalCustomApps?: Ref<any[]>
}) {
  const selectedAppId = externalRefs?.selectedAppId || ref<string>('')
  const activeAppTab = ref('categories')
  const recentApps = externalRefs?.recentApps || ref<App[]>([])
  const currentUserApp = externalRefs?.currentUserApp || ref<any>(null)
  const showMiniAppList = externalRefs?.showMiniAppList || ref(false)

  const builtInApps = ref<any[]>([])
  const customApps = externalRefs?.externalCustomApps || ref<any[]>([])
  const userApps = ref<any[]>([])
  const categoryExpanded = ref<Record<string, boolean>>({ '1': true, '2': true })

  const appCategories = computed(() => [
    {
      id: '1', name: '内置应用', expanded: categoryExpanded.value['1'],
      apps: builtInApps.value.map((a: any) => ({ ...a, id: a.code || String(a.id) }))
    },
    {
      id: '2', name: '自定义应用', expanded: categoryExpanded.value['2'],
      apps: customApps.value.map((a: any) => ({ ...a, id: String(a.id) }))
    },
  ])

  const showMsg = (message: string, type: 'success' | 'error' | 'warning' | 'info' = 'info', duration?: number) => {
    if (type === 'success') QMessage.success(message, duration)
    else if (type === 'error') QMessage.error(message, duration)
    else if (type === 'warning') QMessage.warning(message, duration)
    else QMessage.info(message, duration)
  }

  const loadRecentApps = (): App[] => {
    try {
      const stored = localStorage.getItem('recentApps')
      return stored ? JSON.parse(stored) : []
    } catch { return [] }
  }

  const addToRecentApps = (appId: string, appName: string, appIcon: string) => {
    recentApps.value = recentApps.value.filter(a => a.id !== appId)
    recentApps.value.unshift({ id: appId, name: appName, icon: appIcon })
    if (recentApps.value.length > 5) recentApps.value = recentApps.value.slice(0, 5)
    localStorage.setItem('recentApps', JSON.stringify(recentApps.value))
  }

  const loadUserApps = async () => {
    try {
      const response = await request('/api/v1/user-apps')
      if (response.code === 0) userApps.value = response.data || []
    } catch (error) { logger.error('加载用户应用失败:', error) }
  }

  const loadBuiltInApps = async () => {
    try {
      const response = await request('/api/v1/apps/built-in')
      if (response.code === 0) builtInApps.value = Array.isArray(response.data) ? response.data : []
    } catch (error) { logger.error('加载内置应用失败:', error) }
  }
  const openExternalLink = (url: string) => {
    if (typeof window === 'undefined') return
    try {
      if ((window as any).electron?.shell?.openExternal) {
        (window as any).electron.shell.openExternal(url)
      } else {
        window.open(url, '_blank', 'noopener,noreferrer')
      }
    } catch (error) {
      logger.error('打开外部链接失败:', error)
      window.open(url, '_blank', 'noopener,noreferrer')
    }
  }

  const openApp = async (appId: string) => {
    let appName = ''; let appIcon = ''; let appUrl = ''; let appCode = ''; let openType = 'in-app'

    for (const category of appCategories.value) {
      if (!category.apps) continue
      const app = category.apps.find((a: any) => String(a.id) === appId || a.id === appId)
      if (app) {
        appName = app.name; appIcon = app.icon; appUrl = app.url || ''
        appCode = app.code || ''; openType = app.openType || 'in-app'
        break
      }
    }

    if (appName && appIcon) addToRecentApps(appId, appName, appIcon)

    if (appCode) { selectedAppId.value = appCode; return }

    if (appId === 'mini-app') { showMiniAppList.value = true; return }

    if (appUrl) {
      if (openType === 'external') {
        openExternalLink(appUrl)
      } else {
        selectedAppId.value = 'user-app'
        currentUserApp.value = { id: appId, name: appName, icon: appIcon, url: appUrl }
      }
    } else {
      selectedAppId.value = appId
    }
  }

  const openUserApp = (app: any) => {
    if (app.name && app.icon) addToRecentApps(app.id, app.name, app.icon)

    if (app.code) { selectedAppId.value = app.code; return }

    const openType = app.openType || app.open_type || 'in-app'
    if (openType === 'external') {
      if (app.url) openExternalLink(app.url)
    } else {
      selectedAppId.value = 'user-app'
      currentUserApp.value = app
    }
  }

  const openExternalApp = (url: string) => {
    let appName = ''; let appIcon = ''
    for (const category of appCategories.value) {
      if (!category.apps) continue
      const app = category.apps.find((a: any) => String(a.url) === url || a.url === url)
      if (app) { appName = app.name; appIcon = app.icon; break }
    }
    if (appName && appIcon) addToRecentApps(url, appName, appIcon)
    openExternalLink(url)
  }

  const closeApp = () => { selectedAppId.value = '' }

  const handleSwitchApp = (app: App | string) => {
    if (typeof app === 'object' && app.code) {
      selectedAppId.value = app.code
      return
    }
    selectedAppId.value = typeof app === 'string' ? app : app.id
  }

  return {
    selectedAppId, activeAppTab, recentApps, appCategories,
    builtInApps, customApps, userApps, currentUserApp, showMiniAppList,
    loadRecentApps, loadUserApps, loadBuiltInApps,
    openApp, closeApp, handleSwitchApp, openUserApp, openExternalApp,
    showMessage: showMsg
  }
}

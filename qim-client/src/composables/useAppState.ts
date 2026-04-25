import { ref } from 'vue'

/**
 * 应用状态 composable
 * 管理应用的全局状态，包括加载状态、网络错误、侧边栏状态、搜索等
 */
export function useAppState() {
  // 当前活动的侧边栏选项
  const activeOption = ref('recent')
  
  // 选中的应用 ID
  const selectedAppId = ref<string | null>(null)
  
  // 搜索查询
  const searchQuery = ref('')
  
  // 搜索结果
  const searchResults = ref<any[]>([])
  
  // 加载状态
  const isLoading = ref(true)
  
  // 网络错误显示状态
  const showNetworkError = ref(false)
  
  // 网络错误消息
  const networkErrorMsg = ref('网络连接失败，请检查网络后重试')
  
  // 侧边栏收缩状态
  const sidebarCollapsed = ref(false)

  /**
   * 切换侧边栏收缩状态
   */
  const toggleSidebar = () => {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  /**
   * 设置加载状态
   */
  const setLoading = (loading: boolean) => {
    isLoading.value = loading
  }

  /**
   * 设置网络错误状态
   */
  const setNetworkError = (show: boolean, msg?: string) => {
    showNetworkError.value = show
    if (msg) {
      networkErrorMsg.value = msg
    }
  }

  /**
   * 设置搜索查询
   */
  const setSearchQuery = (query: string) => {
    searchQuery.value = query
  }

  /**
   * 设置搜索结果
   */
  const setSearchResults = (results: any[]) => {
    searchResults.value = results
  }

  /**
   * 设置侧边栏选项
   */
  const setActiveOption = (option: string) => {
    activeOption.value = option
  }

  /**
   * 重置侧边栏选项和搜索状态
   */
  const resetState = () => {
    activeOption.value = 'recent'
    searchQuery.value = ''
    searchResults.value = []
    sidebarCollapsed.value = false
  }

  return {
    // 状态
    activeOption,
    selectedAppId,
    searchQuery,
    searchResults,
    isLoading,
    showNetworkError,
    networkErrorMsg,
    sidebarCollapsed,
    
    // 操作方法
    toggleSidebar,
    setLoading,
    setNetworkError,
    setSearchQuery,
    setSearchResults,
    setActiveOption,
    resetState
  }
}

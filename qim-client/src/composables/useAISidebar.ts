import { ref } from 'vue'

/** AI 侧边栏全局状态 */
const showAISidebar = ref(false)

/** AI 悬浮球是否显示（用户可关闭） */
const showFloatingAIBall = ref(
  localStorage.getItem('showFloatingAIBall') !== 'false' // 默认显示
)

/** 当前会话 ID（由 Main.vue 注入） */
const currentConversationId = ref<number | string | null>(null)

export function useAISidebar() {
  const toggleSidebar = () => {
    showAISidebar.value = !showAISidebar.value
  }

  const openSidebar = () => {
    showAISidebar.value = true
  }

  const closeSidebar = () => {
    showAISidebar.value = false
  }

  const toggleFloatingBall = (value?: boolean) => {
    showFloatingAIBall.value = value ?? !showFloatingAIBall.value
    localStorage.setItem('showFloatingAIBall', String(showFloatingAIBall.value))
  }

  const setConversationId = (id: number | string | null) => {
    currentConversationId.value = id
  }

  return {
    showAISidebar,
    showFloatingAIBall,
    currentConversationId,
    toggleSidebar,
    openSidebar,
    closeSidebar,
    toggleFloatingBall,
    setConversationId,
  }
}

import { ref, onUnmounted } from 'vue'
import { avatarAPI } from '../api/avatar'
import type { AvatarLearnStatus } from '../types/avatar'

export function useAvatarPersona() {
  const learnStatus = ref<AvatarLearnStatus>({
    status: 'idle',
    progress: 0,
    messageCount: 0,
    error: null
  })
  const learnedPersona = ref('')
  const loading = ref(false)
  const error = ref('')
  let pollTimer: ReturnType<typeof setInterval> | null = null

  async function triggerLearn() {
    loading.value = true
    error.value = ''
    try {
      await avatarAPI.triggerLearnPersona()
      startPolling()
    } catch (e: any) {
      error.value = e.response?.data?.message || '触发风格学习失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchLearnStatus() {
    try {
      learnStatus.value = await avatarAPI.getLearnStatus()
      if (learnStatus.value.status === 'completed' || learnStatus.value.status === 'failed') {
        stopPolling()
      }
    } catch (e: any) {
      error.value = e.response?.data?.message || '查询学习进度失败'
    }
  }

  async function fetchLearnedPersona() {
    loading.value = true
    error.value = ''
    try {
      learnedPersona.value = await avatarAPI.getLearnedPersona()
    } catch (e: any) {
      error.value = e.response?.data?.message || '获取学习结果失败'
    } finally {
      loading.value = false
    }
  }

  async function previewReply(message: string): Promise<string> {
    loading.value = true
    error.value = ''
    try {
      return await avatarAPI.previewReply(message)
    } catch (e: any) {
      error.value = e.response?.data?.message || '预览回复失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function clearLearnedPersona(): Promise<void> {
    loading.value = true
    error.value = ''
    try {
      await avatarAPI.clearLearnedPersona()
      learnedPersona.value = ''
      learnStatus.value = {
        status: 'idle',
        progress: 0,
        messageCount: 0,
        error: null
      }
    } catch (e: any) {
      error.value = e.response?.data?.message || '清除学习结果失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  function startPolling() {
    stopPolling()
    fetchLearnStatus()
    pollTimer = setInterval(fetchLearnStatus, 3000)
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  onUnmounted(() => {
    stopPolling()
  })

  return {
    learnStatus,
    learnedPersona,
    loading,
    error,
    triggerLearn,
    fetchLearnStatus,
    fetchLearnedPersona,
    previewReply,
    clearLearnedPersona,
    startPolling,
    stopPolling
  }
}

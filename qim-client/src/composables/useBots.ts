import { ref } from 'vue'
import axios from 'axios'
import { API_BASE_URL } from '../config'

const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

function getToken() {
  return localStorage.getItem('token')
}

export function useBots() {
  const loading = ref(false)
  const error = ref('')
  const botCount = ref(0)

  const fetchBots = async () => {
    loading.value = true
    try {
      const response = await axios.get(`${serverUrl.value}/api/v1/bots`, {
        headers: { Authorization: `Bearer ${getToken()}` }
      })
      return response.data.data
    } catch (e: any) {
      error.value = e.message
      return []
    } finally {
      loading.value = false
    }
  }

  const fetchTemplates = async () => {
    loading.value = true
    try {
      const response = await axios.get(`${serverUrl.value}/api/v1/bots/templates`, {
        headers: { Authorization: `Bearer ${getToken()}` }
      })
      return response.data.data
    } catch (e: any) {
      error.value = e.message
      return []
    } finally {
      loading.value = false
    }
  }

  const fetchMyBots = async () => {
    loading.value = true
    try {
      const response = await axios.get(`${serverUrl.value}/api/v1/bots/my`, {
        headers: { Authorization: `Bearer ${getToken()}` }
      })
      return response.data.data
    } catch (e: any) {
      error.value = e.message
      return []
    } finally {
      loading.value = false
    }
  }

  const fetchMyBotCount = async () => {
    try {
      const response = await axios.get(`${serverUrl.value}/api/v1/bots/my-count`, {
        headers: { Authorization: `Bearer ${getToken()}` }
      })
      botCount.value = response.data.data.count
      return botCount.value
    } catch {
      return 0
    }
  }

  const createBot = async (data: Record<string, unknown>) => {
    loading.value = true
    try {
      const response = await axios.post(`${serverUrl.value}/api/v1/bots`, data, {
        headers: { Authorization: `Bearer ${getToken()}` }
      })
      return response.data
    } catch (e: any) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  const updateBot = async (id: number, data: Record<string, unknown>) => {
    loading.value = true
    try {
      const response = await axios.put(`${serverUrl.value}/api/v1/bots/${id}`, data, {
        headers: { Authorization: `Bearer ${getToken()}` }
      })
      return response.data
    } catch (e: any) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  const deleteBot = async (id: number) => {
    loading.value = true
    try {
      const response = await axios.delete(`${serverUrl.value}/api/v1/bots/${id}`, {
        headers: { Authorization: `Bearer ${getToken()}` }
      })
      return response.data
    } catch (e: any) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    loading,
    error,
    botCount,
    fetchBots,
    fetchTemplates,
    fetchMyBots,
    fetchMyBotCount,
    createBot,
    updateBot,
    deleteBot,
  }
}

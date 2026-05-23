import axios from 'axios'
import { getStoredServerUrl } from '../composables/useServerUrl'
import type { UserAIConfig, CreateConfigRequest } from '../types/ai'

function getToken() {
  return localStorage.getItem('token')
}

const baseURL = () => getStoredServerUrl()

export const aiConfigAPI = {
  async listMyConfigs(): Promise<UserAIConfig[]> {
    const response = await axios.get(`${baseURL()}/api/v1/ai/configs/my`, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    const data = response.data.data
    return data?.list ?? data ?? []
  },

  async createConfig(data: CreateConfigRequest): Promise<{ id: number; is_verified: boolean }> {
    const response = await axios.post(`${baseURL()}/api/v1/ai/configs/my`, data, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    return response.data.data
  },

  async updateConfig(id: number, data: CreateConfigRequest): Promise<{ id: number; is_verified: boolean }> {
    const response = await axios.put(`${baseURL()}/api/v1/ai/configs/my/${id}`, data, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    return response.data.data
  },

  async deleteConfig(id: number): Promise<void> {
    await axios.delete(`${baseURL()}/api/v1/ai/configs/my/${id}`, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
  },

  async testConfig(id: number): Promise<{ success: boolean; message: string }> {
    const response = await axios.post(`${baseURL()}/api/v1/ai/configs/my/${id}/test`, {}, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    return response.data.data
  }
}

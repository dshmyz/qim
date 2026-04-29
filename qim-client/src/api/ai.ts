import axios from 'axios'
import { API_BASE_URL } from '../config'
import type { UserAIConfig, CreateConfigRequest } from '../types/ai'

function getToken() {
  return localStorage.getItem('token')
}

export const aiConfigAPI = {
  async listMyConfigs(): Promise<UserAIConfig[]> {
    const response = await axios.get(`${API_BASE_URL}/api/v1/ai/configs/my`, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    return response.data.data
  },

  async createConfig(data: CreateConfigRequest): Promise<{ id: number; is_verified: boolean }> {
    const response = await axios.post(`${API_BASE_URL}/api/v1/ai/configs/my`, data, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    return response.data.data
  },

  async updateConfig(id: number, data: CreateConfigRequest): Promise<{ id: number; is_verified: boolean }> {
    const response = await axios.put(`${API_BASE_URL}/api/v1/ai/configs/my/${id}`, data, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    return response.data.data
  },

  async deleteConfig(id: number): Promise<void> {
    await axios.delete(`${API_BASE_URL}/api/v1/ai/configs/my/${id}`, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
  },

  async testConfig(id: number): Promise<{ success: boolean; message: string }> {
    const response = await axios.post(`${API_BASE_URL}/api/v1/ai/configs/my/${id}/test`, {}, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    return response.data.data
  }
}

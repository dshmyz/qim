import { ref } from 'vue'
import { aiConfigAPI } from '../api/ai'
import type { UserAIConfig, CreateConfigRequest } from '../types/ai'

export function useModelConfigs() {
  const configs = ref<UserAIConfig[]>([])
  const loading = ref(false)
  const error = ref('')

  async function fetchConfigs() {
    loading.value = true
    error.value = ''
    try {
      configs.value = await aiConfigAPI.listMyConfigs()
    } catch (e: any) {
      error.value = e.response?.data?.message || '加载配置失败'
    } finally {
      loading.value = false
    }
  }

  async function createConfig(data: CreateConfigRequest) {
    loading.value = true
    error.value = ''
    try {
      const result = await aiConfigAPI.createConfig(data)
      await fetchConfigs()
      return result
    } catch (e: any) {
      error.value = e.response?.data?.message || '创建配置失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function updateConfig(id: number, data: CreateConfigRequest) {
    loading.value = true
    error.value = ''
    try {
      const result = await aiConfigAPI.updateConfig(id, data)
      await fetchConfigs()
      return result
    } catch (e: any) {
      error.value = e.response?.data?.message || '更新配置失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function deleteConfig(id: number) {
    loading.value = true
    error.value = ''
    try {
      await aiConfigAPI.deleteConfig(id)
      await fetchConfigs()
    } catch (e: any) {
      error.value = e.response?.data?.message || '删除配置失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function testConfig(id: number) {
    loading.value = true
    error.value = ''
    try {
      const result = await aiConfigAPI.testConfig(id)
      await fetchConfigs()
      return result
    } catch (e: any) {
      error.value = e.response?.data?.message || '测试连接失败'
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    configs,
    loading,
    error,
    fetchConfigs,
    createConfig,
    updateConfig,
    deleteConfig,
    testConfig,
  }
}

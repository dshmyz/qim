import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  getProviders,
  createProvider,
  updateProvider,
  deleteProvider,
  testProviderConnection,
  toggleProviderStatus,
} from '@/api/ai'
import type { AIProvider, CreateProviderParams, UpdateProviderParams, TestConnectionResult } from '@/types/ai'

export const useAIStore = defineStore('ai', () => {
  const providers = ref<AIProvider[]>([])
  const loading = ref(false)
  const testingId = ref<number | null>(null)

  async function loadProviders() {
    loading.value = true
    try {
      const { data } = await getProviders()
      providers.value = data.data
    } finally {
      loading.value = false
    }
  }

  async function addProvider(params: CreateProviderParams) {
    loading.value = true
    try {
      const { data } = await createProvider(params)
      providers.value.unshift(data.data)
      return data.data
    } finally {
      loading.value = false
    }
  }

  async function editProvider(id: number, params: UpdateProviderParams) {
    loading.value = true
    try {
      const { data } = await updateProvider(id, params)
      const index = providers.value.findIndex(p => p.id === id)
      if (index !== -1) {
        providers.value[index] = data.data
      }
      return data.data
    } finally {
      loading.value = false
    }
  }

  async function removeProvider(id: number) {
    loading.value = true
    try {
      await deleteProvider(id)
      providers.value = providers.value.filter(p => p.id !== id)
    } finally {
      loading.value = false
    }
  }

  async function toggleProvider(id: number) {
    loading.value = true
    try {
      const provider = providers.value.find(p => p.id === id)
      if (!provider) return
      const { data } = await toggleProviderStatus(id, !provider.enabled)
      const index = providers.value.findIndex(p => p.id === id)
      if (index !== -1) {
        providers.value[index] = data.data
      }
      return data.data
    } finally {
      loading.value = false
    }
  }

  async function testConnection(id: number): Promise<TestConnectionResult> {
    testingId.value = id
    try {
      const { data } = await testProviderConnection(id)
      const provider = providers.value.find(p => p.id === id)
      if (provider) {
        provider.status = data.data.success ? 'connected' : 'error'
        provider.lastTestAt = new Date().toISOString()
      }
      return data.data
    } finally {
      testingId.value = null
    }
  }

  return {
    providers,
    loading,
    testingId,
    loadProviders,
    addProvider,
    editProvider,
    removeProvider,
    toggleProvider,
    testConnection,
  }
})

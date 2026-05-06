import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getVersions, createVersion, updateVersion, deleteVersion, getVersionDistribution } from '@/api/client'
import type { ClientVersion, VersionDistribution, CreateVersionParams, UpdateVersionParams } from '@/types/client'

export const useClientStore = defineStore('client', () => {
  const versions = ref<ClientVersion[]>([])
  const distribution = ref<VersionDistribution[]>([])
  const loading = ref(false)

  async function loadVersions() {
    loading.value = true
    try {
      const { data } = await getVersions()
      versions.value = data.data.list || []
    } finally {
      loading.value = false
    }
  }

  async function addVersion(params: CreateVersionParams) {
    loading.value = true
    try {
      const { data } = await createVersion(params)
      versions.value.unshift(data.data)
      return data.data
    } finally {
      loading.value = false
    }
  }

  async function editVersion(id: number, params: UpdateVersionParams) {
    loading.value = true
    try {
      const { data } = await updateVersion(id, params)
      const index = versions.value.findIndex(v => v.id === id)
      if (index !== -1) {
        versions.value[index] = data.data
      }
      return data.data
    } finally {
      loading.value = false
    }
  }

  async function removeVersion(id: number) {
    loading.value = true
    try {
      await deleteVersion(id)
      versions.value = versions.value.filter(v => v.id !== id)
    } finally {
      loading.value = false
    }
  }

  async function loadDistribution() {
    loading.value = true
    try {
      const { data } = await getVersionDistribution()
      distribution.value = data.data
    } finally {
      loading.value = false
    }
  }

  return {
    versions,
    distribution,
    loading,
    loadVersions,
    addVersion,
    editVersion,
    removeVersion,
    loadDistribution,
  }
})

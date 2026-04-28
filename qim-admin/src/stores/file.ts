import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getFileStatistics, getLargeFiles } from '@/api/files'
import type { FileStatistics, LargeFile } from '@/types/file'

export const useFileStore = defineStore('file', () => {
  const statistics = ref<FileStatistics | null>(null)
  const largeFiles = ref<LargeFile[]>([])
  const loading = ref(false)

  async function loadStatistics() {
    loading.value = true
    try {
      const { data } = await getFileStatistics()
      statistics.value = data.data
    } finally {
      loading.value = false
    }
  }

  async function loadLargeFiles(limit: number = 10) {
    loading.value = true
    try {
      const { data } = await getLargeFiles(limit)
      largeFiles.value = data.data
    } finally {
      loading.value = false
    }
  }

  return {
    statistics,
    largeFiles,
    loading,
    loadStatistics,
    loadLargeFiles,
  }
})

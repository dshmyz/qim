import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getServerMetrics, getServiceStatus, getServerMetricsHistory, getAIReplyQualityMetrics } from '@/api/monitor'
import type { ServerMetrics, ServiceStatus, AIReplyQualityMetrics } from '@/types/monitor'

export const useMonitorStore = defineStore('monitor', () => {
  const serverMetrics = ref<ServerMetrics | null>(null)
  const serviceStatus = ref<ServiceStatus[]>([])
  const loading = ref(false)
  const metricsHistory = ref<ServerMetrics[]>([])
  const aiReplyQuality = ref<AIReplyQualityMetrics | null>(null)
  
  async function loadServerMetrics() {
    loading.value = true
    try {
      const { data } = await getServerMetrics()
      serverMetrics.value = data.data
    } finally {
      loading.value = false
    }
  }
  
  async function loadServiceStatus() {
    loading.value = true
    try {
      const { data } = await getServiceStatus()
      serviceStatus.value = data.data
    } finally {
      loading.value = false
    }
  }

  async function loadMetricsHistory(params: { startTime: string; endTime: string; interval: number }) {
    loading.value = true
    try {
      const { data } = await getServerMetricsHistory(params)
      metricsHistory.value = data.data
    } finally {
      loading.value = false
    }
  }

  async function loadAIReplyQuality() {
    const { data } = await getAIReplyQualityMetrics()
    aiReplyQuality.value = data.data
  }
  
  return {
    serverMetrics,
    serviceStatus,
    loading,
    metricsHistory,
    loadServerMetrics,
    loadServiceStatus,
    loadMetricsHistory,
    aiReplyQuality,
    loadAIReplyQuality
  }
})

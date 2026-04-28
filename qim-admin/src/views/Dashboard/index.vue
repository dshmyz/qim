<template>
  <div class="dashboard-container">
    <div class="dashboard-header">
      <div class="header-content">
        <h1 class="dashboard-title">欢迎回来，{{ userName }}</h1>
        <p class="dashboard-subtitle">{{ greeting }}</p>
      </div>
      <el-button type="primary" :icon="Refresh" @click="refreshData" :loading="refreshing">
        刷新数据
      </el-button>
    </div>

    <StatCards :stats="dashboardStats" :loading="statsLoading" />

    <ChartPlaceholders />

    <RecentActivityTable
      :registrations="recentRegistrations"
      :loading="registrationsLoading"
      @view-all="handleViewAll"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import StatCards from '@/components/dashboard/StatCards.vue'
import RecentActivityTable from '@/components/dashboard/RecentActivityTable.vue'
import ChartPlaceholders from '@/components/dashboard/ChartPlaceholders.vue'
import { getDashboardStats, getRecentRegistrations } from '@/api/statistics'
import { useAuthStore } from '@/stores/auth'
import type { DashboardData, RecentRegistration } from '@/types'

const router = useRouter()
const authStore = useAuthStore()

const userName = computed(() => (authStore.user as Record<string, unknown>)?.nickname || authStore.user?.username || '管理员')

const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 6) return '夜深了，注意休息'
  if (hour < 9) return '早上好，新的一天开始了'
  if (hour < 12) return '上午好，今天工作顺利吗？'
  if (hour < 14) return '中午好，记得吃午饭'
  if (hour < 18) return '下午好，继续加油'
  if (hour < 22) return '晚上好，辛苦了一天'
  return '夜深了，注意休息'
})

const dashboardStats = ref<DashboardData>({
  totalUsers: 0,
  onlineUsers: 0,
  totalGroups: 0,
  totalMessages: 0,
})

const recentRegistrations = ref<RecentRegistration[]>([])

const statsLoading = ref(false)
const registrationsLoading = ref(false)
const refreshing = ref(false)

const fetchDashboardStats = async () => {
  statsLoading.value = true
  try {
    const { data } = await getDashboardStats()
    dashboardStats.value = data.data ?? {
      totalUsers: 0,
      onlineUsers: 0,
      totalGroups: 0,
      totalMessages: 0,
    }
  } catch (error) {
    console.error('Failed to fetch dashboard stats:', error)
    ElMessage.error('获取仪表盘数据失败')
  } finally {
    statsLoading.value = false
  }
}

const fetchRecentRegistrations = async () => {
  registrationsLoading.value = true
  try {
    const { data } = await getRecentRegistrations({ page: 1, pageSize: 10 })
    recentRegistrations.value = data.data?.list ?? []
  } catch (error) {
    console.error('Failed to fetch recent registrations:', error)
    ElMessage.error('获取注册数据失败')
  } finally {
    registrationsLoading.value = false
  }
}

const refreshData = async () => {
  refreshing.value = true
  try {
    await Promise.all([
      fetchDashboardStats(),
      fetchRecentRegistrations(),
    ])
    ElMessage.success('数据已刷新')
  } catch (error) {
    console.error('Failed to refresh dashboard:', error)
  } finally {
    refreshing.value = false
  }
}

const handleViewAll = () => {
  router.push('/users')
}

onMounted(() => {
  fetchDashboardStats()
  fetchRecentRegistrations()
})
</script>

<style scoped>
.dashboard-container {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.dashboard-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-5);
  background: linear-gradient(135deg, #0ea5e9 0%, #6366f1 100%);
  border-radius: var(--radius-xl);
  color: white;
}

.header-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.dashboard-title {
  margin: 0;
  font-size: 24px;
  font-weight: 800;
  color: white;
  letter-spacing: -0.02em;
  line-height: 1.2;
}

.dashboard-subtitle {
  margin: 0;
  font-size: 14px;
  color: rgba(255, 255, 255, 0.85);
  font-weight: 500;
}

:deep(.el-button--primary) {
  background: rgba(255, 255, 255, 0.15) !important;
  border-color: rgba(255, 255, 255, 0.3) !important;
  color: white !important;
  backdrop-filter: blur(8px);
}

:deep(.el-button--primary:hover) {
  background: rgba(255, 255, 255, 0.25) !important;
  border-color: rgba(255, 255, 255, 0.5) !important;
}

@media (max-width: 640px) {
  .dashboard-header {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-4);
    padding: var(--space-4);
  }

  .dashboard-title {
    font-size: 24px;
  }

  .dashboard-subtitle {
    font-size: 14px;
  }
}
</style>

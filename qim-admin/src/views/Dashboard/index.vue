<template>
  <div class="dashboard-container">
    <div class="dashboard-header">
      <div>
        <h1 class="dashboard-title">仪表盘</h1>
        <p class="dashboard-subtitle">系统运行概览</p>
      </div>
      <div class="header-actions">
        <el-button type="primary" text @click="refreshData" :loading="refreshing">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <div class="stat-cards-section">
      <StatCards :stats="dashboardStats" :loading="statsLoading" />
    </div>

    <div class="charts-section">
      <ChartPlaceholders />
    </div>

    <div class="activity-section">
      <RecentActivityTable
        :registrations="recentRegistrations"
        :loading="registrationsLoading"
        @view-all="handleViewAll"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import StatCards from '@/components/dashboard/StatCards.vue'
import RecentActivityTable from '@/components/dashboard/RecentActivityTable.vue'
import ChartPlaceholders from '@/components/dashboard/ChartPlaceholders.vue'
import { getDashboardStats, getRecentRegistrations } from '@/api/statistics'
import type { DashboardData, RecentRegistration } from '@/types'

const router = useRouter()

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
    dashboardStats.value = data.data
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
    recentRegistrations.value = data.data.list
  } catch (error) {
    console.error('Failed to fetch recent registrations:', error)
    ElMessage.error('获取注册数据失败')
  } finally {
    registrationsLoading.value = false
  }
}

const refreshData = async () => {
  refreshing.value = true
  await Promise.all([
    fetchDashboardStats(),
    fetchRecentRegistrations(),
  ])
  refreshing.value = false
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
  gap: var(--space-6);
}

/* ==========================================
   仪表盘头部
   ========================================== */
.dashboard-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: var(--space-4);
  border-bottom: 2px solid var(--color-border-light);
}

.dashboard-title {
  margin: 0 0 var(--space-1);
  font-size: 28px;
  font-weight: 800;
  color: var(--color-text-primary);
  letter-spacing: -0.02em;
}

.dashboard-subtitle {
  margin: 0;
  font-size: 14px;
  color: var(--color-text-secondary);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

/* ==========================================
   统计卡片区域
   ========================================== */
.stat-cards-section {
  animation: fadeInUp 0.5s var(--ease-out);
}

/* ==========================================
   图表区域
   ========================================== */
.charts-section {
  animation: fadeInUp 0.5s var(--ease-out) 0.1s backwards;
}

/* ==========================================
   活动表格区域
   ========================================== */
.activity-section {
  animation: fadeInUp 0.5s var(--ease-out) 0.2s backwards;
}

/* ==========================================
   动画
   ========================================== */
@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* ==========================================
   响应式
   ========================================== */
@media (max-width: 768px) {
  .dashboard-header {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-4);
  }
}
</style>

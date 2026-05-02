<template>
  <div class="stats-cards">
    <div class="stat-card">
      <div class="stat-label">总链接数</div>
      <div class="stat-value">{{ stats.totalLinks }}</div>
      <div class="stat-trend positive" v-if="stats.totalLinksTrend">
        + {{ stats.totalLinksTrend }}% 本周
      </div>
    </div>

    <div class="stat-card">
      <div class="stat-label">总访问量</div>
      <div class="stat-value">{{ stats.totalVisits }}</div>
      <div class="stat-trend positive" v-if="stats.totalVisitsTrend">
        + {{ stats.totalVisitsTrend }}% 本周
      </div>
    </div>

    <div class="stat-card">
      <div class="stat-label">今日访问</div>
      <div class="stat-value">{{ stats.todayVisits }}</div>
      <div class="stat-trend positive" v-if="stats.todayVisitsTrend">
        + {{ stats.todayVisitsTrend }}% 今日
      </div>
    </div>

    <div class="stat-card">
      <div class="stat-label">活跃链接</div>
      <div class="stat-value">{{ stats.activeLinks }}</div>
      <div class="stat-trend neutral" v-if="stats.activeRate">
        {{ stats.activeRate }}% 活跃率
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
export interface Stats {
  totalLinks: number
  totalLinksTrend?: number
  totalVisits: number
  totalVisitsTrend?: number
  todayVisits: number
  todayVisitsTrend?: number
  activeLinks: number
  activeRate?: number
}

defineProps<{
  stats: Stats
}>()
</script>

<style scoped>
.stats-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: var(--card-bg, white);
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 1px 3px var(--shadow-color, rgba(0, 0, 0, 0.1));
  border: 1px solid var(--border-color, transparent);
}

.stat-label {
  color: var(--text-secondary, #6b7280);
  font-size: 13px;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary, #1f2937);
  margin-bottom: 4px;
}

.stat-trend {
  font-size: 12px;
  margin-top: 4px;
}

.stat-trend.positive {
  color: #10b981;
}

.stat-trend.negative {
  color: #ef4444;
}

.stat-trend.neutral {
  color: var(--text-secondary, #6b7280);
}

@media (max-width: 768px) {
  .stats-cards {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 480px) {
  .stats-cards {
    grid-template-columns: 1fr;
  }

  .stat-card {
    padding: 16px;
  }

  .stat-value {
    font-size: 24px;
  }
}
</style>

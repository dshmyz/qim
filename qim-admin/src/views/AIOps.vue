<template>
  <div class="ai-ops-page">
    <el-row :gutter="20">
      <!-- 概览卡片 -->
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" :class="{ 'is-active': dashboard.ai_configured }">
              <el-icon :size="32"><Cpu /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">AI 服务状态</div>
              <div class="stat-value" :class="{ active: dashboard.ai_configured }">
                {{ dashboard.ai_configured ? '运行中' : '未配置' }}
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon blue">
              <el-icon :size="32"><ChatDotRound /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">AI 消息数</div>
              <div class="stat-value">{{ dashboard.stats.ai_messages }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon green">
              <el-icon :size="32"><DataAnalysis /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">知识库笔记</div>
              <div class="stat-value">{{ knowledgeStats.active_notes || 0 }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon orange">
              <el-icon :size="32"><Tools /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">可用工具</div>
              <div class="stat-value">{{ dashboard.tools.length }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px">
      <!-- AI 工具列表 -->
      <el-col :span="12">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>🛠️ AI 可用工具</span>
              <el-button size="small" @click="refreshDashboard">刷新</el-button>
            </div>
          </template>
          <el-table :data="dashboard.tools" v-loading="loading">
            <el-table-column prop="name" label="工具名称" min-width="150" />
            <el-table-column label="参数">
              <template #default="{ row }">
                <el-tag
                  v-for="param in Object.keys(row.parameters?.properties || {})"
                  :key="param"
                  size="small"
                  class="param-tag"
                >
                  {{ param }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <!-- 知识库统计 -->
      <el-col :span="12">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>📚 知识库统计</span>
              <el-button size="small" @click="refreshKnowledgeStats">刷新</el-button>
            </div>
          </template>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="总笔记数">
              {{ knowledgeStats.total_notes || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="活跃笔记数">
              {{ knowledgeStats.active_notes || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="贡献用户数">
              {{ knowledgeStats.user_count || 0 }}
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px">
      <!-- AI 功能说明 -->
      <el-col :span="24">
        <el-card shadow="never">
          <template #header>
            <span>🤖 AI 智能化功能</span>
          </template>
          <el-row :gutter="20">
            <el-col :span="8">
              <div class="feature-card">
                <h4>智能回复</h4>
                <p>AI 自动识别群聊中的问题并回复，基于意图分析和知识库内容</p>
                <el-tag type="success" size="small">已启用</el-tag>
              </div>
            </el-col>
            <el-col :span="8">
              <div class="feature-card">
                <h4>待办提取</h4>
                <p>从群聊消息中自动识别任务、分配负责人、设置截止时间</p>
                <el-tag type="success" size="small">已启用</el-tag>
              </div>
            </el-col>
            <el-col :span="8">
              <div class="feature-card">
                <h4>群聊总结</h4>
                <p>每日 22:00 自动生成群聊日报，包含热门话题、待办和决策</p>
                <el-tag type="warning" size="small">定时任务</el-tag>
              </div>
            </el-col>
          </el-row>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Cpu, ChatDotRound, DataAnalysis, Tools } from '@element-plus/icons-vue'
import request from '@/utils/request'

const dashboard = ref({
  ai_configured: false,
  provider: '',
  tools: [] as any[],
  stats: {
    total_bots: 0,
    active_bots: 0,
    total_messages: 0,
    ai_messages: 0,
  },
})

const knowledgeStats = ref({
  total_notes: 0,
  active_notes: 0,
  user_count: 0,
})

const loading = ref(false)

const fetchDashboard = async () => {
  try {
    loading.value = true
    const res = await request.get('/v1/admin/ai/dashboard')
    dashboard.value = res.data.data
  } catch (error) {
    ElMessage.error('获取 AI 运维数据失败')
  } finally {
    loading.value = false
  }
}

const fetchKnowledgeStats = async () => {
  // 后端暂未提供知识库统计 API，保留默认值
  // TODO: 后端实现 /admin/ai/knowledge-stats 后对接
}

const refreshDashboard = () => {
  fetchDashboard()
  fetchKnowledgeStats()
  ElMessage.success('已刷新')
}

const refreshKnowledgeStats = () => {
  fetchKnowledgeStats()
  ElMessage.success('已刷新')
}

onMounted(() => {
  fetchDashboard()
  fetchKnowledgeStats()
})
</script>

<style scoped>
.ai-ops-page {
  padding: 20px;
}

.stat-card {
  margin-bottom: 20px;
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-icon {
  width: 56px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.04);
  color: var(--color-text-secondary);
}

.stat-icon.is-active,
.stat-icon.blue {
  background: rgba(14, 165, 233, 0.1);
  color: #0ea5e9;
}

.stat-icon.green {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.stat-icon.orange {
  background: rgba(249, 115, 22, 0.1);
  color: #f97316;
}

.stat-info {
  flex: 1;
}

.stat-label {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-bottom: 4px;
}

.stat-value {
  font-size: 22px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.stat-value.active {
  color: #22c55e;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.param-tag {
  margin-right: 4px;
  margin-bottom: 4px;
}

.feature-card {
  padding: 20px;
  border-radius: 8px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
}

.feature-card h4 {
  margin: 0 0 8px;
  font-size: 16px;
  font-weight: 600;
}

.feature-card p {
  margin: 0 0 12px;
  font-size: 13px;
  color: var(--color-text-secondary);
  line-height: 1.6;
}
</style>

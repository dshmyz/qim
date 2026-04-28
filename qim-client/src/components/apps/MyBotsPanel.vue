<template>
  <div class="my-bots-panel">
    <div v-if="loading" class="loading">
      <i class="fas fa-spinner fa-spin"></i>
      <span>加载中...</span>
    </div>

    <div v-else-if="myBots.length === 0" class="empty-state">
      <i class="fas fa-robot"></i>
      <p>你还没有创建过机器人</p>
      <button class="create-btn" @click="$emit('open-create')">
        <i class="fas fa-plus"></i> 创建第一个机器人
      </button>
    </div>

    <div v-else class="bot-grid">
      <div v-for="bot in myBots" :key="bot.id" class="bot-card" :class="bot.approval_status">
        <div class="bot-header">
          <div class="bot-avatar">
            <img :src="bot.avatar" :alt="bot.name" v-if="bot.avatar">
            <i class="fas fa-robot" v-else></i>
          </div>
          <span class="status-badge" :class="bot.approval_status">
            {{ statusLabel(bot.approval_status) }}
          </span>
        </div>
        <div class="bot-body">
          <h4>{{ bot.name }}</h4>
          <p>{{ bot.description }}</p>
          <p v-if="bot.approval_status === 'rejected' && bot.reject_reason" class="reject-reason">
            拒绝原因：{{ bot.reject_reason }}
          </p>
        </div>
        <div class="bot-actions">
          <button v-if="bot.approval_status === 'approved'" class="action-btn primary" @click="$emit('use-bot', bot)">
            <i class="fas fa-comment"></i> 使用
          </button>
          <button v-if="['pending', 'rejected'].includes(bot.approval_status)" class="action-btn" @click="$emit('edit-bot', bot)">
            <i class="fas fa-edit"></i> 编辑
          </button>
          <button class="action-btn danger" @click="confirmDelete(bot)">
            <i class="fas fa-trash"></i> 删除
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useBots } from '../../composables/useBots'

const emit = defineEmits(['open-create', 'edit-bot', 'use-bot'])

const { loading, fetchMyBots, deleteBot } = useBots()
const myBots = ref<any[]>([])

const loadMyBots = async () => {
  myBots.value = (await fetchMyBots()) || []
}

const statusLabel = (status: string) => {
  const map: Record<string, string> = { approved: '可用', pending: '待审批', rejected: '已拒绝' }
  return map[status] || status
}

const confirmDelete = async (bot: any) => {
  if (confirm(`确定删除「${bot.name}」吗？删除后将释放一个创建配额。`)) {
    await deleteBot(bot.id)
    await loadMyBots()
  }
}

onMounted(loadMyBots)
</script>

<style scoped>
.my-bots-panel {
  padding: 10px 0;
}

.loading {
  text-align: center;
  padding: 40px;
  color: var(--text-secondary);
}

.loading i {
  font-size: 24px;
  margin-right: 8px;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: var(--text-secondary);
}

.empty-state i {
  font-size: 48px;
  margin-bottom: 12px;
  color: var(--text-tertiary);
}

.create-btn {
  margin-top: 20px;
  padding: 10px 24px;
  border: 1px solid var(--primary-color);
  border-radius: 6px;
  background: var(--primary-color);
  color: white;
  cursor: pointer;
  font-size: 14px;
}

.bot-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.bot-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 16px;
  transition: all 0.2s;
}

.bot-card.pending {
  opacity: 0.6;
}

.bot-card.rejected {
  border-color: #f44336;
}

.bot-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.bot-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--bg-color);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.bot-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.status-badge {
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.status-badge.approved {
  background: #E8F5E8;
  color: #388E3C;
}

.status-badge.pending {
  background: #FFF8E1;
  color: #FF9800;
}

.status-badge.rejected {
  background: #FFEBEE;
  color: #F44336;
}

.bot-body h4 {
  margin: 0 0 6px;
  font-size: 15px;
  color: var(--text-primary);
}

.bot-body p {
  margin: 0;
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.4;
}

.reject-reason {
  margin-top: 8px !important;
  color: #F44336 !important;
  font-size: 12px !important;
}

.bot-actions {
  display: flex;
  gap: 8px;
  margin-top: 16px;
}

.action-btn {
  flex: 1;
  padding: 8px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--card-bg);
  cursor: pointer;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  transition: all 0.2s;
}

.action-btn:hover {
  background: var(--hover-color);
}

.action-btn.primary {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.action-btn.danger {
  color: #F44336;
  border-color: #F44336;
}

.action-btn.danger:hover {
  background: #FFEBEE;
}

@media (max-width: 768px) {
  .bot-grid {
    grid-template-columns: 1fr;
  }
}
</style>

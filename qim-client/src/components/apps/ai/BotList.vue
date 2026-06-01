<template>
  <div class="bot-list">
    <!-- Loading 状态 -->
    <div v-if="loading" class="loading-container">
      <i class="fas fa-spinner fa-spin"></i>
      <span>加载中...</span>
    </div>

    <!-- 空状态 -->
    <div v-else-if="bots.length === 0" class="empty-state">
      <i class="fas fa-robot"></i>
      <p>暂无可用的机器人</p>
      <button class="create-btn" @click="$emit('createBot')">
        <i class="fas fa-plus"></i> 创建第一个机器人
      </button>
    </div>

    <!-- 机器人列表 -->
    <div v-else class="list">
      <div
        v-for="bot in bots"
        :key="bot.id"
        class="bot-item"
        :class="{ disabled: bot.approval_status !== 'approved' }"
        @click="bot.approval_status === 'approved' && $emit('select', bot.id)"
      >
        <div class="avatar">
          <Avatar v-if="bot.avatar" :src="bot.avatar" :name="bot.name" :alt="bot.name" size="sm" />
          <i class="fas fa-robot" v-else></i>
        </div>
        <div class="info">
          <div class="name-row">
            <h4>{{ bot.name }}</h4>
            <span v-if="bot.approval_status === 'pending'" class="status-badge pending">审批中</span>
            <span v-else-if="bot.approval_status === 'rejected'" class="status-badge rejected">已拒绝</span>
            <span v-else-if="bot.approval_status === 'approved'" class="status-badge approved">已启用</span>
          </div>
          <p>{{ bot.description }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import Avatar from '../../shared/Avatar.vue'

interface Bot {
  id: number
  name: string
  description: string
  avatar?: string
  type?: string
  status?: string
  approval_status?: string
}

defineProps<{
  bots: Bot[]
  loading?: boolean
}>()

defineEmits<{
  select: [botId: number]
  createBot: []
}>()
</script>

<style scoped>
.bot-list {
  padding: 20px;
}

.list {
  display: grid;
  gap: 16px;
}

.bot-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.bot-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
}

.avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--bg-color);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  flex-shrink: 0;
}

.avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar i {
  font-size: 24px;
  color: var(--primary-color);
}

.info .name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.info h4 {
  margin: 0;
  font-size: 15px;
}

.status-badge {
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.status-badge.pending {
  background: #fff3cd;
  color: #856404;
}

.status-badge.approved {
  background: #d4edda;
  color: #155724;
}

.status-badge.rejected {
  background: #f8d7da;
  color: #721c24;
}

.bot-item.disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.bot-item.disabled:hover {
  transform: none;
  box-shadow: none;
}

.info p {
  margin: 0;
  font-size: 13px;
  color: var(--text-secondary);
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
}

.empty-state i {
  font-size: 48px;
  color: var(--text-tertiary);
  margin-bottom: 10px;
}

.empty-state p {
  color: var(--text-secondary);
}

.create-btn {
  margin-top: 16px;
  padding: 10px 20px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s;
}

.create-btn:hover {
  background: var(--primary-hover);
  transform: translateY(-1px);
}

.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  gap: 12px;
}

.loading-container i {
  font-size: 32px;
  color: var(--primary-color);
}

.loading-container span {
  color: var(--text-secondary);
  font-size: 14px;
}
</style>

<template>
  <div class="my-assets">
    <section class="asset-section">
      <div class="asset-header">
        <h3>
          <i class="fas fa-robot"></i>
          我的机器人
          <span class="count">{{ bots.length }}</span>
        </h3>
        <button class="add-btn" @click="$emit('create-bot')">
          <i class="fas fa-plus"></i> 创建
        </button>
      </div>

      <div v-if="bots.length === 0" class="empty-state">
        <i class="fas fa-robot"></i>
        <p>还没有创建机器人</p>
        <button class="create-first-btn" @click="$emit('create-bot')">
          创建第一个机器人
        </button>
      </div>

      <div v-else class="bot-grid">
        <div
          v-for="bot in bots"
          :key="bot.id"
          class="bot-card"
          :class="bot.approval_status"
          @click="$emit('use-bot', bot)"
        >
          <div class="bot-avatar">
            <Avatar v-if="bot.avatar" :src="bot.avatar" :name="bot.name" :alt="bot.name" size="sm" />
            <i v-else class="fas fa-robot"></i>
          </div>
          <div class="bot-info">
            <h4>{{ bot.name }}</h4>
            <p>{{ bot.description || '暂无描述' }}</p>
            <span v-if="bot.approval_status === 'pending'" class="status-tag pending">待审批</span>
            <span v-else-if="bot.approval_status === 'rejected'" class="status-tag rejected">已拒绝</span>
          </div>
          <div class="bot-actions" v-if="bot.approval_status === 'approved'">
            <button class="config-btn" @click.stop="$emit('config-bot', bot)" title="配置令牌/Webhook">
              <i class="fas fa-cog"></i>
            </button>
            <button class="config-btn delete-btn" @click.stop="$emit('delete-bot', bot)" title="删除">
              <i class="fas fa-trash"></i>
            </button>
          </div>
          <button v-else class="config-btn delete-btn" @click.stop="$emit('delete-bot', bot)" title="删除">
            <i class="fas fa-trash"></i>
          </button>
          <button class="use-btn" @click.stop="$emit('use-bot', bot)" :disabled="bot.approval_status !== 'approved'">
            <i class="fas fa-comment"></i> 使用
          </button>
        </div>
      </div>
    </section>

    <section class="asset-section">
      <div class="asset-header">
        <h3>
          <i class="fas fa-key"></i>
          我的模型配置
          <span class="count">{{ configs.length }}</span>
        </h3>
        <button class="add-btn" @click="$emit('add-config')">
          <i class="fas fa-plus"></i> 添加
        </button>
      </div>

      <div v-if="configs.length === 0" class="empty-state small">
        <i class="fas fa-key"></i>
        <p>暂无配置</p>
        <button class="create-first-btn" @click="$emit('add-config')">
          添加配置
        </button>
      </div>

      <div v-else class="config-list">
        <div
          v-for="config in configs"
          :key="config.id"
          class="config-item"
          @click="$emit('edit-config', config)"
        >
          <div class="config-icon">
            <i class="fas fa-cube"></i>
          </div>
          <div class="config-info">
            <span class="config-name">{{ config.config_name }}</span>
            <span class="config-status">
              <span :class="['status-dot', config.is_verified ? 'active' : 'inactive']"></span>
              {{ config.is_verified ? '已验证' : '未验证' }}
            </span>
          </div>
          <div class="config-actions">
            <button class="icon-btn" @click.stop="$emit('test-config', config.id)">
              <i class="fas fa-plug"></i>
            </button>
            <button class="icon-btn" @click.stop="$emit('delete-config', config)">
              <i class="fas fa-trash"></i>
            </button>
          </div>
        </div>
      </div>
    </section>

  </div>
</template>

<script setup lang="ts">
import Avatar from '../../shared/Avatar.vue'

interface Bot {
  id: number
  name: string
  description?: string
  avatar?: string
  approval_status: string
}

interface Config {
  id: number
  config_name: string
  is_verified: boolean
}

const props = defineProps<{
  bots: Bot[]
  configs: Config[]
}>()

defineEmits([
  'create-bot',
  'use-bot',
  'config-bot',
  'add-config',
  'edit-config',
  'test-config',
  'delete-config',
  'delete-bot'
])
</script>

<style scoped>
.my-assets {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.asset-section {
  background: var(--card-bg);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-color);
  overflow: hidden;
}

.asset-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
}

.asset-header h3 {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: 8px;
}

.asset-header h3 i {
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
}

.asset-header .count {
  font-size: var(--font-size-xxs);
  font-weight: normal;
  color: var(--text-secondary);
  background: var(--hover-color);
  padding: 2px 8px;
  border-radius: 10px;
}

.add-btn {
  padding: 8px 14px;
  border: 1px solid var(--primary-color);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--primary-color);
  font-size: var(--font-size-xs);
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 6px;
}

.add-btn:hover {
  background: var(--primary-color);
  color: white;
}

.empty-state {
  padding: 40px 20px;
  text-align: center;
  color: var(--text-secondary);
}

.empty-state i {
  font-size: var(--font-size-4xl);
  margin-bottom: 12px;
  color: var(--text-tertiary);
}

.empty-state.small {
  padding: 24px 20px;
}

.empty-state.small i {
  font-size: 28px;
}

.empty-state p {
  margin-bottom: 16px;
  font-size: var(--font-size-sm);
}

.create-first-btn {
  padding: 10px 20px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--primary-color);
  color: white;
  font-size: var(--font-size-sm);
  cursor: pointer;
  transition: all 0.2s;
}

.create-first-btn:hover {
  opacity: 0.9;
}

.bot-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
  padding: 20px;
}

.bot-card {
  position: relative;
  background: var(--hover-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 16px;
  transition: all 0.2s;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.bot-card:hover {
  border-color: var(--primary-color);
  background: var(--primary-light);
}

.bot-card .bot-avatar {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--primary-dark, #003d99) 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: var(--font-size-xl);
  margin-bottom: 12px;
  overflow: hidden;
}

.bot-card .bot-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.bot-card h4 {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.bot-info {
  flex: 1;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 12px;
}

.bot-card p {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  width: 100%;
}

.status-tag {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: var(--font-size-xxxs);
  font-weight: 500;
}
.status-tag.pending { background: #fff7e6; color: #d46b08; border: 1px solid #ffd591; }
.status-tag.rejected { background: #fff1f0; color: #cf1322; border: 1px solid #ffa39e; }

.use-btn {
  width: 100%;
  margin-top: auto;
  padding: 8px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--primary-color);
  color: white;
  font-size: var(--font-size-xs);
  font-weight: 500;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  transition: all 0.2s;
}

.use-btn:hover {
  opacity: 0.9;
}
.use-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.delete-btn { color: #999; }
.delete-btn:hover { color: #e74c3c; }
.bot-actions { position: absolute; top: 8px; right: 8px; display: flex; gap: 4px; }
.bot-actions .config-btn { position: static; }

.config-btn {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 28px;
  height: 28px;
  border: 1px solid var(--border-color, #e0e0e0);
  border-radius: var(--radius-sm);
  background: rgba(255, 255, 255, 0.85);
  color: #888;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-xxs);
  transition: all 0.2s;
}

.config-btn:hover {
  background: #fff;
  color: var(--primary-color);
  border-color: var(--primary-color);
}

.config-list {
  padding: 12px 20px;
}

.config-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid var(--border-color);
  cursor: pointer;
  transition: all 0.2s;
}

.config-item:last-child {
  border-bottom: none;
}

.config-item:hover {
  background: var(--hover-color);
  margin: 0 -20px;
  padding: 12px 20px;
}

.config-icon {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-sm);
  background: var(--hover-color);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.config-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.config-name {
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--text-primary);
}

.config-status {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  display: flex;
  align-items: center;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
  margin-right: 6px;
}

.status-dot.active {
  background: var(--color-success-500, #26b361);
}

.status-dot.inactive {
  background: var(--text-tertiary);
}

.config-actions {
  display: flex;
  gap: 8px;
}

.icon-btn {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s;
}

.icon-btn:hover {
  background: var(--hover-color);
  color: var(--primary-color);
}
</style>
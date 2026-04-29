<template>
  <div class="config-card">
    <div class="card-header">
      <div class="provider-icon">{{ providerInfo.icon }}</div>
      <div class="card-info">
        <h4>{{ config.config_name }}</h4>
        <p>{{ providerInfo.name }} - {{ config.model_name }}</p>
      </div>
      <div class="card-actions">
        <button class="action-btn" @click="$emit('edit')" title="编辑">
          <i class="fas fa-edit"></i>
        </button>
        <button class="action-btn" @click="$emit('test')" title="测试">
          <i class="fas fa-vial"></i>
        </button>
        <button class="action-btn delete" @click="$emit('delete')" title="删除">
          <i class="fas fa-trash"></i>
        </button>
      </div>
    </div>
    <div class="card-footer">
      <span class="status" :class="config.is_verified ? 'verified' : 'unverified'">
        <i :class="config.is_verified ? 'fas fa-check-circle' : 'fas fa-exclamation-circle'"></i>
        {{ config.is_verified ? '已验证' : '未验证' }}
      </span>
      <span class="date">{{ formatDate(config.created_at) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { AI_PROVIDERS, type UserAIConfig } from '../../../types/ai'

const props = defineProps<{
  config: UserAIConfig
}>()

defineEmits(['edit', 'test', 'delete'])

const providerInfo = computed(() => {
  return AI_PROVIDERS.find(p => p.id === props.config.provider) || { icon: '⚙️', name: '自定义', defaultModel: '', defaultBaseURL: '' }
})

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('zh-CN')
}
</script>

<style scoped>
.config-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 16px;
  transition: all 0.2s;
}

.config-card:hover {
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.provider-icon {
  font-size: 32px;
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-color);
  border-radius: 50%;
}

.card-info h4 {
  margin: 0;
  font-size: 15px;
  color: var(--text-color);
}

.card-info p {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--text-secondary);
}

.card-actions {
  margin-left: auto;
  display: flex;
  gap: 8px;
}

.action-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 6px;
  transition: all 0.2s;
}

.action-btn:hover {
  background: var(--hover-color);
  color: var(--primary-color);
}

.action-btn.delete:hover {
  color: #d32f2f;
}

.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 12px;
  border-top: 1px solid var(--border-color);
}

.status {
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.status.verified {
  color: #388e3c;
}

.status.unverified {
  color: #ff9800;
}

.date {
  font-size: 12px;
  color: var(--text-secondary);
}
</style>

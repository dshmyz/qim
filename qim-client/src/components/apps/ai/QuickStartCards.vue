<template>
  <div class="quick-start">
    <div
      v-for="card in cards"
      :key="card.id"
      :class="['quick-card', card.id]"
      @click="$emit('action', card.action)"
    >
      <div class="quick-icon">
        <i :class="card.icon"></i>
      </div>
      <h3>{{ card.title }}</h3>
      <p>{{ card.description }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
defineEmits(['action'])

const cards = [
  {
    id: 'chat',
    title: '找AI聊聊',
    description: '选择助手开始对话',
    icon: 'fas fa-comment-dots',
    action: 'chat'
  },
  {
    id: 'create',
    title: '创建助手',
    description: '打造专属AI伙伴',
    icon: 'fas fa-robot',
    action: 'create'
  },
  {
    id: 'avatar',
    title: '数字分身',
    description: 'AI驱动的另一个你',
    icon: 'fas fa-user-circle',
    action: 'avatar'
  }
]
</script>

<style scoped>
.quick-start {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  grid-auto-rows: 1fr;
  gap: 20px;
  margin-bottom: 32px;
}

.quick-card {
  background: #ffffff;
  border-radius: 16px;
  padding: 28px 24px;
  text-align: center;
  cursor: pointer;
  transition: all 0.25s ease;
  border: 1px solid #e5e7eb;
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-start;
  min-height: 220px;
  width: 100%;
  box-sizing: border-box;
}

.quick-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 4px;
  opacity: 0;
  transition: opacity 0.25s;
}

.quick-card.chat::before {
  background: #3385ff;
}

.quick-card.create::before {
  background: #26b361;
}

.quick-card.avatar::before {
  background: #f7a826;
}

.quick-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 12px 24px rgba(0, 0, 0, 0.12);
  border-color: transparent;
}

.quick-card:hover::before {
  opacity: 1;
}

.quick-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 16px;
  font-size: 24px;
  line-height: 1;
}

.quick-card.chat .quick-icon {
  background: #e6f0ff;
  color: #3385ff;
}

.quick-card.create .quick-icon {
  background: #ecfdf5;
  color: #26b361;
}

.quick-card.avatar .quick-icon {
  background: #fffdf6;
  color: #f7a826;
}

.quick-icon i {
  font-family: 'Font Awesome 5 Free';
  font-weight: 900;
  font-style: normal;
}

.quick-card h3 {
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
  margin-bottom: 8px;
}

.quick-card p {
  font-size: 13px;
  color: #6b7280;
  margin: 0;
}

@media (max-width: 768px) {
  .quick-start {
    grid-template-columns: 1fr;
  }
}
</style>
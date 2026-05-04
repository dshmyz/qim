<template>
  <div class="ai-assistant-app">
    <div class="ai-assistant-header">
      <div class="header-left">
        <button class="back-btn" @click="$emit('back')">
          <i class="fas fa-chevron-left"></i>
        </button>
        <button class="toggle-sidebar-btn" @click="$emit('toggleSidebar')">
          <i class="fas fa-compress"></i>
        </button>
        <h2>AI 助手</h2>
      </div>
    </div>

    <div class="tab-nav">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        :class="['tab-btn', { active: activeTab === tab.id }]"
        @click="activeTab = tab.id"
      >
        <i :class="tab.icon"></i>
        {{ tab.label }}
      </button>
    </div>

    <div class="tab-content">
      <ChatCenter v-if="activeTab === 'chat'" @switch-tab="switchTab" />
      <MyBotsPanel
        v-if="activeTab === 'my-bots'"
        @create="showCreateModal = true"
        @edit-bot="handleEditBot"
        @use-bot="handleUseBot"
      />
      <MyModelConfigs v-if="activeTab === 'configs'" />
      
      <QDialog v-model:visible="showCreateModal" title="创建机器人" width="600px">
        <CreateBotWizard @close="showCreateModal = false" />
      </QDialog>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import ChatCenter from './ai/ChatCenter.vue'
import MyBotsPanel from './MyBotsPanel.vue'
import MyModelConfigs from './ai/MyModelConfigs.vue'
import CreateBotWizard from './ai/CreateBotWizard.vue'
import QDialog from '../shared/QDialog.vue'

defineEmits(['back', 'toggleSidebar'])

const activeTab = ref('chat')
const showCreateModal = ref(false)

const tabs = [
  { id: 'chat', label: '对话', icon: 'fas fa-comments' },
  { id: 'my-bots', label: '我的机器人', icon: 'fas fa-robot' },
  { id: 'configs', label: '我的模型配置', icon: 'fas fa-key' }
]

function switchTab(tabId: string) {
  activeTab.value = tabId
}

function handleEditBot(bot: any) {
  console.log('Edit bot:', bot)
}

function handleUseBot(bot: any) {
  activeTab.value = 'chat'
}
</script>

<style scoped>
.ai-assistant-app {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--content-bg);
  overflow: hidden;
}

.ai-assistant-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  background: var(--card-bg);
  height: 72px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.back-btn,
.toggle-sidebar-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: var(--hover-color);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary-color);
}

.tab-nav {
  display: flex;
  border-bottom: 2px solid var(--border-color);
}

.tab-btn {
  padding: 12px 20px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 14px;
  border-bottom: 2px solid transparent;
  margin-bottom: -2px;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 6px;
}

.tab-btn:hover {
  color: var(--text-primary);
}

.tab-btn.active {
  color: var(--primary-color);
  border-bottom-color: var(--primary-color);
}

.tab-content {
  flex: 1;
  overflow-y: auto;
}

.placeholder-tab {
  padding: 60px 20px;
  text-align: center;
  color: var(--text-secondary);
}
</style>

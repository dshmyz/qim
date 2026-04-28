<template>
  <div class="ai-assistant-page">
    <el-tabs v-model="activeTab">
      <el-tab-pane label="AI 助手管理" name="management">
        <AIAssistantList />
      </el-tab-pane>
      <el-tab-pane label="Bot 审批" name="approval">
        <BotApprovalPanel />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import AIAssistantList from './AIAssistantList.vue'
import BotApprovalPanel from './BotApprovalPanel.vue'

const activeTab = ref('management')
</script>

<style scoped>
.ai-assistant-page {
  padding: var(--space-4);
}
</style>

<template>
  <div class="ai-assistant-page">
    <AIAssistantList />
  </div>
</template>

<script setup lang="ts">
import AIAssistantList from './AIAssistantList.vue'
</script>

<style scoped>
.ai-assistant-page {
  padding: var(--space-4);
}
</style>

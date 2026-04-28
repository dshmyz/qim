<!-- src/components/layout/Header/index.vue -->
<template>
  <el-header class="admin-header">
    <div class="header-left">
      <slot name="breadcrumb"></slot>
    </div>
    <div class="header-right">
      <ThemeToggle />
      <UserDropdown />
    </div>
  </el-header>
</template>

<script setup lang="ts">
import ThemeToggle from './ThemeToggle.vue'
import UserDropdown from './UserDropdown.vue'
</script>

<style scoped>
.admin-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 64px;
  background-color: var(--color-surface);
  padding: 0 var(--space-6);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.header-right {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
</style>

<!-- src/components/layout/Header/index.vue -->
<template>
  <el-header class="admin-header">
    <div class="header-left">
      <button v-if="showHamburger" class="hamburger-btn" @click="$emit('toggleSidebar')">
        <el-icon :size="20">
          <Fold v-if="!sidebarOpen" />
          <Expand v-else />
        </el-icon>
      </button>
      <slot name="breadcrumb"></slot>
    </div>
    <div class="header-right">
      <ThemeToggle />
      <UserDropdown />
    </div>
  </el-header>
</template>

<script setup lang="ts">
import { Fold, Expand } from '@element-plus/icons-vue'
import ThemeToggle from './ThemeToggle.vue'
import UserDropdown from './UserDropdown.vue'

interface Props {
  showHamburger?: boolean
  sidebarOpen?: boolean
}

const props = defineProps<Props>()

interface Emits {
  (e: 'toggleSidebar'): void
}

const emit = defineEmits<Emits>()
</script>

<style scoped>
.admin-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 64px;
  background-color: var(--color-surface);
  padding: 0 var(--space-6);
  box-shadow: none;
  width: 100%;
  flex-shrink: 0;
  border-radius: var(--radius-lg);
}

.header-left {
  display: flex;
  align-items: center;
  gap: var(--space-4);
}

.hamburger-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: none;
  background: transparent;
  color: var(--color-text-secondary);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-out);
}

.hamburger-btn:hover {
  background-color: var(--color-primary-lighter);
  color: var(--color-primary);
}

.header-right {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
</style>

<!-- src/components/layout/Header/UserDropdown.vue -->
<template>
  <el-dropdown trigger="click" @command="handleCommand">
    <span class="user-dropdown">
      <el-avatar :size="34">
        {{ authStore.user?.username?.charAt(0) || 'A' }}
      </el-avatar>
      <span class="username">{{ authStore.user?.username || 'Admin' }}</span>
      <el-icon :size="14"><ArrowDown /></el-icon>
    </span>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item command="logout">
          <el-icon><SwitchButton /></el-icon>
          <span>退出登录</span>
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { ArrowDown, SwitchButton } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()

function handleCommand(command: string) {
  if (command === 'logout') {
    authStore.logout()
    router.push('/login')
  }
}
</script>

<style scoped>
.user-dropdown {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-lg);
  transition: background-color var(--duration-fast) var(--ease-out);
}

.user-dropdown:hover {
  background-color: var(--color-surface-hover);
}

.username {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}

@media (max-width: 768px) {
  .username { display: none; }
}
</style>

<!-- src/components/layout/Sidebar/index.vue -->
<template>
  <el-aside :width="collapsed ? '64px' : '240px'" class="sidebar" :class="{ 'is-collapsed': collapsed }">
    <div class="logo-container">
      <img src="/app-logo-v1.png" alt="QIM Logo" class="logo-image" />
      <h2 class="logo-text" v-show="!collapsed">{{ adminTitle }}</h2>
    </div>

    <div class="menu-wrapper">
      <el-menu :default-active="activeMenu" :collapse="collapsed" router class="sidebar-menu">
        <el-sub-menu v-for="group in sidebarModuleGroups" :key="group.key" :index="group.key">
          <template #title>
            <el-icon><component :is="group.icon" /></el-icon>
            <span>{{ group.title }}</span>
          </template>
          <el-menu-item
            v-for="item in group.items"
            :key="item.path"
            :index="item.path"
            v-permission="item.permission"
            v-role="item.role"
          >
            <el-icon><component :is="item.icon" /></el-icon>
            <template #title>{{ item.title }}</template>
          </el-menu-item>
        </el-sub-menu>
      </el-menu>
    </div>

    <button class="collapse-btn" @click="$emit('toggle')" :title="collapsed ? '展开' : '收起'">
      <el-icon :size="18">
        <Fold v-if="!collapsed" />
        <Expand v-else />
      </el-icon>
    </button>
  </el-aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Fold, Expand } from '@element-plus/icons-vue'
import { getAdminTitle } from '@/config/appConfig'
import { sidebarModuleGroups } from '@/config/adminModules'

defineEmits<{
  'toggle': []
}>()

defineProps<{
  collapsed: boolean
}>()

const route = useRoute()
const activeMenu = computed(() => route.path)
const adminTitle = getAdminTitle()
</script>

<style scoped>
.sidebar {
  background: var(--sidebar-bg);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
  transition: width var(--duration-normal) var(--ease-out);
  box-shadow: 4px 0 16px rgba(0, 0, 0, 0.08);
  z-index: 10;
  height: 100vh;
}

.logo-container {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 0 var(--space-4);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.logo-image {
  width: 36px;
  height: 36px;
  object-fit: contain;
  flex-shrink: 0;
}

.logo-text {
  color: white;
  font-size: 18px;
  font-weight: 800;
  margin: 0;
  white-space: nowrap;
  letter-spacing: -0.02em;
}

.menu-wrapper {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: var(--space-2) 0;
}

.sidebar-menu {
  background: transparent !important;
  border-right: none !important;
}

:deep(.el-menu-item),
:deep(.el-sub-menu__title) {
  color: var(--sidebar-text) !important;
  height: 44px !important;
  line-height: 44px !important;
  font-weight: 500 !important;
}

:deep(.el-menu-item:hover),
:deep(.el-sub-menu__title:hover) {
  background-color: rgba(255, 255, 255, 0.06) !important;
  color: var(--sidebar-text-active) !important;
}

:deep(.el-menu-item.is-active) {
  background: rgba(255, 255, 255, 0.1) !important;
  color: var(--sidebar-text-active) !important;
  font-weight: 700 !important;
}

:deep(.el-sub-menu .el-menu-item) {
  min-width: auto !important;
  margin: 2px 8px !important;
  background: rgba(255, 255, 255, 0.03) !important;
  border-radius: var(--radius-sm) !important;
}

:deep(.el-sub-menu .el-menu-item:hover) {
  background: rgba(255, 255, 255, 0.08) !important;
}

:deep(.el-sub-menu .el-menu-item.is-active) {
  background: rgba(255, 255, 255, 0.15) !important;
  color: white !important;
}

:deep(.el-sub-menu .el-menu) {
  background: rgba(0, 0, 0, 0.12) !important;
  border-radius: var(--radius-lg);
  margin: 4px 8px;
}

.collapse-btn {
  position: absolute;
  bottom: var(--space-4);
  left: 50%;
  transform: translateX(-50%);
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: var(--radius-md);
  color: rgba(255, 255, 255, 0.7);
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-out);
}

.collapse-btn:hover {
  background: rgba(255, 255, 255, 0.15);
  color: white;
  transform: translateX(-50%) scale(1.05);
}
</style>

<!-- src/layouts/AdminLayout.vue -->
<template>
  <el-container class="admin-layout">
    <Sidebar v-if="!isMobile" :collapsed="isCollapsed" @toggle="isCollapsed = !isCollapsed" />
    <SidebarDrawer :visible="isDrawerOpen" @close="isDrawerOpen = false" />
    <MobileOverlay :visible="isDrawerOpen" @close="isDrawerOpen = false" />
    <el-container class="main-container">
      <Header 
        :show-hamburger="isMobile" 
        :sidebar-open="isDrawerOpen" 
        @toggle-sidebar="isDrawerOpen = !isDrawerOpen"
      >
        <template #breadcrumb>
          <Breadcrumb :title="currentTitle" />
        </template>
      </Header>
      <el-main class="admin-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import Sidebar from '@/components/layout/Sidebar/index.vue'
import Header from '@/components/layout/Header/index.vue'
import Breadcrumb from '@/components/layout/Breadcrumb/index.vue'
import SidebarDrawer from '@/components/layout/SidebarDrawer.vue'
import MobileOverlay from '@/components/layout/MobileOverlay.vue'

const route = useRoute()
const isCollapsed = ref(false)
const isMobile = ref(false)
const isDrawerOpen = ref(false)

const MOBILE_BREAKPOINT = 768

const checkMobile = () => {
  isMobile.value = window.innerWidth <= MOBILE_BREAKPOINT
  if (!isMobile.value) {
    isDrawerOpen.value = false
  }
}

watch(isDrawerOpen, (open) => {
  if (open) {
    document.body.classList.add('mobile-drawer-open')
  } else {
    document.body.classList.remove('mobile-drawer-open')
  }
})

const touchStartX = ref(0)
const touchStartY = ref(0)

const onTouchStart = (e: TouchEvent) => {
  touchStartX.value = e.touches[0].clientX
  touchStartY.value = e.touches[0].clientY
}

const onTouchEnd = (e: TouchEvent) => {
  const deltaX = e.changedTouches[0].clientX - touchStartX.value
  const deltaY = Math.abs(e.changedTouches[0].clientY - touchStartY.value)
  
  // 从左边缘向右滑动打开抽屉
  if (deltaX > 80 && deltaY < 100 && touchStartX.value < 40) {
    isDrawerOpen.value = true
  }
  
  // 从右向左滑动关闭抽屉
  if (deltaX < -80 && deltaY < 100 && isDrawerOpen.value) {
    isDrawerOpen.value = false
  }
}

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
  window.addEventListener('touchstart', onTouchStart, { passive: true })
  window.addEventListener('touchend', onTouchEnd, { passive: true })
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
  window.removeEventListener('touchstart', onTouchStart)
  window.removeEventListener('touchend', onTouchEnd)
})

// 面包屑标题直接从路由 meta.title 读取，避免维护重复的 titleMap
const currentTitle = computed(() => (route.meta.title as string) || '仪表盘')
</script>

<style scoped>
.admin-layout {
  height: 100vh;
  background-color: var(--color-bg-page);
  display: flex;
  overflow: hidden;
}

.main-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background-color: var(--color-bg-page);
  flex: 1;
  min-width: 0;
  padding: var(--space-4);
  gap: var(--space-4);
}

.admin-main {
  background-color: var(--color-surface);
  padding: var(--space-6);
  overflow-y: auto;
  flex: 1;
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);
}

:deep(.el-main) {
  padding: 0;
  display: flex;
  flex-direction: column;
}

:deep(.el-header) {
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);
}

@media (max-width: 768px) {
  .main-container {
    padding: var(--space-2);
    gap: var(--space-2);
  }

  .admin-main {
    padding: var(--space-4);
    border-radius: var(--radius-md);
  }

  :deep(.el-header) {
    border-radius: var(--radius-md);
  }
}
</style>

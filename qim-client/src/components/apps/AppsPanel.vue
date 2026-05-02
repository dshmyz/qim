<template>
  <div class="right-content">
    <div class="right-content-header">
      <div class="header-left-group">
        <button class="toggle-sidebar-btn" @click="$emit('toggleSidebar')">
          <i class="fas fa-compress"></i>
        </button>
        <h2>{{ pageTitle }}</h2>
      </div>
    </div>
    <div class="apps-content">
      <!-- 快速工具区域 -->
      <div v-if="quickTools && quickTools.length > 0" class="quick-tools-section">
        <div class="section-header">
          <h3>快速工具</h3>
          <span class="section-badge">常用</span>
        </div>
        <div class="quick-tools-grid">
          <div
            v-for="app in quickTools"
            :key="app.id"
            class="quick-tool-item"
            @click="$emit('openApp', app.id)"
          >
            <div class="quick-tool-icon"><i :class="app.icon"></i></div>
            <div class="quick-tool-info">
              <span class="quick-tool-name">{{ app.name }}</span>
              <span class="quick-tool-desc">{{ app.description || '快速访问' }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="recent-apps-section">
        <div class="section-header">
          <h3>最近使用</h3>
        </div>
        <div class="recent-apps-grid">
          <div
            v-for="app in recentApps"
            :key="app.id"
            class="recent-app-grid-item"
            @click="$emit('openApp', app.id)"
          >
            <div class="recent-app-grid-icon"><i :class="app.icon"></i></div>
            <span class="recent-app-grid-name">{{ app.name }}</span>
          </div>
          <div v-if="recentApps.length === 0" class="empty-recent-apps">
            <p>暂无最近使用的应用</p>
          </div>
        </div>
      </div>

      <div class="all-apps-section">
        <div class="section-header">
          <h3>所有应用</h3>
        </div>
        <div class="main-apps-grid">
          <div
            v-for="app in allApps"
            :key="app.id"
            class="main-app-item"
            @click="$emit('openApp', app.id)"
          >
            <div class="main-app-icon"><i :class="app.icon"></i></div>
            <div class="main-app-name">{{ app.name }}</div>
          </div>
          <div v-if="allApps.length === 0" class="empty-all-apps">
            <p>暂无应用</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface App {
  id: string
  name: string
  icon: string
  url?: string
  description?: string
}

interface Props {
  recentApps: App[]
  allApps: App[]
  pageTitle: string
  quickTools?: App[]
}

defineProps<Props>()

defineEmits<{
  'toggleSidebar': []
  'openApp': [appId: string]
}>()
</script>

<style scoped>
.right-content {
  flex: 1;
  background: var(--right-content-bg, #f5f5f5);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.right-content-header {
  padding: 16px 20px;
  background: var(--right-content-header-bg, #fff);
  height: 72px;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.right-content-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 500;
  color: var(--text-color, #333);
}

.header-left-group {
  display: flex;
  align-items: center;
  gap: 12px;
}

.toggle-sidebar-btn {
  background: none;
  border: none;
  cursor: pointer;
  padding: 8px;
  color: var(--text-color, #333);
}

.apps-content {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.quick-tools-section,
.recent-apps-section,
.all-apps-section {
  margin-bottom: 24px;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.section-header h3 {
  margin: 0;
  font-size: 16px;
  color: var(--text-color, #333);
}

.section-badge {
  padding: 2px 8px;
  background: var(--primary-color, #409eff);
  color: white;
  font-size: 11px;
  border-radius: 10px;
}

.quick-tools-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
}

.quick-tool-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: var(--card-bg, white);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
  border: 1px solid var(--border-color, transparent);
  box-shadow: 0 1px 3px var(--shadow-color, rgba(0, 0, 0, 0.1));
}

.quick-tool-item:hover {
  background: var(--hover-color, #f0f0f0);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px var(--shadow-color, rgba(0, 0, 0, 0.15));
}

.quick-tool-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--primary-color, #409eff), var(--accent-color, #667eea));
  border-radius: 10px;
  color: white;
  font-size: 18px;
}

.quick-tool-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.quick-tool-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color, #333);
}

.quick-tool-desc {
  font-size: 12px;
  color: var(--text-secondary, #999);
}

.recent-apps-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  gap: 16px;
}

.recent-app-grid-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.recent-app-grid-item:hover {
  background: var(--hover-color, #f0f0f0);
}

.recent-app-grid-icon {
  font-size: 32px;
  margin-bottom: 8px;
  color: var(--primary-color, #409eff);
}

.recent-app-grid-name {
  font-size: 12px;
  text-align: center;
  color: var(--text-color, #333);
}

.empty-recent-apps,
.empty-all-apps {
  grid-column: 1 / -1;
  text-align: center;
  padding: 40px 0;
  color: var(--text-secondary, #999);
}

.main-apps-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  gap: 16px;
}

.main-app-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.main-app-item:hover {
  background: var(--hover-color, #f0f0f0);
}

.main-app-icon {
  font-size: 32px;
  margin-bottom: 8px;
  color: var(--primary-color, #409eff);
}

.main-app-name {
  font-size: 12px;
  text-align: center;
  color: var(--text-color, #333);
}
</style>

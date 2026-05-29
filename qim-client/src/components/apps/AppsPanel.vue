<template>
  <div class="right-content">
    <div class="panel-header">
      <div class="header-left-group">
        <ToggleSidebarBtn
          icon="fas fa-compress"
          title="收起侧边栏"
          @click="$emit('toggleSidebar')"
        />
        <h2>{{ pageTitle }}</h2>
      </div>
    </div>
    <div class="apps-content">
      <!-- 主要应用区域 -->
      <div class="main-apps-section">
        <div class="section-header">
          <h3>主要应用</h3>
          <span class="section-badge">核心功能</span>
        </div>
        <div class="main-apps-grid">
          <div
            v-for="app in mainApps"
            :key="app.id"
            class="main-app-item"
            @click="$emit('openApp', app.id)"
          >
            <div class="main-app-icon"><i :class="app.icon"></i></div>
            <div class="main-app-name">{{ app.name }}</div>
          </div>
          <div v-if="mainApps.length === 0" class="empty-apps">
            <p>暂无主要应用</p>
          </div>
        </div>
      </div>

      <!-- 自定义应用区域 -->
      <div v-if="customApps && customApps.length > 0" class="custom-apps-section">
        <div class="section-header">
          <h3>自定义应用</h3>
          <span class="section-badge">{{ customApps.length }}个</span>
        </div>
        <div class="custom-apps-grid">
          <div
            v-for="app in customApps"
            :key="app.id"
            class="custom-app-item"
            @click="$emit('openApp', app.id)"
          >
            <div class="custom-app-icon"><i :class="app.icon"></i></div>
            <div class="custom-app-name">{{ app.name }}</div>
            <div v-if="app.url" class="custom-app-url">{{ app.url }}</div>
          </div>
        </div>
      </div>

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
            :class="['quick-tool-item', { 'quick-tool-highlight': app.id === 'short_link' }]"
            @click="$emit('openApp', app.id)"
          >
            <div class="quick-tool-icon"><i :class="app.icon"></i></div>
            <div class="quick-tool-info">
              <span class="quick-tool-name">{{ app.name }}</span>
              <span class="quick-tool-desc">{{ app.description || '快速访问' }}</span>
            </div>
            <div v-if="app.id === 'short_link'" class="quick-tool-badge">快速工具</div>
          </div>
        </div>
      </div>

      <!-- 系统区域 -->
      <div class="system-apps-section">
        <div class="section-header">
          <h3>系统</h3>
        </div>
        <div class="system-apps-grid">
          <div
            v-for="app in systemApps"
            :key="app.id"
            class="system-app-item"
            @click="$emit('openApp', app.id)"
          >
            <div class="system-app-icon"><i :class="app.icon"></i></div>
            <div class="system-app-name">{{ app.name }}</div>
          </div>
          <div v-if="systemApps.length === 0" class="empty-apps">
            <p>暂无系统应用</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import ToggleSidebarBtn from '../shared/ToggleSidebarBtn.vue'

interface App {
  id: string
  name: string
  icon: string
  url?: string
  description?: string
}

interface Props {
  mainApps: App[]
  quickTools?: App[]
  customApps?: App[]
  systemApps: App[]
  pageTitle: string
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

.panel-header {
  padding: 0 20px;
  height: 56px;
  background: var(--right-content-header-bg, #fff);
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  flex-shrink: 0;
}

.panel-header h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color, #333);
}

.header-left-group {
  display: flex;
  align-items: center;
  gap: 12px;
}

.apps-content {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.main-apps-section,
.quick-tools-section,
.system-apps-section {
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

/* 主要应用样式 */
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
  background: var(--card-bg, white);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
  border: 1px solid var(--border-color, transparent);
  box-shadow: 0 1px 3px var(--shadow-color, rgba(0, 0, 0, 0.1));
}

.main-app-item:hover {
  background: var(--hover-color, #f0f0f0);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px var(--shadow-color, rgba(0, 0, 0, 0.15));
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

/* 快速工具样式 */
.quick-tools-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
}

.quick-tool-item {
  position: relative;
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

/* 短链接管理突出显示样式 */
.quick-tool-highlight {
  background: var(--card-bg, #fff) !important;
  border: 2px solid var(--primary-color, #409eff) !important;
}

.quick-tool-highlight:hover {
  background: var(--hover-color, #f5f5f5) !important;
  border-color: var(--active-color, #66b1ff) !important;
}

.quick-tool-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--primary-color, #409eff);
  border-radius: 10px;
  color: white;
  font-size: 18px;
}

.quick-tool-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
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

.quick-tool-badge {
  position: absolute;
  top: 8px;
  right: 8px;
  padding: 2px 8px;
  background: var(--primary-color, #409eff);
  color: white;
  font-size: 10px;
  border-radius: 8px;
  font-weight: 500;
}

/* 自定义应用样式 */
.custom-apps-section {
  margin-bottom: 24px;
}

.custom-apps-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  gap: 16px;
}

.custom-app-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px;
  background: var(--card-bg, white);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
  border: 1px solid var(--border-color, transparent);
  box-shadow: 0 1px 3px var(--shadow-color, rgba(0, 0, 0, 0.1));
}

.custom-app-item:hover {
  background: var(--hover-color, #f0f0f0);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px var(--shadow-color, rgba(0, 0, 0, 0.15));
}

.custom-app-icon {
  font-size: 32px;
  margin-bottom: 8px;
  color: var(--primary-color, #409eff);
}

.custom-app-name {
  font-size: 12px;
  text-align: center;
  color: var(--text-color, #333);
}

.custom-app-url {
  font-size: 10px;
  text-align: center;
  color: var(--text-secondary, #999);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
  margin-top: 2px;
}

/* 系统应用样式 */
.system-apps-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  gap: 16px;
}

.system-app-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px;
  background: var(--card-bg, white);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
  border: 1px solid var(--border-color, transparent);
  box-shadow: 0 1px 3px var(--shadow-color, rgba(0, 0, 0, 0.1));
}

.system-app-item:hover {
  background: var(--hover-color, #f0f0f0);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px var(--shadow-color, rgba(0, 0, 0, 0.15));
}

.system-app-icon {
  font-size: 32px;
  margin-bottom: 8px;
  color: var(--text-secondary, #999);
}

.system-app-name {
  font-size: 12px;
  text-align: center;
  color: var(--text-color, #333);
}

.empty-apps {
  grid-column: 1 / -1;
  text-align: center;
  padding: 40px 0;
  color: var(--text-secondary, #999);
}
</style>

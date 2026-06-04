<!-- src/components/layout/Sidebar/index.vue -->
<template>
  <el-aside :width="collapsed ? '64px' : '240px'" class="sidebar" :class="{ 'is-collapsed': collapsed }">
    <div class="logo-container">
      <img src="/app-logo-v1.png" alt="QIM Logo" class="logo-image" />
      <h2 class="logo-text" v-show="!collapsed">{{ adminTitle }}</h2>
    </div>

    <div class="menu-wrapper">
      <el-menu :default-active="activeMenu" :collapse="collapsed" router class="sidebar-menu">
        <el-sub-menu index="dashboard-group">
          <template #title>
            <el-icon><DataAnalysis /></el-icon>
            <span>数据概览</span>
          </template>
          <el-menu-item index="/">
            <el-icon><HomeFilled /></el-icon>
            <template #title>仪表盘</template>
          </el-menu-item>
          <el-menu-item index="/statistics">
            <el-icon><TrendCharts /></el-icon>
            <template #title>数据统计</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="user-group">
          <template #title>
            <el-icon><User /></el-icon>
            <span>用户与组织</span>
          </template>
          <el-menu-item index="/users" v-permission="'user:read'">
            <el-icon><User /></el-icon>
            <template #title>用户管理</template>
          </el-menu-item>
          <el-menu-item index="/organization" v-permission="'organization:read'">
            <el-icon><School /></el-icon>
            <template #title>组织架构</template>
          </el-menu-item>
          <el-menu-item index="/roles" v-permission="'role:read'">
            <el-icon><Key /></el-icon>
            <template #title>角色权限</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="chat-group">
          <template #title>
            <el-icon><ChatDotRound /></el-icon>
            <span>会话与群组</span>
          </template>
          <el-menu-item index="/groups" v-permission="'group:read'">
            <el-icon><UserFilled /></el-icon>
            <template #title>群组管理</template>
          </el-menu-item>
          <el-menu-item index="/conversations" v-permission="'conversation:read'">
            <el-icon><ChatLineSquare /></el-icon>
            <template #title>会话管理</template>
          </el-menu-item>
          <el-menu-item index="/channels" v-permission="'channel:read'">
            <el-icon><Connection /></el-icon>
            <template #title>频道管理</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="app-group">
          <template #title>
            <el-icon><Grid /></el-icon>
            <span>应用生态</span>
          </template>
          <el-menu-item index="/apps" v-permission="'app:read'">
            <el-icon><Monitor /></el-icon>
            <template #title>应用管理</template>
          </el-menu-item>
          <el-menu-item index="/mini-apps" v-permission="'miniapp:read'">
            <el-icon><Cellphone /></el-icon>
            <template #title>小程序管理</template>
          </el-menu-item>
          <el-menu-item index="/ai-assistant" v-permission="'ai:read'">
            <el-icon><Cpu /></el-icon>
            <template #title>AI 助手</template>
          </el-menu-item>
          <el-menu-item index="/ai-ops" v-permission="'ai:read'">
            <el-icon><Monitor /></el-icon>
            <template #title>AI 运维面板</template>
          </el-menu-item>
          <el-menu-item index="/ai-config" v-permission="'ai:read'">
            <el-icon><Setting /></el-icon>
            <template #title>AI 模型配置</template>
          </el-menu-item>
          <el-menu-item index="/mcp-tools" v-permission="'ai:read'">
            <el-icon><Tools /></el-icon>
            <template #title>MCP 工具管理</template>
          </el-menu-item>
          <el-menu-item index="/knowledge-graph" v-permission="'ai:read'">
            <el-icon><Connection /></el-icon>
            <template #title>知识图谱</template>
          </el-menu-item>
          <el-menu-item index="/vector-data" v-permission="'ai:read'">
            <el-icon><DataAnalysis /></el-icon>
            <template #title>向量数据</template>
          </el-menu-item>
          <el-menu-item index="/approvals" v-permission="'ai:read'">
            <el-icon><Checked /></el-icon>
            <template #title>审批管理</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="msg-group">
          <template #title>
            <el-icon><Bell /></el-icon>
            <span>消息与通知</span>
          </template>
          <el-menu-item index="/messages" v-permission="'message:read'">
            <el-icon><ChatDotRound /></el-icon>
            <template #title>系统消息</template>
          </el-menu-item>
          <el-menu-item index="/message-search" v-permission="'message:read'">
            <el-icon><Search /></el-icon>
            <template #title>消息搜索</template>
          </el-menu-item>
          <el-menu-item index="/notifications" v-permission="'notification:read'">
            <el-icon><BellFilled /></el-icon>
            <template #title>通知管理</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="security-group">
          <template #title>
            <el-icon><Lock /></el-icon>
            <span>安全与合规</span>
          </template>
          <el-menu-item index="/blacklist" v-permission="'blacklist:read'">
            <el-icon><CircleCloseFilled /></el-icon>
            <template #title>黑名单管理</template>
          </el-menu-item>
          <el-menu-item index="/sensitive-words" v-permission="'sensitive:read'">
            <el-icon><Warning /></el-icon>
            <template #title>敏感词管理</template>
          </el-menu-item>
          <el-menu-item index="/operation-logs" v-permission="'log:read'">
            <el-icon><Document /></el-icon>
            <template #title>操作日志</template>
          </el-menu-item>
          <el-menu-item index="/feedbacks" v-permission="'feedback:read'">
            <el-icon><Message /></el-icon>
            <template #title>意见反馈</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="system-group">
          <template #title>
            <el-icon><Setting /></el-icon>
            <span>系统设置</span>
          </template>
          <el-menu-item index="/system-config" v-permission="'config:read'">
            <el-icon><Tools /></el-icon>
            <template #title>系统配置</template>
          </el-menu-item>
          <el-menu-item index="/auth-config" v-permission="'auth:read'">
            <el-icon><Key /></el-icon>
            <template #title>认证配置</template>
          </el-menu-item>
          <el-menu-item index="/org-sync" v-permission="'org:read'">
            <el-icon><Connection /></el-icon>
            <template #title>组织架构同步</template>
          </el-menu-item>
          <el-menu-item index="/version-management" v-permission="'version:read'">
            <el-icon><Upload /></el-icon>
            <template #title>版本管理</template>
          </el-menu-item>
          <el-menu-item index="/file-storage" v-permission="'file:read'">
            <el-icon><Folder /></el-icon>
            <template #title>文件存储管理</template>
          </el-menu-item>
          <el-menu-item index="/server-monitor" v-permission="'monitor:read'">
            <el-icon><Monitor /></el-icon>
            <template #title>服务器监控</template>
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
import {
  HomeFilled, User, UserFilled, ChatDotRound, Bell,
  CircleCloseFilled, TrendCharts, School, ChatLineSquare,
  Connection, Grid, Monitor, Cellphone, BellFilled,
  Fold, Expand, DataAnalysis, Key, Cpu, Warning, Document,
  Lock, Setting, Tools, Upload, Search, Folder, Checked,
  Message,
} from '@element-plus/icons-vue'
import { getProductName, getAdminTitle } from '@/config/appConfig'

defineEmits<{
  'toggle': []
}>()

defineProps<{
  collapsed: boolean
}>()

const route = useRoute()
const activeMenu = computed(() => route.path)
const productName = getProductName()
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

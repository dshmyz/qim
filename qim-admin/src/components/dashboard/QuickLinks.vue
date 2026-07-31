<template>
  <div class="quick-links-section">
    <div class="section-header">
      <h3 class="section-title">快速入口</h3>
    </div>
    <div class="links-grid">
      <div
        v-for="link in links"
        :key="link.path"
        class="link-card"
        @click="navigateTo(link.path)"
      >
        <div class="link-icon" :style="{ background: link.gradient }">
          <el-icon :size="20"><component :is="link.icon" /></el-icon>
        </div>
        <div class="link-info">
          <span class="link-title">{{ link.title }}</span>
          <span class="link-desc">{{ link.description }}</span>
        </div>
        <el-icon class="link-arrow"><ArrowRight /></el-icon>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import {
  ArrowRight,
  Connection,
  Document,
  Monitor,
  Tools,
} from '@element-plus/icons-vue'
import type { Component } from 'vue'

interface QuickLink {
  title: string
  description: string
  path: string
  icon: Component
  gradient: string
}

const router = useRouter()

const links: QuickLink[] = [
  {
    title: 'CLI 使用指南',
    description: '命令行工具安装、配置与消息收发',
    path: '/docs/cli',
    icon: Document,
    gradient: 'linear-gradient(135deg, #0ea5e9 0%, #06b6d4 100%)',
  },
  {
    title: 'MCP 接入指南',
    description: 'Claude Code / Cursor 标准协议接入',
    path: '/docs/mcp',
    icon: Connection,
    gradient: 'linear-gradient(135deg, #8b5cf6 0%, #a78bfa 100%)',
  },
  {
    title: 'AI 工具管理',
    description: '查看和管理已注册的 AI 工具',
    path: '/ai-tools',
    icon: Tools,
    gradient: 'linear-gradient(135deg, #f59e0b 0%, #fbbf24 100%)',
  },
  {
    title: 'Bot 运维',
    description: 'Bot 投递监控与令牌管理',
    path: '/bot-ops',
    icon: Monitor,
    gradient: 'linear-gradient(135deg, #10b981 0%, #34d399 100%)',
  },
]

const navigateTo = (path: string) => {
  router.push(path)
}
</script>

<style scoped>
.quick-links-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.section-title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.links-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--space-3);
}

.link-card {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-4);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--radius-xl);
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.link-card:hover {
  border-color: var(--el-color-primary-light-5);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  transform: translateY(-1px);
}

.link-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  color: white;
  flex-shrink: 0;
}

.link-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.link-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.3;
}

.link-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.link-arrow {
  color: var(--el-text-color-placeholder);
  flex-shrink: 0;
  transition: transform 0.2s ease;
}

.link-card:hover .link-arrow {
  color: var(--el-color-primary);
  transform: translateX(2px);
}

@media (max-width: 640px) {
  .links-grid {
    grid-template-columns: 1fr;
  }
}
</style>

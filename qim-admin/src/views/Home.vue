<template>
  <div class="landing-page">
    <!-- Hero Section -->
    <section class="hero">
      <div class="hero-content">
        <div class="logo">
          <img src="/app-logo-v1.png" alt="QIM" class="logo-img" />
        </div>
        <h1 class="title">QIM <span class="title-cn">青雀</span></h1>
        <p class="subtitle">企业级智能协作平台</p>
        <p class="subtitle-desc">集成 AI 能力的现代化企业通讯解决方案，让团队协作更高效、更智能</p>
        
        <div class="feature-tags">
          <span class="tag">AI 驱动</span>
          <span class="tag">多平台支持</span>
        </div>

        <div class="hero-actions">
          <el-button size="large" @click="scrollToFeatures">了解更多</el-button>
          <el-button type="primary" size="large" @click="downloadClient">
            <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" v-html="downloadIcon"></svg>
            {{ downloadLabel }}
          </el-button>
        </div>
      </div>
      <div class="hero-bg"></div>
    </section>

    <!-- Features Section -->
    <section class="features" id="features">
      <h2 class="section-title">核心功能</h2>
      <div class="features-grid">
        <div class="feature-card" v-for="feature in features" :key="feature.title">
          <div class="feature-icon" :style="{ background: feature.color }">
            <svg class="feature-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" v-html="feature.icon"></svg>
          </div>
          <h3>{{ feature.title }}</h3>
          <p>{{ feature.desc }}</p>
        </div>
      </div>
    </section>

    <!-- Built-in Apps Section -->
    <section class="apps" id="apps">
      <h2 class="section-title">内置应用</h2>
      <p class="apps-desc">开箱即用的生产力工具，满足日常办公需求</p>
      <div class="apps-grid">
        <div class="app-card" v-for="app in apps" :key="app.name">
          <div class="app-icon" :style="{ background: app.color }">
            <svg class="app-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" v-html="app.icon"></svg>
          </div>
          <h3>{{ app.name }}</h3>
          <p>{{ app.desc }}</p>
        </div>
      </div>
    </section>

    <!-- Platforms Section -->
    <section class="platforms" id="platforms">
      <h2 class="section-title">多平台支持</h2>
      <p class="platforms-desc">支持主流操作系统，随时随地接入协作</p>
      <div class="platforms-grid">
        <div class="platform-card" v-for="platform in platforms" :key="platform.name">
          <div class="platform-icon" :style="{ background: platform.color }">
            <svg class="platform-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" v-html="platform.icon"></svg>
          </div>
          <h3>{{ platform.name }}</h3>
          <div class="platform-downloads">
            <a 
              v-for="dl in platform.downloads" 
              :key="dl.label"
              :href="dl.url"
              class="download-link"
              target="_blank"
            >
              <svg class="dl-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" v-html="platform.dlIcon"></svg>
              {{ dl.label }}
            </a>
          </div>
        </div>
      </div>
    </section>

    <!-- About Section -->
    <section class="about" id="about">
      <h2 class="section-title">关于 QIM 青雀</h2>
      <div class="about-content">
        <ul class="about-list">
          <li>QIM 青雀是一款面向企业团队的智能协作平台，致力于将即时通讯、AI 能力与办公应用深度融合。</li>
          <li>通过内置的文件箱、笔记、任务管理、日历等应用，团队可以在一个平台内完成沟通、协作与项目管理，无需在多个工具之间切换。</li>
          <li>同时，青雀支持自定义 AI 助手和数字分身，让 AI 真正成为团队的生产力伙伴。</li>
        </ul>
      </div>
    </section>

    <!-- Footer -->
    <footer class="footer">
      <div class="footer-content">
        <p>&copy; {{ new Date().getFullYear() }} QIM 青雀. All rights reserved.</p>
        <p class="footer-meta">
          <span class="footer-version">v1.2.0 · 2026-05-09 更新</span>
          <span class="footer-meta-divider">|</span>
          <svg class="contact-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
            <polyline points="22,6 12,13 2,6"/>
          </svg>
          联系作者：<a href="mailto:huangqun@buaa.edu.cn">huangqun@buaa.edu.cn</a>
        </p>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const currentPlatform = computed(() => {
  const ua = navigator.userAgent
  if (/Windows NT/.test(ua)) return 'windows'
  if (/Macintosh|Mac OS X/.test(ua)) return 'macos'
  if (/Linux/.test(ua)) return 'linux'
  return 'windows'
})

const downloadIcon = computed(() => {
  const p = currentPlatform.value
  if (p === 'windows') return '<rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/>'
  if (p === 'macos') return '<path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.8-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z"/>'
  return '<circle cx="12" cy="12" r="10"/><path d="M8 14s1.5 2 4 2 4-2 4-2"/><line x1="9" y1="9" x2="9.01" y2="9"/><line x1="15" y1="9" x2="15.01" y2="9"/>'
})

const downloadLabel = computed(() => {
  const p = currentPlatform.value
  if (p === 'windows') return '下载 Windows 版 (.exe)'
  if (p === 'macos') return '下载 macOS 版 (.dmg)'
  return '下载 Linux 版 (.deb)'
})

const features = [
  {
    icon: '<path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>',
    title: '即时通讯',
    desc: '支持单聊、群聊、频道等多种通讯模式，消息实时同步，支持 @ 提醒与已读回执',
    color: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
  },
  {
    icon: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>',
    title: 'AI 助手',
    desc: '内置 AI 工作台，支持智能问答、内容生成、代码辅助，可自定义 AI 模型与数字分身',
    color: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
  },
  {
    icon: '<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/>',
    title: '组织架构',
    desc: '完整的企业组织架构管理，部门、员工、角色权限一目了然，支持审批流程',
    color: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
  },
  {
    icon: '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/>',
    title: '文件协作',
    desc: '内置文件箱应用，支持文件上传、分享、在线预览、版本管理，团队协作更高效',
    color: 'linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)',
  },
  {
    icon: '<path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/>',
    title: '消息通知',
    desc: '系统消息、@ 提醒、审批通知等多维度消息推送，确保重要信息不遗漏',
    color: 'linear-gradient(135deg, #fa709a 0%, #fee140 100%)',
  },
  {
    icon: '<rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>',
    title: '安全管控',
    desc: '黑名单、敏感词过滤、操作审计日志等企业级安全管控能力，保障数据安全',
    color: 'linear-gradient(135deg, #a8edea 0%, #fed6e3 100%)',
  },
]

const apps = [
  {
    icon: '<path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>',
    name: '文件箱',
    desc: '文件存储、分享、在线预览，支持文件夹管理与版本控制',
    color: 'linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)',
  },
  {
    icon: '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/>',
    name: '笔记',
    desc: 'Markdown 笔记编辑，支持 AI 分析、全文搜索、导入导出',
    color: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
  },
  {
    icon: '<rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/>',
    name: '便签',
    desc: '快速记录灵感与待办，支持多色标签、AI 分析与消息转发',
    color: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
  },
  {
    icon: '<path d="M9 11l3 3L22 4"/><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/>',
    name: '任务管理',
    desc: '看板视图管理任务，支持优先级、截止日期、任务分配与进度追踪',
    color: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
  },
  {
    icon: '<rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/>',
    name: '日历',
    desc: '日程安排与事件管理，支持创建、编辑、提醒，与团队共享日程',
    color: 'linear-gradient(135deg, #fa709a 0%, #fee140 100%)',
  },
  {
    icon: '<path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>',
    name: '短链接',
    desc: '生成与管理短链接，支持批量生成、自定义别名、访问统计与导出',
    color: 'linear-gradient(135deg, #a8edea 0%, #fed6e3 100%)',
  },
]

const platforms = [
  {
    name: 'Windows',
    icon: '<rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/>',
    dlIcon: '<rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/>',
    color: 'linear-gradient(135deg, #0078d4 0%, #005a9e 100%)',
    downloads: [
      { label: 'QIM x64.exe', url: '/downloads/qim-windows-x64.exe' },
      { label: 'QIM x86.exe', url: '/downloads/qim-windows-x86.exe' },
    ],
  },
  {
    name: 'macOS',
    icon: '<path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.8-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z"/>',
    dlIcon: '<path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.8-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z"/>',
    color: 'linear-gradient(135deg, #333 0%, #555 100%)',
    downloads: [
      { label: 'QIM Intel.dmg', url: '/downloads/qim-macos-intel.dmg' },
      { label: 'QIM Apple Silicon.dmg', url: '/downloads/qim-macos-arm.dmg' },
    ],
  },
  {
    name: 'Linux',
    icon: '<circle cx="12" cy="12" r="10"/><path d="M8 14s1.5 2 4 2 4-2 4-2"/><line x1="9" y1="9" x2="9.01" y2="9"/><line x1="15" y1="9" x2="15.01" y2="9"/>',
    dlIcon: '<circle cx="12" cy="12" r="10"/><path d="M8 14s1.5 2 4 2 4-2 4-2"/><line x1="9" y1="9" x2="9.01" y2="9"/><line x1="15" y1="9" x2="15.01" y2="9"/>',
    color: 'linear-gradient(135deg, #e95420 0%, #772953 100%)',
    downloads: [
      { label: 'QIM .deb', url: '/downloads/qim-linux.deb' },
      { label: 'QIM .rpm', url: '/downloads/qim-linux.rpm' },
    ],
  },
]

const scrollToFeatures = () => {
  document.getElementById('features')?.scrollIntoView({ behavior: 'smooth' })
}

const downloadClient = () => {
  const p = currentPlatform.value
  let url = ''
  
  if (p === 'windows') {
    url = '/downloads/qim-windows-x64.exe'
  } else if (p === 'macos') {
    if (/Apple Silicon|arm64|aarch64/.test(navigator.userAgent)) {
      url = '/downloads/qim-macos-arm.dmg'
    } else {
      url = '/downloads/qim-macos-intel.dmg'
    }
  } else {
    url = '/downloads/qim-linux.deb'
  }
  
  window.open(url, '_blank')
}
</script>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Noto+Serif+SC:wght@400;600;700&display=swap');

.landing-page {
  min-height: 100vh;
  background: #fafbfc;
  overflow-y: auto;
  overflow-x: hidden;
}

:global(body) {
  overflow: auto !important;
}

.hero {
  position: relative;
  min-height: 700px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%);
  color: #fff;
  overflow: hidden;
}

.hero-content {
  position: relative;
  z-index: 1;
  text-align: center;
  max-width: 800px;
  padding: 0 24px;
}

.hero-bg {
  position: absolute;
  inset: 0;
  background: 
    radial-gradient(circle at 20% 50%, rgba(120, 119, 198, 0.3) 0%, transparent 50%),
    radial-gradient(circle at 80% 20%, rgba(255, 119, 198, 0.2) 0%, transparent 50%),
    radial-gradient(circle at 40% 80%, rgba(120, 219, 255, 0.2) 0%, transparent 50%);
}

.logo {
  margin-bottom: 32px;
}

.logo-img {
  width: 160px;
  height: 160px;
  border-radius: 36px;
}

.title {
  font-size: 64px;
  font-weight: 700;
  margin: 0 0 8px;
  letter-spacing: -1px;
}

.title-cn {
  font-family: 'Noto Serif SC', serif;
  font-weight: 600;
  background: linear-gradient(135deg, #2d6a4f 0%, #52b788 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.subtitle {
  font-size: 24px;
  color: rgba(255, 255, 255, 0.9);
  margin: 0 0 12px;
  font-weight: 500;
}

.subtitle-desc {
  font-size: 18px;
  color: rgba(255, 255, 255, 0.7);
  margin: 0 0 32px;
  line-height: 1.6;
}

.feature-tags {
  display: flex;
  gap: 10px;
  justify-content: center;
  margin-bottom: 32px;
  flex-wrap: wrap;
}

.tag {
  padding: 4px 14px;
  background: rgba(82, 183, 136, 0.2);
  border: 1px solid rgba(82, 183, 136, 0.4);
  border-radius: 20px;
  font-size: 13px;
  color: #95d5b2;
}

.hero-actions {
  display: flex;
  gap: 16px;
  justify-content: center;
}

.hero-actions .el-button {
  min-width: 180px;
}

.btn-icon {
  width: 18px;
  height: 18px;
  margin-right: 6px;
  vertical-align: -3px;
}

.features,
.apps,
.platforms,
.about {
  padding: 80px 24px;
  max-width: 1200px;
  margin: 0 auto;
}

.apps {
  background: #fff;
  max-width: 100%;
}

.apps .section-title,
.apps .apps-desc {
  max-width: 1200px;
  margin-left: auto;
  margin-right: auto;
}

.apps .apps-desc {
  margin-top: -32px;
}

.apps-grid {
  max-width: 1200px;
  margin: 0 auto;
}

.section-title {
  text-align: center;
  font-size: 36px;
  font-weight: 700;
  color: #1a1a2e;
  margin: 0 0 48px;
}

.apps-desc,
.platforms-desc {
  text-align: center;
  font-size: 16px;
  color: #666;
  margin: -32px 0 48px;
}

.features-grid,
.apps-grid,
.platforms-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 32px;
}

.platforms-grid {
  max-width: 900px;
  margin: 0 auto;
}

.feature-card,
.app-card {
  background: #fff;
  border-radius: 16px;
  padding: 32px;
  text-align: center;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.06);
  transition: transform 0.3s, box-shadow 0.3s;
}

.app-card {
  border: 1px solid #f0f0f0;
}

.feature-card:hover,
.app-card:hover {
  transform: translateY(-8px);
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.12);
}

.feature-icon,
.app-icon {
  width: 80px;
  height: 80px;
  border-radius: 20px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 20px;
}

.feature-svg,
.app-svg {
  width: 40px;
  height: 40px;
  color: #fff;
}

.feature-card h3,
.app-card h3 {
  font-size: 20px;
  font-weight: 600;
  color: #1a1a2e;
  margin: 0 0 12px;
}

.feature-card p,
.app-card p {
  font-size: 14px;
  color: #666;
  line-height: 1.6;
  margin: 0;
}

.platform-card {
  text-align: center;
  padding: 32px;
  border-radius: 16px;
  border: 1px solid #eee;
  transition: border-color 0.3s;
}

.platform-card:hover {
  border-color: #667eea;
}

.platform-icon {
  width: 64px;
  height: 64px;
  border-radius: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 16px;
}

.platform-svg {
  width: 36px;
  height: 36px;
  color: #fff;
}

.platform-card h3 {
  font-size: 20px;
  font-weight: 600;
  color: #1a1a2e;
  margin: 0 0 20px;
}

.platform-downloads {
  display: flex;
  gap: 12px;
  justify-content: center;
  flex-wrap: wrap;
}

.download-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: #f5f7fa;
  border-radius: 8px;
  color: #333;
  font-size: 13px;
  text-decoration: none;
  transition: background 0.2s, color 0.2s;
}

.download-link:hover {
  background: #667eea;
  color: #fff;
}

.dl-icon {
  width: 14px;
  height: 14px;
}

.about {
  padding: 80px 24px;
  background: #fff;
  max-width: 100%;
}

.about .section-title {
  margin-bottom: 32px;
}

.about-content {
  max-width: 700px;
  margin: 0 auto;
}

.about-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.about-list li {
  position: relative;
  padding-left: 24px;
  font-size: 16px;
  color: #555;
  line-height: 1.8;
  margin-bottom: 16px;
}

.about-list li:last-child {
  margin-bottom: 0;
}

.about-list li::before {
  content: '';
  position: absolute;
  left: 0;
  top: 10px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: linear-gradient(135deg, #2d6a4f 0%, #52b788 100%);
}

.footer {
  padding: 32px 24px;
  background: #1a1a2e;
  color: rgba(255, 255, 255, 0.6);
  text-align: center;
}

.footer-content p {
  margin: 0 0 8px;
  font-size: 13px;
}

.footer-content p:last-child {
  margin-bottom: 0;
}

.footer-meta {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.footer-version {
  color: rgba(255, 255, 255, 0.5);
  font-size: 12px;
}

.footer-meta-divider {
  color: rgba(255, 255, 255, 0.3);
}

.contact-icon {
  width: 14px;
  height: 14px;
  vertical-align: -2px;
}

.footer-meta a {
  color: rgba(255, 255, 255, 0.8);
  text-decoration: none;
}

.footer-meta a:hover {
  color: #52b788;
}

@media (max-width: 768px) {
  .title {
    font-size: 40px;
  }

  .subtitle {
    font-size: 18px;
  }

  .subtitle-desc {
    font-size: 14px;
  }

  .features-grid,
  .apps-grid,
  .platforms-grid {
    grid-template-columns: 1fr;
  }

  .hero-actions {
    flex-direction: column;
    align-items: center;
  }

  .footer-meta {
    flex-direction: column;
    gap: 4px;
  }
}
</style>
  font-size: 18px;
  color: rgba(255, 255, 255, 0.7);
  margin: 0 0 32px;
  line-height: 1.6;
}

.subtitle-desc {

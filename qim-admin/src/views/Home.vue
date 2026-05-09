<template>
  <div class="landing-page">
    <!-- Hero Section -->
    <section class="hero">
      <div class="hero-content">
        <div class="logo">
          <div class="logo-icon">QIM</div>
        </div>
        <h1 class="title">企业级智能协作平台</h1>
        <p class="subtitle">集成 AI 能力的现代化企业通讯解决方案，让团队协作更高效、更智能</p>
        <div class="hero-actions">
          <el-button type="primary" size="large" @click="goToLogin">立即体验</el-button>
          <el-button size="large" @click="scrollToFeatures">了解更多</el-button>
          <el-button size="large" plain @click="goToAdmin">管理后台</el-button>
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
            <span>{{ feature.icon }}</span>
          </div>
          <h3>{{ feature.title }}</h3>
          <p>{{ feature.desc }}</p>
        </div>
      </div>
    </section>

    <!-- Platforms Section -->
    <section class="platforms">
      <h2 class="section-title">多平台支持</h2>
      <p class="platforms-desc">支持主流操作系统，随时随地接入协作</p>
      <div class="platforms-grid">
        <div class="platform-card" v-for="platform in platforms" :key="platform.name">
          <div class="platform-icon">{{ platform.icon }}</div>
          <h3>{{ platform.name }}</h3>
          <div class="platform-downloads">
            <el-button 
              v-for="dl in platform.downloads" 
              :key="dl.label"
              size="small" 
              type="primary" 
              link
              @click="download(dl.url)"
            >
              {{ dl.label }}
            </el-button>
          </div>
        </div>
      </div>
    </section>

    <!-- Footer -->
    <footer class="footer">
      <div class="footer-content">
        <p>&copy; {{ new Date().getFullYear() }} QIM. All rights reserved.</p>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()

const features = [
  {
    icon: '💬',
    title: '即时通讯',
    desc: '支持单聊、群聊、频道等多种通讯模式，消息实时同步',
    color: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
  },
  {
    icon: '🤖',
    title: 'AI 助手',
    desc: '内置 AI 助手，支持智能问答、内容生成、代码辅助等能力',
    color: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
  },
  {
    icon: '👥',
    title: '组织架构',
    desc: '完整的企业组织架构管理，部门、员工、角色权限一目了然',
    color: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
  },
  {
    icon: '📁',
    title: '文件协作',
    desc: '支持文件上传、分享、在线预览，团队协作更高效',
    color: 'linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)',
  },
  {
    icon: '🔔',
    title: '消息通知',
    desc: '系统消息、@提醒、审批通知等多维度消息推送',
    color: 'linear-gradient(135deg, #fa709a 0%, #fee140 100%)',
  },
  {
    icon: '🛡️',
    title: '安全管控',
    desc: '黑名单、敏感词、操作日志等企业级安全管控能力',
    color: 'linear-gradient(135deg, #a8edea 0%, #fed6e3 100%)',
  },
]

const platforms = [
  {
    name: 'Windows',
    icon: '🪟',
    downloads: [
      { label: '下载 x64', url: '/downloads/qim-windows-x64.exe' },
      { label: '下载 x86', url: '/downloads/qim-windows-x86.exe' },
    ],
  },
  {
    name: 'macOS',
    icon: '🍎',
    downloads: [
      { label: '下载 Intel', url: '/downloads/qim-macos-intel.dmg' },
      { label: '下载 Apple Silicon', url: '/downloads/qim-macos-arm.dmg' },
    ],
  },
  {
    name: 'Linux',
    icon: '🐧',
    downloads: [
      { label: '下载 .deb', url: '/downloads/qim-linux.deb' },
      { label: '下载 .rpm', url: '/downloads/qim-linux.rpm' },
    ],
  },
]

const goToLogin = () => {
  router.push('/login')
}

const goToAdmin = () => {
  router.push('/admin')
}

const scrollToFeatures = () => {
  document.getElementById('features')?.scrollIntoView({ behavior: 'smooth' })
}

const download = (url: string) => {
  window.open(url, '_blank')
}
</script>

<style scoped>
.landing-page {
  min-height: 100vh;
  background: #fafbfc;
}

.hero {
  position: relative;
  min-height: 600px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%);
  color: #fff;
  overflow: hidden;
}

.hero-bg {
  position: absolute;
  inset: 0;
  background: 
    radial-gradient(circle at 20% 50%, rgba(120, 119, 198, 0.3) 0%, transparent 50%),
    radial-gradient(circle at 80% 20%, rgba(255, 119, 198, 0.2) 0%, transparent 50%),
    radial-gradient(circle at 40% 80%, rgba(120, 219, 255, 0.2) 0%, transparent 50%);
}

.hero-content {
  position: relative;
  z-index: 1;
  text-align: center;
  max-width: 800px;
  padding: 0 24px;
}

.logo {
  margin-bottom: 32px;
}

.logo-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 80px;
  height: 80px;
  border-radius: 20px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  font-size: 24px;
  font-weight: 700;
  letter-spacing: 2px;
}

.title {
  font-size: 48px;
  font-weight: 700;
  margin: 0 0 16px;
  letter-spacing: -1px;
}

.subtitle {
  font-size: 20px;
  color: rgba(255, 255, 255, 0.8);
  margin: 0 0 40px;
  line-height: 1.6;
}

.hero-actions {
  display: flex;
  gap: 16px;
  justify-content: center;
}

.hero-actions .el-button {
  min-width: 140px;
}

.features {
  padding: 80px 24px;
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

.features-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 32px;
}

.feature-card {
  background: #fff;
  border-radius: 16px;
  padding: 32px;
  text-align: center;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.06);
  transition: transform 0.3s, box-shadow 0.3s;
}

.feature-card:hover {
  transform: translateY(-8px);
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.12);
}

.feature-icon {
  width: 64px;
  height: 64px;
  border-radius: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  margin-bottom: 20px;
}

.feature-card h3 {
  font-size: 20px;
  font-weight: 600;
  color: #1a1a2e;
  margin: 0 0 12px;
}

.feature-card p {
  font-size: 14px;
  color: #666;
  line-height: 1.6;
  margin: 0;
}

.platforms {
  padding: 80px 24px;
  background: #fff;
}

.platforms-desc {
  text-align: center;
  font-size: 16px;
  color: #666;
  margin: -32px 0 48px;
}

.platforms-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 32px;
  max-width: 900px;
  margin: 0 auto;
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
  font-size: 48px;
  margin-bottom: 16px;
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

.footer {
  padding: 40px 24px;
  background: #1a1a2e;
  color: rgba(255, 255, 255, 0.6);
  text-align: center;
}

.footer-content p {
  margin: 0;
  font-size: 14px;
}

@media (max-width: 768px) {
  .title {
    font-size: 32px;
  }

  .subtitle {
    font-size: 16px;
  }

  .features-grid,
  .platforms-grid {
    grid-template-columns: 1fr;
  }

  .hero-actions {
    flex-direction: column;
    align-items: center;
  }
}
</style>

import { defineConfig } from 'vitepress'
import { copyrightText } from './meta'

export default defineConfig({
  title: 'NUIM 青雀',
  description: '企业级智能协作平台',
  lang: 'zh-CN',
  base: '/',
  cleanUrls: true,
  outDir: 'dist',

  // 站点级 head 资源
  head: [
    ['link', { rel: 'icon', type: 'image/png', href: '/app-logo-v1.png' }],
    ['meta', { name: 'viewport', content: 'width=device-width, initial-scale=1' }],
  ],

  // 开发环境代理：页面通过相对路径 /api/... 拉取版本与下载信息，
  // 生产由部署侧反向代理到后端，此处仅让 vitepress dev 也能联调。
  vite: {
    server: {
      proxy: {
        '/api': 'http://localhost:8080',
      },
    },
  },

  themeConfig: {
    // 文档页顶部导航栏：仅用于文档页面，提供返回首页的入口
    // 注意：首页导航由 Home.vue 组件独立实现，与此处配置无关
    nav: [
      { text: '← 返回首页', link: '/' },
    ],
    sidebar: {
      collapsible: true,
      '/docs/': [
        {
          text: '开发者文档',
          collapsible: true,
          items: [
            { text: '功能介绍', link: '/docs/features' },
            { text: '详细使用手册', link: '/docs/usage' },
            { text: 'CLI 使用指南', link: '/docs/cli' },
            { text: 'MCP 接入指南', link: '/docs/mcp' },
          ],
        },
      ],
    },
    socialLinks: [],
    footer: {
      message: copyrightText,
      copyright: '',
    },
    outline: {
      label: '本页目录',
      level: [2, 3],
    },
    docFooter: {
      prev: '上一页',
      next: '下一页',
    },
    darkModeSwitchLabel: '主题',
    sidebarMenuLabel: '菜单',
    returnToTopLabel: '回到顶部',
    langMenuLabel: '语言',
  },
})

import { defineConfig } from 'vitepress'
import { copyrightText } from './meta'

export default defineConfig({
  title: 'QIM 青雀',
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

  themeConfig: {
    // 隐藏 VitePress 默认导航，首页完全自定义
    nav: [],
    sidebar: {
      '/docs/': [
        {
          text: '开发者文档',
          items: [
            { text: 'CLI 使用指南', link: '/docs/cli' },
            { text: 'MCP 接入指南', link: '/docs/mcp' },
          ],
        },
      ],
    },
    // 文档页顶部导航
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

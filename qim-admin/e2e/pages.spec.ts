import { test, expect } from '@playwright/test'

const PAGES_TO_TEST = [
  { path: '/', name: '仪表盘' },
  { path: '/users', name: '用户管理' },
  { path: '/organization', name: '组织架构' },
  { path: '/groups', name: '群组管理' },
  { path: '/conversations', name: '会话管理' },
  { path: '/channels', name: '频道管理' },
  { path: '/apps', name: '应用管理' },
  { path: '/mini-apps', name: '小程序管理' },
  { path: '/messages', name: '系统消息' },
  { path: '/message-search', name: '消息搜索' },
  { path: '/file-storage', name: '文件存储管理' },
  { path: '/server-monitor', name: '服务器监控' },
  { path: '/notifications', name: '通知管理' },
  { path: '/blacklist', name: '黑名单管理' },
  { path: '/statistics', name: '数据统计' },
  { path: '/roles', name: '角色权限' },
  { path: '/ai-assistant', name: 'AI 助手' },
  { path: '/ai-ops', name: 'AI 运维面板' },
  { path: '/ai-config', name: 'AI 模型配置' },
  { path: '/approvals', name: '审批管理' },
  { path: '/sensitive-words', name: '敏感词管理' },
  { path: '/operation-logs', name: '操作日志' },
  { path: '/system-config', name: '系统配置' },
  { path: '/version-management', name: '版本管理' },
]

test.describe('管理后台功能测试', () => {
  test.beforeEach(async ({ page }) => {
    page.on('console', msg => {
      if (msg.type() === 'error') {
        console.log(`Console Error on ${page.url()}: ${msg.text()}`)
      }
    })
    page.on('pageerror', error => {
      console.log(`Page Error on ${page.url()}: ${error.message}`)
    })
  })

  for (const pageInfo of PAGES_TO_TEST) {
    test(`${pageInfo.name} (${pageInfo.path}) - 页面加载测试`, async ({ page }) => {
      const errors: string[] = []
      page.on('pageerror', error => {
        errors.push(error.message)
      })
      page.on('console', msg => {
        if (msg.type() === 'error') {
          const text = msg.text()
          if (!text.includes('favicon') && !text.includes('net::ERR')) {
            errors.push(`Console Error: ${text}`)
          }
        }
      })

      await page.goto(pageInfo.path, { waitUntil: 'networkidle' })

      await expect(page).toHaveTitle(/QIM Admin/)

      const mainContent = page.locator('.admin-layout, .app-main, main, #app')
      await expect(mainContent).toBeVisible()

      if (errors.length > 0) {
        console.log(`Errors found on ${pageInfo.name}:`, errors)
      }
      expect(errors.filter(e => !e.includes('401') && !e.includes('403') && !e.includes('404'))).toHaveLength(0)
    })
  }
})
import { test, expect } from '@playwright/test'

const BASE_URL = 'http://localhost:8080'

test.describe('QIM 前端 E2E 回归测试', () => {
  
  test.describe('认证模块', () => {
    test('应该成功登录并跳转到主页面', async ({ page }) => {
      await page.goto('http://localhost:5173/#/login')
      
      await page.fill('input[placeholder="请输入用户名"]', 'admin')
      await page.fill('input[placeholder="请输入密码"]', '123456')
      await page.click('button:has-text("登录")')
      
      await page.waitForTimeout(2000)
      
      const currentUrl = page.url()
      expect(currentUrl).toContain('/main')
    })

    test('应该显示登录错误信息当密码错误时', async ({ page }) => {
      await page.goto('http://localhost:5173/#/login')
      
      await page.fill('input[placeholder="请输入用户名"]', 'admin')
      await page.fill('input[placeholder="请输入密码"]', 'wrong')
      await page.click('button:has-text("登录")')
      
      await expect(page.locator('.el-message--error')).toBeVisible({ timeout: 5000 })
    })
  })

  test.describe('会话管理', () => {
    test.use({ storageState: { cookies: [], origins: [] } })
    
    test.beforeEach(async ({ page }) => {
      await page.goto('http://localhost:5173/#/login')
      await page.fill('input[placeholder="请输入用户名"]', 'admin')
      await page.fill('input[placeholder="请输入密码"]', '123456')
      await page.click('button:has-text("登录")')
      await page.waitForURL('**/main', { timeout: 10000 })
    })

    test('应该显示侧边栏导航', async ({ page }) => {
      await expect(page.locator('.sidebar-left')).toBeVisible()
    })

    test('应该能切换到组织架构面板', async ({ page }) => {
      await page.click('.sidebar-left .sidebar-item:nth-child(2)')
      await expect(page.locator('.org-chart-panel')).toBeVisible()
    })

    test('应该能打开搜索功能', async ({ page }) => {
      await page.keyboard.press('Control+Shift+F')
      await expect(page.locator('.search-modal')).toBeVisible()
    })
  })

  test.describe('聊天功能', () => {
    test.use({ storageState: { cookies: [], origins: [] } })
    
    test.beforeEach(async ({ page }) => {
      await page.goto('http://localhost:5173/#/login')
      await page.fill('input[placeholder="请输入用户名"]', 'admin')
      await page.fill('input[placeholder="请输入密码"]', '123456')
      await page.click('button:has-text("登录")')
      await page.waitForURL('**/main', { timeout: 10000 })
    })

    test('应该能发送消息', async ({ page }) => {
      await page.click('.conversation-item', { timeout: 5000 })
      
      await page.fill('.message-input textarea', 'E2E 测试消息')
      await page.keyboard.press('Enter')
      
      await expect(page.locator('.message-list .message-item')).toHaveCount({ min: 1 })
    })

    test('应该能点击表情按钮', async ({ page }) => {
      await expect(page.locator('.message-input-area .emoji-btn')).toBeVisible()
      await page.click('.message-input-area .emoji-btn')
      await expect(page.locator('.emoji-picker')).toBeVisible()
    })
  })

  test.describe('应用中心', () => {
    test.use({ storageState: { cookies: [], origins: [] } })
    
    test.beforeEach(async ({ page }) => {
      await page.goto('http://localhost:5173/#/login')
      await page.fill('input[placeholder="请输入用户名"]', 'admin')
      await page.fill('input[placeholder="请输入密码"]', '123456')
      await page.click('button:has-text("登录")')
      await page.waitForURL('**/main', { timeout: 10000 })
    })

    test('应该能切换到应用面板', async ({ page }) => {
      await page.click('.sidebar-left .sidebar-item:nth-child(4)')
      await expect(page.locator('.app-center')).toBeVisible()
    })

    test('应该显示多个应用卡片', async ({ page }) => {
      await page.click('.sidebar-left .sidebar-item:nth-child(4)')
      const appCards = page.locator('.app-item')
      await expect(appCards).toHaveCount({ min: 1 })
    })
  })
})

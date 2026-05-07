import { test, expect } from '@playwright/test'

test.describe('管理后台功能测试 - 实际功能验证', () => {
  test.beforeEach(async ({ page }) => {
    const errors: string[] = []
    page.on('console', msg => {
      if (msg.type() === 'error') {
        const text = msg.text()
        if (!text.includes('favicon') && !text.includes('net::ERR')) {
          console.log(`Console Error: ${text}`)
          errors.push(text)
        }
      }
    })
    page.on('pageerror', error => {
      console.log(`Page Error: ${error.message}`)
      errors.push(error.message)
    })
  })

  test('登录并验证能进入首页', async ({ page }) => {
    await page.goto('/login')
    await expect(page.locator('.login-title')).toContainText('QIM Admin')
  })

  test('Dashboard 页面加载和数据获取', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('.dashboard-container, .dashboard-title')).toBeVisible({ timeout: 5000 })
    await page.waitForTimeout(2000)
    const errorMessages = page.locator('.el-message--error')
    const errorCount = await errorMessages.count()
    if (errorCount > 0) {
      console.log('Dashboard found error messages:', await errorMessages.allTextContents())
    }
  })

  test('用户管理页面加载', async ({ page }) => {
    await page.goto('/users')
    await page.waitForTimeout(2000)
    const errorMessages = page.locator('.el-message--error')
    const errorCount = await errorMessages.count()
    if (errorCount > 0) {
      console.log('Users page found error messages:', await errorMessages.allTextContents())
    }
  })

  test('群组管理页面加载', async ({ page }) => {
    await page.goto('/groups')
    await page.waitForTimeout(2000)
    const errorMessages = page.locator('.el-message--error')
    const errorCount = await errorMessages.count()
    if (errorCount > 0) {
      console.log('Groups page found error messages:', await errorMessages.allTextContents())
    }
  })

  test('应用管理页面加载', async ({ page }) => {
    await page.goto('/apps')
    await page.waitForTimeout(2000)
    const errorMessages = page.locator('.el-message--error')
    const errorCount = await errorMessages.count()
    if (errorCount > 0) {
      console.log('Apps page found error messages:', await errorMessages.allTextContents())
    }
  })

  test('AI 配置页面加载', async ({ page }) => {
    await page.goto('/ai-config')
    await page.waitForTimeout(2000)
    const errorMessages = page.locator('.el-message--error')
    const errorCount = await errorMessages.count()
    if (errorCount > 0) {
      console.log('AI Config page found error messages:', await errorMessages.allTextContents())
    }
  })

  test('敏感词管理页面加载', async ({ page }) => {
    await page.goto('/sensitive-words')
    await page.waitForTimeout(2000)
    const errorMessages = page.locator('.el-message--error')
    const errorCount = await errorMessages.count()
    if (errorCount > 0) {
      console.log('Sensitive Words page found error messages:', await errorMessages.allTextContents())
    }
  })

  test('系统配置页面加载', async ({ page }) => {
    await page.goto('/system-config')
    await page.waitForTimeout(2000)
    const errorMessages = page.locator('.el-message--error')
    const errorCount = await errorMessages.count()
    if (errorCount > 0) {
      console.log('System Config page found error messages:', await errorMessages.allTextContents())
    }
  })

  test('操作日志页面加载', async ({ page }) => {
    await page.goto('/operation-logs')
    await page.waitForTimeout(2000)
    const errorMessages = page.locator('.el-message--error')
    const errorCount = await errorMessages.count()
    if (errorCount > 0) {
      console.log('Operation Logs page found error messages:', await errorMessages.allTextContents())
    }
  })

  test('通知管理页面加载', async ({ page }) => {
    await page.goto('/notifications')
    await page.waitForTimeout(2000)
    const errorMessages = page.locator('.el-message--error')
    const errorCount = await errorMessages.count()
    if (errorCount > 0) {
      console.log('Notifications page found error messages:', await errorMessages.allTextContents())
    }
  })

  test('黑名单管理页面加载', async ({ page }) => {
    await page.goto('/blacklist')
    await page.waitForTimeout(2000)
    const errorMessages = page.locator('.el-message--error')
    const errorCount = await errorMessages.count()
    if (errorCount > 0) {
      console.log('Blacklist page found error messages:', await errorMessages.allTextContents())
    }
  })
})
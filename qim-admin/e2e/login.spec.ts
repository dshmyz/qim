import { test, expect } from '@playwright/test'

test.describe('登录页面', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login')
  })

  test('应该正确显示登录页面', async ({ page }) => {
    await expect(page.locator('.login-title')).toContainText('QIM Admin')
    await expect(page.locator('.login-subtitle')).toContainText('企业级即时通讯管理后台')
  })

  test('应该显示用户名和密码输入框', async ({ page }) => {
    const usernameInput = page.locator('input[placeholder="请输入用户名"]')
    const passwordInput = page.locator('input[placeholder="请输入密码"]')
    const loginButton = page.locator('.login-btn')

    await expect(usernameInput).toBeVisible()
    await expect(passwordInput).toBeVisible()
    await expect(loginButton).toBeVisible()
    await expect(loginButton).toContainText('登 录')
  })

  test('应该显示版权信息', async ({ page }) => {
    const currentYear = new Date().getFullYear()
    await expect(page.locator('.login-footer')).toContainText(String(currentYear))
  })

  test('未填写表单时点击登录应该显示验证错误', async ({ page }) => {
    await page.locator('.login-btn').click()
    await expect(page.locator('.el-form-item__error').first()).toBeVisible()
  })

  test('只填写用户名不填密码应该显示密码验证错误', async ({ page }) => {
    await page.locator('input[placeholder="请输入用户名"]').fill('admin')
    await page.locator('.login-btn').click()

    const errors = page.locator('.el-form-item__error')
    await expect(errors).toHaveCount(1)
  })
})

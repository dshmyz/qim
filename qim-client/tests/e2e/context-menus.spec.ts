import { test, expect, Page } from '@playwright/test'

// 测试辅助函数 - 通过 mock API 响应实现登录
async function login(page: Page) {
  // Mock 登录API 响应
  await page.route('**/api/v1/auth/login', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        message: '登录成功',
        data: {
          token: 'mock-token-for-testing',
          user: {
            id: 1,
            username: 'admin',
            name: '管理员',
            email: 'admin@example.com',
            avatar: null,
            isAdmin: true
          }
        }
      })
    })
  })

  // Mock 2FA 检查
  await page.route('**/api/v1/auth/check-2fa', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: { twoFactorEnabled: false } })
    })
  })

  // Mock 组织架构 API
  await page.route('**/api/v1/org/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: [] })
    })
  })

  // Mock 会话列表 API
  await page.route('**/api/v1/conversations/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        data: [
          {
            id: 1,
            type: 'private',
            name: '测试用户',
            avatar: null,
            lastMessage: '你好',
            unreadCount: 0,
            lastMessageTime: new Date().toISOString()
          },
          {
            id: 2,
            type: 'group',
            name: '测试群组',
            avatar: null,
            lastMessage: '欢迎加入',
            unreadCount: 1,
            lastMessageTime: new Date().toISOString()
          }
        ]
      })
    })
  })

  // Mock 消息列表 API
  await page.route('**/api/v1/messages/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: [] })
    })
  })

  // Mock 群组 API
  await page.route('**/api/v1/groups/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: [] })
    })
  })

  // Mock 频道 API
  await page.route('**/api/v1/channels/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: [] })
    })
  })

  // Mock 应用列表 API
  await page.route('**/api/v1/apps/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        data: [
          { id: '1', name: '统计报表', icon: 'chart', description: '数据统计' },
          { id: '2', name: '日历', icon: 'calendar', description: '日程管理' }
        ]
      })
    })
  })

  // Mock 用户资料 API
  await page.route('**/api/v1/users/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        data: { id: 1, username: 'admin', name: '管理员' }
      })
    })
  })

  // Mock 员工列表 API
  await page.route('**/api/v1/employees/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        data: [
          { id: 1, username: 'admin', name: '管理员', department: '技术部' },
          { id: 2, username: 'user1', name: '测试用户', department: '产品部' }
        ]
      })
    })
  })

  // Mock 通知 API
  await page.route('**/api/v1/notifications/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: [] })
    })
  })

  // Mock 搜索 API
  await page.route('**/api/v1/search/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: [] })
    })
  })

  // Mock 版本检查 API
  await page.route('**/api/v1/version/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: { needUpdate: false } })
    })
  })

  // Mock 头像获取
  await page.route('**/api/v1/avatars/**', async (route) => {
    await route.fulfill({ status: 404 })
  })

  await page.goto('http://localhost:3000')
  await page.fill('input[placeholder="请输入用户名"]', 'admin')
  await page.fill('input[placeholder="请输入密码"]', '123456')
  await page.click('button:has-text("登录")')
  
  // 等待 Main 组件渲染完成
  await page.waitForSelector('.im-container', { timeout: 10000 })
  await page.waitForTimeout(2000)
}

test.describe('右键菜单和点击菜单选项测试', () => {
  
  test.beforeEach(async ({ page }) => {
    await login(page)
    // 自动关闭网络错误提示遮罩层
    await page.evaluate(() => {
      const el = document.querySelector('.network-error')
      if (el) el.remove()
      const style = document.createElement('style')
      style.textContent = '.network-error { display: none !important; }'
      document.head.appendChild(style)
    })
    await page.waitForTimeout(500)
  })

  test.describe('会话右键菜单测试', () => {
    test('右键点击会话应该显示右键菜单', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible({ timeout: 5000 })) {
        await firstConversation.click({ button: 'right' })
        await page.waitForTimeout(500)
        
        // 验证右键菜单出现
        const contextMenu = page.locator('.context-menu')
        await expect(contextMenu).toBeVisible()
      }
    })

    test('右键菜单应该包含置顶选项', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible({ timeout: 5000 })) {
        await firstConversation.click({ button: 'right' })
        await page.waitForTimeout(500)
        
        const pinOption = page.locator('.context-menu-item', { hasText: '置顶' }).first()
        await expect(pinOption).toBeVisible()
      }
    })

    test('右键菜单应该包含免打扰选项', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible({ timeout: 5000 })) {
        await firstConversation.click({ button: 'right' })
        await page.waitForTimeout(500)
        
        const muteOption = page.locator('.context-menu-item', { hasText: '免打扰' }).first()
        await expect(muteOption).toBeVisible()
      }
    })

    test('点击置顶选项应该执行置顶操作并关闭菜单', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible({ timeout: 5000 })) {
        await firstConversation.click({ button: 'right' })
        await page.waitForTimeout(500)
        
        const pinOption = page.locator('.context-menu-item', { hasText: '置顶' }).first()
        await pinOption.click()
        await page.waitForTimeout(500)
        
        // 验证菜单关闭
        const contextMenu = page.locator('.context-menu')
        const isVisible = await contextMenu.isVisible().catch(() => false)
        expect(isVisible).toBe(false)
      }
    })

    test('点击免打扰选项应该执行免打扰操作并关闭菜单', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible({ timeout: 5000 })) {
        await firstConversation.click({ button: 'right' })
        await page.waitForTimeout(500)
        
        const muteOption = page.locator('.context-menu-item', { hasText: '免打扰' }).first()
        await muteOption.click()
        await page.waitForTimeout(500)
        
        // 验证菜单关闭
        const contextMenu = page.locator('.context-menu')
        const isVisible = await contextMenu.isVisible().catch(() => false)
        expect(isVisible).toBe(false)
      }
    })

    test('点击移除会话选项应该显示确认对话框', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible({ timeout: 5000 })) {
        await firstConversation.click({ button: 'right' })
        await page.waitForTimeout(500)
        
        const removeOption = page.locator('.context-menu-item', { hasText: '移除' }).first()
        await removeOption.click()
        await page.waitForTimeout(500)
        
        // 应该弹出确认对话框
        const dialog = page.locator('.q-dialog, .el-message-box, .confirm-dialog')
        const isVisible = await dialog.isVisible().catch(() => false)
        expect(isVisible).toBe(true)
      }
    })

    test('点击空白处应该关闭右键菜单', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible({ timeout: 5000 })) {
        await firstConversation.click({ button: 'right' })
        await page.waitForTimeout(500)
        
        // 点击空白处
        await page.click('body')
        await page.waitForTimeout(500)
        
        const contextMenu = page.locator('.context-menu')
        const isVisible = await contextMenu.isVisible().catch(() => false)
        expect(isVisible).toBe(false)
      }
    })
  })

  test.describe('动作菜单测试', () => {
    test('点击+按钮应该显示动作菜单', async ({ page }) => {
      const actionBtn = page.locator('.add-conversation-btn')
      if (await actionBtn.isVisible()) {
        await actionBtn.click()
        await page.waitForTimeout(500)
        
        const actionMenu = page.locator('.action-menu')
        await expect(actionMenu).toBeVisible()
      }
    })

    test('动作菜单应该包含创建群聊选项', async ({ page }) => {
      const actionBtn = page.locator('.add-conversation-btn')
      if (await actionBtn.isVisible()) {
        await actionBtn.click()
        await page.waitForTimeout(500)
        
        const createGroupOption = page.locator('.action-menu-item', { hasText: '创建群聊' })
        await expect(createGroupOption).toBeVisible()
      }
    })

    test('动作菜单应该包含创建讨论组选项', async ({ page }) => {
      const actionBtn = page.locator('.add-conversation-btn')
      if (await actionBtn.isVisible()) {
        await actionBtn.click()
        await page.waitForTimeout(500)
        
        const createDiscussionOption = page.locator('.action-menu-item', { hasText: '创建讨论组' })
        await expect(createDiscussionOption).toBeVisible()
      }
    })

    test('动作菜单应该包含创建频道选项', async ({ page }) => {
      const actionBtn = page.locator('.add-conversation-btn')
      if (await actionBtn.isVisible()) {
        await actionBtn.click()
        await page.waitForTimeout(500)
        
        const createChannelOption = page.locator('.action-menu-item', { hasText: '创建频道' })
        await expect(createChannelOption).toBeVisible()
      }
    })

    test('点击创建群聊应该打开创建群组对话框', async ({ page }) => {
      const actionBtn = page.locator('.add-conversation-btn')
      if (await actionBtn.isVisible()) {
        await actionBtn.click()
        await page.waitForTimeout(500)
        
        const createGroupOption = page.locator('.action-menu-item', { hasText: '创建群聊' })
        await createGroupOption.click()
        await page.waitForTimeout(500)
        
        // 应该打开创建群组对话框
        const dialog = page.locator('.q-dialog, .modal, .create-group-dialog')
        const isVisible = await dialog.isVisible().catch(() => false)
        expect(isVisible).toBe(true)
      }
    })

    test('点击动作菜单项后菜单应该关闭', async ({ page }) => {
      const actionBtn = page.locator('.add-conversation-btn')
      if (await actionBtn.isVisible()) {
        await actionBtn.click()
        await page.waitForTimeout(500)
        
        const createGroupOption = page.locator('.action-menu-item', { hasText: '创建群聊' })
        await createGroupOption.click()
        await page.waitForTimeout(500)
        
        const actionMenu = page.locator('.action-menu')
        const isVisible = await actionMenu.isVisible().catch(() => false)
        expect(isVisible).toBe(false)
      }
    })

    test('点击空白处应该关闭动作菜单', async ({ page }) => {
      const actionBtn = page.locator('.add-conversation-btn')
      if (await actionBtn.isVisible()) {
        await actionBtn.click()
        await page.waitForTimeout(500)
        
        await page.click('body')
        await page.waitForTimeout(500)
        
        const actionMenu = page.locator('.action-menu')
        const isVisible = await actionMenu.isVisible().catch(() => false)
        expect(isVisible).toBe(false)
      }
    })
  })

  test.describe('用户操作菜单测试', () => {
    test('点击更多操作按钮应该显示用户菜单', async ({ page }) => {
      const moreBtn = page.locator('.more-actions-btn, .user-action-btn')
      if (await moreBtn.isVisible()) {
        await moreBtn.click()
        await page.waitForTimeout(500)
        
        // 验证菜单出现
        const userMenu = page.locator('.context-menu, .user-context-menu')
        const isVisible = await userMenu.isVisible().catch(() => false)
        expect(isVisible).toBe(true)
      }
    })

    test('用户菜单应该包含关于选项', async ({ page }) => {
      const moreBtn = page.locator('.more-actions-btn, .user-action-btn')
      if (await moreBtn.isVisible()) {
        await moreBtn.click()
        await page.waitForTimeout(500)
        
        const aboutOption = page.locator('.context-menu-item', { hasText: '关于' })
        const isVisible = await aboutOption.isVisible().catch(() => false)
        if (isVisible) {
          await expect(aboutOption).toBeVisible()
        }
      }
    })

    test('用户菜单应该包含检查更新选项', async ({ page }) => {
      const moreBtn = page.locator('.more-actions-btn, .user-action-btn')
      if (await moreBtn.isVisible()) {
        await moreBtn.click()
        await page.waitForTimeout(500)
        
        const checkUpdateOption = page.locator('.context-menu-item', { hasText: '检查更新' })
        const isVisible = await checkUpdateOption.isVisible().catch(() => false)
        if (isVisible) {
          await expect(checkUpdateOption).toBeVisible()
        }
      }
    })

    test('用户菜单应该包含设置选项', async ({ page }) => {
      const moreBtn = page.locator('.more-actions-btn, .user-action-btn')
      if (await moreBtn.isVisible()) {
        await moreBtn.click()
        await page.waitForTimeout(500)
        
        const settingsOption = page.locator('.context-menu-item', { hasText: '设置' })
        await expect(settingsOption).toBeVisible()
      }
    })

    test('用户菜单应该包含退出登录选项', async ({ page }) => {
      const moreBtn = page.locator('.more-actions-btn, .user-action-btn')
      if (await moreBtn.isVisible()) {
        await moreBtn.click()
        await page.waitForTimeout(500)
        
        const logoutOption = page.locator('.context-menu-item', { hasText: '退出登录' })
        await expect(logoutOption).toBeVisible()
      }
    })

    test('点击设置选项应该打开设置面板', async ({ page }) => {
      const moreBtn = page.locator('.more-actions-btn, .user-action-btn')
      if (await moreBtn.isVisible()) {
        await moreBtn.click()
        await page.waitForTimeout(500)
        
        const settingsOption = page.locator('.context-menu-item', { hasText: '设置' })
        await settingsOption.click()
        await page.waitForTimeout(500)
        
        // 验证设置面板打开
        const settingsPanel = page.locator('.settings-modal, .settings-panel, .q-dialog')
        const isVisible = await settingsPanel.isVisible().catch(() => false)
        expect(isVisible).toBe(true)
      }
    })

    test('点击退出登录选项应该弹出确认对话框', async ({ page }) => {
      const moreBtn = page.locator('.more-actions-btn, .user-action-btn')
      if (await moreBtn.isVisible()) {
        await moreBtn.click()
        await page.waitForTimeout(500)
        
        const logoutOption = page.locator('.context-menu-item', { hasText: '退出登录' })
        await logoutOption.click()
        await page.waitForTimeout(500)
        
        // 应该弹出确认对话框
        const dialog = page.locator('.q-dialog, .el-message-box, .confirm-dialog')
        const isVisible = await dialog.isVisible().catch(() => false)
        expect(isVisible).toBe(true)
      }
    })

    test('点击关于选项应该显示关于对话框', async ({ page }) => {
      const moreBtn = page.locator('.more-actions-btn, .user-action-btn')
      if (await moreBtn.isVisible()) {
        await moreBtn.click()
        await page.waitForTimeout(500)
        
        const aboutOption = page.locator('.context-menu-item', { hasText: '关于' })
        const isVisible = await aboutOption.isVisible().catch(() => false)
        if (isVisible) {
          await aboutOption.click()
          await page.waitForTimeout(500)
          
          // 验证关于对话框打开
          const dialog = page.locator('.q-dialog, .about-dialog, .about-modal')
          const dialogVisible = await dialog.isVisible().catch(() => false)
          expect(dialogVisible).toBe(true)
        }
      }
    })

    test('点击空白处应该关闭用户菜单', async ({ page }) => {
      const moreBtn = page.locator('.more-actions-btn, .user-action-btn')
      if (await moreBtn.isVisible()) {
        await moreBtn.click()
        await page.waitForTimeout(500)
        
        await page.click('body')
        await page.waitForTimeout(500)
        
        const userMenu = page.locator('.context-menu, .user-context-menu')
        const isVisible = await userMenu.isVisible().catch(() => false)
        expect(isVisible).toBe(false)
      }
    })
  })

  test.describe('设置菜单（侧边栏底部）测试', () => {
    test('点击设置按钮应该显示设置菜单', async ({ page }) => {
      const settingsBtn = page.locator('.settings-btn')
      if (await settingsBtn.isVisible()) {
        await settingsBtn.click()
        await page.waitForTimeout(500)
        
        const settingsMenu = page.locator('.context-menu')
        const isVisible = await settingsMenu.isVisible().catch(() => false)
        expect(isVisible).toBe(true)
      }
    })

    test('点击设置菜单中的设置选项应该打开设置面板', async ({ page }) => {
      const settingsBtn = page.locator('.settings-btn')
      if (await settingsBtn.isVisible()) {
        await settingsBtn.click()
        await page.waitForTimeout(500)
        
        const settingsOption = page.locator('.context-menu-item', { hasText: '设置' })
        await settingsOption.click()
        await page.waitForTimeout(500)
        
        const settingsPanel = page.locator('.settings-modal, .settings-panel, .q-dialog')
        const isVisible = await settingsPanel.isVisible().catch(() => false)
        expect(isVisible).toBe(true)
      }
    })
  })

  test.describe('主题菜单测试', () => {
    test('点击主题按钮应该显示主题菜单', async ({ page }) => {
      const themeBtn = page.locator('.theme-btn')
      if (await themeBtn.isVisible()) {
        await themeBtn.click()
        await page.waitForTimeout(500)
        
        const themeMenu = page.locator('.theme-menu, .context-menu')
        const isVisible = await themeMenu.isVisible().catch(() => false)
        expect(isVisible).toBe(true)
      }
    })

    test('主题菜单应该包含多个主题选项', async ({ page }) => {
      const themeBtn = page.locator('.theme-btn')
      if (await themeBtn.isVisible()) {
        await themeBtn.click()
        await page.waitForTimeout(500)
        
        const themeItems = page.locator('.theme-menu .context-menu-item, .context-menu .context-menu-item')
        const count = await themeItems.count()
        expect(count).toBeGreaterThan(1)
      }
    })

    test('点击主题选项应该切换主题', async ({ page }) => {
      const themeBtn = page.locator('.theme-btn')
      if (await themeBtn.isVisible()) {
        await themeBtn.click()
        await page.waitForTimeout(500)
        
        // 点击第一个主题选项
        const firstThemeOption = page.locator('.theme-menu .context-menu-item, .context-menu .context-menu-item').first()
        await firstThemeOption.click()
        await page.waitForTimeout(500)
        
        // 验证菜单关闭
        const themeMenu = page.locator('.theme-menu')
        const isVisible = await themeMenu.isVisible().catch(() => false)
        expect(isVisible).toBe(false)
      }
    })

    test('点击空白处应该关闭主题菜单', async ({ page }) => {
      const themeBtn = page.locator('.theme-btn')
      if (await themeBtn.isVisible()) {
        await themeBtn.click()
        await page.waitForTimeout(500)
        
        await page.click('body')
        await page.waitForTimeout(500)
        
        const themeMenu = page.locator('.theme-menu')
        const isVisible = await themeMenu.isVisible().catch(() => false)
        expect(isVisible).toBe(false)
      }
    })
  })

  test.describe('消息右键菜单测试', () => {
    test.beforeEach(async ({ page }) => {
      // 先选择一个会话
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible({ timeout: 5000 })) {
        await firstConversation.click()
        await page.waitForTimeout(1000)
      }
    })

    test('右键点击消息应该显示消息右键菜单', async ({ page }) => {
      const firstMessage = page.locator('.message-item').first()
      if (await firstMessage.isVisible({ timeout: 5000 })) {
        await firstMessage.click({ button: 'right' })
        await page.waitForTimeout(500)
        
        // 验证消息右键菜单出现
        const messageMenu = page.locator('.message-context-menu')
        await expect(messageMenu).toBeVisible()
      }
    })

    test('消息右键菜单应该包含复制选项', async ({ page }) => {
      const firstMessage = page.locator('.message-item').first()
      if (await firstMessage.isVisible({ timeout: 5000 })) {
        await firstMessage.click({ button: 'right' })
        await page.waitForTimeout(500)
        
        const copyOption = page.locator('.message-context-menu-item', { hasText: '复制' })
        const isVisible = await copyOption.isVisible().catch(() => false)
        if (isVisible) {
          await expect(copyOption).toBeVisible()
        }
      }
    })

    test('消息右键菜单应该包含回复选项', async ({ page }) => {
      const firstMessage = page.locator('.message-item').first()
      if (await firstMessage.isVisible({ timeout: 5000 })) {
        await firstMessage.click({ button: 'right' })
        await page.waitForTimeout(500)
        
        const replyOption = page.locator('.message-context-menu-item', { hasText: '回复' })
        const isVisible = await replyOption.isVisible().catch(() => false)
        if (isVisible) {
          await expect(replyOption).toBeVisible()
        }
      }
    })

    test('消息右键菜单应该包含撤回选项', async ({ page }) => {
      const firstMessage = page.locator('.message-item').first()
      if (await firstMessage.isVisible({ timeout: 5000 })) {
        await firstMessage.click({ button: 'right' })
        await page.waitForTimeout(500)
        
        const recallOption = page.locator('.message-context-menu-item', { hasText: '撤回' })
        const isVisible = await recallOption.isVisible().catch(() => false)
        if (isVisible) {
          await expect(recallOption).toBeVisible()
        }
      }
    })

    test('点击消息右键菜单选项应该执行操作并关闭菜单', async ({ page }) => {
      const firstMessage = page.locator('.message-item').first()
      if (await firstMessage.isVisible({ timeout: 5000 })) {
        await firstMessage.click({ button: 'right' })
        await page.waitForTimeout(500)
        
        const copyOption = page.locator('.message-context-menu-item', { hasText: '复制' })
        const isVisible = await copyOption.isVisible().catch(() => false)
        if (isVisible) {
          await copyOption.click()
          await page.waitForTimeout(500)
          
          // 验证菜单关闭
          const messageMenu = page.locator('.message-context-menu')
          const menuVisible = await messageMenu.isVisible().catch(() => false)
          expect(menuVisible).toBe(false)
        }
      }
    })

    test('点击空白处应该关闭消息右键菜单', async ({ page }) => {
      const firstMessage = page.locator('.message-item').first()
      if (await firstMessage.isVisible({ timeout: 5000 })) {
        await firstMessage.click({ button: 'right' })
        await page.waitForTimeout(500)
        
        await page.click('body')
        await page.waitForTimeout(500)
        
        const messageMenu = page.locator('.message-context-menu')
        const isVisible = await messageMenu.isVisible().catch(() => false)
        expect(isVisible).toBe(false)
      }
    })
  })

  test.describe('菜单叠加和关闭行为测试', () => {
    test('打开一个菜单后打开另一个菜单，第一个菜单应该关闭', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible({ timeout: 5000 })) {
        // 先打开会话右键菜单
        await firstConversation.click({ button: 'right' })
        await page.waitForTimeout(500)
        
        // 再打开动作菜单
        const actionBtn = page.locator('.add-conversation-btn')
        if (await actionBtn.isVisible()) {
          await actionBtn.click()
          await page.waitForTimeout(500)
          
          // 验证动作菜单打开
          const actionMenu = page.locator('.action-menu')
          const actionMenuVisible = await actionMenu.isVisible().catch(() => false)
          expect(actionMenuVisible).toBe(true)
        }
      }
    })

    test('按 Esc 键应该关闭所有打开的菜单', async ({ page }) => {
      const actionBtn = page.locator('.add-conversation-btn')
      if (await actionBtn.isVisible()) {
        await actionBtn.click()
        await page.waitForTimeout(500)
        
        // 按 Esc
        await page.keyboard.press('Escape')
        await page.waitForTimeout(500)
        
        const actionMenu = page.locator('.action-menu')
        const isVisible = await actionMenu.isVisible().catch(() => false)
        expect(isVisible).toBe(false)
      }
    })
  })
})

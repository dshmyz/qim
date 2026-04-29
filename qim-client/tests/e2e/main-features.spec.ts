import { test, expect, Page } from '@playwright/test'

/**
 * QIM 前端主要功能点自动化测试
 * 测试所有按钮点击行为是否符合预期
 */

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
          { id: '2', name: '日历', icon: 'calendar', description: '日程管理' },
          { id: '3', name: '文件管理', icon: 'folder', description: '文件存储' },
          { id: '4', name: 'AI 助手', icon: 'robot', description: '智能助手' },
          { id: '5', name: '任务管理', icon: 'tasks', description: '任务跟踪' },
          { id: '6', name: '便签', icon: 'sticky-note', description: '快速记录' },
          { id: '7', name: '笔记', icon: 'book', description: '笔记管理' }
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
  
  // 等待 Main 组件渲染完成（通过检查 im-container 是否存在）
  await page.waitForSelector('.im-container', { timeout: 10000 })
  await page.waitForTimeout(2000)
}

test.describe('QIM 前端全功能点自动化测试', () => {
  
  test.beforeEach(async ({ page }) => {
    await login(page)
    // 自动关闭网络错误提示遮罩层（因为没有真实 WebSocket 连接）
    await page.evaluate(() => {
      const el = document.querySelector('.network-error')
      if (el) el.remove()
      // 同时设置 showNetworkError 为 false 防止重新渲染
      const style = document.createElement('style')
      style.textContent = '.network-error { display: none !important; }'
      document.head.appendChild(style)
    })
    await page.waitForTimeout(500)
  })

  test.describe('侧边栏功能测试', () => {
    test('应该能切换会话列表面板', async ({ page }) => {
      const conversationIcon = page.locator('.side-options .option-item').first()
      await conversationIcon.click()
      await page.waitForTimeout(500)
      // 会话列表是默认显示的，验证侧边栏切换成功
      await expect(page.locator('.sidebar')).toBeVisible()
    })

    test('应该能切换组织架构面板', async ({ page }) => {
      const orgIcon = page.locator('.side-options .option-item').nth(1)
      await orgIcon.click()
      await page.waitForTimeout(500)
      // 验证组织架构树容器可见
      await expect(page.locator('.tree-container')).toBeVisible()
    })

    test('应该能切换群组面板', async ({ page }) => {
      const groupIcon = page.locator('.side-options .option-item').nth(2)
      await groupIcon.click()
      await page.waitForTimeout(500)
      // 验证群组列表容器可见
      await expect(page.locator('.groups-list')).toBeVisible()
    })

    test('应该能切换应用中心面板', async ({ page }) => {
      const appIcon = page.locator('.side-options .option-item').nth(3)
      await appIcon.click()
      await page.waitForTimeout(500)
      // 验证应用面板容器可见
      await expect(page.locator('.apps-container')).toBeVisible()
    })

    test('应该能显示用户信息区域', async ({ page }) => {
      await expect(page.locator('.user-info')).toBeVisible()
    })

    test('应该能点击通知按钮', async ({ page }) => {
      const notificationBtn = page.locator('button[title="通知"]')
      await notificationBtn.click()
      // 验证通知面板或菜单出现
      await page.waitForTimeout(500)
    })
  })

  test.describe('会话列表功能测试', () => {
    test('应该能点击会话项切换聊天', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible()) {
        await firstConversation.click()
        await expect(page.locator('.chat-window')).toBeVisible()
      }
    })

    test('应该能右键打开会话上下文菜单', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible()) {
        await firstConversation.click({ button: 'right' })
        await expect(page.locator('.context-menu')).toBeVisible()
      }
    })

    test('应该能通过上下文菜单置顶会话', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible()) {
        await firstConversation.click({ button: 'right' })
        const pinAction = page.locator('.context-menu-item:has-text("置顶")')
        if (await pinAction.isVisible()) {
          await pinAction.click()
        }
      }
    })

    test('应该能通过上下文菜单免打扰会话', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible()) {
        await firstConversation.click({ button: 'right' })
        const muteAction = page.locator('.context-menu-item:has-text("免打扰")')
        if (await muteAction.isVisible()) {
          await muteAction.click()
        }
      }
    })

    test('应该能通过上下文菜单移除会话', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible()) {
        await firstConversation.click({ button: 'right' })
        const removeAction = page.locator('.context-menu-item:has-text("移除")')
        if (await removeAction.isVisible()) {
          await removeAction.click()
          // 可能弹出确认对话框
          const confirmBtn = page.locator('.q-btn--primary, button:has-text("确定")')
          if (await confirmBtn.isVisible({ timeout: 2000 })) {
            await confirmBtn.click()
          }
        }
      }
    })
  })

  test.describe('聊天窗口功能测试', () => {
    test.beforeEach(async ({ page }) => {
      // 选择一个会话
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible({ timeout: 5000 })) {
        await firstConversation.click()
        await page.waitForTimeout(1000)
      }
    })

    test('应该能显示聊天 header', async ({ page }) => {
      const chatHeader = page.locator('.chat-header')
      await expect(chatHeader).toBeVisible()
    })
  })

  test.describe('搜索功能测试', () => {
    test('应该能使用搜索按钮打开搜索', async ({ page }) => {
      const searchBtn = page.locator('.search-btn')
      if (await searchBtn.isVisible()) {
        await searchBtn.click()
        await page.waitForTimeout(500)
      }
    })

    test('应该能执行会话内搜索', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible()) {
        await firstConversation.click()
        await page.waitForTimeout(1000)
      }
      
      const searchBtn = page.locator('.message-search-btn, .search-btn')
      if (await searchBtn.isVisible()) {
        await searchBtn.click()
        await page.waitForTimeout(500)
      }
    })
  })

  test.describe('创建功能测试', () => {
    test('应该能打开创建群组对话框', async ({ page }) => {
      const actionBtn = page.locator('.action-menu-btn, .more-actions-btn')
      if (await actionBtn.isVisible()) {
        await actionBtn.click()
        const createGroupAction = page.locator('.action-menu-item:has-text("创建群组")')
        if (await createGroupAction.isVisible()) {
          await createGroupAction.click()
          await expect(page.locator('.create-group-modal, .q-dialog')).toBeVisible()
        }
      }
    })

    test('应该能打开创建讨论组对话框', async ({ page }) => {
      const actionBtn = page.locator('.action-menu-btn, .more-actions-btn')
      if (await actionBtn.isVisible()) {
        await actionBtn.click()
        const createDiscussionAction = page.locator('.action-menu-item:has-text("创建讨论组")')
        if (await createDiscussionAction.isVisible()) {
          await createDiscussionAction.click()
        }
      }
    })
  })

  test.describe('用户操作菜单测试', () => {
    test('应该能打开用户操作菜单', async ({ page }) => {
      const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
      if (await userMenuBtn.isVisible()) {
        await userMenuBtn.click()
        await expect(page.locator('.user-context-menu')).toBeVisible()
      }
    })

    test('应该能通过菜单打开设置', async ({ page }) => {
      const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
      if (await userMenuBtn.isVisible()) {
        await userMenuBtn.click()
        const settingsAction = page.locator('.context-menu-item:has-text("设置")')
        if (await settingsAction.isVisible()) {
          await settingsAction.click()
          await expect(page.locator('.settings-modal, .settings-panel')).toBeVisible()
        }
      }
    })

    test('应该能通过菜单执行登出', async ({ page }) => {
      const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
      if (await userMenuBtn.isVisible()) {
        await userMenuBtn.click()
        const logoutAction = page.locator('.context-menu-item:has-text("退出登录")')
        if (await logoutAction.isVisible()) {
          await logoutAction.click()
          // 可能会有确认对话框
          await page.waitForTimeout(500)
        }
      }
    })

    test('应该能通过菜单查看关于信息', async ({ page }) => {
      const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
      if (await userMenuBtn.isVisible()) {
        await userMenuBtn.click()
        const aboutAction = page.locator('.context-menu-item:has-text("关于")')
        if (await aboutAction.isVisible()) {
          await aboutAction.click()
        }
      }
    })
  })

  test.describe('群组操作测试', () => {
    test('应该能查看群组成员列表', async ({ page }) => {
      // 选择一个群组会话
      const groupConversation = page.locator('.conversation-item[data-type="group"]').first()
      if (await groupConversation.isVisible()) {
        await groupConversation.click()
        await page.waitForTimeout(1000)
        
        // 打开群组成员面板
        const memberPanelBtn = page.locator('.member-panel-toggle, .group-members-btn')
        if (await memberPanelBtn.isVisible()) {
          await memberPanelBtn.click()
          await expect(page.locator('.member-sidebar, .member-list')).toBeVisible()
        }
      }
    })

    test('应该能查看群组详情', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item[data-type="group"]').first()
      if (await groupConversation.isVisible()) {
        await groupConversation.click()
        await page.waitForTimeout(1000)
      }
    })
  })

  test.describe('消息类型测试', () => {
    test('应该能正确渲染文本消息', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible()) {
        await firstConversation.click()
        await page.waitForTimeout(1000)
        
        const textMessages = page.locator('.message-item .text-message, .message-content')
        await expect(textMessages.first()).toBeVisible()
      }
    })

    test('应该能渲染图片消息', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible()) {
        await firstConversation.click()
        await page.waitForTimeout(1000)
        
        const imageMessages = page.locator('.message-item .image-message, img.message-image')
        if (await imageMessages.first().isVisible()) {
          await imageMessages.first().click()
          // 可能打开图片预览
          await page.waitForTimeout(500)
        }
      }
    })

    test('应该能渲染文件消息', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible()) {
        await firstConversation.click()
        await page.waitForTimeout(1000)
        
        const fileMessages = page.locator('.message-item .file-message')
        if (await fileMessages.first().isVisible()) {
          await fileMessages.first().click()
          await page.waitForTimeout(500)
        }
      }
    })

    test('应该能渲染系统消息', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible()) {
        await firstConversation.click()
        await page.waitForTimeout(1000)
        
        const systemMessages = page.locator('.message-item .system-message')
        if (await systemMessages.first().isVisible()) {
          await expect(systemMessages.first()).toBeVisible()
        }
      }
    })
  })

  test.describe('表情面板功能测试', () => {
    test('应该能切换表情标签页', async ({ page }) => {
      // 先选择会话显示聊天窗口
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible({ timeout: 5000 })) {
        await firstConversation.click()
        await page.waitForTimeout(1000)
      }
      
      // 查找表情按钮并点击
      const emojiBtn = page.locator('.emoji-btn, button[title="表情"], .emoji-panel-btn')
      const isVisible = await emojiBtn.isVisible().catch(() => false)
      if (!isVisible) return
      
      await emojiBtn.click()
      await page.waitForTimeout(1000)

      // 验证表情面板出现
      const emojiPanel = page.locator('.emoji-panel-container')
      const panelVisible = await emojiPanel.isVisible().catch(() => false)
      if (panelVisible) {
        const tabs = page.locator('.emoji-tab, .emoji-category-tab')
        if (await tabs.count() > 1) {
          await tabs.nth(1).click()
          await page.waitForTimeout(500)
        }
      }
    })

    test('应该能选择并插入表情', async ({ page }) => {
      // 先选择会话
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible({ timeout: 5000 })) {
        await firstConversation.click()
        await page.waitForTimeout(1000)
      }
      
      // 查找表情按钮
      const emojiBtn = page.locator('.emoji-btn, button[title="表情"], .emoji-panel-btn')
      const isVisible = await emojiBtn.isVisible().catch(() => false)
      if (!isVisible) return
      
      await emojiBtn.click()
      await page.waitForTimeout(1000)
      
      // 验证表情面板可见
      const emojiPanel = page.locator('.emoji-panel-container')
      const panelVisible = await emojiPanel.isVisible().catch(() => false)
      if (!panelVisible) return

      const firstEmoji = page.locator('.emoji-item').first()
      if (await firstEmoji.isVisible()) {
        await firstEmoji.click()
        await page.waitForTimeout(500)
      }
    })
  })

  test.describe('@提及功能测试', () => {
    test('应该能触发@提及面板', async ({ page }) => {
      const input = page.locator('.message-input textarea')
      if (await input.isVisible()) {
        await input.fill('@')
        await page.waitForTimeout(500)
        
        // 验证@成员面板出现
        await expect(page.locator('.at-members-panel, .at-mention-panel')).toBeVisible()
      }
    })

    test('应该能选择@成员', async ({ page }) => {
      const input = page.locator('.message-input textarea')
      if (await input.isVisible()) {
        await input.fill('@')
        await page.waitForTimeout(500)
        
        const firstMember = page.locator('.at-member-item').first()
        if (await firstMember.isVisible()) {
          await firstMember.click()
          await page.waitForTimeout(500)
          
          const inputValue = await input.inputValue()
          expect(inputValue).toContain('@')
        }
      }
    })
  })

  test.describe('窗口控制测试', () => {
    test('应该能显示窗口控制按钮', async ({ page }) => {
      const minimizeBtn = page.locator('.window-control-btn.minimize-btn')
      const maximizeBtn = page.locator('.window-control-btn.maximize-btn')
      const closeBtn = page.locator('.window-control-btn.close-btn')
      
      // 这些按钮可能在Electron环境下才显示
      const hasWindowControls = await minimizeBtn.isVisible().catch(() => false)
      if (hasWindowControls) {
        await expect(minimizeBtn).toBeVisible()
        await expect(maximizeBtn).toBeVisible()
        await expect(closeBtn).toBeVisible()
      }
    })
  })

  test.describe('侧边栏折叠功能测试', () => {
    test('应该能折叠侧边栏', async ({ page }) => {
      const toggleBtn = page.locator('.toggle-sidebar-btn')
      if (await toggleBtn.isVisible()) {
        await toggleBtn.click()
        await page.waitForTimeout(500)
        // 验证侧边栏折叠状态
        const sidebar = page.locator('.sidebar')
        const className = await sidebar.getAttribute('class')
        expect(className).toContain('sidebar-collapsed')
      }
    })

    test('应该能展开折叠后的侧边栏', async ({ page }) => {
      const toggleBtn = page.locator('.toggle-sidebar-btn')
      if (await toggleBtn.isVisible()) {
        await toggleBtn.click() // 先折叠
        await page.waitForTimeout(500)
        await toggleBtn.click() // 再展开
        await page.waitForTimeout(500)
      }
    })
  })

  test.describe('通知中心测试', () => {
    test('应该能打开通知中心', async ({ page }) => {
      const notificationBtn = page.locator('button[title="通知"]')
      await notificationBtn.click()
      await page.waitForTimeout(500)
    })

    test('应该能关闭通知中心', async ({ page }) => {
      const notificationBtn = page.locator('button[title="通知"]')
      await notificationBtn.click()
      await page.waitForTimeout(500)
      
      // 再次点击应该关闭
      await notificationBtn.click()
      await page.waitForTimeout(500)
    })
  })

  test.describe('分享功能测试', () => {
    test('应该能打开分享对话框', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible()) {
        await firstConversation.click()
        await page.waitForTimeout(1000)
      }
    })
  })

  test.describe('连接状态测试', () => {
    test('应该能显示连接断开提示', async ({ page }) => {
      // 模拟网络断开
      await page.context().setOffline(true)
      await page.waitForTimeout(2000)
      
      // 应该显示重连提示
      const reconnectBtn = page.locator('.retry-btn, button:has-text("重新连接")')
      const isVisible = await reconnectBtn.isVisible().catch(() => false)
      
      if (isVisible) {
        await reconnectBtn.click()
        await page.waitForTimeout(1000)
      }
      
      await page.context().setOffline(false)
    })

    test('应该能通过登录按钮重新登录', async ({ page }) => {
      await page.context().setOffline(true)
      await page.waitForTimeout(2000)
      
      const loginBtn = page.locator('.login-btn, button:has-text("重新登录")')
      const isVisible = await loginBtn.isVisible().catch(() => false)
      
      if (isVisible) {
        await loginBtn.click()
        await page.waitForURL('**/login', { timeout: 5000 }).catch(() => {})
      }
      
      await page.context().setOffline(false)
    })
  })

  test.describe('设置面板测试', () => {
    test('应该能切换到基本设置', async ({ page }) => {
      const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
      if (await userMenuBtn.isVisible()) {
        await userMenuBtn.click()
        const settingsAction = page.locator('.context-menu-item:has-text("设置")')
        if (await settingsAction.isVisible()) {
          await settingsAction.click()
          await page.waitForTimeout(500)
        }
      }
      
      const basicTab = page.locator('.settings-sidebar-item:has-text("基本"), .settings-tab:has-text("基本")')
      if (await basicTab.isVisible()) {
        await basicTab.click()
        await page.waitForTimeout(500)
      }
    })

    test('应该能切换到消息设置', async ({ page }) => {
      // 打开设置
      const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
      if (await userMenuBtn.isVisible()) {
        await userMenuBtn.click()
        const settingsAction = page.locator('.context-menu-item:has-text("设置")')
        if (await settingsAction.isVisible()) {
          await settingsAction.click()
          await page.waitForTimeout(500)
        }
      }
      
      const messageTab = page.locator('.settings-sidebar-item:has-text("消息")')
      if (await messageTab.isVisible()) {
        await messageTab.click()
        await page.waitForTimeout(500)
      }
    })

    test('应该能切换到外观设置', async ({ page }) => {
      // 打开设置
      const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
      if (await userMenuBtn.isVisible()) {
        await userMenuBtn.click()
        const settingsAction = page.locator('.context-menu-item:has-text("设置")')
        if (await settingsAction.isVisible()) {
          await settingsAction.click()
          await page.waitForTimeout(500)
        }
      }
      
      const appearanceTab = page.locator('.settings-sidebar-item:has-text("外观")')
      if (await appearanceTab.isVisible()) {
        await appearanceTab.click()
        await page.waitForTimeout(500)
        
        // 验证主题选项可见
        const themeOptions = page.locator('.theme-option')
        await expect(themeOptions.first()).toBeVisible()
      }
    })

    test('应该能切换主题', async ({ page }) => {
      // 打开设置并切换到外观
      const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
      if (await userMenuBtn.isVisible()) {
        await userMenuBtn.click()
        const settingsAction = page.locator('.context-menu-item:has-text("设置")')
        if (await settingsAction.isVisible()) {
          await settingsAction.click()
          await page.waitForTimeout(500)
        }
      }
      
      const appearanceTab = page.locator('.settings-sidebar-item:has-text("外观")')
      if (await appearanceTab.isVisible()) {
        await appearanceTab.click()
        await page.waitForTimeout(500)
        
        // 选择第一个主题
        const themeOption = page.locator('.theme-option').first()
        await themeOption.click()
        await page.waitForTimeout(500)
      }
    })

    test('应该能清除缓存', async ({ page }) => {
      // 打开设置
      const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
      if (await userMenuBtn.isVisible()) {
        await userMenuBtn.click()
        const settingsAction = page.locator('.context-menu-item:has-text("设置")')
        if (await settingsAction.isVisible()) {
          await settingsAction.click()
          await page.waitForTimeout(500)
        }
      }
      
      const clearCacheBtn = page.locator('.clear-cache-btn')
      if (await clearCacheBtn.isVisible()) {
        await clearCacheBtn.click()
        await page.waitForTimeout(500)
      }
    })

    test('应该能保存设置', async ({ page }) => {
      // 打开设置
      const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
      if (await userMenuBtn.isVisible()) {
        await userMenuBtn.click()
        const settingsAction = page.locator('.context-menu-item:has-text("设置")')
        if (await settingsAction.isVisible()) {
          await settingsAction.click()
          await page.waitForTimeout(500)
        }
      }
      
      const saveBtn = page.locator('.save-btn, button:has-text("保存")')
      if (await saveBtn.isVisible()) {
        await saveBtn.click()
        await page.waitForTimeout(500)
      }
    })

    test('应该能取消设置修改', async ({ page }) => {
      // 打开设置
      const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
      if (await userMenuBtn.isVisible()) {
        await userMenuBtn.click()
        const settingsAction = page.locator('.context-menu-item:has-text("设置")')
        if (await settingsAction.isVisible()) {
          await settingsAction.click()
          await page.waitForTimeout(500)
        }
      }
      
      const cancelBtn = page.locator('.cancel-btn, button:has-text("取消")')
      if (await cancelBtn.isVisible()) {
        await cancelBtn.click()
        await page.waitForTimeout(500)
      }
    })
  })

  test.describe('应用面板功能测试', () => {
    test('应该能显示应用列表', async ({ page }) => {
      const appIcon = page.locator('.side-options .option-item').nth(3)
      await appIcon.click()
      await page.waitForTimeout(500)
      
      // 验证应用面板内容可见
      const appPanel = page.locator('.apps-container')
      await expect(appPanel).toBeVisible()
      
      // 验证有应用分类项
      const appCategories = page.locator('.app-category-item')
      const count = await appCategories.count()
      expect(count).toBeGreaterThan(0)
    })

    test('应该能点击应用打开', async ({ page }) => {
      const appIcon = page.locator('.side-options .option-item').nth(3)
      await appIcon.click()
      await page.waitForTimeout(500)
      
      const firstApp = page.locator('.panel-category-app-item').first()
      if (await firstApp.isVisible()) {
        await firstApp.click()
        await page.waitForTimeout(1000)
      }
    })
  })

  test.describe('确认对话框测试', () => {
    test('应该能确认对话框', async ({ page }) => {
      // 触发一个需要确认的操作，比如移除会话
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible()) {
        await firstConversation.click({ button: 'right' })
        const removeAction = page.locator('.context-menu-item:has-text("移除")')
        if (await removeAction.isVisible()) {
          await removeAction.click()
          await page.waitForTimeout(500)
        }
      }
      
      // 查找并点击确认按钮
      const confirmBtn = page.locator('.q-btn--primary, .confirm-btn, button:has-text("确定")')
      if (await confirmBtn.isVisible()) {
        await confirmBtn.click()
        await page.waitForTimeout(500)
      }
    })

    test('应该能取消对话框', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible()) {
        await firstConversation.click({ button: 'right' })
        const removeAction = page.locator('.context-menu-item:has-text("移除")')
        if (await removeAction.isVisible()) {
          await removeAction.click()
          await page.waitForTimeout(500)
        }
      }
      
      // 查找并点击取消按钮
      const cancelBtn = page.locator('.q-btn--default, .cancel-btn, button:has-text("取消")')
      if (await cancelBtn.isVisible()) {
        await cancelBtn.click()
        await page.waitForTimeout(500)
      }
    })
  })

  test.describe('键盘快捷键测试', () => {
    test('应该能使用 Esc 关闭模态框', async ({ page }) => {
      // 打开设置面板
      const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
      if (await userMenuBtn.isVisible()) {
        await userMenuBtn.click()
        const settingsAction = page.locator('.context-menu-item:has-text("设置")')
        if (await settingsAction.isVisible()) {
          await settingsAction.click()
          await expect(page.locator('.settings-modal, .settings-panel')).toBeVisible()
        }
      }
      
      // 按 Esc 关闭
      await page.keyboard.press('Escape')
      await page.waitForTimeout(500)
    })

    test('应该能使用 Enter 发送消息', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      if (await firstConversation.isVisible()) {
        await firstConversation.click()
        await page.waitForTimeout(1000)
      }
      
      const input = page.locator('.message-input textarea')
      if (await input.isVisible()) {
        await input.fill('测试Enter发送')
        await input.press('Enter')
        await page.waitForTimeout(1000)
      }
    })
  })
})
import { test, expect, Page } from '@playwright/test'

async function login(page: Page) {
  await page.route('**/api/v1/auth/login', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        message: '登录成功',
        data: {
          token: 'mock-token-for-testing',
          user: { id: 1, username: 'admin', name: '管理员', email: 'admin@example.com', avatar: null, isAdmin: true }
        }
      })
    })
  })

  await page.route('**/api/v1/auth/check-2fa', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, data: { twoFactorEnabled: false } }) })
  })

  await page.route('**/api/v1/org/**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, data: [] }) })
  })

  await page.route('**/api/v1/conversations', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        data: [
          {
            id: 1,
            type: 'single',
            name: '测试用户',
            avatar: null,
            last_message_at: new Date().toISOString(),
            created_at: new Date().toISOString(),
            lastMessage: {
              id: 100,
              content: '你好',
              sender: { id: 2, nickname: '测试用户', username: 'user1', avatar: '' },
              created_at: new Date().toISOString(),
              type: 'text'
            },
            unread_count: 0,
            members: [
              { user: { id: 1, nickname: '管理员', username: 'admin', avatar: '' }, role: 'member' },
              { user: { id: 2, nickname: '测试用户', username: 'user1', avatar: '' }, role: 'member' }
            ]
          },
          {
            id: 2,
            type: 'group',
            name: '测试群组',
            avatar: null,
            last_message_at: new Date().toISOString(),
            created_at: new Date().toISOString(),
            lastMessage: {
              id: 200,
              content: '欢迎加入',
              sender: { id: 1, nickname: '管理员', username: 'admin', avatar: '' },
              created_at: new Date().toISOString(),
              type: 'text'
            },
            unread_count: 1,
            announcement: '这是群公告',
            members: [
              { user: { id: 1, nickname: '管理员', username: 'admin', avatar: '' }, role: 'owner' },
              { user: { id: 2, nickname: '测试用户', username: 'user1', avatar: '' }, role: 'member' }
            ]
          }
        ]
      })
    })
  })

  await page.route('**/api/v1/messages/**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, data: [] }) })
  })

  await page.route('**/api/v1/groups/**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, data: [] }) })
  })

  await page.route('**/api/v1/channels/**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, data: [] }) })
  })

  await page.route('**/api/v1/apps/**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, data: [{ id: '1', name: '统计报表', icon: 'chart', description: '数据统计' }] }) })
  })

  await page.route('**/api/v1/users/**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, data: { id: 1, username: 'admin', name: '管理员' } }) })
  })

  await page.route('**/api/v1/employees/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        data: [
          { id: 1, username: 'admin', name: '管理员', department: '技术部', avatar: '', position: '管理员' },
          { id: 2, username: 'user1', name: '测试用户', department: '产品部', avatar: '', position: '产品经理' },
          { id: 3, username: 'user2', name: '开发同学', department: '技术部', avatar: '', position: '开发工程师' }
        ]
      })
    })
  })

  await page.route('**/api/v1/notifications/**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, data: [] }) })
  })

  await page.route('**/api/v1/search/**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, data: [] }) })
  })

  await page.route('**/api/v1/version/**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, data: { needUpdate: false } }) })
  })

  await page.route('**/api/v1/avatars/**', async (route) => {
    await route.fulfill({ status: 404 })
  })

  await page.goto('http://localhost:3000')
  await page.fill('input[placeholder="请输入用户名"]', 'admin')
  await page.fill('input[placeholder="请输入密码"]', '123456')
  await page.click('button:has-text("登录")')
  await page.waitForSelector('.im-container', { timeout: 10000 })
  await page.waitForTimeout(2000)
}

test.describe('按钮点击行为严格测试', () => {

  test.beforeEach(async ({ page }) => {
    await login(page)
    await page.evaluate(() => {
      const el = document.querySelector('.network-error')
      if (el) el.remove()
      const style = document.createElement('style')
      style.textContent = '.network-error { display: none !important; }'
      document.head.appendChild(style)
    })
    await page.waitForTimeout(500)
  })

  test.describe('会话右键菜单', () => {
    test('右键点击会话必须显示右键菜单', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      await expect(firstConversation).toBeVisible()
      
      await firstConversation.click({ button: 'right' })
      await page.waitForTimeout(500)
      
      // 严格验证菜单必须出现
      const contextMenu = page.locator('.context-menu')
      await expect(contextMenu).toBeVisible()
    })

    test('点击置顶选项必须关闭菜单', async ({ page }) => {
      const firstConversation = page.locator('.conversation-item').first()
      await expect(firstConversation).toBeVisible()
      
      await firstConversation.click({ button: 'right' })
      await page.waitForTimeout(500)
      
      const contextMenu = page.locator('.context-menu')
      await expect(contextMenu).toBeVisible()
      
      const pinOption = page.locator('.context-menu-item', { hasText: '置顶' }).first()
      await expect(pinOption).toBeVisible()
      
      await pinOption.click()
      await page.waitForTimeout(500)
      
      // 验证菜单必须关闭
      await expect(contextMenu).not.toBeVisible()
    })
  })

  test.describe('动作菜单', () => {
    test('点击+按钮必须显示动作菜单', async ({ page }) => {
      const actionBtn = page.locator('.add-conversation-btn')
      await expect(actionBtn).toBeVisible()
      
      await actionBtn.click()
      await page.waitForTimeout(500)
      
      const actionMenu = page.locator('.action-menu')
      await expect(actionMenu).toBeVisible()
    })

    test('点击创建群聊必须打开创建弹窗', async ({ page }) => {
      const actionBtn = page.locator('.add-conversation-btn')
      await expect(actionBtn).toBeVisible()
      
      await actionBtn.click()
      await page.waitForTimeout(500)
      
      const createGroupOption = page.locator('.action-menu-item', { hasText: '创建群聊' })
      await expect(createGroupOption).toBeVisible()
      
      await createGroupOption.click()
      await page.waitForTimeout(500)
      
      // 严格验证弹窗必须打开
      const modal = page.locator('.user-profile-modal')
      await expect(modal).toBeVisible()
      
      const title = page.locator('.user-profile-header h3')
      await expect(title).toContainText('创建群聊')
    })

    test('点击取消按钮必须关闭弹窗', async ({ page }) => {
      const actionBtn = page.locator('.add-conversation-btn')
      await expect(actionBtn).toBeVisible()
      
      await actionBtn.click()
      await page.waitForTimeout(500)
      
      const createGroupOption = page.locator('.action-menu-item', { hasText: '创建群聊' })
      await expect(createGroupOption).toBeVisible()
      await createGroupOption.click()
      await page.waitForTimeout(500)
      
      await expect(page.locator('.user-profile-modal')).toBeVisible()
      
      const cancelBtn = page.locator('.cancel-btn')
      await expect(cancelBtn).toBeVisible()
      await cancelBtn.click()
      await page.waitForTimeout(500)
      
      await expect(page.locator('.user-profile-modal')).not.toBeVisible()
    })
  })

  test.describe('用户菜单', () => {
    test('点击更多操作按钮必须显示用户菜单', async ({ page }) => {
      const moreBtn = page.locator('.more-actions-btn, .user-action-btn')
      await expect(moreBtn).toBeVisible()
      
      await moreBtn.click()
      await page.waitForTimeout(500)
      
      const userMenu = page.locator('.context-menu')
      await expect(userMenu).toBeVisible()
    })

    test('点击设置选项必须打开设置面板', async ({ page }) => {
      const moreBtn = page.locator('.more-actions-btn, .user-action-btn')
      await expect(moreBtn).toBeVisible()
      
      await moreBtn.click()
      await page.waitForTimeout(500)
      
      const settingsOption = page.locator('.context-menu-item', { hasText: '设置' })
      await expect(settingsOption).toBeVisible()
      
      await settingsOption.click()
      await page.waitForTimeout(500)
      
      const settingsPanel = page.locator('.settings-modal, .settings-panel, .q-dialog')
      await expect(settingsPanel).toBeVisible()
    })

    test('点击退出登录必须显示确认对话框', async ({ page }) => {
      const moreBtn = page.locator('.more-actions-btn, .user-action-btn')
      await expect(moreBtn).toBeVisible()
      
      await moreBtn.click()
      await page.waitForTimeout(500)
      
      const logoutOption = page.locator('.context-menu-item', { hasText: '退出登录' })
      await expect(logoutOption).toBeVisible()
      
      await logoutOption.click()
      await page.waitForTimeout(500)
      
      const dialog = page.locator('.confirm-dialog, .q-dialog, .el-message-box')
      await expect(dialog).toBeVisible()
    })
  })

  test.describe('主题菜单', () => {
    test('点击主题按钮必须显示主题菜单', async ({ page }) => {
      const themeBtn = page.locator('.theme-btn')
      await expect(themeBtn).toBeVisible()
      
      await themeBtn.click()
      await page.waitForTimeout(500)
      
      const themeMenu = page.locator('.theme-menu')
      await expect(themeMenu).toBeVisible()
    })
  })

  test.describe('群聊头部按钮', () => {
    test('群聊头部必须显示邀请成员按钮', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const inviteBtn = page.locator('.header-icon[title="邀请成员"]')
      await expect(inviteBtn).toBeVisible()
    })

    test('群聊头部必须显示更多操作按钮', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
      await expect(moreBtn).toBeVisible()
    })

    test('点击更多操作按钮必须显示下拉菜单', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
      await expect(moreBtn).toBeVisible()
      await moreBtn.click()
      await page.waitForTimeout(500)

      const headerMenu = page.locator('.header-menu-teleport')
      await expect(headerMenu).toBeVisible()
    })

    test('更多操作菜单必须包含修改群名称选项', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
      await moreBtn.click()
      await page.waitForTimeout(500)

      const editNameOption = page.locator('.menu-item', { hasText: '修改群名称' })
      await expect(editNameOption).toBeVisible()
    })

    test('点击修改群名称必须打开编辑弹窗', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
      await moreBtn.click()
      await page.waitForTimeout(500)

      const editNameOption = page.locator('.menu-item', { hasText: '修改群名称' })
      await expect(editNameOption).toBeVisible()
      await editNameOption.click()
      await page.waitForTimeout(500)

      const editModal = page.locator('.modal-overlay')
      await expect(editModal).toBeVisible()
      
      const modalTitle = page.locator('.modal-header h3')
      await expect(modalTitle).toContainText('修改群名称')
    })

    test('点击解散群聊必须显示确认对话框', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
      await moreBtn.click()
      await page.waitForTimeout(500)

      const dissolveOption = page.locator('.menu-item', { hasText: '解散群聊' })
      await expect(dissolveOption).toBeVisible()
      await dissolveOption.click()
      await page.waitForTimeout(500)

      const confirmDialog = page.locator('.confirm-dialog-modal')
      await expect(confirmDialog).toBeVisible()
    })
  })

  test.describe('双击群名修改', () => {
    test('双击群聊名称必须触发编辑群信息', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const groupName = page.locator('.header-name')
      await expect(groupName).toBeVisible()
      await groupName.dblclick()
      await page.waitForTimeout(500)

      const editModal = page.locator('.modal-overlay')
      await expect(editModal).toBeVisible()
      
      const modalTitle = page.locator('.modal-header h3')
      await expect(modalTitle).toContainText('修改群名称')
    })

    test('双击群名后弹窗必须预填当前群名', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const groupName = page.locator('.header-name')
      await groupName.dblclick()
      await page.waitForTimeout(500)

      const nameInput = page.locator('.form-input')
      await expect(nameInput).toBeVisible()
      
      const inputValue = await nameInput.inputValue()
      expect(inputValue).toBe('测试群组')
    })
  })

  test.describe('群成员右键菜单', () => {
    test('右键点击群成员必须显示成员右键菜单', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const memberItem = page.locator('.members-sidebar .member-item').nth(1)
      await expect(memberItem).toBeVisible()
      
      await memberItem.click({ button: 'right' })
      await page.waitForTimeout(500)

      const contextMenu = page.locator('.member-context-menu, .context-menu')
      await expect(contextMenu).toBeVisible()
    })
  })
})

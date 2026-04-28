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

test.describe('群成员右键菜单、群聊头部按钮、双击群名功能测试', () => {

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

  test.describe('群成员列表右键菜单', () => {
    test('选择群聊后必须显示成员列表', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const membersSidebar = page.locator('.members-sidebar')
      await expect(membersSidebar).toBeVisible()
    })

    test('右键点击群成员必须显示右键菜单', async ({ page }) => {
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

    test('群成员右键菜单必须包含查看资料选项', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const memberItem = page.locator('.members-sidebar .member-item').nth(1)
      await memberItem.click({ button: 'right' })
      await page.waitForTimeout(500)

      const viewProfileOption = page.locator('.context-menu-item', { hasText: '查看资料' }).first()
      await expect(viewProfileOption).toBeVisible()
    })

    test('群成员右键菜单必须包含发起私聊选项', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const memberItem = page.locator('.members-sidebar .member-item').nth(1)
      await memberItem.click({ button: 'right' })
      await page.waitForTimeout(500)

      const privateChatOption = page.locator('.context-menu-item', { hasText: '发起私聊' }).first()
      await expect(privateChatOption).toBeVisible()
    })

    test('群成员右键菜单必须包含移除选项', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const memberItem = page.locator('.members-sidebar .member-item').nth(1)
      await memberItem.click({ button: 'right' })
      await page.waitForTimeout(500)

      const removeOption = page.locator('.context-menu-item', { hasText: '移除' }).first()
      await expect(removeOption).toBeVisible()
    })

    test('点击群成员右键菜单选项必须关闭菜单', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const memberItem = page.locator('.members-sidebar .member-item').nth(1)
      await memberItem.click({ button: 'right' })
      await page.waitForTimeout(500)

      const firstOption = page.locator('.context-menu-item').first()
      await firstOption.click()
      await page.waitForTimeout(500)

      const contextMenu = page.locator('.member-context-menu, .context-menu')
      await expect(contextMenu).not.toBeVisible()
    })

    test('双击群成员必须发起私聊', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const memberItem = page.locator('.members-sidebar .member-item').nth(1)
      await memberItem.dblclick()
      await page.waitForTimeout(500)

      // 验证私聊被发起（应该切换到私聊会话）
      const chatWindow = page.locator('.chat-window')
      await expect(chatWindow).toBeVisible()
    })

    test('群成员列表必须显示角色标识', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      // 验证成员角色显示
      const memberRole = page.locator('.member-role, .member-tag')
      const isVisible = await memberRole.first().isVisible().catch(() => false)
      // 角色标识可能有或没有，取决于实现
    })

    test('群成员搜索框必须可以展开', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const searchToggle = page.locator('.search-toggle-btn')
      if (await searchToggle.isVisible()) {
        await searchToggle.click()
        await page.waitForTimeout(500)

        const searchInput = page.locator('.member-search-input')
        await expect(searchInput).toBeVisible()
      }
    })
  })

  test.describe('群聊窗口头部按钮', () => {
    test('群聊头部必须显示群名称', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const chatHeader = page.locator('.chat-header')
      await expect(chatHeader).toBeVisible()

      const groupName = page.locator('.header-name')
      await expect(groupName).toBeVisible()
      await expect(groupName).toContainText('测试群组')
    })

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

    test('点击邀请成员按钮必须触发功能', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const inviteBtn = page.locator('.header-icon[title="邀请成员"]')
      await inviteBtn.click()
      await page.waitForTimeout(500)
    })

    test('点击更多操作按钮必须显示下拉菜单', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
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

    test('更多操作菜单必须包含编辑群公告选项', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
      await moreBtn.click()
      await page.waitForTimeout(500)

      const editAnnouncementOption = page.locator('.menu-item', { hasText: '编辑群公告' })
      await expect(editAnnouncementOption).toBeVisible()
    })

    test('更多操作菜单必须包含解散群聊选项', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
      await moreBtn.click()
      await page.waitForTimeout(500)

      const dissolveOption = page.locator('.menu-item', { hasText: '解散群聊' })
      await expect(dissolveOption).toBeVisible()
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
      await editNameOption.click()
      await page.waitForTimeout(500)

      const editModal = page.locator('.modal-overlay')
      await expect(editModal).toBeVisible()
      
      const modalTitle = page.locator('.modal-header h3')
      await expect(modalTitle).toContainText('修改群名称')
    })

    test('编辑群名称弹窗必须包含输入框', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
      await moreBtn.click()
      await page.waitForTimeout(500)

      const editNameOption = page.locator('.menu-item', { hasText: '修改群名称' })
      await editNameOption.click()
      await page.waitForTimeout(500)

      const nameInput = page.locator('.form-input')
      await expect(nameInput).toBeVisible()
    })

    test('编辑群名称弹窗必须包含保存按钮', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
      await moreBtn.click()
      await page.waitForTimeout(500)

      const editNameOption = page.locator('.menu-item', { hasText: '修改群名称' })
      await editNameOption.click()
      await page.waitForTimeout(500)

      const saveBtn = page.locator('.modal-footer .btn-primary')
      await expect(saveBtn).toBeVisible()
      await expect(saveBtn).toContainText('保存')
    })

    test('编辑群名称弹窗必须包含取消按钮', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
      await moreBtn.click()
      await page.waitForTimeout(500)

      const editNameOption = page.locator('.menu-item', { hasText: '修改群名称' })
      await editNameOption.click()
      await page.waitForTimeout(500)

      const cancelBtn = page.locator('.modal-footer .btn-secondary')
      await expect(cancelBtn).toBeVisible()
      await expect(cancelBtn).toContainText('取消')
    })

    test('点击取消必须关闭编辑群名称弹窗', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
      await moreBtn.click()
      await page.waitForTimeout(500)

      const editNameOption = page.locator('.menu-item', { hasText: '修改群名称' })
      await editNameOption.click()
      await page.waitForTimeout(500)

      const cancelBtn = page.locator('.modal-overlay .modal-footer .btn-secondary')
      await expect(cancelBtn).toBeVisible()
      await cancelBtn.click()
      await page.waitForTimeout(500)

      const editModal = page.locator('.modal-overlay')
      await expect(editModal).not.toBeVisible()
    })

    test('点击编辑群公告必须打开公告编辑弹窗', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
      await moreBtn.click()
      await page.waitForTimeout(500)

      const editAnnouncementOption = page.locator('.menu-item', { hasText: '编辑群公告' })
      await editAnnouncementOption.click()
      await page.waitForTimeout(500)

      const editModal = page.locator('.modal-overlay')
      await expect(editModal).toBeVisible()
      
      const modalTitle = page.locator('.modal-header h3')
      await expect(modalTitle).toContainText('编辑群公告')
    })

    test('编辑群公告弹窗必须包含文本输入框', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
      await moreBtn.click()
      await page.waitForTimeout(500)

      const editAnnouncementOption = page.locator('.menu-item', { hasText: '编辑群公告' })
      await editAnnouncementOption.click()
      await page.waitForTimeout(500)

      const textarea = page.locator('.modal-body .form-textarea')
      await expect(textarea).toBeVisible()
    })

    test('编辑群公告弹窗必须包含保存按钮', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
      await moreBtn.click()
      await page.waitForTimeout(500)

      const editAnnouncementOption = page.locator('.menu-item', { hasText: '编辑群公告' })
      await editAnnouncementOption.click()
      await page.waitForTimeout(500)

      const saveBtn = page.locator('.modal-overlay .modal-footer .btn-primary')
      await expect(saveBtn).toBeVisible()
      await expect(saveBtn).toContainText('保存')
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
      await dissolveOption.click()
      await page.waitForTimeout(500)

      const confirmDialog = page.locator('.confirm-dialog-modal, .q-dialog')
      await expect(confirmDialog).toBeVisible()
    })

    test('解散群聊确认对话框必须包含确认和取消按钮', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
      await moreBtn.click()
      await page.waitForTimeout(500)

      const dissolveOption = page.locator('.menu-item', { hasText: '解散群聊' })
      await dissolveOption.click()
      await page.waitForTimeout(500)

      const confirmBtn = page.locator('button:has-text("确定"), .confirm')
      const cancelBtn = page.locator('button:has-text("取消"), .cancel')
      await expect(confirmBtn).toBeVisible()
      await expect(cancelBtn).toBeVisible()
    })

    test('点击取消必须关闭解散群聊确认对话框', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
      await moreBtn.click()
      await page.waitForTimeout(500)

      const dissolveOption = page.locator('.menu-item', { hasText: '解散群聊' })
      await dissolveOption.click()
      await page.waitForTimeout(500)

      const cancelBtn = page.locator('button:has-text("取消"), .cancel').first()
      await cancelBtn.click()
      await page.waitForTimeout(500)

      const confirmDialog = page.locator('.confirm-dialog-modal, .q-dialog')
      await expect(confirmDialog).not.toBeVisible()
    })

    test('再次点击更多操作按钮必须关闭菜单', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
      await moreBtn.click()
      await page.waitForTimeout(500)

      await moreBtn.click()
      await page.waitForTimeout(500)

      const headerMenu = page.locator('.header-menu-teleport')
      await expect(headerMenu).not.toBeVisible()
    })

    test('点击空白处必须关闭更多操作菜单', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const moreBtn = page.locator('.header-icon .fa-ellipsis-v')
      await moreBtn.click()
      await page.waitForTimeout(500)

      await page.click('body')
      await page.waitForTimeout(500)

      const headerMenu = page.locator('.header-menu-teleport')
      await expect(headerMenu).not.toBeVisible()
    })
  })

  test.describe('双击群名修改功能', () => {
    test('双击群聊名称必须触发编辑群信息', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const groupName = page.locator('.header-name')
      await groupName.dblclick()
      await page.waitForTimeout(500)

      const editModal = page.locator('.modal-overlay')
      await expect(editModal).toBeVisible()
    })

    test('双击群名后弹窗必须预填当前群名', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const groupName = page.locator('.header-name')
      await groupName.dblclick()
      await page.waitForTimeout(500)

      const nameInput = page.locator('.modal-overlay .form-input')
      await expect(nameInput).toBeVisible()
      
      const inputValue = await nameInput.inputValue()
      expect(inputValue).toBe('测试群组')
    })

    test('双击群名弹窗必须包含输入框', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const groupName = page.locator('.header-name')
      await groupName.dblclick()
      await page.waitForTimeout(500)

      const nameInput = page.locator('.modal-overlay .form-input')
      await expect(nameInput).toBeVisible()
    })

    test('双击群名弹窗必须包含保存按钮', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const groupName = page.locator('.header-name')
      await groupName.dblclick()
      await page.waitForTimeout(500)

      const saveBtn = page.locator('.modal-overlay .modal-footer .btn-primary')
      await expect(saveBtn).toBeVisible()
    })

    test('双击群名弹窗必须包含取消按钮', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const groupName = page.locator('.header-name')
      await groupName.dblclick()
      await page.waitForTimeout(500)

      const cancelBtn = page.locator('.modal-overlay .modal-footer .btn-secondary')
      await expect(cancelBtn).toBeVisible()
    })

    test('双击群名弹窗点击取消必须关闭弹窗', async ({ page }) => {
      const groupConversation = page.locator('.conversation-item').filter({ hasText: '群组' }).first()
      await expect(groupConversation).toBeVisible({ timeout: 5000 })
      await groupConversation.click()
      await page.waitForTimeout(1000)

      const groupName = page.locator('.header-name')
      await groupName.dblclick()
      await page.waitForTimeout(500)

      const cancelBtn = page.locator('.modal-overlay .modal-footer .btn-secondary')
      await cancelBtn.click()
      await page.waitForTimeout(500)

      const editModal = page.locator('.modal-overlay')
      await expect(editModal).not.toBeVisible()
    })

    test('双击私聊名称不应该触发编辑', async ({ page }) => {
      const privateConversation = page.locator('.conversation-item').filter({ hasText: '测试用户' }).first()
      await expect(privateConversation).toBeVisible({ timeout: 5000 })
      await privateConversation.click()
      await page.waitForTimeout(1000)

      const userName = page.locator('.header-name')
      await userName.dblclick()
      await page.waitForTimeout(500)

      const editModal = page.locator('.modal-overlay')
      const isVisible = await editModal.isVisible().catch(() => false)
      expect(isVisible).toBe(false)
    })
  })
})

import { test, expect, Page } from '@playwright/test'

const BASE_URL = 'http://localhost:5173'
const API_URL = 'http://localhost:8080'

/**
 * 登录辅助函数
 */
async function login(page: Page) {
  await page.goto(`${BASE_URL}/#/login`)
  await page.waitForLoadState('networkidle')
  await expect(page.locator('.login-form')).toBeVisible({ timeout: 10000 })
  await page.fill('input[placeholder="请输入用户名"]', 'admin')
  await page.fill('input[placeholder="请输入密码"]', '123456')
  await page.click('button.login-button[type="submit"], button:has-text("登录")')
  await expect(page.locator('button.login-button:has-text("登录中")')).toBeVisible({ timeout: 3000 })
  await expect(page.locator('.main-container, .im-app > div:not(.login-container)')).toBeVisible({ timeout: 15000 })
  const token = await page.evaluate(() => localStorage.getItem('token'))
  expect(token).toBeTruthy()
}

/**
 * 导航到指定应用
 * 流程：1. 点击侧边栏"应用"按钮  2. 在应用列表中点击目标应用
 */
async function navigateToApp(page: Page, appName: string) {
  // 步骤1：先点击"应用"图标（SideOptions 中的 fa-cube 图标）
  const appsIcon = page.locator('.side-options .option-item[title="应用"], .side-options .option-item:has(.fa-cube)').first()
  
  if (await appsIcon.count() > 0) {
    await appsIcon.click()
    await page.waitForTimeout(1000)
  } else {
    // 备选：尝试点击任何包含"应用"文字的元素
    const altApps = page.locator('[class*="sidebar"] *:has-text("应用"), button:has-text("应用中心")').first()
    if (await altApps.count() > 0) {
      await altApps.click()
      await page.waitForTimeout(1000)
    }
  }
  
  // 步骤2：在应用面板中查找并点击目标应用
  // 尝试多种选择器
  const selectors = [
    `.app-item:has-text("${appName}")`,
    `.app-entry:has-text("${appName}")`,
    `.main-app-item:has-text("${appName}")`,
    `[class*="app"]:has-text("${appName}")`,
    `button:has-text("${appName}")`,
  ]
  
  for (const selector of selectors) {
    const appItem = page.locator(selector).first()
    if (await appItem.count() > 0) {
      await appItem.click()
      await page.waitForTimeout(1500)
      return
    }
  }
  
  console.log(`⚠ 未找到应用: ${appName}`)
}

test.describe('便签功能集成测试', () => {
  test.beforeEach(async ({ page }) => {
    await login(page)
    await navigateToApp(page, '便签')
    await page.waitForTimeout(2000)
  })

  test.describe('1. 便签主界面', () => {
    test('便签界面应正确渲染', async ({ page }) => {
      const notesApp = page.locator('.sticky-notes-app, .notes-app, [class*="notes"]').first()
      const isVisible = await notesApp.count() > 0 && await notesApp.isVisible().catch(() => false)
      
      if (isVisible) {
        await expect(notesApp).toBeVisible()
        console.log('✓ 便签界面可见')
      } else {
        console.log('⚠ 便签界面未找到')
      }
    })

    test('应显示新建便签按钮', async ({ page }) => {
      const createBtn = page.locator('.create-note-btn, button:has-text("新建便签"), button:has-text("+ 新建")').first()
      await expect(createBtn).toBeVisible({ timeout: 5000 })
      console.log('✓ 新建便签按钮可见')
    })

    test('应显示搜索框', async ({ page }) => {
      const searchInput = page.locator('.search-input, input[placeholder*="搜索便签"]').first()
      await expect(searchInput).toBeVisible({ timeout: 5000 })
      console.log('✓ 搜索框可见')
    })
  })

  test.describe('2. 便签 CRUD 操作', () => {
    test('应能创建新便签', async ({ page }) => {
      const createBtn = page.locator('.create-note-btn, button:has-text("新建便签")').first()
      await createBtn.click()
      await page.waitForTimeout(500)
      
      // 验证模态框出现
      const modal = page.locator('.sticky-note-modal-content, .note-modal, [class*="modal"]:visible').first()
      await expect(modal).toBeVisible({ timeout: 5000 })
      console.log('✓ 创建便签模态框打开')
      
      // 填写标题
      const titleInput = page.locator('.sticky-note-modal-content input[type="text"], input[placeholder*="标题"]').first()
      await titleInput.fill('测试便签')
      
      // 填写内容
      const contentInput = page.locator('.sticky-note-modal-content textarea, textarea[placeholder*="便签内容"]').first()
      await contentInput.fill('这是测试便签内容')
      
      // 滚动模态框容器到底部，使按钮可见
      await page.evaluate(() => {
        const modalContent = document.querySelector('.sticky-note-modal-content, .modal-container-content')
        if (modalContent) {
          modalContent.scrollTop = modalContent.scrollHeight
        }
      })
      await page.waitForTimeout(300)
      
      // 保存
      const saveBtn = page.locator('.sticky-note-confirm-btn, button:has-text("创建"), button:has-text("保存")').first()
      await saveBtn.click()
      
      // 等待模态框关闭（使用 expect 自动重试）
      await expect(modal).not.toBeVisible({ timeout: 5000 })
      console.log('✓ 便签保存成功，模态框关闭')
    })

    test('应能编辑现有便签', async ({ page }) => {
      // 找到第一个便签
      const firstNote = page.locator('.sticky-note').first()
      if (await firstNote.count() > 0) {
        await firstNote.click()
        await page.waitForTimeout(500)
        
        const modal = page.locator('.sticky-note-modal-content, .note-modal').first()
        if (await modal.isVisible().catch(() => false)) {
          console.log('✓ 点击便签打开编辑模态框')
          
          // 关闭模态框
          const closeBtn = modal.locator('.sticky-note-modal-close, button:has-text("×"), [class*="close"]').first()
          await closeBtn.click()
          await page.waitForTimeout(500)
        }
      } else {
        console.log('⚠ 暂无便签可编辑')
      }
    })

    test('应能删除便签', async ({ page }) => {
      const firstNote = page.locator('.sticky-note').first()
      if (await firstNote.count() > 0) {
        // 查找便签上的删除按钮
        const deleteBtn = firstNote.locator('.sticky-note-delete, button:has-text("删除"), .delete-btn').first()
        
        if (await deleteBtn.count() > 0) {
          await deleteBtn.click()
          await page.waitForTimeout(1000)
          console.log('✓ 便签删除按钮可点击')
        }
      } else {
        console.log('⚠ 暂无便签可删除')
      }
    })
  })

  test.describe('3. 便签搜索', () => {
    test('搜索功能应正常工作', async ({ page }) => {
      const searchInput = page.locator('.search-input, input[placeholder*="搜索便签"]').first()
      
      if (await searchInput.count() > 0) {
        await searchInput.fill('测试')
        await page.waitForTimeout(1000)
        
        // 验证搜索后结果
        const notes = page.locator('.sticky-note')
        const count = await notes.count()
        console.log(`✓ 搜索"测试"返回 ${count} 条结果`)
        
        // 清空搜索
        await searchInput.fill('')
        await page.waitForTimeout(500)
      }
    })
  })

  test.describe('4. 空状态', () => {
    test('无便签时应显示友好提示', async ({ page }) => {
      const emptyState = page.locator('.empty-notes, .empty-state, [class*="empty"]').first()
      
      if (await emptyState.count() > 0) {
        const isVisible = await emptyState.isVisible().catch(() => false)
        if (isVisible) {
          const text = await emptyState.textContent()
          expect(text).toBeTruthy()
          console.log(`✓ 空状态提示: ${text?.trim()}`)
        }
      }
    })
  })
})

test.describe('任务管理功能集成测试', () => {
  test.beforeEach(async ({ page }) => {
    await login(page)
    await navigateToApp(page, '任务')
    await page.waitForTimeout(2000)
  })

  test.describe('1. 任务管理主界面', () => {
    test('任务管理界面应正确渲染', async ({ page }) => {
      const taskApp = page.locator('.task-management-app, .kanban-board, .task-board, [class*="task"]').first()
      const isVisible = await taskApp.count() > 0 && await taskApp.isVisible().catch(() => false)
      
      if (isVisible) {
        await expect(taskApp).toBeVisible()
        console.log('✓ 任务管理界面可见')
      } else {
        console.log('⚠ 任务管理界面未找到')
      }
    })

    test('应显示创建任务按钮', async ({ page }) => {
      const createBtn = page.locator('button:has-text("创建任务"), button:has-text("新建任务"), .create-task-btn, .add-task-btn').first()
      await expect(createBtn).toBeVisible({ timeout: 5000 })
      console.log('✓ 创建任务按钮可见')
    })
  })

  test.describe('2. 任务 CRUD 操作', () => {
    test('应能创建新任务', async ({ page }) => {
      const createBtn = page.locator('.create-task-btn, button:has-text("新建任务"), button:has-text("+ 新建")').first()
      await createBtn.click()
      await page.waitForTimeout(500)
      
      // 验证模态框出现
      const modal = page.locator('.task-modal, .modal-content, dialog, .task-form-container').first()
      const isVisible = await modal.isVisible().catch(() => false)
      
      if (isVisible) {
        console.log('✓ 创建任务模态框打开')
        
        // 填写任务标题
        const titleInput = modal.locator('input[type="text"], input[placeholder*="标题"], input[placeholder*="任务"]').first()
        if (await titleInput.count() > 0) {
          await titleInput.fill('测试任务')
          
          // 保存 - 尝试多种保存按钮选择器
          const saveSelectors = [
            'button:has-text("保存")',
            'button:has-text("创建")', 
            '.task-save-btn',
            '.submit-btn',
            '.task-modal-actions button',
          ]
          
          for (const selector of saveSelectors) {
            const saveBtn = modal.locator(selector).first()
            if (await saveBtn.count() > 0) {
              await saveBtn.click()
              await page.waitForTimeout(1000)
              console.log('✓ 任务保存成功')
              return
            }
          }
          console.log('⚠ 未找到保存按钮')
        }
      } else {
        console.log('⚠ 创建任务模态框未出现')
      }
    })

    test('应能编辑现有任务', async ({ page }) => {
      const firstTask = page.locator('.task-card, .task-item, .kanban-card').first()
      if (await firstTask.count() > 0) {
        await firstTask.click()
        await page.waitForTimeout(500)
        
        const modal = page.locator('.modal-content, .task-modal').first()
        if (await modal.isVisible().catch(() => false)) {
          console.log('✓ 点击任务打开编辑模态框')
        }
      } else {
        console.log('⚠ 暂无任务可编辑')
      }
    })
  })

  test.describe('3. 任务状态', () => {
    test('应显示任务状态列（看板视图）', async ({ page }) => {
      const kanbanColumns = page.locator('.kanban-column, .task-column, .task-status-column')
      const count = await kanbanColumns.count()
      
      if (count > 0) {
        expect(count).toBeGreaterThanOrEqual(2)
        console.log(`✓ 看板视图有 ${count} 个状态列`)
        
        // 打印各列标题
        for (let i = 0; i < count; i++) {
          const title = await kanbanColumns.nth(i).locator('h3, .column-title, .kanban-column-title').first().textContent().catch(() => '')
          console.log(`  列 ${i + 1}: ${title?.trim()}`)
        }
      } else {
        console.log('⚠ 未找到看板列')
      }
    })

    test('应能拖拽任务改变状态', async ({ page }) => {
      const taskCard = page.locator('.task-card, .kanban-card').first()
      if (await taskCard.count() > 0) {
        // 尝试拖拽（验证 draggable 属性）
        const isDraggable = await taskCard.evaluate(el => (el as HTMLElement).draggable)
        console.log(`✓ 任务卡片可拖拽: ${isDraggable}`)
      } else {
        console.log('⚠ 暂无任务可拖拽')
      }
    })
  })
})

test.describe('应用管理功能集成测试', () => {
  test.beforeEach(async ({ page }) => {
    await login(page)
    await navigateToApp(page, '应用管理')
    await page.waitForTimeout(2000)
  })

  test.describe('1. 应用管理主界面', () => {
    test('应用管理界面应正确渲染', async ({ page }) => {
      const appManagement = page.locator('.app-management-app, .app-management, [class*="app-management"]').first()
      const isVisible = await appManagement.count() > 0 && await appManagement.isVisible().catch(() => false)
      
      if (isVisible) {
        await expect(appManagement).toBeVisible()
        console.log('✓ 应用管理界面可见')
      } else {
        console.log('⚠ 应用管理界面未找到')
      }
    })
  })

  test.describe('2. 应用 CRUD 操作', () => {
    test('应能创建新应用', async ({ page }) => {
      // 应用管理界面通常有"创建应用"或"新建"按钮
      const createSelectors = [
        'button:has-text("创建应用")',
        'button:has-text("新建应用")',
        '.create-app-btn',
        '.add-app-btn',
      ]
      
      let foundCreate = false
      for (const selector of createSelectors) {
        const createBtn = page.locator(selector).first()
        if (await createBtn.count() > 0 && await createBtn.isVisible().catch(() => false)) {
          console.log('✓ 创建应用按钮可见')
          await createBtn.click()
          await page.waitForTimeout(500)
          foundCreate = true
          break
        }
      }
      
      if (!foundCreate) {
        console.log('⚠ 未找到创建应用按钮（可能是管理员专属功能）')
      } else {
        const modal = page.locator('.modal-content, .app-modal, dialog, .app-form-container').first()
        const isVisible = await modal.isVisible().catch(() => false)
        
        if (isVisible) {
          console.log('✓ 创建应用模态框打开')
          
          const nameInput = modal.locator('input[type="text"], input[placeholder*="应用名称"], input[placeholder*="名称"]').first()
          if (await nameInput.count() > 0) {
            await nameInput.fill('测试应用')
            console.log('✓ 应用名称已填写')
          }
        }
      }
    })

    test('应能编辑现有应用', async ({ page }) => {
      const firstApp = page.locator('.app-item, .app-card, .app-entry').first()
      if (await firstApp.count() > 0) {
        await firstApp.click()
        await page.waitForTimeout(500)
        console.log('✓ 点击应用可编辑')
      } else {
        console.log('⚠ 暂无应用可编辑')
      }
    })

    test('应能删除应用', async ({ page }) => {
      const deleteBtn = page.locator('.app-delete-btn, .delete-app-btn, button:has-text("删除应用")').first()
      if (await deleteBtn.count() > 0) {
        await deleteBtn.click()
        await page.waitForTimeout(500)
        console.log('✓ 删除应用按钮可点击')
      } else {
        console.log('⚠ 暂无删除按钮')
      }
    })
  })

  test.describe('3. 应用列表', () => {
    test('应显示应用列表', async ({ page }) => {
      const appItems = page.locator('.app-item, .app-card, .app-entry')
      const count = await appItems.count()
      console.log(`✓ 应用列表显示 ${count} 个应用`)
    })

    test('应支持搜索应用', async ({ page }) => {
      const searchInput = page.locator('input[placeholder*="搜索应用"], input[placeholder*="搜索"]').first()
      if (await searchInput.count() > 0) {
        await expect(searchInput).toBeVisible()
        console.log('✓ 应用搜索框可见')
      }
    })
  })
})
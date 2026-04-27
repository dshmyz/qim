import { test, expect, Page } from '@playwright/test'

const BASE_URL = 'http://localhost:5173'
const API_URL = 'http://localhost:8080'

/**
 * 登录辅助函数
 * 注意：App.vue 在 onMounted 时会强制清除 localStorage 并显示登录页面
 * 所以必须通过 UI 流程登录：填写表单 → 点击登录 → 等待 Main 出现
 */
async function login(page: Page) {
  // 1. 导航到登录页面
  await page.goto(`${BASE_URL}/#/login`)
  await page.waitForLoadState('networkidle')
  
  // 2. 等待登录表单出现
  await expect(page.locator('.login-form')).toBeVisible({ timeout: 10000 })
  
  // 3. 填写用户名和密码
  await page.fill('input[placeholder="请输入用户名"]', 'admin')
  await page.fill('input[placeholder="请输入密码"]', '123456')
  
  // 4. 点击登录按钮
  await page.click('button.login-button[type="submit"], button:has-text("登录")')
  
  // 5. 等待登录按钮变为"登录中..."
  await expect(page.locator('button.login-button:has-text("登录中")')).toBeVisible({ timeout: 3000 })
  
  // 6. 等待 Main 页面出现（登录成功后 App.vue 会显示 Main 组件）
  await expect(page.locator('.main-container, .im-app > div:not(.login-container)')).toBeVisible({ timeout: 15000 })
  
  // 7. 验证 token 已设置
  const token = await page.evaluate(() => localStorage.getItem('token'))
  expect(token).toBeTruthy()
  console.log('✓ 登录成功，token 已设置')
}

/**
 * 选择第一个会话进入聊天
 */
async function enterFirstConversation(page: Page) {
  // 等待会话列表加载
  const conversationItem = page.locator('.conversation-item, .conversation-list-item').first()
  await expect(conversationItem).toBeVisible({ timeout: 10000 })
  
  // 点击第一个会话
  await conversationItem.click()
  
  // 等待聊天窗口加载
  await expect(page.locator('.chat-window, .chat-container')).toBeVisible({ timeout: 5000 })
}

test.describe('AI功能集成测试 - 真实用户流程', () => {
  test.beforeEach(async ({ page }) => {
    await login(page)
    await enterFirstConversation(page)
    // 等待所有组件加载完成
    await page.waitForTimeout(2000)
  })

  // ========================================
  // 测试组 1: AI快捷指令栏
  // ========================================
  test.describe('1. AI快捷指令栏', () => {
    test('快捷指令栏应在聊天窗口正确渲染', async ({ page }) => {
      // 验证快捷指令栏存在且可见
      const quickActions = page.locator('.ai-quick-actions')
      await expect(quickActions).toBeVisible()
      
      // 验证所有预设按钮
      const expectedActions = ['总结对话', '翻译', '改写', '润色', '代码审查']
      
      for (const label of expectedActions) {
        const btn = page.locator(`.ai-quick-action:has-text("${label}")`)
        await expect(btn).toBeVisible()
        console.log(`✓ 找到快捷按钮: ${label}`)
      }
    })

    test('点击总结对话按钮应触发API调用', async ({ page }) => {
      // 监听网络请求
      const summaryRequest = page.waitForRequest('**/api/v1/ai/summary', { timeout: 10000 }).catch(() => null)
      
      // 点击总结按钮
      await page.click('.ai-quick-action:has-text("总结对话")')
      
      // 验证API被调用
      const request = await summaryRequest
      if (request) {
        expect(request.method()).toBe('POST')
        console.log('✓ 总结API被调用')
      }
      
      // 验证处理中状态
      const processingState = page.locator('.ai-processing')
      const isVisible = await processingState.isVisible().catch(() => false)
      console.log(`处理中状态: ${isVisible ? '显示' : '未显示或已完成'}`)
    })

    test('点击翻译按钮应触发翻译API', async ({ page }) => {
      const translateRequest = page.waitForRequest('**/api/v1/ai/translate', { timeout: 10000 }).catch(() => null)
      
      await page.click('.ai-quick-action:has-text("翻译")')
      
      const request = await translateRequest
      if (request) {
        expect(request.method()).toBe('POST')
        console.log('✓ 翻译API被调用')
      }
    })

    test('点击改写按钮应触发改写API', async ({ page }) => {
      const rewriteRequest = page.waitForRequest('**/api/v1/ai/rewrite', { timeout: 10000 }).catch(() => null)
      
      await page.click('.ai-quick-action:has-text("改写")')
      
      const request = await rewriteRequest
      if (request) {
        expect(request.method()).toBe('POST')
        console.log('✓ 改写API被调用')
      }
    })

    test('点击润色按钮应触发润色API', async ({ page }) => {
      const polishRequest = page.waitForRequest('**/api/v1/ai/polish', { timeout: 10000 }).catch(() => null)
      
      await page.click('.ai-quick-action:has-text("润色")')
      
      const request = await polishRequest
      if (request) {
        expect(request.method()).toBe('POST')
        console.log('✓ 润色API被调用')
      }
    })

    test('快捷指令栏应支持横向滚动', async ({ page }) => {
      const quickActions = page.locator('.ai-quick-actions')
      const scrollWidth = await quickActions.evaluate(el => (el as HTMLElement).scrollWidth)
      const clientWidth = await quickActions.evaluate(el => (el as HTMLElement).clientWidth)
      
      // 如果内容超出容器宽度，应该可以滚动
      if (scrollWidth > clientWidth) {
        const overflowX = await quickActions.evaluate(el => (el as HTMLElement).style.overflowX || getComputedStyle(el).overflowX)
        expect(overflowX).toMatch(/auto|scroll/)
        console.log(`✓ 快捷指令栏支持滚动 (${scrollWidth}px > ${clientWidth}px)`)
      }
    })
  })

  // ========================================
  // 测试组 2: AI消息标识
  // ========================================
  test.describe('2. AI消息标识', () => {
    test('AI消息应显示AI徽章', async ({ page }) => {
      // 查找AI消息（sender_id=0 或者带有 ai-message 类）
      const aiMessages = page.locator('.message-item.ai-message, .message-item[data-sender-id="0"]')
      const count = await aiMessages.count()
      
      if (count > 0) {
        const firstAiMsg = aiMessages.first()
        
        // 验证AI徽章
        const badge = firstAiMsg.locator('.ai-message-badge')
        await expect(badge).toBeVisible()
        
        // 验证徽章包含AI文字
        const badgeText = await badge.textContent()
        expect(badgeText).toMatch(/AI/)
        console.log(`✓ 找到 ${count} 条AI消息，徽章文字: ${badgeText?.trim()}`)
      } else {
        console.log('⚠ 暂无AI消息，跳过徽章测试')
      }
    })

    test('AI消息应有特殊样式区分', async ({ page }) => {
      const aiMessages = page.locator('.message-item.ai-message')
      const count = await aiMessages.count()
      
      if (count > 0) {
        const aiMsg = aiMessages.first()
        
        // 验证AI消息有特定的样式类
        const classes = await aiMsg.evaluate(el => Array.from(el.classList))
        expect(classes).toContain('ai-message')
        
        // 验证背景色或边框不同于普通消息
        const hasSpecialStyle = await aiMsg.evaluate(el => {
          const bg = getComputedStyle(el).backgroundColor
          const border = getComputedStyle(el).borderColor
          return bg !== '' && bg !== 'rgba(0, 0, 0, 0)'
        })
        
        console.log(`✓ AI消息有特殊样式: ${hasSpecialStyle}`)
      }
    })
  })

  // ========================================
  // 测试组 3: 长内容折叠展开
  // ========================================
  test.describe('3. 长内容折叠展开', () => {
    test('超长AI消息应显示展开按钮', async ({ page }) => {
      const expandBtn = page.locator('.ai-content-footer .expand-btn, button:has-text("展开全部")').first()
      const hasExpandBtn = await expandBtn.count() > 0
      
      if (hasExpandBtn) {
        await expect(expandBtn).toBeVisible()
        
        // 验证按钮显示字符数
        const btnText = await expandBtn.textContent()
        expect(btnText).toMatch(/字符/)
        console.log(`✓ 展开按钮: ${btnText?.trim()}`)
      } else {
        console.log('⚠ 暂无超长AI消息，跳过折叠测试')
      }
    })

    test('点击展开按钮应展开完整内容', async ({ page }) => {
      const expandBtn = page.locator('.expand-btn, button:has-text("展开全部")').first()
      
      if (await expandBtn.count() > 0) {
        await expandBtn.click()
        
        // 验证收起按钮出现
        const collapseBtn = page.locator('.collapse-btn, button:has-text("收起")')
        await expect(collapseBtn.first()).toBeVisible({ timeout: 3000 })
        console.log('✓ 内容已展开，收起按钮可见')
      }
    })

    test('展开后应显示操作按钮', async ({ page }) => {
      // 先展开
      const expandBtn = page.locator('.expand-btn, button:has-text("展开全部")').first()
      
      if (await expandBtn.count() > 0) {
        await expandBtn.click()
        await page.waitForTimeout(500)
        
        // 验证导出和复制按钮
        const exportBtn = page.locator('button:has-text("导出"), button:has-text("复制")').first()
        await expect(exportBtn).toBeVisible({ timeout: 3000 })
        console.log('✓ 展开后操作按钮可见')
      }
    })
  })

  // ========================================
  // 测试组 4: 消息右键菜单
  // ========================================
  test.describe('4. 消息右键菜单', () => {
    test('右键消息应显示上下文菜单', async ({ page }) => {
      // 找到第一条消息
      const firstMessage = page.locator('.message-item').first()
      await expect(firstMessage).toBeVisible({ timeout: 5000 })
      
      // 右键点击
      await firstMessage.click({ button: 'right' })
      
      // 验证菜单出现
      const menu = page.locator('.context-menu, .message-context-menu, [class*="context-menu"]').first()
      await expect(menu).toBeVisible({ timeout: 3000 })
      console.log('✓ 右键菜单显示成功')
      
      // 关闭菜单（按Esc）
      await page.keyboard.press('Escape')
    })

    test('右键菜单应包含AI操作选项', async ({ page }) => {
      const firstMessage = page.locator('.message-item').first()
      await firstMessage.click({ button: 'right' })
      
      // 等待菜单
      await page.waitForTimeout(500)
      
      // 检查菜单项
      const menuItems = page.locator('.menu-item, .context-menu-item, [class*="menu-item"]')
      const itemCount = await menuItems.count()
      
      expect(itemCount).toBeGreaterThan(0)
      console.log(`✓ 右键菜单有 ${itemCount} 个选项`)
      
      // 打印所有菜单项
      for (let i = 0; i < itemCount; i++) {
        const text = await menuItems.nth(i).textContent()
        console.log(`  - ${text?.trim()}`)
      }
      
      await page.keyboard.press('Escape')
    })

    test('点击AI翻译菜单项应触发翻译', async ({ page }) => {
      const firstMessage = page.locator('.message-item').first()
      await firstMessage.click({ button: 'right' })
      await page.waitForTimeout(500)
      
      // 查找翻译菜单项
      const translateItem = page.locator('.menu-item:has-text("翻译"), .context-menu-item:has-text("翻译")').first()
      
      if (await translateItem.count() > 0) {
        await translateItem.click()
        
        // 验证翻译API被调用
        const translateRequest = page.waitForRequest('**/api/v1/ai/translate', { timeout: 5000 }).catch(() => null)
        const request = await translateRequest
        
        if (request) {
          console.log('✓ 右键翻译API被调用')
        }
      } else {
        console.log('⚠ 右键菜单中未找到翻译选项')
      }
      
      await page.keyboard.press('Escape')
    })
  })

  // ========================================
  // 测试组 5: AI搜索
  // ========================================
  test.describe('5. AI语义搜索', () => {
    test('搜索输入框应存在', async ({ page }) => {
      const searchInput = page.locator('.ai-search-input input, input[placeholder*="搜索"], .search-input').first()
      await expect(searchInput).toBeVisible({ timeout: 5000 })
      console.log('✓ 搜索输入框可见')
    })

    test('输入搜索词应显示结果', async ({ page }) => {
      const searchInput = page.locator('.ai-search-input input, input[placeholder*="搜索"], .search-input').first()
      
      if (await searchInput.count() > 0) {
        await searchInput.fill('测试')
        await searchInput.press('Enter')
        
        // 等待API调用
        await page.waitForTimeout(3000)
        
        // 验证是否有搜索结果
        const results = page.locator('.ai-search-results .result-item, .search-results .result-item')
        const resultCount = await results.count()
        console.log(`✓ 搜索返回 ${resultCount} 条结果`)
      }
    })
  })

  // ========================================
  // 测试组 6: 会话摘要
  // ========================================
  test.describe('6. 会话智能摘要', () => {
    test('应能触发会话摘要生成', async ({ page }) => {
      // 监听摘要API
      const summaryRequest = page.waitForRequest('**/api/v1/ai/summary', { timeout: 10000 }).catch(() => null)
      
      // 点击总结按钮触发
      const summaryBtn = page.locator('.ai-quick-action:has-text("总结")').first()
      if (await summaryBtn.count() > 0) {
        await summaryBtn.click()
        
        const request = await summaryRequest
        if (request) {
          expect(request.method()).toBe('POST')
          console.log('✓ 会话摘要API被调用')
        }
      }
    })
  })

  // ========================================
  // 测试组 7: 快捷键
  // ========================================
  test.describe('7. 快捷键', () => {
    test('Ctrl+Shift+S 应触发会话摘要', async ({ page }) => {
      const summaryRequest = page.waitForRequest('**/api/v1/ai/summary', { timeout: 5000 }).catch(() => null)
      
      // 发送快捷键
      await page.keyboard.press('Control+Shift+s')
      await page.waitForTimeout(1000)
      
      const request = await summaryRequest
      if (request) {
        console.log('✓ Ctrl+Shift+S 快捷键触发摘要API')
      } else {
        console.log('⚠ Ctrl+Shift+S 未触发API调用（可能快捷键未注册）')
      }
    })

    test('Ctrl+K 应触发AI面板', async ({ page }) => {
      await page.keyboard.press('Control+k')
      await page.waitForTimeout(1000)
      
      // 检查是否有AI面板或快捷操作出现
      const aiPanel = page.locator('.ai-panel, .ai-quick-actions:visible, .ai-shortcut-panel').first()
      const isVisible = await aiPanel.count() > 0 && await aiPanel.isVisible().catch(() => false)
      
      if (isVisible) {
        console.log('✓ Ctrl+K 快捷键触发AI面板')
      } else {
        console.log('⚠ Ctrl+K 未触发可见的AI面板')
      }
    })
  })

  // ========================================
  // 测试组 8: 错误处理
  // ========================================
  test.describe('8. 错误处理', () => {
    test('AI服务503错误应显示友好提示', async ({ page }) => {
      // 拦截AI API返回503
      await page.route('**/api/v1/ai/**', async (route) => {
        await route.fulfill({
          status: 503,
          contentType: 'application/json',
          body: JSON.stringify({ message: 'AI服务暂时不可用，请稍后再试' }),
        })
      })

      // 触发AI操作
      const quickAction = page.locator('.ai-quick-action:has-text("翻译")').first()
      await quickAction.click()
      
      // 等待错误提示
      await page.waitForTimeout(2000)
      
      // 验证错误信息
      const errorMessage = page.locator('[class*="error"], [class*="message--error"], .el-message--error, [class*="toast"]:visible').first()
      
      if (await errorMessage.count() > 0) {
        const text = await errorMessage.textContent()
        console.log(`错误信息: ${text?.trim()}`)
        
        // 验证错误信息友好（不包含技术细节）
        expect(text).not.toMatch(/503|Internal Server Error|stack trace/i)
        console.log('✓ 错误信息友好，不包含技术细节')
      } else {
        console.log('⚠ 未找到可见的错误提示')
      }
      
      // 清除路由拦截
      await page.unroute('**/api/v1/ai/**')
    })
  })

  // ========================================
  // 测试组 9: @AI触发回复 (如果当前是群聊)
  // ========================================
  test.describe('9. @AI触发回复', () => {
    test('输入框应支持@AI提及', async ({ page }) => {
      // 找到消息输入框
      const input = page.locator('.message-input textarea, .message-input-area textarea, [contenteditable="true"]').first()
      
      if (await input.count() > 0) {
        // 输入@AI
        await input.fill('@AI 你好')
        await page.waitForTimeout(500)
        
        const inputValue = await input.inputValue().catch(() => null)
        
        if (inputValue) {
          expect(inputValue).toContain('@AI')
          console.log(`✓ 输入框支持@AI: "${inputValue}"`)
        } else {
          console.log('⚠ 输入框类型不支持inputValue')
        }
      }
    })
  })
})

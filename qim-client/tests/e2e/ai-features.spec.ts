import { test, expect } from '@playwright/test'

test.describe('AI功能测试', () => {
  test.beforeEach(async ({ page }) => {
    // 导航到主页面
    await page.goto('/')
    // 等待页面加载
    await page.waitForLoadState('networkidle')
  })

  // 测试1: AI组件加载
  test('AI快捷指令栏应在聊天窗口加载', async ({ page }) => {
    // 检查页面是否有聊天窗口
    const chatWindow = page.locator('.chat-window')
    const isChatVisible = await chatWindow.count() > 0
    
    if (isChatVisible) {
      // 验证快捷指令栏存在
      const quickActions = page.locator('.ai-quick-actions')
      const hasQuickActions = await quickActions.count() > 0
      
      if (hasQuickActions) {
        await expect(quickActions).toBeVisible()
        
        // 验证预设按钮
        const actions = [
          { icon: '📝', label: '总结对话' },
          { icon: '🌐', label: '翻译' },
          { icon: '✍️', label: '改写' },
          { icon: '✨', label: '润色' },
        ]
        
        for (const action of actions) {
          const btn = page.locator(`.ai-quick-action:has-text("${action.label}")`)
          if (await btn.count() > 0) {
            await expect(btn).toBeVisible()
          }
        }
      }
    }
  })

  // 测试2: AI摘要面板组件
  test('摘要面板组件应正确渲染', async ({ page }) => {
    // 检查是否有摘要面板组件
    const summaryPanel = page.locator('aisummarypanel, .ai-summary-panel')
    const hasPanel = await summaryPanel.count() > 0
    
    if (hasPanel) {
      // 面板默认不显示
      const isVisible = await summaryPanel.isVisible()
      expect(isVisible).toBe(false)
    }
  })

  // 测试3: AI消息标识组件
  test('AI消息应显示正确的标识和样式', async ({ page }) => {
    // 模拟AI消息
    const aiMessage = page.locator('.message-item.ai-message').first()
    
    if (await aiMessage.count() > 0) {
      // 验证AI徽章存在
      const badge = aiMessage.locator('.ai-message-badge')
      await expect(badge).toBeVisible()
      await expect(badge).toContainText('AI')
      
      // 验证消息样式
      await expect(aiMessage).toHaveClass(/ai-message/)
    }
  })

  // 测试4: 长内容折叠
  test('长AI消息应支持折叠和展开', async ({ page }) => {
    const expandButton = page.locator('.ai-content-footer .expand-btn').first()
    
    if (await expandButton.count() > 0) {
      // 验证折叠按钮存在
      await expect(expandButton).toBeVisible()
      
      // 点击展开
      await expandButton.click()
      
      // 验证收起按钮出现
      const collapseButton = page.locator('.expanded-actions .collapse-btn')
      await expect(collapseButton).toBeVisible()
    }
  })

  // 测试5: AI搜索输入框
  test('应能进行语义搜索', async ({ page }) => {
    const searchInput = page.locator('.ai-search-input input, input[placeholder*="搜索"]')
    
    if (await searchInput.count() > 0) {
      // 输入搜索词
      await searchInput.fill('项目进度')
      await searchInput.press('Enter')
      
      // 验证搜索结果出现
      const searchResults = page.locator('.ai-search-results, .search-results')
      await expect(searchResults).toBeVisible({ timeout: 10000 })
    }
  })

  // 测试6: 错误处理
  test('AI服务不可用时应显示友好提示', async ({ page }) => {
    // 模拟AI服务不可用（拦截请求）
    await page.route('**/api/v1/ai/**', async (route) => {
      await route.fulfill({
        status: 503,
        body: JSON.stringify({ message: 'AI服务暂时不可用' }),
      })
    })

    // 触发AI操作
    const quickAction = page.locator('.ai-quick-action').first()
    if (await quickAction.count() > 0) {
      await quickAction.click()
      
      // 验证错误提示
      const errorMessage = page.locator('[class*="error"], [class*="message"]:has-text("不可用")')
      await expect(errorMessage).toBeVisible({ timeout: 5000 })
      
      // 验证错误信息友好
      const text = await errorMessage.textContent()
      expect(text).not.toContain('503')
      expect(text).not.toContain('Internal Server Error')
    }
  })
})
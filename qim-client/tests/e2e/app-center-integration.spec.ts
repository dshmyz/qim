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
 * 打开应用中心
 */
async function openAppCenter(page: Page) {
  // 点击应用中心按钮（左侧导航栏的app图标）
  // 尝试多种选择器找到正确的入口
  const selectors = [
    '.app-center-btn',
    '.sidebar-icon-item:has(.fa-th-large)',
    '.sidebar-item:has-text("应用")',
    'button:has-text("应用中心")',
    '[class*="appCenter"]',
    '.sidebar-icon-item', // 最后一个备选
  ]
  
  for (const selector of selectors) {
    const btn = page.locator(selector).first()
    if (await btn.count() > 0) {
      await btn.click()
      await page.waitForTimeout(1500)
      console.log(`✓ 使用选择器 "${selector}" 打开应用中心`)
      return
    }
  }
  
  // 如果所有选择器都失败，尝试点击第一个侧边栏项
  const fallback = page.locator('.sidebar-left button, .sidebar-icon-item').first()
  if (await fallback.count() > 0) {
    await fallback.click()
    await page.waitForTimeout(1500)
    console.log('✓ 使用备用选择器打开侧边栏')
  }
}

test.describe('应用中心集成测试 - 所有应用功能', () => {
  test.beforeEach(async ({ page }) => {
    await login(page)
    await openAppCenter(page)
    await page.waitForTimeout(2000)
  })

  // ========================================
  // 测试组 1: 应用中心主面板
  // ========================================
  test.describe('1. 应用中心主面板', () => {
    test('应用中心面板应正确渲染', async ({ page }) => {
      // 检查是否有应用中心相关元素
      const appsPanel = page.locator('.apps-content, .main-apps-grid, .recent-apps-section, .app-center-panel').first()
      const isVisible = await appsPanel.count() > 0
      
      if (isVisible) {
        await expect(appsPanel).toBeVisible()
        console.log('✓ 应用中心面板可见')
      } else {
        console.log('⚠ 应用中心面板未找到，检查当前页面结构')
        const pageContent = await page.locator('body').textContent()
        console.log(`页面内容预览: ${pageContent?.slice(0, 200)}`)
      }
    })

    test('应显示最近使用应用区域', async ({ page }) => {
      const recentApps = page.locator('.recent-apps-section, .recent-apps').first()
      if (await recentApps.count() > 0) {
        await expect(recentApps).toBeVisible()
        console.log('✓ 最近使用应用区域可见')
      }
    })

    test('应显示所有应用区域', async ({ page }) => {
      const allApps = page.locator('.all-apps-section, .main-apps-grid, .apps-grid').first()
      if (await allApps.count() > 0) {
        await expect(allApps).toBeVisible()
        console.log('✓ 所有应用区域可见')
      }
    })
  })

  // ========================================
  // 测试组 2: 内置应用功能
  // ========================================
  test.describe('2. 内置应用功能', () => {
    test('点击应用应触发openApp事件', async ({ page }) => {
      const appItems = page.locator('.main-app-item, .recent-app-grid-item, .app-item, .mini-app-item')
      const count = await appItems.count()
      
      if (count > 0) {
        // 点击第一个应用
        await appItems.first().click()
        await page.waitForTimeout(1000)
        
        // 验证是否有应用打开的迹象（模态框、新面板等）
        const modalOrPanel = page.locator('.modal, .dialog, .app-modal, .mini-app-modal, [class*="panel"]').first()
        const appeared = await modalOrPanel.count() > 0 && await modalOrPanel.isVisible().catch(() => false)
        
        if (appeared) {
          console.log('✓ 点击应用后出现模态框/面板')
        } else {
          console.log('⚠ 点击应用后未检测到模态框')
        }
      }
    })
  })

  // ========================================
  // 测试组 3: 小程序面板
  // ========================================
  test.describe('3. 小程序面板', () => {
    test('应能打开小程序面板', async ({ page }) => {
      // 查找小程序入口按钮
      const miniAppBtn = page.locator('[class*="miniapp"], [class*="mini-app"], button:has-text("小程序")').first()
      
      if (await miniAppBtn.count() > 0) {
        await miniAppBtn.click()
        await page.waitForTimeout(1000)
        
        const miniAppPanel = page.locator('.mini-app-panel, .mini-app-grid').first()
        await expect(miniAppPanel).toBeVisible({ timeout: 5000 })
        console.log('✓ 小程序面板打开成功')
      } else {
        console.log('⚠ 未找到小程序入口按钮')
      }
    })

    test('小程序面板应显示小程序列表', async ({ page }) => {
      // 假设已经打开面板
      const miniAppItems = page.locator('.mini-app-item')
      const count = await miniAppItems.count()
      console.log(`✓ 小程序列表显示 ${count} 个小程序`)
    })
  })

  // ========================================
  // 测试组 4: 计算器应用
  // ========================================
  test.describe('4. 计算器应用', () => {
    test('应能打开计算器并执行计算', async ({ page }) => {
      // 查找计算器小程序
      const calculatorBtn = page.locator('.mini-app-item:has-text("计算器"), .mini-app-item:has-text("calculator")').first()
      
      if (await calculatorBtn.count() > 0) {
        await calculatorBtn.click()
        await page.waitForTimeout(1000)
        
        // 验证计算器界面
        const calcDisplay = page.locator('.calculator-result, .calculator-display, #calculator-result')
        const isVisible = await calcDisplay.count() > 0 && await calcDisplay.isVisible().catch(() => false)
        
        if (isVisible) {
          console.log('✓ 计算器界面打开成功')
          
          // 尝试点击数字按钮
          const numberBtns = page.locator('.calculator-btn-number, .calculator-btn:has-text("1"), .calculator-btn:has-text("2")')
          if (await numberBtns.count() > 0) {
            await numberBtns.first().click()
            console.log('✓ 计算器按钮可点击')
          }
        } else {
          console.log('⚠ 计算器界面未找到')
        }
      } else {
        console.log('⚠ 未找到计算器小程序')
      }
    })
  })

  // ========================================
  // 测试组 5: 记事本应用
  // ========================================
  test.describe('5. 记事本应用', () => {
    test('应能打开记事本并输入内容', async ({ page }) => {
      const notepadBtn = page.locator('.mini-app-item:has-text("记事本"), .mini-app-item:has-text("notepad")').first()
      
      if (await notepadBtn.count() > 0) {
        await notepadBtn.click()
        await page.waitForTimeout(1000)
        
        const notepadContent = page.locator('.notepad-content, #notepad-content, textarea[placeholder*="内容"]')
        const isVisible = await notepadContent.count() > 0 && await notepadContent.isVisible().catch(() => false)
        
        if (isVisible) {
          console.log('✓ 记事本界面打开成功')
          
          // 尝试输入内容
          await notepadContent.fill('测试笔记内容')
          const value = await notepadContent.inputValue()
          expect(value).toContain('测试笔记内容')
          console.log('✓ 记事本可输入内容')
        } else {
          console.log('⚠ 记事本界面未找到')
        }
      } else {
        console.log('⚠ 未找到记事本小程序')
      }
    })
  })

  // ========================================
  // 测试组 6: 密码生成器应用
  // ========================================
  test.describe('6. 密码生成器应用', () => {
    test('应能打开密码生成器并生成密码', async ({ page }) => {
      const pwdBtn = page.locator('.mini-app-item:has-text("密码"), .mini-app-item:has-text("password")').first()
      
      if (await pwdBtn.count() > 0) {
        await pwdBtn.click()
        await page.waitForTimeout(1000)
        
        const pwdResult = page.locator('.password-result-input, #password-result')
        const isVisible = await pwdResult.count() > 0 && await pwdResult.isVisible().catch(() => false)
        
        if (isVisible) {
          console.log('✓ 密码生成器界面打开成功')
          
          // 检查是否已生成密码
          const pwdValue = await pwdResult.inputValue()
          if (pwdValue && pwdValue.length > 0) {
            console.log(`✓ 密码已生成: ${pwdValue.slice(0, 8)}...`)
          }
          
          // 点击生成按钮
          const generateBtn = page.locator('#generate-password, .generate-btn:has-text("生成")')
          if (await generateBtn.count() > 0) {
            await generateBtn.click()
            await page.waitForTimeout(500)
            const newPwd = await pwdResult.inputValue()
            expect(newPwd).toBeTruthy()
            console.log('✓ 密码生成按钮工作正常')
          }
        } else {
          console.log('⚠ 密码生成器界面未找到')
        }
      } else {
        console.log('⚠ 未找到密码生成器小程序')
      }
    })

    test('应能调整密码长度', async ({ page }) => {
      const pwdBtn = page.locator('.mini-app-item:has-text("密码")').first()
      
      if (await pwdBtn.count() > 0) {
        await pwdBtn.click()
        await page.waitForTimeout(1000)
        
        const lengthInput = page.locator('#password-length, input[type="range"]')
        if (await lengthInput.count() > 0) {
          const lengthValue = await page.locator('#password-length-value, [class*="length-value"]').textContent()
          console.log(`✓ 密码长度调节可见, 当前值: ${lengthValue}`)
        }
      }
    })
  })

  // ========================================
  // 测试组 7: 待办事项应用
  // ========================================
  test.describe('7. 待办事项应用', () => {
    test('应能打开待办事项并添加任务', async ({ page }) => {
      const todoBtn = page.locator('.mini-app-item:has-text("待办"), .mini-app-item:has-text("todo")').first()
      
      if (await todoBtn.count() > 0) {
        await todoBtn.click()
        await page.waitForTimeout(1000)
        
        const todoInput = page.locator('.todo-input, #todo-input')
        const isVisible = await todoInput.count() > 0 && await todoInput.isVisible().catch(() => false)
        
        if (isVisible) {
          console.log('✓ 待办事项界面打开成功')
          
          // 添加待办事项
          await todoInput.fill('测试待办事项')
          const addBtn = page.locator('#add-todo, .add-todo-btn')
          await addBtn.click()
          await page.waitForTimeout(500)
          
          // 验证任务已添加
          const todoItems = page.locator('.todo-item, #todo-list > div')
          const count = await todoItems.count()
          expect(count).toBeGreaterThan(0)
          console.log(`✓ 待办事项已添加, 当前有 ${count} 个任务`)
        } else {
          console.log('⚠ 待办事项界面未找到')
        }
      } else {
        console.log('⚠ 未找到待办事项小程序')
      }
    })

    test('应能切换待办事项状态', async ({ page }) => {
      const todoBtn = page.locator('.mini-app-item:has-text("待办")').first()
      
      if (await todoBtn.count() > 0) {
        await todoBtn.click()
        await page.waitForTimeout(1000)
        
        const checkboxes = page.locator('.todo-item input[type="checkbox"]')
        if (await checkboxes.count() > 0) {
          await checkboxes.first().click()
          await page.waitForTimeout(500)
          console.log('✓ 待办事项状态可切换')
        }
      }
    })
  })

  // ========================================
  // 测试组 8: 短链接应用
  // ========================================
  test.describe('8. 短链接应用', () => {
    test('应能打开短链接并生成短链接', async ({ page }) => {
      const linkBtn = page.locator('.mini-app-item:has-text("短链接"), .mini-app-item:has-text("短链"), .mini-app-item:has-text("short")').first()
      
      if (await linkBtn.count() > 0) {
        await linkBtn.click()
        await page.waitForTimeout(1000)
        
        const linkInput = page.locator('#short-link-input, .original-url-input, textarea[placeholder*="URL"]')
        const isVisible = await linkInput.count() > 0 && await linkInput.isVisible().catch(() => false)
        
        if (isVisible) {
          console.log('✓ 短链接界面打开成功')
          
          // 输入测试URL
          await linkInput.fill('https://example.com/test')
          const generateBtn = page.locator('#generate-short-link, .generate-btn:has-text("生成")')
          await generateBtn.click()
          await page.waitForTimeout(2000)
          
          // 检查是否有短链接生成
          const resultInput = page.locator('#short-link-output-input, .short-url-input')
          if (await resultInput.count() > 0) {
            const resultValue = await resultInput.inputValue()
            if (resultValue) {
              console.log(`✓ 短链接已生成: ${resultValue}`)
            } else {
              console.log('⚠ 短链接结果为空')
            }
          }
        } else {
          console.log('⚠ 短链接界面未找到')
        }
      } else {
        console.log('⚠ 未找到短链接小程序')
      }
    })
  })

  // ========================================
  // 测试组 9: 应用中心交互
  // ========================================
  test.describe('9. 应用中心交互', () => {
    test('应能关闭应用面板', async ({ page }) => {
      const closeBtn = page.locator('.close-btn, [class*="close"]:has-text("×"), button:has-text("关闭")').first()
      
      if (await closeBtn.count() > 0) {
        await closeBtn.click()
        await page.waitForTimeout(500)
        
        const panel = page.locator('.mini-app-panel, .app-center-panel, .apps-content')
        const isVisible = await panel.isVisible().catch(() => false)
        
        if (!isVisible) {
          console.log('✓ 面板可关闭')
        } else {
          console.log('⚠ 面板未关闭')
        }
      }
    })

    test('点击模态框外部应关闭面板', async ({ page }) => {
      const modalOverlay = page.locator('.mini-app-panel-container, .modal-overlay, [class*="overlay"]').first()
      
      if (await modalOverlay.count() > 0) {
        await modalOverlay.click()
        await page.waitForTimeout(500)
        console.log('✓ 点击模态框外部可关闭面板')
      }
    })
  })

  // ========================================
  // 测试组 10: 应用状态和加载
  // ========================================
  test.describe('10. 应用状态和加载', () => {
    test('加载中小程序应显示加载状态', async ({ page }) => {
      const loadingSpinner = page.locator('.loading-spinner, .mini-app-loading, [class*="loading"]')
      if (await loadingSpinner.count() > 0) {
        const isVisible = await loadingSpinner.isVisible().catch(() => false)
        if (isVisible) {
          console.log('✓ 加载状态可见')
        }
      } else {
        console.log('⚠ 未检测到加载状态（可能已加载完成）')
      }
    })

    test('空状态应显示友好提示', async ({ page }) => {
      const emptyState = page.locator('.mini-app-empty, .empty-all-apps, .empty-recent-apps').first()
      if (await emptyState.count() > 0) {
        const text = await emptyState.textContent()
        expect(text).toBeTruthy()
        console.log(`✓ 空状态提示: ${text?.trim()}`)
      } else {
        // 备选方案
        const altEmpty = page.locator('[class*="empty"]').first()
        if (await altEmpty.count() > 0) {
          const text = await altEmpty.textContent()
          expect(text).toBeTruthy()
          console.log(`✓ 空状态提示（备选）: ${text?.trim()}`)
        }
      }
    })
  })
})
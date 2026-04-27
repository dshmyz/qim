# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: main-features.spec.ts >> QIM 前端全功能点自动化测试 >> 聊天窗口功能测试 >> 应该能显示聊天 header
- Location: tests/e2e/main-features.spec.ts:325:5

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('.chat-header')
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 5000ms
  - waiting for locator('.chat-header')

```

# Page snapshot

```yaml
- generic [ref=e4]:
  - generic [ref=e8]:
    - button "—" [ref=e9] [cursor=pointer]
    - button "☐" [ref=e10] [cursor=pointer]
    - button "×" [ref=e11] [cursor=pointer]
  - generic [ref=e12]:
    - generic [ref=e13]: QIM
    - generic "最近联系人" [ref=e14] [cursor=pointer]
    - generic "组织架构" [ref=e17] [cursor=pointer]
    - generic "群聊" [ref=e20] [cursor=pointer]
    - generic "应用" [ref=e23] [cursor=pointer]
    - generic "更多" [ref=e26] [cursor=pointer]
    - generic "皮肤" [ref=e29] [cursor=pointer]
    - generic "设置" [ref=e32] [cursor=pointer]
  - generic [ref=e36]:
    - generic [ref=e37]:
      - generic [ref=e38]:
        - generic [ref=e39] [cursor=pointer]:
          - img "admin" [ref=e40]
          - generic [ref=e41]: admin
        - generic [ref=e42]:
          - button "通知" [ref=e43] [cursor=pointer]
          - button [ref=e45] [cursor=pointer]
      - textbox "搜索用户或群组..." [ref=e48]
    - generic [ref=e52]:
      - generic [ref=e53]:
        - heading "最近会话" [level=2] [ref=e54]
        - button [ref=e55] [cursor=pointer]
      - paragraph [ref=e61]: 选择一个会话开始聊天
```

# Test source

```ts
  227 |       // 验证组织架构树容器可见
  228 |       await expect(page.locator('.tree-container')).toBeVisible()
  229 |     })
  230 | 
  231 |     test('应该能切换群组面板', async ({ page }) => {
  232 |       const groupIcon = page.locator('.side-options .option-item').nth(2)
  233 |       await groupIcon.click()
  234 |       await page.waitForTimeout(500)
  235 |       // 验证群组列表容器可见
  236 |       await expect(page.locator('.groups-list')).toBeVisible()
  237 |     })
  238 | 
  239 |     test('应该能切换应用中心面板', async ({ page }) => {
  240 |       const appIcon = page.locator('.side-options .option-item').nth(3)
  241 |       await appIcon.click()
  242 |       await page.waitForTimeout(500)
  243 |       // 验证应用面板容器可见
  244 |       await expect(page.locator('.apps-container')).toBeVisible()
  245 |     })
  246 | 
  247 |     test('应该能显示用户信息区域', async ({ page }) => {
  248 |       await expect(page.locator('.user-info')).toBeVisible()
  249 |     })
  250 | 
  251 |     test('应该能点击通知按钮', async ({ page }) => {
  252 |       const notificationBtn = page.locator('button[title="通知"]')
  253 |       await notificationBtn.click()
  254 |       // 验证通知面板或菜单出现
  255 |       await page.waitForTimeout(500)
  256 |     })
  257 |   })
  258 | 
  259 |   test.describe('会话列表功能测试', () => {
  260 |     test('应该能点击会话项切换聊天', async ({ page }) => {
  261 |       const firstConversation = page.locator('.conversation-item').first()
  262 |       if (await firstConversation.isVisible()) {
  263 |         await firstConversation.click()
  264 |         await expect(page.locator('.chat-window')).toBeVisible()
  265 |       }
  266 |     })
  267 | 
  268 |     test('应该能右键打开会话上下文菜单', async ({ page }) => {
  269 |       const firstConversation = page.locator('.conversation-item').first()
  270 |       if (await firstConversation.isVisible()) {
  271 |         await firstConversation.click({ button: 'right' })
  272 |         await expect(page.locator('.context-menu')).toBeVisible()
  273 |       }
  274 |     })
  275 | 
  276 |     test('应该能通过上下文菜单置顶会话', async ({ page }) => {
  277 |       const firstConversation = page.locator('.conversation-item').first()
  278 |       if (await firstConversation.isVisible()) {
  279 |         await firstConversation.click({ button: 'right' })
  280 |         const pinAction = page.locator('.context-menu-item:has-text("置顶")')
  281 |         if (await pinAction.isVisible()) {
  282 |           await pinAction.click()
  283 |         }
  284 |       }
  285 |     })
  286 | 
  287 |     test('应该能通过上下文菜单免打扰会话', async ({ page }) => {
  288 |       const firstConversation = page.locator('.conversation-item').first()
  289 |       if (await firstConversation.isVisible()) {
  290 |         await firstConversation.click({ button: 'right' })
  291 |         const muteAction = page.locator('.context-menu-item:has-text("免打扰")')
  292 |         if (await muteAction.isVisible()) {
  293 |           await muteAction.click()
  294 |         }
  295 |       }
  296 |     })
  297 | 
  298 |     test('应该能通过上下文菜单移除会话', async ({ page }) => {
  299 |       const firstConversation = page.locator('.conversation-item').first()
  300 |       if (await firstConversation.isVisible()) {
  301 |         await firstConversation.click({ button: 'right' })
  302 |         const removeAction = page.locator('.context-menu-item:has-text("移除")')
  303 |         if (await removeAction.isVisible()) {
  304 |           await removeAction.click()
  305 |           // 可能弹出确认对话框
  306 |           const confirmBtn = page.locator('.q-btn--primary, button:has-text("确定")')
  307 |           if (await confirmBtn.isVisible({ timeout: 2000 })) {
  308 |             await confirmBtn.click()
  309 |           }
  310 |         }
  311 |       }
  312 |     })
  313 |   })
  314 | 
  315 |   test.describe('聊天窗口功能测试', () => {
  316 |     test.beforeEach(async ({ page }) => {
  317 |       // 选择一个会话
  318 |       const firstConversation = page.locator('.conversation-item').first()
  319 |       if (await firstConversation.isVisible({ timeout: 5000 })) {
  320 |         await firstConversation.click()
  321 |         await page.waitForTimeout(1000)
  322 |       }
  323 |     })
  324 | 
  325 |     test('应该能显示聊天 header', async ({ page }) => {
  326 |       const chatHeader = page.locator('.chat-header')
> 327 |       await expect(chatHeader).toBeVisible()
      |                                ^ Error: expect(locator).toBeVisible() failed
  328 |     })
  329 |   })
  330 | 
  331 |   test.describe('搜索功能测试', () => {
  332 |     test('应该能使用搜索按钮打开搜索', async ({ page }) => {
  333 |       const searchBtn = page.locator('.search-btn')
  334 |       if (await searchBtn.isVisible()) {
  335 |         await searchBtn.click()
  336 |         await page.waitForTimeout(500)
  337 |       }
  338 |     })
  339 | 
  340 |     test('应该能执行会话内搜索', async ({ page }) => {
  341 |       const firstConversation = page.locator('.conversation-item').first()
  342 |       if (await firstConversation.isVisible()) {
  343 |         await firstConversation.click()
  344 |         await page.waitForTimeout(1000)
  345 |       }
  346 |       
  347 |       const searchBtn = page.locator('.message-search-btn, .search-btn')
  348 |       if (await searchBtn.isVisible()) {
  349 |         await searchBtn.click()
  350 |         await page.waitForTimeout(500)
  351 |       }
  352 |     })
  353 |   })
  354 | 
  355 |   test.describe('创建功能测试', () => {
  356 |     test('应该能打开创建群组对话框', async ({ page }) => {
  357 |       const actionBtn = page.locator('.action-menu-btn, .more-actions-btn')
  358 |       if (await actionBtn.isVisible()) {
  359 |         await actionBtn.click()
  360 |         const createGroupAction = page.locator('.action-menu-item:has-text("创建群组")')
  361 |         if (await createGroupAction.isVisible()) {
  362 |           await createGroupAction.click()
  363 |           await expect(page.locator('.create-group-modal, .q-dialog')).toBeVisible()
  364 |         }
  365 |       }
  366 |     })
  367 | 
  368 |     test('应该能打开创建讨论组对话框', async ({ page }) => {
  369 |       const actionBtn = page.locator('.action-menu-btn, .more-actions-btn')
  370 |       if (await actionBtn.isVisible()) {
  371 |         await actionBtn.click()
  372 |         const createDiscussionAction = page.locator('.action-menu-item:has-text("创建讨论组")')
  373 |         if (await createDiscussionAction.isVisible()) {
  374 |           await createDiscussionAction.click()
  375 |         }
  376 |       }
  377 |     })
  378 |   })
  379 | 
  380 |   test.describe('用户操作菜单测试', () => {
  381 |     test('应该能打开用户操作菜单', async ({ page }) => {
  382 |       const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
  383 |       if (await userMenuBtn.isVisible()) {
  384 |         await userMenuBtn.click()
  385 |         await expect(page.locator('.user-context-menu')).toBeVisible()
  386 |       }
  387 |     })
  388 | 
  389 |     test('应该能通过菜单打开设置', async ({ page }) => {
  390 |       const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
  391 |       if (await userMenuBtn.isVisible()) {
  392 |         await userMenuBtn.click()
  393 |         const settingsAction = page.locator('.context-menu-item:has-text("设置")')
  394 |         if (await settingsAction.isVisible()) {
  395 |           await settingsAction.click()
  396 |           await expect(page.locator('.settings-modal, .settings-panel')).toBeVisible()
  397 |         }
  398 |       }
  399 |     })
  400 | 
  401 |     test('应该能通过菜单执行登出', async ({ page }) => {
  402 |       const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
  403 |       if (await userMenuBtn.isVisible()) {
  404 |         await userMenuBtn.click()
  405 |         const logoutAction = page.locator('.context-menu-item:has-text("退出登录")')
  406 |         if (await logoutAction.isVisible()) {
  407 |           await logoutAction.click()
  408 |           // 可能会有确认对话框
  409 |           await page.waitForTimeout(500)
  410 |         }
  411 |       }
  412 |     })
  413 | 
  414 |     test('应该能通过菜单查看关于信息', async ({ page }) => {
  415 |       const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
  416 |       if (await userMenuBtn.isVisible()) {
  417 |         await userMenuBtn.click()
  418 |         const aboutAction = page.locator('.context-menu-item:has-text("关于")')
  419 |         if (await aboutAction.isVisible()) {
  420 |           await aboutAction.click()
  421 |         }
  422 |       }
  423 |     })
  424 |   })
  425 | 
  426 |   test.describe('群组操作测试', () => {
  427 |     test('应该能查看群组成员列表', async ({ page }) => {
```
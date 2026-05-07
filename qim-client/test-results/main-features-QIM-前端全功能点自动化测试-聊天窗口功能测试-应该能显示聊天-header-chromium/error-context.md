# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: main-features.spec.ts >> QIM 前端全功能点自动化测试 >> 聊天窗口功能测试 >> 应该能显示聊天 header
- Location: tests/e2e/main-features.spec.ts:324:5

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
  - generic [ref=e8]: 加载中...
  - generic [ref=e12]:
    - button "—" [ref=e13] [cursor=pointer]
    - button "☐" [ref=e14] [cursor=pointer]
    - button "×" [ref=e15] [cursor=pointer]
  - generic [ref=e16]:
    - generic [ref=e17]: QIM
    - generic "最近联系人" [ref=e18] [cursor=pointer]
    - generic "组织架构" [ref=e21] [cursor=pointer]
    - generic "群聊" [ref=e24] [cursor=pointer]
    - generic "应用" [ref=e27] [cursor=pointer]
    - generic "频道" [ref=e30] [cursor=pointer]
    - generic "皮肤" [ref=e33] [cursor=pointer]
    - generic "设置" [ref=e36] [cursor=pointer]
  - generic [ref=e40]:
    - generic [ref=e41]:
      - generic [ref=e42]:
        - generic [ref=e43] [cursor=pointer]:
          - img "admin" [ref=e44]
          - generic [ref=e45]: admin
        - generic [ref=e46]:
          - button "通知" [ref=e47] [cursor=pointer]
          - button [ref=e49] [cursor=pointer]
      - textbox "搜索用户或群组..." [ref=e52]
    - generic [ref=e56]:
      - generic [ref=e58]:
        - button [ref=e59] [cursor=pointer]
        - heading "最近会话" [level=2] [ref=e61]
      - paragraph [ref=e66]: 选择一个会话开始聊天
```

# Test source

```ts
  226 |       // 验证组织架构树容器可见
  227 |       await expect(page.locator('.tree-container')).toBeVisible()
  228 |     })
  229 | 
  230 |     test('应该能切换群组面板', async ({ page }) => {
  231 |       const groupIcon = page.locator('.side-options .option-item').nth(2)
  232 |       await groupIcon.click()
  233 |       await page.waitForTimeout(500)
  234 |       // 验证群组列表容器可见
  235 |       await expect(page.locator('.groups-list')).toBeVisible()
  236 |     })
  237 | 
  238 |     test('应该能切换应用中心面板', async ({ page }) => {
  239 |       const appIcon = page.locator('.side-options .option-item').nth(3)
  240 |       await appIcon.click()
  241 |       await page.waitForTimeout(500)
  242 |       // 验证应用面板容器可见
  243 |       await expect(page.locator('.apps-container')).toBeVisible()
  244 |     })
  245 | 
  246 |     test('应该能显示用户信息区域', async ({ page }) => {
  247 |       await expect(page.locator('.user-info')).toBeVisible()
  248 |     })
  249 | 
  250 |     test('应该能点击通知按钮', async ({ page }) => {
  251 |       const notificationBtn = page.locator('button[title="通知"]')
  252 |       await notificationBtn.click()
  253 |       // 验证通知面板或菜单出现
  254 |       await page.waitForTimeout(500)
  255 |     })
  256 |   })
  257 | 
  258 |   test.describe('会话列表功能测试', () => {
  259 |     test('应该能点击会话项切换聊天', async ({ page }) => {
  260 |       const firstConversation = page.locator('.conversation-item').first()
  261 |       if (await firstConversation.isVisible()) {
  262 |         await firstConversation.click()
  263 |         await expect(page.locator('.chat-window')).toBeVisible()
  264 |       }
  265 |     })
  266 | 
  267 |     test('应该能右键打开会话上下文菜单', async ({ page }) => {
  268 |       const firstConversation = page.locator('.conversation-item').first()
  269 |       if (await firstConversation.isVisible()) {
  270 |         await firstConversation.click({ button: 'right' })
  271 |         await expect(page.locator('.context-menu')).toBeVisible()
  272 |       }
  273 |     })
  274 | 
  275 |     test('应该能通过上下文菜单置顶会话', async ({ page }) => {
  276 |       const firstConversation = page.locator('.conversation-item').first()
  277 |       if (await firstConversation.isVisible()) {
  278 |         await firstConversation.click({ button: 'right' })
  279 |         const pinAction = page.locator('.context-menu-item:has-text("置顶")')
  280 |         if (await pinAction.isVisible()) {
  281 |           await pinAction.click()
  282 |         }
  283 |       }
  284 |     })
  285 | 
  286 |     test('应该能通过上下文菜单免打扰会话', async ({ page }) => {
  287 |       const firstConversation = page.locator('.conversation-item').first()
  288 |       if (await firstConversation.isVisible()) {
  289 |         await firstConversation.click({ button: 'right' })
  290 |         const muteAction = page.locator('.context-menu-item:has-text("免打扰")')
  291 |         if (await muteAction.isVisible()) {
  292 |           await muteAction.click()
  293 |         }
  294 |       }
  295 |     })
  296 | 
  297 |     test('应该能通过上下文菜单移除会话', async ({ page }) => {
  298 |       const firstConversation = page.locator('.conversation-item').first()
  299 |       if (await firstConversation.isVisible()) {
  300 |         await firstConversation.click({ button: 'right' })
  301 |         const removeAction = page.locator('.context-menu-item:has-text("移除")')
  302 |         if (await removeAction.isVisible()) {
  303 |           await removeAction.click()
  304 |           // 可能弹出确认对话框
  305 |           const confirmBtn = page.locator('.q-btn--primary, button:has-text("确定")')
  306 |           if (await confirmBtn.isVisible({ timeout: 2000 })) {
  307 |             await confirmBtn.click()
  308 |           }
  309 |         }
  310 |       }
  311 |     })
  312 |   })
  313 | 
  314 |   test.describe('聊天窗口功能测试', () => {
  315 |     test.beforeEach(async ({ page }) => {
  316 |       // 选择一个会话
  317 |       const firstConversation = page.locator('.conversation-item').first()
  318 |       if (await firstConversation.isVisible({ timeout: 5000 })) {
  319 |         await firstConversation.click()
  320 |         await page.waitForTimeout(1000)
  321 |       }
  322 |     })
  323 | 
  324 |     test('应该能显示聊天 header', async ({ page }) => {
  325 |       const chatHeader = page.locator('.chat-header')
> 326 |       await expect(chatHeader).toBeVisible()
      |                                ^ Error: expect(locator).toBeVisible() failed
  327 |     })
  328 |   })
  329 | 
  330 |   test.describe('搜索功能测试', () => {
  331 |     test('应该能使用搜索按钮打开搜索', async ({ page }) => {
  332 |       const searchBtn = page.locator('.search-btn')
  333 |       if (await searchBtn.isVisible()) {
  334 |         await searchBtn.click()
  335 |         await page.waitForTimeout(500)
  336 |       }
  337 |     })
  338 | 
  339 |     test('应该能执行会话内搜索', async ({ page }) => {
  340 |       const firstConversation = page.locator('.conversation-item').first()
  341 |       if (await firstConversation.isVisible()) {
  342 |         await firstConversation.click()
  343 |         await page.waitForTimeout(1000)
  344 |       }
  345 |       
  346 |       const searchBtn = page.locator('.message-search-btn, .search-btn')
  347 |       if (await searchBtn.isVisible()) {
  348 |         await searchBtn.click()
  349 |         await page.waitForTimeout(500)
  350 |       }
  351 |     })
  352 |   })
  353 | 
  354 |   test.describe('创建功能测试', () => {
  355 |     test('应该能打开创建群组对话框', async ({ page }) => {
  356 |       const actionBtn = page.locator('.action-menu-btn, .more-actions-btn')
  357 |       if (await actionBtn.isVisible()) {
  358 |         await actionBtn.click()
  359 |         const createGroupAction = page.locator('.action-menu-item:has-text("创建群组")')
  360 |         if (await createGroupAction.isVisible()) {
  361 |           await createGroupAction.click()
  362 |           await expect(page.locator('.create-group-modal, .q-dialog')).toBeVisible()
  363 |         }
  364 |       }
  365 |     })
  366 | 
  367 |     test('应该能打开创建讨论组对话框', async ({ page }) => {
  368 |       const actionBtn = page.locator('.action-menu-btn, .more-actions-btn')
  369 |       if (await actionBtn.isVisible()) {
  370 |         await actionBtn.click()
  371 |         const createDiscussionAction = page.locator('.action-menu-item:has-text("创建讨论组")')
  372 |         if (await createDiscussionAction.isVisible()) {
  373 |           await createDiscussionAction.click()
  374 |         }
  375 |       }
  376 |     })
  377 |   })
  378 | 
  379 |   test.describe('用户操作菜单测试', () => {
  380 |     test('应该能打开用户操作菜单', async ({ page }) => {
  381 |       const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
  382 |       if (await userMenuBtn.isVisible()) {
  383 |         await userMenuBtn.click()
  384 |         await expect(page.locator('.user-context-menu')).toBeVisible()
  385 |       }
  386 |     })
  387 | 
  388 |     test('应该能通过菜单打开设置', async ({ page }) => {
  389 |       const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
  390 |       if (await userMenuBtn.isVisible()) {
  391 |         await userMenuBtn.click()
  392 |         const settingsAction = page.locator('.context-menu-item:has-text("设置")')
  393 |         if (await settingsAction.isVisible()) {
  394 |           await settingsAction.click()
  395 |           await expect(page.locator('.settings-modal, .settings-panel')).toBeVisible()
  396 |         }
  397 |       }
  398 |     })
  399 | 
  400 |     test('应该能通过菜单执行登出', async ({ page }) => {
  401 |       const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
  402 |       if (await userMenuBtn.isVisible()) {
  403 |         await userMenuBtn.click()
  404 |         const logoutAction = page.locator('.context-menu-item:has-text("退出登录")')
  405 |         if (await logoutAction.isVisible()) {
  406 |           await logoutAction.click()
  407 |           // 可能会有确认对话框
  408 |           await page.waitForTimeout(500)
  409 |         }
  410 |       }
  411 |     })
  412 | 
  413 |     test('应该能通过菜单查看关于信息', async ({ page }) => {
  414 |       const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
  415 |       if (await userMenuBtn.isVisible()) {
  416 |         await userMenuBtn.click()
  417 |         const aboutAction = page.locator('.context-menu-item:has-text("关于")')
  418 |         if (await aboutAction.isVisible()) {
  419 |           await aboutAction.click()
  420 |         }
  421 |       }
  422 |     })
  423 |   })
  424 | 
  425 |   test.describe('群组操作测试', () => {
  426 |     test('应该能查看群组成员列表', async ({ page }) => {
```
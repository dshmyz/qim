# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: main-features.spec.ts >> QIM 前端全功能点自动化测试 >> 侧边栏功能测试 >> 应该能切换组织架构面板
- Location: tests/e2e/main-features.spec.ts:223:5

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator:  locator('.tree-container')
Expected: visible
Received: hidden
Timeout:  5000ms

Call log:
  - Expect "toBeVisible" with timeout 5000ms
  - waiting for locator('.tree-container')
    9 × locator resolved to <div data-v-b6e27d1c="" data-v-7ade5772="" class="tree-container"></div>
      - unexpected value "hidden"

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
      - textbox "搜索..." [ref=e48]
    - generic [ref=e51]:
      - heading "组织架构" [level=2] [ref=e53]
      - paragraph [ref=e55]: 选择左侧的组织架构查看详情
```

# Test source

```ts
  128 | 
  129 |   // Mock 用户资料 API
  130 |   await page.route('**/api/v1/users/**', async (route) => {
  131 |     await route.fulfill({
  132 |       status: 200,
  133 |       contentType: 'application/json',
  134 |       body: JSON.stringify({
  135 |         code: 0,
  136 |         data: { id: 1, username: 'admin', name: '管理员' }
  137 |       })
  138 |     })
  139 |   })
  140 | 
  141 |   // Mock 员工列表 API
  142 |   await page.route('**/api/v1/employees/**', async (route) => {
  143 |     await route.fulfill({
  144 |       status: 200,
  145 |       contentType: 'application/json',
  146 |       body: JSON.stringify({
  147 |         code: 0,
  148 |         data: [
  149 |           { id: 1, username: 'admin', name: '管理员', department: '技术部' },
  150 |           { id: 2, username: 'user1', name: '测试用户', department: '产品部' }
  151 |         ]
  152 |       })
  153 |     })
  154 |   })
  155 | 
  156 |   // Mock 通知 API
  157 |   await page.route('**/api/v1/notifications/**', async (route) => {
  158 |     await route.fulfill({
  159 |       status: 200,
  160 |       contentType: 'application/json',
  161 |       body: JSON.stringify({ code: 0, data: [] })
  162 |     })
  163 |   })
  164 | 
  165 |   // Mock 搜索 API
  166 |   await page.route('**/api/v1/search/**', async (route) => {
  167 |     await route.fulfill({
  168 |       status: 200,
  169 |       contentType: 'application/json',
  170 |       body: JSON.stringify({ code: 0, data: [] })
  171 |     })
  172 |   })
  173 | 
  174 |   // Mock 版本检查 API
  175 |   await page.route('**/api/v1/version/**', async (route) => {
  176 |     await route.fulfill({
  177 |       status: 200,
  178 |       contentType: 'application/json',
  179 |       body: JSON.stringify({ code: 0, data: { needUpdate: false } })
  180 |     })
  181 |   })
  182 | 
  183 |   // Mock 头像获取
  184 |   await page.route('**/api/v1/avatars/**', async (route) => {
  185 |     await route.fulfill({ status: 404 })
  186 |   })
  187 | 
  188 |   await page.goto('http://localhost:3000')
  189 |   await page.fill('input[placeholder="请输入用户名"]', 'admin')
  190 |   await page.fill('input[placeholder="请输入密码"]', '123456')
  191 |   await page.click('button:has-text("登录")')
  192 |   
  193 |   // 等待 Main 组件渲染完成（通过检查 im-container 是否存在）
  194 |   await page.waitForSelector('.im-container', { timeout: 10000 })
  195 |   await page.waitForTimeout(2000)
  196 | }
  197 | 
  198 | test.describe('QIM 前端全功能点自动化测试', () => {
  199 |   
  200 |   test.beforeEach(async ({ page }) => {
  201 |     await login(page)
  202 |     // 自动关闭网络错误提示遮罩层（因为没有真实 WebSocket 连接）
  203 |     await page.evaluate(() => {
  204 |       const el = document.querySelector('.network-error')
  205 |       if (el) el.remove()
  206 |       // 同时设置 showNetworkError 为 false 防止重新渲染
  207 |       const style = document.createElement('style')
  208 |       style.textContent = '.network-error { display: none !important; }'
  209 |       document.head.appendChild(style)
  210 |     })
  211 |     await page.waitForTimeout(500)
  212 |   })
  213 | 
  214 |   test.describe('侧边栏功能测试', () => {
  215 |     test('应该能切换会话列表面板', async ({ page }) => {
  216 |       const conversationIcon = page.locator('.side-options .option-item').first()
  217 |       await conversationIcon.click()
  218 |       await page.waitForTimeout(500)
  219 |       // 会话列表是默认显示的，验证侧边栏切换成功
  220 |       await expect(page.locator('.sidebar')).toBeVisible()
  221 |     })
  222 | 
  223 |     test('应该能切换组织架构面板', async ({ page }) => {
  224 |       const orgIcon = page.locator('.side-options .option-item').nth(1)
  225 |       await orgIcon.click()
  226 |       await page.waitForTimeout(500)
  227 |       // 验证组织架构树容器可见
> 228 |       await expect(page.locator('.tree-container')).toBeVisible()
      |                                                     ^ Error: expect(locator).toBeVisible() failed
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
  327 |       await expect(chatHeader).toBeVisible()
  328 |     })
```
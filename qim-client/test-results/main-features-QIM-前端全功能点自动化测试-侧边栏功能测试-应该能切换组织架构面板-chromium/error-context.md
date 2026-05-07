# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: main-features.spec.ts >> QIM 前端全功能点自动化测试 >> 侧边栏功能测试 >> 应该能切换组织架构面板
- Location: tests/e2e/main-features.spec.ts:222:5

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
    - generic "频道" [ref=e26] [cursor=pointer]
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
  127 | 
  128 |   // Mock 用户资料 API
  129 |   await page.route('**/api/v1/users/**', async (route) => {
  130 |     await route.fulfill({
  131 |       status: 200,
  132 |       contentType: 'application/json',
  133 |       body: JSON.stringify({
  134 |         code: 0,
  135 |         data: { id: 1, username: 'admin', name: '管理员' }
  136 |       })
  137 |     })
  138 |   })
  139 | 
  140 |   // Mock 员工列表 API
  141 |   await page.route('**/api/v1/employees/**', async (route) => {
  142 |     await route.fulfill({
  143 |       status: 200,
  144 |       contentType: 'application/json',
  145 |       body: JSON.stringify({
  146 |         code: 0,
  147 |         data: [
  148 |           { id: 1, username: 'admin', name: '管理员', department: '技术部' },
  149 |           { id: 2, username: 'user1', name: '测试用户', department: '产品部' }
  150 |         ]
  151 |       })
  152 |     })
  153 |   })
  154 | 
  155 |   // Mock 通知 API
  156 |   await page.route('**/api/v1/notifications/**', async (route) => {
  157 |     await route.fulfill({
  158 |       status: 200,
  159 |       contentType: 'application/json',
  160 |       body: JSON.stringify({ code: 0, data: [] })
  161 |     })
  162 |   })
  163 | 
  164 |   // Mock 搜索 API
  165 |   await page.route('**/api/v1/search/**', async (route) => {
  166 |     await route.fulfill({
  167 |       status: 200,
  168 |       contentType: 'application/json',
  169 |       body: JSON.stringify({ code: 0, data: [] })
  170 |     })
  171 |   })
  172 | 
  173 |   // Mock 版本检查 API
  174 |   await page.route('**/api/v1/version/**', async (route) => {
  175 |     await route.fulfill({
  176 |       status: 200,
  177 |       contentType: 'application/json',
  178 |       body: JSON.stringify({ code: 0, data: { needUpdate: false } })
  179 |     })
  180 |   })
  181 | 
  182 |   // Mock 头像获取
  183 |   await page.route('**/api/v1/avatars/**', async (route) => {
  184 |     await route.fulfill({ status: 404 })
  185 |   })
  186 | 
  187 |   await page.goto('http://localhost:3000')
  188 |   await page.fill('input[placeholder="请输入用户名"]', 'admin')
  189 |   await page.fill('input[placeholder="请输入密码"]', '123456')
  190 |   await page.click('button:has-text("登录")')
  191 |   
  192 |   // 等待 Main 组件渲染完成（通过检查 im-container 是否存在）
  193 |   await page.waitForSelector('.im-container', { timeout: 10000 })
  194 |   await page.waitForTimeout(2000)
  195 | }
  196 | 
  197 | test.describe('QIM 前端全功能点自动化测试', () => {
  198 |   
  199 |   test.beforeEach(async ({ page }) => {
  200 |     await login(page)
  201 |     // 自动关闭网络错误提示遮罩层（因为没有真实 WebSocket 连接）
  202 |     await page.evaluate(() => {
  203 |       const el = document.querySelector('.network-error')
  204 |       if (el) el.remove()
  205 |       // 同时设置 showNetworkError 为 false 防止重新渲染
  206 |       const style = document.createElement('style')
  207 |       style.textContent = '.network-error { display: none !important; }'
  208 |       document.head.appendChild(style)
  209 |     })
  210 |     await page.waitForTimeout(500)
  211 |   })
  212 | 
  213 |   test.describe('侧边栏功能测试', () => {
  214 |     test('应该能切换会话列表面板', async ({ page }) => {
  215 |       const conversationIcon = page.locator('.side-options .option-item').first()
  216 |       await conversationIcon.click()
  217 |       await page.waitForTimeout(500)
  218 |       // 会话列表是默认显示的，验证侧边栏切换成功
  219 |       await expect(page.locator('.sidebar')).toBeVisible()
  220 |     })
  221 | 
  222 |     test('应该能切换组织架构面板', async ({ page }) => {
  223 |       const orgIcon = page.locator('.side-options .option-item').nth(1)
  224 |       await orgIcon.click()
  225 |       await page.waitForTimeout(500)
  226 |       // 验证组织架构树容器可见
> 227 |       await expect(page.locator('.tree-container')).toBeVisible()
      |                                                     ^ Error: expect(locator).toBeVisible() failed
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
  326 |       await expect(chatHeader).toBeVisible()
  327 |     })
```
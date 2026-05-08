# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: main-features.spec.ts >> QIM 前端全功能点自动化测试 >> 侧边栏折叠功能测试 >> 应该能展开折叠后的侧边栏
- Location: tests/e2e/main-features.spec.ts:623:5

# Error details

```
Error: locator.isVisible: Error: strict mode violation: locator('.toggle-sidebar-btn') resolved to 2 elements:
    1) <button data-v-9562428a="" class="toggle-sidebar-btn">…</button> aka getByRole('button').nth(5)
    2) <button data-v-a4922fc3="" class="toggle-sidebar-btn">…</button> aka locator('.header-left > .toggle-sidebar-btn')

Call log:
    - checking visibility of locator('.toggle-sidebar-btn')

```

# Page snapshot

```yaml
- generic [active] [ref=e1]:
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
        - textbox "搜索用户或群组..." [ref=e48]
      - generic [ref=e52]:
        - generic [ref=e54]:
          - button [ref=e55] [cursor=pointer]
          - heading "最近会话" [level=2] [ref=e57]
        - paragraph [ref=e62]: 选择一个会话开始聊天
  - generic [ref=e63]:
    - img [ref=e65]
    - generic [ref=e68]: 加载便签失败，请稍后重试
    - button [ref=e69] [cursor=pointer]:
      - img [ref=e70]
```

# Test source

```ts
  525 |       const panelVisible = await emojiPanel.isVisible().catch(() => false)
  526 |       if (panelVisible) {
  527 |         const tabs = page.locator('.emoji-tab, .emoji-category-tab')
  528 |         if (await tabs.count() > 1) {
  529 |           await tabs.nth(1).click()
  530 |           await page.waitForTimeout(500)
  531 |         }
  532 |       }
  533 |     })
  534 | 
  535 |     test('应该能选择并插入表情', async ({ page }) => {
  536 |       // 先选择会话
  537 |       const firstConversation = page.locator('.conversation-item').first()
  538 |       if (await firstConversation.isVisible({ timeout: 5000 })) {
  539 |         await firstConversation.click()
  540 |         await page.waitForTimeout(1000)
  541 |       }
  542 |       
  543 |       // 查找表情按钮
  544 |       const emojiBtn = page.locator('.emoji-btn, button[title="表情"], .emoji-panel-btn')
  545 |       const isVisible = await emojiBtn.isVisible().catch(() => false)
  546 |       if (!isVisible) return
  547 |       
  548 |       await emojiBtn.click()
  549 |       await page.waitForTimeout(1000)
  550 |       
  551 |       // 验证表情面板可见
  552 |       const emojiPanel = page.locator('.emoji-panel-container')
  553 |       const panelVisible = await emojiPanel.isVisible().catch(() => false)
  554 |       if (!panelVisible) return
  555 | 
  556 |       const firstEmoji = page.locator('.emoji-item').first()
  557 |       if (await firstEmoji.isVisible()) {
  558 |         await firstEmoji.click()
  559 |         await page.waitForTimeout(500)
  560 |       }
  561 |     })
  562 |   })
  563 | 
  564 |   test.describe('@提及功能测试', () => {
  565 |     test('应该能触发@提及面板', async ({ page }) => {
  566 |       const input = page.locator('.message-input textarea')
  567 |       if (await input.isVisible()) {
  568 |         await input.fill('@')
  569 |         await page.waitForTimeout(500)
  570 |         
  571 |         // 验证@成员面板出现
  572 |         await expect(page.locator('.at-members-panel, .at-mention-panel')).toBeVisible()
  573 |       }
  574 |     })
  575 | 
  576 |     test('应该能选择@成员', async ({ page }) => {
  577 |       const input = page.locator('.message-input textarea')
  578 |       if (await input.isVisible()) {
  579 |         await input.fill('@')
  580 |         await page.waitForTimeout(500)
  581 |         
  582 |         const firstMember = page.locator('.at-member-item').first()
  583 |         if (await firstMember.isVisible()) {
  584 |           await firstMember.click()
  585 |           await page.waitForTimeout(500)
  586 |           
  587 |           const inputValue = await input.inputValue()
  588 |           expect(inputValue).toContain('@')
  589 |         }
  590 |       }
  591 |     })
  592 |   })
  593 | 
  594 |   test.describe('窗口控制测试', () => {
  595 |     test('应该能显示窗口控制按钮', async ({ page }) => {
  596 |       const minimizeBtn = page.locator('.window-control-btn.minimize-btn')
  597 |       const maximizeBtn = page.locator('.window-control-btn.maximize-btn')
  598 |       const closeBtn = page.locator('.window-control-btn.close-btn')
  599 |       
  600 |       // 这些按钮可能在Electron环境下才显示
  601 |       const hasWindowControls = await minimizeBtn.isVisible().catch(() => false)
  602 |       if (hasWindowControls) {
  603 |         await expect(minimizeBtn).toBeVisible()
  604 |         await expect(maximizeBtn).toBeVisible()
  605 |         await expect(closeBtn).toBeVisible()
  606 |       }
  607 |     })
  608 |   })
  609 | 
  610 |   test.describe('侧边栏折叠功能测试', () => {
  611 |     test('应该能折叠侧边栏', async ({ page }) => {
  612 |       const toggleBtn = page.locator('.toggle-sidebar-btn')
  613 |       if (await toggleBtn.isVisible()) {
  614 |         await toggleBtn.click()
  615 |         await page.waitForTimeout(500)
  616 |         // 验证侧边栏折叠状态
  617 |         const sidebar = page.locator('.sidebar')
  618 |         const className = await sidebar.getAttribute('class')
  619 |         expect(className).toContain('sidebar-collapsed')
  620 |       }
  621 |     })
  622 | 
  623 |     test('应该能展开折叠后的侧边栏', async ({ page }) => {
  624 |       const toggleBtn = page.locator('.toggle-sidebar-btn')
> 625 |       if (await toggleBtn.isVisible()) {
      |                           ^ Error: locator.isVisible: Error: strict mode violation: locator('.toggle-sidebar-btn') resolved to 2 elements:
  626 |         await toggleBtn.click() // 先折叠
  627 |         await page.waitForTimeout(500)
  628 |         await toggleBtn.click() // 再展开
  629 |         await page.waitForTimeout(500)
  630 |       }
  631 |     })
  632 |   })
  633 | 
  634 |   test.describe('通知中心测试', () => {
  635 |     test('应该能打开通知中心', async ({ page }) => {
  636 |       const notificationBtn = page.locator('button[title="通知"]')
  637 |       await notificationBtn.click()
  638 |       await page.waitForTimeout(500)
  639 |     })
  640 | 
  641 |     test('应该能关闭通知中心', async ({ page }) => {
  642 |       const notificationBtn = page.locator('button[title="通知"]')
  643 |       await notificationBtn.click()
  644 |       await page.waitForTimeout(500)
  645 |       
  646 |       // 再次点击应该关闭
  647 |       await notificationBtn.click()
  648 |       await page.waitForTimeout(500)
  649 |     })
  650 |   })
  651 | 
  652 |   test.describe('分享功能测试', () => {
  653 |     test('应该能打开分享对话框', async ({ page }) => {
  654 |       const firstConversation = page.locator('.conversation-item').first()
  655 |       if (await firstConversation.isVisible()) {
  656 |         await firstConversation.click()
  657 |         await page.waitForTimeout(1000)
  658 |       }
  659 |     })
  660 |   })
  661 | 
  662 |   test.describe('连接状态测试', () => {
  663 |     test('应该能显示连接断开提示', async ({ page }) => {
  664 |       // 模拟网络断开
  665 |       await page.context().setOffline(true)
  666 |       await page.waitForTimeout(2000)
  667 |       
  668 |       // 应该显示重连提示
  669 |       const reconnectBtn = page.locator('.retry-btn, button:has-text("重新连接")')
  670 |       const isVisible = await reconnectBtn.isVisible().catch(() => false)
  671 |       
  672 |       if (isVisible) {
  673 |         await reconnectBtn.click()
  674 |         await page.waitForTimeout(1000)
  675 |       }
  676 |       
  677 |       await page.context().setOffline(false)
  678 |     })
  679 | 
  680 |     test('应该能通过登录按钮重新登录', async ({ page }) => {
  681 |       await page.context().setOffline(true)
  682 |       await page.waitForTimeout(2000)
  683 |       
  684 |       const loginBtn = page.locator('.login-btn, button:has-text("重新登录")')
  685 |       const isVisible = await loginBtn.isVisible().catch(() => false)
  686 |       
  687 |       if (isVisible) {
  688 |         await loginBtn.click()
  689 |         await page.waitForURL('**/login', { timeout: 5000 }).catch(() => {})
  690 |       }
  691 |       
  692 |       await page.context().setOffline(false)
  693 |     })
  694 |   })
  695 | 
  696 |   test.describe('设置面板测试', () => {
  697 |     test('应该能切换到基本设置', async ({ page }) => {
  698 |       const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
  699 |       if (await userMenuBtn.isVisible()) {
  700 |         await userMenuBtn.click()
  701 |         const settingsAction = page.locator('.context-menu-item:has-text("设置")')
  702 |         if (await settingsAction.isVisible()) {
  703 |           await settingsAction.click()
  704 |           await page.waitForTimeout(500)
  705 |         }
  706 |       }
  707 |       
  708 |       const basicTab = page.locator('.settings-sidebar-item:has-text("基本"), .settings-tab:has-text("基本")')
  709 |       if (await basicTab.isVisible()) {
  710 |         await basicTab.click()
  711 |         await page.waitForTimeout(500)
  712 |       }
  713 |     })
  714 | 
  715 |     test('应该能切换到消息设置', async ({ page }) => {
  716 |       // 打开设置
  717 |       const userMenuBtn = page.locator('.user-action-btn, .user-more-actions')
  718 |       if (await userMenuBtn.isVisible()) {
  719 |         await userMenuBtn.click()
  720 |         const settingsAction = page.locator('.context-menu-item:has-text("设置")')
  721 |         if (await settingsAction.isVisible()) {
  722 |           await settingsAction.click()
  723 |           await page.waitForTimeout(500)
  724 |         }
  725 |       }
```
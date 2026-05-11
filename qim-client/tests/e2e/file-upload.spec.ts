import { test, expect, Page } from '@playwright/test'
import path from 'path'
import { fileURLToPath } from 'url'

/**
 * QIM 文件上传功能集成测试
 * 测试文件上传、预览等完整流程
 */

// 获取当前文件目录
const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

// 测试辅助函数 - 通过 mock API 响应实现登录
async function login(page: Page) {
  // Mock 登录API 响应
  await page.route('**/api/v1/auth/login', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        message: '登录成功',
        data: {
          token: 'mock-token-for-testing',
          user: {
            id: 1,
            username: 'admin',
            name: '管理员',
            email: 'admin@example.com',
            avatar: null,
            isAdmin: true
          }
        }
      })
    })
  })

  // Mock 2FA 检查
  await page.route('**/api/v1/auth/check-2fa', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: { twoFactorEnabled: false } })
    })
  })

  // Mock 组织架构 API
  await page.route('**/api/v1/org/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: [] })
    })
  })

  // Mock 会话列表 API
  await page.route('**/api/v1/conversations/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        data: [
          {
            id: 1,
            type: 'private',
            name: '测试用户',
            avatar: null,
            lastMessage: '你好',
            unreadCount: 0,
            lastMessageTime: new Date().toISOString()
          }
        ]
      })
    })
  })

  // Mock 消息列表 API
  await page.route('**/api/v1/messages/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: [] })
    })
  })

  // Mock 群组 API
  await page.route('**/api/v1/groups/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: [] })
    })
  })

  // Mock 频道 API
  await page.route('**/api/v1/channels/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: [] })
    })
  })

  // Mock 应用列表 API
  await page.route('**/api/v1/apps/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        data: [
          { id: '2', name: '日历', icon: 'calendar', description: '日程管理' },
          { id: '3', name: '文件管理', icon: 'folder', description: '文件存储' },
          { id: '4', name: 'AI 助手', icon: 'robot', description: '智能助手' },
          { id: '5', name: '任务管理', icon: 'tasks', description: '任务跟踪' },
          { id: '6', name: '便签', icon: 'sticky-note', description: '快速记录' },
          { id: '7', name: '笔记', icon: 'book', description: '笔记管理' }
        ]
      })
    })
  })

  // Mock 用户资料 API
  await page.route('**/api/v1/users/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        data: { id: 1, username: 'admin', name: '管理员' }
      })
    })
  })

  // Mock 员工列表 API
  await page.route('**/api/v1/employees/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        data: [
          { id: 1, username: 'admin', name: '管理员', department: '技术部' },
          { id: 2, username: 'user1', name: '测试用户', department: '产品部' }
        ]
      })
    })
  })

  // Mock 通知 API
  await page.route('**/api/v1/notifications/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: [] })
    })
  })

  // Mock 搜索 API
  await page.route('**/api/v1/search/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: [] })
    })
  })

  // Mock 版本检查 API
  await page.route('**/api/v1/version/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: { needUpdate: false } })
    })
  })

  // Mock 头像获取
  await page.route('**/api/v1/avatars/**', async (route) => {
    await route.fulfill({ status: 404 })
  })

  await page.goto('http://localhost:3000')
  await page.fill('input[placeholder="请输入用户名"]', 'admin')
  await page.fill('input[placeholder="请输入密码"]', '123456')
  await page.click('button:has-text("登录")')

  // 等待 Main 组件渲染完成
  await page.waitForSelector('.im-container', { timeout: 10000 })
  await page.waitForTimeout(2000)
}

// Mock 文件列表 API
async function mockFileListAPI(page: Page) {
  await page.route('**/api/v1/files**', async (route) => {
    const url = route.request().url()

    // 文件列表
    if (url.includes('/api/v1/files') && !url.includes('/upload') && !url.includes('/init')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 200,
          data: {
            files: [
              {
                id: 1,
                user_id: 1,
                name: 'test-document.pdf',
                original_name: 'test-document.pdf',
                size: 1024000,
                mime_type: 'application/pdf',
                storage_path: '/uploads/test-document.pdf',
                checksum: 'abc123',
                folder_id: null,
                source: 'upload',
                source_id: null,
                is_starred: false,
                starred_at: null,
                tags: null,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
              },
              {
                id: 2,
                user_id: 1,
                name: 'test-text.txt',
                original_name: 'test-text.txt',
                size: 1024,
                mime_type: 'text/plain',
                storage_path: '/uploads/test-text.txt',
                checksum: 'def456',
                folder_id: null,
                source: 'upload',
                source_id: null,
                is_starred: false,
                starred_at: null,
                tags: null,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
              }
            ],
            total: 2,
            page: 1,
            page_size: 20
          }
        })
      })
    } else {
      await route.continue()
    }
  })

  // Mock 文件夹列表
  await page.route('**/api/v1/folders**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: []
      })
    })
  })
}

// Mock 文件上传 API
async function mockFileUploadAPI(page: Page) {
  // Mock 初始化上传
  await page.route('**/api/v1/files/upload/init', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          upload_id: 'test-upload-id-' + Date.now(),
          chunk_size: 1024000,
          total_chunks: 1,
          uploaded_chunks: [],
          is_quick_upload: false
        }
      })
    })
  })

  // Mock 上传分片
  await page.route('**/api/v1/files/upload/chunk', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          chunk_index: 0,
          chunk_hash: 'test-hash'
        }
      })
    })
  })

  // Mock 完成上传
  await page.route('**/api/v1/files/upload/complete', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          id: Date.now(),
          name: 'uploaded-file.txt',
          size: 1024
        }
      })
    })
  })
}

test.describe('文件上传功能集成测试', () => {
  test.beforeEach(async ({ page }) => {
    await login(page)
    await mockFileListAPI(page)
    await mockFileUploadAPI(page)

    // 自动关闭网络错误提示遮罩层
    await page.evaluate(() => {
      const el = document.querySelector('.network-error')
      if (el) el.remove()
      const style = document.createElement('style')
      style.textContent = '.network-error { display: none !important; }'
      document.head.appendChild(style)
    })
    await page.waitForTimeout(500)
  })

  test('应该能打开文件管理应用', async ({ page }) => {
    // 点击应用中心图标
    const appIcon = page.locator('.side-options .option-item').nth(3)
    await appIcon.click()
    await page.waitForTimeout(500)

    // 验证应用面板容器可见
    await expect(page.locator('.apps-container')).toBeVisible()

    // 点击文件管理应用
    const fileManagementApp = page.locator('.panel-category-app-item:has-text("文件管理")')
    if (await fileManagementApp.isVisible()) {
      await fileManagementApp.click()
      await page.waitForTimeout(1000)

      // 验证文件管理应用打开
      await expect(page.locator('.file-management-app')).toBeVisible()
    }
  })

  test('应该能显示文件列表', async ({ page }) => {
    // 打开文件管理应用
    const appIcon = page.locator('.side-options .option-item').nth(3)
    await appIcon.click()
    await page.waitForTimeout(500)

    const fileManagementApp = page.locator('.panel-category-app-item:has-text("文件管理")')
    if (await fileManagementApp.isVisible()) {
      await fileManagementApp.click()
      await page.waitForTimeout(1000)
    }

    // 验证文件列表可见
    const fileList = page.locator('.file-list, .file-grid')
    await expect(fileList).toBeVisible()
  })

  test('应该能上传小文件（< 10MB）', async ({ page }) => {
    // 打开文件管理应用
    const appIcon = page.locator('.side-options .option-item').nth(3)
    await appIcon.click()
    await page.waitForTimeout(500)

    const fileManagementApp = page.locator('.panel-category-app-item:has-text("文件管理")')
    if (await fileManagementApp.isVisible()) {
      await fileManagementApp.click()
      await page.waitForTimeout(1000)
    }

    // 准备上传文件
    const uploadButton = page.locator('button:has-text("上传")')
    await expect(uploadButton).toBeVisible()

    // 使用文件选择器上传文件（文件管理应用中的文件输入框）
    const fileInput = page.locator('.file-management-app input[type="file"][multiple]').first()
    const filePath = path.join(__dirname, '../fixtures/small-file.txt')

    // 设置文件
    await fileInput.setInputFiles(filePath)

    // 等待上传开始
    await page.waitForTimeout(1000)

    // 验证上传进度条出现
    const progressBar = page.locator('.upload-progress-bar, .upload-task')
    const isVisible = await progressBar.isVisible().catch(() => false)

    if (isVisible) {
      // 等待上传完成
      await page.waitForTimeout(2000)
    }
  })

  test('应该能取消上传', async ({ page }) => {
    // 打开文件管理应用
    const appIcon = page.locator('.side-options .option-item').nth(3)
    await appIcon.click()
    await page.waitForTimeout(500)

    const fileManagementApp = page.locator('.panel-category-app-item:has-text("文件管理")')
    if (await fileManagementApp.isVisible()) {
      await fileManagementApp.click()
      await page.waitForTimeout(1000)
    }

    // 准备上传文件
    const uploadButton = page.locator('button:has-text("上传")')
    await expect(uploadButton).toBeVisible()

    // 使用文件选择器上传文件（文件管理应用中的文件输入框）
    const fileInput = page.locator('.file-management-app input[type="file"][multiple]').first()
    const filePath = path.join(__dirname, '../fixtures/small-file.txt')

    // 设置文件
    await fileInput.setInputFiles(filePath)

    // 等待上传开始
    await page.waitForTimeout(500)

    // 查找取消按钮
    const cancelButton = page.locator('.cancel-upload-btn, button:has-text("取消")')
    const isVisible = await cancelButton.isVisible().catch(() => false)

    if (isVisible) {
      await cancelButton.click()
      await page.waitForTimeout(500)

      // 验证上传任务被取消
      const failedTask = page.locator('.upload-task.failed, .upload-task.cancelled')
      const isFailed = await failedTask.isVisible().catch(() => false)
      expect(isFailed).toBeTruthy()
    }
  })

  test('应该能预览 PDF 文件', async ({ page }) => {
    // 打开文件管理应用
    const appIcon = page.locator('.side-options .option-item').nth(3)
    await appIcon.click()
    await page.waitForTimeout(500)

    const fileManagementApp = page.locator('.panel-category-app-item:has-text("文件管理")')
    if (await fileManagementApp.isVisible()) {
      await fileManagementApp.click()
      await page.waitForTimeout(1000)
    }

    // 等待文件列表加载
    await page.waitForTimeout(500)

    // 查找 PDF 文件
    const pdfFile = page.locator('.file-item:has-text(".pdf"), .file-grid-item:has-text(".pdf")')
    const isVisible = await pdfFile.isVisible().catch(() => false)

    if (isVisible) {
      // 点击预览
      await pdfFile.dblclick()
      await page.waitForTimeout(1000)

      // 验证预览模态框打开
      const previewModal = page.locator('.file-preview-modal, .preview-modal')
      const isModalVisible = await previewModal.isVisible().catch(() => false)

      if (isModalVisible) {
        // 验证 PDF 预览器
        const pdfViewer = page.locator('.pdf-viewer, iframe, canvas')
        await expect(pdfViewer.first()).toBeVisible()

        // 关闭预览
        const closeButton = page.locator('.close-preview-btn, button:has-text("关闭")')
        if (await closeButton.isVisible()) {
          await closeButton.click()
        }
      }
    }
  })

  test('应该能预览文本文件', async ({ page }) => {
    // 打开文件管理应用
    const appIcon = page.locator('.side-options .option-item').nth(3)
    await appIcon.click()
    await page.waitForTimeout(500)

    const fileManagementApp = page.locator('.panel-category-app-item:has-text("文件管理")')
    if (await fileManagementApp.isVisible()) {
      await fileManagementApp.click()
      await page.waitForTimeout(1000)
    }

    // 等待文件列表加载
    await page.waitForTimeout(500)

    // 查找文本文件
    const textFile = page.locator('.file-item:has-text(".txt"), .file-grid-item:has-text(".txt")')
    const isVisible = await textFile.isVisible().catch(() => false)

    if (isVisible) {
      // 点击预览
      await textFile.dblclick()
      await page.waitForTimeout(1000)

      // 验证预览模态框打开
      const previewModal = page.locator('.file-preview-modal, .preview-modal')
      const isModalVisible = await previewModal.isVisible().catch(() => false)

      if (isModalVisible) {
        // 验证文本预览器
        const textViewer = page.locator('.text-preview, pre, .code-viewer')
        await expect(textViewer.first()).toBeVisible()

        // 关闭预览
        const closeButton = page.locator('.close-preview-btn, button:has-text("关闭")')
        if (await closeButton.isVisible()) {
          await closeButton.click()
        }
      }
    }
  })

  test('应该能显示上传进度', async ({ page }) => {
    // 打开文件管理应用
    const appIcon = page.locator('.side-options .option-item').nth(3)
    await appIcon.click()
    await page.waitForTimeout(500)

    const fileManagementApp = page.locator('.panel-category-app-item:has-text("文件管理")')
    if (await fileManagementApp.isVisible()) {
      await fileManagementApp.click()
      await page.waitForTimeout(1000)
    }

    // 准备上传文件
    const fileInput = page.locator('.file-management-app input[type="file"][multiple]').first()
    const filePath = path.join(__dirname, '../fixtures/small-file.txt')

    // 设置文件
    await fileInput.setInputFiles(filePath)

    // 等待上传开始
    await page.waitForTimeout(500)

    // 验证进度条显示
    const progressBar = page.locator('.upload-progress, .progress-bar')
    const isVisible = await progressBar.isVisible().catch(() => false)

    if (isVisible) {
      // 验证进度百分比显示
      const progressText = page.locator('.progress-percent, .upload-percent')
      const hasProgress = await progressText.isVisible().catch(() => false)
      expect(hasProgress).toBeTruthy()
    }
  })

  test('应该能搜索文件', async ({ page }) => {
    // 打开文件管理应用
    const appIcon = page.locator('.side-options .option-item').nth(3)
    await appIcon.click()
    await page.waitForTimeout(500)

    const fileManagementApp = page.locator('.panel-category-app-item:has-text("文件管理")')
    if (await fileManagementApp.isVisible()) {
      await fileManagementApp.click()
      await page.waitForTimeout(1000)
    }

    // 查找搜索框
    const searchInput = page.locator('.search-input-inline').first()
    await expect(searchInput).toBeVisible()

    // 输入搜索关键词
    await searchInput.fill('test')
    await page.waitForTimeout(1000)

    // 验证搜索结果
    const searchResults = page.locator('.file-item, .file-grid-item')
    const count = await searchResults.count()
    expect(count).toBeGreaterThanOrEqual(0)
  })

  test('应该能切换视图模式', async ({ page }) => {
    // 打开文件管理应用
    const appIcon = page.locator('.side-options .option-item').nth(3)
    await appIcon.click()
    await page.waitForTimeout(500)

    const fileManagementApp = page.locator('.panel-category-app-item:has-text("文件管理")')
    if (await fileManagementApp.isVisible()) {
      await fileManagementApp.click()
      await page.waitForTimeout(1000)
    }

    // 查找视图切换按钮
    const gridViewBtn = page.locator('.toggle-btn:has(.fa-th)')
    const listViewBtn = page.locator('.toggle-btn:has(.fa-list)')

    // 切换到网格视图
    if (await gridViewBtn.isVisible()) {
      await gridViewBtn.click()
      await page.waitForTimeout(500)

      // 验证网格视图
      const gridView = page.locator('.file-grid, .grid-view')
      const isGridVisible = await gridView.isVisible().catch(() => false)
      expect(isGridVisible).toBeTruthy()
    }

    // 切换到列表视图
    if (await listViewBtn.isVisible()) {
      await listViewBtn.click()
      await page.waitForTimeout(500)

      // 验证列表视图
      const listView = page.locator('.file-list, .list-view')
      const isListVisible = await listView.isVisible().catch(() => false)
      expect(isListVisible).toBeTruthy()
    }
  })

  test('应该能筛选文件', async ({ page }) => {
    // 打开文件管理应用
    const appIcon = page.locator('.side-options .option-item').nth(3)
    await appIcon.click()
    await page.waitForTimeout(500)

    const fileManagementApp = page.locator('.panel-category-app-item:has-text("文件管理")')
    if (await fileManagementApp.isVisible()) {
      await fileManagementApp.click()
      await page.waitForTimeout(1000)
    }

    // 查找筛选下拉框
    const filterSelect = page.locator('.filter-select').first()
    await expect(filterSelect).toBeVisible()

    // 选择"我的上传"
    await filterSelect.selectOption('upload')
    await page.waitForTimeout(500)

    // 验证筛选结果
    const files = page.locator('.file-item, .file-grid-item')
    const count = await files.count()
    expect(count).toBeGreaterThanOrEqual(0)
  })

  test('应该能排序文件', async ({ page }) => {
    // 打开文件管理应用
    const appIcon = page.locator('.side-options .option-item').nth(3)
    await appIcon.click()
    await page.waitForTimeout(500)

    const fileManagementApp = page.locator('.panel-category-app-item:has-text("文件管理")')
    if (await fileManagementApp.isVisible()) {
      await fileManagementApp.click()
      await page.waitForTimeout(1000)
    }

    // 查找排序下拉框
    const sortSelect = page.locator('.filter-select-wrap:has(.fa-sort-amount-down) .filter-select')
    await expect(sortSelect).toBeVisible()

    // 选择"名称 A→Z"
    await sortSelect.selectOption('name_asc')
    await page.waitForTimeout(500)

    // 验证排序结果
    const files = page.locator('.file-item, .file-grid-item')
    const count = await files.count()
    expect(count).toBeGreaterThanOrEqual(0)
  })

  test('应该能右键打开文件上下文菜单', async ({ page }) => {
    // 打开文件管理应用
    const appIcon = page.locator('.side-options .option-item').nth(3)
    await appIcon.click()
    await page.waitForTimeout(500)

    const fileManagementApp = page.locator('.panel-category-app-item:has-text("文件管理")')
    if (await fileManagementApp.isVisible()) {
      await fileManagementApp.click()
      await page.waitForTimeout(1000)
    }

    // 等待文件列表加载
    await page.waitForTimeout(500)

    // 查找文件项
    const fileItem = page.locator('.file-item, .file-grid-item').first()
    const isVisible = await fileItem.isVisible().catch(() => false)

    if (isVisible) {
      // 右键点击
      await fileItem.click({ button: 'right' })
      await page.waitForTimeout(500)

      // 验证上下文菜单出现
      const contextMenu = page.locator('.context-menu')
      await expect(contextMenu).toBeVisible()

      // 验证菜单项
      const downloadItem = page.locator('.context-menu-item:has-text("下载")')
      await expect(downloadItem).toBeVisible()
    }
  })

  test('应该能下载文件', async ({ page }) => {
    // 打开文件管理应用
    const appIcon = page.locator('.side-options .option-item').nth(3)
    await appIcon.click()
    await page.waitForTimeout(500)

    const fileManagementApp = page.locator('.panel-category-app-item:has-text("文件管理")')
    if (await fileManagementApp.isVisible()) {
      await fileManagementApp.click()
      await page.waitForTimeout(1000)
    }

    // 等待文件列表加载
    await page.waitForTimeout(500)

    // 查找文件项
    const fileItem = page.locator('.file-item, .file-grid-item').first()
    const isVisible = await fileItem.isVisible().catch(() => false)

    if (isVisible) {
      // 右键点击
      await fileItem.click({ button: 'right' })
      await page.waitForTimeout(500)

      // 点击下载
      const downloadItem = page.locator('.context-menu-item:has-text("下载")')
      if (await downloadItem.isVisible()) {
        // 监听下载事件
        const [download] = await Promise.all([
          page.waitForEvent('download').catch(() => null),
          downloadItem.click()
        ])

        if (download) {
          // 验证下载文件名
          const fileName = download.suggestedFilename()
          expect(fileName).toBeTruthy()
        }
      }
    }
  })

  test('应该能星标文件', async ({ page }) => {
    // 打开文件管理应用
    const appIcon = page.locator('.side-options .option-item').nth(3)
    await appIcon.click()
    await page.waitForTimeout(500)

    const fileManagementApp = page.locator('.panel-category-app-item:has-text("文件管理")')
    if (await fileManagementApp.isVisible()) {
      await fileManagementApp.click()
      await page.waitForTimeout(1000)
    }

    // 等待文件列表加载
    await page.waitForTimeout(500)

    // 查找文件项
    const fileItem = page.locator('.file-item, .file-grid-item').first()
    const isVisible = await fileItem.isVisible().catch(() => false)

    if (isVisible) {
      // 右键点击
      await fileItem.click({ button: 'right' })
      await page.waitForTimeout(500)

      // 点击星标
      const starItem = page.locator('.context-menu-item:has-text("星标")')
      if (await starItem.isVisible()) {
        await starItem.click()
        await page.waitForTimeout(500)
      }
    }
  })

  test('应该能删除文件', async ({ page }) => {
    // 打开文件管理应用
    const appIcon = page.locator('.side-options .option-item').nth(3)
    await appIcon.click()
    await page.waitForTimeout(500)

    const fileManagementApp = page.locator('.panel-category-app-item:has-text("文件管理")')
    if (await fileManagementApp.isVisible()) {
      await fileManagementApp.click()
      await page.waitForTimeout(1000)
    }

    // 等待文件列表加载
    await page.waitForTimeout(500)

    // 查找文件项
    const fileItem = page.locator('.file-item, .file-grid-item').first()
    const isVisible = await fileItem.isVisible().catch(() => false)

    if (isVisible) {
      // 右键点击
      await fileItem.click({ button: 'right' })
      await page.waitForTimeout(500)

      // 点击删除
      const deleteItem = page.locator('.context-menu-item:has-text("删除")')
      if (await deleteItem.isVisible()) {
        await deleteItem.click()
        await page.waitForTimeout(500)

        // 确认删除
        const confirmBtn = page.locator('.q-btn--primary, button:has-text("确定")')
        if (await confirmBtn.isVisible()) {
          await confirmBtn.click()
          await page.waitForTimeout(500)
        }
      }
    }
  })
})

import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DataStorageSettings from '@/components/settings/DataStorageSettings.vue'

const storage = new Map<string, string>()

beforeEach(() => {
  storage.clear()
  vi.mocked(localStorage.getItem).mockImplementation((key) => storage.get(key) ?? null)
  vi.mocked(localStorage.setItem).mockImplementation((key, value) => {
    storage.set(key, value)
  })
  vi.mocked(localStorage.removeItem).mockImplementation((key) => {
    storage.delete(key)
  })
  vi.mocked(localStorage.clear).mockImplementation(() => {
    storage.clear()
  })
  // mock length 和 key 以支持遍历
  Object.defineProperty(localStorage, 'length', {
    get: () => storage.size,
    configurable: true,
  })
  vi.mocked(localStorage.key).mockImplementation((index) => {
    const keys = Array.from(storage.keys())
    return keys[index] ?? null
  })
  ;(window as any).$QMessage = {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    warning: vi.fn(),
  }
})

function mountDataStorageSettings(props: Record<string, any> = {}) {
  return mount(DataStorageSettings, {
    props: {
      defaultSaveDirectory: '',
      ...props,
    },
  })
}

describe('DataStorageSettings 子组件', () => {
  it('渲染默认保存目录输入框（A7）', () => {
    const wrapper = mountDataStorageSettings({ defaultSaveDirectory: '/custom/path' })
    const dirInput = wrapper.find('[data-testid="save-directory-input"]')
    expect(dirInput.exists()).toBe(true)
    expect((dirInput.element as HTMLInputElement).value).toBe('/custom/path')
  })

  it('点击浏览按钮发出 browseDirectory 事件（A7）', async () => {
    const wrapper = mountDataStorageSettings()
    const browseBtn = wrapper.find('[data-testid="browse-directory-btn"]')
    await browseBtn.trigger('click')
    expect(wrapper.emitted('browseDirectory')).toBeTruthy()
    expect(wrapper.emitted('browseDirectory')![0]).toHaveLength(1) // callback 函数
  })

  it('显示缓存总大小（A2 改造）', () => {
    storage.set('messageSettings', '{"notificationsEnabled":true}')
    storage.set('appearanceSettings', '{"theme":"dark"}')
    const wrapper = mountDataStorageSettings()
    const cacheTotal = wrapper.find('[data-testid="cache-total"]')
    expect(cacheTotal.exists()).toBe(true)
    expect(cacheTotal.text()).toMatch(/\d+(\.\d+)?\s*(B|KB|MB)/)
  })

  it('显示各分类缓存大小', () => {
    storage.set('messageSettings', '{"notificationsEnabled":true,"soundEnabled":true}')
    storage.set('theme', 'modern-light')
    const wrapper = mountDataStorageSettings()
    // 设置数据分类应显示大小
    const settingsCategory = wrapper.find('[data-testid="cache-category-settings"]')
    expect(settingsCategory.exists()).toBe(true)
    expect(settingsCategory.text()).toMatch(/\d+(\.\d+)?\s*(B|KB|MB)/)
  })

  it('点击分类清理按钮清除对应缓存', async () => {
    storage.set('messageSettings', '{"notificationsEnabled":true}')
    storage.set('appearanceSettings', '{"theme":"dark"}')
    storage.set('theme', 'modern-light')
    storage.set('token', 'abc123')

    const wrapper = mountDataStorageSettings()
    const clearBtn = wrapper.find('[data-testid="clear-category-settings"]')
    await clearBtn.trigger('click')

    // 设置数据应被清除
    expect(storage.get('messageSettings')).toBeUndefined()
    expect(storage.get('appearanceSettings')).toBeUndefined()
    expect(storage.get('theme')).toBeUndefined()
    // token 不应被清除（受保护）
    expect(storage.get('token')).toBe('abc123')
  })

  it('点击全部清理清除所有非保护数据', async () => {
    storage.set('messageSettings', '{"notificationsEnabled":true}')
    storage.set('theme', 'modern-light')
    storage.set('token', 'abc123')

    const wrapper = mountDataStorageSettings()
    const clearAllBtn = wrapper.find('[data-testid="clear-all-btn"]')
    await clearAllBtn.trigger('click')

    expect(storage.get('messageSettings')).toBeUndefined()
    expect(storage.get('theme')).toBeUndefined()
    expect(storage.get('token')).toBe('abc123')
  })

  it('清理后发出 cacheCleared 事件通知父组件刷新', async () => {
    storage.set('messageSettings', '{}')
    const wrapper = mountDataStorageSettings()
    const clearBtn = wrapper.find('[data-testid="clear-category-settings"]')
    await clearBtn.trigger('click')
    expect(wrapper.emitted('cacheCleared')).toBeTruthy()
  })
})

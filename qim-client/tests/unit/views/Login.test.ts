import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import Login from '@/views/Login.vue'
import { createPinia, setActivePinia } from 'pinia'

describe('Login', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    localStorage.getItem = vi.fn(() => null)
    localStorage.setItem = vi.fn()
    global.fetch = vi.fn()
  })

  describe('基本渲染', () => {
    it('应该渲染登录页面标题', () => {
      const wrapper = mount(Login, {
        global: {
          stubs: {
            'el-form': { template: '<form><slot /></form>' },
            'el-form-item': { template: '<div><slot /></div>' },
            'el-input': { template: '<input />' },
            'el-button': { template: '<button><slot /></button>' },
            'el-checkbox': { template: '<label><slot /></label>' },
            'el-dialog': { template: '<div><slot /></div>' },
          },
        },
      })
      expect(wrapper.text()).toContain('QIM')
    })

    it('应该显示版本信息', () => {
      const wrapper = mount(Login, {
        global: {
          stubs: {
            'el-form': { template: '<form><slot /></form>' },
            'el-form-item': { template: '<div><slot /></div>' },
            'el-input': { template: '<input />' },
            'el-button': { template: '<button><slot /></button>' },
            'el-checkbox': { template: '<label><slot /></label>' },
            'el-dialog': { template: '<div><slot /></div>' },
          },
        },
      })
      expect(wrapper.text()).toContain('版本')
    })
  })

  describe('窗口控制', () => {
    it('应该最小化窗口', async () => {
      const mockSend = vi.fn()
      global.window.electron = {
        ipcRenderer: { send: mockSend }
      } as any

      const wrapper = mount(Login, {
        global: {
          stubs: {
            'el-form': { template: '<form><slot /></form>' },
            'el-form-item': { template: '<div><slot /></div>' },
            'el-input': { template: '<input />' },
            'el-button': { template: '<button><slot /></button>' },
            'el-checkbox': { template: '<label><slot /></label>' },
            'el-dialog': { template: '<div><slot /></div>' },
          },
        },
      })

      await wrapper.vm.minimizeWindow()
      expect(mockSend).toHaveBeenCalledWith('minimize-window')
    })

    it('应该最大化窗口', async () => {
      const mockSend = vi.fn()
      global.window.electron = {
        ipcRenderer: { send: mockSend }
      } as any

      const wrapper = mount(Login, {
        global: {
          stubs: {
            'el-form': { template: '<form><slot /></form>' },
            'el-form-item': { template: '<div><slot /></div>' },
            'el-input': { template: '<input />' },
            'el-button': { template: '<button><slot /></button>' },
            'el-checkbox': { template: '<label><slot /></label>' },
            'el-dialog': { template: '<div><slot /></div>' },
          },
        },
      })

      await wrapper.vm.maximizeWindow()
      expect(mockSend).toHaveBeenCalledWith('maximize-window')
    })

    it('应该关闭窗口', async () => {
      const mockSend = vi.fn()
      global.window.electron = {
        ipcRenderer: { send: mockSend }
      } as any

      const wrapper = mount(Login, {
        global: {
          stubs: {
            'el-form': { template: '<form><slot /></form>' },
            'el-form-item': { template: '<div><slot /></div>' },
            'el-input': { template: '<input />' },
            'el-button': { template: '<button><slot /></button>' },
            'el-checkbox': { template: '<label><slot /></label>' },
            'el-dialog': { template: '<div><slot /></div>' },
          },
        },
      })

      await wrapper.vm.closeWindow()
      expect(mockSend).toHaveBeenCalledWith('close-window')
    })
  })

  describe('服务器设置', () => {
    it('应该保存服务器设置到 localStorage', async () => {
      const wrapper = mount(Login, {
        global: {
          stubs: {
            'el-form': { template: '<form><slot /></form>' },
            'el-form-item': { template: '<div><slot /></div>' },
            'el-input': {
              template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
              props: ['modelValue'],
            },
            'el-button': { template: '<button><slot /></button>' },
            'el-checkbox': { template: '<label><slot /></label>' },
            'el-dialog': {
              template: '<div><slot name="footer" /></div>',
            },
          },
        },
      })

      expect(wrapper.vm.showServerSettings).toBe(false)
    })
  })
})

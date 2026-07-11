import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ModalContainer from '@/components/shared/ModalContainer.vue'

describe('ModalContainer', () => {
  describe('尺寸 props 生效', () => {
    it('width prop 应应用到内容容器', () => {
      const wrapper = mount(ModalContainer, {
        props: {
          visible: true,
          title: '测试',
          width: '680px',
        },
        global: {
          stubs: { Teleport: { template: '<div><slot /></div>' } },
        },
      })

      const content = wrapper.find('.modal-container-content')
      expect(content.attributes('style')).toContain('width: 680px')
    })

    it('minWidth prop 应应用于内容容器', () => {
      const wrapper = mount(ModalContainer, {
        props: {
          visible: true,
          title: '测试',
          minWidth: '500px',
        },
        global: {
          stubs: { Teleport: { template: '<div><slot /></div>' } },
        },
      })

      const content = wrapper.find('.modal-container-content')
      expect(content.attributes('style')).toContain('min-width: 500px')
    })

    it('maxWidth prop 应应用于内容容器', () => {
      const wrapper = mount(ModalContainer, {
        props: {
          visible: true,
          title: '测试',
          maxWidth: '90vw',
        },
        global: {
          stubs: { Teleport: { template: '<div><slot /></div>' } },
        },
      })

      const content = wrapper.find('.modal-container-content')
      expect(content.attributes('style')).toContain('max-width: 90vw')
    })

    it('maxHeight prop 应应用于内容容器', () => {
      const wrapper = mount(ModalContainer, {
        props: {
          visible: true,
          title: '测试',
          maxHeight: '70vh',
        },
        global: {
          stubs: { Teleport: { template: '<div><slot /></div>' } },
        },
      })

      const content = wrapper.find('.modal-container-content')
      expect(content.attributes('style')).toContain('max-height: 70vh')
    })

    it('contentStyle prop 应合并到内容容器样式', () => {
      const wrapper = mount(ModalContainer, {
        props: {
          visible: true,
          title: '测试',
          width: '500px',
          contentStyle: { top: '20px' },
        },
        global: {
          stubs: { Teleport: { template: '<div><slot /></div>' } },
        },
      })

      const content = wrapper.find('.modal-container-content')
      const style = content.attributes('style') || ''
      expect(style).toContain('width: 500px')
      expect(style).toContain('top: 20px')
    })
  })
})

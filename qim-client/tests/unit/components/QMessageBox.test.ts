import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import QMessageBox from '@/components/shared/QMessageBox.vue'

const mountBox = () => mount(QMessageBox, { attachTo: document.body })

describe('QMessageBox confirm', () => {
  it('resolves with confirm action when the confirm button is clicked', async () => {
    const wrapper = mountBox()
    const promise = wrapper.vm.confirm('确定要解散群聊吗？', '确认解散群聊')
    await wrapper.vm.$nextTick()

    await wrapper.find('.q-button--primary').trigger('click')

    await expect(promise).resolves.toMatchObject({ action: 'confirm' })
  })

  it('resolves with cancel action when the cancel button is clicked', async () => {
    const wrapper = mountBox()
    const promise = wrapper.vm.confirm('确定要解散群聊吗？', '确认解散群聊')
    await wrapper.vm.$nextTick()

    await wrapper.find('.q-button--default').trigger('click')

    await expect(promise).resolves.toMatchObject({ action: 'cancel' })
  })

  it('resolves with close action when the close button is clicked', async () => {
    const wrapper = mountBox()
    const promise = wrapper.vm.confirm('确定要解散群聊吗？', '确认解散群聊')
    await wrapper.vm.$nextTick()

    await wrapper.find('.q-message-box__close').trigger('click')

    await expect(promise).resolves.toMatchObject({ action: 'close' })
  })
})

describe('QMessageBox alert', () => {
  it('resolves on confirm and has no cancel button', async () => {
    const wrapper = mountBox()
    const promise = wrapper.vm.alert('操作完成', '提示')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.q-button--default').exists()).toBe(false)
    await wrapper.find('.q-button--primary').trigger('click')

    await expect(promise).resolves.toMatchObject({ action: 'confirm' })
  })
})

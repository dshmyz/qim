import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import ChatToolbar from '@/components/chat/ChatToolbar.vue'

const mountToolbar = (props: Record<string, unknown> = {}) =>
  mount(ChatToolbar, {
    props: { isElectron: false, showAiActions: false, ...props },
    global: {
      stubs: { ChatToolbarButton: { template: '<button @click="$emit(\'click\')"><slot/></button>', emits: ['click'] } },
    },
  })

describe('ChatToolbar code block button', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders a code block button', () => {
    const wrapper = mountToolbar()
    const btn = wrapper.find('[title="代码块"]')
    expect(btn.exists()).toBe(true)
  })

  it('emits open-code-block when code block button is clicked', async () => {
    const wrapper = mountToolbar()
    const btn = wrapper.find('[title="代码块"]')
    await btn.trigger('click')
    expect(wrapper.emitted('open-code-block')).toBeTruthy()
    expect(wrapper.emitted('open-code-block')).toHaveLength(1)
  })
})

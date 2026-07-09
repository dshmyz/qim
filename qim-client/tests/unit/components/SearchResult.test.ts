import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SearchResult from '@/components/conversation/SearchResult.vue'

describe('SearchResult', () => {
  it('shows private chat action for non-group account results such as admin users', async () => {
    const wrapper = mount(SearchResult, {
      props: {
        searchQuery: 'admin',
        searchResults: [
          {
            id: '7',
            name: '管理员',
            username: 'admin',
            type: 'admin',
            status: 'online',
          } as any,
        ],
      },
      global: {
        stubs: {
          Avatar: { template: '<div />' },
        },
      },
    })

    const button = wrapper.find('.search-popup-btn')
    expect(button.exists()).toBe(true)

    await button.trigger('click')

    expect(wrapper.emitted('privateChat')?.[0]?.[0]).toMatchObject({ id: '7', type: 'admin' })
  })
})

import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ShareModal from '@/components/modals/ShareModal.vue'

describe('ShareModal', () => {
  it('filters users by username when sharing', async () => {
    const wrapper = mount(ShareModal, {
      props: {
        visible: true,
        shareType: 'file',
        users: [
          {
            id: '1',
            name: '张三',
            username: 'zhangsan',
            avatar: '',
            department: '研发部',
          },
          {
            id: '2',
            name: '李四',
            username: 'lisi',
            avatar: '',
            department: '产品部',
          },
        ],
        groups: [],
      },
      global: {
        stubs: {
          Avatar: true,
          ModalContainer: {
            template: '<div v-if="visible"><slot /><slot name="footer" /></div>',
            props: ['visible'],
          },
        },
      },
    })

    await wrapper.find('.share-search-input').setValue('lisi')

    expect(wrapper.text()).toContain('李四')
    expect(wrapper.text()).not.toContain('张三')
  })
})

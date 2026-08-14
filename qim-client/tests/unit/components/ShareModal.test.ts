import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ShareModal from '@/components/modals/ShareModal.vue'

describe('ShareModal', () => {
  it('filters users by username when sharing', async () => {
    const wrapper = mount(ShareModal, {
      props: {
        visible: true,
        share: { type: 'file', data: {} },
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
        // 「用户」tab 通过 OrgTreePicker 渲染组织架构中的员工
        departments: [
          {
            id: 'dept-1',
            name: '产品部',
            employees: [
              {
                id: '2',
                name: '李四',
                username: 'lisi',
                avatar: '',
                department: '产品部',
                position: '产品经理',
              },
              {
                id: '1',
                name: '张三',
                username: 'zhangsan',
                avatar: '',
                department: '研发部',
                position: '工程师',
              },
            ],
            subDepartments: [],
          },
        ],
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

    // 切换到「用户」tab 后才渲染组织架构成员列表
    await wrapper.findAll('.share-tab')[1].trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('李四')
    expect(wrapper.text()).toContain('张三')

    await wrapper.find('.share-search-input').setValue('lisi')
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('李四')
    expect(wrapper.text()).not.toContain('张三')
  })
})

import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SelfProfileModal from '@/components/modals/SelfProfileModal.vue'

const mountModal = (props: Partial<any> = {}) => {
  return mount(SelfProfileModal, {
    props: {
      visible: true,
      currentUser: {
        username: 'alice',
        avatar: '/avatar.png',
        id: 42,
      },
      serverUrl: 'http://localhost:3000',
      profile: {
        nickname: '爱丽丝',
        username: 'alice',
        signature: '你好',
        phone: '13800138000',
        email: 'alice@example.com',
        gender: 'female',
        department: '研发部',
        joinDate: '2024-03-15',
      },
      ...props,
    },
    global: {
      stubs: {
        Teleport: { template: '<div><slot /></div>' },
        AvatarCropper: true,
      },
    },
  })
}

describe('SelfProfileModal', () => {
  describe('字段展示', () => {
    it('应展示昵称、账号、签名', async () => {
      const wrapper = mountModal()
      await wrapper.vm.$nextTick()
      const text = wrapper.text()
      expect(text).toContain('爱丽丝')
      expect(text).toContain('alice')
      const textarea = wrapper.find('textarea')
      expect((textarea.element as HTMLTextAreaElement).value).toBe('你好')
    })

    it('应展示手机号', async () => {
      const wrapper = mountModal()
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).toContain('13800138000')
    })

    it('应展示邮箱', async () => {
      const wrapper = mountModal()
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).toContain('alice@example.com')
    })

    it('应展示性别', async () => {
      const wrapper = mountModal()
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).toContain('女')
    })

    it('应展示部门', async () => {
      const wrapper = mountModal()
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).toContain('研发部')
    })

    it('应展示加入时间', async () => {
      const wrapper = mountModal()
      await wrapper.vm.$nextTick()
      expect(wrapper.text()).toContain('2024-03-15')
    })

    it('不应展示 ID 字段', async () => {
      const wrapper = mountModal()
      await wrapper.vm.$nextTick()
      const labels = wrapper.findAll('.form-label').map(l => l.text())
      expect(labels).not.toContain('ID')
    })
  })

  describe('空值处理', () => {
    it('部门和加入时间为空时显示未设置', async () => {
      const wrapper = mountModal({
        profile: {
          nickname: '爱丽丝',
          username: 'alice',
          signature: '',
          phone: '',
          email: '',
          gender: 'secret',
          department: '',
          joinDate: '',
        },
      })
      await wrapper.vm.$nextTick()
      const text = wrapper.text()
      expect(text).toContain('未设置')
      expect(text).toContain('保密')
    })
  })

  describe('可编辑字段', () => {
    it('签名应是可编辑的 textarea', async () => {
      const wrapper = mountModal()
      await wrapper.vm.$nextTick()
      const textarea = wrapper.find('textarea')
      expect(textarea.exists()).toBe(true)
    })

    it('手机号应是只读展示', async () => {
      const wrapper = mountModal()
      await wrapper.vm.$nextTick()
      const phoneInput = wrapper.find('input[data-field="phone"]')
      expect(phoneInput.exists()).toBe(false)
      expect(wrapper.text()).toContain('13800138000')
    })

    it('邮箱应是只读展示', async () => {
      const wrapper = mountModal()
      await wrapper.vm.$nextTick()
      const emailInput = wrapper.find('input[data-field="email"]')
      expect(emailInput.exists()).toBe(false)
      expect(wrapper.text()).toContain('alice@example.com')
    })
  })

  describe('保存', () => {
    it('保存时应 emit 含 phone 和 email 的 profile', async () => {
      const wrapper = mountModal()
      await wrapper.vm.$nextTick()

      await wrapper.find('.action-btn.primary').trigger('click')

      const emitted = wrapper.emitted('save')
      expect(emitted).toBeTruthy()
      const savedProfile = emitted![0][0] as any
      expect(savedProfile.phone).toBe('13800138000')
      expect(savedProfile.email).toBe('alice@example.com')
      expect(savedProfile.signature).toBe('你好')
    })
  })
})

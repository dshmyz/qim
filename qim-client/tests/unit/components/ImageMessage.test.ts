import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import ImageMessage from '@/components/message/ImageMessage.vue'
import ImagePlaceholder from '@/components/message/ImagePlaceholder.vue'

vi.mock('@/composables/useIntersectionObserver', () => ({
  useIntersectionObserver: () => ({ isVisible: ref(true) }),
}))

describe('image message media preview style', () => {
  it('renders images as media previews instead of attachment cards', () => {
    const wrapper = mount(ImageMessage, {
      props: {
        src: JSON.stringify({ url: '/uploads/photo.png' }),
        serverUrl: 'http://localhost:8080',
      },
    })

    expect(wrapper.find('.media-preview').exists()).toBe(true)
    expect(wrapper.find('.media-preview__image').exists()).toBe(true)
    expect(wrapper.find('.attachment-card').exists()).toBe(false)
  })

  it('uses the same media preview shell for loading and error states', () => {
    const wrapper = mount(ImagePlaceholder, {
      props: {
        text: '加载中...',
        isLoading: true,
      },
    })

    expect(wrapper.find('.media-preview-placeholder').exists()).toBe(true)
  })
})

import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ImagePreviewDialog from '@/components/chat/ImagePreviewDialog.vue'

describe('ImagePreviewDialog', () => {
  it('renders a viewer source image for Viewer.js preview', () => {
    const wrapper = mount(ImagePreviewDialog, {
      props: {
        visible: true,
        imageUrl: 'http://localhost:8080/uploads/photo.png',
      },
    })

    expect(wrapper.find('.image-viewer-source').exists()).toBe(true)
    expect(wrapper.find('.image-viewer-source img').attributes('src')).toBe('http://localhost:8080/uploads/photo.png')
    expect(wrapper.find('.image-preview-viewer').exists()).toBe(false)
  })
})

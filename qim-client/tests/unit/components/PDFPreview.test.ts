import { mount, flushPromises } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PDFPreview from '../../../src/components/apps/file/PDFPreview.vue'

const { getPage, renderPage } = vi.hoisted(() => ({
  getPage: vi.fn(),
  renderPage: vi.fn(),
}))

vi.mock('pdfjs-dist', () => ({
  GlobalWorkerOptions: {
    workerSrc: '',
  },
  VerbosityLevel: {
    ERRORS: 0,
  },
  getDocument: vi.fn(() => ({
    promise: Promise.resolve({
      numPages: 1,
      getPage,
      destroy: vi.fn(),
    }),
  })),
}))

vi.mock('pdfjs-dist/build/pdf.worker.min.mjs?url', () => ({
  default: '/mock-pdf-worker.mjs',
}))

describe('PDFPreview', () => {
  function createDeferred() {
    let resolve!: () => void
    const promise = new Promise<void>((res) => {
      resolve = res
    })
    return { promise, resolve }
  }

  beforeEach(() => {
    getPage.mockReset()
    renderPage.mockReset()
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({} as CanvasRenderingContext2D)
    getPage.mockImplementation(() => Promise.resolve({
      getViewport: ({ scale }: { scale: number }) => ({
        width: Math.round(100 * scale),
        height: Math.round(120 * scale),
      }),
      render: renderPage.mockImplementation(() => ({
        promise: Promise.resolve(),
      })),
    }))
  })

  it('keeps the loading state visible until the first page finishes rendering', async () => {
    const firstRender = createDeferred()
    renderPage.mockImplementationOnce(() => ({
      promise: firstRender.promise,
    }))

    const wrapper = mount(PDFPreview, {
      props: {
        url: 'blob:pdf',
        filename: 'report.pdf',
      },
    })

    await flushPromises()
    await flushPromises()

    expect(wrapper.find('.pdf-loading').exists()).toBe(true)

    firstRender.resolve()
    await flushPromises()
    await flushPromises()

    expect(wrapper.find('.pdf-loading').exists()).toBe(false)
    expect((wrapper.find('canvas').element as HTMLCanvasElement).width).toBe(100)
  })

  it('rerenders the canvas at the new scale when zooming in', async () => {
    const wrapper = mount(PDFPreview, {
      props: {
        url: 'blob:pdf',
        filename: 'report.pdf',
      },
    })

    await flushPromises()
    await flushPromises()
    expect(wrapper.find('.scale-info').text()).toBe('100%')
    expect((wrapper.find('canvas').element as HTMLCanvasElement).width).toBe(100)

    await wrapper.find('button[title="放大"]').trigger('click')
    await flushPromises()
    await flushPromises()

    expect(wrapper.find('.scale-info').text()).toBe('125%')
    expect((wrapper.find('canvas').element as HTMLCanvasElement).width).toBe(125)
  })
})

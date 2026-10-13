import { beforeEach, describe, expect, it, vi } from 'vitest'
import { downloadBlob } from '@/utils/download'

describe('download utils', () => {
  let click: ReturnType<typeof vi.fn>

  beforeEach(() => {
    click = vi.fn()
    vi.spyOn(document, 'createElement').mockReturnValue({
      href: '',
      download: '',
      click,
    } as unknown as HTMLAnchorElement)
    vi.spyOn(document.body, 'appendChild').mockImplementation((node: Node) => node)
    vi.spyOn(document.body, 'removeChild').mockImplementation((node: Node) => node)
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:download')
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
  })

  it('downloads an existing blob through an object URL', () => {
    downloadBlob(new Blob(['hello']), 'hello.txt')

    expect(URL.createObjectURL).toHaveBeenCalled()
    expect(click).toHaveBeenCalled()
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:download')
  })
})

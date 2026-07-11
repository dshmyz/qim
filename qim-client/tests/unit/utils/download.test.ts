import { beforeEach, describe, expect, it, vi } from 'vitest'
import { downloadBlob, downloadUrl } from '@/utils/download'

describe('download utils', () => {
  let click: ReturnType<typeof vi.fn>

  beforeEach(() => {
    click = vi.fn()
    vi.stubGlobal('fetch', vi.fn())
    vi.spyOn(document, 'createElement').mockReturnValue({
      href: '',
      download: '',
      click,
    } as unknown as HTMLAnchorElement)
    vi.spyOn(document.body, 'appendChild').mockImplementation((node: Node) => node)
    vi.spyOn(document.body, 'removeChild').mockImplementation((node: Node) => node)
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:download')
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
    delete (window as any).electron
  })

  it('downloads an existing blob through an object URL', () => {
    downloadBlob(new Blob(['hello']), 'hello.txt')

    expect(URL.createObjectURL).toHaveBeenCalled()
    expect(click).toHaveBeenCalled()
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:download')
  })

  it('delegates URL downloads to Electron IPC without buffering in the renderer', async () => {
    const send = vi.fn()
    ;(window as any).electron = {
      ipcRenderer: { send },
    }

    const result = await downloadUrl({
      url: 'http://localhost/files/big.zip',
      filename: 'big.zip',
      token: 'token-1',
      saveDir: '/tmp',
      downloadId: '42',
    })

    expect(result.mode).toBe('electron')
    expect(fetch).not.toHaveBeenCalled()
    expect(send).toHaveBeenCalledWith('download-file', {
      url: 'http://localhost/files/big.zip',
      token: 'token-1',
      fileName: 'big.zip',
      saveDir: '/tmp',
      downloadId: '42',
    })
  })

  it('falls back to fetch and blob download in browser environments', async () => {
    ;(fetch as any).mockResolvedValue({
      ok: true,
      blob: () => Promise.resolve(new Blob(['hello'])),
    })

    const result = await downloadUrl({
      url: 'http://localhost/files/hello.txt',
      filename: 'hello.txt',
      token: 'token-1',
    })

    expect(result.mode).toBe('browser')
    expect(fetch).toHaveBeenCalledWith('http://localhost/files/hello.txt', {
      method: 'GET',
      cache: 'no-store',
      headers: {
        Authorization: 'Bearer token-1',
      },
    })
    expect(click).toHaveBeenCalled()
  })
})

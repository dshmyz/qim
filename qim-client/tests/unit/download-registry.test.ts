import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { DownloadRegistry } from '../../electron/download-registry.js'

describe('DownloadRegistry', () => {
  it('uses original URL directly (no nonce), so same URL overwrites previous meta', () => {
    const registry = new DownloadRegistry()
    const first = registry.create({
      url: 'https://example.com/files/1/download',
      token: 'first-token',
      fileName: 'first.txt',
      downloadId: 'message-1',
      completeChannel: 'download-complete'
    })
    const second = registry.create({
      url: 'https://example.com/files/1/download',
      token: 'second-token',
      fileName: 'second.txt',
      downloadId: 'message-2',
      completeChannel: 'download-complete'
    })

    // 新实现直接用原始 URL，不加 nonce，因此 requestUrl 相同
    expect(first.requestUrl).toBe(second.requestUrl)
    // 第二次 create 覆盖了第一次的 meta
    expect(registry.consume(first.requestUrl)?.fileName).toBe('second.txt')
    // consume 之后 registry 清空，再次 consume 返回 null
    expect(registry.consume(second.requestUrl)).toBeNull()
  })

  it('looks up the correct auth header for each pending request URL', () => {
    const registry = new DownloadRegistry()
    const first = registry.create({
      url: 'https://example.com/files/1/download',
      token: 'first-token',
      downloadId: 'message-1',
      completeChannel: 'download-complete'
    })
    const second = registry.create({
      url: 'https://example.com/files/2/download',
      token: 'second-token',
      downloadId: 'message-2',
      completeChannel: 'download-complete'
    })

    expect(registry.getAuthHeader(first.requestUrl)).toBe('Bearer first-token')
    expect(registry.getAuthHeader(second.requestUrl)).toBe('Bearer second-token')
  })

  it('main process uses one download handler and unique registry URLs', () => {
    const mainProcess = readFileSync(resolve(__dirname, '../../electron/main.js'), 'utf8')

    expect(mainProcess).toContain("sess.webRequest.onBeforeSendHeaders({ urls: ['<all_urls>'] }")
    expect(mainProcess).toContain('const meta = downloadRegistry.create')
    expect(mainProcess).toContain('contents.downloadURL(meta.requestUrl)')
    expect(mainProcess).not.toContain('pendingDownloads.set(url')
    expect(mainProcess).not.toContain('onBeforeSendHeaders({ urls: [url] }')
  })
})

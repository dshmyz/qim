import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { DownloadRegistry } from '../../electron/download-registry.js'

describe('DownloadRegistry', () => {
  it('same URL queues concurrent downloads FIFO (no nonce), first-in first-out', () => {
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

    // 同 URL 不加 nonce，requestUrl 相同
    expect(first.requestUrl).toBe(second.requestUrl)
    // FIFO：第一次 consume 返回先进入队列的条目（first.txt）
    expect(registry.consume(first.requestUrl)?.fileName).toBe('first.txt')
    // 第二次 consume 返回后进入的条目（second.txt）
    expect(registry.consume(first.requestUrl)?.fileName).toBe('second.txt')
    // 队列已空，再次 consume 返回 null
    expect(registry.consume(first.requestUrl)).toBeNull()
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

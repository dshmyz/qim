import { describe, expect, it } from 'vitest'
import { EventEmitter } from 'node:events'
import { PassThrough } from 'node:stream'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { DownloadManager } from '../../electron/download-manager.js'

interface FakeRequest extends EventEmitter {
  sentHeaders: Record<string, string>
  end: () => void
}

function makeRequestFn() {
  const requests: FakeRequest[] = []
  const fn = (opts: any) => {
    const req = new EventEmitter() as FakeRequest
    req.sentHeaders = opts.headers || {}
    req.end = () => {}
    requests.push(req)
    return req
  }
  return { fn, requests }
}

function makeResponse(statusCode: number, headers: Record<string, string> = {}) {
  const res = new PassThrough() as any
  res.statusCode = statusCode
  res.headers = headers
  return res
}

function runDownload(manager: DownloadManager, opts: Record<string, unknown>) {
  const progressEvents: any[][] = []
  const done = new Promise<any>((resolve) => {
    manager.start({
      url: 'http://localhost:8080/api/v1/files/1/download',
      fileName: 'a.txt',
      ...opts,
      onProgress: (...args: any[]) => progressEvents.push(args),
      onDone: (id: string, result: any) => resolve({ id, result })
    })
  })
  return { progressEvents, done }
}

function tmpDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'dlm-'))
}

describe('DownloadManager', () => {
  it('downloads to disk and reports real percentage from content-length', async () => {
    const { fn, requests } = makeRequestFn()
    const dir = tmpDir()
    const manager = new DownloadManager({ requestFn: fn, getDownloadsDir: () => dir })
    const { progressEvents, done } = runDownload(manager, { downloadId: 'msg-1', token: 'tok' })

    const res = makeResponse(200, { 'content-length': '5' })
    requests[0].emit('response', res)
    res.write('hello')
    res.end()

    const { result } = await done
    expect(result.success).toBe(true)
    expect(result.filePath).toBe(path.join(dir, 'a.txt'))
    expect(fs.readFileSync(result.filePath, 'utf8')).toBe('hello')

    const states = progressEvents.map(e => e[4])
    expect(states).toContain('progressing')
    expect(states).toContain('completed')
    const completed = progressEvents[progressEvents.length - 1]
    expect(completed[1]).toBe(100)
    expect(completed[3]).toBe(5)
  })

  it('sends indeterminate when content-length is missing (chunked)', async () => {
    const { fn, requests } = makeRequestFn()
    const manager = new DownloadManager({ requestFn: fn, getDownloadsDir: tmpDir })
    const { progressEvents, done } = runDownload(manager, { downloadId: 'msg-2' })

    const res = makeResponse(200, {})
    requests[0].emit('response', res)
    res.write('abc')
    res.end()

    const { result } = await done
    expect(result.success).toBe(true)

    const indeterminate = progressEvents.find(e => e[4] === 'indeterminate')
    expect(indeterminate).toBeDefined()
    expect(indeterminate![2]).toBeGreaterThan(0) // 有字节在流，只是总量未知
  })

  it('reports HTTP error as failure and leaves no partial file', async () => {
    const { fn, requests } = makeRequestFn()
    const dir = tmpDir()
    const manager = new DownloadManager({ requestFn: fn, getDownloadsDir: () => dir })
    const { progressEvents, done } = runDownload(manager, { downloadId: 'msg-3' })

    const res = makeResponse(404)
    requests[0].emit('response', res)
    res.end()

    const { result } = await done
    expect(result.success).toBe(false)
    expect(result.error).toContain('404')
    expect(progressEvents.map(e => e[4])).toContain('interrupted')
    expect(fs.existsSync(path.join(dir, 'a.txt'))).toBe(false)
  })

  it('reports network-level failure (server down) instead of hanging', async () => {
    const { fn, requests } = makeRequestFn()
    const manager = new DownloadManager({ requestFn: fn, getDownloadsDir: tmpDir })
    const { progressEvents, done } = runDownload(manager, { downloadId: 'msg-4' })

    requests[0].emit('error', new Error('connect ECONNREFUSED 127.0.0.1:8080'))

    const { result } = await done
    expect(result.success).toBe(false)
    expect(result.error).toContain('ECONNREFUSED')
    expect(progressEvents.map(e => e[4])).toContain('interrupted')
  })

  it('removes partial file when the stream errors mid-download', async () => {
    const { fn, requests } = makeRequestFn()
    const dir = tmpDir()
    const manager = new DownloadManager({ requestFn: fn, getDownloadsDir: () => dir })
    const { done } = runDownload(manager, { downloadId: 'msg-5' })

    const res = makeResponse(200, { 'content-length': '100' })
    requests[0].emit('response', res)
    res.write('partial-data')
    res.emit('error', new Error('socket hang up'))

    const { result } = await done
    expect(result.success).toBe(false)
    expect(fs.existsSync(path.join(dir, 'a.txt'))).toBe(false)
  })

  it('follows redirect and drops Authorization across origins', async () => {
    const { fn, requests } = makeRequestFn()
    const manager = new DownloadManager({ requestFn: fn, getDownloadsDir: tmpDir })
    const { done } = runDownload(manager, { downloadId: 'msg-6', token: 'secret' })

    const res1 = makeResponse(302, { location: 'https://other.example/file.bin' })
    requests[0].emit('response', res1)
    res1.end()

    const res2 = makeResponse(200, { 'content-length': '3' })
    requests[1].emit('response', res2)
    res2.write('abc')
    res2.end()

    const { result } = await done
    expect(result.success).toBe(true)
    expect(requests).toHaveLength(2)
    expect(requests[0].sentHeaders.Authorization).toBe('Bearer secret')
    expect(requests[1].sentHeaders.Authorization).toBeUndefined()
  })

  it('auto-generates downloadId so concurrent no-id downloads do not collide', async () => {
    const { fn, requests } = makeRequestFn()
    const dir = tmpDir()
    const manager = new DownloadManager({ requestFn: fn, getDownloadsDir: () => dir })
    const start = (url: string, fileName: string) => new Promise<any>((resolve) => {
      manager.start({
        url,
        fileName,
        onProgress: () => {},
        onDone: (_id: string, r: any) => resolve(r)
      })
    })
    const p1 = start('http://localhost:8080/api/v1/files/1/download', 'a.txt')
    const p2 = start('http://localhost:8080/api/v1/files/2/download', 'b.txt')
    // 两个请求都发起了，而不是第二个被"正在下载中"拒绝
    expect(requests).toHaveLength(2)

    const res1 = makeResponse(200, { 'content-length': '1' })
    requests[0].emit('response', res1)
    res1.write('x')
    res1.end()
    const res2 = makeResponse(200, { 'content-length': '1' })
    requests[1].emit('response', res2)
    res2.write('y')
    res2.end()

    const [r1, r2] = await Promise.all([p1, p2])
    expect(r1.success).toBe(true)
    expect(r2.success).toBe(true)
  })

  it('rejects a truncated download when bytes mismatch Content-Length', async () => {
    const { fn, requests } = makeRequestFn()
    const dir = tmpDir()
    const manager = new DownloadManager({ requestFn: fn, getDownloadsDir: () => dir })
    const { done } = runDownload(manager, { downloadId: 'msg-trunc' })

    const res = makeResponse(200, { 'content-length': '10' })
    requests[0].emit('response', res)
    res.write('hello') // 只写了 5 字节，服务器声明 10
    res.end()          // 干净结束，不报错

    const { result } = await done
    expect(result.success).toBe(false)
    expect(result.error).toContain('下载不完整')
    expect(fs.existsSync(path.join(dir, 'a.txt'))).toBe(false)
  })

  it('falls back to url when id-based download returns 404 (migrated data)', async () => {
    const { fn, requests } = makeRequestFn()
    const dir = tmpDir()
    const manager = new DownloadManager({ requestFn: fn, getDownloadsDir: () => dir })
    const { progressEvents, done } = runDownload(manager, {
      downloadId: 'msg-migrated',
      token: 'tok',
      fallbackUrl: 'http://localhost:8080/static/uploads/2026/09/migrated.pdf',
    })

    // 主地址：id 接口 404
    const res1 = makeResponse(404)
    requests[0].emit('response', res1)
    res1.end()

    // 回退：原始 url 正常
    const res2 = makeResponse(200, { 'content-length': '3' })
    requests[1].emit('response', res2)
    res2.write('abc')
    res2.end()

    const { result } = await done
    expect(result.success).toBe(true)
    expect(requests).toHaveLength(2)
    expect(requests[1].sentHeaders.Authorization).toBe('Bearer tok')
    expect(fs.readFileSync(result.filePath, 'utf8')).toBe('abc')
    expect(progressEvents.map(e => e[4])).toContain('completed')
  })

  it('fails when the fallback url also returns an error (no infinite retry)', async () => {
    const { fn, requests } = makeRequestFn()
    const manager = new DownloadManager({ requestFn: fn, getDownloadsDir: tmpDir })
    const { done } = runDownload(manager, {
      downloadId: 'msg-migrated2',
      fallbackUrl: 'http://localhost:8080/static/uploads/2026/09/gone.pdf',
    })

    const res1 = makeResponse(404)
    requests[0].emit('response', res1)
    res1.end()
    const res2 = makeResponse(500)
    requests[1].emit('response', res2)
    res2.end()

    const { result } = await done
    expect(result.success).toBe(false)
    expect(requests).toHaveLength(2) // 只回退一次，不再无限重试
  })

  it('rejects a duplicate start for the same downloadId', () => {
    const { fn } = makeRequestFn()
    const manager = new DownloadManager({ requestFn: fn, getDownloadsDir: tmpDir })
    runDownload(manager, { downloadId: 'dup' })

    let second: any
    manager.start({
      url: 'http://localhost:8080/api/v1/files/2/download',
      fileName: 'b.txt',
      downloadId: 'dup',
      onProgress: () => {},
      onDone: (_id: string, r: any) => { second = r }
    })
    expect(second.success).toBe(false)
  })

  it('rejects concurrent downloads targeting the same file path (different ids)', async () => {
    const { fn, requests } = makeRequestFn()
    // 固定目录：p1/p2/p3 必须解析到同一个目标路径才能触发/释放去重
    const dir = tmpDir()
    const manager = new DownloadManager({ requestFn: fn, getDownloadsDir: () => dir })

    // 第一个下载发起但尚未收到响应，占用目标路径
    const first = runDownload(manager, { downloadId: 'p1' })
    const res1 = makeResponse(200, { 'content-length': '2' })
    requests[0].emit('response', res1)

    // 同名文件、不同 downloadId 的第二次下载 → 按目标路径拒绝
    const second = new Promise<any>((resolve) => {
      manager.start({
        url: 'http://localhost:8080/api/v1/files/9/download',
        fileName: 'a.txt',
        downloadId: 'p2',
        onProgress: () => {},
        onDone: (_id: string, r: any) => resolve(r)
      })
    })
    const r2 = await second
    expect(r2.success).toBe(false)
    expect(r2.error).toBe('该文件正在下载中')

    // 首个下载完成后释放路径，同名下载可再次进行
    res1.write('ok')
    res1.end()
    const { result } = await first.done
    expect(result.success).toBe(true)

    const third = new Promise<any>((resolve) => {
      manager.start({
        url: 'http://localhost:8080/api/v1/files/9/download',
        fileName: 'a.txt',
        downloadId: 'p3',
        onProgress: () => {},
        onDone: (_id: string, r: any) => resolve(r)
      })
    })
    const res3 = makeResponse(200, { 'content-length': '2' })
    requests[1].emit('response', res3)
    res3.write('hi')
    res3.end()
    const r3 = await third
    expect(r3.success).toBe(true)
  })

  it('prioritizes savePath and sanitizes fileName to prevent traversal', () => {
    const dl = tmpDir()
    const manager = new DownloadManager({ requestFn: () => { throw new Error('unused') }, getDownloadsDir: () => dl })
    expect(manager.resolveTargetPath({ savePath: '/tmp/x/y.bin', fileName: 'a.txt' })).toBe('/tmp/x/y.bin')
    const dir = tmpDir()
    expect(manager.resolveTargetPath({ saveDir: dir, fileName: '../../evil.txt' })).toBe(path.join(dir, 'evil.txt'))
    expect(manager.resolveTargetPath({ saveDir: undefined, fileName: '../up.bin' })).toBe(path.join(dl, 'up.bin'))
  })
})

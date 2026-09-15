// 显式下载管理器：不再用 webContents.downloadURL（fire-and-forget，服务端连不上时
// Electron 不派发 DownloadItem、will-download 不触发，前端永远收不到终态而卡在 0%）。
// 改用注入的 requestFn（生产传 Electron net.request）显式请求→流式写盘，
// 成功/失败/进度全部通过 onProgress/onDone 明确回传；requestFn/getDownloadsDir 可注入以便单测。
import fs from 'fs'
import path from 'path'

const MAX_REDIRECTS = 5

export class DownloadManager {
  constructor({ requestFn, getDownloadsDir }) {
    if (typeof requestFn !== 'function') throw new Error('DownloadManager requires requestFn')
    this.requestFn = requestFn
    this.getDownloadsDir = getDownloadsDir || (() => '')
    this.counter = 0
    this.active = new Set()
  }

  start({ url, token, fileName, saveDir, savePath, downloadId, fallbackUrl, onProgress, onDone }) {
    // 未传 downloadId（如 MessageManager 的 downloadUrl）时自动生成，避免 undefined 互相误判为重复下载
    const id = downloadId || `download-${Date.now()}-${++this.counter}`
    if (this.active.has(id)) {
      onDone(id, { success: false, error: '该文件正在下载中' })
      return id
    }
    let target
    try {
      target = this.resolveTargetPath({ saveDir, savePath, fileName })
    } catch (err) {
      onProgress(id, 0, 0, 0, 'interrupted')
      onDone(id, { success: false, error: err?.message || '无法创建保存目录' })
      return id
    }
    this.active.add(id)
    this.requestWithRedirects({ url, token, target, downloadId: id, fallbackUrl, fallbackUsed: false, onProgress, onDone, redirects: 0 })
    return id
  }

  resolveTargetPath({ saveDir, savePath, fileName }) {
    if (savePath) return savePath
    const dir = saveDir && saveDir !== '~/Downloads' ? saveDir : this.getDownloadsDir()
    if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true })
    return path.join(dir, sanitizeFileName(fileName))
  }

  requestWithRedirects({ url, token, target, downloadId, fallbackUrl, fallbackUsed, onProgress, onDone, redirects }) {
    const headers = {}
    if (token) headers.Authorization = `Bearer ${token}`

    let req
    try {
      req = this.requestFn({ url, method: 'GET', headers })
    } catch (err) {
      this.fail(downloadId, target, onProgress, onDone, err)
      return
    }

    // 收尾只执行一次（error/finish 竞争时）
    let settled = false
    const settle = (fn) => {
      if (settled) return
      settled = true
      fn()
    }

    req.on('response', (response) => {
      const status = response.statusCode
      const location = response.headers?.location

      if (status >= 300 && status < 400 && location) {
        response.resume()
        if (redirects >= MAX_REDIRECTS) {
          settle(() => this.fail(downloadId, target, onProgress, onDone, new Error('重定向次数过多')))
          return
        }
        const nextUrl = new URL(location, url).toString()
        // 跨域重定向丢弃 Authorization，避免 token 泄漏到第三方
        const nextToken = sameOrigin(url, nextUrl) ? token : ''
        this.requestWithRedirects({ url: nextUrl, token: nextToken, target, downloadId, fallbackUrl, fallbackUsed, onProgress, onDone, redirects: redirects + 1 })
        return
      }

      if (status < 200 || status >= 300) {
        response.resume()
        // 迁移数据兜底：content.id 是旧系统 ID，新库查不到（400 超长 / 404 不存在），
        // 有 fallbackUrl（content.url）时回退到原始 url 重试一次，避免这类文件下载直接失败。
        if (!fallbackUsed && (status === 400 || status === 404) && fallbackUrl) {
          const nextToken = sameOrigin(url, fallbackUrl) ? token : ''
          this.requestWithRedirects({ url: fallbackUrl, token: nextToken, target, downloadId, fallbackUrl, fallbackUsed: true, onProgress, onDone, redirects: 0 })
          return
        }
        settle(() => this.fail(downloadId, target, onProgress, onDone, new Error(`服务器返回 HTTP ${status}`)))
        return
      }

      const total = parseContentLength(response.headers?.['content-length'])
      let received = 0

      let fileStream
      try {
        fileStream = fs.createWriteStream(target)
      } catch (err) {
        response.resume()
        settle(() => this.fail(downloadId, target, onProgress, onDone, err))
        return
      }

      const sendProgress = (state) => {
        const percent = total > 0 ? Math.min(100, Math.floor((received / total) * 100)) : 0
        onProgress(downloadId, percent, received, total, state)
      }

      response.pipe(fileStream)

      response.on('data', (chunk) => {
        received += chunk.length
        sendProgress(total > 0 ? 'progressing' : 'indeterminate')
      })
      response.on('error', (err) => {
        fileStream.destroy()
        settle(() => this.fail(downloadId, target, onProgress, onDone, err))
      })
      fileStream.on('error', (err) => {
        response.destroy()
        settle(() => this.fail(downloadId, target, onProgress, onDone, err))
      })
      fileStream.on('finish', () => {
        settle(() => {
          // 已知 Content-Length 但实际字节数不符 → 下载不完整（截断），按失败处理并删半成品，
          // 避免留下损坏的文件（zip 等压缩包打不开）。
          if (total > 0 && received !== total) {
            this.fail(downloadId, target, onProgress, onDone, new Error(`下载不完整 (${received}/${total} 字节)`))
            return
          }
          this.succeed(downloadId, target, onProgress, onDone, received, total > 0 ? total : received)
        })
      })
    })

    req.on('error', (err) => {
      settle(() => this.fail(downloadId, target, onProgress, onDone, err))
    })

    req.end()
  }

  succeed(downloadId, target, onProgress, onDone, received, total) {
    this.active.delete(downloadId)
    onProgress(downloadId, 100, received, total, 'completed')
    onDone(downloadId, { success: true, filePath: target })
  }

  fail(downloadId, target, onProgress, onDone, err) {
    this.active.delete(downloadId)
    cleanupPartial(target)
    const message = err?.message || '下载失败'
    onProgress(downloadId, 0, 0, 0, 'interrupted')
    onDone(downloadId, { success: false, error: message })
  }
}

function sanitizeFileName(name) {
  const base = path.basename(String(name || 'download'))
  const safe = base.replace(/[\x00-\x1f]/g, '_').trim()
  return safe || 'download'
}

function parseContentLength(value) {
  if (value == null) return 0
  const n = parseInt(String(value).trim(), 10)
  return Number.isFinite(n) && n > 0 ? n : 0
}

function sameOrigin(a, b) {
  try {
    return new URL(a).origin === new URL(b).origin
  } catch {
    return false
  }
}

function cleanupPartial(target) {
  try {
    if (fs.existsSync(target)) fs.rmSync(target, { force: true })
  } catch {
    // 忽略清理失败（下次下载会覆盖）
  }
}

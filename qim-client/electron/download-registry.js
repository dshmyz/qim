export class DownloadRegistry {
  constructor() {
    this.counter = 0
    this.byOriginalUrl = new Map()
  }

  create({ url, token, fileName, saveDir, savePath, downloadId, completeChannel }) {
    const id = downloadId || `download-${Date.now()}-${++this.counter}`
    const meta = { url, token, fileName, saveDir, savePath, downloadId: id, completeChannel }
    this.byOriginalUrl.set(url, meta)
    // 直接用原始 URL，不加 nonce——avoid 重定向/编码导致匹配失败
    return { requestUrl: url, ...meta }
  }

  consume(requestUrl) {
    // 先精确匹配，失败则尝试只匹配 pathname（忽略 query/编码差异）
    let meta = this.byOriginalUrl.get(requestUrl)
    if (!meta) {
      const parsed = new URL(requestUrl)
      for (const [url, m] of this.byOriginalUrl.entries()) {
        const parsedM = new URL(url)
        if (parsed.pathname === parsedM.pathname) {
          meta = m
          break
        }
      }
    }
    if (meta) this.byOriginalUrl.delete(meta.url)
    return meta || null
  }

  getAuthHeader(requestUrl) {
    const meta = this.byOriginalUrl.get(requestUrl)
    return meta?.token ? `Bearer ${meta.token}` : null
  }
}

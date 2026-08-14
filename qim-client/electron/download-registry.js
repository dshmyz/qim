export class DownloadRegistry {
  constructor() {
    this.counter = 0
    // url -> { token, queue: meta[], active: number }
    // queue: 同 URL 待发生的下载实例 FIFO（每次 downloadURL 触发一个 will-download，consume 时 shift 一个）
    // active: 该 URL 尚未 done 的实例数。create +1、done 收尾 finalize -1；归零才清 token 记录。
    //   用 active 而非 queue.length 判清理——否则并发同 URL 下，先完成的实例会在后者仍在下载时误删 token。
    this.byUrl = new Map()
  }

  create({ url, token, fileName, saveDir, savePath, downloadId, completeChannel }) {
    const id = downloadId || `download-${Date.now()}-${++this.counter}`
    const entry = { url, token, fileName, saveDir, savePath, downloadId: id, completeChannel }
    let rec = this.byUrl.get(url)
    if (!rec) {
      rec = { token: '', queue: [], active: 0 }
      this.byUrl.set(url, rec)
    }
    if (token) rec.token = token
    rec.active += 1
    rec.queue.push(entry)
    // 直接用原始 URL，不加 nonce——avoid 重定向/编码导致匹配失败
    return { requestUrl: url, ...entry }
  }

  // 取该 URL 最早发起的那一次下载实例（FIFO），不删除 URL 记录，避免同 URL 后续实例丢事件。
  consume(requestUrl) {
    const rec = this.byUrl.get(requestUrl)
    // 该 URL 已有精确记录：即使队列此刻为空，也直接返回 null，绝不掉进下方向 pathname 兜底。
    // 否则一个"精确 URL 已被跟踪但队列临时空"的 will-download 会误取另一个同 pathname、
    // 不同 query 的并发下载的 entry（跨 URL 窃取元数据 / downloadId / token，还让被窃方丢事件）。
    if (rec) return rec.queue.length ? rec.queue.shift() : null
    // 该 URL 完全无记录时才按 pathname 兜底：用于处理 item.getURL() 与注册 key 有 query/编码差异的
    // 历史/兜底场景；不匹配任何记录则返回 null。
    let parsed
    try {
      parsed = new URL(requestUrl)
    } catch {
      return null
    }
    for (const [url, r] of this.byUrl.entries()) {
      if (!r.queue.length) continue
      let matched = false
      try {
        matched = parsed.pathname === new URL(url).pathname
      } catch {
        matched = false
      }
      if (matched) return r.queue.shift() || null
    }
    return null
  }

  // 按 URL 取已注册的 Authorization token。
  // 注：token 是"按 URL 一个槽位"（rec.token，最近一次 create 写入），而非按 entry 各自存——因为
  // onBeforeSendHeaders 只能看到 details.url，无法区分同 URL 下第几个 entry 的请求。聊天场景下
  // 同一文件（同 URL）由同一登录用户下载，token 必然一致，故此限制不构成实际问题；若未来需要
  // "同 URL 不同 token"，须给下载 URL 加持 nonce 区分（会改变 getAuthHeader/consume 的匹配键）。
  getAuthHeader(requestUrl) {
    const rec = this.byUrl.get(requestUrl)
    if (!rec || !rec.token) return null
    return `Bearer ${rec.token}`
  }

  // 下载实例收尾：该 URL 无任何未 done 实例时，清理 token 记录。
  // 注：active 只在成功 consume 并走到 done 时递减。若某 create 的下载永不触发 will-download/
  // done（如 URL 直接失败且 Electron 未派发 item），该 entry 会滞留（active>0、queue 非空），
  // 记录与 token 无法清理——此为已知边界（Electron 对多数失败下载仍会派发 will-download+done，
  // 故实际较少触发）；需超时空闲清理或 main.js 兜底 finalize 方能根治，未在此实现以免扩大改动。
  finalize(url) {
    const rec = this.byUrl.get(url)
    if (!rec) return
    rec.active -= 1
    if (rec.active <= 0 && rec.queue.length === 0) this.byUrl.delete(url)
  }
}
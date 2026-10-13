import { useDownloadStore } from '../stores/download'
import { getStoredServerUrl } from './useServerUrl'
import { getToken } from './useRequest'
import { downloadBlob } from '../utils/download'

/**
 * 全客户端唯一的文件下载入口。
 *
 * Electron：主进程 net.request 流式写盘，避免大文件整块读入渲染进程内存——
 * - downloadFile：静默下载到默认目录（'download-file' IPC）
 * - saveFileAs：弹出另存为对话框（'save-file-as' IPC）
 * 进度与终态经 'download-progress' / 'download-complete' / 'save-file-complete' 回传，
 * 统一注册一次并喂给全局下载管理器（useDownloadStore）。
 *
 * 浏览器回退：fetch + blob，仅开发环境使用。
 */
export interface DownloadableFile {
  /** 文件记录 ID。提供 url 时以 url 为准（群空间等非 owner 授权接口场景） */
  id?: number
  name: string
  size?: number
  /** 直接下载 URL（绝对地址，或以 / 开头的存储路径） */
  url?: string
  /** 主 URL 400/404（迁移数据 content.id 失效）时回退的原始 URL */
  fallbackUrl?: string
  /** 调用方跟踪键（如消息 id），进度/终态事件原样带回；不传则自动生成 */
  downloadId?: string
  /** Electron 静默下载的目标目录（不传用系统下载目录） */
  saveDir?: string
}

// 全局 IPC 监听只注册一次
let listenersRegistered = false

function ensureDownloadListeners() {
  if (listenersRegistered) return
  const electron = window.electron
  if (!electron?.ipcRenderer) return

  const downloadStore = useDownloadStore()

  electron.ipcRenderer.on('download-progress', (_event, data) => {
    if (!data?.downloadId) return
    downloadStore.updateProgress(
      data.downloadId,
      data.percent ?? 0,
      data.received ?? 0,
      data.total ?? 0
    )
  })

  // 两个终态通道共享同一套任务簿：download-file → download-complete、save-file-as → save-file-complete
  const onTerminal = (_event: unknown, data: { downloadId?: string; success?: boolean; cancelled?: boolean; error?: string; filePath?: string }) => {
    if (!data?.downloadId) return
    if (data.success) {
      downloadStore.markCompleted(data.downloadId, data.filePath || '')
    } else if (data.cancelled) {
      downloadStore.markFailed(data.downloadId, '已取消')
    } else {
      downloadStore.markFailed(data.downloadId, data.error || '下载失败')
    }
  }
  electron.ipcRenderer.on('download-complete', onTerminal)
  electron.ipcRenderer.on('save-file-complete', onTerminal)

  listenersRegistered = true
}

function absoluteUrl(url: string): string {
  if (url.startsWith('http')) return url
  return `${getStoredServerUrl().replace(/\/$/, '')}/${url.replace(/^\//, '')}`
}

function resolveDownloadUrl(file: DownloadableFile): string {
  if (file.url) return absoluteUrl(file.url)
  if (file.id && file.id > 0) return `${getStoredServerUrl().replace(/\/$/, '')}/api/v1/files/${file.id}/download`
  return ''
}

export function useFileDownload() {
  const downloadStore = useDownloadStore()

  // 任务登记：调用方指定 downloadId 时以其为键（供消息级进度/锁复用），否则自动生成
  function beginTask(file: DownloadableFile): { downloadId: string; url: string; fallback?: string } {
    const url = resolveDownloadUrl(file)
    if (!url) throw new Error('下载地址为空')
    const fallback = file.fallbackUrl && file.fallbackUrl !== url ? absoluteUrl(file.fallbackUrl) : undefined
    const downloadId = file.downloadId || `download_${Date.now()}_${Math.random().toString(36).slice(2, 9)}`
    downloadStore.addTask({ downloadId, fileName: file.name, size: file.size ?? 0 })
    return { downloadId, url, fallback }
  }

  async function fetchAndSave(url: string, filename: string) {
    const token = getToken()
    const response = await fetch(url, {
      method: 'GET',
      cache: 'no-store',
      headers: token ? { Authorization: `Bearer ${token}` } : {}
    })
    if (!response.ok) {
      const error = new Error(response.status === 403 ? '权限不足，请检查您的权限' : `服务器错误 (${response.status})`) as Error & { status: number }
      error.status = response.status
      throw error
    }
    downloadBlob(await response.blob(), filename)
  }

  async function browserDownload(file: DownloadableFile) {
    const url = resolveDownloadUrl(file)
    if (!url) throw new Error('下载地址为空')
    try {
      await fetchAndSave(url, file.name)
    } catch (error: any) {
      const fallback = file.fallbackUrl && file.fallbackUrl !== url ? absoluteUrl(file.fallbackUrl) : ''
      if (fallback && (error?.status === 400 || error?.status === 404)) {
        await fetchAndSave(fallback, file.name)
        return
      }
      throw error
    }
  }

  /** 静默下载到默认目录。Electron 返回 downloadId（后台进行，进度见下载管理器），浏览器返回 null（完成即返回） */
  async function downloadFile(file: DownloadableFile): Promise<string | null> {
    const electron = window.electron
    if (electron?.ipcRenderer) {
      ensureDownloadListeners()
      const { downloadId, url, fallback } = beginTask(file)
      electron.ipcRenderer.send('download-file', {
        url,
        token: getToken() || '',
        fileName: file.name,
        saveDir: file.saveDir,
        downloadId,
        fallbackUrl: fallback
      })
      return downloadId
    }
    await browserDownload(file)
    return null
  }

  /** 另存为对话框。用户取消返回 null */
  async function saveFileAs(file: DownloadableFile): Promise<string | null> {
    const electron = window.electron
    if (electron?.ipcRenderer) {
      ensureDownloadListeners()
      const result = await electron.ipcRenderer.invoke('show-save-dialog', { defaultPath: file.name })
      if (!result || result.canceled || !result.filePath) return null
      const { downloadId, url, fallback } = beginTask(file)
      electron.ipcRenderer.send('save-file-as', {
        url,
        token: getToken() || '',
        fileName: file.name,
        savePath: result.filePath,
        downloadId,
        fallbackUrl: fallback
      })
      return downloadId
    }
    await browserDownload(file)
    return null
  }

  return { downloadFile, saveFileAs }
}

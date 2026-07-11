import { useDownloadStore } from '../stores/download'
import { getStoredServerUrl } from './useServerUrl'
import { getToken } from './useRequest'
import { downloadUrl } from '../utils/download'

interface DownloadableFile {
  id: number
  name: string
  size: number
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

  electron.ipcRenderer.on('save-file-complete', (_event, data) => {
    if (!data?.downloadId) return
    if (data.success) {
      downloadStore.markCompleted(data.downloadId, data.filePath)
    } else if (data.cancelled) {
      downloadStore.markFailed(data.downloadId, '已取消')
    } else {
      downloadStore.markFailed(data.downloadId, data.error || '下载失败')
    }
  })

  listenersRegistered = true
}

export function useFileDownload() {
  const downloadStore = useDownloadStore()

  async function downloadFile(file: DownloadableFile) {
    const electron = window.electron
    // Electron 模式：弹出另存为对话框 + 走主进程下载（带进度）
    if (electron?.ipcRenderer) {
      ensureDownloadListeners()

      const result = await electron.ipcRenderer.invoke('show-save-dialog', {
        defaultPath: file.name
      })
      // 用户取消
      if (!result || result.canceled || !result.filePath) return

      const downloadId = downloadStore.addTask({ fileName: file.name, size: file.size })
      const url = `${getStoredServerUrl()}/api/v1/files/${file.id}/download`

      electron.ipcRenderer.send('save-file-as', {
        url,
        token: getToken(),
        fileName: file.name,
        savePath: result.filePath,
        downloadId
      })
      return
    }

    // 浏览器回退：无进度，直接 blob 下载
    const url = `${getStoredServerUrl()}/api/v1/files/${file.id}/download`
    try {
      await downloadUrl({ url, filename: file.name, token: getToken() })
    } catch (error) {
      console.error('文件下载失败:', error)
      throw error
    }
  }

  return { downloadFile }
}

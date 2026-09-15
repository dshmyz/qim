export type DownloadMode = 'electron' | 'browser'

export interface DownloadUrlOptions {
  url: string
  filename: string
  token?: string | null
  saveDir?: string
  downloadId?: string
  /** content.id 接口 400/404（迁移数据 id 失效）时回退的原始 url */
  fallbackUrl?: string
}

export interface DownloadResult {
  mode: DownloadMode
}

export function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

export async function downloadUrl(options: DownloadUrlOptions): Promise<DownloadResult> {
  const electron = window.electron
  if (electron?.ipcRenderer?.send) {
    electron.ipcRenderer.send('download-file', {
      url: options.url,
      token: options.token || '',
      fileName: options.filename,
      saveDir: options.saveDir,
      downloadId: options.downloadId,
      fallbackUrl: options.fallbackUrl,
    })
    return { mode: 'electron' }
  }

  const response = await fetch(options.url, {
    method: 'GET',
    cache: 'no-store',
    headers: {
      ...(options.token ? { Authorization: `Bearer ${options.token}` } : {}),
    },
  })

  if (!response.ok) {
    throw new Error(response.status === 403 ? '权限不足，请检查您的权限' : '服务器错误')
  }

  const blob = await response.blob()
  downloadBlob(blob, options.filename)
  return { mode: 'browser' }
}

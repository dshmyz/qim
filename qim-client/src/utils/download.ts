/**
 * 浏览器回退用的小工具：blob → 非常驻 <a> 触发保存。
 * 真正的下载统一入口在 composables/useFileDownload.ts；
 * Electron 下任何大文件都不应经过这里（blob 会把整个文件读进渲染进程内存）。
 */
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

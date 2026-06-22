export interface FileStatistics {
  totalSize: number
  usedSize: number
  fileCount: number
  sizeByType: Array<{
    type: string
    size: number
    count: number
  }>
}

export interface LargeFile {
  id: number
  fileName: string
  fileSize: number
  fileType: string
  uploaderId: number
  uploaderName: string
  createdAt: string
}

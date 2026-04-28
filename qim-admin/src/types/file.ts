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

export interface CleanupRule {
  id: number
  name: string
  type: 'time' | 'size' | 'type'
  value: string
  enabled: boolean
  createdAt: string
}

export interface FileAccessLog {
  id: number
  fileId: number
  fileName: string
  action: 'upload' | 'download' | 'delete'
  userId: number
  userName: string
  ipAddress: string
  createdAt: string
}

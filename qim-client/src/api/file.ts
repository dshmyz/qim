import { api } from './core'

export interface FileItem {
  id: number
  user_id: number
  name: string
  original_name: string
  size: number
  mime_type: string
  storage_path: string
  checksum: string
  folder_id: number | null
  source: string
  source_id: string | null
  is_starred: boolean
  starred_at: string | null
  tags: string | null
  created_at: string
  updated_at: string
}

export interface FolderItem {
  id: number
  user_id: number
  name: string
  parent_id: number | null
  sort_order: number
  icon: string | null
  color: string | null
  has_children?: boolean
  file_count?: number
  created_at: string
  updated_at: string
}

export interface FileListParams {
  folder_id?: number | null
  source?: string
  starred?: boolean
  type?: string
  search?: string
  sort_by?: string
  sort_order?: string
  date_from?: string
  date_to?: string
  page?: number
  page_size?: number
}

export interface FileListResponse {
  files: FileItem[]
  total: number
  page: number
  page_size: number
}

// 分片上传相关类型
export interface InitUploadRequest {
  filename: string
  file_size: number
  file_hash: string
  folder_id?: number | null
  mime_type: string
}

export interface InitUploadResponse {
  upload_id: string
  chunk_size: number
  total_chunks: number
  uploaded_chunks: number[]
  is_quick_upload: boolean
  file_id?: number
}

export interface UploadChunkResponse {
  chunk_index: number
  chunk_hash: string
}

export interface CompleteUploadRequest {
  upload_id: string
  file_hash: string
  total_chunks: number
}

export interface CancelUploadRequest {
  upload_id: string
}

// 文件相关 API
export const fileApi = {
  // 获取文件列表
  getFiles(params: FileListParams = {}) {
    return api.get<{ code: number; data: FileListResponse }>('/api/v1/files', { params })
  },

  // 上传文件
  uploadFile(file: File, folderId?: number) {
    const formData = new FormData()
    formData.append('file', file)
    if (folderId) formData.append('folder_id', String(folderId))
    return api.post<{ code: number; data: FileItem }>('/api/v1/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },

  // 下载文件
  downloadFile(fileId: number) {
    return api.get(`/api/v1/files/${fileId}/download`, { responseType: 'blob' })
  },

  // 删除文件
  deleteFile(fileId: number) {
    return api.delete<{ code: number }>(`/api/v1/files/${fileId}`)
  },

  // 更新文件
  updateFile(fileId: number, data: { name?: string; folder_id?: number | null; tags?: string[] }) {
    return api.put<{ code: number; data: FileItem }>(`/api/v1/files/${fileId}`, data)
  },

  // 星标/取消星标
  toggleStar(fileId: number, starred: boolean) {
    return api.put<{ code: number }>(`/api/v1/files/${fileId}/star`, { starred })
  },

  // 批量操作
  batchOperation(fileIds: number[], action: string, extra?: Record<string, any>) {
    return api.post<{ code: number }>('/api/v1/files/batch', {
      file_ids: fileIds,
      action,
      ...extra
    })
  },

  // 获取星标文件
  getStarredFiles(params: Omit<FileListParams, 'starred'> = {}) {
    return api.get<{ code: number; data: FileListResponse }>('/api/v1/files/starred', {
      params: { ...params, starred: true }
    })
  },

  // 获取文件统计
  getStats() {
    return api.get<{ code: number; data: Record<string, any> }>('/api/v1/files/stats')
  },

  // 初始化分片上传
  initUpload(data: InitUploadRequest) {
    return api.post<{ code: number; data: InitUploadResponse }>('/api/v1/files/upload/init', data)
  },

  // 上传分片
  uploadChunk(formData: FormData) {
    return api.post<{ code: number; data: UploadChunkResponse }>('/api/v1/files/upload/chunk', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },

  // 完成上传
  completeUpload(data: CompleteUploadRequest) {
    return api.post<{ code: number; data: FileItem }>('/api/v1/files/upload/complete', data)
  },

  // 取消上传
  cancelUpload(data: CancelUploadRequest) {
    return api.post<{ code: number }>('/api/v1/files/upload/cancel', data)
  }
}

// 文件夹相关 API
export const folderApi = {
  // 获取文件夹树（懒加载）
  getFolderTree(parentId: number | null = null) {
    return api.get<{ code: number; data: FolderItem[] }>('/api/v1/folders/tree', {
      params: { lazy: true, parent_id: parentId }
    })
  },

  // 创建文件夹
  createFolder(name: string, parentId?: number | null) {
    return api.post<{ code: number; data: FolderItem }>('/api/v1/folders', {
      name,
      parent_id: parentId ?? null
    })
  },

  // 更新文件夹
  updateFolder(folderId: number, data: { name?: string; parent_id?: number | null; icon?: string; color?: string; sort_order?: number }) {
    return api.put<{ code: number; data: FolderItem }>(`/api/v1/folders/${folderId}`, data)
  },

  // 删除文件夹
  deleteFolder(folderId: number) {
    return api.delete<{ code: number }>(`/api/v1/folders/${folderId}`)
  },

  // 获取文件夹内文件
  getFolderFiles(folderId: number, params: Omit<FileListParams, 'folder_id'> = {}) {
    return api.get<{ code: number; data: FileListResponse }>(`/api/v1/folders/${folderId}/files`, {
      params: { ...params, folder_id: folderId }
    })
  }
}

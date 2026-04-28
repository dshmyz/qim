import axios, {
  AxiosInstance,
  AxiosRequestConfig,
  AxiosResponse,
  InternalAxiosRequestConfig
} from 'axios'
import { API_BASE_URL } from '../config'

// ============================================================
// TypeScript Interfaces
// ============================================================

/**
 * 文件项接口
 */
export interface FileItem {
  id: number | string
  name: string
  original_name: string
  file_type: string
  file_size: number
  mime_type: string
  path: string
  url?: string
  folder_id: number | string | null
  parent_id?: number | string | null
  is_starred?: boolean
  created_at: string | number
  updated_at: string | number
  created_by?: string | number
  description?: string
  thumbnail?: string
}

/**
 * 文件夹项接口
 */
export interface FolderItem {
  id: number | string
  name: string
  parent_id: number | string | null
  path?: string
  color?: string
  icon?: string
  sort_order?: number
  file_count?: number
  created_at: string | number
  updated_at: string | number
  created_by?: string | number
  children?: FolderItem[]
}

/**
 * 文件列表查询参数
 */
export interface FileListParams {
  /** 页码，从 1 开始 */
  page?: number
  /** 每页条数 */
  pageSize?: number
  /** 文件夹 ID */
  folder_id?: number | string
  /** 搜索关键词 */
  keyword?: string
  /** 文件类型过滤 */
  file_type?: string
  /** 排序字段 */
  sort_by?: 'name' | 'size' | 'created_at' | 'updated_at'
  /** 排序方向 */
  sort_order?: 'asc' | 'desc'
  /** 是否只获取收藏的文件 */
  starred?: boolean
  /** 是否懒加载（用于树形结构） */
  lazy?: boolean
  /** 父文件夹 ID */
  parent_id?: number | string | null
}

/**
 * 文件列表响应
 */
export interface FileListResponse {
  list: FileItem[]
  total: number
  page: number
  pageSize: number
}

/**
 * 文件统计信息
 */
export interface FileStats {
  totalFiles: number
  totalFolders: number
  totalSize: number
  starredCount: number
  byType?: Record<string, number>
}

/**
 * 批量操作参数
 */
export interface BatchOperationParams {
  /** 文件/文件夹 ID 列表 */
  ids: (number | string)[]
  /** 操作类型：move | delete | star | unstar */
  operation: 'move' | 'delete' | 'star' | 'unstar'
  /** 目标文件夹 ID（移动操作时需要） */
  target_folder_id?: number | string
}

/**
 * 创建文件夹参数
 */
export interface CreateFolderParams {
  name: string
  parent_id?: number | string | null
  color?: string
  icon?: string
  sort_order?: number
}

/**
 * 更新文件夹参数
 */
export interface UpdateFolderParams {
  name?: string
  color?: string
  icon?: string
  sort_order?: number
}

/**
 * 更新文件参数
 */
export interface UpdateFileParams {
  name?: string
  folder_id?: number | string | null
  description?: string
}

/**
 * 通用 API 响应结构
 */
interface ApiResponse<T = unknown> {
  code: number
  data: T
  message?: string
}

// ============================================================
// Axios Instance Configuration
// ============================================================

/**
 * 创建 axios 实例
 */
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

/**
 * 请求拦截器：自动添加 Authorization token
 */
apiClient.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = localStorage.getItem('token')
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

/**
 * 响应拦截器：统一处理响应数据
 */
apiClient.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    return response
  },
  (error) => {
    if (error.response) {
      const { status, data } = error.response
      if (status === 401) {
        return Promise.reject(new Error('UNAUTHORIZED'))
      }
      if (status === 403) {
        return Promise.reject(
          new Error(data?.message || '权限不足，请检查您的权限')
        )
      }
      return Promise.reject(
        new Error(data?.message || `请求失败 (${status})`)
      )
    }
    if (error.code === 'ECONNABORTED') {
      return Promise.reject(new Error('请求超时，请稍后重试'))
    }
    return Promise.reject(error)
  }
)

/**
 * 辅助函数：提取响应中的 data 字段
 */
function extractResponseData<T>(response: AxiosResponse<ApiResponse<T>>): T {
  return response.data.data
}

// ============================================================
// File API
// ============================================================

export const fileApi = {
  /**
   * 获取文件列表（支持分页和筛选）
   */
  getFiles: async (params?: FileListParams): Promise<FileListResponse> => {
    const response = await apiClient.get<ApiResponse<FileListResponse>>(
      '/api/v1/files',
      { params }
    )
    return extractResponseData(response)
  },

  /**
   * 上传文件
   */
  uploadFile: async (
    file: File,
    folderId?: number | string,
    onProgress?: (progressEvent: unknown) => void
  ): Promise<FileItem> => {
    const formData = new FormData()
    formData.append('file', file)
    if (folderId !== undefined && folderId !== null) {
      formData.append('folder_id', String(folderId))
    }

    const config: AxiosRequestConfig = {
      headers: {
        'Content-Type': 'multipart/form-data'
      },
      onUploadProgress: onProgress
    }

    const response = await apiClient.post<ApiResponse<FileItem>>(
      '/api/v1/upload',
      formData,
      config
    )
    return extractResponseData(response)
  },

  /**
   * 下载文件
   */
  downloadFile: async (fileId: number | string): Promise<Blob> => {
    const response = await apiClient.get(`/api/v1/files/${fileId}/download`, {
      responseType: 'blob'
    })
    return response.data as Blob
  },

  /**
   * 删除文件
   */
  deleteFile: async (fileId: number | string): Promise<void> => {
    await apiClient.delete(`/api/v1/files/${fileId}`)
  },

  /**
   * 更新文件信息
   */
  updateFile: async (
    fileId: number | string,
    data: UpdateFileParams
  ): Promise<FileItem> => {
    const response = await apiClient.put<ApiResponse<FileItem>>(
      `/api/v1/files/${fileId}`,
      data
    )
    return extractResponseData(response)
  },

  /**
   * 收藏/取消收藏文件
   */
  toggleStar: async (fileId: number | string): Promise<FileItem> => {
    const response = await apiClient.post<ApiResponse<FileItem>>(
      `/api/v1/files/${fileId}/star`
    )
    return extractResponseData(response)
  },

  /**
   * 批量操作（移动、删除、收藏等）
   */
  batchOperation: async (
    params: BatchOperationParams
  ): Promise<{ successCount: number }> => {
    const response = await apiClient.post<
      ApiResponse<{ successCount: number }>
    >('/api/v1/files/batch', params)
    return extractResponseData(response)
  },

  /**
   * 获取收藏的文件列表
   */
  getStarredFiles: async (params?: Omit<FileListParams, 'starred'>): Promise<FileListResponse> => {
    const response = await apiClient.get<ApiResponse<FileListResponse>>(
      '/api/v1/files/starred',
      { params: { ...params, starred: true } }
    )
    return extractResponseData(response)
  },

  /**
   * 获取文件统计信息
   */
  getStats: async (): Promise<FileStats> => {
    const response = await apiClient.get<ApiResponse<FileStats>>(
      '/api/v1/files/stats'
    )
    return extractResponseData(response)
  }
}

// ============================================================
// Folder API
// ============================================================

export const folderApi = {
  /**
   * 获取文件夹树
   */
  getFolderTree: async (params?: {
    lazy?: boolean
    parent_id?: number | string | null
  }): Promise<FolderItem[]> => {
    const response = await apiClient.get<ApiResponse<FolderItem[]>>(
      '/api/v1/folders/tree',
      { params }
    )
    return extractResponseData(response)
  },

  /**
   * 创建文件夹
   */
  createFolder: async (data: CreateFolderParams): Promise<FolderItem> => {
    const response = await apiClient.post<ApiResponse<FolderItem>>(
      '/api/v1/folders',
      data
    )
    return extractResponseData(response)
  },

  /**
   * 更新文件夹
   */
  updateFolder: async (
    folderId: number | string,
    data: UpdateFolderParams
  ): Promise<FolderItem> => {
    const response = await apiClient.put<ApiResponse<FolderItem>>(
      `/api/v1/folders/${folderId}`,
      data
    )
    return extractResponseData(response)
  },

  /**
   * 删除文件夹
   */
  deleteFolder: async (folderId: number | string): Promise<void> => {
    await apiClient.delete(`/api/v1/folders/${folderId}`)
  },

  /**
   * 获取文件夹下的文件列表
   */
  getFolderFiles: async (
    folderId: number | string,
    params?: Omit<FileListParams, 'folder_id'>
  ): Promise<FileListResponse> => {
    const response = await apiClient.get<ApiResponse<FileListResponse>>(
      `/api/v1/folders/${folderId}/files`,
      { params: { ...params, folder_id: folderId } }
    )
    return extractResponseData(response)
  }
}

// ============================================================
// Export for easy import
// ============================================================

export default {
  file: fileApi,
  folder: folderApi
}

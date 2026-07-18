import { api } from './core'

export interface GroupFile {
  id: number
  user_id: number
  name: string
  original_name: string
  size: number
  mime_type: string
  storage_path: string
  folder_id: number | null
  source: string
  source_id: string | null
  created_at: string
  updated_at: string
  uploader?: { id: number; name: string }
}

export interface GroupFolder {
  id: number
  name: string
  parent_id: number | null
  created_at: string
}

export interface GroupFileFilters {
  page?: number
  page_size?: number
  folder_id?: number | null
  search?: string
}

export interface GroupFileList {
  files: GroupFile[]
  folders: GroupFolder[]
  total: number
  page: number
  page_size: number
}

export const groupFiles = {
  list(groupId: string | number, filters: GroupFileFilters = {}) {
    return api.get<{ code: number; data: GroupFileList }>(`/api/v1/groups/${groupId}/files`, {
      params: filters,
    })
  },

  createFolder(groupId: string | number, name: string, parentId: number | null = null) {
    return api.post<{ code: number; data: GroupFolder }>(`/api/v1/groups/${groupId}/folders`, {
      name,
      parent_id: parentId,
    })
  },

  attach(groupId: string | number, fileId: number, folderId: number | null = null) {
    return api.post<{ code: number; data: GroupFile }>(`/api/v1/groups/${groupId}/files`, {
      file_id: fileId,
      folder_id: folderId,
    })
  },

  download(groupId: string | number, fileId: number) {
    return api.get(`/api/v1/groups/${groupId}/files/${fileId}/download`, { responseType: 'blob' })
  },

  move(groupId: string | number, fileId: number, folderId: number | null) {
    return api.patch<{ code: number }>(`/api/v1/groups/${groupId}/files/${fileId}`, {
      folder_id: folderId,
    })
  },

  remove(groupId: string | number, fileId: number) {
    return api.delete<{ code: number }>(`/api/v1/groups/${groupId}/files/${fileId}`)
  },

  shareReference(groupId: string | number, fileId: number, folderId: number | null = null) {
    return api.post<{ code: number; data: GroupFile }>(`/api/v1/groups/${groupId}/files/references`, {
      file_id: fileId,
      folder_id: folderId,
    })
  },
}

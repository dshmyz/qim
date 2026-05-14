import { ref } from 'vue'
import { useRequest, request } from './useRequest'
import { API_BASE_URL } from '../config'
import type { Note, AIAnalyzeResult } from '../types/note'

export function useNotes() {
  const { get, post, put, delete: del } = useRequest()
  const loading = ref(false)
  const error = ref<string | null>(null)

  const fetchNotes = async (): Promise<Note[]> => {
    loading.value = true
    error.value = null
    try {
      const response = await get<any>('/api/v1/notes')
      const notes = response?.data || []
      return notes.map((n: any) => ({
        ...n,
        tags: parseTags(n.tags)
      }))
    } catch (e: any) {
      error.value = e.message
      return []
    } finally {
      loading.value = false
    }
  }

  const createNote = async (data: Partial<Note>): Promise<Note | null> => {
    loading.value = true
    error.value = null
    try {
      const response = await post<any>('/api/v1/notes', {
        title: data.title || '新笔记',
        content: data.content || '',
        type: data.type || 'note',
        tags: JSON.stringify(data.tags || [])
      })
      if (!response) return null
      return { ...response.data, tags: parseTags(response.data.tags) }
    } catch (e: any) {
      error.value = e.message
      return null
    } finally {
      loading.value = false
    }
  }

  const updateNote = async (id: number, data: Partial<Note>): Promise<boolean> => {
    loading.value = true
    error.value = null
    try {
      const response = await put<any>(`/api/v1/notes/${id}`, {
        title: data.title,
        content: data.content,
        tags: JSON.stringify(data.tags || [])
      })
      return response !== null
    } catch (e: any) {
      error.value = e.message
      return false
    } finally {
      loading.value = false
    }
  }

  const deleteNote = async (id: number): Promise<boolean> => {
    loading.value = true
    error.value = null
    try {
      const response = await del<any>(`/api/v1/notes/${id}`)
      return response !== null
    } catch (e: any) {
      error.value = e.message
      return false
    } finally {
      loading.value = false
    }
  }

  const analyzeNote = async (id: number): Promise<AIAnalyzeResult | null> => {
    loading.value = true
    error.value = null
    try {
      const response = await post<any>(`/api/v1/notes/${id}/analyze`, {})
      return response?.data || null
    } catch (e: any) {
      error.value = e.message
      return null
    } finally {
      loading.value = false
    }
  }

  const updateNoteTags = async (id: number, tags: string[]): Promise<boolean> => {
    loading.value = true
    error.value = null
    try {
      await request(`/api/v1/notes/${id}/tags`, {
        method: 'PATCH',
        body: JSON.stringify({ tags })
      })
      return true
    } catch (e: any) {
      error.value = e.message
      return false
    } finally {
      loading.value = false
    }
  }

  const updateNoteSummary = async (id: number, summary: string): Promise<boolean> => {
    loading.value = true
    error.value = null
    try {
      await request(`/api/v1/notes/${id}/summary`, {
        method: 'PATCH',
        body: JSON.stringify({ summary })
      })
      return true
    } catch (e: any) {
      error.value = e.message
      return false
    } finally {
      loading.value = false
    }
  }

  const exportNote = async (id: number, title: string) => {
    const token = localStorage.getItem('token')
    const baseUrl = localStorage.getItem('serverUrl') || API_BASE_URL
    const url = `${baseUrl}/api/v1/notes/${id}/export`

    try {
      const response = await fetch(url, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })

      if (!response.ok) {
        throw new Error('导出失败')
      }

      const blob = await response.blob()
      const downloadUrl = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = downloadUrl
      link.download = `${title}.md`
      link.click()
      URL.revokeObjectURL(downloadUrl)
    } catch (e) {
      console.error('导出失败:', e)
    }
  }

  const searchNotesSemantic = async (query: string, topK: number = 5): Promise<NoteVectorSearchResult[]> => {
    loading.value = true
    error.value = null
    try {
      const response = await post<any>('/api/v1/notes/search', { query, top_k: topK })
      return response?.data?.results || []
    } catch (e: any) {
      error.value = e.message
      return []
    } finally {
      loading.value = false
    }
  }

  return {
    loading,
    error,
    fetchNotes,
    createNote,
    updateNote,
    deleteNote,
    analyzeNote,
    updateNoteTags,
    updateNoteSummary,
    exportNote,
    searchNotesSemantic
  }
}

function parseTags(tags: any): string[] {
  if (!tags) return []
  if (Array.isArray(tags)) return tags
  if (typeof tags === 'string') {
    try {
      const parsed = JSON.parse(tags)
      return Array.isArray(parsed) ? parsed : []
    } catch {
      return []
    }
  }
  return []
}

export interface Note {
  id: number
  user_id: number
  title: string
  content: string
  type: 'note' | 'sticky'
  style: string
  tags: string[]
  summary: string
  ai_accessible?: boolean
  created_at: string
  updated_at: string
}

export interface AIAnalyzeResult {
  summary: string
  tags: string[]
  action_items: string[]
}

/** AI 格式化接口 /notes/:id/format 的返回 */
export interface NoteFormatResult {
  content: string
  truncated: boolean
}

export interface NoteVectorSearchResult {
  content: string
  score: number
  title: string
  note_id: string
}

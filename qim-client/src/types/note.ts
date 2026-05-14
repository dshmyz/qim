export interface Note {
  id: number
  user_id: number
  title: string
  content: string
  type: 'note' | 'sticky'
  style: string
  tags: string[]
  summary: string
  created_at: string
  updated_at: string
}

export interface AIAnalyzeResult {
  summary: string
  tags: string[]
  action_items: string[]
}

export interface NoteVectorSearchResult {
  content: string
  score: number
  title: string
  note_id: string
}

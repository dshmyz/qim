const DRAFT_KEY_PREFIX = 'qim_note_draft_'

interface DraftPayload {
  title: string
  content: string
  tags: string[]
  /** 草稿落盘时间（epoch ms），save 时自动写入。用于与服务器 updated_at 对比，
   *  判断本地草稿是否比服务器内容更新（防止旧草稿覆盖新内容） */
  savedAt?: number
}

/**
 * 笔记草稿 localStorage 缓存。
 *
 * 每次击键即时落盘（<1ms），服务器同步成功后清理。
 * 断网、崩溃、意外关闭时可从 localStorage 恢复未同步内容。
 */
export function useNoteDraft() {
  function save(noteId: number, payload: DraftPayload): void {
    try {
      localStorage.setItem(DRAFT_KEY_PREFIX + noteId, JSON.stringify({ ...payload, savedAt: Date.now() }))
    } catch {
      // quota exceeded — 静默降级，不影响编辑体验
    }
  }

  function load(noteId: number): DraftPayload | null {
    try {
      const raw = localStorage.getItem(DRAFT_KEY_PREFIX + noteId)
      if (!raw) return null
      return JSON.parse(raw) as DraftPayload
    } catch {
      return null
    }
  }

  function clear(noteId: number): void {
    try {
      localStorage.removeItem(DRAFT_KEY_PREFIX + noteId)
    } catch {
      // ignore
    }
  }

  return { save, load, clear }
}

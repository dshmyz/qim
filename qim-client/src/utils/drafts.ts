export const DRAFT_CHANGED_EVENT = 'qim-draft-changed'

export interface DraftChangedDetail {
  conversationId: string
}

export function saveDraft(conversationId: string, text: string, quoted: unknown): void {
  const key = `qim_draft_${conversationId}`

  if (!text.trim()) {
    localStorage.removeItem(key)
  } else {
    localStorage.setItem(key, JSON.stringify({ text, quoted }))
  }

  window.dispatchEvent(new CustomEvent<DraftChangedDetail>(DRAFT_CHANGED_EVENT, {
    detail: { conversationId },
  }))
}

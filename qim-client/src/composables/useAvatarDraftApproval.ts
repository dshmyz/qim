import { ref } from 'vue'
import { avatarAPI } from '../api/avatar'
import type { AvatarReplyDraft } from '../types/avatar'

// 分身质量核验被拒后的待审批草稿。模块级单例：MessageListView 渲染审批卡片、ChatWindow
// 响应「编辑后发送」、Main.vue 响应 WS 推送，三处共用同一份状态，不靠逐层事件透传。
const draftsByConv = ref<Map<string, AvatarReplyDraft[]>>(new Map())
const editRequest = ref<AvatarReplyDraft | null>(null)
const loadingConv = new Set<string>()

// editCapture 编辑后发送的待捕获信号：草稿回填输入框时记录原内容，
// 用户改写并发出时把 original→final 的 diff 发后端留学习样本。
interface EditCapture {
  convId: number
  draftId: number
  originalContent: string
  rejectReason: string
}
const editCapture = ref<EditCapture | null>(null)

export function useAvatarDraftApproval() {
  function draftsFor(convId: string | number): AvatarReplyDraft[] {
    return draftsByConv.value.get(String(convId)) || []
  }

  // 打开会话时拉取：卡片在会话内常驻，不依赖在线推送
  async function loadForConversation(convId: string | number) {
    const key = String(convId)
    if (!key || loadingConv.has(key)) return
    loadingConv.add(key)
    try {
      const list = await avatarAPI.listReplyDrafts(key)
      draftsByConv.value.set(key, list)
    } finally {
      loadingConv.delete(key)
    }
  }

  function removeDraft(convId: string | number, draftId: number) {
    const key = String(convId)
    const list = draftsByConv.value.get(key)
    if (!list) return
    draftsByConv.value.set(key, list.filter(d => d.id !== draftId))
  }

  async function approve(convId: string | number, draftId: number) {
    await avatarAPI.approveReplyDraft(draftId)
    removeDraft(convId, draftId)
  }

  async function discard(convId: string | number, draftId: number) {
    await avatarAPI.discardReplyDraft(draftId)
    removeDraft(convId, draftId)
  }

  // 编辑接管：ChatWindow 把草稿填入输入框后，草稿本身即被丢弃
  function requestEdit(draft: AvatarReplyDraft) {
    editRequest.value = draft
  }
  function clearEditRequest() {
    editRequest.value = null
  }

  // 记录待捕获的编辑信号（回填输入框时调用）
  function beginEditCapture(draft: AvatarReplyDraft) {
    editCapture.value = {
      convId: draft.conversation_id,
      draftId: draft.id,
      originalContent: draft.content,
      rejectReason: draft.reason || ''
    }
  }

  // 用户改写后发出：发送 diff 学习信号。信号失败不阻断发送，不重试不打扰。
  function captureEditedSend(convId: number, finalContent: string) {
    const capture = editCapture.value
    if (!capture || capture.convId !== convId) return
    editCapture.value = null
    if (!finalContent.trim()) return
    avatarAPI.recordEditSignal({
      conversation_id: capture.convId,
      draft_id: capture.draftId,
      original_content: capture.originalContent,
      final_content: finalContent,
      reject_reason: capture.rejectReason
    }).catch(() => {})
  }

  // 放弃捕获（输入被清空/会话切换，未实际改写发送）
  function cancelEditCapture(convId?: number) {
    if (convId === undefined || editCapture.value?.convId === convId) {
      editCapture.value = null
    }
  }

  return {
    draftsFor,
    loadForConversation,
    approve,
    discard,
    requestEdit,
    clearEditRequest,
    editRequest,
    beginEditCapture,
    captureEditedSend,
    cancelEditCapture,
  }
}

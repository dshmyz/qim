import { describe, expect, it, vi } from 'vitest'
import { requestMessageAttention } from '../../../src/utils/messageAttentionOrchestrator'

describe('requestMessageAttention', () => {
  it('does not alert for the selected conversation while the window is active', async () => {
    const alert = vi.fn()

    requestMessageAttention({
      isCurrentConversation: true,
      isStreaming: false,
      getIsWindowActive: async () => true,
      onAttention: alert,
    })
    await Promise.resolve()

    expect(alert).not.toHaveBeenCalled()
  })

  it('alerts for the selected conversation while the window is inactive', async () => {
    const alert = vi.fn()

    requestMessageAttention({
      isCurrentConversation: true,
      isStreaming: false,
      getIsWindowActive: async () => false,
      onAttention: alert,
    })
    await Promise.resolve()

    expect(alert).toHaveBeenCalledOnce()
  })

  it('treats a rejected window-state query as inactive', async () => {
    const alert = vi.fn()
    const error = new Error('IPC failed')
    const reportError = vi.fn()

    requestMessageAttention({
      isCurrentConversation: true,
      isStreaming: false,
      getIsWindowActive: async () => { throw error },
      onAttention: alert,
      onWindowStateError: reportError,
    })
    await Promise.resolve()
    await Promise.resolve()

    expect(reportError).toHaveBeenCalledWith(error)
    expect(alert).toHaveBeenCalledOnce()
  })

  it('alerts for another conversation without querying window state', () => {
    const getIsWindowActive = vi.fn(async () => true)
    const alert = vi.fn()

    requestMessageAttention({
      isCurrentConversation: false,
      isStreaming: false,
      getIsWindowActive,
      onAttention: alert,
    })

    expect(getIsWindowActive).not.toHaveBeenCalled()
    expect(alert).toHaveBeenCalledOnce()
  })

  it('does not query or alert for streaming messages', () => {
    const getIsWindowActive = vi.fn(async () => false)
    const alert = vi.fn()

    requestMessageAttention({
      isCurrentConversation: true,
      isStreaming: true,
      getIsWindowActive,
      onAttention: alert,
    })

    expect(getIsWindowActive).not.toHaveBeenCalled()
    expect(alert).not.toHaveBeenCalled()
  })

  it('does not block core message storage on a pending window-state query', () => {
    const getIsWindowActive = vi.fn(() => new Promise<boolean>(() => {}))
    const receiveMessage = vi.fn()

    requestMessageAttention({
      isCurrentConversation: true,
      isStreaming: false,
      getIsWindowActive,
      onAttention: vi.fn(),
    })
    receiveMessage()

    expect(getIsWindowActive).toHaveBeenCalledOnce()
    expect(receiveMessage).toHaveBeenCalledOnce()
  })
})

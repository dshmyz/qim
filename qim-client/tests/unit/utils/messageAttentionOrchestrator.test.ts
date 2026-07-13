import { describe, expect, it, vi } from 'vitest'
import {
  completeCoreMessageThenRequestAttention,
  requestMessageAttention,
} from '../../../src/utils/messageAttentionOrchestrator'

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

  it('completes core message handling before starting non-blocking attention', () => {
    const getIsWindowActive = vi.fn(() => new Promise<boolean>(() => {}))
    const calls: string[] = []

    const result = completeCoreMessageThenRequestAttention({
      completeCoreMessage: () => {
        calls.push('core')
      },
      requestAttention: () => {
        calls.push('attention')
        requestMessageAttention({
          isCurrentConversation: true,
          isStreaming: false,
          getIsWindowActive,
          onAttention: vi.fn(),
        })
      },
    })

    expect(result).toBeUndefined()
    expect(calls).toEqual(['core', 'attention'])
    expect(getIsWindowActive).toHaveBeenCalledOnce()
  })
})

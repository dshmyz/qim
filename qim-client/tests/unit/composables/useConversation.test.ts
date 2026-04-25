import { describe, it, expect, beforeEach } from 'vitest'
import { useConversation } from '@/composables/useConversation'
import type { Conversation, Message } from '@/types'

describe('useConversation', () => {
  let conversation: ReturnType<typeof useConversation>

  beforeEach(() => {
    conversation = useConversation()
  })

  it('应该初始化为空会话列表', () => {
    expect(conversation.conversations.value).toEqual([])
    expect(conversation.currentConversationId.value).toBeNull()
    expect(conversation.selectedConversation.value).toBeNull()
  })

  it('应该能选择会话', () => {
    const mockConv: Partial<Conversation> = { id: 'conv1', name: 'Test' }
    conversation.selectConversation(mockConv as Conversation)
    
    expect(conversation.currentConversationId.value).toBe('conv1')
    expect(conversation.selectedConversation.value).toEqual(mockConv)
  })

  it('应该能更新已存在的会话', () => {
    conversation.conversations.value = [
      { id: 'conv1', name: 'Old' } as Conversation
    ]

    conversation.updateConversation({ id: 'conv1', name: 'New' } as Conversation)

    expect(conversation.conversations.value[0].name).toBe('New')
    expect(conversation.conversations.value).toHaveLength(1)
  })

  it('应该能添加新会话到列表顶部', () => {
    conversation.conversations.value = [
      { id: 'conv1' } as Conversation
    ]

    conversation.updateConversation({ id: 'conv2' } as Conversation)

    expect(conversation.conversations.value).toHaveLength(2)
    expect(conversation.conversations.value[0].id).toBe('conv2')
  })

  it('应该能添加消息', () => {
    const mockMsg: Partial<Message> = { id: 'msg1', content: 'Hello' }
    conversation.addMessage(mockMsg as Message)
    
    expect(conversation.messages.value).toHaveLength(1)
    expect(conversation.messages.value[0]).toEqual(mockMsg)
  })

  it('应该能清除消息', () => {
    conversation.messages.value = [
      { id: 'msg1' } as Message,
      { id: 'msg2' } as Message
    ]
    
    conversation.clearMessages()
    
    expect(conversation.messages.value).toEqual([])
  })
})

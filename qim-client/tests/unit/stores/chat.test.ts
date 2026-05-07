import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useChatStore } from '@/stores/chat'
import type { Conversation, Message } from '@/types'

const localStorageStore: Record<string, string> = {}

vi.stubGlobal('localStorage', {
  getItem: (key: string) => localStorageStore[key] || null,
  setItem: (key: string, value: string) => { localStorageStore[key] = value },
  removeItem: (key: string) => { delete localStorageStore[key] },
  clear: () => { Object.keys(localStorageStore).forEach(k => delete localStorageStore[k]) },
  get length() { return Object.keys(localStorageStore).length },
  key: (index: number) => Object.keys(localStorageStore)[index] || null,
})

describe('useChatStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  describe('基础状态', () => {
    it('应该初始化为空状态', () => {
      const store = useChatStore()
      
      expect(store.conversations).toEqual([])
      expect(store.currentConversationId).toBeNull()
      expect(store.currentConversation).toBeNull()
      expect(store.currentMessages).toEqual([])
    })
  })

  describe('会话管理', () => {
    it('应该能设置会话列表', () => {
      const store = useChatStore()
      const mockConvs: Conversation[] = [
        { id: '1', name: '会话1', type: 'private' } as Conversation,
        { id: '2', name: '会话2', type: 'group' } as Conversation
      ]
      
      store.setConversations(mockConvs)
      
      expect(store.conversations).toEqual(mockConvs)
    })

    it('应该能更新已存在的会话', () => {
      const store = useChatStore()
      store.setConversations([
        { id: '1', name: '旧名称', type: 'private' } as Conversation
      ])
      
      store.updateConversation({ id: '1', name: '新名称', type: 'private' } as Conversation)
      
      expect(store.conversations[0].name).toBe('新名称')
      expect(store.conversations).toHaveLength(1)
    })

    it('应该能设置当前会话', () => {
      const store = useChatStore()
      store.setConversations([
        { id: '1', name: '会话1', type: 'private' } as Conversation
      ])
      
      store.setCurrentConversation('1')
      
      expect(store.currentConversationId).toBe('1')
      expect(store.currentConversation?.name).toBe('会话1')
    })

    it('设置不存在的会话ID时，currentConversation 应该返回 null', () => {
      const store = useChatStore()
      store.setConversations([
        { id: '1', name: '会话1', type: 'private' } as Conversation
      ])
      
      store.setCurrentConversation('999')
      
      expect(store.currentConversationId).toBe('999')
      expect(store.currentConversation).toBeNull()
    })
  })

  describe('消息管理', () => {
    it('应该能为会话设置消息', () => {
      const store = useChatStore()
      const mockMsgs: Message[] = [
        { id: 'm1', content: '消息1' } as Message,
        { id: 'm2', content: '消息2' } as Message
      ]
      
      store.setMessages('conv1', mockMsgs)
      
      expect(store.messages.get('conv1')).toEqual(mockMsgs)
    })

    it('应该能追加消息', () => {
      const store = useChatStore()
      store.setMessages('conv1', [{ id: 'm1', content: '消息1' } as Message])
      
      store.appendMessage('conv1', { id: 'm2', content: '消息2' } as Message)
      
      const msgs = store.messages.get('conv1')
      expect(msgs).toHaveLength(2)
      expect(msgs?.[1].content).toBe('消息2')
    })

    it('应该能在头部插入消息（加载历史）', () => {
      const store = useChatStore()
      store.setMessages('conv1', [{ id: 'm2', content: '消息2' } as Message])
      
      store.prependMessages('conv1', [{ id: 'm1', content: '消息1' } as Message])
      
      const msgs = store.messages.get('conv1')
      expect(msgs).toHaveLength(2)
      expect(msgs?.[0].content).toBe('消息1')
    })

    it('应该能更新消息', () => {
      const store = useChatStore()
      store.setMessages('conv1', [
        { id: 'm1', content: '原始内容', isRecalled: false } as Message
      ])
      
      store.updateMessage('conv1', 'm1', { content: '[消息已撤回]', isRecalled: true })
      
      const msgs = store.messages.get('conv1')
      expect(msgs?.[0].content).toBe('[消息已撤回]')
      expect(msgs?.[0].isRecalled).toBe(true)
    })

    it('currentMessages 应该返回当前会话的消息', () => {
      const store = useChatStore()
      store.setConversations([{ id: '1', name: '会话1', type: 'private' } as Conversation])
      store.setCurrentConversation('1')
      store.setMessages('1', [{ id: 'm1', content: '消息1' } as Message])
      
      expect(store.currentMessages).toHaveLength(1)
      expect(store.currentMessages[0].content).toBe('消息1')
    })
  })

  describe('草稿管理', () => {
    it('应该能保存和获取草稿', () => {
      const store = useChatStore()
      
      store.setDraft('conv1', '正在输入...')
      
      expect(store.getDraft('conv1')).toBe('正在输入...')
    })

    it('应该能清除草稿', () => {
      const store = useChatStore()
      store.setDraft('conv1', '正在输入...')
      
      store.clearDraft('conv1')
      
      expect(store.getDraft('conv1')).toBe('')
    })
  })

  describe('分页状态', () => {
    it('应该能设置和获取分页状态', () => {
      const store = useChatStore()
      
      store.setPage('conv1', 2)
      store.setHasMore('conv1', false)
      
      expect(store.getPage('conv1')).toBe(2)
      expect(store.getHasMore('conv1')).toBe(false)
    })

    it('未设置时应该返回默认值', () => {
      const store = useChatStore()
      
      expect(store.getPage('unknown')).toBe(1)
      expect(store.getHasMore('unknown')).toBe(true)
    })
  })

  describe('已读用户', () => {
    it('应该能设置已读用户信息', () => {
      const store = useChatStore()
      const readInfo = {
        read_users: [{ id: '1', name: '用户1' } as any],
        total_members: 5
      }
      
      store.setReadUsersMap('conv1', readInfo)
      
      const result = store.getReadUsersMap()
      expect(result.get('conv1')).toEqual(readInfo)
    })
  })

  describe('业务逻辑方法', () => {
    describe('pinConversation', () => {
      it('应该能置顶会话', () => {
        const store = useChatStore()
        store.setConversations([
          { id: '1', name: '会话1', pinned: false } as Conversation
        ])
        
        store.pinConversation('1', true)
        
        expect(store.conversations[0].pinned).toBe(true)
        expect(store.conversations[0].pinnedAt).toBeDefined()
      })

      it('应该能取消置顶会话', () => {
        const store = useChatStore()
        store.setConversations([
          { id: '1', name: '会话1', pinned: true, pinnedAt: 123456 } as Conversation
        ])
        
        store.pinConversation('1', false)
        
        expect(store.conversations[0].pinned).toBe(false)
        expect(store.conversations[0].pinnedAt).toBeUndefined()
      })

      it('置顶后应该保存到 localStorage', () => {
        const store = useChatStore()
        store.setConversations([
          { id: '1', name: '会话1', pinned: false } as Conversation
        ])
        
        store.pinConversation('1', true)
        
        const stored = JSON.parse(localStorage.getItem('qim_conversations') || '[]')
        expect(stored.length).toBe(1)
        expect(stored[0]?.pinned).toBe(true)
      })
    })

    describe('muteConversation', () => {
      it('应该能设置免打扰', () => {
        const store = useChatStore()
        store.setConversations([
          { id: '1', name: '会话1', muted: false } as Conversation
        ])
        
        store.muteConversation('1', true)
        
        expect(store.conversations[0].muted).toBe(true)
      })

      it('应该能取消免打扰', () => {
        const store = useChatStore()
        store.setConversations([
          { id: '1', name: '会话1', muted: true } as Conversation
        ])
        
        store.muteConversation('1', false)
        
        expect(store.conversations[0].muted).toBe(false)
      })
    })

    describe('removeConversation', () => {
      it('应该能移除会话', () => {
        const store = useChatStore()
        store.setConversations([
          { id: '1', name: '会话1' } as Conversation,
          { id: '2', name: '会话2' } as Conversation
        ])
        store.setMessages('1', [{ id: 'm1' } as Message])
        store.setDraft('1', '草稿')
        
        store.removeConversation('1')
        
        expect(store.conversations).toHaveLength(1)
        expect(store.conversations[0].id).toBe('2')
        expect(store.messages.get('1')).toBeUndefined()
        expect(store.getDraft('1')).toBe('')
      })
    })

    describe('recallMessage', () => {
      it('应该能撤回消息', () => {
        const store = useChatStore()
        store.setConversations([
          { id: '1', name: '会话1' } as Conversation
        ])
        store.setMessages('1', [
          { id: 'm1', content: '原始内容', isRecalled: false } as Message
        ])
        
        store.recallMessage('1', 'm1')
        
        const msgs = store.messages.get('1')
        expect(msgs?.[0].content).toBe('[消息已撤回]')
        expect(msgs?.[0].isRecalled).toBe(true)
      })

      it('撤回最后一条消息时应该更新会话的 lastMessage', () => {
        const store = useChatStore()
        store.setConversations([
          { 
            id: '1', 
            name: '会话1',
            lastMessage: { id: 'm1', content: '原始内容' } as Message
          } as Conversation
        ])
        store.setMessages('1', [
          { id: 'm1', content: '原始内容' } as Message
        ])
        
        store.recallMessage('1', 'm1')
        
        expect(store.conversations[0].lastMessage?.content).toBe('[消息已撤回]')
        expect(store.conversations[0].lastMessage?.isRecalled).toBe(true)
      })

      it('撤回非最后一条消息时不应更新会话的 lastMessage', () => {
        const store = useChatStore()
        store.setConversations([
          { 
            id: '1', 
            name: '会话1',
            lastMessage: { id: 'm2', content: '最后消息' } as Message
          } as Conversation
        ])
        store.setMessages('1', [
          { id: 'm1', content: '消息1' } as Message,
          { id: 'm2', content: '消息2' } as Message
        ])
        
        store.recallMessage('1', 'm1')
        
        expect(store.conversations[0].lastMessage?.content).toBe('最后消息')
      })
    })

    describe('receiveMessage', () => {
      it('应该能接收新消息', () => {
        const store = useChatStore()
        store.setConversations([
          { id: '1', name: '会话1' } as Conversation
        ])
        store.setMessages('1', [])
        
        store.receiveMessage('1', { id: 'm1', content: '新消息', timestamp: 1000 } as Message, true)
        
        const msgs = store.messages.get('1')
        expect(msgs).toHaveLength(1)
        expect(msgs?.[0].content).toBe('新消息')
      })

      it('接收消息时应该更新会话的 lastMessage 和 timestamp', () => {
        const store = useChatStore()
        store.setConversations([
          { id: '1', name: '会话1', timestamp: 0 } as Conversation
        ])
        
        store.receiveMessage('1', { id: 'm1', content: '新消息', timestamp: 1000 } as Message, true)
        
        expect(store.conversations[0].lastMessage?.content).toBe('新消息')
        expect(store.conversations[0].timestamp).toBe(1000)
      })

      it('非当前会话接收消息时应该增加未读计数', () => {
        const store = useChatStore()
        store.setConversations([
          { id: '1', name: '会话1', unreadCount: 0 } as Conversation
        ])
        
        store.receiveMessage('1', { id: 'm1', content: '新消息' } as Message, false)
        
        expect(store.conversations[0].unreadCount).toBe(1)
      })

      it('当前会话接收消息时不应增加未读计数', () => {
        const store = useChatStore()
        store.setConversations([
          { id: '1', name: '会话1', unreadCount: 0 } as Conversation
        ])
        
        store.receiveMessage('1', { id: 'm1', content: '新消息' } as Message, true)
        
        expect(store.conversations[0].unreadCount).toBe(0)
      })

      it('流式消息不应增加未读计数', () => {
        const store = useChatStore()
        store.setConversations([
          { id: '1', name: '会话1', unreadCount: 0 } as Conversation
        ])
        
        store.receiveMessage('1', { id: 'm1', content: '流式消息', isStreaming: true } as Message, false)
        
        expect(store.conversations[0].unreadCount).toBe(0)
      })
    })

    describe('markConversationRead', () => {
      it('应该能标记会话为已读', () => {
        const store = useChatStore()
        store.setConversations([
          { id: '1', name: '会话1', unreadCount: 5 } as Conversation
        ])
        
        store.markConversationRead('1')
        
        expect(store.conversations[0].unreadCount).toBe(0)
      })
    })

    describe('sortedConversations', () => {
      it('置顶会话应该排在前面', () => {
        const store = useChatStore()
        store.setConversations([
          { id: '1', name: '会话1', pinned: false, timestamp: 3000 } as Conversation,
          { id: '2', name: '会话2', pinned: true, timestamp: 1000 } as Conversation,
          { id: '3', name: '会话3', pinned: false, timestamp: 2000 } as Conversation
        ])
        
        const sorted = store.sortedConversations
        
        expect(sorted[0].id).toBe('2')
        expect(sorted[1].id).toBe('1')
        expect(sorted[2].id).toBe('3')
      })

      it('非置顶会话应该按时间戳降序排列', () => {
        const store = useChatStore()
        store.setConversations([
          { id: '1', name: '会话1', pinned: false, timestamp: 1000 } as Conversation,
          { id: '2', name: '会话2', pinned: false, timestamp: 3000 } as Conversation,
          { id: '3', name: '会话3', pinned: false, timestamp: 2000 } as Conversation
        ])
        
        const sorted = store.sortedConversations
        
        expect(sorted[0].id).toBe('2')
        expect(sorted[1].id).toBe('3')
        expect(sorted[2].id).toBe('1')
      })
    })

    describe('loadConversationsFromStorage', () => {
      it('应该能从 localStorage 加载会话', () => {
        const storedConvs = [
          { id: '1', name: '存储的会话1' },
          { id: '2', name: '存储的会话2' }
        ]
        localStorage.setItem('qim_conversations', JSON.stringify(storedConvs))
        
        const store = useChatStore()
        store.loadConversationsFromStorage()
        
        expect(store.conversations).toHaveLength(2)
        expect(store.conversations[0].name).toBe('存储的会话1')
      })

      it('localStorage 为空时不应覆盖现有状态', () => {
        const store = useChatStore()
        store.setConversations([
          { id: '1', name: '现有会话' } as Conversation
        ])
        
        store.loadConversationsFromStorage()
        
        expect(store.conversations).toHaveLength(1)
        expect(store.conversations[0].name).toBe('现有会话')
      })
    })
  })
})

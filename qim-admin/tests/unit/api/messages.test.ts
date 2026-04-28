import { describe, it, expect, vi } from 'vitest'
import { searchMessages, getMessageDetail } from '@/api/messages'
import request from '@/utils/request'

vi.mock('@/utils/request')

describe('Messages API', () => {
  it('should search messages', async () => {
    const mockResponse = {
      list: [
        {
          id: 1,
          conversationId: 1,
          senderId: 1,
          senderName: 'user1',
          messageType: 'text',
          content: 'hello',
          createdAt: '2026-04-28T10:00:00Z',
          updatedAt: '2026-04-28T10:00:00Z'
        }
      ],
      total: 1,
      page: 1,
      pageSize: 20
    }
    
    vi.mocked(request.get).mockResolvedValue({ data: mockResponse })
    
    const params = { keyword: 'hello', page: 1, pageSize: 20 }
    const result = await searchMessages(params)
    
    expect(request.get).toHaveBeenCalledWith('/api/messages/search', { params })
    expect(result.data).toEqual(mockResponse)
  })
  
  it('should get message detail', async () => {
    const mockResponse = {
      id: 1,
      conversationId: 1,
      senderId: 1,
      senderName: 'user1',
      messageType: 'text',
      content: 'hello',
      createdAt: '2026-04-28T10:00:00Z',
      updatedAt: '2026-04-28T10:00:00Z'
    }
    
    vi.mocked(request.get).mockResolvedValue({ data: mockResponse })
    
    const result = await getMessageDetail(1)
    
    expect(request.get).toHaveBeenCalledWith('/api/messages/1')
    expect(result.data).toEqual(mockResponse)
  })
})

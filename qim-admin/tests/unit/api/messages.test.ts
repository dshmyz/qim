import { describe, it, expect, vi, beforeEach } from 'vitest'
import { searchMessages, getMessageDetail } from '@/api/messages'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('Messages API', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('should search messages', async () => {
    const mockResponse = {
      data: {
        code: 0,
        message: 'success',
        data: {
          list: [
            {
              id: 1,
              conversationId: 1,
              senderId: 1,
              senderName: 'user1',
              messageType: 'text',
              content: 'hello',
              createdAt: '2026-04-28T10:00:00Z',
              updatedAt: '2026-04-28T10:00:00Z',
            },
          ],
          total: 1,
          page: 1,
          pageSize: 20,
        },
      },
    }

    mockRequest.mockResolvedValue(mockResponse)

    const params = { keyword: 'hello', page: 1, pageSize: 20 }
    const result = await searchMessages(params as any)

    expect(mockRequest).toHaveBeenCalledWith({
      url: '/v1/messages/search',
      method: 'get',
      params,
    })
    expect(result.data.data).toEqual(mockResponse.data.data)
  })

  it('should get message detail', async () => {
    const mockResponse = {
      data: {
        code: 0,
        message: 'success',
        data: {
          id: 1,
          conversationId: 1,
          senderId: 1,
          senderName: 'user1',
          messageType: 'text',
          content: 'hello',
          createdAt: '2026-04-28T10:00:00Z',
          updatedAt: '2026-04-28T10:00:00Z',
        },
      },
    }

    mockRequest.mockResolvedValue(mockResponse)

    const result = await getMessageDetail(1)

    expect(mockRequest).toHaveBeenCalledWith({
      url: '/v1/messages/1',
      method: 'get',
    })
    expect(result.data.data).toEqual(mockResponse.data.data)
  })
})

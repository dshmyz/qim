import { request } from './useRequest'
import { logger } from '../utils/logger'

export interface UserProfileInfo {
  id: string | number
  name: string
  username?: string
  email?: string
  mobile?: string
  department?: string
  ip?: string
  avatar?: string
}

export interface FetchUserProfileResult {
  profile: UserProfileInfo
  success: boolean
}

function isValidProfileUserId(userId: string | number): boolean {
  if (typeof userId === 'number') {
    return Number.isInteger(userId) && userId > 0
  }

  const normalized = userId.trim()
  if (!normalized || normalized === '0') return false

  const numericId = Number(normalized)
  return Number.isInteger(numericId) && numericId > 0
}

/**
 * 根据 userId 获取用户详情，映射为统一的 UserProfileInfo 格式
 * API 调用失败时回退到 fallbackUser，通过 success 标志告知调用方
 */
export async function fetchUserProfile(userId: string | number, fallbackUser?: any): Promise<FetchUserProfileResult> {
  if (!isValidProfileUserId(userId)) {
    return {
      success: false,
      profile: fallbackUser || { id: userId, name: '' }
    }
  }

  try {
    const response = await request(`/api/v1/users/${userId}`)
    if (response.code === 0 && response.data) {
      const userData = response.data
      return {
        success: true,
        profile: {
          id: userData.id,
          name: userData.nickname || userData.username || fallbackUser?.name || '',
          username: userData.username,
          email: userData.email,
          mobile: userData.phone,
          department: userData.department,
          ip: userData.ip,
          avatar: userData.avatar || fallbackUser?.avatar
        }
      }
    }
  } catch (error) {
    logger.error('获取用户信息失败:', error)
  }

  // 回退到传入的用户数据
  return {
    success: false,
    profile: fallbackUser || { id: userId, name: '' }
  }
}

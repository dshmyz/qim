import { ref, watch } from 'vue'
import { generateAvatar, isAbsoluteUrl } from '../utils/avatar'
import { getStoredServerUrl } from './useServerUrl'

export interface UserProfile {
  nickname: string
  username: string
  signature: string
  id: string
  joinDate: string
}

export interface CurrentUser {
  id: string | number
  username: string
  nickname?: string
  avatar?: string
  signature?: string
  isAdmin?: boolean
  roles?: string[]
  [key: string]: any
}

export function useCurrentUser() {
  const currentUser = ref<CurrentUser | null>(getCurrentUser())

  const userProfile = ref<UserProfile>({
    nickname: currentUser.value?.nickname || currentUser.value?.username || '我的账号',
    username: currentUser.value?.username || '',
    signature: currentUser.value?.signature || '这个人很懒，什么都没留下',
    id: currentUser.value?.id?.toString() || 'user_123456',
    joinDate: '2023-01-01'
  })

  function getCurrentUser(): CurrentUser | null {
    const userStr = localStorage.getItem('user')
    if (userStr) {
      try {
        const user = JSON.parse(userStr)
        if (user && user.id) {
          user.isAdmin = user.roles?.includes('system_admin') || false
          return user
        }
      } catch (error) {
        console.error('解析用户信息失败:', error)
      }
    }
    return null
  }

  const refreshUser = async () => {
    const token = localStorage.getItem('token')
    const serverUrl = getStoredServerUrl()
    if (!token || !serverUrl) return

    try {
      const response = await fetch(`${serverUrl}/api/v1/users/me`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      })
      const data = await response.json()
      if (data.code === 0 && data.data) {
        const user = data.data
        user.isAdmin = user.roles?.includes('system_admin') || false
        localStorage.setItem('user', JSON.stringify(user))
        currentUser.value = user
      }
    } catch (error) {
      console.error('刷新用户信息失败:', error)
    }
  }

  const syncUserProfile = () => {
    userProfile.value = {
      nickname: currentUser.value?.nickname || currentUser.value?.username || '我的账号',
      username: currentUser.value?.username || '',
      signature: currentUser.value?.signature || '这个人很懒，什么都没留下',
      id: currentUser.value?.id?.toString() || 'user_123456',
      joinDate: '2023-01-01'
    }
  }

  watch(() => currentUser.value, () => {
    syncUserProfile()
  }, { deep: true })

  const getProfileAvatar = (serverUrl: string): string => {
    if (!currentUser.value?.avatar) return generateAvatar('me')
    if (isAbsoluteUrl(currentUser.value.avatar)) return currentUser.value.avatar
    return serverUrl + currentUser.value.avatar
  }

  return {
    currentUser,
    userProfile,
    syncUserProfile,
    getProfileAvatar,
    refreshUser
  }
}

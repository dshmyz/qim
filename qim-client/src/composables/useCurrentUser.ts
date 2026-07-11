import { ref, watch } from 'vue'
import { generateAvatar, isAbsoluteUrl } from '../utils/avatar'
import { getStoredServerUrl } from './useServerUrl'

export interface UserProfile {
  nickname: string
  username: string
  signature: string
  phone: string
  email: string
  gender: string
  department: string
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

  const formatJoinDate = (createdAt?: string | Date): string => {
    if (!createdAt) return ''
    const d = new Date(createdAt)
    if (isNaN(d.getTime())) return ''
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
  }

  const buildProfile = (): UserProfile => ({
    nickname: currentUser.value?.nickname || currentUser.value?.username || '我的账号',
    username: currentUser.value?.username || '',
    signature: currentUser.value?.signature || '',
    phone: currentUser.value?.phone || '',
    email: currentUser.value?.email || '',
    gender: currentUser.value?.gender || 'secret',
    department: currentUser.value?.organization || '',
    joinDate: formatJoinDate(currentUser.value?.created_at),
  })

  const userProfile = ref<UserProfile>(buildProfile())

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
    userProfile.value = buildProfile()
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

import { ref, watch } from 'vue'

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
          user.isAdmin = true
          return user
        }
      } catch (error) {
        console.error('解析用户信息失败:', error)
      }
    }
    return {
      id: '1',
      username: 'admin',
      nickname: '管理员',
      avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=admin',
      isAdmin: true
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
    if (!currentUser.value?.avatar) return 'https://api.dicebear.com/7.x/avataaars/svg?seed=me'
    if (currentUser.value.avatar.startsWith('http')) return currentUser.value.avatar
    return serverUrl + currentUser.value.avatar
  }

  return {
    currentUser,
    userProfile,
    syncUserProfile,
    getProfileAvatar
  }
}

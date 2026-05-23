export interface User {
  id: string | number
  username: string
  nickname?: string
  avatar?: string
  isAdmin?: boolean
  [key: string]: any
}

const defaultUser: User = {
  id: '1',
  username: 'admin',
  nickname: '管理员',
  avatar: '',
  isAdmin: false
}

export const getCurrentUser = (): User => {
  const userStr = localStorage.getItem('user')
  if (userStr) {
    try {
      const user = JSON.parse(userStr) as User
      if (user && user.id) {
        user.isAdmin = user.roles?.includes('system_admin') || false
        return user
      }
    } catch (error) {
      console.error('解析用户信息失败:', error)
    }
  }
  return { ...defaultUser }
}

export const setCurrentUser = (user: User): void => {
  localStorage.setItem('user', JSON.stringify(user))
}

export const clearCurrentUser = (): void => {
  localStorage.removeItem('user')
}

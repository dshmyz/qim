import { Ref } from 'vue'
import { request } from './useRequest'
import { logger } from '../utils/logger'
import QMessage from '../utils/qmessage'

export function useUserProfile(
  currentUser: Ref<any>,
  closeUserProfile: () => void
) {
  const showMessage = (options: { message: string, type?: string, duration?: number }) => {
    const { message, type = 'info', duration } = options
    if (type === 'success') QMessage.success(message, duration)
    else if (type === 'error') QMessage.error(message, duration)
    else if (type === 'warning') QMessage.warning(message, duration)
    else QMessage.info(message, duration)
  }

  const triggerAvatarInput = () => {
    const input = document.querySelector('.avatar-input') as HTMLInputElement
    if (input) {
      input.click()
    }
  }

  const handleAvatarChange = async (event: Event) => {
    const input = event.target as HTMLInputElement
    if (input.files && input.files.length > 0) {
      const file = input.files[0]

      if (!file.type.startsWith('image/')) {
        showMessage({ message: '请选择图片文件', type: 'error' })
        return
      }

      if (file.size > 5 * 1024 * 1024) {
        showMessage({ message: '图片大小不能超过5MB', type: 'error' })
        return
      }

      try {
        const formData = new FormData()
        formData.append('file', file)

        const response = await request('/api/v1/upload', {
          method: 'POST',
          body: formData
        })

        if (response.code === 0 && response.data && response.data.url) {
          const updateResponse = await request('/api/v1/users/me', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ avatar: response.data.url })
          })

          if (updateResponse.code === 0 && updateResponse.data) {
            if (currentUser.value) {
              currentUser.value.avatar = updateResponse.data.avatar || response.data.url
              localStorage.setItem('user', JSON.stringify(currentUser.value))
            }
            showMessage({ message: '头像更新成功', type: 'success' })
          } else {
            showMessage({ message: '头像更新失败: ' + updateResponse.message, type: 'error' })
          }
        } else {
          showMessage({ message: '文件上传失败: ' + response.message, type: 'error' })
        }
      } catch (error: any) {
        logger.error('头像上传失败:', error)
        showMessage({ message: '头像上传失败: ' + error.message, type: 'error' })
      }
    }
  }

  const saveUserProfile = async (profile: any) => {
    try {
      const updateData: any = {
        nickname: profile.nickname,
        signature: profile.signature
      }

      if (profile.avatarFile) {
        const formData = new FormData()
        formData.append('file', profile.avatarFile)

        const uploadResponse = await request('/api/v1/upload', {
          method: 'POST',
          body: formData
        })

        if (uploadResponse.code === 0 && uploadResponse.data && uploadResponse.data.url) {
          updateData.avatar = uploadResponse.data.url
        } else {
          showMessage({ message: '头像上传失败: ' + uploadResponse.message, type: 'error' })
          return
        }
      }

      const response = await request('/api/v1/users/me', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updateData)
      })

      if (response.code === 0) {
        if (currentUser.value) {
          currentUser.value.nickname = profile.nickname
          if (updateData.avatar) {
            currentUser.value.avatar = updateData.avatar
          }
        }
        showMessage({ message: '保存成功', type: 'success' })
        closeUserProfile()
      } else {
        showMessage({ message: '保存失败: ' + response.message, type: 'error' })
      }
    } catch (error: any) {
      logger.error('保存用户资料失败:', error)
      showMessage({ message: '保存失败: ' + error.message, type: 'error' })
    }
  }

  return {
    triggerAvatarInput,
    handleAvatarChange,
    saveUserProfile
  }
}

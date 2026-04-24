import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { request } from './useRequest'

interface Conversation {
  id: string
  name: string
  type: string
  members?: any[]
  [key: string]: any
}

export function useGroup() {
  const selectedGroup = ref<Conversation | null>(null)
  const showGroupContextMenuFlag = ref(false)
  const groupContextMenuPosition = ref({ x: 0, y: 0 })
  const selectedGroupForContextMenu = ref<Conversation | null>(null)

  const showGroupContextMenu = (event: MouseEvent, group: Conversation) => {
    event.preventDefault()
    showGroupContextMenuFlag.value = true
    groupContextMenuPosition.value = { x: event.clientX, y: event.clientY }
    selectedGroupForContextMenu.value = group
  }

  const closeGroupContextMenu = () => {
    showGroupContextMenuFlag.value = false
    selectedGroupForContextMenu.value = null
  }

  const dissolveGroup = async (group: Conversation) => {
    if (!group) {
      closeGroupContextMenu()
      return
    }

    try {
      await ElMessageBox.confirm(
        `确定要解散群聊 "${group.name}" 吗？此操作不可恢复。`,
        '确认解散群聊',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )
    } catch {
      return
    }

    try {
      const response: any = await request(`/api/v1/conversations/${group.id}`, {
        method: 'DELETE'
      })

      if (response.code === 0) {
        ElMessage.success('群聊已成功解散')
        closeGroupContextMenu()
        return true
      } else {
        ElMessage.error(response.message || '解散群聊失败')
      }
    } catch (error) {
      console.error('解散群聊失败:', error)
      ElMessage.error('网络错误，解散群聊失败')
    }

    return false
  }

  const getGroupOwner = (group: Conversation | null): string => {
    if (!group || !group.members) return ''
    const owner = group.members.find((member: any) => member.role === 'owner')
    return owner ? owner.name : ''
  }

  const isGroupOwnerCheck = (group: Conversation | null, currentUserId: string | undefined): boolean => {
    if (!group || !group.members || !currentUserId) return false
    const owner = group.members.find((member: any) => member.role === 'owner')
    return owner && owner.id === currentUserId
  }

  return {
    selectedGroup,
    showGroupContextMenuFlag,
    groupContextMenuPosition,
    selectedGroupForContextMenu,
    showGroupContextMenu,
    closeGroupContextMenu,
    dissolveGroup,
    getGroupOwner,
    isGroupOwnerCheck
  }
}

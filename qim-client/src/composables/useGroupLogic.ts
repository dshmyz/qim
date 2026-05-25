import { ref } from 'vue'
import type { Conversation } from '../types'
import { useChatStore } from '../stores/chat'
import { useGroup } from './useGroup'
import { request } from '../composables/useRequest'
import { logger } from '../utils/logger'
import QMessage from '../utils/qmessage'

export function useGroupLogic() {
  const chatStore = useChatStore()
  const groupState = useGroup()
  
  const selectedGroup = ref<Conversation | null>(null)
  const selectedMember = ref<any>(null)
  const editGroupName = ref('')
  const editAnnouncementContent = ref('')
  const showAddMembersModal = ref(false)
  const selectedAddMembers = ref<any[]>([])
  const addMembersSearchQuery = ref('')
  const groupMembers = ref<any[]>([])

  const showMessage = (options: { message: string, type?: 'success' | 'warning' | 'error' | 'info', duration?: number }) => {
    const { message, type = 'info', duration } = options
    if (type === 'success') QMessage.success(message, duration)
    else if (type === 'error') QMessage.error(message, duration)
    else if (type === 'warning') QMessage.warning(message, duration)
    else QMessage.info(message, duration)
  }

  const selectGroup = (group: Conversation | null) => {
    selectedGroup.value = group
    if (group) {
      editGroupName.value = group.name || ''
      groupMembers.value = group.members || []
    }
  }

  const handleInviteMembers = (groupOrId: any) => {
    let group = null
    if (typeof groupOrId === 'string') {
      group = chatStore.conversations.find(c => c.id === groupOrId)
    } else {
      group = groupOrId
    }
    
    if (group) {
      selectedGroup.value = group
      selectedAddMembers.value = []
      addMembersSearchQuery.value = ''
      showAddMembersModal.value = true
    }
  }

  const updateGroupName = async (newName: string) => {
    if (!selectedGroup.value || !newName.trim()) return
    
    const success = await groupState.updateGroup(selectedGroup.value.id, { name: newName.trim() })
    if (success) {
      selectedGroup.value.name = newName.trim()
      chatStore.patchConversation(selectedGroup.value.id, { name: newName.trim() })
      showMessage({ message: '群名称更新成功', type: 'success' })
    }
  }

  const updateAnnouncement = async (content: string) => {
    if (!selectedGroup.value) return
    
    const success = await groupState.updateAnnouncement(selectedGroup.value.id, content)
    if (success) {
      selectedGroup.value.announcement = content
      chatStore.patchConversation(selectedGroup.value.id, { announcement: content })
      showMessage({ message: '群公告更新成功', type: 'success' })
    }
  }

  const removeMember = async (member: any) => {
    if (!selectedGroup.value) return
    
    const success = await groupState.removeGroupMember(selectedGroup.value.id, member.id)
    if (success) {
      if (selectedGroup.value.members) {
        selectedGroup.value.members = selectedGroup.value.members.filter(m => m.id !== member.id)
      }
      chatStore.removeGroupMember(selectedGroup.value.id, member.id)
      showMessage({ message: '成员移除成功', type: 'success' })
    }
  }

  const addMembers = async (members: any[]) => {
    if (!selectedGroup.value || !members || members.length === 0) return
    
    try {
      const response = await request(`/api/v1/groups/${selectedGroup.value.id}/members`, {
        method: 'POST',
        body: JSON.stringify({ user_ids: members.map(m => m.id) })
      })
      
      if (response.code === 0) {
        const newMembers = response.data || []
        if (selectedGroup.value.members) {
          selectedGroup.value.members = [...selectedGroup.value.members, ...newMembers]
        } else {
          selectedGroup.value.members = newMembers
        }
        
        groupMembers.value = selectedGroup.value.members || []
        
        newMembers.forEach(member => {
          chatStore.addGroupMember(selectedGroup.value!.id, member)
        })
        
        showMessage({ message: '成员添加成功', type: 'success' })
        showAddMembersModal.value = false
      }
    } catch (error) {
      logger.error('添加成员失败:', error)
      showMessage({ message: '添加成员失败', type: 'error' })
    }
  }

  const setAsAdmin = async (member: any) => {
    if (!selectedGroup.value) return
    
    await groupState.setAsAdmin(selectedGroup.value.id, member.id)
    const foundMember = selectedGroup.value.members.find(m => m.id === member.id)
    if (foundMember) {
      foundMember.role = 'admin'
    }
    showMessage({ message: '已设置为管理员', type: 'success' })
  }

  const exitGroup = async () => {
    if (!selectedGroup.value) return
    
    if (confirm(`确定要退出${selectedGroup.value.name}吗？`)) {
      try {
        const response = await request(`/api/v1/groups/${selectedGroup.value.id}/exit`, {
          method: 'POST'
        })
        
        if (response.code === 0) {
          showMessage({ message: '已退出群聊', type: 'success' })
          chatStore.patchConversation(selectedGroup.value.id, { isExited: true } as any)
          selectedGroup.value = null
        }
      } catch (error) {
        logger.error('退出群聊失败:', error)
        showMessage({ message: '退出群聊失败', type: 'error' })
      }
    }
  }

  const updateAISettings = async (settings: any) => {
    if (!selectedGroup.value) return
    
    try {
      const response = await request(`/api/v1/groups/${selectedGroup.value.id}/ai-settings`, {
        method: 'PUT',
        body: JSON.stringify(settings)
      })
      
      if (response.code === 0) {
        selectedGroup.value.ai_config = {
          ...selectedGroup.value.ai_config,
          ...settings
        }
        showMessage({ message: 'AI 设置更新成功', type: 'success' })
      }
    } catch (error) {
      logger.error('更新 AI 设置失败:', error)
      showMessage({ message: '更新 AI 设置失败', type: 'error' })
    }
  }

  return {
    selectedGroup,
    selectedMember,
    editGroupName,
    editAnnouncementContent,
    showAddMembersModal,
    selectedAddMembers,
    addMembersSearchQuery,
    groupMembers,
    selectGroup,
    handleInviteMembers,
    updateGroupName,
    updateAnnouncement,
    removeMember,
    addMembers,
    setAsAdmin,
    exitGroup,
    updateAISettings,
    showMessage
  }
}

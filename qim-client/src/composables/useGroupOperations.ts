import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

interface Conversation {
  id: string
  name: string
  type: string
  members?: any[]
  announcement?: string
  [key: string]: any
}

export function useGroupOperations(request: any, conversations: any) {
  const editAnnouncementContent = ref('')
  const showEditAnnouncementModal = ref(false)
  const showGroupMembersModal = ref(false)
  const showGroupInfoModal = ref(false)
  const showAddMembersModal = ref(false)
  const selectedGroup = ref<Conversation | null>(null)
  const selectedMember = ref<any>(null)
  const groupMembers = ref<any[]>([])
  const selectedAddMembers = ref<any[]>([])
  const addMembersSearchQuery = ref('')

  const closeEditAnnouncementModal = () => {
    showEditAnnouncementModal.value = false
    editAnnouncementContent.value = ''
  }

  const closeMemberContextMenu = () => {
    selectedMember.value = null
  }

  const editAnnouncement = (group: Conversation) => {
    if (!group) return
    selectedGroup.value = group
    editAnnouncementContent.value = group.announcement || ''
    showEditAnnouncementModal.value = true
  }

  const saveAnnouncement = async () => {
    if (!selectedGroup.value) {
      closeEditAnnouncementModal()
      return
    }
    
    try {
      const response = await request(`/api/v1/conversations/${selectedGroup.value.id}/announcement`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ announcement: editAnnouncementContent.value })
      })
      
      if (response.code === 0) {
        ElMessage.success('群公告已成功更新')
        selectedGroup.value.announcement = editAnnouncementContent.value
        const index = conversations.value.findIndex((c: Conversation) => c.id === selectedGroup.value?.id)
        if (index > -1) {
          conversations.value[index].announcement = editAnnouncementContent.value
        }
      } else {
        ElMessage.error(response.message || '更新群公告失败')
      }
    } catch (error) {
      console.error('更新群公告失败:', error)
      ElMessage.error('网络错误，更新群公告失败')
    }
    closeEditAnnouncementModal()
  }

  const removeMemberFromGroup = async () => {
    if (selectedMember.value && selectedGroup.value) {
      try {
        const response = await request(`/api/v1/conversations/${selectedGroup.value.id}/members/${selectedMember.value.id}`, {
          method: 'DELETE',
          headers: {
            'Content-Type': 'application/json'
          }
        })
        
        if (response.code === 0) {
          ElMessage.success('成员已成功移除')
          const index = selectedGroup.value.members!.findIndex((member: any) => member.id === selectedMember.value.id)
          if (index > -1) {
            selectedGroup.value.members!.splice(index, 1)
          }
        } else {
          ElMessage.error(response.message || '移除成员失败')
        }
      } catch (error) {
        console.error('移除成员失败:', error)
        ElMessage.error('网络错误，移除成员失败')
      }
    }
    closeMemberContextMenu()
  }

  const viewMemberInfo = () => {
    if (selectedMember.value) {
      ElMessage.info(`查看${selectedMember.value.name}的资料`)
      console.log('查看成员资料:', selectedMember.value)
    }
    closeMemberContextMenu()
  }

  const setAsAdmin = async () => {
    if (selectedMember.value && selectedGroup.value) {
      try {
        const response = await request(`/api/v1/conversations/${selectedGroup.value.id}/members/${selectedMember.value.id}/role`, {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({ role: 'admin' })
        })
        
        if (response.code === 0) {
          ElMessage.success('已成功设为管理员')
          const member = selectedGroup.value.members!.find((m: any) => m.id === selectedMember.value.id)
          if (member) {
            member.role = 'admin'
          }
        } else {
          ElMessage.error(response.message || '设置管理员失败')
        }
      } catch (error) {
        console.error('设置管理员失败:', error)
        ElMessage.error('网络错误，设置管理员失败')
      }
    }
    closeMemberContextMenu()
  }

  const viewGroupMembers = (group: Conversation | null, serverUrl: string) => {
    if (!group) return
    groupMembers.value = (group.members || []).map((member: any) => ({
      id: member.user && member.user.id ? member.user.id.toString() : (member.id ? member.id.toString() : ''),
      name: member.user ? (member.user.nickname || member.user.username || '') : (member.name || ''),
      avatar: member.user ? (
        member.user.avatar && member.user.avatar.startsWith('http')
          ? member.user.avatar
          : (member.user.avatar ? serverUrl + member.user.avatar : '')
      ) : (member.avatar || ''),
      position: member.user ? (member.user.position || '无职位信息') : (member.position || '无职位信息')
    }))
    showGroupMembersModal.value = true
  }

  const viewGroupInfo = (group: Conversation | null) => {
    if (!group) return
    selectedGroup.value = group
    showGroupInfoModal.value = true
  }

  const addMembersToGroup = (group: Conversation | null) => {
    if (!group) return
    selectedGroup.value = group
    selectedAddMembers.value = []
    addMembersSearchQuery.value = ''
    showAddMembersModal.value = true
  }

  const handleInviteMembers = (groupOrId: string | Conversation) => {
    let group: Conversation | null = null
    if (typeof groupOrId === 'string') {
      group = conversations.value.find((c: Conversation) => c.id === groupOrId) || null
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

  return {
    editAnnouncementContent,
    showEditAnnouncementModal,
    showGroupMembersModal,
    showGroupInfoModal,
    showAddMembersModal,
    selectedGroup,
    selectedMember,
    groupMembers,
    selectedAddMembers,
    addMembersSearchQuery,
    editAnnouncement,
    saveAnnouncement,
    removeMemberFromGroup,
    viewMemberInfo,
    setAsAdmin,
    viewGroupMembers,
    viewGroupInfo,
    addMembersToGroup,
    handleInviteMembers,
    closeEditAnnouncementModal,
    closeMemberContextMenu
  }
}

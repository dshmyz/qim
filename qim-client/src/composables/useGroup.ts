import { ref, computed, readonly } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { request, useRequest } from './useRequest'

/**
 * 群组成员信息接口
 */
export interface GroupMember {
  id: string
  name: string
  avatar?: string
  position?: string
  role?: 'owner' | 'admin' | 'member'
  user?: {
    id?: string | number
    nickname?: string
    username?: string
    avatar?: string
    position?: string
  }
}

/**
 * 群组/会话信息接口
 */
export interface GroupInfo {
  id: string
  name: string
  type: string
  members?: GroupMember[]
  avatar?: string
  announcement?: string
  created_at?: number
  [key: string]: any
}

/**
 * 群组管理 composable
 * 提供群组创建、解散、退出、管理等功能
 */
export function useGroup() {
  const {
    isRequesting,
    lastError
  } = useRequest()

  // 选中的群组
  const selectedGroup = ref<GroupInfo | null>(null)

  // 群组右键菜单状态
  const showGroupContextMenuFlag = ref(false)
  const groupContextMenuPosition = ref({ x: 0, y: 0 })
  const selectedGroupForContextMenu = ref<GroupInfo | null>(null)

  // 群组成员列表
  const groupMembers = ref<GroupMember[]>([])

  // 群组列表
  const groups = ref<GroupInfo[]>([])

  // 当前用户ID（由外部传入）
  let currentUserId: string | undefined = undefined

  /**
   * 设置当前用户ID
   */
  const setCurrentUserId = (userId: string | undefined) => {
    currentUserId = userId
  }

  /**
   * 显示群组右键菜单
   */
  const showGroupContextMenu = (event: MouseEvent, group: GroupInfo) => {
    event.preventDefault()
    showGroupContextMenuFlag.value = true
    groupContextMenuPosition.value = { x: event.clientX, y: event.clientY }
    selectedGroupForContextMenu.value = group
    // 同时设置 selectedGroup
    selectedGroup.value = group
  }

  /**
   * 关闭群组右键菜单
   */
  const closeGroupContextMenu = () => {
    showGroupContextMenuFlag.value = false
    selectedGroupForContextMenu.value = null
  }

  /**
   * 创建群聊
   */
  const createGroup = async (name: string, memberIds: string[]) => {
    try {
      const response: any = await request('/api/v1/conversations', {
        method: 'POST',
        body: JSON.stringify({
          name,
          type: 'group',
          member_ids: memberIds
        })
      })

      if (response.code === 0) {
        ElMessage.success('群聊创建成功')
        return response.data
      } else {
        ElMessage.error(response.message || '创建群聊失败')
        return null
      }
    } catch (error) {
      console.error('创建群聊失败:', error)
      ElMessage.error('网络错误，创建群聊失败')
      return null
    }
  }

  /**
   * 解散群聊
   * 注意：Main.vue 中使用 /dissolve 端点
   */
  const dissolveGroup = async (group: GroupInfo) => {
    if (!group) {
      closeGroupContextMenu()
      return false
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
      return false
    }

    try {
      // 使用 /dissolve 端点，与 Main.vue 一致
      const response: any = await request(`/api/v1/conversations/${group.id}/dissolve`, {
        method: 'POST'
      })

      if (response.code === 0 || response.code === 200) {
        ElMessage.success('群聊已成功解散')
        closeGroupContextMenu()
        return true
      } else {
        ElMessage.error(response.message || '解散群聊失败')
        return false
      }
    } catch (error) {
      console.error('解散群聊失败:', error)
      ElMessage.error('网络错误，解散群聊失败')
      return false
    }
  }

  /**
   * 退出群聊（使用 /exit 端点，与 Main.vue 一致）
   * 注意：此函数与 leaveGroup 不同，使用不同的 API 端点和响应码
   */
  const exitGroup = async (group: GroupInfo) => {
    if (!group) return false

    try {
      await ElMessageBox.confirm(
        `确定要退出群聊 "${group.name}" 吗？`,
        '确认退出群聊',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )
    } catch {
      return false
    }

    try {
      // 使用 /exit 端点，与 Main.vue 一致
      const response: any = await request(`/api/v1/conversations/${group.id}/exit`, {
        method: 'POST'
      })

      if (response.code === 0 || response.code === 200) {
        ElMessage.success('已退出群聊')
        closeGroupContextMenu()
        return true
      } else {
        ElMessage.error(response.message || '退出群聊失败')
        return false
      }
    } catch (error) {
      console.error('退出群聊失败:', error)
      ElMessage.error('网络错误，退出群聊失败')
      return false
    }
  }

  /**
   * 退出群聊（使用 /leave 端点）
   * 这是通用的 leaveGroup 实现，保留作为备用
   */
  const leaveGroup = async (group: GroupInfo) => {
    if (!group) return false

    try {
      await ElMessageBox.confirm(
        `确定要退出群聊 "${group.name}" 吗？`,
        '确认退出群聊',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )
    } catch {
      return false
    }

    try {
      const response: any = await request(`/api/v1/conversations/${group.id}/leave`, {
        method: 'POST'
      })

      if (response.code === 0) {
        ElMessage.success('已退出群聊')
        closeGroupContextMenu()
        return true
      } else {
        ElMessage.error(response.message || '退出群聊失败')
        return false
      }
    } catch (error) {
      console.error('退出群聊失败:', error)
      ElMessage.error('网络错误，退出群聊失败')
      return false
    }
  }

  /**
   * 更新群组信息
   */
  const updateGroup = async (groupId: string, data: Partial<GroupInfo>) => {
    try {
      const response: any = await request(`/api/v1/conversations/${groupId}`, {
        method: 'PUT',
        body: JSON.stringify(data)
      })

      if (response.code === 0) {
        ElMessage.success('群组信息已更新')
        return response.data
      } else {
        ElMessage.error(response.message || '更新群组信息失败')
        return null
      }
    } catch (error) {
      console.error('更新群组信息失败:', error)
      ElMessage.error('网络错误，更新群组信息失败')
      return null
    }
  }

  /**
   * 添加群组成员
   */
  const addGroupMembers = async (groupId: string, memberIds: string[]) => {
    try {
      const response: any = await request(`/api/v1/conversations/${groupId}/members`, {
        method: 'POST',
        body: JSON.stringify({ member_ids: memberIds })
      })

      if (response.code === 0) {
        ElMessage.success('成员已添加')
        return response.data || []
      } else {
        ElMessage.error(response.message || '添加成员失败')
        return null
      }
    } catch (error) {
      console.error('添加群组成员失败:', error)
      ElMessage.error('网络错误，添加成员失败')
      return null
    }
  }

  /**
   * 移除群组成员
   */
  const removeGroupMember = async (groupId: string, memberId: string) => {
    try {
      const response: any = await request(`/api/v1/conversations/${groupId}/members/${memberId}`, {
        method: 'DELETE'
      })

      if (response.code === 0) {
        ElMessage.success('成员已移除')
        // 更新本地群组成员列表
        groupMembers.value = groupMembers.value.filter(m => m.id !== memberId)
        return true
      } else {
        ElMessage.error(response.message || '移除成员失败')
        return false
      }
    } catch (error) {
      console.error('移除群组成员失败:', error)
      ElMessage.error('网络错误，移除成员失败')
      return false
    }
  }

  /**
   * 设置群成员角色（设为管理员）
   */
  const setAsAdmin = async (groupId: string, memberId: string) => {
    try {
      const response: any = await request(`/api/v1/conversations/${groupId}/members/${memberId}/role`, {
        method: 'PUT',
        body: JSON.stringify({ role: 'admin' })
      })

      if (response.code === 0) {
        ElMessage.success('已成功设为管理员')
        // 更新本地成员角色
        const member = groupMembers.value.find(m => m.id === memberId)
        if (member) {
          member.role = 'admin'
        }
        return true
      } else {
        ElMessage.error(response.message || '设置管理员失败')
        return false
      }
    } catch (error) {
      console.error('设置管理员失败:', error)
      ElMessage.error('网络错误，设置管理员失败')
      return false
    }
  }

  /**
   * 更新群公告
   */
  const updateAnnouncement = async (groupId: string, announcement: string) => {
    try {
      const response: any = await request(`/api/v1/conversations/${groupId}/announcement`, {
        method: 'PUT',
        body: JSON.stringify({ announcement })
      })

      if (response.code === 0) {
        ElMessage.success('群公告已更新')
        return true
      } else {
        ElMessage.error(response.message || '更新群公告失败')
        return false
      }
    } catch (error) {
      console.error('更新群公告失败:', error)
      ElMessage.error('网络错误，更新群公告失败')
      return false
    }
  }

  /**
   * 获取群组所有者名称
   */
  const getGroupOwner = (group: GroupInfo | null): string => {
    if (!group || !group.members) return ''
    const owner = group.members.find((member: any) => member.role === 'owner')
    return owner ? (owner.name || owner.user?.nickname || owner.user?.username || '') : ''
  }

  /**
   * 检查是否是群组所有者
   */
  const isGroupOwnerCheck = (group: GroupInfo | null, userId?: string | undefined): boolean => {
    if (!group || !group.members || !userId) return false
    const owner = group.members.find((member: any) => {
      const memberId = member.user?.id?.toString() || member.id?.toString()
      return memberId === userId && member.role === 'owner'
    })
    return !!owner
  }

  /**
   * 检查是否是群组管理员
   */
  const isGroupAdmin = (group: GroupInfo | null, userId?: string | undefined): boolean => {
    if (!group || !group.members || !userId) return false
    const admin = group.members.find((member: any) => {
      const memberId = member.user?.id?.toString() || member.id?.toString()
      return memberId === userId && (member.role === 'owner' || member.role === 'admin')
    })
    return !!admin
  }

  /**
   * 准备群组成员的显示数据（用于模态框）
   */
  const prepareGroupMembersForDisplay = (group: GroupInfo | null, serverUrl: string = ''): GroupMember[] => {
    if (!group) return []
    return (group.members || []).map((member: any) => ({
      id: member.user && member.user.id ? member.user.id.toString() : (member.id ? member.id.toString() : ''),
      name: member.user ? (member.user.nickname || member.user.username || '') : (member.name || ''),
      avatar: member.user ? (
        member.user.avatar && member.user.avatar.startsWith('http')
          ? member.user.avatar
          : (member.user.avatar ? serverUrl + member.user.avatar : '')
      ) : (member.avatar || ''),
      position: member.user ? (member.user.position || '无职位信息') : (member.position || '无职位信息'),
      role: member.role,
      user: member.user
    }))
  }

  /**
   * 加载群组列表
   */
  const loadGroups = async () => {
    try {
      const response: any = await request('/api/v1/conversations', {
        params: { type: 'group' }
      })
      if (response.code === 0) {
        groups.value = response.data || []
      }
    } catch (error) {
      console.error('加载群组列表失败:', error)
    }
  }

  /**
   * 加载群组成员
   */
  const loadGroupMembers = async (groupId: string) => {
    try {
      const response: any = await request(`/api/v1/conversations/${groupId}/members`)
      if (response.code === 0) {
        groupMembers.value = response.data || []
      }
    } catch (error) {
      console.error('加载群组成员失败:', error)
      groupMembers.value = []
    }
  }

  /**
   * 获取当前选中的群组
   */
  const getSelectedGroup = () => selectedGroup.value

  /**
   * 设置选中的群组
   */
  const setSelectedGroup = (group: GroupInfo | null) => {
    selectedGroup.value = group
  }

  /**
   * 设置群组成员列表（用于更新本地状态）
   */
  const setGroupMembers = (members: GroupMember[]) => {
    groupMembers.value = members
  }

  return {
    // 状态
    selectedGroup: readonly(selectedGroup),
    showGroupContextMenuFlag: readonly(showGroupContextMenuFlag),
    groupContextMenuPosition: readonly(groupContextMenuPosition),
    selectedGroupForContextMenu: readonly(selectedGroupForContextMenu),
    groupMembers: readonly(groupMembers),
    groups: readonly(groups),
    isRequesting,
    lastError,

    // 操作方法
    setCurrentUserId,
    showGroupContextMenu,
    closeGroupContextMenu,
    createGroup,
    dissolveGroup,
    exitGroup,
    leaveGroup,
    updateGroup,
    addGroupMembers,
    removeGroupMember,
    setAsAdmin,
    updateAnnouncement,
    getGroupOwner,
    isGroupOwnerCheck,
    isGroupAdmin,
    prepareGroupMembersForDisplay,
    loadGroups,
    loadGroupMembers,
    getSelectedGroup,
    setSelectedGroup,
    setGroupMembers
  }
}
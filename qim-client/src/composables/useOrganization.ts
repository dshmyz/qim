import { ref } from 'vue'
import QMessage from '../utils/qmessage'
import { generateAvatar, isAbsoluteUrl } from '../utils/avatar'

interface Employee {
  id: string
  name: string
  username?: string
  avatar: string
  position?: string
  department?: string
  status?: string
}

interface Department {
  id: string
  name: string
  subDepartments: Department[]
  employees: Employee[]
}

export function useOrganization(serverUrl: any, request: any) {
  const orgStructure = ref<Department[]>([])
  const unassignedUsers = ref<Employee[]>([])
  const selectedUser = ref<Employee | null>(null)

  const convertEmployee = (emp: any, deptName: string): Employee => {
    console.log('员工数据:', emp) // 调试日志
    return {
      id: emp.id ? emp.id.toString() : '',
      name: emp.nickname || emp.username || emp.real_name || '',
      username: emp.username || '',
      avatar: (emp.avatar && isAbsoluteUrl(emp.avatar)) ? emp.avatar : (emp.avatar ? serverUrl.value + emp.avatar : generateAvatar(emp.nickname || emp.username || emp.real_name || '员工')),
      position: '',
      department: deptName,
      status: emp.status || 'offline'
    }
  }

  const loadOrganizationTree = async () => {
    try {
      const response = await request('/api/v1/organization/tree')
      if (response.code === 0) {
        const convertDepartments = (departments: any[]): Department[] => {
          return departments.map(dept => ({
            id: dept.id ? dept.id.toString() : '',
            name: dept.name || '',
            subDepartments: dept.subDepartments ? convertDepartments(dept.subDepartments) : [],
            employees: dept.employees ? dept.employees.map((emp: any) => convertEmployee(emp, dept.name)) : []
          }))
        }

        const data = response.data
        if (data && typeof data === 'object' && Array.isArray(data.departments)) {
          orgStructure.value = convertDepartments(data.departments)
          unassignedUsers.value = Array.isArray(data.unassignedUsers)
            ? data.unassignedUsers.map((emp: any) => convertEmployee(emp, '未分配部门'))
            : []
        } else if (Array.isArray(data)) {
          orgStructure.value = convertDepartments(data)
          unassignedUsers.value = []
        }
      }
    } catch (error) {
      console.error('加载组织架构失败:', error)
    }
  }

  const handleUserClick = (employee: Employee) => {
    selectedUser.value = employee
  }

  const getDepartmentStats = (department: Department): { total: number; online: number } => {
    let totalCount = 0
    let onlineCount = 0

    if (department.employees) {
      totalCount += department.employees.length
      onlineCount += department.employees.filter(emp => emp.status === 'online').length
    }

    if (department.subDepartments) {
      department.subDepartments.forEach(subDept => {
        const stats = getDepartmentStats(subDept)
        totalCount += stats.total
        onlineCount += stats.online
      })
    }

    return { total: totalCount, online: onlineCount }
  }

  return {
    orgStructure,
    unassignedUsers,
    selectedUser,
    loadOrganizationTree,
    handleUserClick,
    getDepartmentStats
  }
}

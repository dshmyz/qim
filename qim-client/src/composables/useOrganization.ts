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
  const selectedUser = ref<Employee | null>(null)

  const loadOrganizationTree = async () => {
    try {
      const response = await request('/api/v1/organization/tree')
      if (response.code === 0) {
        const convertDepartments = (departments: any[]): Department[] => {
          return departments.map(dept => ({
            id: dept.id ? dept.id.toString() : '',
            name: dept.name || '',
            subDepartments: dept.subDepartments ? convertDepartments(dept.subDepartments) : [],
            employees: dept.employees ? dept.employees.map((emp: any) => ({
              id: emp.id ? emp.id.toString() : '',
              name: emp.nickname || emp.username || '',
              username: emp.username || '',
              avatar: (emp.avatar && isAbsoluteUrl(emp.avatar)) ? emp.avatar : (emp.avatar ? serverUrl.value + emp.avatar : generateAvatar('员工')),
              position: '',
              department: dept.name,
              status: emp.status || 'offline'
            })) : []
          }))
        }
        orgStructure.value = convertDepartments(response.data)
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
    selectedUser,
    loadOrganizationTree,
    handleUserClick,
    getDepartmentStats
  }
}

import { ref } from 'vue'
import { request } from '../composables/useRequest'
import { useServerUrl } from './useServerUrl'
import { getAvatarUrl, generateAvatar, isAbsoluteUrl } from '../utils/avatar'
import { logger } from '../utils/logger'

export interface Employee {
  id: string
  name: string
  nickname?: string
  username: string
  avatar: string
  email?: string
  mobile?: string
  position: string
  department: string
  status: string
}

export interface Department {
  id: string
  name: string
  subDepartments: Department[]
  employees: Employee[]
}

export function useOrganizationLogic() {
  const { serverUrl } = useServerUrl()
  const orgStructure = ref<Department[]>([])

  const loadOrganizationTree = async () => {
    try {
      const response = await request('/api/v1/organization/tree')
      if (response.code === 0) {
        logger.log('组织架构数据:', response.data)
        const convertDepartments = (departments: any[]): Department[] => {
          return departments.map(dept => ({
            id: dept.id ? dept.id.toString() : '',
            name: dept.name || '',
            subDepartments: dept.subDepartments ? convertDepartments(dept.subDepartments) : [],
            employees: dept.employees ? dept.employees.map(emp => ({
              id: emp.id ? emp.id.toString() : '',
              name: emp.nickname || emp.username || '',
              nickname: emp.nickname || '',
              username: emp.username || '',
              avatar: (emp.avatar && isAbsoluteUrl(emp.avatar))
                ? emp.avatar
                : (emp.avatar ? serverUrl.value + emp.avatar : generateAvatar('员工')),
              email: emp.email || '',
              mobile: emp.mobile || emp.phone || '',
              position: emp.position || '',
              department: dept.name,
              status: emp.status || 'offline'
            })) : []
          }))
        }
        orgStructure.value = convertDepartments(response.data)
      }
    } catch (error) {
      logger.error('加载组织架构失败:', error)
    }
  }

  const handleUserClick = (employee: Employee) => {
    logger.log('点击用户:', employee)
    return employee
  }

  const collectEmployees = (departments: Department[]): Employee[] => {
    const employees: Employee[] = []
    
    const collect = (depts: Department[]) => {
      depts.forEach(dept => {
        employees.push(...dept.employees)
        if (dept.subDepartments && dept.subDepartments.length > 0) {
          collect(dept.subDepartments)
        }
      })
    }
    
    collect(departments)
    return employees
  }

  const findDepartmentById = (id: string, departments: Department[] = orgStructure.value): Department | null => {
    for (const dept of departments) {
      if (dept.id === id) {
        return dept
      }
      if (dept.subDepartments) {
        const found = findDepartmentById(id, dept.subDepartments)
        if (found) return found
      }
    }
    return null
  }

  const findEmployeeById = (id: string, departments: Department[] = orgStructure.value): Employee | null => {
    for (const dept of departments) {
      const employee = dept.employees.find(emp => emp.id === id)
      if (employee) return employee
      
      if (dept.subDepartments) {
        const found = findEmployeeById(id, dept.subDepartments)
        if (found) return found
      }
    }
    return null
  }

  return {
    orgStructure,
    loadOrganizationTree,
    handleUserClick,
    collectEmployees,
    findDepartmentById,
    findEmployeeById
  }
}

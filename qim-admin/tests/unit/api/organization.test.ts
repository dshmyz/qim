import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  getOrganizationTree, createDepartment, updateDepartment, deleteDepartment,
  addEmployeeToDepartment, removeEmployeeFromDepartment, getDepartmentEmployees,
} from '@/api/organization'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('organization API', () => {
  const mockOrg = {
    id: 1,
    name: '技术部',
    code: 'TECH',
    parentId: null as number | null,
    leaderId: 1,
    description: '技术部门',
    status: 'active' as const,
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
    children: [],
  }

  beforeEach(() => { vi.clearAllMocks() })

  describe('getOrganizationTree', () => {
    it('应该获取组织架构树', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: [mockOrg] } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getOrganizationTree()

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/organization/tree', method: 'get' })
      expect(response.data.data).toHaveLength(1)
    })
  })

  describe('createDepartment', () => {
    it('应该正确创建部门', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockOrg } }
      mockRequest.mockResolvedValue(mockResponse)

      const createData = { name: '新部门', parentId: null, description: '新部门描述' }
      const response = await createDepartment(createData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/departments', method: 'post', data: createData })
      expect(response.data.data).toEqual(mockOrg)
    })
  })

  describe('updateDepartment', () => {
    it('应该正确更新部门', async () => {
      const updatedOrg = { ...mockOrg, name: '更新后的部门' }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedOrg } }
      mockRequest.mockResolvedValue(mockResponse)

      const updateData = { name: '更新后的部门' }
      const response = await updateDepartment(1, updateData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/departments/1', method: 'put', data: updateData })
      expect(response.data.data).toEqual(updatedOrg)
    })
  })

  describe('deleteDepartment', () => {
    it('应该正确删除部门', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await deleteDepartment(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/departments/1', method: 'delete' })
      expect(response.data.code).toBe(0)
    })
  })

  describe('addEmployeeToDepartment', () => {
    it('应该添加员工到部门', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const data = { departmentId: 1, userId: 2 }
      const response = await addEmployeeToDepartment(data)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/department-employees', method: 'post', data })
      expect(response.data.code).toBe(0)
    })
  })

  describe('removeEmployeeFromDepartment', () => {
    it('应该从部门移除员工', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await removeEmployeeFromDepartment(1, 2)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/department-employees/1/2', method: 'delete' })
      expect(response.data.code).toBe(0)
    })
  })

  describe('getDepartmentEmployees', () => {
    it('应该获取部门员工列表', async () => {
      const mockEmployees = [{ id: 1, username: 'user1', nickname: '员工1' }]
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: mockEmployees, total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getDepartmentEmployees(1, { page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/departments/1/employees', method: 'get', params: { page: 1, pageSize: 10 },
      })
      expect(response.data.data.list).toHaveLength(1)
    })
  })
})

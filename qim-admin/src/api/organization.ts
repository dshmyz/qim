import type { ApiResponse, Organization, PaginationParams, PaginatedResponse, User } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export interface CreateDepartmentParams {
  name: string
  parentId?: number | null
  code?: string
  leaderId?: number | null
  description?: string
}

export interface DepartmentEmployeeParams {
  departmentId: number
  userId: number
}

export interface OrganizationTreeResponse {
  departments: Organization[]
  unassignedUsers?: User[]
}

export const getOrganizationTree = (): Promise<AxiosResponse<ApiResponse<OrganizationTreeResponse>>> => {
  return request({
    url: '/v1/organization/tree',
    method: 'get',
  })
}

export const createDepartment = (data: CreateDepartmentParams): Promise<AxiosResponse<ApiResponse<Organization>>> => {
  return request({
    url: '/v1/departments',
    method: 'post',
    data,
  })
}

export const updateDepartment = (id: number, data: Partial<CreateDepartmentParams>): Promise<AxiosResponse<ApiResponse<Organization>>> => {
  return request({
    url: `/v1/departments/${id}`,
    method: 'put',
    data,
  })
}

export const deleteDepartment = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/departments/${id}`,
    method: 'delete',
  })
}

export const addEmployeeToDepartment = (data: DepartmentEmployeeParams): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: '/v1/department-employees',
    method: 'post',
    data,
  })
}

export const removeEmployeeFromDepartment = (departmentId: number, userId: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/department-employees/${departmentId}/${userId}`,
    method: 'delete',
  })
}

export const getDepartmentEmployees = (departmentId: number, params?: PaginationParams): Promise<AxiosResponse<ApiResponse<PaginatedResponse<any>>>> => {
  return request({
    url: `/v1/departments/${departmentId}/employees`,
    method: 'get',
    params,
  })
}

import request from '@/utils/request'

/** 列出所有可用文档（公开接口） */
export function listDocs() {
  return request.get('/v1/public/docs')
}

/** 获取指定文档内容（公开接口） */
export function getDocContent(slug: string) {
  return request.get(`/v1/public/docs/${slug}`)
}

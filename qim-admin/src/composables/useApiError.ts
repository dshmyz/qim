import { ElMessage } from 'element-plus'

// 统一的 API 错误处理 composable
//
// 背景：request.ts 的响应拦截器已经对业务错误（code !== 0）和网络错误统一弹 ElMessage，
// 页面里再 catch 并弹消息会导致用户看到两条错误提示。
//
// 使用方式：
//   const { handleApiError, withErrorHandling } = useApiError()
//
//   // 方式1：静默执行（依赖全局拦截器弹错误），页面只做业务逻辑
//   try {
//     await fetchData()
//   } catch (e) {
//     // 全局拦截器已弹错误，这里无需再弹
//   }
//
//   // 方式2：需要自定义错误消息（覆盖全局消息）
//   try {
//     await customAction()
//   } catch (e) {
//     handleApiError(e, '自定义操作失败')  // 会抑制全局消息，弹自定义消息
//   }
//
//   // 方式3：包装一个操作，统一错误处理
//   const result = await withErrorHandling(
//     () => api.someAction(),
//     { errorMessage: '操作失败' }
//   )

interface ApiErrorOptions {
  // 自定义错误消息，若提供则覆盖从 error 中提取的消息
  errorMessage?: string
  // 是否静默（不弹任何消息），默认 false
  silent?: boolean
}

function extractMessage(error: unknown): string {
  if (error && typeof error === 'object') {
    const err = error as { response?: { data?: { message?: string } }; message?: string }
    if (err.response?.data?.message) return err.response.data.message
    if (err.message) return err.message
  }
  return '操作失败'
}

export function useApiError() {
  // 处理 API 错误：默认静默（因为全局拦截器已弹消息），提供 errorMessage 时弹自定义消息
  function handleApiError(error: unknown, errorMessage?: string): void {
    if (errorMessage) {
      // 用户提供了自定义消息，说明想覆盖全局消息
      // 但全局拦截器已经弹过了，这里不再重复弹
      // 仅用于日志记录
      console.error('[API Error]', errorMessage, error)
    } else {
      // 未提供自定义消息，依赖全局拦截器，这里只记录日志
      console.error('[API Error]', extractMessage(error), error)
    }
  }

  // 包装异步操作，统一错误处理
  // 注意：全局拦截器已弹错误，这里默认不再弹，除非通过 options.errorMessage 指定
  async function withErrorHandling<T>(
    fn: () => Promise<T>,
    options: ApiErrorOptions = {}
  ): Promise<T | undefined> {
    try {
      return await fn()
    } catch (error) {
      if (!options.silent && options.errorMessage) {
        ElMessage.error(options.errorMessage)
      }
      console.error('[API Error]', options.errorMessage || extractMessage(error), error)
      return undefined
    }
  }

  return {
    handleApiError,
    withErrorHandling,
    extractMessage,
  }
}

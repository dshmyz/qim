// API配置文件
export const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

// 产品名单源：产品改名只动这里（前端 UI 文案统一引用）。
// 注意：localStorage 键名前缀（qim_）、环境变量（QIM_*）、令牌前缀（qusr_/qbot_）
// 属于技术命名空间，改名需要数据迁移逻辑，不做简单字符串替换。
export const PRODUCT_NAME = 'QIM'

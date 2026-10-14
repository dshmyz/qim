// API配置文件
export const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

import { getProductName } from './config/appConfig'

// 产品名出口：真源是 package.json 的 build.productName（经 vite define 注入 appConfig），
// 此处仅为兼容既有 import 路径的转发，不要再在这里写字面量。
// 注意：localStorage 键名前缀（qim_）、环境变量（QIM_*）、令牌前缀（qusr_/qbot_）
// 属于技术命名空间，改名需要数据迁移逻辑，不做简单字符串替换。
export const PRODUCT_NAME = getProductName()

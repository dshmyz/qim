import pkg from '../package.json' with { type: 'json' }

// 统一的产品元数据来源：构建时从 package.json 的 build.extraMetadata 读取
const extra = (pkg as any).build?.extraMetadata || {}

export const productName = (extra.productName || 'NUIM').toUpperCase()
export const productNameCN: string = extra.productNameCN || '青雀'
export const productFullName = `${productName}（${productNameCN}）`
export const copyrightYear: string = extra.copyrightYear || '2026'
export const copyrightText = `© ${copyrightYear} ${productNameCN} ${productName}. All rights reserved.`
export const appVersion: string = pkg.version || '1.0.0'

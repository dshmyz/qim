export const APP_CONFIG = {
  productName: 'QIM',
  productNameCN: '青雀',
  productFullName: 'QIM（青雀）',
  copyrightYear: '2026',
  version: '1.0.0'
}

export const getProductName = (): string => {
  return APP_CONFIG.productName
}

export const getProductNameCN = (): string => {
  return APP_CONFIG.productNameCN
}

export const getProductFullName = (): string => {
  return APP_CONFIG.productFullName
}

export const getCopyrightText = (): string => {
  return `© ${APP_CONFIG.copyrightYear} ${APP_CONFIG.productNameCN} ${APP_CONFIG.productName}. All rights reserved.`
}

export const getCopyrightShort = (): string => {
  return `© ${APP_CONFIG.copyrightYear} ${APP_CONFIG.productName}`
}

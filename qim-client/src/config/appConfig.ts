const productName = __APP_NAME__.toUpperCase()
const productNameCN = __APP_PRODUCT_NAME_CN__

export const APP_CONFIG = {
  productName,
  productNameCN,
  productFullName: `${productName} ${productNameCN}`,
  copyrightYear: __APP_COPYRIGHT_YEAR__,
  version: __APP_VERSION__,
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

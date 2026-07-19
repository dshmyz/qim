import { describe, expect, it } from 'vitest'
import { APP_CONFIG, getProductName } from '../../src/config/appConfig'

describe('appConfig', () => {
  it('uses Electron build productName as the display name', () => {
    expect(APP_CONFIG.productName).toBe('青雀 QIM')
    expect(getProductName()).toBe('青雀 QIM')
  })
})

import { describe, expect, it } from 'vitest'
import { sha256Hex } from '@/utils/sha256'

function strToBytes(s: string): Uint8Array {
  return new TextEncoder().encode(s)
}

describe('sha256Hex', () => {
  // NIST FIPS 180-4 标准测试向量
  it('空字符串', () => {
    expect(sha256Hex(strToBytes(''))).toBe('e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855')
  })

  it('abc（3 字节）', () => {
    expect(sha256Hex(strToBytes('abc'))).toBe('ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad')
  })

  it('abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq（56 字节，跨块边界）', () => {
    expect(sha256Hex(strToBytes('abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq')))
      .toBe('248d6a61d20638b8e5c026930c3e6039a33ce45964ff2167f6ecedd419db06c1')
  })

  // 跨 64/128 字节块边界与含中文/特殊字符：与 crypto.subtle 交叉验证
  it('与 crypto.subtle 结果一致（含中文/块边界/特殊字符）', async () => {
    const cases = [
      '你好，世界！',
      'a'.repeat(64),   // 恰好一块
      'a'.repeat(128),  // 恰好两块
      'a'.repeat(55),   // 单块 padding 边界
      'a'.repeat(56),   // 跨块 padding 边界
      '\x00\x01\x02\x03\xff\xfe\xfd',
    ]
    for (const s of cases) {
      const bytes = strToBytes(s)
      const expected = await crypto.subtle.digest('SHA-256', bytes)
      const expectedHex = Array.from(new Uint8Array(expected))
        .map(b => b.toString(16).padStart(2, '0'))
        .join('')
      expect(sha256Hex(bytes), `input=${JSON.stringify(s).slice(0, 40)}`).toBe(expectedHex)
    }
  })
})

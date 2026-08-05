/**
 * SHA-256 纯 JS 实现。
 * 仅在 Web Crypto API（crypto.subtle）不可用时（非 HTTPS / 非 localhost 的非安全上下文）作为兜底，
 * 保证管理员在内网 HTTP 访问时仍可计算文件哈希。
 */
export function sha256Hex(bytes: Uint8Array): string {
  const rightRotate = (value: number, amount: number) => (value >>> amount) | (value << (32 - amount))
  const words = new Uint32Array(((bytes.length + 9 + 63) >> 6) << 4)
  const bitLen = bytes.length * 8
  for (let i = 0; i < bytes.length; i++) {
    words[i >> 2] |= bytes[i] << ((3 - (i % 4)) * 8)
  }
  const wlen = ((bytes.length + 8) >> 6) + 1
  words[bytes.length >> 2] |= 0x80 << ((((3 - (bytes.length % 4)) * 8) % 32) + 0)
  words[wlen * 16 - 2] = Math.floor(bitLen / 0x100000000)
  words[wlen * 16 - 1] = bitLen

  const k = [
    0x428a2f98,0x71374491,0xb5c0fbcf,0xe9b5dba5,0x3956c25b,0x59f111f1,0x923f82a4,0xab1c5ed5,
    0xd807aa98,0x12835b01,0x243185be,0x550c7dc3,0x72be5d74,0x80deb1fe,0x9bdc06a7,0xc19bf174,
    0xe49b69c1,0xefbe4786,0x0fc19dc6,0x240ca1cc,0x2de92c6f,0x4a7484aa,0x5cb0a9dc,0x76f988da,
    0x983e5152,0xa831c66d,0xb00327c8,0xbf597fc7,0xc6e00bf3,0xd5a79147,0x06ca6351,0x14292967,
    0x27b70a85,0x2e1b2138,0x4d2c6dfc,0x53380d13,0x650a7354,0x766a0abb,0x81c2c92e,0x92722c85,
    0xa2bfe8a1,0xa81a664b,0xc24b8b70,0xc76c51a3,0xd192e819,0xd6990624,0xf40e3585,0x106aa070,
    0x19a4c116,0x1e376c08,0x2748774c,0x34b0bcb5,0x391c0cb3,0x4ed8aa4a,0x5b9cca4f,0x682e6ff3,
    0x748f82ee,0x78a5636f,0x84c87814,0x8cc70208,0x90befffa,0xa4506ceb,0xbef9a3f7,0xc67178f2,
  ]
  let h = [0x6a09e667,0xbb67ae85,0x3c6ef372,0xa54ff53a,0x510e527f,0x9b05688c,0x1f83d9ab,0x5be0cd19]
  const w = new Int32Array(64)
  for (let off = 0; off < wlen * 16; off += 16) {
    const oldHash = h.slice(0)
    for (let t = 0; t < 16; t++) w[t] = words[off + t] | 0
    for (let t = 16; t < 64; t++) {
      const w15 = w[t - 15], w2 = w[t - 2]
      w[t] = (w[t - 16]
        + (rightRotate(w15, 7) ^ rightRotate(w15, 18) ^ (w15 >>> 3))
        + w[t - 7]
        + (rightRotate(w2, 17) ^ rightRotate(w2, 19) ^ (w2 >>> 10))) | 0
    }
    let h0 = h[0], h1 = h[1], h2 = h[2], h3 = h[3], h4 = h[4], h5 = h[5], h6 = h[6], h7 = h[7]
    for (let t = 0; t < 64; t++) {
      const temp1 = (h7
        + (rightRotate(h4, 6) ^ rightRotate(h4, 11) ^ rightRotate(h4, 25))
        + ((h4 & h5) ^ (~h4 & h6)) + k[t] + w[t]) | 0
      const temp2 = ((rightRotate(h0, 2) ^ rightRotate(h0, 13) ^ rightRotate(h0, 22))
        + ((h0 & h1) ^ (h0 & h2) ^ (h1 & h2))) | 0
      h7 = h6; h6 = h5; h5 = h4; h4 = (h3 + temp1) | 0; h3 = h2; h2 = h1; h1 = h0; h0 = (temp1 + temp2) | 0
    }
    h[0] = (h0 + oldHash[0]) | 0; h[1] = (h1 + oldHash[1]) | 0; h[2] = (h2 + oldHash[2]) | 0; h[3] = (h3 + oldHash[3]) | 0
    h[4] = (h4 + oldHash[4]) | 0; h[5] = (h5 + oldHash[5]) | 0; h[6] = (h6 + oldHash[6]) | 0; h[7] = (h7 + oldHash[7]) | 0
  }
  const HEX = '0123456789abcdef'
  let out = ''
  for (let i = 0; i < 8; i++) {
    for (let j = 3; j >= 0; j--) {
      const b = (h[i] >>> (j * 8)) & 255
      out += HEX[b >> 4] + HEX[b & 0xf]
    }
  }
  return out
}

/**
 * 计算 File 的 SHA-256 哈希（十六进制）。
 * 优先使用 Web Crypto API（crypto.subtle）；非安全上下文（HTTP 内网）下退回纯 JS 实现。
 */
export async function calculateSHA256(file: File): Promise<string> {
  const buffer = await file.arrayBuffer()
  if (crypto?.subtle) {
    const digest = await crypto.subtle.digest('SHA-256', buffer)
    return Array.from(new Uint8Array(digest))
      .map(byte => byte.toString(16).padStart(2, '0'))
      .join('')
  }
  return sha256Hex(new Uint8Array(buffer))
}

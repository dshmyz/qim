export function countAliasWeight(name: string): number {
  let weight = 0
  for (const char of name) {
    const code = char.codePointAt(0) ?? 0
    if (
      (code >= 0x4E00 && code <= 0x9FFF) ||
      (code >= 0x3400 && code <= 0x4DBF) ||
      (code >= 0x20000 && code <= 0x2A6DF) ||
      (code >= 0xF900 && code <= 0xFAFF)
    ) {
      weight += 2
    } else {
      weight += 1
    }
  }
  return weight
}

const MAX_ALIAS_WEIGHT = 8

export function validateAliasName(name: string): { valid: boolean; message: string } {
  if (!name || name.trim().length === 0) {
    return { valid: false, message: '名称不能为空' }
  }

  const weight = countAliasWeight(name.trim())
  if (weight > MAX_ALIAS_WEIGHT) {
    return { valid: false, message: '别名最多允许4个中文字或8个字母' }
  }

  return { valid: true, message: '' }
}
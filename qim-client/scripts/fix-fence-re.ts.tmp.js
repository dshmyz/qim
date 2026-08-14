const fs = require('fs')
const p = 'qim-client/src/composables/useMarkdownRender.ts'
let s = fs.readFileSync(p, 'utf8')

const E004 = ''
const E005 = ''
const oldSeg = `let finalHtml = emojiHtml.replace(/${E004}(\\d+)${E005}/g, (_: string, n: string) => {`
const newSeg = `let finalHtml = emojiHtml.replace(/<p>${E004}(\\d+)${E005}<\\/p>/g, (_: string, n: string) => {`
if (!s.includes(oldSeg)) {
  console.error('OLD NOT FOUND; lines with E004:')
  s.split('\n').forEach((line, i) => {
    if (line.includes(E004) || line.includes(E005)) console.log(i + 1, JSON.stringify(line))
  })
  process.exit(1)
}
s = s.replace(oldSeg, newSeg)
s = s.replace(
  '  // 围栏 code 原样恢复为 <pre><code>（内容转义，绕过 marked 的围栏解析但保持等价渲染）',
  '  // 围栏 code 原样恢复为 <pre><code>（内容转义；marked 会把占位符包进 <p>，\n  // 连同包裹一并替换，避免留下空 <p> 的排版空隙）'
)
fs.writeFileSync(p, s)
console.log('OK')

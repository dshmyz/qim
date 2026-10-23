// 构建门禁：检测 dist/assets 下 Vite 产物 chunk 间的静态 import 循环。
// chunk 级循环依赖本身不一定炸，但环内任何模块顶层跨 chunk 取值（const 不提升、
// SFC 编译产物、logger 这类单例初始化）都会在打包后报
// "Cannot access 't' before initialization"，且 dev 模式不分包永远复现不了。
// 因此有环即 fail（exit 1），把静默地雷变成构建报错。
// 背景：2026-10 v2.0.44 线上 TDZ，根因 chat↔group↔settings↔ai↔calendar↔file
// 六 chunk 互相成环，见 memory/vite-config-js-shadow-chunk-cycle-tdz.md
const fs = require('fs')
const path = require('path')

const dir = process.argv[2]
  ? path.resolve(process.argv[2])
  : path.resolve(__dirname, '..', 'dist', 'assets')
if (!fs.existsSync(dir)) {
  console.error('[chunk-cycle-guard] 未找到 dist/assets，请先执行 vite build')
  process.exit(1)
}

const files = fs.readdirSync(dir).filter((f) => f.endsWith('.js'))
const graph = {}
for (const f of files) {
  const src = fs.readFileSync(path.join(dir, f), 'utf8')
  const deps = []
  // 只看静态 import/export-from（动态 import() 不参与初始化顺序，不构成 TDZ 风险）
  const re = /(?:import|export)\s*[^;]*?from\s*"\.\/([^"]+\.js)"/g
  let m
  while ((m = re.exec(src))) deps.push(m[1])
  graph[f] = deps
}

const cycles = []
const state = {} // 1=在栈中 2=已完成
const stack = []
function dfs(n) {
  state[n] = 1
  stack.push(n)
  for (const d of graph[n] || []) {
    if (state[d] === 1) cycles.push([...stack.slice(stack.indexOf(d)), d])
    else if (!state[d]) dfs(d)
  }
  stack.pop()
  state[n] = 2
}
for (const f of files) if (!state[f]) dfs(f)

if (cycles.length > 0) {
  console.error(`[chunk-cycle-guard] 检测到 ${cycles.length} 条 chunk 循环依赖，打包后存在 TDZ 风险，构建失败：`)
  for (const c of cycles.slice(0, 20)) console.error('  CYCLE:', c.join(' -> '))
  console.error('修复方式：把成环的 manualChunks 规则合并为一个 chunk，或消除模块顶层的跨 chunk 副作用。')
  process.exit(1)
}

console.log(`[chunk-cycle-guard] ${files.length} 个 chunk，无循环依赖 ✓`)

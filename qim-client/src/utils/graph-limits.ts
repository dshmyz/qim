// 前端兜底边预算，与后端 groupGraphEdgesPerNode 对齐：
// 服务端已按 max_nodes × 4 截断，这里再防一手旧接口/异常数据导致的边爆炸。
export const graphEdgeBudget = (nodeCount: number) => Math.max(120, nodeCount * 4)

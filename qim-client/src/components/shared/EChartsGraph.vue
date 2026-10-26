<template>
  <div ref="elRef" class="echarts-graph"></div>
</template>

<script setup lang="ts">
// EChartsGraph — 统一的知识图谱渲染组件（client / admin 各有一份，接口一致）。
// 只负责「ECharts 力导向渲染 + 生命周期 + 节点点击冒泡」；数据获取、契约归一化、
// 详情展示由宿主组件各自实现。懒加载 echarts，只在真正画图时才拉取，避免打进首屏。
import { ref, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'

export interface GraphNode {
  id: string
  label: string
  type: string
  count?: number
  data?: Record<string, any>
}

export interface GraphEdge {
  source: string
  target: string
  weight?: number
  label?: string
}

const props = withDefaults(defineProps<{
  nodes: GraphNode[]
  edges?: GraphEdge[]
  // 节点 type → 渐变 [浅色, 深色]。默认覆盖各站点常见类型。
  typeColors?: Record<string, [string, string]>
  // 是否显示图例（分身显示实体/主题；群助手单类可不显示）
  showLegend?: boolean
  // 是否在边上显示 label（管理后台的边带 label）
  showEdgeLabel?: boolean
  // 节点大小是否按 count 缩放
  sizeByCount?: boolean
  // 节点数超过该值就关闭常显标签。0 = 不限制：标签是 admin/分身图谱的主要阅读路径，
  // 只有群图谱（节点多且密）才传 40 做减负。
  maxLabeledNodes?: number
  // 前端兜底边预算。0 = 不限制：只有后端已按预算截断、又需防旧接口爆边的场景才传。
  maxEdges?: number
}>(), {
  edges: () => [],
  typeColors: () => ({
    entity: ['#ffd08a', '#f97316'], // 橙 · 实体
    theme: ['#c4b5fd', '#8b5cf6'],  // 紫 · 主题
    knowledge: ['#5eead4', '#14b8a6'], // 青 · 知识
    query: ['#ffd08a', '#f97316'],   // 橙 · 查询
    note: ['#5eead4', '#14b8a6'],    // 青 · 笔记片段
  }),
  showLegend: false,
  showEdgeLabel: false,
  sizeByCount: false,
  maxLabeledNodes: 0,
  maxEdges: 0,
})

const emit = defineEmits<{
  (e: 'node-click', node: GraphNode): void
  (e: 'blank-click'): void
}>()

const elRef = ref<HTMLDivElement | null>(null)
let chart: any = null
let echartsLib: any = null
let renderSeq = 0

const BASE_SIZE = 22 // 节点基础直径（sizeByCount=false 时用）
const MAX_COUNT_SCALE = 4 // count 驱动大小时的放大档位
// 节点过多时关闭力导向逐帧动画（每帧 O(n²) 且不收敛）。与标签显隐解耦：
// 标签策略由宿主的 maxLabeledNodes 决定，动画阈值仍按规模兜底。
const MAX_ANIMATED_NODES = 40

function colorsFor(type: string): [string, string] {
  return props.typeColors[type] || props.typeColors.knowledge || ['#5eead4', '#14b8a6']
}

function legendData(): { name: string; itemStyle: { color: string } }[] {
  if (!props.showLegend) return []
  return Object.entries(props.typeColors).map(([type, [_, deep]]) => ({
    name: type,
    itemStyle: { color: deep },
  }))
}

async function ensureECharts(): Promise<any> {
  if (echartsLib) return echartsLib
  const mod = await import('echarts')
  echartsLib = (mod as any).default ?? mod
  return echartsLib
}

function buildOption() {
  const totalNodes = props.nodes.length
  // 标签是否常显由宿主决定（maxLabeledNodes=0 表示不限制）。
  const showAllLabels = props.maxLabeledNodes <= 0 || totalNodes <= props.maxLabeledNodes
  // 小图保留力导向动画（有机收拢的观感几乎零成本）；大图一次性布局，
  // 关动画避免"每帧 O(n²) + 永不收敛"持续吃满主线程。
  const animate = totalNodes <= MAX_ANIMATED_NODES
  const nodes = props.nodes.map((n) => {
    const size = props.sizeByCount && n.count
      ? BASE_SIZE + Math.min(Math.max(n.count, 1), MAX_COUNT_SCALE) * 5
      : BASE_SIZE
    // ECharts graph 系列的 links[].source/target 按「节点 name」解析（而非 id）。
    // 各数据源（分身文档/群文档）边都引用节点 id（如 doc:1 / entity:xxx），
    // 因此这里用 id 作为 name 保证边能解析；真正展示的文字走 label.formatter 用 _raw.label。
    const nodeId = String(n.id ?? n.label ?? '')
    return {
      id: n.id,
      name: nodeId,
      type: n.type,
      count: n.count ?? 1,
      symbolSize: size,
      // 纯色 + 无阴影：力导向每帧重绘时，逐节点渐变与 shadowBlur 是主要瓶颈。
      itemStyle: {
        color: colorsFor(n.type)[1],
        borderColor: '#fff',
        borderWidth: 1,
      },
      _raw: n,
    }
  })

  const known = new Set(nodes.map((n) => n.name))
  // 预算由宿主传（0=不限制）：群图谱后端已按 max_nodes×4 截断，这里再防一手旧接口；
  // 分身图谱后端不截边，传预算会静默丢边、数字与画面不符，故不传。
  const edgeLimit = props.maxEdges > 0 ? props.maxEdges : Number.POSITIVE_INFINITY
  const links = (props.edges || [])
    .filter((e) => known.has(String(e.source)) && known.has(String(e.target)))
    .slice(0, edgeLimit)
    .map((e) => {
      const weight = e.weight || 1
      const link: Record<string, any> = {
        source: e.source,
        target: e.target,
        weight,
        lineStyle: {
          width: Math.max(1.2, Math.min(1.2 + weight, 3.5)),
          opacity: 0.6,
          curveness: 0.05,
          color: '#8f6fa6',
        },
      }
      if (props.showEdgeLabel && e.label) {
        link.label = { show: true, color: '#7c8698', fontSize: 10 }
      }
      return link
    })

  return {
    backgroundColor: 'transparent',
    animation: animate,
    tooltip: {
      trigger: 'item',
      confine: true,
      formatter: (p: any) => {
        if (p.dataType === 'edge') return ''
        const countStr = props.sizeByCount ? ` · 出现 ${p.data.count ?? 1} 处` : ''
        const label = p.data?._raw?.label ?? p.data.name
        return `<b>${label}</b>${countStr}`
      },
    },
    legend: props.showLegend ? [{ data: legendData(), top: 8, left: 'center', textStyle: { fontSize: 12, color: '#666' } }] : undefined,
    series: [
      {
        type: 'graph',
        layout: 'force',
        roam: true,
        draggable: true,
        categories: legendData().map((d) => ({ name: d.name })),
        data: nodes,
        links,
        force: {
          repulsion: 320,
          edgeLength: 120,
          gravity: 0.1,
          friction: 0.6,
          // 大图关闭逐帧布局动画（每帧 O(n²) 永不收敛）；小图保留收拢过程。
          layoutAnimation: animate,
        },
        emphasis: {
          focus: 'adjacency',
          label: { show: true },
          lineStyle: { width: 3, opacity: 1 },
        },
        label: {
          show: showAllLabels,
          fontSize: 12,
          color: '#333',
          position: 'bottom',
          distance: 4,
          // name 已是 id（用于边解析），显示文案用可读的 _raw.label
          formatter: (p: any) => p.data?._raw?.label ?? p.data.name ?? '',
        },
        lineStyle: {
          curveness: 0.05,
          color: '#8f6fa6',
        },
        edgeSymbol: ['none', 'none'],
      },
    ],
  }
}

async function render() {
  if (!elRef.value) return
  const seq = ++renderSeq
  let echarts: any
  try {
    echarts = await ensureECharts()
  } catch (e) {
    console.error('加载图谱引擎失败', e)
    return
  }
  // 并发重绘保护：等待 echarts 加载期间若 props 又变了，以最后一次为准。
  if (seq !== renderSeq || !elRef.value) return

  if (!chart) {
    chart = echarts.init(elRef.value)
    chart.on('click', (params: any) => {
      if (params.dataType === 'node' && params.data?._raw) {
        emit('node-click', params.data._raw as GraphNode)
      } else {
        emit('blank-click')
      }
    })
  }
  // 复用实例 + notMerge：避免每次刷新 dispose/init 造成的整图重建与实例泄漏。
  chart.setOption(buildOption(), true)
}

function disposeChart() {
  if (chart) {
    chart.dispose && chart.dispose()
    chart = null
  }
}

// 容器尺寸变化时同步画布（如右侧详情卡片出现/收起改变容器宽度）。
// 防抖：拖拽窗口/过渡动画期间会连续触发，逐帧 resize 会叠加在布局计算上。
let resizeTimer: ReturnType<typeof setTimeout> | null = null
function handleResize() {
  if (resizeTimer) clearTimeout(resizeTimer)
  resizeTimer = setTimeout(() => {
    resizeTimer = null
    if (!chart || !elRef.value) return
    if (elRef.value.clientWidth <= 0 || elRef.value.clientHeight <= 0) return
    chart.resize && chart.resize()
  }, 150)
}

let ro: ResizeObserver | null = null

watch(
  () => [props.nodes, props.edges, props.showLegend, props.showEdgeLabel, props.sizeByCount, props.typeColors],
  () => {
    // 等宿主把新数据绑定到页面后再重画
    nextTick(render)
  },
  { deep: false }
)

onMounted(() => {
  render()
  if (typeof ResizeObserver !== 'undefined' && elRef.value) {
    ro = new ResizeObserver(handleResize)
    ro.observe(elRef.value)
  }
})

onBeforeUnmount(() => {
  if (ro) ro.disconnect()
  if (resizeTimer) clearTimeout(resizeTimer)
  renderSeq++
  disposeChart()
})
</script>

<style scoped>
.echarts-graph {
  width: 100%;
  height: 100%;
  min-height: 360px;
}
.echarts-graph :deep(canvas) {
  display: block;
}
</style>

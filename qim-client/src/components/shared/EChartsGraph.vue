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
})

const emit = defineEmits<{
  (e: 'node-click', node: GraphNode): void
  (e: 'blank-click'): void
}>()

const elRef = ref<HTMLDivElement | null>(null)
let chart: any = null

const BASE_SIZE = 22 // 节点基础直径（sizeByCount=false 时用）
const MAX_COUNT_SCALE = 4 // count 驱动大小时的放大档位

function gradientFor(echarts: any, type: string): any {
  const colors = props.typeColors[type] || props.typeColors.knowledge || ['#5eead4', '#14b8a6']
  return new (echarts.graphic as any).LinearGradient(0, 0, 0, 1, [
    { offset: 0, color: colors[0] },
    { offset: 1, color: colors[1] },
  ])
}

function legendData(): { name: string; itemStyle: { color: string } }[] {
  if (!props.showLegend) return []
  return Object.entries(props.typeColors).map(([type, [_, deep]]) => ({
    name: type,
    itemStyle: { color: deep },
  }))
}

async function render() {
  if (!elRef.value) return
  disposeChart()

  // 懒加载 echarts：Vite 把 echarts(+zrender) 拆成独立 chunk，仅在渲染时拉取
  let echarts: any
  try {
    const mod = await import('echarts')
    // echarts ESM 暴露具名导出；个别构建下也有 default 入口，取其一即可
    echarts = (mod as any).default ?? mod
  } catch (e) {
    console.error('加载图谱引擎失败', e)
    return
  }

  const totalNodes = props.nodes.length
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
      itemStyle: {
        color: gradientFor(echarts, n.type),
        borderColor: '#fff',
        borderWidth: 1.5,
        shadowBlur: 12,
        shadowColor: 'rgba(0,0,0,0.25)',
      },
      _raw: n,
    }
  })

  const links = (props.edges || []).map((e) => {
    const weight = e.weight || 1
    const link: Record<string, any> = {
      source: e.source,
      target: e.target,
      weight,
      lineStyle: {
        width: Math.max(1.6, Math.min(1.6 + weight * 1.5, 4.5)),
        opacity: Math.max(0.55, Math.min(0.55 + weight * 0.2, 0.95)),
        curveness: 0.05,
        color: '#8f6fa6',
      },
    }
    if (props.showEdgeLabel && e.label) {
      link.label = { show: true, color: '#7c8698', fontSize: 10 }
    }
    return link
  })

  chart = echarts.init(elRef.value)
  const option = {
    backgroundColor: 'transparent',
    animationDuration: 600,
    animationEasingUpdate: 'quinticInOut',
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
          layoutAnimation: true,
        },
        emphasis: {
          focus: 'adjacency',
          lineStyle: { width: 3, opacity: 1 },
          itemStyle: { shadowBlur: 20, shadowColor: 'rgba(0,0,0,0.35)' },
        },
        label: {
          show: true,
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
  chart.setOption(option)

  chart.on('click', (params: any) => {
    if (params.dataType === 'node' && params.data?._raw) {
      emit('node-click', params.data._raw as GraphNode)
    } else {
      emit('blank-click')
    }
  })

  // totalNodes===0 时也画一个空容器，宿主用 v-else 控制空态展示即可
  void totalNodes
}

function disposeChart() {
  if (chart) {
    chart.dispose && chart.dispose()
    chart = null
  }
}

watch(
  () => [props.nodes, props.edges, props.showLegend, props.showEdgeLabel, props.sizeByCount, props.typeColors],
  () => {
    // 等宿主把新数据绑定到页面后再重画
    nextTick(render)
  },
  { deep: false }
)

onMounted(render)

onBeforeUnmount(disposeChart)
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

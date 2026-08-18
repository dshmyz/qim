<template>
  <div ref="chartRef" :style="{ width: width, height: height }"></div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, onUnmounted } from 'vue'
import * as echarts from 'echarts'

interface Props {
  data: Array<{ name: string; value: number }>
  width?: string
  height?: string
}

interface ChartItem {
  name: string
  value: number
}

const props = withDefaults(defineProps<Props>(), {
  width: '100%',
  height: '300px'
})

const emit = defineEmits<{
  'item-click': [item: ChartItem]
}>()

const chartRef = ref<HTMLDivElement>()
let chart: echarts.ECharts | null = null

onMounted(() => {
  initChart()
})

onUnmounted(() => {
  chart?.dispose()
})

watch(() => props.data, () => {
  updateChart()
}, { deep: true })

function initChart() {
  if (!chartRef.value) return

  chart = echarts.init(chartRef.value)
  // 点击扇区回调；chart.on 在 setOption 之间持续生效，只需注册一次
  chart.on('click', (params: any) => {
    emit('item-click', { name: params.name, value: params.value })
  })
  updateChart()
}

function updateChart() {
  if (!chart) return

  const option = {
    tooltip: {
      trigger: 'item',
      formatter: '{a} <br/>{b}: {c} ({d}%)'
    },
    legend: {
      orient: 'vertical',
      left: 'left'
    },
    series: [
      {
        name: '文件类型',
        type: 'pie',
        radius: ['40%', '70%'],
        avoidLabelOverlap: false,
        label: {
          show: false,
          position: 'center'
        },
        emphasis: {
          label: {
            show: true,
            fontSize: '20',
            fontWeight: 'bold'
          }
        },
        labelLine: {
          show: false
        },
        data: props.data
      }
    ]
  }

  chart.setOption(option)
}
</script>

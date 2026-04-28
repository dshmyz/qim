<template>
  <div ref="chartRef" :style="{ width: width, height: height }"></div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, onUnmounted } from 'vue'
import * as echarts from 'echarts'

interface Props {
  xAxisData: string[]
  seriesData: Array<{
    name: string
    data: number[]
    color?: string
  }>
  width?: string
  height?: string
}

const props = withDefaults(defineProps<Props>(), {
  width: '100%',
  height: '300px'
})

const chartRef = ref<HTMLDivElement>()
let chart: echarts.ECharts | null = null

onMounted(() => {
  initChart()
})

onUnmounted(() => {
  chart?.dispose()
})

watch([() => props.xAxisData, () => props.seriesData], () => {
  updateChart()
}, { deep: true })

function initChart() {
  if (!chartRef.value) return
  
  chart = echarts.init(chartRef.value)
  updateChart()
}

function updateChart() {
  if (!chart) return
  
  const option = {
    tooltip: {
      trigger: 'axis'
    },
    legend: {
      data: props.seriesData.map(s => s.name)
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: props.xAxisData
    },
    yAxis: {
      type: 'value',
      axisLabel: {
        formatter: '{value}%'
      }
    },
    series: props.seriesData.map(s => ({
      name: s.name,
      type: 'line',
      smooth: true,
      data: s.data,
      itemStyle: {
        color: s.color
      }
    }))
  }
  
  chart.setOption(option)
}
</script>

<template>
  <div class="file-storage">
    <el-row :gutter="20">
      <el-col :span="8">
        <el-card>
          <template #header>
            <span>总容量</span>
          </template>
          <div class="stat-value">
            {{ formatSize(fileStore.statistics?.totalSize) }}
          </div>
        </el-card>
      </el-col>

      <el-col :span="8">
        <el-card>
          <template #header>
            <span>已用容量</span>
          </template>
          <div class="stat-value">
            {{ formatSize(fileStore.statistics?.usedSize) }}
          </div>
          <el-progress
            :percentage="usedPercentage"
            :color="progressColor"
          />
        </el-card>
      </el-col>

      <el-col :span="8">
        <el-card>
          <template #header>
            <span>文件数量</span>
          </template>
          <div class="stat-value">
            {{ fileStore.statistics?.fileCount || 0 }}
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="12">
        <el-card>
          <template #header>
            <span>文件类型分布</span>
          </template>
          <PieChart
            v-if="chartData.length > 0"
            :data="chartData"
            height="350px"
          />
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>大文件排行</span>
              <el-button type="primary" link @click="loadMoreLargeFiles">
                查看更多
              </el-button>
            </div>
          </template>

          <el-table
            v-loading="fileStore.loading"
            :data="fileStore.largeFiles"
            border
            stripe
          >
            <el-table-column prop="fileName" label="文件名" show-overflow-tooltip />
            <el-table-column prop="fileSize" label="大小" width="120">
              <template #default="{ row }">
                {{ formatSize(row.fileSize) }}
              </template>
            </el-table-column>
            <el-table-column prop="uploaderName" label="上传者" width="120" />
            <el-table-column prop="createdAt" label="上传时间" width="180">
              <template #default="{ row }">
                {{ formatTime(row.createdAt) }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useFileStore } from '@/stores/file'
import PieChart from '@/components/charts/PieChart.vue'

const fileStore = useFileStore()

const usedPercentage = computed(() => {
  if (!fileStore.statistics) return 0
  return Math.round((fileStore.statistics.usedSize / fileStore.statistics.totalSize) * 100)
})

const progressColor = computed(() => {
  if (usedPercentage.value > 80) return '#f56c6c'
  if (usedPercentage.value > 60) return '#e6a23c'
  return '#67c23a'
})

const chartData = computed(() => {
  if (!fileStore.statistics?.sizeByType) return []

  const typeNames: Record<string, string> = {
    image: '图片',
    document: '文档',
    video: '视频',
    audio: '音频',
    other: '其他'
  }

  return fileStore.statistics.sizeByType.map(item => ({
    name: typeNames[item.type] || item.type,
    value: item.size
  }))
})

onMounted(() => {
  fileStore.loadStatistics()
  fileStore.loadLargeFiles(10)
})

function loadMoreLargeFiles() {
  fileStore.loadLargeFiles(50)
}

function formatSize(size?: number) {
  if (!size) return '0 B'

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let index = 0
  let value = size

  while (value >= 1024 && index < units.length - 1) {
    value /= 1024
    index++
  }

  return `${value.toFixed(2)} ${units[index]}`
}

function formatTime(time?: string) {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}
</script>

<style scoped>
.file-storage {
  padding: 20px;
}

.stat-value {
  font-size: 32px;
  font-weight: bold;
  color: #409eff;
  text-align: center;
  margin: 20px 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>

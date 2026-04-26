<template>
  <div class="base-table-layout">
    <el-card shadow="never">
      <div v-if="$slots.search" class="search-bar">
        <slot name="search"></slot>
        <el-button type="success" @click="$emit('create')">
          {{ createButtonText }}
        </el-button>
      </div>
      <el-table :data="data" v-loading="loading">
        <slot></slot>
      </el-table>
      <div class="pagination-container" v-if="showPagination">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="pageSizes"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="$emit('page-change')"
          @current-change="$emit('page-change')"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
interface Props {
  data: any[]
  loading: boolean
  total: number
  currentPage: number
  pageSize: number
  createButtonText?: string
  showPagination?: boolean
  pageSizes?: number[]
}

withDefaults(defineProps<Props>(), {
  createButtonText: '创建',
  showPagination: true,
  pageSizes: () => [10, 20, 50, 100]
})

defineEmits<{
  'page-change': []
  'create': []
}>()
</script>

<style scoped>
.base-table-layout {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}
.search-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: var(--space-4);
  flex-wrap: wrap;
  gap: var(--space-3);
}
.pagination-container {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}
</style>

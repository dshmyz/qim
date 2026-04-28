<!-- src/components/data/DataTable.vue -->
<template>
  <div class="data-table">
    <div class="table-header">
      <div class="search-area">
        <slot name="search"></slot>
      </div>
      <div class="actions-area">
        <slot name="actions"></slot>
        <el-button type="primary" @click="$emit('refresh')">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <el-table :data="data" v-loading="loading" stripe>
      <slot></slot>
    </el-table>

    <div class="pagination-area" v-if="showPagination">
      <el-pagination
        :current-page="pagination.page"
        :page-size="pagination.pageSize"
        :total="total"
        :page-sizes="pageSizes"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="$emit('page-change', $event)"
        @current-change="$emit('page-change', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Refresh } from '@element-plus/icons-vue'

interface PaginationConfig {
  page: number
  pageSize: number
  total: number
}

interface Props {
  data: unknown[]
  loading: boolean
  pagination: PaginationConfig
  showPagination?: boolean
  pageSizes?: number[]
}

const props = withDefaults(defineProps<Props>(), {
  showPagination: true,
  pageSizes: () => [10, 20, 50, 100]
})

defineEmits<{
  'page-change': [value: number]
  'refresh': []
}>()

const total = computed(() => props.pagination.total)
</script>

<style scoped>
.data-table {
  background: var(--color-surface);
  border-radius: var(--radius-xl);
  padding: var(--space-6);
  box-shadow: var(--shadow-card);
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: var(--space-5);
  gap: var(--space-4);
  flex-wrap: wrap;
}

.search-area {
  flex: 1;
  min-width: 0;
}

.actions-area {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  flex-shrink: 0;
}

.pagination-area {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}
</style>

<!-- src/components/data/SearchForm.vue -->
<template>
  <el-form :model="{}" inline>
    <slot></slot>
    <el-form-item>
      <el-button type="primary" @click="$emit('search')">搜索</el-button>
      <el-button @click="$emit('reset')">重置</el-button>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
defineEmits<{
  'search': []
  'reset': []
}>()
</script>

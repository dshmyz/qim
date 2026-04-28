<!-- src/components/layout/Breadcrumb/index.vue -->
<template>
  <el-breadcrumb separator="/">
    <el-breadcrumb-item :to="{ path: '/' }">首页</el-breadcrumb-item>
    <el-breadcrumb-item>{{ title }}</el-breadcrumb-item>
  </el-breadcrumb>
</template>

<script setup lang="ts">
defineProps<{
  title: string
}>()
</script>

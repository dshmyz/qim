<template>
  <div class="forbidden-page">
    <div class="forbidden-content">
      <h1 class="error-code">403</h1>
      <h2 class="error-title">权限不足</h2>
      <p class="error-message">您没有访问此页面的权限。请联系管理员获取相应权限。</p>
      <el-button type="primary" @click="router.push('/')">返回首页</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
const router = useRouter()
</script>

<style scoped>
.forbidden-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: var(--color-bg-page);
}
.forbidden-content {
  text-align: center;
}
.error-code {
  font-size: 96px;
  font-weight: 900;
  background: var(--gradient-primary);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  margin: 0;
}
.error-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin: var(--space-4) 0 var(--space-2);
}
.error-message {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-bottom: var(--space-6);
}
</style>

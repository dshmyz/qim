---
layout: false
---

<script setup>
import Home from './components/Home.vue'
</script>

<Home />

<style>
/* layout: false 下无 VitePress Layout 包装，Home 直接渲染 */
html, body, #app {
  margin: 0;
  padding: 0;
  width: 100%;
  min-height: 100vh;
  background: #fafbfc;
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

/* VitePress 全局样式覆盖：仅修正必要的属性 */
.landing-page img {
  max-width: none !important;
}
.landing-page button {
  font-family: inherit !important;
}
</style>

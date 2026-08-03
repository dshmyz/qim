---
layout: home

hero: false
sidebar: false
aside: false
outline: false
---

<script setup>
import Home from './components/Home.vue'
</script>

<Home />

<style>
/* 隐藏 VitePress 默认布局，让 Home 组件全屏（只影响首页） */
.VPHome {
  min-height: 100vh !important;
  padding: 0 !important;
  margin-bottom: 0 !important;
}
.Layout {
  min-height: 100vh !important;
}
/* 取消 VitePress 默认内容容器的宽度限制（只影响首页，文档页需要保留 padding 给 sidebar 让位） */
.VPHome .vp-doc.container {
  max-width: 100% !important;
  padding: 0 !important;
}
/* 隐藏默认 hero */
.VPHero {
  display: none !important;
}
/* 隐藏 VitePress 默认导航栏，首页使用 Home 组件自带的 top-nav */
.VPNav {
  display: none !important;
}
/* 隐藏默认 features */
.VPFeatures {
  display: none !important;
}
/* 隐藏 VitePress 默认 footer，使用 Home 组件自带的 footer */
.VPFooter {
  display: none !important;
}
</style>

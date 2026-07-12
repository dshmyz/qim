import { watch, nextTick, type Ref } from 'vue'
import hljs from 'highlight.js'
// 引入浅色主题（用户偏好浅色，不使用暗色主题）
import 'highlight.js/styles/github.css'

/**
 * 监听容器内 markdown 渲染结果，对 pre>code 元素应用 highlight.js 语法高亮。
 * 已高亮的元素不会重复处理。
 * @param containerRef 容器元素引用
 * @param trigger 触发重新高亮的响应式值（通常是渲染后的 HTML 字符串）
 */
export function useCodeHighlight(
  containerRef: Ref<HTMLElement | null>,
  trigger: Ref<string>
): void {
  watch(
    trigger,
    () => {
      nextTick(() => {
        if (!containerRef.value) return
        const blocks = containerRef.value.querySelectorAll<HTMLElement>('pre code')
        blocks.forEach((el) => {
          if (el.dataset.highlighted) return
          try {
            hljs.highlightElement(el)
            el.dataset.highlighted = 'true'
          } catch {
            // 忽略无法识别的语言，保持原始文本
          }
        })
      })
    },
    { immediate: true }
  )
}

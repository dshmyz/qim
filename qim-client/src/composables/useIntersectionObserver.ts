import { onMounted, onUnmounted, ref, type Ref } from 'vue'

export function useIntersectionObserver(
  targetRef: Ref<HTMLElement | null>,
  options?: IntersectionObserverInit
) {
  const isVisible = ref(false)
  let observer: IntersectionObserver | null = null

  onMounted(() => {
    if (!targetRef.value) return

    observer = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) {
          isVisible.value = true
          if (observer && targetRef.value) {
            observer.unobserve(targetRef.value)
          }
        }
      }
    }, {
      rootMargin: '200px 0px',
      threshold: 0,
      ...options,
    })

    observer.observe(targetRef.value)
  })

  onUnmounted(() => {
    if (observer && targetRef.value) {
      observer.unobserve(targetRef.value)
    }
    observer?.disconnect()
    observer = null
  })

  return { isVisible }
}
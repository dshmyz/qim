import { describe, expect, it, vi, beforeEach } from 'vitest'
import { ref, nextTick } from 'vue'
import { useCodeHighlight } from '@/composables/useCodeHighlight'

vi.mock('highlight.js', () => ({
  default: {
    highlightElement: vi.fn((el: HTMLElement) => {
      el.classList.add('hljs')
      el.setAttribute('data-highlighted', 'true')
    }),
  },
}))

import hljs from 'highlight.js'

describe('useCodeHighlight', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('highlights pre>code elements when trigger changes', async () => {
    const container = document.createElement('div')
    const codeEl = document.createElement('code')
    codeEl.textContent = 'const x = 1'
    const pre = document.createElement('pre')
    pre.appendChild(codeEl)
    container.appendChild(pre)

    const containerRef = ref<HTMLElement | null>(container)
    const trigger = ref('initial')

    useCodeHighlight(containerRef, trigger)

    trigger.value = 'updated'
    await nextTick()
    await nextTick()

    expect(hljs.highlightElement).toHaveBeenCalled()
    expect(codeEl.getAttribute('data-highlighted')).toBe('true')
  })

  it('does not re-highlight already highlighted elements', async () => {
    const container = document.createElement('div')
    const codeEl = document.createElement('code')
    codeEl.textContent = 'const x = 1'
    codeEl.setAttribute('data-highlighted', 'true')
    const pre = document.createElement('pre')
    pre.appendChild(codeEl)
    container.appendChild(pre)

    const containerRef = ref<HTMLElement | null>(container)
    const trigger = ref('initial')

    useCodeHighlight(containerRef, trigger)

    trigger.value = 'updated'
    await nextTick()
    await nextTick()

    expect(hljs.highlightElement).not.toHaveBeenCalled()
  })

  it('does nothing when container is null', async () => {
    const containerRef = ref<HTMLElement | null>(null)
    const trigger = ref('initial')

    useCodeHighlight(containerRef, trigger)

    trigger.value = 'updated'
    await nextTick()
    await nextTick()

    expect(hljs.highlightElement).not.toHaveBeenCalled()
  })
})

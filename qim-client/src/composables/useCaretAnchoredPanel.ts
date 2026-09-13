import { ref, watch, nextTick, onBeforeUnmount, type Ref, type CSSProperties } from 'vue'

/**
 * 光标锚定浮层（IDE 式补全面板）通用组合式函数。
 *
 * 水平贴光标 x（镜像元素量测），垂直优先弹字形下方，放不下完整面板则翻转
 * 到字形上方，上下都放不下才收缩；绝对定位悬浮，不参与文档流，不影响工具栏等布局。
 * 斜杠命令面板与 @ 成员面板共用。
 */

export interface FloatingPanelStyle { top: string; left: string; transform?: string }

/** 面板贴字形盒的间距 */
const PANEL_GAP = 2
/** 下方空间小于此值才收缩（约 头部 + 两条候选项 的最低可用高度） */
const PANEL_MIN_SPACE = 130
/** 面板距窗口边缘的安全边距 */
const PANEL_VIEWPORT_MARGIN = 12
/** 面板与输入区右缘的最小留白 */
const PANEL_RIGHT_MARGIN = 16

/** 用镜像元素测量光标相对输入区的位置（left / 行框与字形盒的 top、bottom） */
const measureCaretRectInArea = (ta: HTMLTextAreaElement, area: HTMLElement) => {
  const cs = getComputedStyle(ta)
  const mirror = document.createElement('div')
  mirror.style.position = 'absolute'
  mirror.style.visibility = 'hidden'
  mirror.style.whiteSpace = 'pre-wrap'
  mirror.style.overflowWrap = 'break-word'
  mirror.style.width = `${ta.clientWidth - parseFloat(cs.paddingLeft) - parseFloat(cs.paddingRight)}px`
  mirror.style.fontFamily = cs.fontFamily
  mirror.style.fontSize = cs.fontSize
  mirror.style.fontWeight = cs.fontWeight
  mirror.style.lineHeight = cs.lineHeight
  mirror.style.letterSpacing = cs.letterSpacing
  mirror.style.tabSize = cs.tabSize
  mirror.textContent = ta.value.slice(0, ta.selectionStart ?? ta.value.length)
  const marker = document.createElement('span')
  marker.textContent = 'Xj'
  mirror.appendChild(marker)
  document.body.appendChild(mirror)
  const caretX = marker.offsetLeft
  const caretLineTop = marker.offsetTop
  const lineHeight = parseFloat(cs.lineHeight) || parseFloat(cs.fontSize) * 1.4
  // 量出文字字形实际边界盒（相对镜像内容原点），面板贴字形而非行框，避免半行空隙显得靠下
  const mirrorRect = mirror.getBoundingClientRect()
  const markerRect = marker.getBoundingClientRect()
  const textTop = markerRect.top - mirrorRect.top
  const textBottom = markerRect.bottom - mirrorRect.top
  document.body.removeChild(mirror)
  const taRect = ta.getBoundingClientRect()
  const areaRect = area.getBoundingClientRect()
  const left = taRect.left - areaRect.left + parseFloat(cs.borderLeftWidth) + parseFloat(cs.paddingLeft) + caretX
  const top = taRect.top - areaRect.top + parseFloat(cs.borderTopWidth) + caretLineTop - ta.scrollTop
  return { left, top, bottom: top + lineHeight, textTop: top + textTop, textBottom: top + textBottom }
}

/** 光标字形盒为锚的同步放置：先按完整高度弹下方，放不下翻转上方，都不行才收缩 */
const placePanelNearCaret = (
  container: HTMLElement | null,
  innerPanelSelector: string,
  caret: ReturnType<typeof measureCaretRectInArea>,
  areaRect: DOMRect,
  left: number,
  set: (placement: { style?: FloatingPanelStyle; maxHeight?: number }) => void
) => {
  const caretBottomVp = areaRect.top + caret.textBottom
  const caretTopVp = areaRect.top + caret.textTop
  const spaceBelow = window.innerHeight - PANEL_VIEWPORT_MARGIN - caretBottomVp
  const spaceAbove = caretTopVp - PANEL_VIEWPORT_MARGIN
  // 自然高度：同步临时解除内层面板的 maxHeight 再量（flex 收缩会让 scrollHeight 失真）；
  // 同一任务内改完即还原，ResizeObserver 观察不到瞬态，不会引发重放震荡
  let naturalHeight = 0
  if (container) {
    const inner = container.querySelector<HTMLElement>(innerPanelSelector)
    const prevMax = inner?.style.maxHeight ?? ''
    if (inner && prevMax) inner.style.maxHeight = ''
    naturalHeight = container.offsetHeight
    if (inner && prevMax) inner.style.maxHeight = prevMax
  }
  if (spaceBelow >= naturalHeight + PANEL_GAP) {
    set({ style: { top: `${caret.textBottom + PANEL_GAP}px`, left: `${left}px` }, maxHeight: undefined })
    return
  }
  const aboveStyle: FloatingPanelStyle = { top: `${caret.textTop - PANEL_GAP}px`, left: `${left}px`, transform: 'translateY(-100%)' }
  if (spaceAbove >= naturalHeight + PANEL_GAP) {
    set({ style: aboveStyle, maxHeight: undefined })
    return
  }
  // 两端都放不下完整面板：选空间更大的一侧收缩
  if (spaceAbove > spaceBelow && spaceAbove >= PANEL_MIN_SPACE) {
    set({ style: aboveStyle, maxHeight: spaceAbove })
  } else if (spaceBelow >= PANEL_MIN_SPACE) {
    set({ style: { top: `${caret.textBottom + PANEL_GAP}px`, left: `${left}px` }, maxHeight: spaceBelow })
  } else {
    // 两端空间都不足最低可用高度：仍按下方收缩放置，宁可小也不落在未定位的默认位置
    set({ style: { top: `${caret.textBottom + PANEL_GAP}px`, left: `${left}px` }, maxHeight: Math.max(spaceBelow, 0) })
  }
}

export interface CaretAnchoredPanelOptions {
  /** 面板是否可见（不可见时清空定位并跳过放置） */
  visible: () => boolean
  /** 消息输入 textarea */
  inputRef: Ref<HTMLTextAreaElement | null>
  /** 面板 absolute 定位的参照容器（输入区） */
  areaRef: Ref<HTMLElement | null>
  /** 面板容器元素（template ref） */
  containerRef: Ref<HTMLElement | null>
  /** 容器内实际面板元素选择器（量自然高度用） */
  innerPanelSelector: string
  /** 面板最大宽度，水平右夹持用 */
  maxWidth: number
  /** 触发重新放置的响应源（通常为 [可见性, 输入内容]） */
  repositionSources: () => unknown[]
}

/**
 * 光标锚定浮层放置。返回 style / maxHeight（供模板绑定），并自行注册
 * 放置触发 watch、内容异步渲染的尺寸纠正 ResizeObserver 与卸载清理。
 */
export const useCaretAnchoredPanel = (options: CaretAnchoredPanelOptions) => {
  const { visible, inputRef, areaRef, containerRef, innerPanelSelector, maxWidth, repositionSources } = options

  const style = ref<CSSProperties | undefined>()
  const maxHeight = ref<number | undefined>(undefined)

  const reposition = () => {
    const ta = inputRef.value
    const area = areaRef.value
    if (!visible() || !ta || !area) {
      style.value = undefined
      maxHeight.value = undefined
      return
    }
    const areaRect = area.getBoundingClientRect()
    const caret = measureCaretRectInArea(ta, area)
    // 水平跟随光标，并夹在输入区内
    const maxLeft = Math.max(0, area.clientWidth - maxWidth - PANEL_RIGHT_MARGIN)
    const left = Math.min(Math.max(0, caret.left), maxLeft)
    placePanelNearCaret(containerRef.value, innerPanelSelector, caret, areaRect, left, (placement) => {
      if (placement.style) style.value = { ...placement.style }
      maxHeight.value = placement.maxHeight
    })
  }

  watch(repositionSources, () => nextTick(reposition), { immediate: true })

  // 窗口尺寸变化会改变 spaceBelow/spaceAbove 的视口基准，已打开面板需重新放置；
  // 卸载时成对移除，防监听器泄漏
  window.addEventListener('resize', reposition)

  /**
   * 面板内容异步渲染（子命令列表/快捷回复拉取、过滤）晚于初次放置时，
   * 量到的高度偏小会漏掉翻转/收缩判定——监听容器尺寸变化后重新放置。
   * 重新放置在尺寸稳定后收敛（transform 不改变尺寸，maxHeight 不再变化即停止），无死循环。
   */
  const resizeObserver = new ResizeObserver(reposition)

  watch(containerRef, (el, prevEl) => {
    if (prevEl && prevEl !== el) resizeObserver.unobserve(prevEl)
    if (el) resizeObserver.observe(el)
  })

  onBeforeUnmount(() => {
    window.removeEventListener('resize', reposition)
    resizeObserver.disconnect()
  })

  return { style, maxHeight }
}

export interface ScrollbarMetrics {
  thumbHeightPercent: number
  thumbTopPercent: number
  scrollable: boolean
}

export function scrollbarMetrics(viewportY: number, baseY: number, rows: number): ScrollbarMetrics {
  const viewport = Math.max(1, rows)
  const scrollback = Math.max(0, baseY)
  const total = Math.max(viewport, scrollback + viewport)
  const thumbHeightPercent = Math.min(100, (viewport / total) * 100)
  const maxViewportY = Math.max(0, scrollback)
  const thumbTopPercent = maxViewportY === 0
    ? 0
    : Math.max(0, Math.min(100 - thumbHeightPercent, (viewportY / maxViewportY) * (100 - thumbHeightPercent)))
  return {
    thumbHeightPercent,
    thumbTopPercent,
    scrollable: scrollback > 0,
  }
}

export function scrollLineFromThumb(
  pointerY: number,
  trackTop: number,
  trackHeight: number,
  thumbHeight: number,
  dragOffset: number,
  baseY: number,
): number {
  if (trackHeight <= 0 || baseY <= 0) return 0
  const thumbTop = pointerY - trackTop - dragOffset
  const usable = Math.max(1, trackHeight - thumbHeight)
  return Math.round(Math.max(0, Math.min(1, thumbTop / usable)) * baseY)
}

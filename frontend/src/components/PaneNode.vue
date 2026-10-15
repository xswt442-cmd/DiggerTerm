<script setup lang="ts">
import { computed, ref } from 'vue'
import type { PaneNode as PaneTreeNode } from '../workspaceLayout'

defineOptions({ name: 'PaneNode' })

type Rect = { x: number; y: number; width: number; height: number }

const props = defineProps<{ node: PaneTreeNode; rect: Rect }>()
const emit = defineEmits<{ resize: [splitId: string, ratio: number] }>()
const root = ref<HTMLElement | null>(null)
let activePointer: number | null = null
let dragBounds: DOMRect | undefined

const isSplit = computed(() => props.node.type === 'split')
const firstRect = computed<Rect | null>(() => {
  if (props.node.type !== 'split') return null
  return props.node.direction === 'horizontal'
    ? { x: 0, y: 0, width: props.node.ratio, height: 1 }
    : { x: 0, y: 0, width: 1, height: props.node.ratio }
})
const secondRect = computed<Rect | null>(() => {
  if (props.node.type !== 'split' || !firstRect.value) return null
  return props.node.direction === 'horizontal'
    ? { x: props.node.ratio, y: 0, width: 1 - props.node.ratio, height: 1 }
    : { x: 0, y: props.node.ratio, width: 1, height: 1 - props.node.ratio }
})
const nodeStyle = computed(() => ({
  left: `${props.rect.x * 100}%`,
  top: `${props.rect.y * 100}%`,
  width: `${props.rect.width * 100}%`,
  height: `${props.rect.height * 100}%`,
}))
const dividerStyle = computed(() => {
  if (props.node.type !== 'split') return {}
  return props.node.direction === 'horizontal'
    ? { left: `${props.node.ratio * 100}%` }
    : { top: `${props.node.ratio * 100}%` }
})

function forwardResize(splitId: string, ratio: number) {
  emit('resize', splitId, ratio)
}

function ratioFromPointer(event: PointerEvent): number | undefined {
  if (props.node.type !== 'split') return
  const bounds = dragBounds ?? root.value?.getBoundingClientRect()
  if (!bounds) return
  return props.node.direction === 'horizontal'
    ? (event.clientX - bounds.left) / Math.max(1, bounds.width)
    : (event.clientY - bounds.top) / Math.max(1, bounds.height)
}

function beginResize(event: PointerEvent) {
  if (props.node.type !== 'split' || event.button !== 0) return
  event.preventDefault()
  activePointer = event.pointerId
  dragBounds = root.value?.getBoundingClientRect()
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
  const ratio = ratioFromPointer(event)
  if (ratio !== undefined) emit('resize', props.node.id, ratio)
}

function continueResize(event: PointerEvent) {
  if (event.pointerId !== activePointer) return
  const ratio = ratioFromPointer(event)
  if (ratio !== undefined && props.node.type === 'split') emit('resize', props.node.id, ratio)
}

function endResize(event: PointerEvent) {
  if (event.pointerId !== activePointer) return
  activePointer = null
  dragBounds = undefined
  const handle = event.currentTarget as HTMLElement
  if (handle.hasPointerCapture(event.pointerId)) handle.releasePointerCapture(event.pointerId)
}

function moveByKeyboard(event: KeyboardEvent) {
  if (props.node.type !== 'split') return
  const delta = props.node.direction === 'horizontal'
    ? event.key === 'ArrowLeft' ? -0.03 : event.key === 'ArrowRight' ? 0.03 : 0
    : event.key === 'ArrowUp' ? -0.03 : event.key === 'ArrowDown' ? 0.03 : 0
  if (!delta) return
  event.preventDefault()
  emit('resize', props.node.id, Math.max(0.15, Math.min(0.85, props.node.ratio + delta)))
}
</script>

<template>
  <div v-if="isSplit && node.type === 'split'" ref="root" class="pane-tree-node" :style="nodeStyle">
    <PaneNode :node="node.children[0]" :rect="firstRect!" @resize="forwardResize" />
    <PaneNode :node="node.children[1]" :rect="secondRect!" @resize="forwardResize" />
    <div
      class="pane-divider"
      :class="`pane-divider--${node.direction}`"
      role="separator"
      tabindex="0"
      :aria-orientation="node.direction === 'horizontal' ? 'vertical' : 'horizontal'"
      aria-label="Resize pane split"
      aria-valuemin="15"
      aria-valuemax="85"
      :aria-valuenow="Math.round(node.ratio * 100)"
      :style="dividerStyle"
      @pointerdown="beginResize"
      @pointermove="continueResize"
      @pointerup="endResize"
      @pointercancel="endResize"
      @keydown="moveByKeyboard"
    />
  </div>
</template>

<style scoped>
.pane-tree-node {
  position: absolute;
  pointer-events: none;
}

.pane-divider {
  position: absolute;
  z-index: 2;
  display: block;
  pointer-events: auto;
  touch-action: none;
  background: color-mix(in srgb, var(--border, #343a44) 80%, transparent);
  outline: none;
}

.pane-divider:hover,
.pane-divider:focus-visible {
  background: var(--accent, #bca58b);
}

.pane-divider:focus-visible {
  box-shadow: 0 0 0 2px var(--canvas, #101318), 0 0 0 4px var(--accent, #bca58b);
}

.pane-divider--horizontal {
  top: 0;
  bottom: 0;
  width: 8px;
  transform: translateX(-50%);
  cursor: col-resize;
}

.pane-divider--vertical {
  left: 0;
  right: 0;
  height: 8px;
  transform: translateY(-50%);
  cursor: row-resize;
}
</style>

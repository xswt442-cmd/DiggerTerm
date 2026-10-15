<script setup lang="ts">
import { computed } from 'vue'
import type { PaneNode, WorkspaceLayout } from '../workspaceLayout'
import PaneNodeView from './PaneNode.vue'

type Leaf = Extract<PaneNode, { type: 'leaf' }>
type Rect = { x: number; y: number; width: number; height: number }

const props = defineProps<{ layout: WorkspaceLayout }>()
const emit = defineEmits<{
  focus: [paneId: string]
  resize: [splitId: string, ratio: number]
}>()

function flatten(node: PaneNode, rect: Rect, output: { pane: Leaf; rect: Rect }[]) {
  if (node.type === 'leaf') {
    output.push({ pane: node, rect })
    return
  }
  if (node.direction === 'horizontal') {
    const width = rect.width * node.ratio
    flatten(node.children[0], { ...rect, width }, output)
    flatten(node.children[1], { ...rect, x: rect.x + width, width: rect.width - width }, output)
  } else {
    const height = rect.height * node.ratio
    flatten(node.children[0], { ...rect, height }, output)
    flatten(node.children[1], { ...rect, y: rect.y + height, height: rect.height - height }, output)
  }
}

function minimumSize(node: PaneNode): { width: number; height: number } {
  if (node.type === 'leaf') return { width: 192, height: 132 }
  const first = minimumSize(node.children[0])
  const second = minimumSize(node.children[1])
  return node.direction === 'horizontal'
    ? {
        width: Math.max(first.width / node.ratio, second.width / (1 - node.ratio)),
        height: Math.max(first.height, second.height),
      }
    : {
        width: Math.max(first.width, second.width),
        height: Math.max(first.height / node.ratio, second.height / (1 - node.ratio)),
      }
}

const paneRects = computed(() => {
  const result: { pane: Leaf; rect: Rect }[] = []
  flatten(props.layout.root, { x: 0, y: 0, width: 1, height: 1 }, result)
  return result
})
const minSize = computed(() => minimumSize(props.layout.root))
const canvasStyle = computed(() => ({
  width: `max(100%, ${minSize.value.width}px)`,
  height: `max(100%, ${minSize.value.height}px)`,
}))

function forwardResize(splitId: string, ratio: number) {
  emit('resize', splitId, ratio)
}
</script>

<template>
  <div class="pane-workspace" role="group" aria-label="Terminal panes">
    <div class="pane-workspace__canvas" :style="canvasStyle">
      <PaneNodeView :node="layout.root" :rect="{ x: 0, y: 0, width: 1, height: 1 }" @resize="forwardResize" />
      <div
        v-for="entry in paneRects"
        :key="entry.pane.paneId"
        class="pane-workspace__leaf"
        :class="{ 'pane-workspace__leaf--focused': layout.focusedPaneId === entry.pane.paneId }"
        :style="{
          '--pane-x': `${entry.rect.x * 100}%`, '--pane-y': `${entry.rect.y * 100}%`,
          '--pane-width': `${entry.rect.width * 100}%`, '--pane-height': `${entry.rect.height * 100}%`,
        }"
        :data-pane-id="entry.pane.paneId"
        @pointerdown="emit('focus', entry.pane.paneId)"
        @focusin="emit('focus', entry.pane.paneId)"
      >
        <slot :pane="entry.pane" :focused="layout.focusedPaneId === entry.pane.paneId" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.pane-workspace {
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  overflow: auto;
  overscroll-behavior: contain;
}

.pane-workspace__canvas {
  position: relative;
  min-width: 100%;
  min-height: 100%;
  overflow: hidden;
  isolation: isolate;
}

.pane-workspace__leaf {
  position: absolute;
  z-index: 1;
  box-sizing: border-box;
  overflow: hidden;
  border: 1px solid var(--border, #343a44);
  border-radius: var(--dt-radius, 8px);
  background: var(--canvas, #101318);
  box-shadow: inset 0 0 0 1px transparent;
  left: calc(var(--pane-x, 0%) + var(--dt-pane-gap, 6px) / 2);
  top: calc(var(--pane-y, 0%) + var(--dt-pane-gap, 6px) / 2);
  width: calc(var(--pane-width, 100%) - var(--dt-pane-gap, 6px));
  height: calc(var(--pane-height, 100%) - var(--dt-pane-gap, 6px));
}

.pane-workspace__leaf--focused {
  border-color: var(--accent, #bca58b);
  box-shadow: inset 0 0 0 1px var(--accent, #bca58b);
}
</style>

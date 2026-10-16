<script setup lang="ts">
import type { ProcessGroup, ProcessNode } from '../dig/processTree'
import DigTreeNode from './DigTreeNode.vue'

defineProps<{
  groups: ProcessGroup[]
  selectedKey: string | null
  showPid: boolean
  locale: string
  label: string
  kindLabel: (kind: string) => string
}>()

const emit = defineEmits<{ select: [process: ProcessNode] }>()
</script>

<template>
  <nav class="dig-tree" :aria-label="label">
    <section v-for="group in groups" :key="group.source + ':' + group.distribution" class="dig-group">
      <h3 class="dig-group-label">{{ group.source === 'wsl' ? `WSL · ${group.distribution}` : 'Windows' }}</h3>
      <ul>
        <DigTreeNode v-for="node in group.roots" :key="node.key" :node="node"
          :selected-key="selectedKey" :show-pid="showPid" :locale="locale" :kind-label="kindLabel"
          :expand-filtered-nodes="group.expandFilteredNodes ?? false"
          @select="emit('select', $event)" />
      </ul>
    </section>
  </nav>
</template>

<style scoped>
.dig-tree { min-height: 0; flex: 1 1 0; overflow: auto; scrollbar-width: thin; scrollbar-color: var(--border) transparent; }
.dig-group-label { margin: 7px 5px 3px; color: var(--muted); font-size: calc(9px * var(--ui-font-scale)); font-weight: 600; text-transform: uppercase; }
ul { list-style: none; margin: 0; padding: 0; }
</style>

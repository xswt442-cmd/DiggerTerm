<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ProcessNode } from '../dig/processTree'
import { classificationLabel } from '../dig/processTree'

const props = defineProps<{
  node: ProcessNode
  selectedKey: string | null
  showPid: boolean
  locale: string
  kindLabel: (kind: string) => string
  expandFilteredNodes: boolean
}>()

const emit = defineEmits<{ select: [process: ProcessNode] }>()
const userExpanded = ref(props.node.children.length > 0)
const expanded = computed(() => props.expandFilteredNodes && props.node.children.length > 0
  ? true
  : userExpanded.value)
</script>

<template>
  <li>
    <div class="dig-row" :class="{ selected: selectedKey === node.key }">
      <button v-if="node.children.length" type="button" class="dig-toggle"
        :disabled="expandFilteredNodes"
        :aria-label="expandFilteredNodes
          ? (locale === 'zh' ? '搜索结果已展开' : 'Expanded to show filtered results')
          : expanded
          ? (locale === 'zh' ? '折叠子进程' : 'Collapse child processes')
          : (locale === 'zh' ? '展开子进程' : 'Expand child processes')"
        :aria-expanded="expanded" @click="userExpanded = !userExpanded">{{ expanded ? '▾' : '▸' }}</button>
      <span v-else class="dig-toggle-space" aria-hidden="true" />
      <button type="button" class="dig-select" :aria-pressed="selectedKey === node.key"
        @click="emit('select', node)">
        <span class="dig-name">{{ node.name }}</span>
        <span class="dig-meta">{{ kindLabel(node.kind) }} · {{ classificationLabel(node) === 'identified' ? (locale === 'zh' ? '名称识别' : 'identified') : classificationLabel(node) === 'inferred' ? (locale === 'zh' ? '父链推断' : 'inferred') : (locale === 'zh' ? '来源未知' : 'unknown source') }}<span v-if="showPid"> · PID {{ node.pid }}</span></span>
      </button>
    </div>
    <ul v-if="expanded && node.children.length">
      <DigTreeNode v-for="child in node.children" :key="child.key" :node="child"
        :selected-key="selectedKey" :show-pid="showPid" :locale="locale" :kind-label="kindLabel"
        :expand-filtered-nodes="expandFilteredNodes"
        @select="emit('select', $event)" />
    </ul>
  </li>
</template>

<style scoped>
li { list-style: none; }
ul { list-style: none; margin: 0; padding-left: 14px; }
.dig-row { display: flex; min-height: 34px; align-items: stretch; border-radius: 5px; }
.dig-row.selected { background: var(--surface-hover); }
.dig-toggle, .dig-toggle-space { display: grid; place-items: center; flex: 0 0 19px; width: 19px; padding: 0; border: 0; background: transparent; color: var(--muted); font-size: 11px; }
.dig-select { display: flex; min-width: 0; flex: 1; flex-direction: column; justify-content: center; gap: 1px; padding: 3px 5px; border: 0; border-radius: 4px; background: transparent; color: var(--text); text-align: left; }
.dig-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: calc(11px * var(--ui-font-scale)); }
.dig-meta { overflow: hidden; color: var(--muted); text-overflow: ellipsis; white-space: nowrap; font-size: calc(9px * var(--ui-font-scale)); }
button:focus-visible { outline: 1px solid var(--accent); outline-offset: -1px; }
</style>

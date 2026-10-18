<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { Locale } from '../uiPreferences'

const props = defineProps<{
  locale: Locale
  query: string
  caseSensitive: boolean
  resultIndex: number
  resultCount: number
}>()
const emit = defineEmits<{
  'update:query': [value: string]
  'update:caseSensitive': [value: boolean]
  next: []
  previous: []
  close: []
}>()
const input = ref<HTMLInputElement | null>(null)
const text = (zh: string, en: string) => props.locale === 'zh' ? zh : en

onMounted(() => {
  input.value?.focus()
  input.value?.select()
})

function handleKeydown(event: KeyboardEvent) {
  event.stopPropagation()
  if (event.isComposing || event.key === 'Process' || event.key === 'Dead') return
  if (event.key === 'Escape') {
    event.preventDefault()
    emit('close')
  } else if (event.key === 'Enter') {
    event.preventDefault()
    if (event.shiftKey) emit('previous')
    else emit('next')
  }
}
</script>

<template>
  <div class="terminal-search" role="search" @keydown="handleKeydown">
    <input
      ref="input"
      :value="query"
      type="search"
      maxlength="256"
      :aria-label="text('搜索当前终端', 'Search this terminal')"
      :placeholder="text('搜索当前终端缓冲', 'Search current terminal buffer')"
      @input="emit('update:query', ($event.target as HTMLInputElement).value)"
    />
    <span class="search-results" role="status" aria-live="polite">
      {{ !query ? text('输入关键词', 'Type to search') : !resultCount ? text('无结果', 'No results') : resultIndex < 0 ? text('结果过多，已达到高亮上限', 'Match limit reached') : `${resultIndex + 1} / ${resultCount}` }}
    </span>
    <button type="button" :aria-pressed="caseSensitive" :aria-label="text('区分大小写', 'Match case')" @click="emit('update:caseSensitive', !caseSensitive)">Aa</button>
    <button type="button" :disabled="!query || !resultCount" :aria-label="text('上一个结果', 'Previous result')" @click="emit('previous')">↑</button>
    <button type="button" :disabled="!query || !resultCount" :aria-label="text('下一个结果', 'Next result')" @click="emit('next')">↓</button>
    <button type="button" :aria-label="text('关闭搜索', 'Close search')" @click="emit('close')">×</button>
  </div>
</template>

<style scoped>
.terminal-search { display: flex; flex: 0 0 34px; min-width: 0; align-items: center; gap: 5px; padding: 3px 6px; border-bottom: 1px solid var(--border-faint); background: var(--surface); }
.terminal-search input { min-width: 0; flex: 1; height: 26px; padding: 3px 7px; color: var(--text); background: var(--canvas); border: 1px solid var(--border); border-radius: var(--dt-radius, 4px); font: inherit; }
.terminal-search button { flex: 0 0 auto; min-width: 27px; height: 26px; color: var(--text); background: var(--canvas); border: 1px solid var(--border); border-radius: var(--dt-radius, 4px); font-size: 11px; }
.search-results { min-width: 62px; color: var(--muted); text-align: center; font-size: 10px; }
</style>

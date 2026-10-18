<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { Locale } from '../uiPreferences'
import type { PositionedQuickSelectCandidate } from '../quickSelect'

type DisplayCandidate = PositionedQuickSelectCandidate & { left: number; top: number; visible: boolean }
const props = defineProps<{ locale: Locale; candidates: DisplayCandidate[]; truncated: boolean; range: 'visible' | 'buffer' }>()
const emit = defineEmits<{
  selected: [candidate: DisplayCandidate]
  close: []
  'update:range': [range: 'visible' | 'buffer']
}>()
const host = ref<HTMLElement | null>(null)
const typed = ref('')
let resetTimer: ReturnType<typeof setTimeout> | undefined
const text = (zh: string, en: string) => props.locale === 'zh' ? zh : en

onMounted(() => host.value?.focus())
onBeforeUnmount(() => clearTimeout(resetTimer))
watch(() => props.candidates, () => {
  typed.value = ''
  clearTimeout(resetTimer)
}, { deep: false })

function handleKeydown(event: KeyboardEvent) {
  event.stopPropagation()
  if (event.isComposing || event.key === 'Process' || event.key === 'Dead') return
  if ((event.target as HTMLElement).closest('button,select')) return
  if (event.key === 'Escape') {
    event.preventDefault()
    emit('close')
    return
  }
  if (event.ctrlKey || event.metaKey || event.altKey || !/^[a-z]$/i.test(event.key)) return
  event.preventDefault()
  const next = typed.value + event.key.toLowerCase()
  const match = props.candidates.find(candidate => candidate.label === next)
  if (match) {
    emit('selected', match)
    return
  }
  if (props.candidates.some(candidate => candidate.label.startsWith(next))) {
    typed.value = next
    clearTimeout(resetTimer)
    resetTimer = setTimeout(() => { typed.value = '' }, 1200)
  } else {
    typed.value = ''
  }
}

function choose(candidate: DisplayCandidate) {
  emit('selected', candidate)
}
</script>

<template>
  <div ref="host" class="quick-select-layer" tabindex="0" role="group"
    :aria-label="text('快速选择终端文字，按标签键选择，Escape 退出', 'Quick Select terminal text. Type a label to copy; Escape closes.')"
    @keydown="handleKeydown">
    <button v-for="candidate in candidates.filter(item => item.visible)" :key="`${candidate.start}:${candidate.end}:${candidate.type}`"
      class="quick-select-badge" :style="{ left: `${candidate.left}px`, top: `${candidate.top}px` }"
      :aria-label="`${candidate.label}: ${candidate.name}, ${candidate.text}`" @pointerdown.prevent @click.stop="choose(candidate)">
      {{ candidate.label }}
    </button>
    <aside v-if="candidates.some(item => !item.visible)" class="quick-select-offscreen" :aria-label="text('屏幕外的候选', 'Off-screen candidates')">
      <button v-for="candidate in candidates.filter(item => !item.visible)" :key="`${candidate.start}:${candidate.end}:${candidate.type}`"
        type="button" @click="choose(candidate)"><kbd>{{ candidate.label }}</kbd><span>{{ candidate.text }}</span></button>
    </aside>
    <div class="quick-select-status" role="status" aria-live="polite">
      {{ text('输入标签复制，Escape 取消', 'Type a label to copy; Escape cancels') }}
      <span v-if="typed"> · {{ typed }}</span>
      <span v-if="truncated"> · {{ text('候选已达上限或扫描文本已截断', 'Candidate limit reached or scan text was truncated') }}</span>
      <select :value="range" :aria-label="text('候选范围', 'Candidate range')"
        @change="emit('update:range', ($event.target as HTMLSelectElement).value as 'visible' | 'buffer')">
        <option value="visible">{{ text('当前屏幕', 'Visible screen') }}</option>
        <option value="buffer">{{ text('当前缓冲', 'Current buffer') }}</option>
      </select>
    </div>
  </div>
</template>

<style scoped>
.quick-select-layer { position: absolute; z-index: 4; inset: 0; outline: none; pointer-events: none; }
.quick-select-badge { position: absolute; min-width: 19px; height: 19px; padding: 0 4px; border: 1px solid var(--accent); border-radius: 4px; color: var(--accent-text); background: var(--accent-bg); box-shadow: 0 1px 4px #0008; font: 700 10px/17px Consolas, monospace; text-align: center; pointer-events: auto; }
.quick-select-badge:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
.quick-select-status { position: absolute; top: 4px; right: 6px; display: flex; align-items: center; gap: 4px; max-width: 90%; padding: 4px 7px; overflow: hidden; border: 1px solid var(--border); border-radius: 4px; color: var(--text); background: var(--surface); font-size: 10px; text-overflow: ellipsis; white-space: nowrap; pointer-events: auto; }
.quick-select-status select { max-width: 110px; border: 1px solid var(--border); border-radius: 3px; padding: 2px; color: var(--text); background: var(--canvas); font-size: 10px; }
.quick-select-offscreen { position: absolute; right: 6px; bottom: 6px; display: grid; gap: 3px; width: min(360px, 60%); max-height: 35%; overflow: auto; padding: 5px; border: 1px solid var(--border); border-radius: 5px; background: var(--surface); pointer-events: auto; }
.quick-select-offscreen button { display: flex; min-width: 0; gap: 7px; padding: 3px 5px; border: 0; color: var(--text); background: transparent; text-align: left; font-size: 10px; }
.quick-select-offscreen button:hover { background: var(--surface-hover); }
.quick-select-offscreen kbd { flex: 0 0 auto; min-width: 20px; color: var(--accent-text); }
.quick-select-offscreen span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import type { PaletteCommand } from '../commandPalette'
import { filterPaletteCommands } from '../commandPalette'
import type { Locale } from '../uiPreferences'

const props = defineProps<{ locale: Locale; commands: PaletteCommand[] }>()
const emit = defineEmits<{ close: []; execute: [id: string] }>()
const input = ref<HTMLInputElement | null>(null)
const query = ref('')
const selectedID = ref('')
const text = (zh: string, en: string) => props.locale === 'zh' ? zh : en
const results = computed(() => filterPaletteCommands(props.commands, query.value))
const selectedIndex = computed(() => results.value.findIndex(command => command.id === selectedID.value))

watch(() => props.commands, commands => {
  if (!commands.some(command => command.id === selectedID.value)) {
    selectedID.value = results.value.find(command => command.enabled)?.id ?? results.value[0]?.id ?? ''
  }
}, { immediate: true })
watch(results, items => {
  if (!items.some(command => command.id === selectedID.value)) {
    selectedID.value = items.find(command => command.enabled)?.id ?? items[0]?.id ?? ''
  }
})
onMounted(() => input.value?.focus())

function move(direction: number) {
  const items = results.value
  if (!items.length) return
  let index = selectedIndex.value
  for (let step = 0; step < items.length; step++) {
    index = (index + direction + items.length) % items.length
    if (items[index]!.enabled) { selectedID.value = items[index]!.id; return }
  }
}

function handleKeydown(event: KeyboardEvent) {
  event.stopPropagation()
  if (event.isComposing || event.key === 'Process' || event.key === 'Dead') return
  if (event.key === 'Escape') {
    event.preventDefault()
    emit('close')
  } else if (event.key === 'ArrowDown') {
    event.preventDefault()
    move(1)
  } else if (event.key === 'ArrowUp') {
    event.preventDefault()
    move(-1)
  } else if (event.key === 'Enter') {
    const command = results.value.find(item => item.id === selectedID.value)
    if (command?.enabled) {
      event.preventDefault()
      emit('execute', command.id)
    }
  }
}

async function choose(command: PaletteCommand) {
  if (!command.enabled) return
  selectedID.value = command.id
  await nextTick()
  emit('execute', command.id)
}
</script>

<template>
  <Teleport to="body">
    <div class="command-palette-backdrop" @pointerdown.self="emit('close')">
      <section class="command-palette" role="dialog" aria-modal="true"
        :aria-label="text('命令面板', 'Command palette')" @keydown="handleKeydown">
        <input ref="input" v-model="query" type="search" maxlength="128"
          :aria-label="text('搜索命令', 'Search commands')"
          :placeholder="text('搜索会话、操作、设置…', 'Search sessions, actions, settings…')"
          :aria-activedescendant="selectedID ? `command-${selectedID}` : undefined" />
        <div v-if="results.length" class="command-results" role="listbox" :aria-label="text('命令结果', 'Command results')">
          <button v-for="command in results" :id="`command-${command.id}`" :key="command.id" type="button"
            class="command-option" role="option" :aria-selected="selectedID === command.id"
            :disabled="!command.enabled" @mouseenter="selectedID = command.id" @click="choose(command)">
            <span class="command-label">{{ command.label }}</span>
            <span class="command-category">{{ command.category }}</span>
          </button>
        </div>
        <p v-else class="command-empty" role="status">{{ text('没有匹配的命令。', 'No matching commands.') }}</p>
      </section>
    </div>
  </Teleport>
</template>

<style scoped>
.command-palette-backdrop { position: fixed; z-index: 30; inset: 0; display: flex; align-items: flex-start; justify-content: center; padding: min(12vh, 90px) 16px 16px; background: #0007; }
.command-palette { width: min(560px, 100%); max-height: min(70vh, 560px); overflow: hidden; border: 1px solid var(--border); border-radius: 9px; background: var(--sidebar); box-shadow: 0 18px 60px #000b; }
.command-palette input { width: 100%; height: 44px; padding: 0 13px; border: 0; border-bottom: 1px solid var(--border); outline: none; color: var(--text); background: var(--surface); font: inherit; }
.command-results { max-height: calc(min(70vh, 560px) - 45px); overflow: auto; padding: 5px; }
.command-option { display: flex; align-items: center; justify-content: space-between; gap: 12px; width: 100%; min-height: 34px; padding: 6px 9px; border: 0; border-radius: 5px; color: var(--text); background: transparent; text-align: left; font: inherit; }
.command-option[aria-selected="true"] { background: var(--surface-active); }
.command-option:disabled { opacity: .45; }
.command-label { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.command-category { flex: 0 0 auto; color: var(--muted); font-size: .85em; }
.command-empty { margin: 0; padding: 18px 12px; color: var(--muted); text-align: center; }
</style>

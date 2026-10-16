<script setup lang="ts">
export type SettingsTab = 'appearance' | 'terminal' | 'sessions' | 'observed' | 'keybindings' | 'help'

defineProps<{
  open: boolean
  tab: SettingsTab
  title: string
  closeLabel: string
  tabLabels: Record<SettingsTab, string>
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  'update:tab': [value: SettingsTab]
}>()

const tabs: SettingsTab[] = ['appearance', 'terminal', 'sessions', 'observed', 'keybindings', 'help']
</script>

<template>
  <div v-if="open" class="settings-scrim" @click="emit('update:open', false)"></div>
  <section v-if="open" class="settings-sheet" role="dialog" aria-modal="true" :aria-label="title">
    <header class="settings-head">
      <strong>{{ title }}</strong>
      <button type="button" :title="closeLabel" :aria-label="closeLabel" @click="emit('update:open', false)">×</button>
    </header>
    <div class="settings-tabs" role="tablist" :aria-label="title">
      <button v-for="item in tabs" :key="item" type="button" role="tab"
        :aria-selected="tab === item" :class="{ chosen: tab === item }"
        @click="emit('update:tab', item)">
        {{ tabLabels[item] }}
      </button>
    </div>
    <div class="settings-scroll" role="tabpanel"><slot /></div>
  </section>
</template>

<style scoped>
.settings-scrim { position: fixed; inset: 0; z-index: 10; }
.settings-sheet { display: flex; flex-direction: column; position: absolute; z-index: 11; left: 6px; bottom: 45px; width: min(304px, calc(100vw - 20px)); max-height: min(495px, calc(100vh - 86px)); overflow: hidden; border: 1px solid var(--border); background: var(--sidebar); box-shadow: 0 12px 36px #0009; border-radius: 8px; }
.settings-head { display: flex; justify-content: space-between; align-items: center; flex-shrink: 0; gap: 8px; padding: 6px 10px; min-height: 34px; border-bottom: 1px solid var(--border-faint); }
.settings-head strong { display: block; font-size: calc(12px * var(--ui-font-scale)); font-weight: 650; }
.settings-head button { flex: 0 0 23px; height: 23px; border: 0; background: var(--surface); color: var(--text); font-size: calc(16px * var(--ui-font-scale)); border-radius: 4px; }
.settings-tabs { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 2px; flex-shrink: 0; padding: 5px 7px 3px; border-bottom: 1px solid var(--border-faint); }
.settings-tabs button { min-width: 0; height: 26px; padding: 0 3px; border: 0; border-radius: 4px; color: var(--muted); background: transparent; font-size: calc(10px * var(--ui-font-scale)); }
.settings-tabs button:hover { color: var(--text); background: var(--surface); }
.settings-tabs button.chosen { color: var(--accent-text); background: var(--accent-bg); }
.settings-scroll { display: flex; flex-direction: column; min-height: 0; overflow-x: hidden; overflow-y: auto; padding: 9px 10px 10px; gap: 9px; scrollbar-width: thin; scrollbar-color: var(--border) transparent; }
.settings-scroll select:disabled { opacity: .4; }
.settings-scroll::-webkit-scrollbar { width: 6px; }
.settings-scroll::-webkit-scrollbar-thumb { background: var(--border); border-radius: 6px; }
@media (max-height: 480px) { .settings-scroll { gap: 9px; } }
</style>

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

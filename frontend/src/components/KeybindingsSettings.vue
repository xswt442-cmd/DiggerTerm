<script setup lang="ts">
import { ref, watch } from 'vue'
import { actionDefinitions, defaultBindings, eventChord, parseBindings, type ActionId, type Binding } from '../keybindings'
import { KeybindingsPath, OpenKeybindings, ReadKeybindings, SaveKeybindings } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/keybindingservice'
import type { Locale } from '../uiPreferences'

const props = defineProps<{ locale: Locale; bindings: Binding[]; isMac: boolean }>()
const emit = defineEmits<{ changed: [bindings: Binding[]] }>()
const path = ref('')
const error = ref('')
const status = ref('')
const drafts = ref<Record<string, string>>({})
const recording = ref<ActionId | null>(null)
watch(() => props.bindings, bindings => {
  drafts.value = Object.fromEntries(bindings.map(binding => [binding.action, binding.keys.join('; ')]))
}, { immediate: true, deep: true })
void KeybindingsPath().then(value => { path.value = value }).catch(err => { error.value = String(err) })
const text = (zh: string, en: string) => props.locale === 'zh' ? zh : en
function effective(overrides: Binding[]) {
  return defaultBindings(props.isMac).map(binding => overrides.find(item => item.action === binding.action) ?? binding)
}
async function reload() {
  error.value = ''; status.value = ''
  try {
    const source = await ReadKeybindings()
    emit('changed', effective(source ? parseBindings(source, props.isMac) : []))
    status.value = text('已重新加载', 'Reloaded')
  } catch (err) { error.value = String(err) }
}
async function save(reset = false) {
  error.value = ''; status.value = ''; recording.value = null
  try {
    const source = JSON.stringify({ version: 1, bindings: reset ? [] : actionDefinitions.map(definition => ({
      action: definition.id, scope: definition.scope,
      keys: (drafts.value[definition.id] ?? '').split(';').map(key => key.trim()).filter(Boolean),
    })) }, null, 2)
    const overrides = parseBindings(source, props.isMac)
    await SaveKeybindings(source)
    emit('changed', effective(overrides))
    status.value = text('已保存', 'Saved')
  } catch (err) { error.value = String(err) }
}
function record(event: KeyboardEvent, action: ActionId) {
  if (recording.value !== action) return
  event.preventDefault(); event.stopPropagation()
  if (event.key === 'Escape') { recording.value = null; return }
  const chord = eventChord(event)
  if (chord) { drafts.value[action] = chord; recording.value = null }
}
async function openFile() {
  try { await OpenKeybindings() } catch (err) { error.value = String(err) }
}
</script>

<template>
  <div class="keybindings-settings">
    <p>{{ text('按焦点终端或工作区生效。多个键位用分号分隔，留空禁用。', 'Bindings apply to the focused terminal or workspace. Separate keys with semicolons; empty disables.') }}</p>
    <p>{{ text('IME/AltGr、输入框和设置对话框优先处理自身按键。Ctrl+C 的智能选区规则独立保留。', 'IME/AltGr, text fields and settings handle their own keys first. Smart Ctrl+C selection behavior stays available.') }}</p>
    <label v-for="definition in actionDefinitions" :key="definition.id" class="binding-row">
      <strong>{{ locale === 'zh' ? definition.labelZh : definition.labelEn }}</strong>
      <small>{{ definition.scope }} · {{ text('默认', 'Default') }}: {{ (isMac ? definition.defaultKeys.mac : definition.defaultKeys.other).join('; ') }}</small>
      <div @keydown.capture="record($event, definition.id)">
        <input v-model="drafts[definition.id]" :aria-label="locale === 'zh' ? definition.labelZh : definition.labelEn"
          spellcheck="false" :placeholder="text('禁用', 'Disabled')" />
        <button type="button" @click="recording = recording === definition.id ? null : definition.id">
          {{ recording === definition.id ? text('按组合键…', 'Press keys…') : text('录制', 'Record') }}
        </button>
      </div>
    </label>
    <p v-if="error" role="alert">{{ error }}</p><p v-if="status" role="status">{{ status }}</p>
    <div class="binding-actions">
      <button type="button" @click="save()">{{ text('保存并检测冲突', 'Validate and save') }}</button>
      <button type="button" @click="reload">{{ text('重新加载', 'Reload') }}</button>
      <button type="button" @click="save(true)">{{ text('恢复默认', 'Restore defaults') }}</button>
      <button type="button" @click="openFile">{{ text('打开配置文件', 'Open configuration') }}</button>
    </div>
    <small>{{ path }}</small>
  </div>
</template>

<style scoped>
.keybindings-settings { display: grid; gap: 10px; color: var(--text); font-size: 11px; }
p { margin: 0; line-height: 1.5; }
.binding-row { display: grid; gap: 4px; }
small { overflow-wrap: anywhere; color: var(--muted); }
.binding-row div, .binding-actions { display: flex; gap: 4px; flex-wrap: wrap; }
input { min-width: 0; flex: 1; width: 130px; }
input, button { padding: 5px; border: 1px solid var(--border); background: var(--surface); color: var(--text); border-radius: var(--dt-radius, 4px); font-size: inherit; }
</style>

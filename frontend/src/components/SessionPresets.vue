<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Dialogs } from '@wailsio/runtime'
import type { Profile } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/session/models'
import type { Locale } from '../uiPreferences'
import { validPresetName, validPresetSSHTarget, type SessionPreset } from '../workspaceLibrary'
import { localDefaultDirectoryAllowed, supportsLocalDefaultDirectory } from '../quickCd'

const props = defineProps<{ presets: SessionPreset[]; profiles: Profile[]; locale: Locale; busy: boolean; connected: boolean; defaultProfile: string; defaultDirectory: string }>()
const emit = defineEmits<{ save: [preset: SessionPreset]; remove: [id: string]; launch: [preset: SessionPreset] }>()
const name = ref('')
const profile = ref('')
const directory = ref('')
const target = ref('')
const port = ref('')
const editing = ref(false)
const editingId = ref<string | null>(null)
const error = ref('')
const browsing = ref(false)
let editRevision = 0
const isWindows = /Windows/i.test(navigator.userAgent)
const availableProfiles = computed(() => props.profiles)
watch(() => props.defaultProfile, value => { if (!profile.value) profile.value = value }, { immediate: true })
function startEditing(preset?: SessionPreset) {
  editRevision++
  editingId.value = preset?.id ?? null
  profile.value = preset?.profile ?? (availableProfiles.value.some(p => p.id === props.defaultProfile) ? props.defaultProfile : availableProfiles.value[0]?.id ?? '')
  directory.value = preset?.directory ?? (supportsLocalDefaultDirectory(profile.value) ? props.defaultDirectory : '')
  target.value = preset?.target ?? ''
  port.value = preset?.port ? String(preset.port) : ''
  name.value = preset?.name ?? ''; error.value = ''; editing.value = true
}
async function browseDirectory() {
  if (browsing.value) return
  const initialProfile = profile.value
  const initialEditingId = editingId.value
  const initialRevision = editRevision
  browsing.value = true
  try {
    const selected = await Dialogs.OpenFile({ Title: props.locale === 'zh' ? '选择启动目录' : 'Choose starting directory', CanChooseDirectories: true, CanChooseFiles: false, AllowsMultipleSelection: false })
    if (selected && editing.value && editRevision === initialRevision && profile.value === initialProfile && editingId.value === initialEditingId) directory.value = selected
  } catch (err) { error.value = String(err) }
  finally { browsing.value = false }
}
function save() {
  const cleanName = name.value.trim()
  const cleanDirectory = directory.value
  if (!validPresetName(cleanName) || !availableProfiles.value.some(p => p.id === profile.value)) { error.value = props.locale === 'zh' ? '请输入有效名称并选择 Shell。' : 'Enter a valid name and select a shell.'; return }
  if (cleanDirectory && !localDefaultDirectoryAllowed(profile.value, cleanDirectory, isWindows)) { error.value = props.locale === 'zh' ? '启动目录必须是本机绝对路径。' : 'The starting directory must be an absolute local path.'; return }
  const sshTarget = target.value.trim()
  const sshPort = port.value === '' ? 0 : Number(port.value)
  if (profile.value === 'ssh' && (!validPresetSSHTarget(sshTarget) || !Number.isInteger(sshPort) || sshPort < 0 || sshPort > 65535 || (port.value !== '' && sshPort === 0))) { error.value = props.locale === 'zh' ? '请输入有效 SSH 主机和端口。' : 'Enter a valid SSH host and port.'; return }
  if (!editingId.value && props.presets.length >= 50) { error.value = props.locale === 'zh' ? '最多保存 50 个预设。' : 'Up to 50 presets can be saved.'; return }
  const id = editingId.value ?? (globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`)
  emit('save', profile.value === 'ssh' ? { id, name: cleanName, profile: profile.value, directory: '', target: sshTarget, port: sshPort } : { id, name: cleanName, profile: profile.value, directory: cleanDirectory })
  editing.value = false
}
</script>

<template>
  <details class="presets">
    <summary>{{ locale === 'zh' ? '会话预设' : 'Session presets' }} <span v-if="presets.length">{{ presets.length }}</span></summary>
    <div class="preset-body">
      <div v-for="preset in presets" :key="preset.id" class="preset-row">
        <button type="button" class="preset-launch" :disabled="busy || !connected || !availableProfiles.some(p => p.id === preset.profile)" :title="preset.target || preset.directory || preset.profile" @click="emit('launch', preset)">{{ preset.name }} <small>{{ preset.profile }}</small></button>
        <button type="button" :aria-label="(locale === 'zh' ? '编辑预设 ' : 'Edit preset ') + preset.name" @click="startEditing(preset)">✎</button>
        <button type="button" :aria-label="(locale === 'zh' ? '删除预设 ' : 'Delete preset ') + preset.name" @click="emit('remove', preset.id)">×</button>
      </div>
      <button v-if="!editing" type="button" class="preset-add" @click="startEditing()">{{ locale === 'zh' ? '+ 新建预设' : '+ Add preset' }}</button>
      <form v-else class="preset-form" @submit.prevent="save">
        <input v-model="name" maxlength="64" :aria-label="locale === 'zh' ? '预设名称' : 'Preset name'" :placeholder="locale === 'zh' ? '预设名称' : 'Preset name'" required />
        <select v-model="profile" :aria-label="locale === 'zh' ? 'Shell 类型' : 'Shell type'" @change="directory = ''">
          <option v-for="p in availableProfiles" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
        <template v-if="profile === 'ssh'">
          <input v-model="target" maxlength="255" :aria-label="locale === 'zh' ? 'SSH 主机' : 'SSH host'" :placeholder="locale === 'zh' ? 'SSH 主机或 user@host' : 'SSH host or user@host'" />
          <input v-model="port" type="number" min="1" max="65535" step="1" :aria-label="locale === 'zh' ? 'SSH 端口' : 'SSH port'" :placeholder="locale === 'zh' ? '端口（可留空）' : 'Port (optional)'" />
        </template>
        <div v-if="supportsLocalDefaultDirectory(profile)" class="preset-directory">
          <input v-model="directory" maxlength="512" :aria-label="locale === 'zh' ? '启动目录' : 'Starting directory'" :placeholder="locale === 'zh' ? '启动目录（可留空）' : 'Starting directory (optional)'" />
          <button type="button" :disabled="browsing" @click="browseDirectory">{{ locale === 'zh' ? '浏览…' : 'Browse…' }}</button>
        </div>
        <p v-if="error" role="alert">{{ error }}</p>
        <div><button type="submit" :disabled="browsing">{{ locale === 'zh' ? '保存' : 'Save' }}</button><button type="button" @click="editing = false">{{ locale === 'zh' ? '取消' : 'Cancel' }}</button></div>
      </form>
    </div>
  </details>
</template>

<style scoped>
.presets { margin: 0 0 8px; color: var(--muted); font-size: calc(10px * var(--ui-font-scale)); }
.presets summary { cursor: pointer; padding: 3px 0; }
.preset-body { display: grid; gap: 4px; max-height: 190px; overflow-y: auto; margin-top: 5px; }
.preset-row { display: flex; gap: 3px; }
.preset-row button, .preset-add, .preset-form button { border: 1px solid var(--border); border-radius: 4px; background: var(--surface); color: var(--text); padding: 4px 7px; }
.preset-launch { flex: 1; min-width: 0; text-align: left; overflow: hidden; text-overflow: ellipsis; }
.preset-launch small { color: var(--muted); }
.preset-form { display: grid; gap: 5px; }
.preset-form input, .preset-form select { min-width: 0; width: 100%; padding: 5px; color: var(--text); background: var(--surface); border: 1px solid var(--border); border-radius: 4px; }
.preset-form p { margin: 0; color: var(--negative); }
.preset-form div { display: flex; gap: 5px; }
</style>

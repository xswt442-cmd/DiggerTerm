<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { browseSSHDirectories } from '../sshDirectoryBrowser'
import { remoteChildPath, remoteParentPath } from '../sshRemotePaths'

const props = defineProps<{ sessionId: string; initialPath: string; locale: 'zh' | 'en' }>()
const emit = defineEmits<{ choose: [path: string]; close: [] }>()
const directory = ref('')
const folders = ref<string[]>([])
const truncated = ref(false)
const busy = ref(false)
const error = ref('')
let revision = 0
const label = (zh: string, en: string) => props.locale === 'zh' ? zh : en

async function openDirectory(path: string) {
  if (busy.value) return
  busy.value = true
  error.value = ''
  const id = ++revision
  try {
    const result = await browseSSHDirectories(props.sessionId, path)
    if (id !== revision) return
    directory.value = result.path
    folders.value = result.directories
    truncated.value = result.truncated
  } catch (cause) {
    if (id === revision) error.value = String(cause)
  } finally {
    if (id === revision) busy.value = false
  }
}
function enter(name: string) {
  try { void openDirectory(remoteChildPath(directory.value, name)) }
  catch (cause) { error.value = String(cause) }
}
function choose() {
  if (!busy.value && directory.value.startsWith('/')) emit('choose', directory.value)
}
onMounted(() => { void openDirectory(props.initialPath.startsWith('/') ? props.initialPath : '') })
onBeforeUnmount(() => { revision++ })
</script>

<template>
  <section class="ssh-browser" :aria-label="label('SSH 远程文件夹', 'SSH remote folders')">
    <header class="ssh-browser-header">
      <strong>{{ label('远程目录', 'Remote directories') }}</strong>
      <button type="button" @click="emit('close')">{{ label('关闭', 'Close') }}</button>
    </header>
    <div class="ssh-browser-path" :title="directory">{{ directory || label('正在读取远程主目录…', 'Loading remote home…') }}</div>
    <nav class="ssh-browser-controls" :aria-label="label('目录导航', 'Directory navigation')">
      <button type="button" :disabled="busy || !directory || directory === '/'" @click="openDirectory(remoteParentPath(directory))">↑ {{ label('上级', 'Parent') }}</button>
      <button type="button" :disabled="busy" @click="openDirectory(directory || '')">↻ {{ label('刷新', 'Refresh') }}</button>
    </nav>
    <p v-if="busy" role="status">{{ label('正在读取目录…', 'Loading directories…') }}</p>
    <p v-if="error" role="alert" class="ssh-browser-error">{{ error }}</p>
    <div class="ssh-browser-folders" role="group" :aria-label="label('子目录', 'Subdirectories')">
      <button v-for="folder in folders" :key="folder" type="button" :disabled="busy" @click="enter(folder)">
        <span aria-hidden="true">▸</span> {{ folder }}
      </button>
      <p v-if="!busy && !folders.length && directory">{{ label('没有可见或可访问的子目录。', 'No visible or accessible subdirectories.') }}</p>
    </div>
    <p v-if="truncated" class="ssh-browser-notice">{{ label('目录过多，仅显示部分结果。', 'Too many directories; showing partial results.') }}</p>
    <footer class="ssh-browser-footer">
      <button type="button" :disabled="busy || !directory" @click="choose">{{ label('选择当前目录', 'Select this folder') }}</button>
    </footer>
  </section>
</template>

<style scoped>
.ssh-browser { width: 100%; box-sizing: border-box; display: flex; flex-direction: column; gap: 8px; background: var(--canvas); color: var(--text); border: 1px solid var(--border); padding: 10px; max-height: min(420px, 52vh); min-height: 220px; border-radius: var(--dt-radius, 4px); }
.ssh-browser-header, .ssh-browser-controls, .ssh-browser-footer { display: flex; gap: 8px; justify-content: space-between; align-items: center; }
.ssh-browser-path { overflow-wrap: anywhere; font-family: monospace; font-size: 12px; color: var(--accent); }
.ssh-browser-folders { overflow-y: auto; display: grid; gap: 3px; min-height: 60px; }
.ssh-browser button { cursor: pointer; padding: 5px 9px; text-align: left; background: var(--surface); color: var(--text); border: 1px solid var(--border); border-radius: var(--dt-radius, 4px); }
.ssh-browser button:disabled { opacity: .5; cursor: default; }
.ssh-browser-error { color: var(--negative); overflow-wrap: anywhere; }
.ssh-browser-notice { color: var(--muted); }
.ssh-browser-footer { margin-top: auto; justify-content: flex-end; }
</style>

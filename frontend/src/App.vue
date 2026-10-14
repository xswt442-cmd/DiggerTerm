<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import TerminalPanel from './components/TerminalPanel.vue'
import { Create, Forget, List, Profiles, Terminate } from '../bindings/github.com/xswt442-cmd/DiggerTerm/sessionservice'
import type { Info, Profile } from '../bindings/github.com/xswt442-cmd/DiggerTerm/internal/session/models'

const sessions = ref<Info[]>([])
const profiles = ref<Profile[]>([])
const selectedId = ref<string | null>(null)
const profileId = ref('')
const creating = ref(false)
const errorMessage = ref('')
const activeSession = computed(() => sessions.value.find((s) => s.id === selectedId.value))
let offExit: (() => void) | undefined
let offQuitBlocked: (() => void) | undefined
let offDaemonDisconnected: (() => void) | undefined

function reportError(err: unknown) {
  errorMessage.value = err instanceof Error ? err.message : String(err)
}

async function createSession() {
  if (!profileId.value || creating.value) return
  creating.value = true
  errorMessage.value = ''
  try {
    const info = await Create(profileId.value, 100, 30)
    sessions.value = [...sessions.value, info]
    selectedId.value = info.id
  } catch (err) {
    reportError(err)
  } finally {
    creating.value = false
  }
}

async function terminateSession(info: Info) {
  if (info.status !== 'running') return
  if (!window.confirm(`Terminate ${info.name} (PID ${info.pid})? This will end the terminal session.`)) return
  try {
    await Terminate(info.id)
  } catch (err) {
    reportError(err)
  }
}

async function removeExited(info: Info) {
  if (info.status === 'running') return
  try {
    await Forget(info.id)
    sessions.value = sessions.value.filter((s) => s.id !== info.id)
    if (selectedId.value === info.id) {
      selectedId.value = sessions.value[0]?.id ?? null
    }
  } catch (err) {
    reportError(err)
  }
}

onMounted(async () => {
  offDaemonDisconnected = Events.On('daemon:disconnected', (event) => {
    errorMessage.value = String(event.data || 'Session daemon disconnected')
  })
  offQuitBlocked = Events.On('terminal:quit-blocked', (event) => {
    errorMessage.value = String(event.data)
  })
  offExit = Events.On('terminal:exit', (event) => {
    const info = event.data as Info
    if (!info) return
    sessions.value = sessions.value.map((s) => s.id === info.id ? info : s)
  })
  try {
    const [available, existing] = await Promise.all([Profiles(), List()])
    profiles.value = available ?? []
    profileId.value = profiles.value[0]?.id ?? ''
    sessions.value = existing ?? []
    selectedId.value = sessions.value[0]?.id ?? null
  } catch (err) {
    reportError(err)
  }
})

onBeforeUnmount(() => {
  offExit?.()
  offQuitBlocked?.()
  offDaemonDisconnected?.()
})
</script>

<template>
  <main class="layout">
    <aside class="sidebar">
      <header class="brand">
        <span class="mark" aria-hidden="true">&gt;_</span>
        <div class="brand-copy">
          <strong>DiggerTerm</strong>
          <small>Terminal workspace</small>
        </div>
      </header>

      <div class="section-head">
        <span>MY SESSIONS</span>
        <span class="count">{{ sessions.length }}</span>
      </div>
      <div class="new-session">
        <select v-model="profileId" aria-label="Terminal profile" :disabled="creating">
          <option v-for="p in profiles" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
        <button type="button" class="create" :disabled="!profileId || creating" @click="createSession" title="Create terminal">
          {{ creating ? '…' : '+' }}
        </button>
      </div>

      <nav class="sessions" aria-label="Terminal sessions">
        <div v-if="sessions.length === 0" class="sidebar-empty">No sessions yet. Create one above.</div>
        <div
          v-for="s in sessions"
          :key="s.id"
          class="session-row"
          :class="{ selected: selectedId === s.id }"
        >
          <button
            type="button"
            class="session-select"
            :aria-current="selectedId === s.id ? 'page' : undefined"
            @click="selectedId = s.id"
          >
            <span class="session-name">{{ s.name }} <span class="session-id">#{{ s.id }}</span></span>
            <span class="session-meta"><span :class="['dot', s.status]" /> {{ s.status }} · PID {{ s.pid }}</span>
          </button>
          <button
            type="button"
            class="session-action"
            :title="s.status === 'running' ? 'Terminate session' : 'Remove finished session'"
            :aria-label="s.status === 'running' ? 'Terminate ' + s.name : 'Remove ' + s.name"
            @click="s.status === 'running' ? terminateSession(s) : removeExited(s)"
          >×</button>
        </div>
      </nav>

      <div class="section-head external"><span>EXTERNAL · READ ONLY</span></div>
      <p class="sidebar-empty">Process discovery is planned. No external processes are controlled.</p>
      <footer class="sidebar-foot">GUI + background daemon · v0.2</footer>
    </aside>

    <section class="content">
      <header class="toolbar">
        <span class="current-name">{{ activeSession ? activeSession.name + ' #' + activeSession.id : 'Terminal' }}</span>
        <span class="status" :class="{ live: activeSession?.status === 'running' }">
          {{ activeSession ? activeSession.status : 'No session selected' }}
        </span>
      </header>

      <div v-if="errorMessage" class="error-bar" role="alert">
        <span>{{ errorMessage }}</span>
        <button type="button" aria-label="Dismiss error" @click="errorMessage = ''">×</button>
      </div>

      <div class="terminal-area">
        <template v-for="s in sessions" :key="s.id">
          <TerminalPanel
            :session="s"
            :active="s.id === selectedId"
            @error="reportError"
          />
        </template>

        <div v-if="sessions.length === 0" class="empty-state">
          <span class="prompt" aria-hidden="true">&gt;_</span>
          <h1>Your terminals, one workspace.</h1>
          <p>Create a local terminal from the left. Windows supports PowerShell, CMD, and WSL (when installed).</p>
          <p class="hint">Sessions keep running across GUI exits and reopen from the background daemon. SSH and external discovery are planned.</p>
        </div>
      </div>

      <div class="statusbar">
        <span>Local PTY · {{ sessions.filter((s) => s.status === 'running').length }} running</span>
        <span>Quit UI safely · background sessions persist</span>
      </div>
    </section>
  </main>
</template>

<style>
:root {
  font-family: ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif;
  color: #d4d8df;
  background: #111318;
  font-synthesis: none;
}
* { box-sizing: border-box; }
body { margin: 0; }
button, select { font: inherit; }
button { cursor: pointer; }
button:disabled { cursor: not-allowed; opacity: .6; }
.layout { display: flex; height: 100vh; min-height: 420px; overflow: hidden; }
.sidebar { width: 252px; flex-shrink: 0; display: flex; flex-direction: column; background: #191c23; border-right: 1px solid #2b2e36; padding: 20px 12px 12px; }
.brand { display: flex; align-items: center; gap: 12px; padding: 3px 8px 26px; }
.brand-copy { display: flex; flex-direction: column; }
.brand strong { font-size: 14px; letter-spacing: .015em; }
.brand small { margin-top: 4px; color: #848c98; font-size: 11px; }
.mark { display: grid; place-items: center; width: 34px; height: 34px; background: #292e37; border-radius: 9px; font-family: monospace; font-size: 15px; color: #c0ad91; }
.section-head { display: flex; justify-content: space-between; align-items: center; padding: 0 10px 10px; color: #858b96; font-size: 10px; font-weight: 700; letter-spacing: .08em; }
.count { border: 1px solid #363b45; border-radius: 8px; padding: 1px 6px; font-size: 10px; }
.new-session { display: flex; gap: 5px; padding: 0 5px 14px; }
.new-session select { min-width: 0; flex: 1; border: 1px solid #3a3e48; background: #272b34; color: #dce0e7; border-radius: 7px; padding: 8px 9px; font-size: 12px; }
.new-session .create { width: 34px; border-radius: 7px; border: 1px solid #5b5145; background: #4b4034; color: #f0dfc2; font-size: 19px; }
.sessions { display: flex; flex-direction: column; gap: 4px; overflow-y: auto; min-height: 100px; flex: 1; }
.session-row { display: flex; gap: 2px; align-items: center; border-radius: 7px; background: transparent; }
.session-row:hover { background: #242832; }
.session-row.selected { background: #30343e; }
.session-select { background: transparent; border: 0; padding: 11px 7px 11px 10px; text-align: left; color: #d4d8df; min-width: 0; flex: 1; }
.session-name { display: block; font-size: 12px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.session-id { font-weight: 400; color: #89909c; }
.session-meta { display: flex; align-items: center; gap: 4px; margin-top: 7px; font-size: 10px; color: #969da9; }
.dot { width: 6px; height: 6px; border-radius: 50%; background: #8c929d; }
.dot.running { background: #84ce9b; }
.session-action { border: 0; background: transparent; color: #9fa4ac; padding: 4px 7px; font-size: 17px; }
.session-action:hover { color: #f09797; }
.external { margin-top: 18px; border-top: 1px solid #30333b; padding-top: 17px; }
.sidebar-empty { color: #818895; font-size: 11px; line-height: 1.6; padding: 0 10px; }
.sidebar-foot { color: #767e89; font-size: 10px; padding: 14px 10px 0; }
.content { display: flex; flex: 1; min-width: 0; flex-direction: column; background: #101217; }
.toolbar { display: flex; align-items: center; justify-content: space-between; min-height: 52px; border-bottom: 1px solid #252831; padding: 0 19px; }
.current-name { font-size: 12px; font-weight: 600; }
.status { color: #969ca7; font-size: 11px; text-transform: capitalize; }
.status.live { color: #84ce9b; }
.terminal-area { position: relative; min-height: 0; flex: 1; }
.error-bar { display: flex; justify-content: space-between; padding: 8px 16px; gap: 12px; color: #f0b3b3; background: #3b2227; font-size: 12px; }
.error-bar button { border: 0; background: transparent; color: inherit; font-size: 16px; }
.empty-state { position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 25px; text-align: center; }
.prompt { font-family: monospace; font-size: 30px; color: #b5a083; }
h1 { margin: 16px 0 3px; font-size: 19px; font-weight: 600; }
p { max-width: 370px; margin: 8px 0 0; color: #959ba6; font-size: 12px; line-height: 1.7; }
.hint { color: #777f8b; font-size: 11px; }
.statusbar { display: flex; justify-content: space-between; gap: 12px; border-top: 1px solid #262930; padding: 7px 15px; color: #7f8792; font-size: 10px; }
@media (max-width: 750px) {
  .sidebar { width: 185px; }
  .statusbar span:last-child { display: none; }
}
</style>

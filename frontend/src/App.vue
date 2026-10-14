<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import TerminalPanel from './components/TerminalPanel.vue'
import { Create, Forget, List, Observe, Profiles, Terminate } from '../bindings/github.com/xswt442-cmd/DiggerTerm/sessionservice'
import type { Info, Profile } from '../bindings/github.com/xswt442-cmd/DiggerTerm/internal/session/models'
import type { Process as ObservedProcess, Snapshot as ObservedSnapshot } from '../bindings/github.com/xswt442-cmd/DiggerTerm/internal/observe/models'

const sessions = ref<Info[]>([])
const profiles = ref<Profile[]>([])
const selectedId = ref<string | null>(null)
const profileId = ref('')
const creating = ref(false)
const errorMessage = ref('')
const externalSnapshot = ref<ObservedSnapshot | null>(null)
const externalError = ref('')
const refreshingExternal = ref(false)
const selectedExternalKey = ref<string | null>(null)
const observedProcesses = computed(() => externalSnapshot.value?.processes ?? [])
const activeObserved = computed(() => observedProcesses.value.find((p) => p.key === selectedExternalKey.value))
const activeSession = computed(() => sessions.value.find((s) => s.id === selectedId.value))
let observationTimer: ReturnType<typeof setInterval> | undefined
let offExit: (() => void) | undefined
let offQuitBlocked: (() => void) | undefined
let offDaemonDisconnected: (() => void) | undefined

function reportError(err: unknown) {
  errorMessage.value = err instanceof Error ? err.message : String(err)
}

// Observed PIDs are display-only. They are never sent to Write/Terminate.
function selectManaged(id: string) {
  selectedExternalKey.value = null
  selectedId.value = id
}

function selectObserved(process: ObservedProcess) {
  selectedId.value = null
  selectedExternalKey.value = process.key
}

async function refreshObserved() {
  if (refreshingExternal.value) return
  refreshingExternal.value = true
  try {
    externalSnapshot.value = await Observe()
    externalError.value = ''
  } catch (err) {
    const message = String(err)
    externalError.value = message.includes('unsupported daemon method')
      ? 'Daemon needs updating. Finish your sessions, stop it, then reopen the app.'
      : 'Process inventory unavailable: ' + message
  } finally {
    refreshingExternal.value = false
  }
}

async function createSession() {
  if (!profileId.value || creating.value) return
  creating.value = true
  errorMessage.value = ''
  try {
    const info = await Create(profileId.value, 100, 30)
    sessions.value = [...sessions.value, info]
    selectManaged(info.id)
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
  void refreshObserved()
  observationTimer = setInterval(() => {
    if (!document.hidden) void refreshObserved()
  }, 15000)
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
  if (observationTimer) clearInterval(observationTimer)
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
            @click="selectManaged(s.id)"
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

      <div class="section-head external">
        <span>OBSERVED · READ ONLY</span>
        <button
          type="button"
          class="refresh-action"
          :disabled="refreshingExternal"
          title="Refresh observed processes"
          aria-label="Refresh observed processes"
          @click="refreshObserved"
        >{{ refreshingExternal ? '…' : '↻' }}</button>
      </div>
      <p v-if="externalError" class="sidebar-empty observed-note" role="status">{{ externalError }}</p>
      <p v-else-if="externalSnapshot && !externalSnapshot.supported" class="sidebar-empty observed-note">Windows process discovery only.</p>
      <p v-else-if="!externalSnapshot" class="sidebar-empty observed-note">Scanning processes…</p>
      <p v-else-if="observedProcesses.length === 0" class="sidebar-empty observed-note">No related external processes found.</p>
      <nav class="observed-list" aria-label="Observed external processes">
        <button
          v-for="process in observedProcesses"
          :key="process.key"
          type="button"
          class="observed-row"
          :class="{ selected: selectedExternalKey === process.key }"
          :aria-current="selectedExternalKey === process.key ? 'page' : undefined"
          @click="selectObserved(process)"
        >
          <span class="session-name">{{ process.name }}</span>
          <span class="session-meta">{{ process.kind }} · PID {{ process.pid }}</span>
        </button>
      </nav>
      <p v-if="externalSnapshot?.truncated" class="sidebar-empty observed-note">Showing first 160 results.</p>
      <footer class="sidebar-foot">Observed processes cannot be controlled.</footer>
    </aside>

    <section class="content">
      <header class="toolbar">
        <span class="current-name">{{ selectedExternalKey ? (activeObserved?.name ?? 'Process no longer detected') : (activeSession ? activeSession.name + ' #' + activeSession.id : 'Terminal') }}</span>
        <span class="status" :class="{ live: activeSession?.status === 'running' }">
          {{ selectedExternalKey ? 'Observed · read only' : (activeSession ? activeSession.status : 'No session selected') }}
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

        <div v-if="selectedExternalKey" class="observed-detail">
          <template v-if="activeObserved">
            <div class="observed-kicker">EXTERNAL PROCESS · READ ONLY</div>
            <h1>{{ activeObserved.name }}</h1>
            <dl>
              <div><dt>Process ID</dt><dd>{{ activeObserved.pid }}</dd></div>
              <div><dt>Parent PID</dt><dd>{{ activeObserved.parentPid || 'Unknown' }}</dd></div>
              <div><dt>Classification</dt><dd>{{ activeObserved.kind }} (inferred)</dd></div>
              <div><dt>Started</dt><dd>{{ new Date(activeObserved.startedAt).toLocaleString() }}</dd></div>
            </dl>
            <p>Only process metadata is visible. DiggerTerm does not own this PTY, cannot read its screen, and will not send input or termination signals.</p>
          </template>
          <template v-else>
            <h1>Process no longer detected</h1>
            <p>It may have exited or become inaccessible. Select another process or refresh the list.</p>
          </template>
        </div>

        <div v-if="sessions.length === 0 && !selectedExternalKey" class="empty-state">
          <span class="prompt" aria-hidden="true">&gt;_</span>
          <h1>Your terminals, one workspace.</h1>
          <p>Create a local terminal from the left. Windows supports PowerShell, CMD, and WSL (when installed).</p>
          <p class="hint">Sessions survive GUI exits while the daemon runs. External process metadata can be observed read-only; SSH is planned.</p>
        </div>
      </div>

      <div class="statusbar">
        <span>Local PTY · {{ sessions.filter((s) => s.status === 'running').length }} running · {{ observedProcesses.length }} observed</span>
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
.external { margin-top: 18px; border-top: 1px solid #30333b; padding-top: 12px; padding-bottom: 4px; }
.refresh-action { border: 0; background: transparent; color: #abb0b9; padding: 2px 7px; font-size: 17px; }
.refresh-action:hover { color: #e0e2e6; }
.observed-list { display: flex; flex-direction: column; gap: 2px; flex: 0 1 37%; min-height: 0; overflow: auto; }
.observed-row { display: flex; flex-direction: column; gap: 5px; text-align: left; width: 100%; border: 0; border-radius: 6px; background: transparent; color: #b8bec9; padding: 9px 10px; }
.observed-row:hover { background: #252932; }
.observed-row.selected { background: #30343e; }
.observed-note { margin: 0 0 7px; overflow-wrap: anywhere; }
.observed-detail { position: absolute; inset: 0; overflow: auto; padding: 42px 50px; background: #101217; }
.observed-detail h1 { margin: 10px 0 24px; font-size: 20px; }
.observed-kicker { color: #a49c91; font-size: 11px; font-weight: 600; letter-spacing: .08em; }
.observed-detail dl { display: grid; gap: 0; margin: 0; max-width: 520px; }
.observed-detail dl > div { display: flex; gap: 14px; padding: 11px 0; border-bottom: 1px solid #252831; }
.observed-detail dt { width: 130px; flex-shrink: 0; color: #89909c; font-size: 12px; }
.observed-detail dd { margin: 0; font-family: ui-monospace, Consolas, monospace; font-size: 12px; overflow-wrap: anywhere; }
.observed-detail p { margin-top: 24px; max-width: 560px; }
@media (max-width: 750px) { .observed-detail { padding: 24px; } .observed-detail dt { width: 95px; } }
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

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Events } from '@wailsio/runtime'
import { mergeSessions } from './sessionState'
import TerminalPanel from './components/TerminalPanel.vue'
import PaneWorkspace from './components/PaneWorkspace.vue'
import { resizeSplit } from './workspaceLayout'
import { useWorkspace } from './composables/useWorkspace'
import SettingsDialog from './components/SettingsDialog.vue'
import SettingsContent from './components/SettingsContent.vue'
import DigTree from './components/DigTree.vue'
import { useDig } from './composables/useDig'
import { matchBinding } from './keybindings'
import { Create, CreateLocal, CreateSSH, Forget, List, Profiles, Reconnect, Rename, SSHHosts, Terminate } from '../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/sessionservice'
import type { Info, Profile } from '../bindings/github.com/xswt442-cmd/DiggerTerm/internal/session/models'
import { useSettings } from './composables/useSettings'

const errorMessage = ref('')
function reportError(err: unknown) {
  errorMessage.value = err instanceof Error ? err.message : String(err)
}
const settings = useSettings({ onError: reportError })
const {
  preferences, locale, t, refreshOptions, scrollbackOptions, lineHeightOptions,
  paddingOptions, scrollSpeedOptions, themeOptions, themeDirectory, themeErrors, palette,
  themeRevision, settingsOpen, settingsTab, settingsTabLabels, appVersion, updateStatus,
  updateAvailable, checkingUpdate, isMac, bindings, helpConfigPaths, fontChoices,
  initialize: initializeSettings, refreshThemes, resetLayout,
  resetSection, updateTheme, updateLanguage, checkForUpdates, openReleasePage,
  openThemeDirectory,
} = settings

const kindLabel = (kind: string) => t(kind === 'terminal' ? 'kindTerminal' : kind === 'agent' ? 'kindAgent' : kind === 'descendant' ? 'kindDescendant' : 'kindOther')
function workspaceKey(event: KeyboardEvent) {
  if (settingsOpen.value || event.defaultPrevented) return
  const target = event.target as HTMLElement
  if (!target.classList.contains('xterm-helper-textarea') && target.closest('input, textarea, select, [contenteditable="true"]')) return
  const action = matchBinding(event, bindings.value, 'workspace')
  if (action === 'openSettings') {
    event.preventDefault(); event.stopPropagation(); settingsOpen.value = true
  } else if (action && !selectedExternalKey.value) {
    event.preventDefault(); event.stopPropagation()
    if (action === 'splitRight') splitPane('horizontal')
    else if (action === 'splitDown') splitPane('vertical')
    else if (action === 'focusNextPane') focusNextPane()
    else if (action === 'closePane') closePane(layout.value.focusedPaneId)
  }
}
const paneStack = ref<HTMLElement | null>(null)
const dragging = ref<'width' | 'height' | null>(null)
let dragOrigin = { x: 0, y: 0, width: 0, ratio: 0, height: 1 }

function clamp(value: number, low: number, high: number): number {
  return Math.max(low, Math.min(high, value))
}
function beginDrag(event: PointerEvent, axis: 'width' | 'height') {
  if (event.button !== 0) return
  event.preventDefault()
  dragging.value = axis
  dragOrigin = {
    x: event.clientX, y: event.clientY,
    width: preferences.value.sidebarWidth, ratio: preferences.value.managedRatio,
    height: paneStack.value?.clientHeight ?? 1,
  }
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
}
function onDrag(event: PointerEvent) {
  if (dragging.value === 'width') {
    preferences.value.sidebarWidth = Math.round(clamp(
      dragOrigin.width + event.clientX - dragOrigin.x,
      216, Math.min(500, window.innerWidth - 320),
    ))
  } else if (dragging.value === 'height') {
    preferences.value.managedRatio = clamp(
      dragOrigin.ratio + (event.clientY - dragOrigin.y) / Math.max(1, dragOrigin.height),
      0.20, 0.80,
    )
  }
}
function endDrag() { dragging.value = null }
function dividerKeys(event: KeyboardEvent, axis: 'width' | 'height') {
  const step = event.shiftKey ? 40 : 12
  if (axis === 'width' && (event.key === 'ArrowLeft' || event.key === 'ArrowRight')) {
    event.preventDefault()
    preferences.value.sidebarWidth = Math.round(clamp(preferences.value.sidebarWidth +
      (event.key === 'ArrowRight' ? step : -step), 216, Math.min(500, window.innerWidth - 320)))
  } else if (axis === 'height' && (event.key === 'ArrowUp' || event.key === 'ArrowDown')) {
    event.preventDefault()
    preferences.value.managedRatio = clamp(preferences.value.managedRatio +
      (event.key === 'ArrowDown' ? step : -step) / Math.max(1, paneStack.value?.clientHeight ?? 1), 0.2, 0.8)
  }
}
const localProfiles = computed(() => profiles.value.filter(p => p.id !== 'ssh'))
watch(() => preferences.value.defaultProfile, (value) => {
  if (value !== 'auto' && profiles.value.some(p => p.id === value)) profileId.value = value
})
const sessions = ref<Info[]>([])
const selectedExternalKey = ref<string | null>(null)
const {
  layout, paneLeaves, selectedId, restoreWorkspace, isReady: workspaceReady,
  splitPane, closePane, focusNextPane, focusWorkspacePane, changePaneSession,
  sanitizeForSessions, selectManagedSession,
} = useWorkspace(sessions, selectedExternalKey, reportError)
const daemonConnected = ref(false)
const reconnecting = ref(false)
const connectionRevision = ref(0)
async function reconnectDaemon() {
  if (reconnecting.value) return
  reconnecting.value = true
  daemonConnected.value = false
  try {
    const existing = await Reconnect()
    sessions.value = mergeSessions(sessions.value, existing ?? [], true)
    if (!workspaceReady()) restoreWorkspace(sessions.value)
    else sanitizeForSessions(sessions.value.filter(session => session.status === 'running'))
    connectionRevision.value++
    daemonConnected.value = true
    errorMessage.value = ''
    if (!profiles.value.length) {
      profiles.value = await Profiles() ?? []
      profileId.value = profiles.value.find(profile => profile.id === preferences.value.defaultProfile)?.id ?? profiles.value[0]?.id ?? ''
    }
  } catch (error) { reportError(error) }
  finally { reconnecting.value = false }
}
const profiles = ref<Profile[]>([])
const sessionById = computed(() => new Map(sessions.value.map(session => [session.id, session])))
const profileId = ref('')
const sshTarget = ref('')
const sshPort = ref('')
const sshAliases = ref<string[]>([])
const query = ref('')
const editingSessionId = ref<string | null>(null)
const renameDraft = ref('')
const renameInput = ref<HTMLInputElement | null>(null)
const renaming = ref(false)
const creating = ref(false)
const dig = useDig({
  observeEnabled: computed(() => preferences.value.observeEnabled),
  query,
  selectedExternalKey,
  observeFailedLabel: () => t('observeFailed'),
})
const {
  snapshot: externalSnapshot, error: externalError, loading: refreshingExternal,
  processes: observedProcesses, distributions: observedDistributions,
  sourceFilter: observedSourceFilter, kindFilter: observedKindFilter,
  distributionFilter: observedDistroFilter, filteredGroups: filteredObservedGroups,
  expansionPreferences: observedExpansionPreferences, setExpanded: setObservedExpanded,
  activeProcess: activeObserved, activePath: activeObservedPath, activeNode: activeObservedNode,
  classificationLabel,
} = dig
const filteredSessions = computed(() => sessions.value.filter((s) => (s.name + ' ' + s.id + ' ' + s.profile + ' ' + s.pid).toLowerCase().includes(query.value.trim().toLowerCase())))
const activeSession = computed(() => sessions.value.find((s) => s.id === selectedId.value))
function sessionStatus(session: Info): string {
  if (!daemonConnected.value) return locale.value === 'zh' ? 'Daemon 失联' : 'Daemon disconnected'
  if (session.status === 'exited') return t('exited')
  if (session.streamClosed) return locale.value === 'zh' ? 'PTY 输出已关闭' : 'PTY stream closed'
  return t('running')
}
let observationTimer: ReturnType<typeof setInterval> | undefined
let offExit: (() => void) | undefined
let offRename: (() => void) | undefined
let offQuitBlocked: (() => void) | undefined
let offDaemonDisconnected: (() => void) | undefined
let offState: (() => void) | undefined

// Observed PIDs are display-only. They are never sent to Write/Terminate.
function selectManaged(id: string) {
  if (!sessions.value.some(session => session.id === id)) return
  selectManagedSession(id)
}

const selectObserved = dig.selectProcess
const refreshObserved = dig.refresh

async function createSession() {
  if (!daemonConnected.value || !profileId.value || creating.value) return
  creating.value = true
  errorMessage.value = ''
  try {
    let info: Info
    if (profileId.value === 'ssh') {
      const target = sshTarget.value.trim()
      if (!target) throw new Error(t('sshRequired'))
      const rawPort = String(sshPort.value).trim()
      const port = rawPort === '' ? 0 : Number(rawPort)
      if (!Number.isInteger(port) || port < 0 || port > 65535 || (rawPort !== '' && port === 0)) {
        throw new Error(t('invalidPort'))
      }
      info = await CreateSSH(target, port, 100, 30)
    } else {
      info = preferences.value.defaultDirectory.trim() && ['powershell', 'cmd', 'shell', 'bash'].includes(profileId.value)
        ? await CreateLocal(profileId.value, preferences.value.defaultDirectory.trim(), 100, 30)
        : await Create(profileId.value, 100, 30)
    }
    sessions.value = mergeSessions(sessions.value, [info])
    if (preferences.value.activateNewSession || (!selectedId.value && !selectedExternalKey.value)) {
      selectManaged(info.id)
    }
  } catch (err) {
    const message = String(err)
    if (message.includes('unsupported daemon method')) {
      errorMessage.value = t('errorDaemon')
    } else {
      reportError(err)
    }
  } finally {
    creating.value = false
  }
}

async function startRename(info: Info) {
  editingSessionId.value = info.id
  renameDraft.value = info.name
  await nextTick()
  renameInput.value?.focus()
  renameInput.value?.select()
}

function cancelRename() {
  editingSessionId.value = null
}

async function saveRename() {
  const id = editingSessionId.value
  if (!id || renaming.value) return
  renaming.value = true
  try {
    const changed = await Rename(id, renameDraft.value)
    sessions.value = mergeSessions(sessions.value, [changed])
    editingSessionId.value = null
  } catch (err) {
    reportError(err)
  } finally {
    renaming.value = false
  }
}

async function terminateSession(info: Info) {
  if (info.status !== 'running') return
  if (!window.confirm(t('confirmTerminate', { name: info.name, pid: info.pid }))) return
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
    sanitizeForSessions(sessions.value)
  } catch (err) {
    reportError(err)
  }
}

function startObservationPolling() {
  if (observationTimer) clearInterval(observationTimer)
  observationTimer = undefined
  if (!preferences.value.observeEnabled) {
    if (selectedExternalKey.value) {
      selectedExternalKey.value = null
    }
    return
  }
  if (!document.hidden) void refreshObserved()
  const interval = preferences.value.observeRefreshSeconds
  if (interval > 0) {
    observationTimer = setInterval(() => {
      if (!document.hidden) void refreshObserved()
    }, interval * 1000)
  }
}
watch([() => preferences.value.observeEnabled, () => preferences.value.observeRefreshSeconds],
  () => startObservationPolling())

onMounted(async () => {
  initializeSettings()
  offDaemonDisconnected = Events.On('daemon:disconnected', (event) => {
    daemonConnected.value = false
    errorMessage.value = String(event.data || 'Session daemon disconnected')
  })
  offRename = Events.On('terminal:renamed', (event) => {
    const info = event.data as Info
    if (info?.id) sessions.value = mergeSessions(sessions.value, [info])
  })
  offQuitBlocked = Events.On('terminal:quit-blocked', (event) => {
    errorMessage.value = String(event.data)
  })
  offState = Events.On('terminal:state', (event) => {
    const info = event.data as Info
    if (info?.id) sessions.value = mergeSessions(sessions.value, [info])
  })
  offExit = Events.On('terminal:exit', (event) => {
    const info = event.data as Info
    if (!info) return
    sessions.value = mergeSessions(sessions.value, [info])
  })
  startObservationPolling()
  // Suggestions contain only literal Host aliases; errors never prevent manual SSH entry.
  void SSHHosts().then((aliases) => { sshAliases.value = aliases ?? [] }).catch(() => {})
  try {
    const [available, existing] = await Promise.all([Profiles(), List()])
    profiles.value = available ?? []
    profileId.value = profiles.value.find(p => p.id === preferences.value.defaultProfile)?.id ??
      profiles.value[0]?.id ?? ''
    sessions.value = mergeSessions(sessions.value, existing ?? [], true)
    restoreWorkspace(sessions.value)
    daemonConnected.value = true
  } catch (err) {
    reportError(err)
  }
})

onBeforeUnmount(() => {
  if (observationTimer) clearInterval(observationTimer)
  offExit?.()
  offRename?.()
  offQuitBlocked?.()
  offDaemonDisconnected?.()
  offState?.()
})
</script>

<template>
  <main class="layout" :data-dt-theme="preferences.theme" :style="{ '--ui-font-scale': preferences.uiFontSize / 12 }" @keydown.capture="workspaceKey" @keydown.esc="settingsOpen = false">
    <aside class="sidebar" :style="{ width: preferences.sidebarWidth + 'px' }">
      <header class="brand">
        <img class="brand-logo" src="/favicon.svg" alt="DiggerTerm" width="31" height="31" />
        <div class="brand-copy">
          <strong>DiggerTerm</strong>
          <small>{{ t('workspace') }}</small>
        </div>
      </header>
      <input
        v-model="query"
        class="sidebar-search"
        type="search"
        :placeholder="t('search')"
        :aria-label="t('search')"
        spellcheck="false"
        @keydown.esc.stop="query = ''"
      />
      <div class="section-head main-heading">
        <span>{{ t('managed') }}</span>
        <span class="count">{{ sessions.length }}</span>
      </div>
      <div class="new-session">
        <select v-model="profileId" :aria-label="t('new')" :disabled="creating">
          <option v-for="p in profiles" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
        <button
          v-if="profileId !== 'ssh'"
          type="button"
          class="create"
          :disabled="!daemonConnected || !profileId || creating"
          :title="t('new')"
          :aria-label="t('new')"
          @click="createSession"
        >{{ creating ? '…' : '+' }}</button>
      </div>
      <form v-if="profileId === 'ssh'" class="ssh-connect" :aria-label="t('connect')" @submit.prevent="createSession">
        <input v-model.trim="sshTarget" type="text" spellcheck="false" autocomplete="off"
          maxlength="255" :placeholder="t('sshHost')" list="ssh-host-aliases"
          :aria-label="t('sshHost')" :disabled="creating" required />
        <datalist id="ssh-host-aliases">
          <option v-for="alias in sshAliases" :key="alias" :value="alias" />
        </datalist>
        <div class="ssh-port-row">
          <input v-model="sshPort" type="number" min="1" max="65535" step="1"
            :placeholder="t('sshPort')" :aria-label="t('sshPort')" :disabled="creating" />
          <button type="submit" :disabled="!daemonConnected || creating || !sshTarget.trim()">{{ t('connect') }}</button>
        </div>
        <small :title="t('connectHint')">{{ t('connectHint') }}</small>
      </form>

      <div ref="paneStack" class="pane-stack">
        <section class="managed-pane" :style="{ flex: (preferences.observeEnabled ? preferences.managedRatio : 1) + ' 1 0%' }">
          <nav class="sessions" :aria-label="t('managed')">
            <div v-if="filteredSessions.length === 0" class="sidebar-empty">
              {{ t(query ? 'searchNone' : 'noSessions') }}
            </div>
            <div
              v-for="s in filteredSessions"
              :key="s.id"
              class="session-row"
              :class="{ selected: selectedId === s.id }"
            >
              <template v-if="editingSessionId === s.id">
                <input v-model="renameDraft" ref="renameInput" class="rename-input" type="text"
                  maxlength="64" :aria-label="t('rename')" :disabled="renaming"
                  @keydown.enter.prevent="saveRename" @keydown.esc.stop.prevent="cancelRename" />
                <button type="button" class="session-action save-action" :title="t('save')"
                  :aria-label="t('save')" :disabled="renaming" @click="saveRename">✓</button>
                <button type="button" class="session-action" :title="t('cancel')"
                  :aria-label="t('cancel')" :disabled="renaming" @click="cancelRename">×</button>
              </template>

              <template v-else>
                <button type="button" class="session-select"
                  :aria-current="selectedId === s.id ? 'page' : undefined"
                  @click="selectManaged(s.id)">
                  <span class="session-name">{{ s.name }} <span class="session-id" :title="s.id">#{{ s.id.split('-').at(-1) }}</span></span>
                  <span class="session-meta"><span class="dot" :class="{ running: daemonConnected && s.status === 'running' && !s.streamClosed }" />
                    {{ sessionStatus(s) }}<span v-if="preferences.showPid"> · PID {{ s.pid }}</span>
                  </span>
                </button>
                <button type="button" class="session-action" :title="t('rename')"
                  :aria-label="t('rename') + ' ' + s.name" @click="startRename(s)">✎</button>
                <button type="button" class="session-action" :title="t(s.status === 'running' ? 'terminate' : 'remove')"
                  :disabled="!daemonConnected"
                  :aria-label="t(s.status === 'running' ? 'terminate' : 'remove') + ' ' + s.name"
                  @click="s.status === 'running' ? terminateSession(s) : removeExited(s)">×</button>
              </template>
            </div>
          </nav>
        </section>

        <div v-if="preferences.observeEnabled" class="pane-resizer"
          role="separator" tabindex="0" aria-orientation="horizontal"
          :aria-label="t('heightHandle')"
          :aria-valuenow="Math.round(preferences.managedRatio * 100)"
          aria-valuemin="20" aria-valuemax="80"
          :class="{ dragging: dragging === 'height' }"
          @pointerdown="beginDrag($event, 'height')"
          @pointermove="onDrag" @pointerup="endDrag" @pointercancel="endDrag"
          @keydown="dividerKeys($event, 'height')">
          <span />
        </div>

        <section v-if="preferences.observeEnabled" class="observed-pane" :style="{ flex: (1 - preferences.managedRatio) + ' 1 0%' }">
          <div class="section-head external">
            <span>Dig · {{ t('observed') }}</span>
            <button type="button" class="refresh-action" :disabled="refreshingExternal"
              :title="t('refresh')" :aria-label="t('refresh')" @click="refreshObserved">
              {{ refreshingExternal ? '…' : '↻' }}
            </button>
          </div>
          <div class="dig-filters">
            <label>
              <span class="visually-hidden">{{ locale === 'zh' ? '平台筛选' : 'Platform filter' }}</span>
              <select v-model="observedSourceFilter">
                <option value="all">{{ locale === 'zh' ? '全部平台' : 'All platforms' }}</option>
                <option value="windows">Windows</option><option value="wsl">WSL</option>
              </select>
            </label>
            <label>
              <span class="visually-hidden">{{ locale === 'zh' ? '类型筛选' : 'Type filter' }}</span>
              <select v-model="observedKindFilter">
                <option value="all">{{ locale === 'zh' ? '全部类型' : 'All types' }}</option>
                <option value="terminal">{{ kindLabel('terminal') }}</option>
                <option value="agent">{{ kindLabel('agent') }}</option>
                <option value="descendant">{{ kindLabel('descendant') }}</option>
              </select>
            </label>
            <label v-if="observedDistributions.length">
              <span class="visually-hidden">{{ locale === 'zh' ? 'WSL 发行版筛选' : 'WSL distribution filter' }}</span>
              <select v-model="observedDistroFilter">
                <option value="all">{{ locale === 'zh' ? '全部发行版' : 'All distributions' }}</option>
                <option v-for="distro in observedDistributions" :key="distro" :value="distro">{{ distro }}</option>
              </select>
            </label>
          </div>
          <p v-if="externalError" class="sidebar-empty observed-message" role="status">{{ externalError }}</p>
          <p v-else-if="externalSnapshot && !externalSnapshot.supported" class="sidebar-empty observed-message">{{ t('windowsOnly') }}</p>
          <p v-else-if="!externalSnapshot" class="sidebar-empty observed-message">{{ t('scanning') }}</p>
          <p v-else-if="filteredObservedGroups.length === 0" class="sidebar-empty observed-message">
            {{ t(query ? 'observedSearchNone' : 'noObserved') }}
          </p>
          <DigTree v-else :groups="filteredObservedGroups" :selected-key="selectedExternalKey"
            :show-pid="preferences.showPid" :locale="locale" :label="`Dig ${t('observed')}`"
            :kind-label="kindLabel" :expansion-preferences="observedExpansionPreferences"
            @select="selectObserved" @toggle="setObservedExpanded" />
          <div v-if="externalSnapshot?.truncated" class="pane-note">{{ t('limited') }}</div>
        </section>
      </div>

      <footer class="sidebar-footer">
        <button type="button" class="settings-trigger" :aria-expanded="settingsOpen"
          :aria-label="t('settings')" @click="settingsOpen = !settingsOpen">
          <span class="gear" aria-hidden="true">⚙</span>
          <span>{{ t('settings') }}</span>
          <span class="settings-arrow" aria-hidden="true">{{ settingsOpen ? '⌄' : '›' }}</span>
        </button>
      </footer>

      <SettingsDialog v-model:open="settingsOpen" v-model:tab="settingsTab"
        :title="t('settingsTitle')" :close-label="t('close')" :tab-labels="settingsTabLabels">
        <SettingsContent v-model:preferences="preferences" :tab="settingsTab" :locale="locale" :translate="t"
          :theme-options="themeOptions" :theme-directory="themeDirectory" :theme-errors="themeErrors"
          :font-choices="fontChoices" :scrollback-options="scrollbackOptions" :line-height-options="lineHeightOptions"
          :scroll-speed-options="scrollSpeedOptions" :padding-options="paddingOptions" :refresh-options="refreshOptions"
          :profiles="localProfiles" :reconnecting="reconnecting" :bindings="bindings" :is-mac="isMac"
          :app-version="appVersion" :help-config-paths="helpConfigPaths" :checking-update="checkingUpdate"
          :update-status="updateStatus" :update-available="updateAvailable"
          @theme-change="updateTheme" @language-change="updateLanguage" @refresh-themes="refreshThemes"
          @open-theme-directory="openThemeDirectory" @reset-layout="resetLayout" @reset-section="resetSection"
          @reconnect="reconnectDaemon" @keybindings-change="bindings = $event"
          @check-updates="checkForUpdates" @open-release-page="openReleasePage" />
      </SettingsDialog>
    </aside>

    <div class="sidebar-resizer"
      role="separator" tabindex="0" aria-orientation="vertical"
      :aria-label="t('widthHandle')"
      :aria-valuenow="preferences.sidebarWidth" aria-valuemin="216" aria-valuemax="500"
      :class="{ dragging: dragging === 'width' }"
      @pointerdown="beginDrag($event, 'width')"
      @pointermove="onDrag" @pointerup="endDrag" @pointercancel="endDrag"
      @keydown="dividerKeys($event, 'width')">
      <span />
    </div>

    <section class="content">
      <header class="toolbar">
        <span class="current-name">{{ selectedExternalKey ? (activeObserved?.name ?? t('processGone')) : (activeSession?.name ?? t('terminal')) }}</span>
        <div v-if="!selectedExternalKey" class="workspace-actions">
          <button type="button" @click="splitPane('horizontal')">{{ locale === 'zh' ? '向右分屏' : 'Split right' }}</button>
          <button type="button" @click="splitPane('vertical')">{{ locale === 'zh' ? '向下分屏' : 'Split down' }}</button>
          <button type="button" @click="focusNextPane">{{ locale === 'zh' ? '下一面板' : 'Next pane' }}</button>
        </div>
        <span class="status" :class="{ live: daemonConnected && activeSession?.status === 'running' && !activeSession.streamClosed }">
          {{ selectedExternalKey ? t('observedStatus') : !daemonConnected ? (locale === 'zh' ? 'Daemon 连接不可用' : 'Daemon unavailable') : (activeSession ? sessionStatus(activeSession) : t('noSelection')) }}
        </span>
      </header>
      <div v-if="errorMessage" class="error-bar" role="alert">
        <span>{{ errorMessage }}</span>
        <button v-if="!daemonConnected" type="button" :disabled="reconnecting" @click="reconnectDaemon">{{ locale === 'zh' ? '重连 Daemon' : 'Reconnect daemon' }}</button>
        <button type="button" aria-label="Dismiss error" @click="errorMessage = ''">×</button>
      </div>
      <div class="terminal-area">
        <PaneWorkspace v-show="!selectedExternalKey" :layout="layout"
          @focus="focusWorkspacePane" @resize="(id, ratio) => layout = resizeSplit(layout, id, ratio)">
          <template #default="{ pane, focused }">
            <header class="workspace-pane-head">
              <select :value="pane.sessionId ?? ''" :aria-label="locale === 'zh' ? '选择托管会话' : 'Choose managed session'"
                @change="changePaneSession(pane.paneId, $event)">
                <option value="">{{ locale === 'zh' ? '选择会话（仅显示）' : 'Choose session' }}</option>
                <option v-for="session in sessions" :key="session.id" :value="session.id">{{ session.name }} · {{ sessionStatus(session) }}</option>
              </select>
              <button type="button" :disabled="paneLeaves.length <= 1" :aria-label="locale === 'zh' ? '关闭面板，保留会话' : 'Close pane, keep session'" @click="closePane(pane.paneId)">×</button>
            </header>
            <div class="workspace-pane-body">
              <TerminalPanel v-if="pane.sessionId && sessionById.has(pane.sessionId)" :key="pane.sessionId + ':' + connectionRevision"
                :session="sessionById.get(pane.sessionId)!" :active="!selectedExternalKey" :focused="focused && !settingsOpen" :connected="daemonConnected"
                :font-size="preferences.fontSize" :font-family="preferences.fontFamily"
                :line-height="preferences.lineHeight" :scrollback="preferences.scrollback"
                :cursor-style="preferences.cursorStyle" :cursor-blink="preferences.cursorBlink"
                :keyboard-mode="preferences.keyboardMode"
                :scroll-sensitivity="preferences.scrollSensitivity"
                :right-click-selects-word="preferences.rightClickSelectsWord"
                :padding="preferences.terminalPadding"
                :colors="palette.terminal" :theme-revision="themeRevision" :locale="locale" :bindings="bindings" @error="reportError" />
              <p v-else-if="sessions.length > 0" class="pane-placeholder">{{ locale === 'zh' ? '从侧栏或上方选择托管会话。关闭面板仅分离视图。' : 'Choose a managed session here or in the sidebar. Closing a pane only detaches its view.' }}</p>
            </div>
          </template>
        </PaneWorkspace>
        <div v-if="selectedExternalKey" class="observed-detail">
          <template v-if="activeObserved">
            <div class="observed-kicker">{{ t('readOnly') }}</div>
            <h1>{{ activeObserved.name }}</h1>
            <p v-if="externalError" class="observed-warning" role="status">{{ locale === 'zh' ? '最近一次扫描失败，以下信息来自上次成功扫描。' : 'The latest scan failed. Details below are from the last successful scan.' }}</p>
            <dl>
              <div><dt>{{ t('source') }}</dt><dd>{{ activeObserved.source === 'wsl' ? 'WSL · ' + activeObserved.distribution : 'Windows' }}</dd></div>
              <div><dt>{{ t('pid') }}</dt><dd>{{ activeObserved.pid }}{{ activeObserved.source === 'wsl' ? ' (Linux)' : '' }}</dd></div>
              <div><dt>{{ t('parent') }}</dt><dd>{{ activeObserved.parentPid > 0 ? activeObserved.parentPid : (locale === 'zh' ? '无' : 'None') }}</dd></div>
              <div><dt>{{ t('kind') }}</dt><dd>{{ kindLabel(activeObserved.kind) }} · {{ classificationLabel(activeObserved) === 'identified' ? (locale === 'zh' ? '按名称识别' : 'Identified by name') : classificationLabel(activeObserved) === 'inferred' ? (locale === 'zh' ? '根据父进程推断' : 'Inferred from parent process') : t('unknown') }}</dd></div>
              <div><dt>{{ t('started') }}</dt><dd>{{ new Date(activeObserved.startedAt).toLocaleString() }}</dd></div>
              <div><dt>{{ locale === 'zh' ? '父进程链' : 'Parent chain' }}</dt><dd>
                <span v-if="activeObservedPath.length > 1">{{ activeObservedPath.slice(0, -1).map(node => `${node.name} (PID ${node.pid})`).join(' › ') }}</span>
                <span v-else-if="activeObservedNode?.parentMissing">{{ locale === 'zh' ? '父进程未显示' : 'Parent not shown' }}</span>
                <span v-else-if="activeObserved.parentPid === 0">{{ locale === 'zh' ? '无' : 'None' }}</span>
                <span v-else>{{ t('unknown') }}</span>
              </dd></div>
              <div v-if="externalSnapshot?.scannedAt"><dt>{{ locale === 'zh' ? '最近扫描' : 'Last scan' }}</dt><dd>{{ new Date(externalSnapshot.scannedAt).toLocaleString() }}</dd></div>
            </dl>
            <p>{{ t('readOnlyText') }}</p>
          </template>
          <template v-else>
            <div class="observed-kicker">{{ t('readOnly') }}</div>
            <h1>{{ locale === 'zh' ? '未在最新扫描中发现' : 'Not found in the latest scan' }}</h1>
            <p>{{ t('processGoneDetail') }}</p>
            <button type="button" @click="selectedExternalKey = null">{{ locale === 'zh' ? '清除选择' : 'Clear selection' }}</button>
          </template>
        </div>
        <div v-if="sessions.length === 0 && !selectedExternalKey && paneLeaves.length === 1" class="empty-state">
          <img src="/favicon.svg" alt="" width="36" height="36" />
          <h1>{{ t('emptyTitle') }}</h1>
          <p>{{ t('emptyDesc') }}</p>
          <p>{{ t('emptyHint') }}</p>
        </div>
      </div>
      <div v-if="preferences.showStatusBar" class="statusbar">
        <span>{{ daemonConnected ? t('activeCount', { count: sessions.filter(s => s.status === 'running').length }) : (locale === 'zh' ? '会话状态未知' : 'Session state unknown') }} · {{ t('observedCount', { count: observedProcesses.length }) }}</span>
        <span>{{ t('persisted') }}</span>
      </div>
    </section>
  </main>
</template>

<style>
:root { font-family: "Segoe UI", system-ui, -apple-system, sans-serif; background: #101318; font-synthesis: none; }
* { box-sizing: border-box; }
html, body, #app { margin: 0; width: 100%; height: 100%; overflow: hidden; }
button, select, input { font: inherit; }
button { cursor: pointer; }
button:disabled { cursor: not-allowed; opacity: .55; }
button:focus-visible, input:focus-visible, select:focus-visible, [role="separator"]:focus-visible {
  outline: 2px solid var(--accent); outline-offset: -1px;
}
.layout { display: flex; height: 100vh; width: 100%; min-height: 0; overflow: hidden; background: var(--canvas); color: var(--text); font-family: var(--dt-ui-font, "Segoe UI", system-ui, sans-serif); font-size: calc(12px * var(--ui-font-scale)); }
.layout button, .layout input, .layout select { border-radius: var(--dt-radius, 5px); }
.workspace-actions { display: flex; gap: 5px; flex-wrap: wrap; }
.workspace-actions button, .workspace-pane-head button, .workspace-pane-head select { color: var(--text); background: var(--surface); border: 1px solid var(--border); padding: 4px; font-size: 10px; }
.workspace-pane-head { height: 30px; display: flex; gap: 5px; padding: 3px; align-items: center; border-bottom: 1px solid var(--border-faint); }
.workspace-pane-head select { min-width: 0; flex: 1; }
.workspace-pane-body { position: absolute; inset: 30px 0 0; }
.pane-placeholder { color: var(--muted); font-size: 12px; padding: 10px; }
.sidebar { position: relative; display: flex; flex-direction: column; flex: 0 0 auto; min-width: 216px; max-width: 500px; min-height: 0; background: var(--sidebar); padding: 11px 10px 0; z-index: 2; }
.brand { display: flex; align-items: center; gap: 9px; height: 43px; padding: 0 5px; flex-shrink: 0; }
.brand-logo { display: block; object-fit: contain; flex-shrink: 0; }
.brand-copy { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.brand strong { font-size: calc(13px * var(--ui-font-scale)); font-weight: 650; letter-spacing: .005em; white-space: nowrap; }
.brand small { color: var(--muted); font-size: calc(10px * var(--ui-font-scale)); white-space: nowrap; }
.sidebar-search { flex: 0 0 auto; height: 29px; width: 100%; margin: 7px 0 11px; padding: 0 9px; border: 1px solid var(--border); border-radius: 6px; background: var(--surface); color: var(--text); font-size: calc(11px * var(--ui-font-scale)); }
.sidebar-search::placeholder { color: var(--faint); }
.section-head { display: flex; justify-content: space-between; align-items: center; min-height: 23px; flex-shrink: 0; color: var(--muted); font-size: calc(10px * var(--ui-font-scale)); font-weight: 650; letter-spacing: .055em; padding: 0 6px; text-transform: uppercase; }
.main-heading { margin-bottom: 4px; }
.count { color: var(--muted); border: 1px solid var(--border); border-radius: 9px; padding: 0 6px; font-size: calc(10px * var(--ui-font-scale)); line-height: 16px; }
.new-session { flex-shrink: 0; display: flex; align-items: stretch; gap: 6px; height: 31px; margin-bottom: 8px; }
.new-session select { min-width: 0; flex: 1; height: 31px; border: 1px solid var(--border); background: var(--surface); color: var(--text); border-radius: 6px; padding: 0 8px; font-size: calc(11px * var(--ui-font-scale)); }
.new-session .create { flex: 0 0 32px; border: 1px solid var(--border); border-radius: 6px; background: var(--accent-bg); color: var(--accent-text); font-size: calc(18px * var(--ui-font-scale)); font-weight: 400; }
.new-session .create:hover, .ssh-port-row button:hover { filter: brightness(1.1); }
.ssh-connect { display: flex; flex-direction: column; gap: 5px; flex-shrink: 0; padding-bottom: 9px; }
.ssh-connect input { width: 100%; min-width: 0; height: 29px; padding: 0 8px; background: var(--surface); color: var(--text); border: 1px solid var(--border); border-radius: 5px; font-size: calc(11px * var(--ui-font-scale)); }
.ssh-connect input::placeholder { color: var(--faint); }
.ssh-port-row { display: flex; gap: 5px; }
.ssh-port-row input { flex: 1; }
.ssh-port-row button { flex: 0 0 auto; border: 1px solid var(--border); border-radius: 5px; padding: 0 9px; background: var(--accent-bg); color: var(--accent-text); font-size: calc(11px * var(--ui-font-scale)); }
.ssh-connect small { color: var(--faint); font-size: calc(10px * var(--ui-font-scale)); line-height: 1.25; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pane-stack { display: flex; flex-direction: column; flex: 1 1 0; min-height: 0; }
.managed-pane, .observed-pane { display: flex; flex-direction: column; min-height: 70px; min-width: 0; overflow: hidden; }
.sessions { min-height: 0; flex: 1 1 0; overflow-x: hidden; overflow-y: auto; scrollbar-width: thin; scrollbar-color: var(--border) transparent; }
.sessions::-webkit-scrollbar { width: 6px; }
.sessions::-webkit-scrollbar-thumb { background: var(--border); border-radius: 6px; }
.session-row { display: flex; align-items: center; gap: 1px; width: 100%; min-height: 40px; padding: 0 2px; border-radius: 5px; }
.session-row:hover, .observed-row:hover { background: var(--surface-hover); }
.session-row.selected, .observed-row.selected { background: var(--surface-active); }
.session-select { display: flex; flex-direction: column; justify-content: center; flex: 1 1 0; min-width: 0; min-height: 40px; border: 0; background: none; color: var(--text); text-align: left; padding: 5px 5px 5px 7px; }
.session-name { display: block; width: 100%; min-width: 0; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; font-size: calc(11px * var(--ui-font-scale)); font-weight: 620; }
.session-id { color: var(--muted); font-weight: 400; }
.session-meta { display: flex; align-items: center; gap: 4px; min-width: 0; margin-top: 3px; font-size: calc(10px * var(--ui-font-scale)); color: var(--muted); }
.dot { display: inline-block; flex: 0 0 6px; width: 6px; height: 6px; border-radius: 100%; background: var(--faint); }
.dot.running { background: var(--positive); }
.session-action { display: grid; place-items: center; flex: 0 0 22px; height: 30px; padding: 0; border: 0; background: transparent; color: var(--muted); font-size: calc(14px * var(--ui-font-scale)); border-radius: 4px; }
.session-action:hover { color: var(--negative); background: var(--surface); }
.session-action.save-action:hover { color: var(--positive); }
.rename-input { min-width: 0; flex: 1 1 0; height: 28px; margin-left: 4px; padding: 0 6px; color: var(--text); background: var(--surface); border: 1px solid var(--accent); border-radius: 4px; font-size: calc(11px * var(--ui-font-scale)); }
.sidebar-empty { font-size: calc(11px * var(--ui-font-scale)); line-height: 1.4; padding: 8px 6px; color: var(--faint); }
.pane-resizer { display: flex; align-items: center; justify-content: center; flex: 0 0 9px; min-height: 9px; border-top: 1px solid var(--border-faint); cursor: row-resize; touch-action: none; user-select: none; }
.pane-resizer span { display: block; width: 30px; height: 3px; border-radius: 4px; background: var(--border); opacity: .35; }
.pane-resizer:hover span, .pane-resizer.dragging span, .pane-resizer:focus-visible span { background: var(--accent); opacity: 1; }
.external { margin: 2px 0 5px; }
.external > span { overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.refresh-action { display: grid; place-items: center; border: 0; padding: 0; height: 20px; width: 22px; background: transparent; color: var(--muted); font-size: calc(16px * var(--ui-font-scale)); }
.refresh-action:hover { color: var(--text); }
.observed-row { display: flex; flex-direction: column; justify-content: center; flex: 0 0 auto; width: 100%; min-height: 37px; padding: 5px 7px; text-align: left; border: 0; border-radius: 5px; background: transparent; color: var(--text); }
.observed-source { display: inline; margin-left: 4px; padding: 1px 3px; border: 1px solid var(--border); border-radius: 3px; color: var(--muted); font-size: calc(9px * var(--ui-font-scale)); font-weight: 400; }
.dig-filters { display: flex; flex-wrap: wrap; gap: 4px; padding: 0 3px 5px; }
.dig-filters select { max-width: 100%; height: 23px; padding: 0 4px; border: 1px solid var(--border); border-radius: 4px; background: var(--surface); color: var(--muted); font-size: calc(9px * var(--ui-font-scale)); }
.observed-warning { color: var(--accent); font-size: calc(11px * var(--ui-font-scale)); }
.visually-hidden { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }
.observed-message { padding-top: 3px; padding-bottom: 3px; }
.pane-note { font-size: calc(10px * var(--ui-font-scale)); color: var(--faint); padding: 4px 6px 2px; flex: 0 0 auto; }
.sidebar-footer { position: relative; flex: 0 0 auto; border-top: 1px solid var(--border-faint); padding: 5px 0 6px; background: var(--sidebar); }
.settings-trigger { display: flex; gap: 9px; width: 100%; align-items: center; height: 31px; padding: 0 8px; background: transparent; color: var(--muted); border: 0; border-radius: 5px; text-align: left; font-size: calc(11px * var(--ui-font-scale)); }
.settings-trigger:hover, .settings-trigger[aria-expanded="true"] { color: var(--text); background: var(--surface-hover); }
.gear { font-size: calc(17px * var(--ui-font-scale)); line-height: 1; }
.settings-arrow { margin-left: auto; font-size: calc(17px * var(--ui-font-scale)); }
.sidebar-resizer { position: relative; display: flex; justify-content: center; align-items: center; flex: 0 0 7px; width: 7px; background: var(--border-faint); touch-action: none; cursor: col-resize; user-select: none; z-index: 1; }
.sidebar-resizer span { display: block; width: 3px; height: 30px; background: var(--border); border-radius: 4px; opacity: .25; }
.sidebar-resizer:hover span, .sidebar-resizer.dragging span, .sidebar-resizer:focus-visible span { background: var(--accent); opacity: 1; }
.content { flex: 1 1 0; min-width: 0; min-height: 0; display: flex; flex-direction: column; background: var(--canvas); }
.toolbar { flex: 0 0 42px; min-height: 42px; display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 0 15px; border-bottom: 1px solid var(--border-faint); }
.current-name { overflow: hidden; white-space: nowrap; text-overflow: ellipsis; font-size: calc(12px * var(--ui-font-scale)); font-weight: 600; }
.status { flex: 0 0 auto; font-size: calc(11px * var(--ui-font-scale)); color: var(--muted); }
.status.live { color: var(--positive); }
.error-bar { display: flex; justify-content: space-between; align-items: center; flex: 0 0 auto; gap: 12px; padding: 6px 12px; color: var(--negative); background: var(--surface); font-size: calc(11px * var(--ui-font-scale)); }
.error-bar button { color: inherit; background: transparent; border: 0; font-size: calc(16px * var(--ui-font-scale)); }
.terminal-area { position: relative; min-height: 0; flex: 1 1 0; background: var(--canvas); }
.observed-detail { position: absolute; inset: 0; overflow: auto; padding: 35px 38px; background: var(--canvas); }
.observed-kicker { color: var(--accent); font-size: calc(10px * var(--ui-font-scale)); font-weight: 600; letter-spacing: .07em; }
.observed-detail h1 { font-size: calc(19px * var(--ui-font-scale)); margin: 11px 0 22px; }
.observed-detail dl { display: grid; gap: 0; margin: 0; max-width: 540px; }
.observed-detail dl > div { display: flex; gap: 12px; padding: 10px 0; border-bottom: 1px solid var(--border-faint); }
.observed-detail dt { flex: 0 0 118px; color: var(--muted); font-size: calc(11px * var(--ui-font-scale)); }
.observed-detail dd { margin: 0; font-family: Consolas, monospace; font-size: calc(11px * var(--ui-font-scale)); color: var(--text); overflow-wrap: anywhere; }
.observed-detail p { max-width: 560px; margin-top: 22px; }
.empty-state { position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 25px; text-align: center; }
.empty-state h1 { margin: 14px 0 6px; font-size: calc(17px * var(--ui-font-scale)); }
.empty-state p { margin: 0; font-size: calc(12px * var(--ui-font-scale)); color: var(--muted); line-height: 1.35; max-width: 380px; }
.empty-state p + p { margin-top: 2px; }
.statusbar { flex: 0 0 26px; height: 26px; display: flex; align-items: center; justify-content: space-between; gap: 14px; padding: 0 12px; border-top: 1px solid var(--border-faint); color: var(--faint); font-size: calc(10px * var(--ui-font-scale)); }
@media (max-width: 820px) { .statusbar span:last-child { display: none; } }
@media (max-height: 480px) { .brand { height: 34px; } .sidebar-search { margin: 3px 0 6px; } }
</style>

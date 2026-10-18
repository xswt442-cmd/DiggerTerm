import { computed, ref, watch, type Ref } from 'vue'
import { Observe, ObserveCapabilities } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/sessionservice'
import type { Process, Snapshot } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/observe/models'
import { DigWatchState, type WatchChange, type WatchSource } from '../dig/watchState'
import {
  buildProcessTree, classificationLabel, filterProcessTree,
  type ProcessGroup, type ProcessNode,
} from '../dig/processTree'

export interface UseDigOptions {
  observeEnabled: Ref<boolean>
  query: Ref<string>
  selectedExternalKey: Ref<string | null>
  observeFailedLabel: () => string
  connectionRevision?: Ref<number>
  watchUnsupportedLabel?: () => string
  watchPlatformUnsupportedLabel?: () => string
}

function findProcessPath(groups: ProcessGroup[], key: string | null): ProcessNode[] {
  if (!key) return []
  const visit = (node: ProcessNode, parents: ProcessNode[]): ProcessNode[] => {
    const path = [...parents, node]
    if (node.key === key) return path
    for (const child of node.children) {
      const childPath = visit(child, path)
      if (childPath.length) return childPath
    }
    return []
  }
  for (const group of groups) {
    for (const root of group.roots) {
      const path = visit(root, [])
      if (path.length) return path
    }
  }
  return []
}

export function useDig(options: UseDigOptions) {
  const snapshot = ref<Snapshot | null>(null)
  const error = ref('')
  const loading = ref(false)
  const sourceFilter = ref('all')
  const kindFilter = ref('all')
  const distributionFilter = ref('all')
  const expansionPreferences = ref<Record<string, boolean>>({})
  const watchRevision = ref(0)
  const watchState = new DigWatchState(() => { watchRevision.value++ })
  const watchSupported = ref(false)
  const trustedSnapshotAvailable = ref(false)
  const watchStatus = ref('')
  const watchSources = computed<WatchSource[]>(() => { void watchRevision.value; return watchState.sources() })
  const watchChanges = computed<WatchChange[]>(() => { void watchRevision.value; return watchState.changesList() })
  const watchedProcesses = computed<Process[]>(() => { void watchRevision.value; return watchState.watchedProcesses() })
  const watchedCount = computed(() => watchedProcesses.value.length)
  const watchFilter = ref<'all' | 'new' | 'not-seen' | 'changed'>('all')
  const filteredWatchChanges = computed(() => watchFilter.value === 'all'
    ? watchChanges.value
    : watchChanges.value.filter(change => change.kind === watchFilter.value))
  let capabilitiesChecked = false
  let capabilitiesRevision = -1
  let scanRequest = 0
  let refreshPending = false
  async function checkCapabilities() {
    const revision = options.connectionRevision?.value ?? 0
    if (capabilitiesChecked && capabilitiesRevision === revision) return
    try {
      const capabilities = await ObserveCapabilities()
      if ((options.connectionRevision?.value ?? 0) !== revision) return
      watchSupported.value = capabilities.observeSourceResults
      watchStatus.value = capabilities.observeSourceResults ? '' : (options.watchUnsupportedLabel?.() ?? 'Dig Watch needs a newer daemon.')
      capabilitiesRevision = revision
      capabilitiesChecked = true
    } catch (error) {
      const message = String(error)
      if (message.includes('unsupported daemon method')) {
        watchSupported.value = false
        watchStatus.value = options.watchUnsupportedLabel?.() ?? 'Dig Watch needs a newer daemon.'
        capabilitiesRevision = revision
        capabilitiesChecked = true
      } else {
        watchSupported.value = false
        watchStatus.value = options.observeFailedLabel() + ': ' + message
        capabilitiesChecked = false
      }
    }
  }

  const processes = computed<Process[]>(() => {
    if (trustedSnapshotAvailable.value) {
      void watchRevision.value
      return watchState.trustedProcesses()
    }
    return snapshot.value?.processes ?? []
  })
  const groups = computed(() => buildProcessTree(processes.value))
  const distributions = computed(() => [...new Set(processes.value
    .filter(process => process.source === 'wsl')
    .map(process => process.distribution))].sort((a, b) => a.localeCompare(b)))
  const filterActive = computed(() => Boolean(options.query.value.trim()) ||
    sourceFilter.value !== 'all' || kindFilter.value !== 'all' || distributionFilter.value !== 'all')
  const filteredGroups = computed(() => {
    const term = options.query.value.trim().toLowerCase()
    return filterProcessTree(groups.value, process => {
      const textMatches = !term || `${process.name} ${process.kind} ${process.pid} ${process.parentPid} ${process.distribution}`.toLowerCase().includes(term)
      const sourceMatches = sourceFilter.value === 'all' || process.source === sourceFilter.value
      const kindMatches = kindFilter.value === 'all' || process.kind === kindFilter.value
      const distributionMatches = distributionFilter.value === 'all' || (process.source === 'wsl' && process.distribution === distributionFilter.value)
      return textMatches && sourceMatches && kindMatches && distributionMatches
    }).map(group => ({ ...group, expandFilteredNodes: filterActive.value }))
  })
  const activeProcess = computed(() => processes.value.find(process => process.key === options.selectedExternalKey.value))
  const activePath = computed(() => findProcessPath(groups.value, options.selectedExternalKey.value))
  const activeNode = computed(() => activePath.value.at(-1))

  async function refresh() {
    if (loading.value || !options.observeEnabled.value) return
    const request = ++scanRequest
    const connection = options.connectionRevision?.value ?? 0
    loading.value = true
    try {
      await checkCapabilities()
      if (request !== scanRequest || !options.observeEnabled.value ||
          connection !== (options.connectionRevision?.value ?? 0)) return
      const nextSnapshot = await Observe()
      if (request !== scanRequest || !options.observeEnabled.value ||
          connection !== (options.connectionRevision?.value ?? 0)) return
      if (!nextSnapshot.supported) {
        watchSupported.value = false
        watchStatus.value = options.watchPlatformUnsupportedLabel?.() ?? 'Dig Watch is unavailable on this platform.'
      }
      snapshot.value = nextSnapshot
      if (watchSupported.value && nextSnapshot.supported) {
        if (nextSnapshot.sources?.length) {
          watchState.update({ ...nextSnapshot, processes: nextSnapshot.processes ?? [] })
          trustedSnapshotAvailable.value = true
        } else {
          watchState.failScan('source-status-missing', nextSnapshot.scannedAt)
        }
      } else if (!watchSupported.value && !nextSnapshot.sources?.length) {
        trustedSnapshotAvailable.value = false
      }
      const currentKeys = new Set<string>()
      const collectKeys = (node: ProcessNode) => {
        currentKeys.add(node.key)
        node.children.forEach(collectKeys)
      }
      buildProcessTree(processes.value).forEach(group => group.roots.forEach(collectKeys))
      for (const key of Object.keys(expansionPreferences.value)) {
        if (!currentKeys.has(key)) delete expansionPreferences.value[key]
      }
      error.value = ''
    } catch (err) {
      if (request !== scanRequest || !options.observeEnabled.value ||
          connection !== (options.connectionRevision?.value ?? 0)) return
      const message = String(err)
      if (watchSupported.value) watchState.failScan('scan-failed')
      error.value = message.includes('unsupported daemon method')
        ? 'Daemon needs updating. Finish your sessions, stop it, then reopen the app.'
        : options.observeFailedLabel() + ': ' + message
    } finally {
      loading.value = false
      if (refreshPending && options.observeEnabled.value) {
        refreshPending = false
        void refresh()
      }
    }
  }

  function selectProcess(process: Process) {
    options.selectedExternalKey.value = process.key
  }

  function clearSelection() {
    options.selectedExternalKey.value = null
  }

  function setExpanded(key: string, expanded: boolean) {
    expansionPreferences.value[key] = expanded
  }

  function toggleWatched(key: string) { return watchState.toggleWatch(key) }
  function isWatched(key: string) { return watchState.isWatched(key) }
  function followedState(key: string) { return watchState.followedState(key) }
  function clearWatchChanges() { watchState.clearChanges() }
  function pruneWatchChanges() { watchState.pruneExpired() }

  watch([() => options.connectionRevision?.value, () => options.observeEnabled.value], (next, previous) => {
    const connectionChanged = next[0] !== previous[0]
    const enabled = next[1]
    const enabledAgain = !previous[1] && enabled
    if (!enabled) refreshPending = false
    else if (connectionChanged || enabledAgain) refreshPending = true
    if (!enabled || connectionChanged || enabledAgain) scanRequest++
    if (!enabled) watchState.pause()
    if (!connectionChanged) return
    capabilitiesChecked = false
    watchSupported.value = false
    trustedSnapshotAvailable.value = false
    watchStatus.value = ''
    watchState.pause('connection-changed')
    if (enabled && !loading.value && refreshPending) {
      refreshPending = false
      void refresh()
    }
  })

  return {
    snapshot, error, loading, processes, groups, distributions,
    sourceFilter, kindFilter, distributionFilter, filteredGroups,
    expansionPreferences, setExpanded,
    watchSupported, watchStatus, watchSources, watchChanges, filteredWatchChanges,
    watchedProcesses, watchedCount, watchFilter, toggleWatched, isWatched, followedState, clearWatchChanges, pruneWatchChanges,
    activeProcess, activePath, activeNode, classificationLabel,
    refresh, selectProcess, clearSelection,
  }
}

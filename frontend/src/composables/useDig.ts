import { computed, ref, type Ref } from 'vue'
import { Observe } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/sessionservice'
import type { Process, Snapshot } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/observe/models'
import {
  buildProcessTree, classificationLabel, filterProcessTree,
  type ProcessGroup, type ProcessNode,
} from '../dig/processTree'

export interface UseDigOptions {
  observeEnabled: Ref<boolean>
  query: Ref<string>
  selectedExternalKey: Ref<string | null>
  observeFailedLabel: () => string
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

  const processes = computed<Process[]>(() => snapshot.value?.processes ?? [])
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
    loading.value = true
    try {
      snapshot.value = await Observe()
      error.value = ''
    } catch (err) {
      const message = String(err)
      error.value = message.includes('unsupported daemon method')
        ? 'Daemon needs updating. Finish your sessions, stop it, then reopen the app.'
        : options.observeFailedLabel() + ': ' + message
    } finally {
      loading.value = false
    }
  }

  function selectProcess(process: Process) {
    options.selectedExternalKey.value = process.key
  }

  function clearSelection() {
    options.selectedExternalKey.value = null
  }

  return {
    snapshot, error, loading, processes, groups, distributions,
    sourceFilter, kindFilter, distributionFilter, filteredGroups,
    activeProcess, activePath, activeNode, classificationLabel,
    refresh, selectProcess, clearSelection,
  }
}

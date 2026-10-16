import { computed, ref, watch, type Ref } from 'vue'
import type { Info } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/session/models'
import {
  assignSession, closeLeaf, emptyLayout, focusPane, leaves, sanitizeLayout, splitLeaf,
  type SplitDirection,
} from '../workspaceLayout'

const storageKey = 'diggerterm:workspace:v1'

export function useWorkspace(
  sessions: Ref<Info[]>,
  selectedExternalKey: Ref<string | null>,
  reportError: (error: unknown) => void,
) {
  const layout = ref(emptyLayout())
  const paneLeaves = computed(() => leaves(layout.value))
  const selectedId = computed(() => selectedExternalKey.value
    ? null
    : paneLeaves.value.find(pane => pane.paneId === layout.value.focusedPaneId)?.sessionId ?? null)
  let ready = false

  watch(layout, value => {
    if (!ready) return
    try { localStorage.setItem(storageKey, JSON.stringify(value)) }
    catch (error) { reportError(error) }
  }, { deep: true })

  function restoreWorkspace(existing: Info[]) {
    let saved: unknown
    try { saved = JSON.parse(localStorage.getItem(storageKey) || 'null') }
    catch (error) { reportError(error) }
    // Numeric IDs from older daemons cannot safely identify persisted sessions.
    const valid = new Set(existing.filter(session => session.status === 'running' && /^[A-Z2-7]{26,}-[0-9]+$/.test(session.id)).map(session => session.id))
    layout.value = sanitizeLayout(saved, valid)
    if (!saved && existing.some(session => session.status === 'running')) {
      layout.value = assignSession(layout.value, layout.value.focusedPaneId, existing.find(session => session.status === 'running')!.id)
    }
    ready = true
  }

  function splitPane(direction: SplitDirection) {
    if (selectedExternalKey.value) return
    layout.value = splitLeaf(layout.value, layout.value.focusedPaneId, direction)
  }

  function closePane(paneId: string) { layout.value = closeLeaf(layout.value, paneId) }

  function focusNextPane() {
    const index = paneLeaves.value.findIndex(pane => pane.paneId === layout.value.focusedPaneId)
    layout.value = focusPane(layout.value, paneLeaves.value[(index + 1) % paneLeaves.value.length]!.paneId)
  }

  function focusWorkspacePane(paneId: string) {
    layout.value = focusPane(layout.value, paneId)
  }

  function changePaneSession(paneId: string, event: Event) {
    const select = event.target as HTMLSelectElement
    const id = select.value || null
    if (id === null || sessions.value.some(session => session.id === id)) {
      layout.value = assignSession(layout.value, paneId, id)
    }
    select.value = paneLeaves.value.find(pane => pane.paneId === paneId)?.sessionId ?? ''
  }

  function sanitizeForSessions(items: Info[]) {
    layout.value = sanitizeLayout(layout.value, new Set(items.map(session => session.id)))
  }

  function selectManagedSession(id: string) {
    selectedExternalKey.value = null
    layout.value = assignSession(layout.value, layout.value.focusedPaneId, id)
  }

  return {
    layout, paneLeaves, selectedId,
    restoreWorkspace, isReady: () => ready, splitPane, closePane,
    focusNextPane, focusWorkspacePane, changePaneSession, sanitizeForSessions, selectManagedSession,
  }
}

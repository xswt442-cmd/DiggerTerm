export type SplitDirection = 'horizontal' | 'vertical'

export type PaneNode =
  | { type: 'leaf'; paneId: string; sessionId: string | null }
  | { type: 'split'; id: string; direction: SplitDirection; ratio: number; children: [PaneNode, PaneNode] }

export type WorkspaceLayout = { version: 1; root: PaneNode; focusedPaneId: string }

const MAX_PANES = 16
const MAX_DEPTH = 12
const MIN_RATIO = 0.15
const MAX_RATIO = 0.85

function newId(prefix: string, used = new Set<string>()): string {
  let index = 1
  while (used.has(`${prefix}-${index}`)) index++
  return `${prefix}-${index}`
}

export function emptyLayout(): WorkspaceLayout {
  const paneId = newId('pane')
  return { version: 1, root: { type: 'leaf', paneId, sessionId: null }, focusedPaneId: paneId }
}

export function leaves(layout: WorkspaceLayout): Extract<PaneNode, { type: 'leaf' }>[] {
  const result: Extract<PaneNode, { type: 'leaf' }>[] = []
  const pending: PaneNode[] = [layout.root]
  while (pending.length > 0) {
    const node = pending.pop()!
    if (node.type === 'leaf') result.push(node)
    else pending.push(node.children[1], node.children[0])
  }
  return result
}

export function splitLeaf(layout: WorkspaceLayout, paneId: string, direction: SplitDirection): WorkspaceLayout {
  if (direction !== 'horizontal' && direction !== 'vertical') return layout
  if (leaves(layout).length >= MAX_PANES) return layout
  const paneDepth = leafDepth(layout.root, paneId)
  if (paneDepth < 0 || paneDepth >= MAX_DEPTH) return layout
  const used = new Set<string>()
  const pending = [layout.root]
  while (pending.length) {
    const node = pending.pop()!
    used.add(node.type === 'leaf' ? node.paneId : node.id)
    if (node.type === 'split') pending.push(...node.children)
  }
  const paneIdNew = newId('pane', used)
  used.add(paneIdNew)
  const splitId = newId('split', used)
  const root = mapNode(layout.root, node => node.type === 'leaf' && node.paneId === paneId
    ? {
        type: 'split', id: splitId, direction, ratio: 0.5,
        children: [node, { type: 'leaf', paneId: paneIdNew, sessionId: null }],
      }
    : node)
  return { ...layout, root, focusedPaneId: paneIdNew }
}

export function closeLeaf(layout: WorkspaceLayout, paneId: string): WorkspaceLayout {
  if (!leaves(layout).some(leaf => leaf.paneId === paneId) || leaves(layout).length <= 1) return layout
  const root = removeLeaf(layout.root, paneId) ?? layout.root
  const remaining = collectLeaves(root)
  const focusedPaneId = layout.focusedPaneId === paneId ? remaining[0].paneId : layout.focusedPaneId
  return { ...layout, root, focusedPaneId }
}

export function focusPane(layout: WorkspaceLayout, paneId: string): WorkspaceLayout {
  return leaves(layout).some(leaf => leaf.paneId === paneId) && layout.focusedPaneId !== paneId
    ? { ...layout, focusedPaneId: paneId }
    : layout
}

export function assignSession(layout: WorkspaceLayout, paneId: string, sessionId: string | null): WorkspaceLayout {
  const target = leaves(layout).find(leaf => leaf.paneId === paneId)
  if (!target) return layout
  if (sessionId !== null) {
    const assigned = leaves(layout).find(leaf => leaf.sessionId === sessionId)
    if (assigned) return focusPane(layout, assigned.paneId)
  }
  const root = mapNode(layout.root, node => node.type === 'leaf' && node.paneId === paneId
    ? { ...node, sessionId }
    : node)
  return { ...layout, root, focusedPaneId: paneId }
}

export function resizeSplit(layout: WorkspaceLayout, splitId: string, ratio: number): WorkspaceLayout {
  if (!Number.isFinite(ratio)) return layout
  const clamped = Math.max(MIN_RATIO, Math.min(MAX_RATIO, ratio))
  let changed = false
  const root = mapNode(layout.root, node => {
    if (node.type !== 'split' || node.id !== splitId || node.ratio === clamped) return node
    changed = true
    return { ...node, ratio: clamped }
  })
  return changed ? { ...layout, root } : layout
}

export function sanitizeLayout(value: unknown, validSessionIds: Set<string>): WorkspaceLayout {
  if (!isRecord(value) || value.version !== 1) return emptyLayout()
  const usedNodeIds = new Set<string>()
  const usedSessions = new Set<string>()
  let paneCount = 0
  const sanitizeNode = (input: unknown, depth: number, seen: WeakSet<object>): PaneNode | null => {
    if (!isRecord(input) || seen.has(input)) return null
    seen.add(input)
    if (input.type === 'leaf') {
      if (paneCount >= MAX_PANES) return null
      paneCount += 1
      const paneId = uniqueId(input.paneId, 'pane', usedNodeIds)
      let sessionId: string | null = null
      if (typeof input.sessionId === 'string' && validSessionIds.has(input.sessionId) && !usedSessions.has(input.sessionId)) {
        sessionId = input.sessionId
        usedSessions.add(sessionId)
      }
      return { type: 'leaf', paneId, sessionId }
    }
    if (input.type !== 'split' || (input.direction !== 'horizontal' && input.direction !== 'vertical') ||
      !Array.isArray(input.children) || input.children.length !== 2) return null

    const [firstInput, secondInput] = input.children
    if (depth >= MAX_DEPTH) return null
    const first = sanitizeNode(firstInput, depth + 1, seen)
    const second = sanitizeNode(secondInput, depth + 1, seen)
    if (!first) return second
    if (!second) return first
    const id = uniqueId(input.id, 'split', usedNodeIds)
    const ratio = typeof input.ratio === 'number' && Number.isFinite(input.ratio)
      ? Math.max(MIN_RATIO, Math.min(MAX_RATIO, input.ratio))
      : 0.5
    return { type: 'split', id, direction: input.direction, ratio, children: [first, second] }
  }

  const root = sanitizeNode(value.root, 0, new WeakSet())
  if (!root) return emptyLayout()
  const paneLeaves = collectLeaves(root)
  const requestedFocus = typeof value.focusedPaneId === 'string' ? value.focusedPaneId : ''
  const focusedPaneId = paneLeaves.some(leaf => leaf.paneId === requestedFocus) ? requestedFocus : paneLeaves[0].paneId
  return { version: 1, root, focusedPaneId }
}

function isRecord(value: unknown): value is Record<string, any> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function uniqueId(value: unknown, prefix: string, used: Set<string>): string {
  if (typeof value === 'string' && value.length > 0 && value.length <= 128 && !/[\u0000-\u001f]/.test(value) && !used.has(value)) {
    used.add(value)
    return value
  }
  const id = newId(prefix, used)
  used.add(id)
  return id
}

function collectLeaves(root: PaneNode): Extract<PaneNode, { type: 'leaf' }>[] {
  const result: Extract<PaneNode, { type: 'leaf' }>[] = []
  const pending: PaneNode[] = [root]
  while (pending.length > 0) {
    const node = pending.pop()!
    if (node.type === 'leaf') result.push(node)
    else pending.push(node.children[1], node.children[0])
  }
  return result
}

function leafDepth(root: PaneNode, paneId: string): number {
  const pending: { node: PaneNode; depth: number }[] = [{ node: root, depth: 0 }]
  while (pending.length > 0) {
    const current = pending.pop()!
    if (current.node.type === 'leaf' && current.node.paneId === paneId) return current.depth
    if (current.node.type === 'split') {
      pending.push({ node: current.node.children[1], depth: current.depth + 1 })
      pending.push({ node: current.node.children[0], depth: current.depth + 1 })
    }
  }
  return -1
}

function mapNode(node: PaneNode, transform: (node: PaneNode) => PaneNode): PaneNode {
  const transformed = transform(node)
  if (transformed !== node || transformed.type === 'leaf') return transformed
  const first = mapNode(transformed.children[0], transform)
  const second = mapNode(transformed.children[1], transform)
  if (first === transformed.children[0] && second === transformed.children[1]) return transformed
  return { ...transformed, children: [first, second] }
}

function removeLeaf(node: PaneNode, paneId: string): PaneNode | null {
  if (node.type === 'leaf') return node.paneId === paneId ? null : node
  const first = removeLeaf(node.children[0], paneId)
  const second = removeLeaf(node.children[1], paneId)
  if (!first) return second
  if (!second) return first
  if (first === node.children[0] && second === node.children[1]) return node
  return { ...node, children: [first, second] }
}

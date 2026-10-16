export type ObservedProcess = {
  key: string
  pid: number
  parentPid: number
  name: string
  kind: string
  source: string
  distribution: string
  startedAt: string
  classificationBasis?: 'name' | 'ancestor' | string
}

export type ProcessNode = ObservedProcess & {
  children: ProcessNode[]
  parentMissing: boolean
}

export type ProcessGroup = {
  source: string
  distribution: string
  roots: ProcessNode[]
}

const maxTreeDepth = 32
const namespace = (p: ObservedProcess) => `${p.source}\u0000${p.source === 'wsl' ? p.distribution : ''}`
const compare = (a: ObservedProcess, b: ObservedProcess) =>
  a.kind === 'descendant' && b.kind !== 'descendant' ? 1 :
    b.kind === 'descendant' && a.kind !== 'descendant' ? -1 :
      a.name.localeCompare(b.name) || a.pid - b.pid || a.key.localeCompare(b.key)

// Build only links justified by an in-snapshot parent in the same PID namespace.
export function buildProcessTree(processes: ObservedProcess[]): ProcessGroup[] {
  const nodes = processes.map(process => ({ ...process, children: [], parentMissing: false }))
  const indexed = new Map<string, ProcessNode | null>()
  for (const node of nodes) {
    const id = `${namespace(node)}\u0000${node.pid}`
    indexed.set(id, indexed.has(id) ? null : node)
  }
  const parentOf = new Map<ProcessNode, ProcessNode>()
  for (const node of nodes) {
    const parent = node.parentPid > 0 ? indexed.get(`${namespace(node)}\u0000${node.parentPid}`) : undefined
    const parentStart = parent ? Date.parse(parent.startedAt) : NaN
    const childStart = Date.parse(node.startedAt)
    if (parent && parent !== node && Number.isFinite(parentStart) && Number.isFinite(childStart) && parentStart <= childStart)
      parentOf.set(node, parent)
    else node.parentMissing = node.parentPid > 0
  }
  // Remove one deterministic edge from any malformed cycle.
  for (const node of [...nodes].sort((a, b) => a.key.localeCompare(b.key))) {
    const path = new Set<ProcessNode>()
    let cursor: ProcessNode | undefined = node
    let previous: ProcessNode | undefined
    while (cursor && parentOf.has(cursor)) {
      if (path.size >= maxTreeDepth) {
        if (previous) {
          parentOf.delete(previous)
          previous.parentMissing = true
        }
        break
      }
      if (path.has(cursor)) {
        if (previous) {
          parentOf.delete(previous)
          previous.parentMissing = true
        }
        break
      }
      path.add(cursor)
      previous = cursor
      cursor = parentOf.get(cursor)
    }
  }
  const roots: ProcessNode[] = []

  for (const node of nodes) {
    const parent = parentOf.get(node)
    if (parent) parent.children.push(node)
    else roots.push(node)
  }

  const sortTree = (list: ProcessNode[]) => {
    list.sort(compare)
    for (const node of list) sortTree(node.children)
  }
  sortTree(roots)

  const grouped = new Map<string, ProcessGroup>()
  for (const root of roots) {
    const id = namespace(root)
    let group = grouped.get(id)
    if (!group) {
      group = { source: root.source, distribution: root.source === 'wsl' ? root.distribution : '', roots: [] }
      grouped.set(id, group)
    }
    group.roots.push(root)
  }
  return [...grouped.values()].sort((a, b) => a.source.localeCompare(b.source) || a.distribution.localeCompare(b.distribution))
}

// Keep matching nodes and their actual ancestors visible during search.
export function filterProcessTree(groups: ProcessGroup[], matches: (process: ObservedProcess) => boolean): ProcessGroup[] {
  const prune = (node: ProcessNode): ProcessNode | undefined => {
    const children = node.children.map(prune).filter((child): child is ProcessNode => child !== undefined)
    if (!matches(node) && children.length === 0) return undefined
    return { ...node, children }
  }
  return groups.map(group => ({ ...group, roots: group.roots.map(prune).filter((root): root is ProcessNode => root !== undefined) }))
    .filter(group => group.roots.length > 0)
}

export function classificationLabel(process: ObservedProcess): 'identified' | 'inferred' | 'unknown' {
  if (process.classificationBasis === 'name') return 'identified'
  if (process.classificationBasis === 'ancestor' || process.kind === 'descendant') return 'inferred'
  return process.kind === 'agent' || process.kind === 'terminal' ? 'identified' : 'unknown'
}

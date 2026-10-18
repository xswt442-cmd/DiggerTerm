import type { Process, SourceStatus } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/observe/models'

export type WatchChangeKind = 'new' | 'not-seen' | 'changed'

export interface WatchChange {
  id: number
  kind: WatchChangeKind
  key: string
  process: Process
  sourceKey: string
  at: number
}

export interface WatchSource extends SourceStatus {
  stale: boolean
}

interface TrustedSource {
  namespace: string
  policy: string
  processes: Map<string, Process>
  status: SourceStatus
  lastTrustedAt: number
}

export interface WatchSnapshot {
  processes: Process[]
  sources: SourceStatus[] | null
  scannedAt: string
  truncated: boolean
}

const maxChanges = 500
const maxWatched = 100
const maxTrustedSources = 32
const changeLifetimeMs = 30 * 60 * 1000
const trustedSourceLifetimeMs = 30 * 60 * 1000

function sourceKey(source: string, distribution: string): string {
  return source === 'wsl' ? `wsl:${distribution}` : source
}

export class DigWatchState {
  private trusted = new Map<string, TrustedSource>()
  private currentStatuses: WatchSource[] = []
  private watched = new Map<string, Process>()
  private uncertainFollowed = new Map<string, string>()
  private changes: WatchChange[] = []
  private baselinePending = new Set<string>()
  private nextID = 0

  constructor(private readonly changed: () => void = () => {}) {}

  update(snapshot: WatchSnapshot, now = Date.now()) {
    this.expire(now, false)
    const statuses = snapshot.sources
    if (!statuses?.length) {
      this.currentStatuses = []
      this.changed()
      return
    }

    const byStatus = new Map<string, SourceStatus>()
    let wslInventory: SourceStatus | undefined
    for (const status of statuses) {
      if (status.source === 'wsl' && !status.distribution) {
        wslInventory = status
        continue
      }
      byStatus.set(sourceKey(status.source, status.distribution), status)
    }
    const processesBySource = new Map<string, Map<string, Process>>()
    for (const process of snapshot.processes ?? []) {
      const key = sourceKey(process.source, process.distribution)
      let processes = processesBySource.get(key)
      if (!processes) { processes = new Map(); processesBySource.set(key, processes) }
      processes.set(process.key, process)
    }

    const views = new Map<string, WatchSource>()
    if (wslInventory) views.set('wsl:', { ...wslInventory, stale: wslInventory.state !== 'success' || !wslInventory.complete })
    const wslQuotaReached = statuses.some(status => status.source === 'wsl' && status.distribution === '*' && status.state === 'skipped')
    for (const [key, status] of byStatus) {
      const old = this.trusted.get(key)
      const complete = status.state === 'success' && status.complete && !status.truncated && !!status.namespace && !!status.policy
      if (!complete) {
        views.set(key, { ...status, stale: true })
        continue
      }
      const current = processesBySource.get(key) ?? new Map<string, Process>()
      if (old && this.baselinePending.has(key)) {
        this.baselinePending.delete(key)
        this.clearFollowUncertainty(key)
      } else if (old && old.namespace === status.namespace && old.policy === status.policy) {
        for (const [processKey, process] of current) {
          const before = old.processes.get(processKey)
          if (!before) this.record('new', key, process, now)
          else if (before.name !== process.name || before.kind !== process.kind ||
              before.parentPid !== process.parentPid || before.classificationBasis !== process.classificationBasis) {
            this.record('changed', key, process, now)
          }
        }
        for (const [processKey, process] of old.processes) {
          if (!current.has(processKey)) this.record('not-seen', key, process, now)
        }
        this.clearFollowUncertainty(key)
      } else if (old) {
        for (const processKey of old.processes.keys()) {
          if (this.watched.has(processKey) && !current.has(processKey)) this.uncertainFollowed.set(processKey, key)
        }
      }
      for (const [processKey, process] of current) if (this.watched.has(processKey)) this.watched.set(processKey, process)
      this.trusted.set(key, { namespace: status.namespace, policy: status.policy, processes: current, status, lastTrustedAt: now })
      views.set(key, { ...status, stale: false })
    }

    for (const [key, old] of this.trusted) {
      if (views.has(key)) continue
      if (old.status.source === 'wsl' && wslInventory?.state === 'success' && wslInventory.complete && !wslQuotaReached &&
          !statuses.some(status => status.source === 'wsl' && status.distribution === old.status.distribution && status.state === 'skipped')) {
        views.set(key, {
          ...old.status, state: 'not-running', complete: false, truncated: false,
          reason: wslInventory.reason || 'not-in-running-set', scannedAt: wslInventory.scannedAt, stale: true,
        })
      } else {
        const reason = wslQuotaReached && old.status.source === 'wsl' ? 'distro-limit' : 'source-status-missing'
        views.set(key, { ...old.status, state: 'unknown', complete: false, truncated: false, reason, scannedAt: snapshot.scannedAt, stale: true })
      }
    }
    for (const [key, status] of byStatus) {
      if (status.state === 'skipped' && !views.has(key)) views.set(key, { ...status, stale: true })
    }
    this.trimTrustedSources(views)
    this.currentStatuses = [...views.values()]
    this.changed()
  }

  trustedProcesses(): Process[] {
    return [...this.trusted.values()].flatMap(source => [...source.processes.values()])
  }

  sources(): WatchSource[] { return this.currentStatuses }

  changesList(): WatchChange[] { return this.changes }

  pruneExpired(now = Date.now()) { this.expire(now) }

  watchedProcesses(): Process[] { return [...this.watched.values()] }

  isWatched(key: string): boolean { return this.watched.has(key) }

  followedState(key: string): 'present' | 'not-seen' | 'unknown' {
    if (this.uncertainFollowed.has(key)) return 'unknown'
    const watched = this.watched.get(key)
    if (!watched) return 'unknown'
    const source = this.currentStatuses.find(item => item.source === watched.source && item.distribution === watched.distribution)
    if (!source || source.stale || !source.complete || source.state !== 'success') return 'unknown'
    if (this.findProcess(key)) return 'present'
    return 'not-seen'
  }

  toggleWatch(key: string): boolean {
    if (this.watched.has(key)) {
      this.watched.delete(key)
      this.uncertainFollowed.delete(key)
      this.changed()
      return false
    }
    if (this.watched.size >= maxWatched) return false
    const process = this.findProcess(key)
    if (!process) return false
    this.watched.set(key, process)
    this.uncertainFollowed.delete(key)
    this.changed()
    return true
  }

  clearChanges() {
    if (!this.changes.length) return
    this.changes = []
    this.changed()
  }

  pause(reason = 'scanning-disabled') {
    for (const key of this.trusted.keys()) this.baselinePending.add(key)
    this.currentStatuses = this.currentStatuses.map(source => ({
      ...source, state: 'unknown', complete: false, reason, stale: true,
    }))
    this.changed()
  }

  reset() {
    this.trusted.clear()
    this.currentStatuses = []
    this.baselinePending.clear()
    this.watched.clear()
    this.uncertainFollowed.clear()
    this.changes = []
    this.nextID = 0
    this.changed()
  }

  failScan(reason: string, scannedAt = new Date().toISOString()) {
    this.currentStatuses = this.currentStatuses.map(source => ({
      ...source, state: 'failed', complete: false, truncated: false, reason, scannedAt, stale: true,
    }))
    this.changed()
  }

  private findProcess(key: string): Process | undefined {
    for (const source of this.trusted.values()) {
      const process = source.processes.get(key)
      if (process) return process
    }
  }

  private clearFollowUncertainty(source: string) {
    for (const [key, uncertainSource] of this.uncertainFollowed) {
      if (uncertainSource === source) this.uncertainFollowed.delete(key)
    }
  }

  private trimTrustedSources(views: Map<string, WatchSource>) {
    while (this.trusted.size > maxTrustedSources) {
      let oldestKey = ''
      let oldestAt = Infinity
      for (const [key, source] of this.trusted) {
        if (source.lastTrustedAt < oldestAt) {
          oldestKey = key
          oldestAt = source.lastTrustedAt
        }
      }
      if (!oldestKey) return
      this.trusted.delete(oldestKey)
      this.baselinePending.delete(oldestKey)
      views.delete(oldestKey)
    }
  }

  private record(kind: WatchChangeKind, source: string, process: Process, at: number) {
    this.changes.unshift({ id: ++this.nextID, kind, key: process.key, process, sourceKey: source, at })
    if (this.changes.length > maxChanges) this.changes.length = maxChanges
  }

  private expire(now: number, notify = true) {
    let didChange = false
    const expiredSources = new Set<string>()
    for (const [key, source] of this.trusted) {
      if (now - source.lastTrustedAt < trustedSourceLifetimeMs) continue
      this.trusted.delete(key)
      this.baselinePending.delete(key)
      expiredSources.add(key)
    }
    if (expiredSources.size) {
      this.currentStatuses = this.currentStatuses.filter(source =>
        (source.source === 'wsl' && !source.distribution) ||
        !expiredSources.has(sourceKey(source.source, source.distribution)))
      didChange = true
    }
    const retained = this.changes.filter(change => change.at + changeLifetimeMs > now)
    if (retained.length !== this.changes.length) {
      this.changes = retained
      didChange = true
    }
    if (didChange && notify) this.changed()
  }
}

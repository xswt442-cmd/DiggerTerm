export interface ActivitySession {
  id: string
  status: string
  outputOffset?: number
}

export interface ActivityOutput {
  id: string
  offset: number
}

interface PendingOutput {
  offset: number
  unfocused: boolean
  expiresAt: number
}

const maxPendingSessions = 256
const pendingLifetimeMs = 10_000

export class SessionActivityTracker {
  private offsets = new Map<string, number>()
  private unread = new Set<string>()
  private pending = new Map<string, PendingOutput>()

  constructor(private readonly changed: () => void = () => {}) {}

  baseline(sessions: ActivitySession[]) {
    this.offsets.clear()
    this.unread.clear()
    this.pending.clear()
    for (const session of sessions) this.offsets.set(session.id, session.outputOffset ?? 0)
    this.changed()
  }

  output(event: ActivityOutput, known: boolean, creating: boolean, terminalFocused: boolean, enabled = true, now = Date.now()) {
    if (!Number.isSafeInteger(event.offset) || event.offset < 0) return
    this.expirePending(now)
    const previous = this.offsets.get(event.id)
    if (previous === undefined) {
      if (known) this.offsets.set(event.id, 0)
      else if (creating) {
        const old = this.pending.get(event.id)
        if (old) {
          old.offset = Math.max(old.offset, event.offset)
          old.unfocused ||= !terminalFocused
          old.expiresAt = now + pendingLifetimeMs
        } else {
          if (this.pending.size >= maxPendingSessions) this.pending.delete(this.pending.keys().next().value!)
          this.pending.set(event.id, { offset: event.offset, unfocused: !terminalFocused, expiresAt: now + pendingLifetimeMs })
        }
      }
      return
    }
    if (event.offset <= previous) return
    this.offsets.set(event.id, event.offset)
    if (terminalFocused || !enabled) this.clear(event.id)
    else this.mark(event.id)
  }

  created(session: ActivitySession, terminalFocused: boolean, enabled = true, now = Date.now()) {
    this.expirePending(now)
    const baseline = session.outputOffset ?? 0
    const pending = this.pending.get(session.id)
    this.pending.delete(session.id)
    this.offsets.set(session.id, Math.max(baseline, pending?.offset ?? 0))
    if (enabled && session.status === 'running' && pending && pending.offset > baseline && pending.unfocused && !terminalFocused) this.mark(session.id)
    else this.clear(session.id)
  }

  focus(sessionID: string | null) {
    if (sessionID) this.clear(sessionID)
  }

  exited(sessionID: string) {
    this.clear(sessionID)
  }

  remove(sessionID: string) {
    this.offsets.delete(sessionID)
    this.pending.delete(sessionID)
    this.clear(sessionID)
  }

  disconnect() {
    this.offsets.clear()
    this.pending.clear()
    this.unread.clear()
    this.changed()
  }

  disable() {
    this.pending.clear()
    if (!this.unread.size) return
    this.unread.clear()
    this.changed()
  }

  has(sessionID: string): boolean {
    return this.unread.has(sessionID)
  }

  private expirePending(now: number) {
    for (const [id, item] of this.pending) if (item.expiresAt <= now) this.pending.delete(id)
  }

  private mark(id: string) {
    if (this.unread.has(id)) return
    this.unread.add(id)
    this.changed()
  }

  private clear(id: string) {
    if (!this.unread.delete(id)) return
    this.changed()
  }
}

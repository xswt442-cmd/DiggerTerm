type SessionState = { id: string; status: string; revision?: number }

// List replies can race exit/rename events. Preserve the newest daemon state.
export function mergeSessions<T extends SessionState>(current: T[], incoming: T[], replace = false): T[] {
  const previous = new Map(current.map(session => [session.id, session]))
  const result = replace ? new Map<string, T>() : new Map(previous)
  for (const session of incoming) {
    const old = previous.get(session.id)
    const stale = old && ((old.revision ?? 0) > (session.revision ?? 0) ||
      ((old.revision ?? 0) === (session.revision ?? 0) && old.status === 'exited' && session.status === 'running'))
    result.set(session.id, stale ? old : session)
  }
  return [...result.values()]
}

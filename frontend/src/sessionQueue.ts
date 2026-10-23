// Operations for one PTY must reach the daemon in order. Terminal input is
// committed as soon as xterm emits it, so unmounting a pane cannot drop input
// that was already accepted before its queued detach.
const pending = new Map<string, Promise<void>>()

export function queueSessionOperation(id: string, execute: () => Promise<void>): Promise<void> {
  const previous = pending.get(id) ?? Promise.resolve()
  const operation = previous.catch(() => {}).then(execute)
  pending.set(id, operation)
  const cleanup = () => { if (pending.get(id) === operation) pending.delete(id) }
  void operation.then(cleanup, cleanup)
  return operation
}

export function queueAcceptedTerminalInput(
  sessionId: string,
  input: string,
  send: (id: string, data: string) => Promise<void>,
): Promise<void> {
  return queueSessionOperation(sessionId, () => send(sessionId, input))
}

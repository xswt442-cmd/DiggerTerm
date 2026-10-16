import { Attach, Detach } from '../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/sessionservice'

// A pane may remount before its asynchronous attach finishes. Keep its detach
// ahead of a replacement pane's attach on the same GUI connection.
const pending = new Map<string, Promise<void>>()
export function updateAttachment(id: string, attach: boolean): Promise<void> {
  return queueSessionOperation(id, () => attach ? Attach(id) : Detach(id))
}

export function queueSessionOperation(id: string, execute: () => Promise<void>): Promise<void> {
  const previous = pending.get(id) ?? Promise.resolve()
  const operation = previous.catch(() => {}).then(execute)
  pending.set(id, operation)
  const cleanup = () => { if (pending.get(id) === operation) pending.delete(id) }
  void operation.then(cleanup, cleanup)
  return operation
}

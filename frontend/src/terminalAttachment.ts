import { Attach, Detach } from '../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/sessionservice'

import { queueSessionOperation } from './sessionQueue'

export { queueSessionOperation } from './sessionQueue'

// A pane may remount before its asynchronous attach finishes. Keep its detach
// behind accepted input and ahead of a replacement pane's attach.
export function updateAttachment(id: string, attach: boolean): Promise<void> {
  return queueSessionOperation(id, () => attach ? Attach(id) : Detach(id))
}

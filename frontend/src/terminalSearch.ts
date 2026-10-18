import type { ITerminalOptions } from '@xterm/xterm'

// SearchAddon uses xterm's proposed buffer and decoration APIs to search,
// highlight matches, and report result counts. Keep that opt-in local to the
// terminal instances created for this application.
export function withTerminalSearchSupport(options: ITerminalOptions): ITerminalOptions {
  return { ...options, allowProposedApi: true }
}

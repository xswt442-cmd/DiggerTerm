import { Call } from '@wailsio/runtime'

export interface SSHDirectoryListing {
  path: string
  directories: string[]
  truncated: boolean
}

// Wails supports method-name bindings without editing generated code.
// Requires a matching desktop GUI/daemon version.
export async function browseSSHDirectories(sessionId: string, directory: string): Promise<SSHDirectoryListing> {
  return await Call.ByName('github.com/xswt442-cmd/DiggerTerm/internal/desktop.SessionService.BrowseSSHDirectories', sessionId, directory) as SSHDirectoryListing
}

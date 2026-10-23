// Quick CD inserts a quoted command, never a newline or a shell side effect.
// A foreground app may still receive typed text, so the user explicitly
// presses Enter at a shell prompt to execute it.
export type CdShell = 'posix' | 'powershell' | 'cmd'

export function cdShellForProfile(profile: string): CdShell {
  if (profile === 'powershell') return 'powershell'
  if (profile === 'cmd') return 'cmd'
  return 'posix' // Bash, WSL, local Unix shell, and default SSH shells
}

export function formatCdCommand(rawPath: string, shell: CdShell): string {
  if (typeof rawPath !== 'string' || rawPath.length === 0 || rawPath.length > 2048) {
    throw new Error('Directory must contain 1–2048 characters')
  }
  // Reject control codes, line separators and invisible bidi formatting.
  if (/[\x00-\x1f\x7f-\x9f\u2028\u2029\u202a-\u202e\u2066-\u2069]/u.test(rawPath)) {
    throw new Error('Directory cannot contain control characters or line breaks')
  }
  if (shell === 'powershell') {
    return `Set-Location -LiteralPath '${rawPath.replaceAll("'", "''")}'`
  }
  if (shell === 'cmd') {
    // CMD expands %VAR% and delayed !VAR! even within quotes; never
    // interpolate such paths into a command line without CMD-aware escaping.
    if (/[%"!^]/.test(rawPath)) {
      throw new Error('CMD paths cannot contain %, !, ^ or double quotes')
    }
    return `cd /d "${rawPath}"`
  }
  if (shell === 'posix') {
    if (rawPath === '~') return 'cd'
    if (rawPath.startsWith('~/')) {
      return `cd "$HOME"/'${rawPath.slice(2).replaceAll("'", "'\\''")}'`
    }
    const safePath = rawPath.startsWith('-') ? './' + rawPath : rawPath
    return `cd '${safePath.replaceAll("'", "'\\''")}'`
  }
  throw new Error('Unsupported shell syntax')
}

export function supportsLocalDefaultDirectory(profile: string): boolean {
  return profile === 'powershell' || profile === 'cmd' || profile === 'bash' || profile === 'shell'
}

export function supportsNativeFolderPicker(profile: string): boolean {
  return supportsLocalDefaultDirectory(profile) || profile === 'wsl'
}

export function localDefaultDirectoryAllowed(profile: string, path: string, windows: boolean): boolean {
  if (!supportsLocalDefaultDirectory(profile) || path.length > 512) return false
  return windows ? /^(?:[A-Za-z]:[\\/]|\\\\[^\\/]+[\\/][^\\/]+)/.test(path) : path.startsWith('/')
}
// The native dialog only browses local folders. WSL and Git Bash need their
// default Windows-drive mount convention, not a literal Windows path.
export function directoryFromNativePicker(profile: string, path: string, windows: boolean): string {
  if (!supportsNativeFolderPicker(profile)) {
    throw new Error('The native folder picker cannot browse remote SSH directories')
  }
  if (!path || path.length > 2048 || /[\x00-\x1f\x7f-\x9f\u2028\u2029\u202a-\u202e\u2066-\u2069]/u.test(path)) {
    throw new Error('Selected directory contains invalid characters')
  }
  if (!windows) {
    if (!path.startsWith('/')) throw new Error('Select an absolute directory')
    return path
  }
  const drive = /^([A-Za-z]):[\\/](.*)$/.exec(path)
  if (profile === 'wsl' || profile === 'bash') {
    if (!drive) throw new Error('Select a drive folder; UNC and WSL network paths cannot be translated')
    const letter = drive[1].toLowerCase()
    const suffix = drive[2].replaceAll('\\', '/')
    return (profile === 'wsl' ? '/mnt/' : '/') + letter + '/' + suffix
  }
  if (profile === 'cmd' && !drive) {
    throw new Error('CMD cannot change to a network UNC folder with cd /d')
  }
  if (drive || (profile === 'powershell' && /^\\\\[^\\]+\\[^\\]+(?:\\.*)?$/.test(path))) return path
  throw new Error('Select an absolute Windows folder')
}

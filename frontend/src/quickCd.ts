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

export function localDefaultDirectoryAllowed(profile: string, path: string, windows: boolean): boolean {
  if (!['powershell', 'cmd', 'bash', 'shell'].includes(profile) || path.length > 512) return false
  return windows ? /^(?:[A-Za-z]:[\\/]|\\\\[^\\/]+[\\/][^\\/]+)/.test(path) : path.startsWith('/')
}

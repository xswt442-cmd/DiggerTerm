/**
 * Keyboard policy only for a focused, daemon-owned xterm instance.
 *
 * On Windows/Linux Ctrl+C copies a selection, or sends ETX (SIGINT) when
 * nothing is selected. Ctrl+V pastes. Ctrl+Shift+C/V and Insert variants
 * are explicit clipboard shortcuts. In smart mode, alternate-screen apps
 * such as nano keep Ctrl+C and Ctrl+V as native terminal input. Every
 * other Ctrl+letter goes to the PTY, never to browser accelerators.
 *
 * macOS keeps Cmd+C/V for the clipboard and Ctrl+C for terminal SIGINT.
 * AltGr (Ctrl+Alt) must never be intercepted as a terminal shortcut.
 */
import type { KeyboardMode } from './uiPreferences'

export type TerminalKeyAction =
  | { kind: 'pass' }
  | { kind: 'copy' }
  | { kind: 'paste' }
  | { kind: 'input'; data: string }

export type TerminalKeyModifiers = Pick<
  KeyboardEvent,
  'key' | 'ctrlKey' | 'metaKey' | 'altKey' | 'shiftKey' | 'isComposing'
>

export function decideTerminalShortcut(
  event: TerminalKeyModifiers,
  hasSelection: boolean,
  isMac: boolean,
  mode: KeyboardMode = 'smart',
  alternateScreen = false,
): TerminalKeyAction {
  const native = mode === 'native' || (mode === 'smart' && alternateScreen)
  const pass: TerminalKeyAction = { kind: 'pass' }
  if (event.isComposing || event.altKey) return pass

  const key = event.key.toLowerCase()
  const control = event.ctrlKey && !event.metaKey
  const command = isMac && event.metaKey && !event.ctrlKey

  if (command) {
    if (key === 'c') return { kind: 'copy' }
    if (key === 'v') return { kind: 'paste' }
    return pass
  }
  if (event.metaKey) return pass

  if (control && key === 'insert' && !event.shiftKey) return { kind: 'copy' }
  if (key === 'insert' && event.shiftKey && !control) return { kind: 'paste' }

  if (!control) return pass

  if (event.shiftKey && key === 'c' && !isMac) return { kind: 'copy' }
  if (event.shiftKey && key === 'v' && !isMac) return { kind: 'paste' }
  if (!event.shiftKey && key === 'c') {
    if (!isMac && hasSelection && !native) return { kind: 'copy' }
    return { kind: 'input', data: '\x03' }
  }
  if (!event.shiftKey && key === 'v' && !isMac && !native) return { kind: 'paste' }

  // Ctrl+A, Ctrl+L, Ctrl+R, Ctrl+W, Ctrl+Z, Ctrl+D, ... are Shell
  // control characters. Sending them here also prevents Ctrl+R from
  // reloading the WebView or Ctrl+W from closing it.
  if (/^[a-z]$/.test(key)) {
    return { kind: 'input', data: String.fromCharCode(key.charCodeAt(0) - 96) }
  }

  // Ctrl+arrows, Ctrl+Home/End, F keys, etc. retain xterm's normal mapping.
  return pass
}

export type ActionId =
  | 'copySelection'
  | 'pasteClipboard'
  | 'toggleCopyMode'
  | 'splitRight'
  | 'splitDown'
  | 'focusNextPane'
  | 'closePane'
  | 'openSettings'

export type KeybindingScope = 'terminal' | 'workspace'

export interface ActionDefinition {
  id: ActionId
  labelZh: string
  labelEn: string
  scope: KeybindingScope
  defaultKeys: { other: string[]; mac: string[] }
}

export interface Binding {
  action: ActionId
  scope: KeybindingScope
  keys: string[]
}

const actionList: ActionDefinition[] = [
  { id: 'copySelection', labelZh: '复制选区', labelEn: 'Copy selection', scope: 'terminal', defaultKeys: { other: ['Ctrl+Shift+C', 'Ctrl+Insert'], mac: ['Meta+C'] } },
  { id: 'pasteClipboard', labelZh: '粘贴剪贴板', labelEn: 'Paste clipboard', scope: 'terminal', defaultKeys: { other: ['Ctrl+Shift+V', 'Shift+Insert'], mac: ['Meta+V'] } },
  { id: 'toggleCopyMode', labelZh: '切换复制模式', labelEn: 'Toggle copy mode', scope: 'terminal', defaultKeys: { other: ['Ctrl+Shift+M'], mac: ['Ctrl+Shift+M'] } },
  { id: 'splitRight', labelZh: '向右分屏', labelEn: 'Split right', scope: 'workspace', defaultKeys: { other: ['Ctrl+Shift+ArrowRight'], mac: ['Ctrl+Shift+ArrowRight'] } },
  { id: 'splitDown', labelZh: '向下分屏', labelEn: 'Split down', scope: 'workspace', defaultKeys: { other: ['Ctrl+Shift+ArrowDown'], mac: ['Ctrl+Shift+ArrowDown'] } },
  { id: 'focusNextPane', labelZh: '聚焦下一个 Pane', labelEn: 'Focus next pane', scope: 'workspace', defaultKeys: { other: ['Ctrl+Shift+F6'], mac: ['Ctrl+Shift+F6'] } },
  { id: 'closePane', labelZh: '关闭 Pane', labelEn: 'Close pane', scope: 'workspace', defaultKeys: { other: ['Ctrl+Shift+F4'], mac: ['Ctrl+Shift+F4'] } },
  { id: 'openSettings', labelZh: '打开设置', labelEn: 'Open settings', scope: 'workspace', defaultKeys: { other: ['Ctrl+Shift+Comma'], mac: ['Ctrl+Shift+Comma'] } },
]

export const actionDefinitions: ReadonlyArray<ActionDefinition> = actionList

const actionById = new Map(actionList.map((definition) => [definition.id, definition]))
const allowedKeys = new Set([
  ...'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'Insert', 'ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown',
  'Comma', 'Period', 'Slash', 'Backslash', 'BracketLeft', 'BracketRight', 'Semicolon', 'Quote',
  'Backquote', 'Minus', 'Equal', 'Escape', 'Tab', 'Enter', 'Home', 'End', 'PageUp', 'PageDown',
  'Delete', 'Backspace', ...Array.from({ length: 24 }, (_, index) => `F${index + 1}`),
])
const modifierOrder = ['Ctrl', 'Meta', 'Alt', 'Shift'] as const
type Modifier = typeof modifierOrder[number]

function normalizeKey(value: string): string {
  const aliases: Record<string, string> = {
    ',': 'Comma', '.': 'Period', '/': 'Slash', '\\': 'Backslash', '[': 'BracketLeft', ']': 'BracketRight',
    ';': 'Semicolon', "'": 'Quote', '`': 'Backquote', '-': 'Minus', '=': 'Equal',
    Left: 'ArrowLeft', Right: 'ArrowRight', Up: 'ArrowUp', Down: 'ArrowDown',
  }
  if (aliases[value]) return aliases[value]
  if (/^[a-z]$/i.test(value)) return value.toUpperCase()
  if (/^f\d{1,2}$/i.test(value)) return value.toUpperCase()
  const named = allowedKeys.has(value) ? value : [...allowedKeys].find((key) => key.toLowerCase() === value.toLowerCase())
  if (!named) throw new Error(`unsupported key ${JSON.stringify(value)}`)
  return named
}

const modifierAliases: Record<string, Modifier> = {
  ctrl: 'Ctrl', control: 'Ctrl', meta: 'Meta', cmd: 'Meta', command: 'Meta', alt: 'Alt', option: 'Alt', shift: 'Shift',
}

export function normalizeChord(value: string): string {
  if (typeof value !== 'string' || !value.trim()) throw new Error('shortcut must be a non-empty string')
  const parts = value.split('+').map((part) => part.trim())
  if (parts.some((part) => !part)) throw new Error(`invalid shortcut ${JSON.stringify(value)}`)
  const modifiers = new Set<Modifier>()
  let key: string | undefined
  for (const part of parts) {
    const modifier = modifierAliases[part.toLowerCase()]
    if (modifier) {
      if (modifiers.has(modifier)) throw new Error(`duplicate modifier in ${JSON.stringify(value)}`)
      modifiers.add(modifier)
    } else {
      if (key) throw new Error(`shortcut must contain one key: ${JSON.stringify(value)}`)
      key = normalizeKey(part)
    }
  }
  if (!key) throw new Error(`shortcut is missing a key: ${JSON.stringify(value)}`)
  if (!modifiers.size) throw new Error(`shortcut needs a modifier: ${JSON.stringify(value)}`)
  if (modifiers.has('Ctrl') && modifiers.has('Alt')) throw new Error('Ctrl+Alt shortcuts are ambiguous with AltGr')
  const chord = [...modifierOrder.filter((modifier) => modifiers.has(modifier)), key].join('+')
  validateChord(chord)
  return chord
}

function validateChord(chord: string): void {
  const [key, ...modifiers] = chord.split('+').reverse()
  const set = new Set(modifiers)
  const ctrlOnly = set.size === 1 && set.has('Ctrl')
  if (ctrlOnly && /^[A-Z]$/.test(key!)) {
    if (key === 'C' || key === 'V') throw new Error(`Ctrl+${key} is reserved for terminal copy/control behavior`)
    throw new Error(`Ctrl+${key} is reserved for terminal or browser behavior`)
  }
  if (set.has('Alt') && key === 'F4') throw new Error('Alt+F4 is reserved by the operating system')
  if (set.has('Meta') && key === 'Q') throw new Error('Meta+Q is reserved by the operating system')
  if (set.has('Ctrl') && set.has('Alt') && key === 'Delete') throw new Error('Ctrl+Alt+Delete is reserved by the operating system')
  if (set.has('Ctrl') && !set.has('Shift') && !set.has('Meta') && ['R', 'W', 'L'].includes(key!)) {
    throw new Error(`Ctrl+${key} is reserved by the browser or terminal`)
  }
}

export function defaultBindings(isMac: boolean): Binding[] {
  return actionList.map(({ id, scope, defaultKeys: keys }) => ({
    action: id, scope, keys: [...(isMac ? keys.mac : keys.other)],
  }))
}

export function parseBindings(text: string, isMac: boolean): Binding[] {
  let document: unknown
  try { document = JSON.parse(text) } catch (error) {
    throw new Error(`invalid keybindings JSON: ${error instanceof Error ? error.message : String(error)}`)
  }
  if (!document || typeof document !== 'object' || Array.isArray(document)) throw new Error('keybindings must be a JSON object')
  const root = document as Record<string, unknown>
  if (root.version !== 1) throw new Error('version must be 1')
  if (Object.keys(root).some((key) => key !== 'version' && key !== 'bindings')) throw new Error('keybindings contains an unknown field')
  if (!Array.isArray(root.bindings)) throw new Error('bindings must be an array')

  const parsed: Binding[] = []
  const seenActions = new Set<ActionId>()
  const seenChords = new Set<string>()
  for (const item of root.bindings) {
    if (!item || typeof item !== 'object' || Array.isArray(item)) throw new Error('each binding must be an object')
    const value = item as Record<string, unknown>
    if (Object.keys(value).some((key) => !['action', 'scope', 'keys'].includes(key))) throw new Error('binding contains an unknown field')
    if (typeof value.action !== 'string' || !actionById.has(value.action as ActionId)) throw new Error(`unknown action ${String(value.action)}`)
    const action = value.action as ActionId
    if (seenActions.has(action)) throw new Error(`duplicate action ${action}`)
    seenActions.add(action)
    const definition = actionById.get(action)!
    if (value.scope !== definition.scope) throw new Error(`${action} must use scope ${definition.scope}`)
    if (!Array.isArray(value.keys) || value.keys.some((key) => typeof key !== 'string')) throw new Error(`${action}.keys must be an array of shortcuts`)
    if (value.keys.length > 16) throw new Error(`${action} may have at most 16 shortcuts`)
    const keys = (value.keys as string[]).map(normalizeChord)
    for (const key of keys) {
      if (seenChords.has(key)) throw new Error(`shortcut ${key} is bound more than once`)
      seenChords.add(key)
    }
    parsed.push({ action, scope: definition.scope, keys })
  }

  const effective = new Map(defaultBindings(isMac).map((binding) => [binding.action, binding]))
  for (const binding of parsed) effective.set(binding.action, binding)
  const effectiveChords = new Set<string>()
  for (const binding of effective.values()) for (const key of binding.keys) {
    if (effectiveChords.has(key)) throw new Error(`effective shortcut ${key} conflicts across actions`)
    effectiveChords.add(key)
  }
  return parsed
}

export function eventChord(event: Pick<KeyboardEvent, 'key' | 'ctrlKey' | 'metaKey' | 'altKey' | 'shiftKey' | 'isComposing' | 'getModifierState'> & { code?: string }): string | null {
  if (event.isComposing || event.getModifierState?.('AltGraph') || (event.ctrlKey && event.altKey)) return null
  const modifiers: Modifier[] = []
  if (event.ctrlKey) modifiers.push('Ctrl')
  if (event.metaKey) modifiers.push('Meta')
  if (event.altKey) modifiers.push('Alt')
  if (event.shiftKey) modifiers.push('Shift')
  if (!modifiers.length || event.key === 'Dead' || event.key === 'Process') return null
  let key: string
  const punctuation = ['Comma', 'Period', 'Slash', 'Backslash', 'BracketLeft', 'BracketRight', 'Semicolon', 'Quote', 'Backquote', 'Minus', 'Equal']
  try { key = normalizeKey(event.code && punctuation.includes(event.code) ? event.code : event.key) } catch { return null }
  return [...modifierOrder.filter((modifier) => modifiers.includes(modifier)), key].join('+')
}

export function matchBinding(event: Parameters<typeof eventChord>[0], bindings: Binding[], scope: KeybindingScope): ActionId | null {
  const chord = eventChord(event)
  if (!chord) return null
  for (const binding of bindings) if (binding.scope === scope && binding.keys.includes(chord)) return binding.action
  return null
}

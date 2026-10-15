import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/keybindings.ts', import.meta.url), 'utf8')
const compiled = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const keybindings = await import('data:text/javascript;base64,' + Buffer.from(compiled).toString('base64'))

test('publishes the whitelist and safe platform defaults', () => {
  assert.deepEqual(keybindings.actionDefinitions.map(({ id }) => id), [
    'copySelection', 'pasteClipboard', 'toggleCopyMode', 'splitRight', 'splitDown',
    'focusNextPane', 'closePane', 'openSettings',
  ])
  const windows = keybindings.defaultBindings(false)
  const mac = keybindings.defaultBindings(true)
  assert.deepEqual(windows.find((binding) => binding.action === 'copySelection').keys, ['Ctrl+Shift+C', 'Ctrl+Insert'])
  assert.deepEqual(mac.find((binding) => binding.action === 'copySelection').keys, ['Meta+C'])
  assert.deepEqual(windows.find((binding) => binding.action === 'pasteClipboard').keys, ['Ctrl+Shift+V', 'Shift+Insert'])
})

test('normalizes aliases and modifier order', () => {
  assert.equal(keybindings.normalizeChord('shift+control+arrowright'), 'Ctrl+Shift+ArrowRight')
  assert.equal(keybindings.normalizeChord('cmd+v'), 'Meta+V')
  assert.equal(keybindings.normalizeChord('Ctrl+,'), 'Ctrl+Comma')
})

test('parses partial overrides while preserving clear bindings', () => {
  const parsed = keybindings.parseBindings(JSON.stringify({ version: 1, bindings: [
    { action: 'splitRight', scope: 'workspace', keys: ['Ctrl+Shift+F8'] },
    { action: 'toggleCopyMode', scope: 'terminal', keys: [] },
  ] }), false)
  assert.deepEqual(parsed, [
    { action: 'splitRight', scope: 'workspace', keys: ['Ctrl+Shift+F8'] },
    { action: 'toggleCopyMode', scope: 'terminal', keys: [] },
  ])
})

test('rejects invalid, reserved, duplicate and cross-scope conflicting shortcuts', () => {
  const invalid = [
    [{ action: 'run', scope: 'workspace', keys: ['Ctrl+Shift+R'] }],
    [{ action: 'copySelection', scope: 'terminal', keys: ['Ctrl+C'] }],
    [{ action: 'splitRight', scope: 'workspace', keys: ['Ctrl+Alt+Right'] }],
    [{ action: 'splitRight', scope: 'workspace', keys: ['Alt+F4'] }],
    [{ action: 'splitRight', scope: 'terminal', keys: ['Ctrl+Shift+F8'] }],
    [
      { action: 'splitRight', scope: 'workspace', keys: ['Ctrl+Shift+F8'] },
      { action: 'splitDown', scope: 'workspace', keys: ['Ctrl+Shift+F8'] },
    ],
    [{ action: 'splitRight', scope: 'workspace', keys: ['Ctrl+Shift+C'] }],
  ]
  for (const bindings of invalid) assert.throws(() => keybindings.parseBindings(JSON.stringify({ version: 1, bindings }), false))
  assert.throws(() => keybindings.parseBindings('{', false), /invalid keybindings JSON/)
  assert.throws(() => keybindings.parseBindings(JSON.stringify({ version: 2, bindings: [] }), false), /version must be 1/)
})

function keyEvent(key, overrides = {}) {
  return {
    key, ctrlKey: false, metaKey: false, altKey: false, shiftKey: false, isComposing: false,
    getModifierState: () => false,
    ...overrides,
  }
}

test('matches only the requested scope and ignores IME and AltGr events', () => {
  const bindings = keybindings.defaultBindings(false)
  assert.equal(keybindings.matchBinding(keyEvent('C', { ctrlKey: true, shiftKey: true }), bindings, 'terminal'), 'copySelection')
  assert.equal(keybindings.matchBinding(keyEvent('C', { ctrlKey: true, shiftKey: true }), bindings, 'workspace'), null)
  assert.equal(keybindings.matchBinding(keyEvent('ArrowRight', { ctrlKey: true, shiftKey: true }), bindings, 'workspace'), 'splitRight')
  assert.equal(keybindings.matchBinding(keyEvent('C', { ctrlKey: true, shiftKey: true, isComposing: true }), bindings, 'terminal'), null)
  assert.equal(keybindings.matchBinding(keyEvent('C', { ctrlKey: true, altKey: true, shiftKey: true }), bindings, 'terminal'), null)
  assert.equal(keybindings.matchBinding(keyEvent('a', { altKey: true, getModifierState: (name) => name === 'AltGraph' }), bindings, 'terminal'), null)
  assert.equal(keybindings.matchBinding(keyEvent('<', { ctrlKey: true, shiftKey: true, code: 'Comma' }), bindings, 'workspace'), 'openSettings')
})

test('allows simultaneous shortcut swaps with final conflict validation', () => {
  assert.equal(keybindings.parseBindings(JSON.stringify({ version: 1, bindings: [
    { action: 'splitRight', scope: 'workspace', keys: ['Ctrl+Shift+ArrowDown'] },
    { action: 'splitDown', scope: 'workspace', keys: ['Ctrl+Shift+ArrowRight'] },
  ] }), false).length, 2)
})

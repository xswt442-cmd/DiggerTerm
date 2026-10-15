import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/terminalShortcuts.ts', import.meta.url), 'utf8')
const compiled = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const { decideTerminalShortcut } = await import('data:text/javascript;base64,' +
  Buffer.from(compiled).toString('base64'))

function event(key, overrides = {}) {
  return {
    key, ctrlKey: false, metaKey: false, altKey: false,
    shiftKey: false, isComposing: false, ...overrides,
  }
}
const ctrl = (key, overrides = {}) => event(key, { ctrlKey: true, ...overrides })
const expectAction = (key, modifiers, selection, mac, expected) =>
  assert.deepEqual(decideTerminalShortcut(event(key, modifiers), selection, mac), expected)

test('Ctrl+C copies an active selection on Windows/Linux', () => {
  expectAction('c', { ctrlKey: true }, true, false, { kind: 'copy' })
  expectAction('C', { ctrlKey: true }, true, false, { kind: 'copy' })
})

test('Ctrl+C without selection sends ETX / SIGINT to the shell', () => {
  expectAction('c', { ctrlKey: true }, false, false, { kind: 'input', data: '\x03' })
  expectAction('c', { ctrlKey: true }, false, true, { kind: 'input', data: '\x03' })
  // On macOS, Cmd+C copies; plain Ctrl+C interrupts even with selection.
  expectAction('c', { ctrlKey: true }, true, true, { kind: 'input', data: '\x03' })
})

test('Ctrl+V and Ctrl+Shift+V paste on Windows/Linux', () => {
  expectAction('v', { ctrlKey: true }, false, false, { kind: 'paste' })
  expectAction('V', { ctrlKey: true, shiftKey: true }, true, false, { kind: 'paste' })
  expectAction('v', { ctrlKey: true, shiftKey: true }, false, false, { kind: 'paste' })
})

test('Ctrl+Shift+C, Ctrl+Insert and Shift+Insert follow terminal clipboard conventions', () => {
  expectAction('C', { ctrlKey: true, shiftKey: true }, true, false, { kind: 'copy' })
  expectAction('C', { ctrlKey: true, shiftKey: true }, false, false, { kind: 'copy' })
  expectAction('Insert', { ctrlKey: true }, true, false, { kind: 'copy' })
  expectAction('Insert', { shiftKey: true }, false, false, { kind: 'paste' })
})

test('Mac Cmd+C / Cmd+V copy/paste, plain Ctrl+V remains a shell control key', () => {
  expectAction('c', { metaKey: true }, true, true, { kind: 'copy' })
  expectAction('v', { metaKey: true }, false, true, { kind: 'paste' })
  expectAction('V', { metaKey: true, shiftKey: true }, false, true, { kind: 'paste' })
  expectAction('v', { ctrlKey: true }, false, true, { kind: 'input', data: '\x16' })
})

test('Common Ctrl keys are forwarded as control bytes, not browser accelerators', () => {
  const keys = { a: '\x01', d: '\x04', e: '\x05', f: '\x06', k: '\x0b',
    l: '\x0c', p: '\x10', q: '\x11', r: '\x12', s: '\x13', t: '\x14',
    u: '\x15', w: '\x17', x: '\x18', y: '\x19', z: '\x1a' }
  for (const [key, data] of Object.entries(keys)) {
    expectAction(key, { ctrlKey: true }, false, false, { kind: 'input', data })
  }
})

test('Ordinary text, Ctrl+arrows, function keys, AltGr and IME never get stolen', () => {
  assert.deepEqual(decideTerminalShortcut(event('a'), false, false), { kind: 'pass' })
  assert.deepEqual(decideTerminalShortcut(ctrl('ArrowLeft'), false, false), { kind: 'pass' })
  assert.deepEqual(decideTerminalShortcut(event('F3'), false, false), { kind: 'pass' })
  assert.deepEqual(decideTerminalShortcut(ctrl('r', { altKey: true }), false, false), { kind: 'pass' })
  assert.deepEqual(decideTerminalShortcut(ctrl('v', { altKey: true }), false, false), { kind: 'pass' })
  assert.deepEqual(decideTerminalShortcut(ctrl('c', { isComposing: true }), false, false), { kind: 'pass' })
  assert.deepEqual(decideTerminalShortcut(event('x', { metaKey: true }), false, false), { kind: 'pass' })
})

test('Smart mode keeps nano and vim native Ctrl shortcuts in alternate screen', () => {
  const smart = (key, extra = {}, selection = false) =>
    decideTerminalShortcut(ctrl(key, extra), selection, false, 'smart', true)
  for (const [key, code] of Object.entries({v:22, o:15, x:24, w:23, k:11, u:21, g:7, r:18})) {
    assert.deepEqual(smart(key, {}, true), {kind:'input',data:String.fromCharCode(code)})
  }
  assert.deepEqual(smart('c', {}, true), {kind:'copy'})
  assert.deepEqual(smart('c', {}, false), {kind:'input',data:'\x03'})
  assert.deepEqual(smart('v', {shiftKey:true}), {kind:'paste'})
  assert.deepEqual(smart('c', {shiftKey:true}), {kind:'copy'})
})

test('Native/desktop shortcut settings override alternate-screen detection', () => {
  assert.deepEqual(decideTerminalShortcut(ctrl('v'), false, false, 'native', false),
    {kind:'input',data:String.fromCharCode(22)})
  assert.deepEqual(decideTerminalShortcut(ctrl('v'), false, false, 'desktop', true),
    {kind:'paste'})
  assert.deepEqual(decideTerminalShortcut(ctrl('c'), true, false, 'native', false),
    {kind:'input',data:String.fromCharCode(3)})
  assert.deepEqual(decideTerminalShortcut(ctrl('c'), true, false, 'desktop', true),
    {kind:'copy'})
})

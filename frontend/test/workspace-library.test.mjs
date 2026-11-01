import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import ts from 'typescript'
const source = readFileSync(new URL('../src/workspaceLibrary.ts', import.meta.url), 'utf8')
const js = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 } }).outputText
const { emptyLibrary, parseLibrary, directoryScope, rememberDirectory, toggleFavorite, validPresetName, validPresetSSHTarget, loadLibrary, saveLibrary } = await import('data:text/javascript;base64,' + Buffer.from(js).toString('base64'))

test('recent and favorite directories stay scoped and deduplicated', () => {
  assert.equal(directoryScope('powershell', 'a'), directoryScope('cmd', 'b'))
  assert.notEqual(directoryScope('wsl', 'a'), directoryScope('powershell', 'a'))
  const data = emptyLibrary()
  const a = { scope: directoryScope('ssh', 'one'), path: '/home/me' }
  const b = { scope: directoryScope('ssh', 'two'), path: '/home/me' }
  rememberDirectory(data, a); rememberDirectory(data, b); rememberDirectory(data, a)
  assert.deepEqual(data.recent, [a, b])
  toggleFavorite(data, a); toggleFavorite(data, b); toggleFavorite(data, a)
  assert.deepEqual(data.favorites, [b])
})

test('workspace information round-trips and invalid storage is harmless', () => {
  let data = ''
  globalThis.localStorage = { getItem: () => data, setItem: (_key, value) => { data = value } }
  const source = emptyLibrary()
  source.marks.session = { note: 'Build server', color: 'amber' }
  source.presets.push({ id: 'preset', name: 'Workspace', profile: 'shell', directory: '/work' })
  rememberDirectory(source, { scope: 'local:shell', path: '/work' })
  saveLibrary(source)
  const restored = loadLibrary()
  assert.deepEqual(restored.marks.session, source.marks.session)
  assert.deepEqual(restored.presets, source.presets)
  assert.deepEqual(restored.recent, source.recent)
  data = '{broken'
  assert.deepEqual(loadLibrary().presets, [])
  delete globalThis.localStorage
})

test('preset names match session name constraints', () => {
  assert.equal(validPresetName('Build 工作区'), true)
  assert.equal(validPresetName(' '), false)
  assert.equal(validPresetName('bad\u2066name'), false)
  assert.equal(validPresetName('x'.repeat(65)), false)
})

test('SSH presets accept only host aliases and optional valid ports', () => {
  assert.equal(validPresetSSHTarget('user@my-host.example'), true)
  for (const target of ['-oProxyCommand', 'user:password@host', 'host;cmd', 'user@-host', 'ssh://host']) assert.equal(validPresetSSHTarget(target), false)
  const data = parseLibrary({ presets: [
    { id: 'safe', name: 'Server', profile: 'ssh', directory: '', target: 'user@server', port: 22 },
    { id: 'unsafe', name: 'Bad', profile: 'ssh', directory: '', target: 'user:password@host', port: 22 },
    { id: 'badport', name: 'Bad', profile: 'ssh', directory: '', target: 'host', port: 65536 },
  ] })
  assert.equal(data.presets.length, 1)
  assert.equal(data.presets[0].target, 'user@server')
})

test('stored state rejects invalid entries and bounds collections', () => {
  const many = Array.from({ length: 130 }, (_, i) => ({ scope: 'local:shell', path: '/p' + i }))
  const data = parseLibrary({ recent: [...many, { scope: 'ssh:x', path: 'a\nb' }], presets: [{ id: 'a', name: 'A', profile: 'shell', directory: '/tmp' }, { id: 'b', name: '', profile: 'shell' }], marks: { x: { note: 'work', color: 'blue' }, y: { note: 'bad', color: 'url(evil)' } } })
  assert.equal(data.recent.length, 100)
  assert.equal(data.presets.length, 1)
  assert.deepEqual(data.marks.x, { note: 'work', color: 'blue' })
  assert.equal(data.marks.y, undefined)
})

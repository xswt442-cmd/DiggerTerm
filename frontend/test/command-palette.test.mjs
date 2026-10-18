import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/commandPalette.ts', import.meta.url), 'utf8')
const compiled = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const { buildPaletteCommands, filterPaletteCommands } = await import('data:text/javascript;base64,' + Buffer.from(compiled).toString('base64'))

const commands = [
  { id: 'split-right', label: 'Split right', category: 'Workspace', keywords: ['pane'], enabled: true },
  { id: 'search', label: 'Search terminal', category: 'Terminal', keywords: ['output', 'find'], enabled: true },
  { id: 'settings', label: 'Session settings', category: 'Settings', keywords: ['shell', 'directory'], enabled: false },
]

test('ranks direct matches before fuzzy matches and keeps stable order for ties', () => {
  assert.deepEqual(filterPaletteCommands(commands, 'search').map(item => item.id), ['search'])
  const tied = [
    { id: 'first', label: 'Open settings', category: 'Settings', keywords: [], enabled: true },
    { id: 'second', label: 'Open settings', category: 'Settings', keywords: [], enabled: true },
  ]
  assert.deepEqual(filterPaletteCommands(tied, 'open').map(item => item.id), ['first', 'second'])
})

test('keeps disabled contextual commands visible and preserves source order for empty queries', () => {
  assert.deepEqual(filterPaletteCommands(commands, '').map(item => item.id), ['split-right', 'search', 'settings'])
  assert.equal(filterPaletteCommands(commands, 'session')[0].enabled, false)
})

test('builds contextual commands without exposing destructive controls', () => {
  const actions = []
  const palette = buildPaletteCommands({
    locale: 'en',
    sessions: [{ id: 's1', name: 'Build', profile: 'powershell', status: 'running' }],
    profiles: [{ id: 'ssh', name: 'SSH', description: 'Remote' }],
    themes: [{ id: 'builtin:nightfall', nameEn: 'Nightfall', nameZh: '夜航' }],
    daemonConnected: false, creating: false, inManagedWorkspace: false, paneCount: 2,
    canUseTerminalTools: false, observeEnabled: true, observationBusy: false,
    actions: {
      openSettings: id => actions.push(['settings', id]), setTheme: id => actions.push(['theme', id]),
      selectSession: id => actions.push(['session', id]), createSession: id => actions.push(['create', id]),
      split: direction => actions.push(['split', direction]), nextPane: () => actions.push(['next']),
      closePane: () => actions.push(['close-pane']), search: () => actions.push(['search']),
      quickSelect: () => actions.push(['quick-select']), refreshDig: () => actions.push(['dig']),
    },
  })
  assert.equal(palette.find(item => item.id === 'terminal:search').enabled, false)
  assert.equal(palette.find(item => item.id === 'workspace:split-right').enabled, false)
  assert.equal(palette.find(item => item.id === 'dig:refresh').enabled, true)
  assert.equal(palette.some(item => /terminate|stop-all/i.test(item.id)), false)
  palette.find(item => item.id === 'settings:quickselect').run()
  assert.deepEqual(actions, [['settings', 'quickselect']])
})

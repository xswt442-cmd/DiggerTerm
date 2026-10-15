import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const compile = path => ts.transpileModule(readFileSync(new URL(path, import.meta.url), 'utf8'), {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const url = source => 'data:text/javascript;base64,' + Buffer.from(source).toString('base64')
const preferences = url(compile('../src/uiPreferences.ts'))
const { themeRegistry, themeVariables } = await import(url(compile('../src/userThemes.ts').replace("'./uiPreferences'", JSON.stringify(preferences))))

test('dynamic themes preserve JSON names, stable IDs and base deep merge', () => {
  const registry = themeRegistry([
    { id: 'user:forest', name: '松林工作台', base: 'pine', colors: { sidebar: '#123456' }, terminal: { brightRed: '#abcdef' }, appearance: { radius: 8 }, css: '' },
    { id: 'user:ocean', name: '松林工作台', base: 'bluehour', colors: {}, terminal: {}, appearance: {}, css: '' },
    { id: 'user:minimal', name: 'minimal', base: 'nightfall', css: '.layout { opacity: .9 }' },
  ])
  assert.equal(registry.length, 7)
  assert.equal(new Set(registry.map(t => t.id)).size, 7)
  const forest = registry.find(t => t.id === 'user:forest')
  assert.equal(forest.palette.zh, '松林工作台')
  assert.equal(forest.palette.css['--sidebar'], '#123456')
  assert.equal(forest.palette.terminal.background, '#191d1b')
  assert.equal(forest.palette.terminal.brightRed, '#abcdef')
  assert.equal(forest.palette.css['--dt-radius'], '8px')
  assert.equal(registry.find(t => t.id === 'builtin:pine').palette.css['--sidebar'], '#232725')
})

test('public variables cover UI and terminal colors with CSS aliases', () => {
  const variables = themeVariables(themeRegistry([])[0].palette)
  assert.equal(variables['--sidebar'], 'var(--dt-sidebar)')
  assert.equal(variables['--dt-sidebar'], '#1b1f26')
  assert.equal(variables['--dt-terminal-bg'], '#101318')
  assert.equal(variables['--dt-terminal-red'], '#d58d8b')
})

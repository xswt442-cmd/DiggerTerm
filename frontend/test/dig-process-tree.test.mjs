import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/dig/processTree.ts', import.meta.url), 'utf8')
const javascript = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const dig = await import('data:text/javascript;base64,' + Buffer.from(javascript).toString('base64'))

const proc = (key, pid, parentPid, source = 'windows', distribution = '', startedAt = `2026-01-01T00:00:0${pid % 10}Z`) => ({
  key, pid, parentPid, source, distribution, startedAt, name: key, kind: 'descendant',
})

test('tree links only same-namespace parents with valid chronology', () => {
  const groups = dig.buildProcessTree([
    proc('w-root', 1, 0), proc('w-child', 2, 1),
    proc('l-root', 1, 0, 'wsl', 'Ubuntu'), proc('l-child', 2, 1, 'wsl', 'Ubuntu'),
    proc('d-root', 1, 0, 'wsl', 'Debian'), proc('missing', 3, 90),
    proc('stale-child', 4, 5, 'windows', '', '2026-01-01T00:00:01Z'),
    proc('late-parent', 5, 0, 'windows', '', '2026-01-01T00:00:02Z'),
  ])
  assert.equal(groups.length, 3)
  assert.equal(groups.find(g => g.source === 'windows').roots.length, 4)
  assert.equal(groups.find(g => g.source === 'wsl' && g.distribution === 'Ubuntu').roots[0].children[0].key, 'l-child')
  assert.equal(groups.find(g => g.source === 'windows').roots.find(n => n.key === 'missing').parentMissing, true)
  assert.equal(groups.find(g => g.source === 'windows').roots.find(n => n.key === 'stale-child').parentMissing, true)
})

test('search keeps matched nodes and their actual ancestors', () => {
  const [group] = dig.buildProcessTree([proc('shell', 10, 0), proc('agent', 11, 10), proc('child', 12, 11)])
  const filtered = dig.filterProcessTree([group], process => process.key === 'child')
  assert.deepEqual(filtered[0].roots.map(root => root.key), ['shell'])
  assert.equal(filtered[0].roots[0].children[0].children[0].key, 'child')
})

test('malformed cycles are cut and classification basis distinguishes inference', () => {
  const groups = dig.buildProcessTree([
    proc('a', 20, 21, 'windows', '', '2026-01-01T00:00:00Z'),
    proc('b', 21, 20, 'windows', '', '2026-01-01T00:00:00Z'),
  ])
  assert.equal(groups.reduce((n, g) => n + g.roots.length, 0), 1)
  assert.equal(dig.classificationLabel({ ...proc('named', 30, 0), kind: 'agent', classificationBasis: 'name' }), 'identified')
  assert.equal(dig.classificationLabel({ ...proc('child', 31, 30), classificationBasis: 'ancestor' }), 'inferred')
  assert.equal(dig.classificationLabel({ ...proc('other', 32, 0), kind: 'other' }), 'unknown')
})

test('excessive ancestry is cut at a bounded depth', () => {
  const rows = Array.from({ length: 40 }, (_, index) => ({
    ...proc(`p${index}`, 100 + index, index ? 99 + index : 0, 'windows', '', new Date(2026, 0, 1, 0, 0, index).toISOString()),
  }))
  const [group] = dig.buildProcessTree(rows)
  const maxDepth = (nodes, depth = 1) => nodes.length ? Math.max(...nodes.map(node => Math.max(depth, maxDepth(node.children, depth + 1)))) : depth - 1
  assert.equal(Math.max(...group.roots.map(root => maxDepth([root]))), 32)
  assert.ok(group.roots.some(root => root.parentMissing))
})

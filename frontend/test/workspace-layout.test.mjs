import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/workspaceLayout.ts', import.meta.url), 'utf8')
const javascript = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const layout = await import('data:text/javascript;base64,' + Buffer.from(javascript).toString('base64'))

test('empty layout creates one independent empty pane', () => {
  const first = layout.emptyLayout()
  const second = layout.emptyLayout()
  assert.equal(first.version, 1)
  assert.equal(first.root.type, 'leaf')
  assert.equal(first.root.sessionId, null)
  assert.equal(first.focusedPaneId, first.root.paneId)
  assert.deepEqual(first, second)
  assert.notEqual(first.root, second.root)
})

test('split, focus, session assignment, and close preserve one interactive pane per session', () => {
  let state = layout.emptyLayout()
  const firstPane = state.focusedPaneId
  state = layout.assignSession(state, firstPane, 'session-a')
  assert.deepEqual(layout.splitLeaf(state, firstPane, 'horizontal'), layout.splitLeaf(state, firstPane, 'horizontal'))
  state = layout.splitLeaf(state, firstPane, 'horizontal')
  assert.equal(layout.leaves(state).length, 2)
  const secondPane = state.focusedPaneId
  assert.notEqual(secondPane, firstPane)
  assert.equal(layout.leaves(state).find(pane => pane.paneId === firstPane).sessionId, 'session-a')
  assert.equal(layout.leaves(state).find(pane => pane.paneId === secondPane).sessionId, null)

  state = layout.assignSession(state, secondPane, 'session-a')
  assert.equal(state.focusedPaneId, firstPane)
  assert.equal(layout.leaves(state).filter(pane => pane.sessionId === 'session-a').length, 1)
  state = layout.assignSession(state, secondPane, 'session-b')
  assert.equal(layout.leaves(state).find(pane => pane.paneId === secondPane).sessionId, 'session-b')

  state = layout.focusPane(state, firstPane)
  state = layout.closeLeaf(state, firstPane)
  assert.equal(layout.leaves(state).length, 1)
  assert.equal(state.root.type, 'leaf')
  assert.equal(state.focusedPaneId, secondPane)
  const unchanged = layout.closeLeaf(state, secondPane)
  assert.equal(unchanged, state)
})

test('nested splits resize independently, clamp ratios, and reject unknown IDs', () => {
  let state = layout.emptyLayout()
  const first = state.focusedPaneId
  state = layout.splitLeaf(state, first, 'vertical')
  const topSplit = state.root.id
  const second = state.focusedPaneId
  state = layout.splitLeaf(state, second, 'horizontal')
  const nestedSplit = state.root.children[1].id
  state = layout.resizeSplit(state, topSplit, 2)
  state = layout.resizeSplit(state, nestedSplit, -1)
  assert.equal(state.root.ratio, 0.85)
  assert.equal(state.root.children[1].ratio, 0.15)
  assert.equal(layout.leaves(state).length, 3)
  assert.equal(layout.focusPane(state, 'missing'), state)
  assert.equal(layout.splitLeaf(state, 'missing', 'horizontal'), state)
  assert.equal(layout.resizeSplit(state, topSplit, Number.NaN), state)
})

test('sanitize filters dead sessions, duplicate sessions and IDs, and bad focus', () => {
  const sharedLeaf = { type: 'leaf', paneId: 'duplicate-pane', sessionId: 'live' }
  const input = {
    version: 1,
    focusedPaneId: 'missing',
    root: {
      type: 'split', id: 'same-id', direction: 'horizontal', ratio: 3,
      children: [
        sharedLeaf,
        {
          type: 'split', id: 'same-id', direction: 'vertical', ratio: Number.POSITIVE_INFINITY,
          children: [
            { type: 'leaf', paneId: 'duplicate-pane', sessionId: 'live' },
            { type: 'leaf', paneId: 'dead-pane', sessionId: 'exited' },
          ],
        },
      ],
    },
  }
  const result = layout.sanitizeLayout(input, new Set(['live']))
  const panes = layout.leaves(result)
  assert.equal(panes.length, 3)
  assert.equal(new Set(panes.map(pane => pane.paneId)).size, 3)
  assert.equal(panes.filter(pane => pane.sessionId === 'live').length, 1)
  assert.equal(panes.find(pane => pane.paneId === 'dead-pane').sessionId, null)
  assert.equal(result.focusedPaneId, panes[0].paneId)
  assert.equal(result.root.type, 'split')
  assert.equal(result.root.ratio, 0.85)
  assert.notEqual(result.root.id, result.root.children[1].id)
})

test('sanitize bounds pane count and depth and tolerates cyclic input', () => {
  const manyLeaves = Array.from({ length: 20 }, (_, index) => ({ type: 'leaf', paneId: `p${index}`, sessionId: null }))
  let root = manyLeaves[0]
  for (const leaf of manyLeaves.slice(1)) {
    root = { type: 'split', id: `s${leaf.paneId}`, direction: 'horizontal', ratio: 0.5, children: [root, leaf] }
  }
  let bounded = layout.sanitizeLayout({ version: 1, root, focusedPaneId: 'p19' }, new Set())
  assert.ok(layout.leaves(bounded).length <= 16)

  let deep = { type: 'leaf', paneId: 'deep', sessionId: null }
  for (let index = 0; index < 40; index++) {
    deep = { type: 'split', id: `deep-s${index}`, direction: 'vertical', ratio: 0.5, children: [deep, { type: 'leaf', paneId: `side-${index}`, sessionId: null }] }
  }
  bounded = layout.sanitizeLayout({ version: 1, root: deep, focusedPaneId: 'deep' }, new Set())
  assert.ok(layout.leaves(bounded).length <= 16)
  assert.doesNotThrow(() => layout.sanitizeLayout({ version: 1, root: null, focusedPaneId: '' }, new Set()))

  const cycle = { type: 'split', id: 'cycle', direction: 'horizontal', ratio: 0.5, children: [] }
  cycle.children = [cycle, { type: 'leaf', paneId: 'safe', sessionId: null }]
  bounded = layout.sanitizeLayout({ version: 1, root: cycle, focusedPaneId: 'safe' }, new Set())
  assert.equal(layout.leaves(bounded).length, 1)
})

test('invalid layouts fall back to a valid empty layout', () => {
  for (const value of [null, {}, { version: 2, root: { type: 'leaf', paneId: 'p' } },
    { version: 1, root: { type: 'split', direction: 'diagonal', children: [] } }]) {
    const result = layout.sanitizeLayout(value, new Set())
    assert.equal(result.version, 1)
    assert.equal(layout.leaves(result).length, 1)
    assert.equal(result.focusedPaneId, layout.leaves(result)[0].paneId)
  }
})

import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/sessionActivity.ts', import.meta.url), 'utf8')
const compiled = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const { SessionActivityTracker } = await import('data:text/javascript;base64,' + Buffer.from(compiled).toString('base64'))

test('uses a list baseline, marks only newer unfocused output, and clears on terminal focus', () => {
  const tracker = new SessionActivityTracker()
  tracker.baseline([{ id: 's1', status: 'running', outputOffset: 100 }])
  assert.equal(tracker.has('s1'), false)
  tracker.output({ id: 's1', offset: 100 }, true, false, false)
  assert.equal(tracker.has('s1'), false)
  tracker.output({ id: 's1', offset: 120 }, true, false, false)
  assert.equal(tracker.has('s1'), true)
  tracker.output({ id: 's1', offset: 120 }, true, false, false)
  assert.equal(tracker.has('s1'), true)
  tracker.focus('s1')
  assert.equal(tracker.has('s1'), false)
  tracker.output({ id: 's1', offset: 121 }, true, false, true)
  assert.equal(tracker.has('s1'), false)
  tracker.output({ id: 's1', offset: 122 }, true, false, false)
  assert.equal(tracker.has('s1'), true)
})

test('binds output that arrived during create and ignores it when the session was focused', () => {
  const tracker = new SessionActivityTracker()
  tracker.baseline([])
  tracker.output({ id: 'new', offset: 5 }, false, true, false)
  tracker.created({ id: 'new', status: 'running', outputOffset: 0 }, false)
  assert.equal(tracker.has('new'), true)
  tracker.created({ id: 'focused', status: 'running', outputOffset: 0 }, false)
  tracker.output({ id: 'focused', offset: 4 }, false, true, true)
  tracker.created({ id: 'focused', status: 'running', outputOffset: 0 }, true)
  assert.equal(tracker.has('focused'), false)
})

test('ignores unknown sessions outside create and clears state on disconnect and removal', () => {
  const tracker = new SessionActivityTracker()
  tracker.baseline([])
  tracker.output({ id: 'external', offset: 10 }, false, false, false)
  assert.equal(tracker.has('external'), false)
  tracker.created({ id: 'managed', status: 'running', outputOffset: 0 }, false)
  tracker.output({ id: 'managed', offset: 1 }, true, false, false)
  assert.equal(tracker.has('managed'), true)
  tracker.remove('managed')
  assert.equal(tracker.has('managed'), false)
  tracker.disconnect()
  assert.equal(tracker.has('managed'), false)
})

test('disabled activity keeps an offset baseline but never marks output retroactively', () => {
  const tracker = new SessionActivityTracker()
  tracker.baseline([{ id: 's1', status: 'running', outputOffset: 0 }])
  tracker.output({ id: 's1', offset: 20 }, true, false, false, false)
  assert.equal(tracker.has('s1'), false)
  tracker.output({ id: 's1', offset: 20 }, true, false, false, true)
  assert.equal(tracker.has('s1'), false)
  tracker.output({ id: 's1', offset: 21 }, true, false, false, true)
  assert.equal(tracker.has('s1'), true)
})

import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/dig/watchState.ts', import.meta.url), 'utf8')
const compiled = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const { DigWatchState } = await import('data:text/javascript;base64,' + Buffer.from(compiled).toString('base64'))

function process(key, source = 'windows', distribution = '') {
  return { key, name: `proc-${key}`, pid: 42, parentPid: 1, kind: 'agent', source, distribution, startedAt: '2026-01-01T00:00:00Z' }
}

function status(source = 'windows', overrides = {}) {
  return {
    source, distribution: '', namespace: 'windows:boot-1', scannedAt: '2026-01-01T00:00:00Z',
    state: 'success', complete: true, truncated: false, policy: 'metadata-v1', reason: '', ...overrides,
  }
}

function snapshot(processes, sources, scannedAt = '2026-01-01T00:00:00Z') {
  return { processes, sources, scannedAt, truncated: false }
}

test('uses the first complete snapshot as baseline and diffs only comparable sources', () => {
  const watch = new DigWatchState()
  watch.update(snapshot([process('a')], [status()]))
  assert.equal(watch.changesList().length, 0)
  watch.update(snapshot([process('a'), process('b')], [status()]))
  assert.deepEqual(watch.changesList().map(change => [change.kind, change.key]), [['new', 'b']])
  watch.update(snapshot([process('b')], [status()]))
  assert.deepEqual(watch.changesList().map(change => change.kind), ['not-seen', 'new'])
})

test('keeps last trusted data when a source is partial or failed and compares other sources', () => {
  const watch = new DigWatchState()
  const windows = status()
  const wsl = status('wsl', { distribution: 'Ubuntu', namespace: 'wsl:Ubuntu:boot-1' })
  watch.update(snapshot([process('win'), process('linux', 'wsl', 'Ubuntu')], [windows, wsl]))
  watch.update(snapshot([process('win-new'), process('linux', 'wsl', 'Ubuntu')], [
    { ...windows, state: 'failed', complete: false, reason: 'process-enumeration-failed' },
    wsl,
  ]))
  assert.deepEqual(watch.trustedProcesses().map(item => item.key).sort(), ['linux', 'win'])
  assert.equal(watch.sources().find(item => item.source === 'windows').stale, true)
  assert.equal(watch.changesList().length, 0)
})

test('marks a stopped or restarted WSL distribution uncertain and never transfers a watched key', () => {
  const watch = new DigWatchState()
  const inventory = status('wsl', { distribution: '', namespace: 'wsl-running-set' })
  const ubuntu = status('wsl', { distribution: 'Ubuntu', namespace: 'wsl:Ubuntu:boot-1' })
  const oldProcess = process('old', 'wsl', 'Ubuntu')
  watch.update(snapshot([oldProcess], [inventory, ubuntu]))
  watch.toggleWatch('old')
  watch.update(snapshot([], [
    { ...inventory, reason: 'no-running-user-distros' },
  ]))
  assert.equal(watch.sources().find(item => item.distribution === 'Ubuntu').state, 'not-running')
  assert.equal(watch.trustedProcesses()[0].key, 'old')
  assert.equal(watch.watchedProcesses()[0].key, 'old')
  assert.equal(watch.followedState('old'), 'unknown')

  const restarted = status('wsl', { distribution: 'Ubuntu', namespace: 'wsl:Ubuntu:boot-2' })
  watch.update(snapshot([process('new', 'wsl', 'Ubuntu')], [inventory, restarted]))
  assert.equal(watch.changesList().some(item => item.key === 'new'), false)
  assert.equal(watch.isWatched('old'), true)
  assert.equal(watch.isWatched('new'), false)
  assert.equal(watch.followedState('old'), 'unknown')
  watch.update(snapshot([process('new', 'wsl', 'Ubuntu')], [inventory, restarted]))
  assert.equal(watch.followedState('old'), 'not-seen')
})

test('keeps Windows and WSL PID namespaces separate and treats a policy change as a new baseline', () => {
  const watch = new DigWatchState()
  const windows = status('windows', { namespace: 'windows:boot' })
  const ubuntu = status('wsl', { distribution: 'Ubuntu', namespace: 'wsl:Ubuntu:boot' })
  const left = { ...process('win-key'), pid: 77 }
  const right = { ...process('wsl-key', 'wsl', 'Ubuntu'), pid: 77 }
  watch.update(snapshot([left, right], [windows, ubuntu]))
  assert.equal(watch.trustedProcesses().length, 2)
  watch.update(snapshot([right], [{ ...windows, policy: 'metadata-v2' }, ubuntu]))
  assert.equal(watch.changesList().some(change => change.kind === 'not-seen'), false)
})

test('bounds followed objects and change history, with explicit clearing', () => {
  const watch = new DigWatchState()
  const initial = Array.from({ length: 101 }, (_, index) => process(`p${index}`))
  watch.update(snapshot(initial, [status()]))
  for (let index = 0; index < 101; index++) assert.equal(watch.toggleWatch(`p${index}`), index < 100)
  watch.update(snapshot([...initial, process('new')], [status()]))
  assert.equal(watch.changesList().length, 1)
  watch.clearChanges()
  assert.equal(watch.changesList().length, 0)
})

test('pausing scans marks sources stale and re-enabled scans establish a new baseline', () => {
  const watch = new DigWatchState()
  const before = [process('old')]
  watch.update(snapshot(before, [status()]))
  watch.pause()
  assert.equal(watch.sources()[0].stale, true)
  watch.update(snapshot([], [status()]))
  assert.equal(watch.changesList().length, 0)
  assert.equal(watch.trustedProcesses().length, 0)
})

test('expires change history by age while followed identities survive record clearing', () => {
  const watch = new DigWatchState()
  watch.update(snapshot([process('first')], [status()]), 0)
  watch.toggleWatch('first')
  watch.update(snapshot([process('second')], [status()]), 100)
  assert.equal(watch.changesList().length, 2)
  watch.pruneExpired(30 * 60 * 1000 + 101)
  assert.equal(watch.changesList().length, 0)
  assert.equal(watch.watchedProcesses()[0].key, 'first')
})

test('bounds the in-memory change record count', () => {
  const watch = new DigWatchState()
  watch.update(snapshot([], [status()]), 0)
  const processes = []
  for (let index = 0; index < 505; index++) {
    processes.push(process(`p${index}`))
    watch.update(snapshot([...processes], [status()]), index + 1)
  }
  assert.equal(watch.changesList().length, 500)
})

import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/sessionState.ts', import.meta.url), 'utf8')
const javascript = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 } }).outputText
const { mergeSessions } = await import('data:text/javascript;base64,' + Buffer.from(javascript).toString('base64'))

test('a delayed list reply cannot turn an exited SSH client back into running', () => {
  const current = [{ id: 'instance-a-1', status: 'exited', revision: 3, message: 'exit status 255' }]
  assert.deepEqual(mergeSessions(current, [{ id: 'instance-a-1', status: 'running', revision: 1 }], true), current)
  assert.equal(mergeSessions([{ id: 'old-1', status: 'exited' }], [{ id: 'old-1', status: 'running' }])[0].status, 'exited')
})

test('newer rename events win and replacement drops another daemon instance', () => {
  const current = [{ id: 'instance-a-1', status: 'running', revision: 4, name: 'renamed' }]
  assert.equal(mergeSessions(current, [{ id: 'instance-a-1', status: 'running', revision: 3, name: 'old' }])[0].name, 'renamed')
  assert.deepEqual(mergeSessions(current, [{ id: 'instance-b-1', status: 'running', revision: 1 }], true), [{ id: 'instance-b-1', status: 'running', revision: 1 }])
})

test('an exit event arriving before create response keeps a unique exited entry', () => {
  const exited = mergeSessions([], [{ id: 'a-1', status: 'exited', revision: 2 }])
  const merged = mergeSessions(exited, [{ id: 'a-1', status: 'running', revision: 1 }])
  assert.equal(merged.length, 1)
  assert.equal(merged[0].status, 'exited')
})

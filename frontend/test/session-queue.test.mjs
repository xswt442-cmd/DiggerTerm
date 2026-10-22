import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/sessionQueue.ts', import.meta.url), 'utf8')
const javascript = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const { queueSessionOperation, queueAcceptedTerminalInput } =
  await import('data:text/javascript;base64,' + Buffer.from(javascript).toString('base64'))

test('accepted input reaches daemon before a queued detach after pane unmount', async () => {
  const id = 'test-input-unique'
  let release
  const blocker = new Promise(resolve => { release = resolve })
  const events = []
  const holding = queueSessionOperation(id, async () => { await blocker })
  let mounted = true
  const send = async (_id, value) => {
    // The transport must not re-check component lifetime for accepted input.
    events.push(`input:${value}:mounted=${mounted}`)
  }
  const first = queueAcceptedTerminalInput(id, 'abc', send)
  const second = queueAcceptedTerminalInput(id, '\r', send)
  mounted = false
  const detach = queueSessionOperation(id, async () => { events.push('detach') })
  release()
  await Promise.all([holding, first, second, detach])
  assert.deepEqual(events, ['input:abc:mounted=false', 'input:\r:mounted=false', 'detach'])
})

test('failed previous write does not prevent later input and detach', async () => {
  const id = 'test-failed-write-unique'
  const failed = queueAcceptedTerminalInput(id, 'bad', async () => { throw new Error('lost connection') })
  const events = []
  const next = queueAcceptedTerminalInput(id, 'next', async (_id, text) => { events.push(text) })
  const detach = queueSessionOperation(id, async () => { events.push('detach') })
  await assert.rejects(failed, /lost connection/)
  await Promise.all([next, detach])
  assert.deepEqual(events, ['next', 'detach'])
})

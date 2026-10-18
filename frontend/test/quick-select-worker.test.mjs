import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/quickSelectWorkerClient.ts', import.meta.url), 'utf8')
const compiled = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText.replace("new URL('./quickSelect.worker.ts', import.meta.url)", "'/fake-quick-select-worker.js'")
const moduleURL = 'data:text/javascript;base64,' + Buffer.from(compiled).toString('base64')
const workerClient = await import(moduleURL)

class FakeWorker {
  static instances = []
  onmessage
  onerror
  terminated = false
  constructor() { FakeWorker.instances.push(this) }
  postMessage(message) { this.message = message }
  terminate() { this.terminated = true }
}

test('terminates the worker when the regex budget expires', async () => {
  const originalWorker = globalThis.Worker
  globalThis.Worker = FakeWorker
  try {
    await assert.rejects(workerClient.scanQuickSelectInWorker('', { version: 1, enabledTypes: [], rules: [] }, undefined, 5), /exceeded 5 ms/)
    assert.equal(FakeWorker.instances.at(-1).terminated, true)
  } finally { globalThis.Worker = originalWorker }
})

test('terminates and rejects immediately when a scan is cancelled', async () => {
  const originalWorker = globalThis.Worker
  globalThis.Worker = FakeWorker
  const controller = new AbortController()
  try {
    const pending = workerClient.scanQuickSelectInWorker('', { version: 1, enabledTypes: [], rules: [] }, controller.signal, 1000)
    controller.abort()
    await assert.rejects(pending, { name: 'AbortError' })
    assert.equal(FakeWorker.instances.at(-1).terminated, true)
  } finally { globalThis.Worker = originalWorker }
})

import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/terminalReplay.ts', import.meta.url), 'utf8')
const javascript = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const { sliceOutput } = await import('data:text/javascript;base64,' + Buffer.from(javascript).toString('base64'))

test('output chunks preserve fragmented UTF-8 and ANSI bytes', () => {
  const expected = new TextEncoder().encode('\u001b[31m中文 🌲\u001b[0m\u001b[2;4Hcursor')
  const collected = []
  let offset = 0
  for (const byte of expected) {
    const data = Uint8Array.of(byte)
    collected.push(sliceOutput(offset, offset + data.length, data))
    offset += data.length
  }
  assert.deepEqual(Buffer.concat(collected.map(bytes => Buffer.from(bytes))), Buffer.from(expected))
})

test('overlapping output contributes only the unseen suffix', () => {
  const data = Uint8Array.from([10, 11, 12, 13, 14, 15])
  assert.deepEqual([...sliceOutput(13, 16, data)], [13, 14, 15])
  assert.deepEqual([...sliceOutput(16, 16, data)], [])
  assert.deepEqual([...sliceOutput(100, 16, data)], [])
})

test('duplicate output is ignored and gaps fail with an offset diagnostic', () => {
  assert.deepEqual([...sliceOutput(12, 12, Uint8Array.from([10, 11, 12]))], [])
  assert.throws(
    () => sliceOutput(5, 10, Uint8Array.from([7, 8, 9, 10])),
    /output gap: expected offset 5, received 6/,
  )
})

test('invalid offsets, short chunks, and non-byte data are rejected', () => {
  for (const offset of [-1, 0.5, Number.NaN, Number.POSITIVE_INFINITY, Number.MAX_SAFE_INTEGER + 1]) {
    assert.throws(() => sliceOutput(offset, 0, new Uint8Array()), /safe non-negative integer/)
    assert.throws(() => sliceOutput(0, offset, new Uint8Array()), /safe non-negative integer/)
  }
  assert.throws(() => sliceOutput(0, 2, Uint8Array.from([1, 2, 3])), /smaller than chunk length/)
  assert.throws(() => sliceOutput(0, 0, null), /Uint8Array/)
})

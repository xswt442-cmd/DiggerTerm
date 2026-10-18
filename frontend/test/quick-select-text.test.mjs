import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/quickSelect.ts', import.meta.url), 'utf8')
const compiled = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const quickSelect = await import('data:text/javascript;base64,' + Buffer.from(compiled).toString('base64'))

function makeLine(text, cells, isWrapped = false) {
  return {
    isWrapped, length: cells.length,
    translateToString: () => text,
    getCell: column => cells[column] ? {
      getChars: () => cells[column].text,
      getWidth: () => cells[column].width,
    } : undefined,
  }
}

test('extracts only visible or current-buffer text and maps UTF-16 offsets to terminal cells', async () => {
  const lines = new Map([
    [0, makeLine('old row', [...'old row'].map(text => ({ text, width: 1 })))],
    [10, makeLine('A😀中', [
      { text: 'A', width: 1 }, { text: '😀', width: 2 }, { text: '', width: 0 }, { text: '中', width: 2 }, { text: '', width: 0 },
    ])],
    [11, makeLine('é tail', [
      { text: 'é', width: 1 }, { text: ' ', width: 1 }, { text: 't', width: 1 }, { text: 'a', width: 1 }, { text: 'i', width: 1 }, { text: 'l', width: 1 },
    ], true)],
  ])
  const terminal = { rows: 2, buffer: { active: {
    baseY: 10, viewportY: 10, length: 12,
    getLine: index => lines.get(index),
  } } }
  const visible = await quickSelect.collectQuickSelectText(terminal, 'visible')
  assert.equal(visible.text, 'A😀中é tail')
  const candidates = [
    { label: 'a', type: 'custom:r', name: 'Emoji', text: '😀', start: 1, end: 3, priority: 100 },
    { label: 'b', type: 'custom:r', name: 'Accent', text: 'é', start: 4, end: 6, priority: 100 },
  ]
  const positioned = quickSelect.positionQuickSelectCandidates({ candidates, truncated: false, errors: [] }, visible)
  assert.deepEqual(positioned.map(({ bufferRow, column }) => [bufferRow, column]), [[10, 1], [11, 0]])

  const buffer = await quickSelect.collectQuickSelectText(terminal, 'buffer')
  assert.ok(buffer.text.startsWith('old row\n'))
  assert.ok(buffer.text.endsWith('A😀中é tail'))
})

test('cancels chunked buffer extraction when the terminal context changes', async () => {
  const empty = makeLine('', [])
  const terminal = { rows: 512, buffer: { active: {
    baseY: 512, viewportY: 0, length: 1024, getLine: () => empty,
  } } }
  const abort = new AbortController()
  const pending = quickSelect.collectQuickSelectText(terminal, 'buffer', abort.signal)
  abort.abort()
  await assert.rejects(pending, { name: 'AbortError' })
})

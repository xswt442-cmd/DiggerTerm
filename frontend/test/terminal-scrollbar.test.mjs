import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/terminalScrollbar.ts', import.meta.url), 'utf8')
const javascript = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const { scrollbarMetrics, scrollLineFromThumb } = await import('data:text/javascript;base64,' + Buffer.from(javascript).toString('base64'))

test('scrollbar thumb reflects the terminal viewport and scrollback', () => {
  assert.deepEqual(scrollbarMetrics(0, 0, 24), {
    thumbHeightPercent: 100,
    thumbTopPercent: 0,
    scrollable: false,
  })
  assert.deepEqual(scrollbarMetrics(0, 976, 24), {
    thumbHeightPercent: 2.4,
    thumbTopPercent: 0,
    scrollable: true,
  })
  const bottom = scrollbarMetrics(976, 976, 24)
  assert.equal(bottom.thumbTopPercent, 97.6)
  assert.equal(bottom.thumbHeightPercent, 2.4)
})

test('dragging the thumb maps its full track to terminal scrollback', () => {
  assert.equal(scrollLineFromThumb(100, 100, 400, 40, 20, 960), 0)
  assert.equal(scrollLineFromThumb(300, 100, 400, 40, 20, 960), 480)
  assert.equal(scrollLineFromThumb(480, 100, 400, 40, 20, 960), 960)
  assert.equal(scrollLineFromThumb(400, 100, 0, 40, 20, 960), 0)
})

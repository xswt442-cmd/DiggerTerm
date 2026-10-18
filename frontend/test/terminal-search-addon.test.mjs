import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { join, resolve } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { after, test } from 'node:test'
import ts from 'typescript'

globalThis.window = {
  setTimeout: (callback, _delay) => setTimeout(callback, 0),
  clearTimeout,
}
globalThis.self = globalThis
const require = createRequire(import.meta.url)
const { Terminal } = require('@xterm/headless')
const { SearchAddon } = require('@xterm/addon-search')
const frontendRoot = resolve(fileURLToPath(new URL('..', import.meta.url)))
const tempRoot = await mkdtemp(join(frontendRoot, 'test', '.terminal-search-addon-'))
const helperSource = await readFile(resolve(frontendRoot, 'src/terminalSearch.ts'), 'utf8')
const helperPath = join(tempRoot, 'terminalSearch.mjs')
await writeFile(helperPath, ts.transpileModule(helperSource, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText)
const { withTerminalSearchSupport } = await import(pathToFileURL(helperPath).href)

function write(terminal, data) {
  return new Promise(resolve => terminal.write(data, resolve))
}

test('SearchAddon finds and reports a match in the active terminal buffer', async () => {
  const terminal = new Terminal(withTerminalSearchSupport({ cols: 80, rows: 24, scrollback: 1000 }))
  // The headless build omits the browser-only decoration implementation.
  terminal.registerMarker = (cursorOffset = 0) => ({
    line: terminal.buffer.active.baseY + terminal.buffer.active.cursorY + cursorOffset,
    dispose() {},
  })
  terminal.registerDecoration = options => ({
    marker: options.marker,
    dispose() {},
    onDispose: () => ({ dispose() {} }),
    onRender: () => ({ dispose() {} }),
  })
  // SearchAddon subscribes to this browser terminal event during activation.
  terminal.onWriteParsed = () => ({ dispose() {} })
  terminal.getSelectionPosition = () => undefined
  terminal.clearSelection = () => {}
  terminal.select = () => {}
  const addon = new SearchAddon()
  terminal.loadAddon(addon)
  try {
    await write(terminal, 'Write-Output PHASE1_QA_ALPHA\r\n')
    const results = []
    addon.onDidChangeResults(result => results.push(result))

    const found = addon.findNext('PHASE1_QA_ALPHA', {
      decorations: {
        matchBackground: 'red',
        matchBorder: 'white',
        matchOverviewRuler: 'white',
        activeMatchBackground: 'blue',
        activeMatchBorder: 'white',
        activeMatchColorOverviewRuler: 'white',
      },
    })

    assert.equal(found, true)
    assert.deepEqual(results.at(-1), { resultIndex: 0, resultCount: 1 })
  } finally {
    addon.dispose()
    terminal.dispose()
  }
})

after(async () => { await rm(tempRoot, { recursive: true, force: true }) })

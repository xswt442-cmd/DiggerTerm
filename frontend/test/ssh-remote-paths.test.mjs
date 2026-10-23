import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const code = readFileSync(new URL('../src/sshRemotePaths.ts', import.meta.url), 'utf8')
const compiled = ts.transpileModule(code, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const { remoteChildPath, remoteParentPath } = await import('data:text/javascript;base64,' + Buffer.from(compiled).toString('base64'))

test('remote browser joins root and nested paths without changing the host', () => {
  assert.equal(remoteChildPath('/', 'srv'), '/srv')
  assert.equal(remoteChildPath('/srv', 'My Project'), '/srv/My Project')
  assert.equal(remoteChildPath('/srv', "O'Brien"), "/srv/O'Brien")
})

test('remote parent navigation stops at root', () => {
  assert.equal(remoteParentPath('/srv/projects'), '/srv')
  assert.equal(remoteParentPath('/srv'), '/')
  assert.equal(remoteParentPath('/'), '/')
})

test('rejects parent traversal and slash-bearing directory labels', () => {
  for (const name of ['', '.', '..', 'a/b', 'a\\b']) {
    assert.throws(() => remoteChildPath('/srv', name))
  }
  assert.throws(() => remoteChildPath('relative', 'valid'))
})

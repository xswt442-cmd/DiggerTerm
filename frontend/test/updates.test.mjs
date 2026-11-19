import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/updates.ts', import.meta.url), 'utf8')
const javascript = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const updates = await import('data:text/javascript;base64,' + Buffer.from(javascript).toString('base64'))

test('compares semantic release versions without lexical mistakes', () => {
  assert.ok(updates.compareVersions('0.0.10', '0.0.9') > 0)
  assert.equal(updates.compareVersions('1.2.3', '1.2.3'), 0)
  assert.ok(updates.compareVersions('1.1.9', '1.2.0') < 0)
  assert.throws(() => updates.compareVersions('1.2', '1.2.3'), /Invalid semantic version/)
})

test('accepts only a stable semantic latest-release tag', () => {
  assert.deepEqual(updates.parseLatestRelease({ tag_name: 'v1.2.3' }), { tag: 'v1.2.3', version: '1.2.3' })
  assert.deepEqual(updates.parseLatestRelease({ tag_name: '1.2.3' }), { tag: '1.2.3', version: '1.2.3' })
  for (const value of [null, {}, { tag_name: 'latest' }, { tag_name: 'v1.2' }, { tag_name: 'v01.2.3' }]) {
    assert.throws(() => updates.parseLatestRelease(value))
  }
})

test('automatic checks run at most once per 24-hour interval', () => {
  const now = Date.UTC(2026, 10, 20, 12)
  assert.equal(updates.automaticUpdateCheckDue(null, now), true)
  assert.equal(updates.automaticUpdateCheckDue(now - 1_000, now), false)
  assert.equal(updates.automaticUpdateCheckDue(now - updates.AUTO_UPDATE_CHECK_INTERVAL_MS + 1, now), false)
  assert.equal(updates.automaticUpdateCheckDue(now - updates.AUTO_UPDATE_CHECK_INTERVAL_MS, now), true)
})

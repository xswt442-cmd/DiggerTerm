import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/quickSelectCore.ts', import.meta.url), 'utf8')
const compiled = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const externalLinks = readFileSync(new URL('../src/externalLinks.ts', import.meta.url), 'utf8')
const linkModule = ts.transpileModule(externalLinks, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const linkURL = 'data:text/javascript;base64,' + Buffer.from(linkModule).toString('base64')
const moduleURL = 'data:text/javascript;base64,' + Buffer.from(compiled.replace("'./externalLinks'", `'${linkURL}'`)).toString('base64')
const { scanQuickSelectCandidates } = await import(moduleURL)

function config(overrides = {}) {
  return { version: 1, enabledTypes: ['url', 'ip', 'path', 'gitHash'], rules: [], ...overrides }
}

test('finds web links, valid IP addresses, common paths, and Git hashes', () => {
  const result = scanQuickSelectCandidates(
    'Open https://example.com/a, then 192.168.1.8 and 999.1.1.1. Use /home/sam/my project/file.go or C:\\Work\\repo. Commit abcdef1234567.',
    config(),
  )
  assert.deepEqual(result.candidates.map(({ text, type }) => [text, type]), [
    ['https://example.com/a', 'url'], ['192.168.1.8', 'ipv4'], ['/home/sam/my project/file.go', 'path'],
    ['C:\\Work\\repo', 'path'], ['abcdef1234567', 'gitHash'],
  ])
  assert.deepEqual(result.candidates.map(({ label }) => label), ['a', 'b', 'c', 'd', 'e'])
})

test('uses stable built-in priority, de-duplicates overlaps, and respects enabled types', () => {
  const result = scanQuickSelectCandidates('https://example.com/deadbeef and deadbeef and ::1', config({
    rules: [{ id: 'r1', name: 'Custom', pattern: 'deadbeef', enabled: true }],
  }))
  assert.equal(result.candidates[0].type, 'url')
  assert.deepEqual(result.candidates.filter(item => item.text === 'deadbeef').map(item => item.type), ['gitHash'])
  assert.ok(result.candidates.some(item => item.type === 'ipv6' && item.text === '::1'))
  const disabled = scanQuickSelectCandidates('https://example.com 192.168.1.1', config({ enabledTypes: [] }))
  assert.equal(disabled.candidates.length, 0)
})

test('skips zero-length custom matches and reports an invalid rule without stopping built-ins', () => {
  const result = scanQuickSelectCandidates('https://example.com', config({
    rules: [
      { id: 'empty', name: 'Empty', pattern: '(?:)', enabled: true },
      { id: 'bad', name: 'Bad', pattern: '[', enabled: true },
    ],
  }))
  assert.equal(result.candidates[0].type, 'url')
  assert.ok(result.errors.some(error => error.startsWith('Bad:')))
  assert.equal(result.candidates.some(candidate => candidate.type === 'custom:empty'), false)
})

test('assigns equally long, prefix-free labels after the single-letter range', () => {
  const text = Array.from({ length: 30 }, (_, index) => `hash${index} ${String(index + 1000000).padStart(7, 'a')}`).join('\n')
  const result = scanQuickSelectCandidates(text, config({ enabledTypes: [], rules: [
    { id: 'numbers', name: 'Numbers', pattern: 'hash\\d+', enabled: true },
  ] }))
  assert.equal(result.candidates.length, 30)
  assert.ok(result.candidates.every(candidate => candidate.label.length === 2))
  for (const left of result.candidates) for (const right of result.candidates) {
    if (left !== right) assert.equal(right.label.startsWith(left.label), false)
  }
})

test('stops at the candidate and input bounds', () => {
  const many = Array.from({ length: 520 }, (_, index) => `hash${index}`).join(' ')
  const result = scanQuickSelectCandidates(many, config({ enabledTypes: [], rules: [
    { id: 'many', name: 'Many', pattern: 'hash\\d+', enabled: true },
  ] }))
  assert.equal(result.candidates.length, 500)
  assert.equal(result.truncated, true)
  assert.throws(() => scanQuickSelectCandidates('x'.repeat(2 * 1024 * 1024 + 1), config()))
})

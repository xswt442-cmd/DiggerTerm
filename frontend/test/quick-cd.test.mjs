import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/quickCd.ts', import.meta.url), 'utf8')
const js = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const { cdShellForProfile, formatCdCommand, localDefaultDirectoryAllowed } =
  await import('data:text/javascript;base64,' + Buffer.from(js).toString('base64'))

test('profile selects correct shell quoting and unknown remote defaults to POSIX', () => {
  assert.equal(cdShellForProfile('powershell'), 'powershell')
  assert.equal(cdShellForProfile('cmd'), 'cmd')
  for (const profile of ['wsl', 'ssh', 'bash', 'shell']) assert.equal(cdShellForProfile(profile), 'posix')
})

test('PowerShell literal quoting blocks interpolation and handles apostrophes', () => {
  assert.equal(formatCdCommand("C:\\A B\\O'Brien;$x", 'powershell'),
    "Set-Location -LiteralPath 'C:\\A B\\O''Brien;$x'")
})

test('POSIX command quotes command substitution, apostrophes and option prefixes', () => {
  assert.equal(formatCdCommand("/tmp/a'$(touch x)", 'posix'), "cd '/tmp/a'\\''$(touch x)'")
  assert.equal(formatCdCommand('-project', 'posix'), "cd './-project'")
  assert.equal(formatCdCommand('~', 'posix'), 'cd')
  assert.equal(formatCdCommand("~/a'b", 'posix'), `cd "$HOME"/'a'\\''b'`)
  assert.equal(formatCdCommand('/mnt/c/Some Folder', 'posix'), "cd '/mnt/c/Some Folder'")
})

test('CMD does not interpret metacharacters and rejects expansion/quote hazards', () => {
  assert.equal(formatCdCommand('C:\\Projects\\Some Folder & tests', 'cmd'), 'cd /d "C:\\Projects\\Some Folder & tests"')
  for (const path of ['C:\\%TEMP%', 'a!b', 'some^name', 'foo"bar']) {
    assert.throws(() => formatCdCommand(path, 'cmd'))
  }
})

test('rejects control chars, multi-line injection and excessive paths', () => {
  for (const shell of ['posix', 'cmd', 'powershell']) {
    for (const path of ['', 'a\nb', '\x1b[0m', 'a\u202eb', 'x'.repeat(2049)]) {
      assert.throws(() => formatCdCommand(path, shell))
    }
  }
})

test('only OS-local absolute paths can be saved as new-session defaults', () => {
  assert.ok(localDefaultDirectoryAllowed('powershell', 'C:\\dev\\repo', true))
  assert.ok(localDefaultDirectoryAllowed('bash', '/home/me/code', false))
  assert.equal(localDefaultDirectoryAllowed('ssh', '/srv/repo', false), false)
  assert.equal(localDefaultDirectoryAllowed('wsl', '/mnt/c/dev', true), false)
  assert.equal(localDefaultDirectoryAllowed('cmd', '..\\repo', true), false)
  assert.equal(localDefaultDirectoryAllowed('shell', './repo', false), false)
})

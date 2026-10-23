import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/quickCd.ts', import.meta.url), 'utf8')
const js = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const { cdShellForProfile, formatCdCommand, localDefaultDirectoryAllowed, directoryFromNativePicker, supportsLocalDefaultDirectory, supportsNativeFolderPicker } =
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


test('native folder picks preserve Windows paths for PowerShell/CMD', () => {
  const local = 'E:\\Work projects\\Code'
  assert.equal(directoryFromNativePicker('powershell', local, true), local)
  assert.equal(directoryFromNativePicker('cmd', local, true), local)
  assert.equal(directoryFromNativePicker('powershell', '\\\\fileserver\\share\\dev', true), '\\\\fileserver\\share\\dev')
})

test('native Windows folders map to WSL automount and Git Bash drives', () => {
  assert.equal(directoryFromNativePicker('wsl', 'C:\\Projects\\Hello World', true), '/mnt/c/Projects/Hello World')
  assert.equal(directoryFromNativePicker('bash', 'E:\\.codes\\DiggerTerm', true), '/e/.codes/DiggerTerm')
  assert.equal(directoryFromNativePicker('wsl', 'D:\\', true), '/mnt/d/')
  assert.equal(directoryFromNativePicker('bash', 'C:/Work', true), '/c/Work')
})

test('native Unix folder picks use POSIX absolute paths unchanged', () => {
  assert.equal(directoryFromNativePicker('shell', "/home/dev/it's here", false), "/home/dev/it's here")
  assert.equal(directoryFromNativePicker('bash', '/tmp/space folder', false), '/tmp/space folder')
})

test('folder picker cannot browse remote hosts or translate UNC to WSL/Git Bash', () => {
  for (const profile of ['ssh', 'unknown']) {
    assert.throws(() => directoryFromNativePicker(profile, 'C:\\Work', true), /remote|native/i)
  }
  for (const profile of ['bash', 'wsl']) {
    assert.throws(() => directoryFromNativePicker(profile, '\\\\wsl.localhost\\Ubuntu\\home', true), /UNC/)
    assert.throws(() => directoryFromNativePicker(profile, '\\\\server\\share', true), /UNC/)
  }
  assert.throws(() => directoryFromNativePicker('cmd', '\\\\server\\share', true), /UNC/)
  assert.throws(() => directoryFromNativePicker('shell', 'relative/folder', false), /absolute/)
  assert.throws(() => directoryFromNativePicker('powershell', 'C:\\abc\nmalicious', true), /invalid/)
  assert.throws(() => directoryFromNativePicker('wsl', 'C:\\' + 'x'.repeat(2050), true), /invalid/)
})

test('profile capabilities consistently distinguish local defaults and native picking', () => {
  for (const profile of ['powershell', 'cmd', 'bash', 'shell']) {
    assert.equal(supportsLocalDefaultDirectory(profile), true)
    assert.equal(supportsNativeFolderPicker(profile), true)
  }
  assert.equal(supportsLocalDefaultDirectory('wsl'), false)
  assert.equal(supportsNativeFolderPicker('wsl'), true)
  assert.equal(supportsLocalDefaultDirectory('ssh'), false)
  assert.equal(supportsNativeFolderPicker('ssh'), false)
})

test('only OS-local absolute paths can be saved as new-session defaults', () => {
  assert.ok(localDefaultDirectoryAllowed('powershell', 'C:\\dev\\repo', true))
  assert.ok(localDefaultDirectoryAllowed('bash', '/home/me/code', false))
  assert.equal(localDefaultDirectoryAllowed('ssh', '/srv/repo', false), false)
  assert.equal(localDefaultDirectoryAllowed('wsl', '/mnt/c/dev', true), false)
  assert.equal(localDefaultDirectoryAllowed('cmd', '..\\repo', true), false)
  assert.equal(localDefaultDirectoryAllowed('shell', './repo', false), false)
})

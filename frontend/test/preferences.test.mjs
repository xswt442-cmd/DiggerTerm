import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test, beforeEach } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/uiPreferences.ts', import.meta.url), 'utf8')
const javascript = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const prefs = await import('data:text/javascript;base64,' + Buffer.from(javascript).toString('base64'))

const values = new Map()
globalThis.localStorage = {
  getItem: key => values.get(key) ?? null,
  setItem: (key, value) => values.set(key, value),
}
beforeEach(() => { values.clear() })

test('defaults work with no saved WebView preferences', () => {
  assert.deepEqual(prefs.readPreferences(), prefs.defaults)
})

test('all expanded preferences persist across GUI reopen', () => {
  const saved = {
    ...prefs.defaults,
    theme: 'builtin:pine',
    fontFamily: 'jetbrains',
    fontSize: 17,
    uiFontSize: 15,
    sidebarCollapsed: true,
    language: 'zh',
    defaultProfile: 'cmd',
    defaultDirectory: 'C:\\dev\\work',
    scrollback: 25000,
    lineHeight: 1.4,
    scrollSensitivity: 5,
    terminalPadding: 6,
    panePromptStyle: 'subtle',
    cursorStyle: 'bar',
    keyboardMode: 'native',
    cursorBlink: false,
    rightClickSelectsWord: true,
    activateNewSession: false,
    observeEnabled: false,
    observeRefreshSeconds: 0,
    showPid: false,
    showStatusBar: false,
    sidebarWidth: 370,
    managedRatio: 0.67,
  }
  prefs.writePreferences(saved)
  assert.deepEqual(prefs.readPreferences(), saved)
})

test('bad saved values are clamped or safely ignored', () => {
  values.set('diggerterm:preferences:v1', JSON.stringify({
    fontSize: 500,
    uiFontSize: 500,
    fontFamily: '__proto__',
    theme: 'malicious',
    cursorStyle: 'hidden',
    keyboardMode: 'untrusted',
    panePromptStyle: 'obtrusive',
    defaultProfile: '-oBadFlag',
    scrollback: 99_999_999,
    terminalPadding: -1,
    observeRefreshSeconds: -1000,
    cursorBlink: 'no',
    sidebarWidth: 100_000,
    managedRatio: -20,
  }))
  const read = prefs.readPreferences()
  assert.equal(read.fontSize, 20)
  assert.equal(read.uiFontSize, 16)
  assert.equal(read.fontFamily, prefs.defaults.fontFamily)
  assert.equal(read.theme, prefs.defaults.theme)
  assert.equal(read.cursorStyle, prefs.defaults.cursorStyle)
  assert.equal(read.keyboardMode, prefs.defaults.keyboardMode)
  assert.equal(read.panePromptStyle, prefs.defaults.panePromptStyle)
  assert.equal(read.defaultProfile, prefs.defaults.defaultProfile)
  assert.equal(read.scrollback, prefs.defaults.scrollback)
  assert.equal(read.terminalPadding, prefs.defaults.terminalPadding)
  assert.equal(read.observeRefreshSeconds, prefs.defaults.observeRefreshSeconds)
  assert.equal(read.cursorBlink, true)
  assert.equal(read.sidebarWidth, 500)
  assert.equal(read.managedRatio, .2)
})

test('corrupt settings recover defaults without crashing UI', () => {
  values.set('diggerterm:preferences:v1', '{broken json')
  assert.deepEqual(prefs.readPreferences(), prefs.defaults)
})

test('theme IDs migrate and preserve user theme identity until catalog refresh', () => {
  values.set('diggerterm:preferences:v1', JSON.stringify({ theme: 'pine' }))
  assert.equal(prefs.readPreferences().theme, 'builtin:pine')
  values.set('diggerterm:preferences:v1', JSON.stringify({ theme: 'user:暮光', customCSS: false }))
  assert.equal(prefs.readPreferences().theme, 'user:暮光')
  assert.equal(prefs.readPreferences().customCSS, false)
})

test('new setting labels appear in both UI languages', () => {
  for (const key of ['terminalTab', 'sessionTab', 'monitorTab', 'fontFamily', 'uiFontSize', 'scrollback',
    'profileDefault', 'observeEnabled', 'refreshSeconds', 'keyboardMode', 'panePromptStyle',
    'panePromptCurrent', 'panePromptSubtle', 'panePromptNone', 'collapseSidebar', 'expandSidebar',
    'sessionDiagnostics', 'showStatusBar']) {
    assert.notEqual(prefs.translate('zh', key), '')
    assert.notEqual(prefs.translate('en', key), '')
  }
})

test('home screen uses the requested concise guidance', () => {
  assert.equal(prefs.translate('zh', 'emptyTitle'), '一个窗口，管理多个终端。')
  assert.equal(prefs.translate('zh', 'emptyDesc'), '从左侧选择 Shell 并打开')
  assert.equal(prefs.translate('zh', 'emptyHint'), '退出 DiggerTerm 后会话保留，外部进程仅只读')
})

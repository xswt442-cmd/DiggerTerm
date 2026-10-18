import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'

const source = readFileSync(new URL('../src/externalLinks.ts', import.meta.url), 'utf8')
const compiled = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText
const { createExplicitWebLinkActivator, validateExternalWebURL } = await import('data:text/javascript;base64,' + Buffer.from(compiled).toString('base64'))

test('accepts absolute HTTP and HTTPS links and exposes the parsed destination', () => {
  assert.equal(validateExternalWebURL('https://example.com/a?q=1').hostname, 'example.com')
  assert.equal(validateExternalWebURL('http://example.com').protocol, 'http:')
  assert.equal(validateExternalWebURL('https://bücher.example').hostname, 'xn--bcher-kva.example')
})

test('rejects non-web protocols, credentials, controls, malformed and oversized URLs', () => {
  for (const value of [
    'javascript:alert(1)', 'file:///C:/Windows/win.ini', 'data:text/html,x', 'ftp://example.com',
    'https://user@example.com', 'https://user:secret@example.com', 'https://example.com\n.evil',
    ' https://example.com', 'https://example.com\u202e.evil', 'https://', 'https://' + 'a'.repeat(4090),
  ]) assert.throws(() => validateExternalWebURL(value))
})

test('requires the platform modifier for terminal text and OSC 8 links', () => {
  for (const [isMac, modifier] of [[false, 'ctrlKey'], [true, 'metaKey']]) {
    const opened = []
    const activator = createExplicitWebLinkActivator(isMac, value => opened.push(validateExternalWebURL(value).href))
    const event = (active) => ({
      ctrlKey: active && modifier === 'ctrlKey',
      metaKey: active && modifier === 'metaKey',
      prevented: false,
      preventDefault() { this.prevented = true },
    })
    const ordinary = event(false)
    activator(ordinary, 'https://example.com')
    assert.equal(ordinary.prevented, false)
    assert.deepEqual(opened, [])

    const explicit = event(true)
    activator(explicit, 'https://example.com/path')
    assert.equal(explicit.prevented, true)
    assert.deepEqual(opened, ['https://example.com/path'])

    const unsafe = event(true)
    assert.throws(() => activator(unsafe, 'https://user@example.com'))
    assert.equal(unsafe.prevented, true)
    assert.deepEqual(opened, ['https://example.com/path'])
  }
})

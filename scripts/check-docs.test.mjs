import assert from 'node:assert/strict'
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, dirname } from 'node:path'
import { test } from 'node:test'
import { checkDocumentation } from './check-docs.mjs'

function fixture(t) {
  const root = mkdtempSync(join(tmpdir(), 'diggerterm-doc-check-'))
  t.after(() => rmSync(root, { recursive: true, force: true }))
  const write = (path, contents) => {
    const destination = join(root, path)
    mkdirSync(dirname(destination), { recursive: true })
    writeFileSync(destination, contents)
  }
  write('README.md', '[English](README.en.md)')
  write('README.en.md', '[中文](README.md)')
  write('AGENTS.md', '# Agent guidelines')
  write('frontend/package.json', JSON.stringify({ version: '0.0.6' }))
  write('docs/development/ARCHITECTURE.zh-CN.md', '[English](ARCHITECTURE.en.md)')
  write('docs/development/ARCHITECTURE.en.md', '[中文](ARCHITECTURE.zh-CN.md)')
  write('docs/development/RELEASE.md', '# Release')
  write('CHANGELOG.md', [
    '# Changelog', '## [Unreleased]', '### 简体中文', '### English',
    '## [0.0.6] - 2026-10-18',
    '### 简体中文', '**修复**', '- 修复会话控制问题。',
    '### English', '**Fixed**', '- Fixed session control.',
  ].join('\n'))
  return { root, write }
}

test('accepts complete paired docs and matching bilingual categories', t => {
  const { root } = fixture(t)
  assert.deepEqual(checkDocumentation(root), [])
})

test('detects missing translations and broken local links', t => {
  const { root, write } = fixture(t)
  write('docs/development/API.en.md', '# API')
  write('README.md', '[missing](does-not-exist.md)')
  const errors = checkDocumentation(root).join('\n')
  assert.match(errors, /Missing translation/)
  assert.match(errors, /Broken local markdown link/)
})

test('detects release version and bilingual category mismatch', t => {
  const { root, write } = fixture(t)
  write('frontend/package.json', JSON.stringify({ version: '0.0.7' }))
  write('CHANGELOG.md', [
    '## [Unreleased]', '### 简体中文', '### English',
    '## [0.0.6] - 2026-10-18',
    '### 简体中文', '**修复**', '- 已修复。',
    '### English', '**Added**', '- Incorrect type.',
  ].join('\n'))
  const errors = checkDocumentation(root).join('\n')
  assert.match(errors, /Latest CHANGELOG version/)
  assert.match(errors, /Category order\/translation mismatch/)
})

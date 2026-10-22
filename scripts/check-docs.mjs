#!/usr/bin/env node
// Validate the repository's user-facing documentation contract on every PR.
// Published release metadata is verified separately by verify-release.mjs.
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { dirname, join, relative, resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'
import { extractReleaseNotes } from './release-notes.mjs'

const categoryTranslations = new Map([
  ['新增', 'Added'], ['改进', 'Improved'], ['修复', 'Fixed'],
  ['安全', 'Security'], ['兼容性', 'Compatibility'],
  ['移除', 'Removed'], ['升级说明', 'Upgrade notes'],
  ['使用说明', 'Usage notes'],
])

export function checkDocumentation(root) {
  const problems = []
  const required = ['README.md', 'README.en.md', 'AGENTS.md', 'CHANGELOG.md',
    'docs/development/ARCHITECTURE.zh-CN.md', 'docs/development/ARCHITECTURE.en.md',
    'docs/development/RELEASE.md']
  for (const name of required) {
    if (!existsSync(join(root, name))) problems.push(`Missing required document: ${name}`)
  }

  const docDir = join(root, 'docs', 'development')
  const names = existsSync(docDir) ? readdirSync(docDir).filter(name => name.endsWith('.md')) : []
  for (const name of names) {
    const match = /^(.*)\.(zh-CN|en)\.md$/.exec(name)
    if (match) {
      const counterpart = `${match[1]}.${match[2] === 'en' ? 'zh-CN' : 'en'}.md`
      if (!names.includes(counterpart)) problems.push(`Missing translation for docs/development/${name}: ${counterpart}`)
    }
  }

  // Check local markdown links without chasing external URLs or fragment IDs.
  const docs = [...required, ...names.map(name => `docs/development/${name}`)]
  for (const name of new Set(docs)) {
    const absolute = join(root, name)
    if (!existsSync(absolute)) continue
    const contents = readFileSync(absolute, 'utf8')
    for (const match of contents.matchAll(/\]\(([^)]+)\)/g)) {
      const raw = match[1].trim().split(/\s+["']/)[0]
      if (!raw || /^(?:[a-z][a-z\d+.-]*:|#)/i.test(raw)) continue
      let path
      try { path = decodeURIComponent(raw.split(/[?#]/, 1)[0]) }
      catch { problems.push(`Invalid markdown link in ${name}: ${raw}`); continue }
      if (!path) continue
      const target = resolve(dirname(absolute), path)
      const rel = relative(root, target)
      if (rel === '..' || rel.startsWith('..' + sep)) {
        problems.push(`Link escapes repository: ${name} -> ${raw}`)
      } else if (!existsSync(target)) {
        problems.push(`Broken local markdown link: ${name} -> ${raw}`)
      }
    }
  }

  const changelogPath = join(root, 'CHANGELOG.md')
  const packagePath = join(root, 'frontend/package.json')
  if (existsSync(changelogPath) && existsSync(packagePath)) {
    const changelog = readFileSync(changelogPath, 'utf8')
    const manifest = JSON.parse(readFileSync(packagePath, 'utf8'))
    const entries = [...changelog.matchAll(/^## \[([^\]]+)\](?: - .+)?$/gm)]
    if (!entries.length || entries[0][1] !== 'Unreleased') {
      problems.push('CHANGELOG must begin with [Unreleased]')
    }
    const versions = entries.filter(entry => entry[1] !== 'Unreleased').map(entry => entry[1])
    if (new Set(versions).size !== versions.length) {
      problems.push('Duplicate version in CHANGELOG')
    }
    if (versions[0] !== manifest.version) {
      problems.push('Latest CHANGELOG version does not match frontend/package.json')
    }
    for (const version of versions) {
      try { extractReleaseNotes(changelog, 'v' + version) }
      catch (error) { problems.push(`Invalid bilingual CHANGELOG [${version}]: ${error.message}`) }
    }
    for (const entry of entries.slice(0, 2)) {
      const until = changelog.indexOf('\n## [', entry.index + entry[0].length)
      const body = changelog.slice(entry.index + entry[0].length, until < 0 ? undefined : until)
      const zh = body.indexOf('### 简体中文')
      const en = body.indexOf('### English')
      if (zh < 0 || en <= zh) {
        problems.push(`Missing or misordered bilingual sections in [${entry[1]}]`)
        continue
      }
      const chinese = [...body.slice(zh, en).matchAll(/^\*\*([^*]+)\*\*$/gm)].map(m => m[1])
      const english = [...body.slice(en).matchAll(/^\*\*([^*]+)\*\*$/gm)].map(m => m[1])
      const translated = chinese.map(category => categoryTranslations.get(category) ?? '(unknown)')
      if (JSON.stringify(translated) !== JSON.stringify(english)) {
        problems.push(`Category order/translation mismatch in CHANGELOG [${entry[1]}]`)
      }
    }
  }
  return problems
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
  const problems = checkDocumentation(root)
  if (problems.length) {
    for (const problem of problems) console.error('ERROR: ' + problem)
    process.exitCode = 1
  } else {
    console.log('Documentation paths, translations and CHANGELOG metadata verified.')
  }
}

#!/usr/bin/env node
// Verify the release identity and its bilingual CHANGELOG section.
import { readFileSync } from 'node:fs'
import { extractReleaseNotes } from './release-notes.mjs'

const tag = process.argv[2]
if (!/^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/.test(tag ?? '')) {
  console.error('Expected a tag such as vX.Y.Z')
  process.exit(2)
}
const read = file => readFileSync(new URL('../' + file, import.meta.url), 'utf8')
const version = tag.slice(1)
const pkg = JSON.parse(read('frontend/package.json'))
const lock = JSON.parse(read('frontend/package-lock.json'))
const config = read('build/config.yml')
const win = JSON.parse(read('build/windows/info.json'))
const linux = read('build/linux/nfpm/nfpm.yaml')
const escapedVersion = version.replaceAll('.', '\\.')
const failures = []
const check = (ok, reason) => { if (!ok) failures.push(reason) }

check(pkg.version === version, 'frontend/package.json version mismatch')
check(lock.version === version && lock.packages?.['']?.version === version,
  'frontend/package-lock.json version mismatch')
check(new RegExp('^  version: "' + escapedVersion + '"', 'm').test(config),
  'build/config.yml version mismatch')
check(win.fixed?.file_version === version && win.info?.['0000']?.ProductVersion === version,
  'build/windows/info.json version mismatch')
check(new RegExp('^version: "' + escapedVersion + '"', 'm').test(linux),
  'build/linux/nfpm/nfpm.yaml version mismatch')
try {
  extractReleaseNotes(read('CHANGELOG.md'), tag)
} catch (error) {
  failures.push(error.message)
}
if (failures.length) {
  for (const failure of failures) console.error('ERROR: ' + failure)
  process.exit(1)
}
console.log('Release metadata and bilingual changelog verified: ' + tag)

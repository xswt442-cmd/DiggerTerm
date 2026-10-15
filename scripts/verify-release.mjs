#!/usr/bin/env node
// Release preflight shared by Windows, Linux and local dry runs.
import { existsSync, readFileSync } from 'node:fs'
import { join } from 'node:path'

const root = new URL('../', import.meta.url)
const version = process.argv[2]
if (!/^v\d+\.\d+\.\d+$/.test(version ?? '')) {
  console.error('Expected a version tag such as vX.Y.Z')
  process.exit(2)
}
const read = file => readFileSync(new URL(file, root), 'utf8')
const pkg = JSON.parse(read('frontend/package.json'))
const lock = JSON.parse(read('frontend/package-lock.json'))
const numeric = version.slice(1)
const config = read('build/config.yml')
const win = JSON.parse(read('build/windows/info.json'))
const failures = []
const check = (ok, why) => { if (!ok) failures.push(why) }
check(pkg.version === numeric, 'frontend/package.json version mismatch')
check(lock.version === numeric && lock.packages?.['']?.version === numeric,
  'frontend/package-lock.json version mismatch')
check(new RegExp('^  version: "' + numeric.replaceAll('.', '\\.') + '"', 'm').test(config),
  'build/config.yml version mismatch')
check(win.fixed?.file_version === numeric && win.info?.['0000']?.ProductVersion === numeric,
  'build/windows/info.json version mismatch')
check(new RegExp('^version: "' + numeric.replaceAll('.', '\\.') + '"', 'm')
  .test(read('build/linux/nfpm/nfpm.yaml')), 'build/linux/nfpm/nfpm.yaml version mismatch')
const notes = '.github/release-notes/' + version + '.md'
check(existsSync(new URL(notes, root)), 'missing bilingual release notes ' + notes)
if (failures.length) {
  for (const failure of failures) console.error('ERROR: ' + failure)
  process.exit(1)
}
console.log('Release metadata verified: ' + version)

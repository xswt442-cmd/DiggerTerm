import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { join, resolve } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { after, test } from 'node:test'
import ts from 'typescript'
import { compileScript, parse } from '@vue/compiler-sfc'
import { createRenderer, h, nextTick } from 'vue'

const root = resolve(fileURLToPath(new URL('..', import.meta.url)))
const temp = await mkdtemp(join(root, 'test', '.presets-build-'))
after(() => rm(temp, { recursive: true, force: true }))
const transpile = source => ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 } }).outputText
for (const module of ['quickCd', 'workspaceLibrary']) {
  await writeFile(join(temp, module + '.mjs'), transpile(await readFile(join(root, 'src', module + '.ts'), 'utf8')))
}
const path = join(root, 'src/components/SessionPresets.vue')
const { descriptor, errors } = parse(await readFile(path, 'utf8'), { filename: path })
assert.deepEqual(errors, [])
let script = compileScript(descriptor, { id: 'session-presets', inlineTemplate: true }).content
for (const module of ['quickCd', 'workspaceLibrary']) script = script.replaceAll(`'../${module}'`, `'${pathToFileURL(join(temp, module + '.mjs')).href}'`).replaceAll(`"../${module}"`, `'${pathToFileURL(join(temp, module + '.mjs')).href}'`)
const compiled = join(temp, 'SessionPresets.mjs')
await writeFile(compiled, transpile(script))
const { default: SessionPresets } = await import(pathToFileURL(compiled).href)

function mount() {
  const events = []
  const node = type => ({ type, text: '', props: {}, children: [], parent: null, listeners: {}, value: '', tagName: type.toUpperCase(), addEventListener(name, fn) { this.listeners[name] = fn }, removeEventListener(name) { delete this.listeners[name] }, getAttribute(name) { return this.props[name] } })
  const renderer = createRenderer({
    createElement: type => node(type), createText: text => ({ ...node('#text'), text }), createComment: text => ({ ...node('#comment'), text }),
    setText: (n, text) => { n.text = text }, setElementText: (n, text) => { n.text = text; n.children = [] },
    patchProp: (n, key, _old, value) => { n.props[key] = value },
    insert(n, parent, anchor) { if (n.parent) { const old = n.parent.children.indexOf(n); if (old >= 0) n.parent.children.splice(old, 1) }; n.parent = parent; const i = anchor ? parent.children.indexOf(anchor) : -1; if (i < 0) parent.children.push(n); else parent.children.splice(i, 0, n) },
    remove(n) { if (!n.parent) return; const i = n.parent.children.indexOf(n); if (i >= 0) n.parent.children.splice(i, 1); n.parent = null },
    parentNode: n => n.parent, nextSibling: n => n.parent?.children[n.parent.children.indexOf(n) + 1] ?? null,
  })
  const target = node('root')
  const app = renderer.createApp({ render: () => h(SessionPresets, { locale: 'en', presets: [{ id: 'p', name: 'Project', profile: 'shell', directory: '/work' }], profiles: [{ id: 'shell', name: 'Shell' }], defaultProfile: 'shell', defaultDirectory: '/work', busy: false, connected: true, onLaunch: p => events.push(['launch', p]), onRemove: id => events.push(['remove', id]) }) })
  app.mount(target)
  const all = () => { const result = []; const visit = n => { result.push(n); n.children.forEach(visit) }; visit(target); return result }
  return { app, events, all }
}

test('preset list launches and removes the selected saved preset', async () => {
  const descriptor = Object.getOwnPropertyDescriptor(globalThis, 'navigator')
  Object.defineProperty(globalThis, 'navigator', { configurable: true, value: { userAgent: 'Linux' } })
  try {
    const view = mount(); await nextTick()
    const launch = view.all().find(n => n.type === 'button' && n.props.title === '/work')
    const remove = view.all().find(n => n.type === 'button' && n.props['aria-label'] === 'Delete preset Project')
    assert.ok(launch); assert.ok(remove)
    launch.props.onClick(); remove.props.onClick()
    assert.deepEqual(view.events.map(e => e[0]), ['launch', 'remove'])
    assert.equal(view.events[0][1].profile, 'shell')
    assert.equal(view.events[1][1], 'p')
    view.app.unmount()
  } finally { if (descriptor) Object.defineProperty(globalThis, 'navigator', descriptor); else delete globalThis.navigator }
})

import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { join, resolve } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { after, test } from 'node:test'
import ts from 'typescript'
import { compileScript, parse } from '@vue/compiler-sfc'
import { createRenderer, defineComponent, h, nextTick } from 'vue'

const frontendRoot = resolve(fileURLToPath(new URL('..', import.meta.url)))
const tempRoot = await mkdtemp(join(frontendRoot, 'test', '.command-palette-build-'))
after(() => rm(tempRoot, { recursive: true, force: true }))

function transpile(source, filename) {
  return ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
    fileName: filename,
  }).outputText
}

const logicSource = await readFile(resolve(frontendRoot, 'src/commandPalette.ts'), 'utf8')
const logicPath = join(tempRoot, 'command-palette-logic.mjs')
await writeFile(logicPath, transpile(logicSource, `${logicPath}.ts`))
const logicURL = pathToFileURL(logicPath).href
const sourcePath = resolve(frontendRoot, 'src/components/CommandPalette.vue')
const { descriptor, errors } = parse(await readFile(sourcePath, 'utf8'), { filename: sourcePath })
assert.deepEqual(errors, [])
const script = compileScript(descriptor, { id: 'command-palette', inlineTemplate: true }).content
const javascript = transpile(script.replaceAll('"../commandPalette"', `'${logicURL}'`).replaceAll("'../commandPalette'", `'${logicURL}'`), `${sourcePath}.ts`)
const componentPath = join(tempRoot, 'CommandPalette.mjs')
await writeFile(componentPath, javascript)
const { default: CommandPalette } = await import(pathToFileURL(componentPath).href)

function mount() {
  let focused
  const node = (type, text = '') => ({
    type, text, value: '', props: {}, children: [], parent: null, listeners: new Map(),
    focus() { focused = this },
    addEventListener(name, handler) {
      const listeners = this.listeners.get(name) ?? []
      listeners.push(handler)
      this.listeners.set(name, listeners)
    },
    removeEventListener(name, handler) {
      this.listeners.set(name, (this.listeners.get(name) ?? []).filter(item => item !== handler))
    },
  })
  const body = node('body')
  const renderer = createRenderer({
    createElement: type => node(type),
    createText: text => node('#text', text),
    createComment: text => node('#comment', text),
    querySelector: selector => selector === 'body' ? body : null,
    setText: (target, text) => { target.text = text },
    setElementText: (target, text) => { target.children = []; target.text = text },
    patchProp: (target, key, _previous, value) => { target.props[key] = value },
    insert(target, parent, anchor = null) {
      if (target.parent) {
        const index = target.parent.children.indexOf(target)
        if (index >= 0) target.parent.children.splice(index, 1)
      }
      target.parent = parent
      const index = anchor ? parent.children.indexOf(anchor) : -1
      if (index < 0) parent.children.push(target)
      else parent.children.splice(index, 0, target)
    },
    remove(target) {
      if (!target.parent) return
      const index = target.parent.children.indexOf(target)
      if (index >= 0) target.parent.children.splice(index, 1)
      target.parent = null
    },
    parentNode: target => target.parent,
    nextSibling(target) { return target.parent?.children[target.parent.children.indexOf(target) + 1] ?? null },
  })
  const root = node('root')
  const commands = [
    { id: 'first', label: 'First', category: 'Test', keywords: [], enabled: true },
    { id: 'disabled', label: 'Disabled', category: 'Test', keywords: [], enabled: false },
    { id: 'second', label: 'Second', category: 'Test', keywords: [], enabled: true },
  ]
  const events = []
  const wrapper = defineComponent({
    setup() {
      return () => h(CommandPalette, {
        locale: 'en', commands,
        onClose: () => events.push('close'),
        onExecute: id => events.push(id),
      })
    },
  })
  const app = renderer.createApp(wrapper)
  app.mount(root)
  const find = predicate => {
    const visit = target => {
      if (predicate(target)) return target
      for (const child of target.children) {
        const found = visit(child)
        if (found) return found
      }
    }
    return visit(body)
  }
  return { app, events, find, focused: () => focused }
}

test('focuses the palette, skips disabled actions, executes the selected ID, and closes with Escape', async () => {
    const view = mount()
    await nextTick()
    const input = view.find(node => node.type === 'input')
    const dialog = view.find(node => node.props.role === 'dialog')
    assert.ok(input)
    assert.equal(view.focused()?.props['aria-label'], 'Search commands')
    dialog.props.onKeydown({ key: 'ArrowDown', isComposing: false, preventDefault() {}, stopPropagation() {} })
    dialog.props.onKeydown({ key: 'Enter', isComposing: false, preventDefault() {}, stopPropagation() {} })
    assert.deepEqual(view.events, ['second'])
    dialog.props.onKeydown({ key: 'Escape', isComposing: false, preventDefault() {}, stopPropagation() {} })
    assert.deepEqual(view.events, ['second', 'close'])
    view.app.unmount()
})

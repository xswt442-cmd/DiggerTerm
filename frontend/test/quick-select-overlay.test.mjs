import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { join, resolve } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { test } from 'node:test'
import ts from 'typescript'
import { compileScript, parse } from '@vue/compiler-sfc'
import { createRenderer, defineComponent, h, nextTick } from 'vue'

const frontendRoot = resolve(fileURLToPath(new URL('..', import.meta.url)))
const tempRoot = await mkdtemp(join(frontendRoot, 'test', '.quick-select-overlay-build-'))
const sourcePath = resolve(frontendRoot, 'src/components/QuickSelectOverlay.vue')
const { descriptor, errors } = parse(await readFile(sourcePath, 'utf8'), { filename: sourcePath })
assert.deepEqual(errors, [])
const javascript = ts.transpileModule(compileScript(descriptor, { id: 'quick-select-overlay', inlineTemplate: true }).content, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
  fileName: `${sourcePath}.ts`,
}).outputText
const componentPath = join(tempRoot, 'QuickSelectOverlay.mjs')
await writeFile(componentPath, javascript)
const { default: QuickSelectOverlay } = await import(pathToFileURL(componentPath).href)

function mount(candidates) {
  let focusTarget
  const node = (type, text = '') => ({
    type, text, props: {}, children: [], parent: null,
    focus() { focusTarget = this },
  })
  const renderer = createRenderer({
    createElement: type => node(type), createText: text => node('#text', text), createComment: text => node('#comment', text),
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
  const events = []
  const host = defineComponent({
    setup() {
      return () => h(QuickSelectOverlay, {
        locale: 'en', candidates, truncated: false, range: 'visible',
        onSelected: value => events.push(['selected', value.text]),
        onClose: () => events.push(['close']),
        'onUpdate:range': value => events.push(['range', value]),
      })
    },
  })
  const app = renderer.createApp(host)
  app.mount(root)
  const find = predicate => {
    const visit = target => {
      if (predicate(target)) return target
      for (const child of target.children) {
        const found = visit(child)
        if (found) return found
      }
    }
    return visit(root)
  }
  return { app, root, events, find, focusTarget: () => focusTarget }
}

const candidate = (label, text, visible = true) => ({
  label, text, visible, start: 0, end: text.length, type: 'url', name: 'url', priority: 0,
  bufferRow: 0, column: 0, left: 0, top: 0,
})

try {
  test('captures two-character labels without forwarding keys and supports escape', async () => {
    const view = mount([candidate('aa', 'first'), candidate('ab', 'second')])
    await nextTick()
    const host = view.find(item => item.props.role === 'group')
    assert.equal(view.focusTarget()?.props.role, 'group')
    for (const key of ['a', 'b']) host.props.onKeydown({
      key, isComposing: false, ctrlKey: false, metaKey: false, altKey: false,
      target: { closest: () => null }, preventDefault() {}, stopPropagation() {},
    })
    assert.deepEqual(view.events, [['selected', 'second']])
    host.props.onKeydown({ key: 'Escape', isComposing: false, target: { closest: () => null }, preventDefault() {}, stopPropagation() {} })
    assert.deepEqual(view.events.at(-1), ['close'])
    view.app.unmount()
  })

  test('shows off-screen candidates in a keyboard reachable list and changes scan range', async () => {
    const view = mount([candidate('a', '/home/sam/a', false)])
    await nextTick()
    const item = view.find(node => node.type === 'button' && node.parent?.props.class === 'quick-select-offscreen')
    assert.ok(item)
    item.props.onClick()
    assert.deepEqual(view.events.at(-1), ['selected', '/home/sam/a'])
    const select = view.find(node => node.type === 'select')
    select.props.onChange({ target: { value: 'buffer' } })
    assert.deepEqual(view.events.at(-1), ['range', 'buffer'])
    view.app.unmount()
  })

  test('Escape closes Quick Select when a candidate or range control has focus', async () => {
    const view = mount([candidate('a', '/home/sam/a', false)])
    await nextTick()
    const host = view.find(item => item.props.role === 'group')
    for (const tag of ['button', 'select']) {
      let prevented = false
      host.props.onKeydown({
        key: 'Escape', isComposing: false,
        target: { closest: selector => selector.split(',').includes(tag) ? {} : null },
        preventDefault() { prevented = true }, stopPropagation() {},
      })
      assert.equal(prevented, true)
    }
    host.props.onKeydown({
      key: 'Escape', isComposing: true, target: { closest: () => ({}) },
      preventDefault() { assert.fail('composing Escape must stay with the IME') }, stopPropagation() {},
    })
    assert.deepEqual(view.events, [['close'], ['close']])
    view.app.unmount()
  })
} finally {
  await rm(tempRoot, { recursive: true, force: true })
}

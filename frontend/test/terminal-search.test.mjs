import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { join, resolve } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { test } from 'node:test'
import ts from 'typescript'
import { compileScript, parse } from '@vue/compiler-sfc'
import { createRenderer, defineComponent, h, nextTick, ref } from 'vue'

const frontendRoot = resolve(fileURLToPath(new URL('..', import.meta.url)))
const tempRoot = await mkdtemp(join(frontendRoot, 'test', '.terminal-search-build-'))

function transpile(source, filename) {
  return ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
    fileName: filename,
  }).outputText
}

const sourcePath = resolve(frontendRoot, 'src/components/TerminalSearch.vue')
const { descriptor, errors } = parse(await readFile(sourcePath, 'utf8'), { filename: sourcePath })
assert.deepEqual(errors, [])
const javascript = transpile(compileScript(descriptor, { id: 'terminal-search', inlineTemplate: true }).content, `${sourcePath}.ts`)
const componentPath = join(tempRoot, 'TerminalSearch.mjs')
await writeFile(componentPath, javascript)
const { default: TerminalSearch } = await import(pathToFileURL(componentPath).href)

function makeRenderer() {
  let focused
  const makeNode = (type, text = '') => ({
    type, text, props: {}, children: [], parent: null,
    focus() { focused = this },
    select() { this.selected = true },
  })
  const renderer = createRenderer({
    createElement: type => makeNode(type),
    createText: text => makeNode('#text', text),
    createComment: text => makeNode('#comment', text),
    setText: (node, text) => { node.text = text },
    setElementText: (node, text) => { node.children = []; node.text = text },
    patchProp: (node, key, _previous, value) => { node.props[key] = value },
    insert(node, parent, anchor = null) {
      if (node.parent) {
        const oldIndex = node.parent.children.indexOf(node)
        if (oldIndex >= 0) node.parent.children.splice(oldIndex, 1)
      }
      node.parent = parent
      const index = anchor ? parent.children.indexOf(anchor) : -1
      if (index < 0) parent.children.push(node)
      else parent.children.splice(index, 0, node)
    },
    remove(node) {
      if (!node.parent) return
      const index = node.parent.children.indexOf(node)
      if (index >= 0) node.parent.children.splice(index, 1)
      node.parent = null
    },
    parentNode: node => node.parent,
    nextSibling(node) {
      if (!node.parent) return null
      return node.parent.children[node.parent.children.indexOf(node) + 1] ?? null
    },
  })
  return { renderer, focused: () => focused }
}

function findNode(root, predicate) {
  if (predicate(root)) return root
  for (const child of root.children) {
    const found = findNode(child, predicate)
    if (found) return found
  }
  return undefined
}

function renderedText(node) {
  return node.text + node.children.map(renderedText).join('')
}

function mountSearch() {
  const { renderer, focused } = makeRenderer()
  const root = { type: 'root', text: '', props: {}, children: [], parent: null }
  const query = ref('')
  const caseSensitive = ref(false)
  const resultIndex = ref(-1)
  const resultCount = ref(0)
  const actions = []
  const host = defineComponent({
    setup() {
      return () => h(TerminalSearch, {
        locale: 'en', query: query.value, caseSensitive: caseSensitive.value,
        resultIndex: resultIndex.value, resultCount: resultCount.value,
        'onUpdate:query': value => { query.value = value },
        'onUpdate:caseSensitive': value => { caseSensitive.value = value },
        onNext: () => actions.push('next'),
        onPrevious: () => actions.push('previous'),
        onClose: () => actions.push('close'),
      })
    },
  })
  const app = renderer.createApp(host)
  app.mount(root)
  return { app, root, query, caseSensitive, resultIndex, resultCount, actions, focused }
}

try {
  test('search input takes focus, stays bounded, and keeps navigation out of the shell', async () => {
    const view = mountSearch()
    await nextTick()
    const input = findNode(view.root, node => node.type === 'input')
    assert.ok(input)
    assert.equal(view.focused()?.type, input.type)
    assert.equal(view.focused()?.props['aria-label'], 'Search this terminal')
    assert.equal(input.props.maxlength, '256')
    input.props.onInput({ target: { value: 'DiggerTerm' } })
    await nextTick()
    assert.equal(view.query.value, 'DiggerTerm')

    const search = findNode(view.root, node => node.props.role === 'search')
    for (const [key, shiftKey, expected] of [['Enter', false, 'next'], ['Enter', true, 'previous'], ['Escape', false, 'close']]) {
      let prevented = false
      let stopped = false
      search.props.onKeydown({ key, shiftKey, isComposing: false, preventDefault() { prevented = true }, stopPropagation() { stopped = true } })
      assert.equal(prevented, true)
      assert.equal(stopped, true)
      assert.equal(view.actions.at(-1), expected)
    }
    const priorActions = view.actions.length
    search.props.onKeydown({ key: 'Enter', shiftKey: false, isComposing: true, preventDefault() {}, stopPropagation() {} })
    assert.equal(view.actions.length, priorActions)
    view.app.unmount()
  })

  test('case toggle, result feedback, and result buttons expose the current state', async () => {
    const view = mountSearch()
    await nextTick()
    const search = findNode(view.root, node => node.props.role === 'search')
    const caseButton = findNode(search, node => node.type === 'button' && node.props['aria-label'] === 'Match case')
    const nextButton = findNode(search, node => node.type === 'button' && node.props['aria-label'] === 'Next result')
    assert.ok(caseButton)
    assert.ok(nextButton)
    assert.equal(renderedText(search).includes('Type to search'), true)
    assert.equal(nextButton.props.disabled, true)

    caseButton.props.onClick()
    view.resultIndex.value = 0
    view.resultCount.value = 3
    view.query.value = 'term'
    await nextTick()
    assert.equal(view.caseSensitive.value, true)
    assert.equal(caseButton.props['aria-pressed'], true)
    assert.equal(renderedText(search).includes('1 / 3'), true)
    assert.equal(nextButton.props.disabled, false)
    nextButton.props.onClick()
    assert.equal(view.actions.at(-1), 'next')

    view.resultCount.value = 0
    await nextTick()
    assert.equal(renderedText(search).includes('No results'), true)
    assert.equal(nextButton.props.disabled, true)
    view.app.unmount()
  })
} finally {
  await rm(tempRoot, { recursive: true, force: true })
}

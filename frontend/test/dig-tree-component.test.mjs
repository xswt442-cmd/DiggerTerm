import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { join, resolve } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { test } from 'node:test'
import ts from 'typescript'
import { compileScript, parse } from '@vue/compiler-sfc'
import { createRenderer, h, nextTick, ref } from 'vue'

const frontendRoot = resolve(fileURLToPath(new URL('..', import.meta.url)))
const tempRoot = await mkdtemp(join(frontendRoot, 'test', '.dig-tree-build-'))

function transpile(source, filename) {
  return ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
    fileName: filename,
  }).outputText
}

async function compileVueComponent(filename, outputName, replacements = []) {
  const sourcePath = resolve(frontendRoot, 'src/components', filename)
  const source = await readFile(sourcePath, 'utf8')
  const { descriptor, errors } = parse(source, { filename: sourcePath })
  assert.deepEqual(errors, [])
  let javascript = transpile(compileScript(descriptor, { id: outputName, inlineTemplate: true }).content, `${sourcePath}.ts`)
  for (const [from, to] of replacements) javascript = javascript.replaceAll(from, to)
  const output = join(tempRoot, outputName)
  await writeFile(output, javascript)
  return pathToFileURL(output).href
}

try {
  const treeModulePath = await readFile(resolve(frontendRoot, 'src/dig/processTree.ts'), 'utf8')
  const treeModule = join(tempRoot, 'processTree.mjs')
  await writeFile(treeModule, transpile(treeModulePath, `${treeModule}.ts`))
  const treeURL = pathToFileURL(treeModule).href
  const nodeURL = await compileVueComponent('DigTreeNode.vue', 'DigTreeNode.mjs', [
    ["'../dig/processTree'", `'${treeURL}'`],
  ])
  const observeStub = join(tempRoot, 'observeStub.mjs')
  await writeFile(observeStub, 'let response; export function setResponse(value) { response = value }; export async function Observe() { return response }\n')
  const useDigSource = await readFile(resolve(frontendRoot, 'src/composables/useDig.ts'), 'utf8')
  const useDigModule = join(tempRoot, 'useDig.mjs')
  const useDigURL = pathToFileURL(useDigModule).href
  const observeStubURL = pathToFileURL(observeStub).href
  let useDigJavascript = transpile(useDigSource, `${useDigModule}.ts`)
    .replace("'../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/sessionservice'", `'${observeStubURL}'`)
    .replace("'../dig/processTree'", `'${treeURL}'`)
  await writeFile(useDigModule, useDigJavascript)
  const digTreeURL = await compileVueComponent('DigTree.vue', 'DigTree.mjs', [
    ["'./DigTreeNode.vue'", `'${nodeURL}'`],
  ])
  const { default: DigTree } = await import(digTreeURL)

  function makeHost() {
    const makeNode = (type, text = '') => ({ type, text, props: {}, children: [], parent: null })
    return {
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
    }
  }

  function mountTree(initialGroups) {
    const groups = ref(initialGroups)
    const selected = []
    const App = {
      setup: () => () => h(DigTree, {
        groups: groups.value,
        selectedKey: selected.at(-1) ?? null,
        showPid: true,
        locale: 'en',
        label: 'Dig observed processes',
        kindLabel: kind => kind,
        onSelect: process => selected.push(process.key),
      }),
    }
    const renderer = createRenderer(makeHost())
    const root = makeHost().createElement('root')
    renderer.createApp(App).mount(root)
    return { root, groups, selected }
  }

  function walk(node, visit) {
    visit(node)
    for (const child of node.children) walk(child, visit)
  }

  function findByClass(root, className, text = undefined) {
    const matches = []
    walk(root, node => {
      if ((node.props.class ?? '').split(' ').includes(className) &&
        (text === undefined || contentText(node).includes(text))) matches.push(node)
    })
    return matches
  }

  function contentText(node) {
    return node.text + node.children.map(contentText).join('')
  }

  async function click(node) {
    assert.notEqual(node.props.disabled, true)
    node.props.onClick?.({})
    await nextTick()
  }

  const process = (key, pid, parentPid, startedAt = `2026-01-01T00:00:0${pid % 10}Z`) => ({
    key, pid, parentPid, name: key, kind: 'descendant', source: 'windows', distribution: '', startedAt,
  })

  test('search expands a folded ancestor temporarily and restores the manual fold after clearing', async () => {
    const { buildProcessTree, filterProcessTree } = await import(treeURL)
    const full = buildProcessTree([process('shell', 10, 0), process('agent', 11, 10), process('worker', 12, 11)])
    const mounted = mountTree(full)
    await click(findByClass(mounted.root, 'dig-toggle')[0])
    assert.equal(findByClass(mounted.root, 'dig-select', 'worker').length, 0)

    mounted.groups.value = filterProcessTree(full, row => row.key === 'worker')
      .map(group => ({ ...group, expandFilteredNodes: true }))
    await nextTick()
    assert.equal(findByClass(mounted.root, 'dig-select', 'worker').length, 1)
    assert.equal(findByClass(mounted.root, 'dig-toggle')[0].props.disabled, true)
    assert.equal(findByClass(mounted.root, 'dig-toggle')[0].props['aria-expanded'], true)

    mounted.groups.value = full
    await nextTick()
    assert.equal(findByClass(mounted.root, 'dig-select', 'worker').length, 0)
    await click(findByClass(mounted.root, 'dig-toggle')[0])
    assert.equal(findByClass(mounted.root, 'dig-select', 'worker').length, 1)
  })

  test('refresh with stable keys preserves fold preference; a changed process key gets fresh state', async () => {
    const { buildProcessTree } = await import(treeURL)
    const initial = buildProcessTree([process('shell', 20, 0), process('child', 21, 20)])
    const mounted = mountTree(initial)
    await click(findByClass(mounted.root, 'dig-toggle')[0])

    mounted.groups.value = buildProcessTree([process('shell', 20, 0), process('child', 21, 20, '2026-01-01T00:00:09Z')])
    await nextTick()
    assert.equal(findByClass(mounted.root, 'dig-select', 'child').length, 0)

    mounted.groups.value = buildProcessTree([process('shell-new-identity', 20, 0), process('child-new-identity', 21, 20)])
    await nextTick()
    assert.equal(findByClass(mounted.root, 'dig-select', 'child-new-identity').length, 1)
  })

  test('tree uses labeled navigation, native keyboard buttons, visible focus styling, and reports selection', async () => {
    const { buildProcessTree } = await import(treeURL)
    const mounted = mountTree(buildProcessTree([process('shell', 30, 0), process('agent', 31, 30)]))
    const treeNav = []
    walk(mounted.root, node => { if (node.type === 'nav') treeNav.push(node) })
    assert.equal(treeNav[0].props['aria-label'], 'Dig observed processes')
    assert.equal(treeNav[0].props.role, undefined)
    const selects = findByClass(mounted.root, 'dig-select')
    assert.ok(selects.every(button => button.type === 'button' && button.props.type === 'button'))
    await click(selects[1])
    assert.deepEqual(mounted.selected, ['agent'])
    assert.match(await readFile(resolve(frontendRoot, 'src/components/DigTreeNode.vue'), 'utf8'), /button:focus-visible/)
  })

  test('useDig marks active text and category filters for temporary ancestor expansion', async () => {
    const { useDig } = await import(useDigURL)
    const observeModule = await import(observeStubURL)
    const query = ref('')
    const selectedExternalKey = ref(null)
    const dig = useDig({ observeEnabled: ref(true), query, selectedExternalKey, observeFailedLabel: () => 'Observe failed' })
    observeModule.setResponse({
      supported: true,
      processes: [
        { ...process('shell', 40, 0), source: 'windows' },
        { ...process('agent', 41, 40), source: 'windows' },
      ],
    })
    await dig.refresh()
    assert.equal(dig.filteredGroups.value[0].expandFilteredNodes, false)
    query.value = 'agent'
    await nextTick()
    assert.equal(dig.filteredGroups.value[0].expandFilteredNodes, true)
    query.value = ''
    await nextTick()
    assert.equal(dig.filteredGroups.value[0].expandFilteredNodes, false)
    dig.kindFilter.value = 'descendant'
    await nextTick()
    assert.equal(dig.filteredGroups.value[0].expandFilteredNodes, true)
  })

  test('useDig keeps a vanished selection stale and does not replace the last good snapshot on failure', async () => {
    const { useDig } = await import(useDigURL)
    const observeModule = await import(observeStubURL)
    const selectedExternalKey = ref(null)
    const dig = useDig({
      observeEnabled: ref(true), query: ref(''), selectedExternalKey,
      observeFailedLabel: () => 'Observe failed',
    })
    observeModule.setResponse({ supported: true, processes: [{
      key: 'external-55', pid: 55, parentPid: 1, name: 'powershell.exe', kind: 'terminal',
      source: 'windows', distribution: '', startedAt: '2026-01-01T00:00:00Z',
      classificationBasis: 'name',
    }] })
    await dig.refresh()
    selectedExternalKey.value = 'external-55'
    assert.equal(dig.activeProcess.value?.pid, 55)

    observeModule.setResponse({ supported: true, processes: [] })
    await dig.refresh()
    assert.equal(dig.activeProcess.value, undefined)
    assert.equal(selectedExternalKey.value, 'external-55')
    assert.equal(dig.snapshot.value?.processes.length, 0)

    observeModule.setResponse(Promise.reject(new Error('scan denied')))
    await dig.refresh()
    assert.match(dig.error.value, /Observe failed: Error: scan denied/)
    assert.equal(dig.snapshot.value?.processes.length, 0)
    dig.clearSelection()
    assert.equal(selectedExternalKey.value, null)
  })
} finally {
  await rm(tempRoot, { recursive: true, force: true })
}

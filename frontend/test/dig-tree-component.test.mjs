import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { join, resolve } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { after, test } from 'node:test'
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

{
  const treeModulePath = await readFile(resolve(frontendRoot, 'src/dig/processTree.ts'), 'utf8')
  const treeModule = join(tempRoot, 'processTree.mjs')
  await writeFile(treeModule, transpile(treeModulePath, `${treeModule}.ts`))
  const treeURL = pathToFileURL(treeModule).href
  const watchStateSource = await readFile(resolve(frontendRoot, 'src/dig/watchState.ts'), 'utf8')
  const watchStateModule = join(tempRoot, 'watchState.mjs')
  await writeFile(watchStateModule, transpile(watchStateSource, `${watchStateModule}.ts`))
  const watchStateURL = pathToFileURL(watchStateModule).href
  const nodeURL = await compileVueComponent('DigTreeNode.vue', 'DigTreeNode.mjs', [
    ["'../dig/processTree'", `'${treeURL}'`],
  ])
  const observeStub = join(tempRoot, 'observeStub.mjs')
  await writeFile(observeStub, 'let response; let capabilities = { observeSourceResults: false }; export function setResponse(value) { response = value }; export function setCapabilities(value) { capabilities = value }; export async function Observe() { return response }; export async function ObserveCapabilities() { return capabilities }\n')
  const useDigSource = await readFile(resolve(frontendRoot, 'src/composables/useDig.ts'), 'utf8')
  const useDigModule = join(tempRoot, 'useDig.mjs')
  const useDigURL = pathToFileURL(useDigModule).href
  const observeStubURL = pathToFileURL(observeStub).href
  let useDigJavascript = transpile(useDigSource, `${useDigModule}.ts`)
    .replace("'../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/sessionservice'", `'${observeStubURL}'`)
    .replace("'../dig/processTree'", `'${treeURL}'`)
    .replace("'../dig/watchState'", `'${watchStateURL}'`)
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
    const expansionPreferences = ref({})
    const selected = []
    const App = {
      setup: () => () => h(DigTree, {
        groups: groups.value,
        selectedKey: selected.at(-1) ?? null,
        showPid: true,
        locale: 'en',
        label: 'Dig observed processes',
        kindLabel: kind => kind,
        expansionPreferences: expansionPreferences.value,
        onSelect: process => selected.push(process.key),
        onToggle: (key, expanded) => { expansionPreferences.value[key] = expanded },
      }),
    }
    const renderer = createRenderer(makeHost())
    const root = makeHost().createElement('root')
    renderer.createApp(App).mount(root)
    return { root, groups, selected, expansionPreferences }
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

  test('manual fold survives a cross-branch search that unmounts and restores the node', async () => {
    const { buildProcessTree, filterProcessTree } = await import(treeURL)
    const full = buildProcessTree([
      process('root-a', 50, 0), process('branch-a', 51, 50), process('match-a', 52, 51),
      process('root-b', 60, 0), process('branch-b', 61, 60), process('match-b', 62, 61),
    ])
    const mounted = mountTree(full)
    const toggleFor = name => findByClass(mounted.root, 'dig-row')
      .find(row => contentText(row).includes(name))?.children.find(child => child.props.class === 'dig-toggle')
    await click(toggleFor('branch-b'))
    assert.equal(findByClass(mounted.root, 'dig-select', 'match-b').length, 0)

    mounted.groups.value = filterProcessTree(full, row => row.key === 'match-a')
      .map(group => ({ ...group, expandFilteredNodes: true }))
    await nextTick()
    assert.equal(findByClass(mounted.root, 'dig-select', 'branch-b').length, 0)

    mounted.groups.value = filterProcessTree(full, row => row.key === 'match-b')
      .map(group => ({ ...group, expandFilteredNodes: true }))
    await nextTick()
    assert.equal(findByClass(mounted.root, 'dig-select', 'match-b').length, 1)

    mounted.groups.value = full
    await nextTick()
    assert.equal(findByClass(mounted.root, 'dig-select', 'match-b').length, 0)
    assert.equal(mounted.expansionPreferences.value['branch-b'], false)
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
    dig.setExpanded('shell', false)
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

    observeModule.setResponse({
      supported: true,
      processes: [{ ...process('replacement-shell', 40, 0), source: 'windows' }],
    })
    await dig.refresh()
    assert.equal(dig.expansionPreferences.value.shell, undefined)
    assert.equal(dig.expansionPreferences.value['replacement-shell'], undefined)
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

  test('useDig baselines source-aware watch data and retains it through a partial source failure', async () => {
    const { useDig } = await import(useDigURL)
    const observeModule = await import(observeStubURL)
    observeModule.setCapabilities({ observeSourceResults: true })
    const dig = useDig({
      observeEnabled: ref(true), connectionRevision: ref(1), query: ref(''),
      selectedExternalKey: ref(null), observeFailedLabel: () => 'Observe failed',
    })
    const status = { source: 'windows', distribution: '', namespace: 'windows:boot', scannedAt: '2026-01-01T00:00:00Z', state: 'success', complete: true, truncated: false, policy: 'policy-v1', reason: '' }
    const first = { key: 'one', pid: 1, parentPid: 0, name: 'pwsh.exe', kind: 'terminal', source: 'windows', distribution: '', startedAt: '2026-01-01T00:00:00Z' }
    observeModule.setResponse({ supported: true, processes: [first], sources: [status], truncated: false, scannedAt: status.scannedAt, note: '' })
    await dig.refresh()
    assert.equal(dig.watchSupported.value, true)
    assert.equal(dig.watchChanges.value.length, 0)

    const newer = { ...first, key: 'two', pid: 2, name: 'codex.exe', kind: 'agent' }
    observeModule.setResponse({ supported: true, processes: [first, newer], sources: [status], truncated: false, scannedAt: status.scannedAt, note: '' })
    await dig.refresh()
    assert.deepEqual(dig.watchChanges.value.map(change => change.key), ['two'])

    observeModule.setResponse({ supported: true, processes: [], sources: [{ ...status, state: 'failed', complete: false, reason: 'enumeration-failed' }], truncated: false, scannedAt: status.scannedAt, note: '' })
    await dig.refresh()
    assert.deepEqual(dig.processes.value.map(item => item.key).sort(), ['one', 'two'])
    assert.equal(dig.watchSources.value[0].stale, true)
    assert.deepEqual(dig.watchChanges.value.map(change => change.key), ['two'])
  })

  test('useDig ignores a scan result that returns after observation is disabled', async () => {
    const { useDig } = await import(useDigURL)
    const observeModule = await import(observeStubURL)
    const observeEnabled = ref(true)
    const dig = useDig({ observeEnabled, connectionRevision: ref(1), query: ref(''),
      selectedExternalKey: ref(null), observeFailedLabel: () => 'Observe failed' })
    let resolveScan
    observeModule.setResponse(new Promise(resolve => { resolveScan = resolve }))
    const pending = dig.refresh()
    await Promise.resolve()
    await Promise.resolve()
    observeEnabled.value = false
    await nextTick()
    resolveScan({ supported: true, processes: [], sources: [], truncated: false, scannedAt: '2026-01-01T00:00:00Z', note: '' })
    await pending
    assert.equal(dig.snapshot.value, null)
    assert.equal(dig.loading.value, false)
  })
}
after(async () => { await rm(tempRoot, { recursive: true, force: true }) })

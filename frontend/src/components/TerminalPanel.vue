<script setup lang="ts">
import { defineAsyncComponent, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import type { ISearchOptions, SearchAddon as SearchAddonInstance } from '@xterm/addon-search'
import { Browser } from '@wailsio/runtime'
import { Events } from '@wailsio/runtime'
import { ReadText, WriteText } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/clipboardservice'
import { Resize, Snapshot, Write } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/sessionservice'
import { QuickSelectConfig } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/quickselectservice'
import type { Info } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/session/models'
import { fonts, type Palette, type FontId, type CursorStyle, type KeyboardMode, type Locale } from '../uiPreferences'
import { decideTerminalShortcut } from '../terminalShortcuts'
import { terminalColors } from '../userThemes'
import { matchBinding, type Binding } from '../keybindings'
import { queueSessionOperation, updateAttachment } from '../terminalAttachment'
import { sliceOutput } from '../terminalReplay'
import { validateExternalWebURL } from '../externalLinks'
import { withTerminalSearchSupport } from '../terminalSearch'
import type { PositionedQuickSelectCandidate } from '../quickSelect'
import TerminalSearch from './TerminalSearch.vue'
const QuickSelectOverlay = defineAsyncComponent(() => import('./QuickSelectOverlay.vue'))
import '@xterm/xterm/css/xterm.css'

const props = defineProps<{
  session: Info
  active: boolean
  focused: boolean
  connected: boolean
  fontSize: number
  fontFamily: FontId
  lineHeight: number
  scrollback: number
  cursorStyle: CursorStyle
  keyboardMode: KeyboardMode
  cursorBlink: boolean
  scrollSensitivity: number
  rightClickSelectsWord: boolean
  padding: number
  colors: Palette['terminal']
  themeRevision: number
  locale: Locale
  bindings: Binding[]
  toolsEnabled: boolean
  toolRequest: { sequence: number; action: 'search' | 'quickSelect' } | null
}>()
const emit = defineEmits<{ error: [message: string] }>()
const container = ref<HTMLElement | null>(null)
const copyMode = ref(false)
const copyText = ref('')
const copyEditor = ref<HTMLTextAreaElement | null>(null)
const notice = ref('')
const controlled = ref(false)
const attachmentError = ref('')
const replayError = ref('')
const replayTruncated = ref(false)
const recovering = ref(true)
const copyMenu = ref<{ x: number; y: number } | null>(null)
const copyMenuElement = ref<HTMLElement | null>(null)
const copyMenuURL = ref('')
const selectedWebURL = ref('')
const searchOpen = ref(false)
const searchQuery = ref('')
const searchCaseSensitive = ref(false)
const searchResultIndex = ref(-1)
const searchResultCount = ref(0)
const quickSelectOpen = ref(false)
const quickSelectLoading = ref(false)
const quickSelectRange = ref<'visible' | 'buffer'>('visible')
const quickSelectTruncated = ref(false)
const quickSelectCandidates = ref<Array<PositionedQuickSelectCandidate & { left: number; top: number; visible: boolean }>>([])
const text = (zh: string, en: string) => props.locale === 'zh' ? zh : en

function openCopyMenu(event: MouseEvent) {
  if (!controlled.value || recovering.value) return
  copyMenuURL.value = ''
  const selection = terminal?.getSelection().trim() ?? ''
  try { copyMenuURL.value = validateExternalWebURL(selection).href }
  catch { /* Only a selected absolute HTTP/HTTPS URL adds the open action. */ }
  copyMenu.value = {
    x: Math.max(0, Math.min(event.clientX, window.innerWidth - 210)),
    y: Math.max(0, Math.min(event.clientY, window.innerHeight - 120)),
  }
}

function dismissCopyMenu(event: PointerEvent) {
  if (!copyMenuElement.value?.contains(event.target as Node)) copyMenu.value = null
}

function runCopyMenu(action: 'copy' | 'paste' | 'mode' | 'openLink') {
  const selectedURL = copyMenuURL.value
  copyMenu.value = null
  copyMenuURL.value = ''
  if (action === 'copy') void copySelection()
  else if (action === 'paste') void pasteClipboard()
  else if (action === 'mode') void toggleCopyMode()
  else void confirmAndOpenWebLink(selectedURL)
}

interface OutputEvent {
  id: string
  data: string
  offset: number
}

let terminal: Terminal | undefined
let fitAddon: FitAddon | undefined
let searchAddon: SearchAddonInstance | undefined
let offSearchResults: { dispose(): void } | undefined
let offQuickSelectScroll: { dispose(): void } | undefined
let quickSelectAbort: AbortController | undefined
let quickSelectRequest = 0
let observer: ResizeObserver | undefined
let offOutput: (() => void) | undefined
let offInput: { dispose(): void } | undefined
let offSelection: { dispose(): void } | undefined
let pending: OutputEvent[] = []
let replaying = true
let lastOffset = 0
let writeQueue: Promise<void> = Promise.resolve()
let lastSize = ''

function decodeBase64(data: string): Uint8Array {
  const raw = atob(data)
  const result = new Uint8Array(raw.length)
  for (let i = 0; i < raw.length; i++) result[i] = raw.charCodeAt(i)
  return result
}

function writeEvent(event: OutputEvent) {
  if (!terminal || replayError.value || event.id !== props.session.id) return
  try {
    const bytes = sliceOutput(lastOffset, event.offset, decodeBase64(event.data))
    if (!bytes.length) return
    if (quickSelectOpen.value) void closeQuickSelect()
    terminal.write(bytes)
    lastOffset = event.offset
  } catch (error) { replayError.value = String(error) }
}

function fit() {
  if (!props.active || !props.connected || !controlled.value || !terminal || !fitAddon || !container.value || container.value.clientWidth === 0 || container.value.clientHeight === 0) return
  try {
    const size = fitAddon.proposeDimensions()
    if (!size || size.cols < 1 || size.rows < 1) return
    const cols = Math.min(500, size.cols)
    const rows = Math.min(500, size.rows)
    if (terminal.cols !== cols || terminal.rows !== rows) terminal.resize(cols, rows)
    const key = `${cols}:${rows}`
    if (quickSelectOpen.value && key !== lastSize) void closeQuickSelect()
    if (key !== lastSize && props.session.status === 'running') {
      lastSize = key
      const term = terminal
      void queueSessionOperation(props.session.id, async () => {
        if (terminal === term && controlled.value && props.connected && props.active && key === lastSize) await Resize(props.session.id, cols, rows)
      }).catch((err: unknown) => {
        lastSize = ''
        emit('error', String(err))
      })
    }
  } catch (error) {
    emit('error', String(error))
  }
}

function focus() {
  if (props.active && props.focused && !copyMode.value) terminal?.focus()
}

function optionsForSearch(incremental: boolean): ISearchOptions {
  return {
    caseSensitive: searchCaseSensitive.value,
    incremental,
    decorations: {
      matchBackground: props.colors.selectionBackground,
      matchBorder: props.colors.foreground,
      matchOverviewRuler: props.colors.foreground,
      activeMatchBackground: props.colors.cursor,
      activeMatchBorder: props.colors.cursor,
      activeMatchColorOverviewRuler: props.colors.cursor,
    },
  }
}

function searchNext() {
  if (searchQuery.value && searchAddon) searchAddon.findNext(searchQuery.value, optionsForSearch(false))
}

async function scanQuickSelect() {
  const term = terminal
  if (!term || !quickSelectOpen.value) return
  quickSelectAbort?.abort()
  const abort = new AbortController()
  quickSelectAbort = abort
  const request = ++quickSelectRequest
  quickSelectLoading.value = true
  quickSelectCandidates.value = []
  quickSelectTruncated.value = false
  notice.value = ''
  try {
    const config = await QuickSelectConfig()
    if (request !== quickSelectRequest || terminal !== term || abort.signal.aborted) return
    const { collectQuickSelectText, positionQuickSelectCandidates } = await import('../quickSelect')
    const snapshot = await collectQuickSelectText(term, quickSelectRange.value, abort.signal)
    const { scanQuickSelectInWorker } = await import('../quickSelectWorkerClient')
    const scan = await scanQuickSelectInWorker(snapshot.text, config, abort.signal)
    if (request !== quickSelectRequest || terminal !== term || abort.signal.aborted) return
    quickSelectTruncated.value = snapshot.truncated || scan.truncated
    if (scan.errors.length) notice.value = text('部分自定义规则无效：', 'Some custom rules are invalid: ') + scan.errors[0]
    await nextTick()
    const screen = container.value?.querySelector('.xterm-screen') as HTMLElement | null
    const surface = container.value?.parentElement
    const screenRect = screen?.getBoundingClientRect()
    const surfaceRect = surface?.getBoundingClientRect()
    if (!screenRect || !surfaceRect || term.cols < 1 || term.rows < 1) {
      quickSelectCandidates.value = []
      return
    }
    const cellWidth = screenRect.width / term.cols
    const cellHeight = screenRect.height / term.rows
    quickSelectCandidates.value = positionQuickSelectCandidates(scan, snapshot).map(candidate => {
      const row = candidate.bufferRow - snapshot.viewportY
      const visible = row >= 0 && row < term.rows
      return {
        ...candidate,
        left: screenRect.left - surfaceRect.left + candidate.column * cellWidth,
        top: screenRect.top - surfaceRect.top + row * cellHeight,
        visible,
      }
    })
    offQuickSelectScroll?.dispose()
    offQuickSelectScroll = term.onScroll(() => { if (quickSelectOpen.value) void closeQuickSelect() })
    if (!quickSelectCandidates.value.length && !scan.errors.length) notice.value = text('当前范围没有可选文字。', 'No selectable text in this range.')
  } catch (error) {
    if (request !== quickSelectRequest || abort.signal.aborted) return
    notice.value = String(error).includes('exceeded')
      ? text('扫描超时；缩小范围或停用较慢的规则。', 'Scan timed out; reduce the range or disable a slow rule.')
      : text('快速选择失败：', 'Quick Select failed: ') + String(error)
  } finally {
    if (request === quickSelectRequest) quickSelectLoading.value = false
  }
}

async function openQuickSelect() {
  if (quickSelectOpen.value || !props.focused || !controlled.value || recovering.value || !terminal) return
  if (searchOpen.value) await closeSearch()
  if (copyMode.value) await toggleCopyMode()
  quickSelectOpen.value = true
  void scanQuickSelect()
}

function toggleQuickSelect() {
  if (quickSelectOpen.value) void closeQuickSelect()
  else void openQuickSelect()
}

async function closeQuickSelect() {
  quickSelectRequest++
  quickSelectAbort?.abort()
  quickSelectAbort = undefined
  offQuickSelectScroll?.dispose()
  offQuickSelectScroll = undefined
  quickSelectOpen.value = false
  quickSelectLoading.value = false
  quickSelectCandidates.value = []
  await nextTick()
  focus()
}

async function selectQuickCandidate(candidate: PositionedQuickSelectCandidate) {
  try {
    await writeClipboard(candidate.text)
    notice.value = text('已复制：', 'Copied: ') + candidate.text
    await closeQuickSelect()
  } catch (error) {
    notice.value = text('复制失败，请重试：', 'Copy failed; retry: ') + String(error)
  }
}

function searchPrevious() {
  if (searchQuery.value && searchAddon) searchAddon.findPrevious(searchQuery.value, optionsForSearch(false))
}

async function confirmAndOpenWebLink(value: string) {
  let url: URL
  try { url = validateExternalWebURL(value) }
  catch (error) {
    notice.value = text('链接未打开：', 'Link not opened: ') + String(error)
    return
  }
  const destination = `${text('主机', 'Host')}: ${url.hostname}\n\n${url.href}`
  if (!window.confirm(text(`将用系统浏览器打开此地址？\n\n${destination}`, `Open this address in the system browser?\n\n${destination}`))) return
  try {
    await Browser.OpenURL(url.href)
    notice.value = text('已交给系统浏览器。', 'Opened in the system browser.')
  } catch (error) {
    notice.value = text('无法打开链接：', 'Could not open link: ') + String(error)
  }
}

async function openTerminalLink(event: MouseEvent, value: string) {
  if (isMac ? !event.metaKey : !event.ctrlKey) return
  event.preventDefault()
  await confirmAndOpenWebLink(value)
}

function updateSelectedWebURL() {
  selectedWebURL.value = ''
  const selected = terminal?.getSelection().trim() ?? ''
  try { selectedWebURL.value = validateExternalWebURL(selected).href }
  catch { /* Only a selected absolute HTTP/HTTPS URL enables the action. */ }
}

function openSelectedWebLink() {
  const value = selectedWebURL.value
  if (value) void confirmAndOpenWebLink(value)
}

function onPanelPaste(event: ClipboardEvent) {
  if ((event.target as HTMLElement).closest('.terminal-search, .quick-select-layer')) return
  if (!props.focused || !props.connected || props.session.streamClosed || props.session.status !== 'running' || copyMode.value || recovering.value || replayError.value) {
    event.preventDefault()
    event.stopPropagation()
  }
}

async function openSearch() {
  if (searchOpen.value) return
  if (quickSelectOpen.value) await closeQuickSelect()
  if (copyMode.value) void toggleCopyMode()
  searchOpen.value = true
  const term = terminal
  if (!term || searchAddon) return
  try {
    const { SearchAddon } = await import('@xterm/addon-search')
    if (terminal !== term || !searchOpen.value) return
    const addon = new SearchAddon({ highlightLimit: 1000 })
    term.loadAddon(addon)
    searchAddon = addon
    offSearchResults = addon.onDidChangeResults(({ resultIndex, resultCount }) => {
      searchResultIndex.value = resultIndex
      searchResultCount.value = resultCount
    })
    if (searchQuery.value) addon.findNext(searchQuery.value, optionsForSearch(true))
  } catch (error) {
    if (terminal !== term) return
    searchOpen.value = false
    notice.value = text('无法加载搜索：', 'Could not load search: ') + String(error)
    await nextTick()
    focus()
  }
}

function toggleSearch() {
  if (searchOpen.value) void closeSearch()
  else void openSearch()
}

async function closeSearch() {
  searchOpen.value = false
  searchQuery.value = ''
  searchResultIndex.value = -1
  searchResultCount.value = 0
  searchAddon?.clearDecorations()
  await nextTick()
  focus()
}

watch(quickSelectRange, () => { if (quickSelectOpen.value) void scanQuickSelect() })

watch([searchQuery, searchCaseSensitive], ([query]) => {
  if (!searchOpen.value || !searchAddon) return
  if (!query) {
    searchAddon.clearDecorations()
    searchResultIndex.value = -1
    searchResultCount.value = 0
    return
  }
  searchAddon.findNext(query, optionsForSearch(true))
})

async function acquireControl() {
  const term = terminal
  if (!props.connected) {
    attachmentError.value = text('Daemon 连接不可用', 'Daemon unavailable')
    return
  }
  try {
    await updateAttachment(props.session.id, true)
    if (terminal !== term) return
    controlled.value = true
    attachmentError.value = ''
    fit()
  } catch (err) {
    if (terminal === term) {
      controlled.value = false
      attachmentError.value = String(err).includes('unsupported daemon method')
        ? text('Daemon 需要更新。请完成现有会话后手动停止旧 Daemon，再重新打开。', 'Daemon update required. Finish sessions and explicitly stop the old daemon, then reopen.')
        : String(err)
    }
  }
}

async function replaySnapshot(term: Terminal) {
  recovering.value = true
  term.options.disableStdin = true
  replayError.value = ''
  replaying = true
  pending = []
  try {
    const snapshot = await Snapshot(props.session.id)
    if (terminal !== term) return
    const raw = decodeBase64(snapshot.data)
    sliceOutput(snapshot.offset - raw.length, snapshot.offset, raw)
    replayTruncated.value = snapshot.truncated || snapshot.offset > raw.length
    if (raw.length) term.write(raw, () => {
      if (terminal !== term) return
      term.options.disableStdin = false
      recovering.value = false
    })
    else {
      term.options.disableStdin = false
      recovering.value = false
    }
    lastOffset = snapshot.offset
  } catch (err) {
    replayError.value = String(err)
    term.options.disableStdin = false
    recovering.value = false
  } finally {
    pending.sort((a, b) => a.offset - b.offset)
    for (const event of pending) writeEvent(event)
    pending = []
    replaying = false
  }
}

async function reconnectView() {
  const term = terminal
  if (!term || !props.connected) return
  recovering.value = true
  copyMode.value = false
  copyText.value = ''
  await acquireControl()
  if (controlled.value && terminal === term) {
    term.reset()
    lastOffset = 0
    await replaySnapshot(term)
    focus()
  }
  if (!controlled.value) recovering.value = false
}

async function copySelection() {
  const editor = copyEditor.value
  const selected = copyMode.value && editor
    ? editor.value.slice(editor.selectionStart, editor.selectionEnd)
    : terminal?.getSelection()
  if (!selected) { notice.value = text('请先选择文字；拖选无效时使用复制模式。', 'Select text first; use Copy mode if dragging does not select.'); return }
  try {
    await writeClipboard(selected)
    notice.value = text('已复制', 'Copied')
  } catch (err) { notice.value = text('复制失败，请重试：', 'Copy failed; retry: ') + String(err) }
}

async function pasteClipboard() {
  const term = terminal
  if (!props.active || !props.focused || !props.connected || props.session.streamClosed || recovering.value || replayError.value || !controlled.value || !term || copyMode.value || props.session.status !== 'running') return
  try {
    const value = await readClipboard()
    if (terminal === term && props.active && props.focused && props.connected && !props.session.streamClosed && !recovering.value && !replayError.value && controlled.value && !copyMode.value && props.session.status === 'running') term.paste(value)
  } catch (err) { notice.value = text('粘贴失败，请重试：', 'Paste failed; retry: ') + String(err) }
}

async function toggleCopyMode() {
  if (!copyMode.value && searchOpen.value) await closeSearch()
  if (!copyMode.value && quickSelectOpen.value) await closeQuickSelect()
  copyMode.value = !copyMode.value
  notice.value = ''
  if (copyMode.value && terminal) {
    const buffer = terminal.buffer.active
    terminal.selectLines(buffer.viewportY, buffer.viewportY + terminal.rows - 1)
    copyText.value = terminal.getSelection()
    terminal.clearSelection()
    await nextTick()
    copyEditor.value?.focus()
  } else {
    copyText.value = ''
    await nextTick()
    focus()
  }
}

function onCopyEditorKey(event: KeyboardEvent) {
  event.stopPropagation()
  if (event.isComposing || event.altKey || event.getModifierState('AltGraph')) return
  const copy = matchBinding(event, props.bindings, 'terminal')
  if (event.key === 'Escape' || copy === 'toggleCopyMode') {
    event.preventDefault(); void toggleCopyMode()
  } else if (copy === 'copySelection' || (event.key.toLowerCase() === 'c' && (isMac ? event.metaKey : event.ctrlKey))) {
    event.preventDefault(); void copySelection()
  }
}

// Wails supplies a native clipboard bridge, which is more dependable than
// navigator.clipboard in desktop WebViews (particularly under Linux/WSLg).
async function writeClipboard(text: string) {
  try {
    await WriteText(text)
  } catch (nativeError) {
    if (!navigator.clipboard?.writeText) throw nativeError
    await navigator.clipboard.writeText(text)
  }
}

async function readClipboard(): Promise<string> {
  try {
    return await ReadText()
  } catch (nativeError) {
    if (!navigator.clipboard?.readText) throw nativeError
    return navigator.clipboard.readText()
  }
}

const isMac = typeof navigator !== 'undefined' &&
  /Mac/i.test(navigator.userAgent)

function onTerminalKeydown(event: KeyboardEvent) {
  // Do not steal shortcuts from the sidebar, search or settings. The
  // listener is attached only to the active terminal's DOM subtree.
  const term = terminal
  if (!props.active || !props.focused || !term) return
  if (copyMenu.value && event.key === 'Escape') {
    copyMenu.value = null
    event.preventDefault(); event.stopPropagation()
    return
  }
  const target = event.target as HTMLElement
  if (!target.classList.contains('xterm-helper-textarea') && target.closest('input, select, textarea')) return
  const binding = matchBinding(event, props.bindings, 'terminal')
  if (binding) {
    event.preventDefault(); event.stopPropagation()
    if (binding === 'copySelection') void copySelection()
    else if (binding === 'pasteClipboard') void pasteClipboard()
    else if (binding === 'toggleCopyMode') void toggleCopyMode()
    else if (binding === 'terminalSearch') toggleSearch()
    else if (binding === 'quickSelect') toggleQuickSelect()
    return
  }
  if (copyMode.value) return
  if (target.closest('button')) return
  const shortcut = decideTerminalShortcut(
    event, term.hasSelection(), isMac, props.keyboardMode, term.buffer.active.type === 'alternate', false,
  )
  if (shortcut.kind === 'pass') return

  // Capture ahead of xterm and the WebView's default Copy/Paste/Reload,
  // and do not allow xterm to send the same control byte twice.
  event.preventDefault()
  event.stopPropagation()

  if (shortcut.kind === 'input') {
    if (props.connected && !props.session.streamClosed && !recovering.value && !replayError.value && controlled.value && props.session.status === 'running') {
      term.input(shortcut.data, true) // goes through onData -> ordered daemon IPC
    }
    return
  }
  if (shortcut.kind === 'copy') {
    void copySelection()
    return
  }
  if (shortcut.kind === 'paste' && props.session.status === 'running') {
    void pasteClipboard()
  }
}

watch(() => props.fontSize, async (size) => {
  if (terminal) {
    terminal.options.fontSize = size
    await nextTick()
    fit()
  }
})
function updateColors() {
  if (terminal && container.value) terminal.options.theme = terminalColors(container.value, props.colors)
  if (searchOpen.value && searchQuery.value && searchAddon) {
    searchAddon.findNext(searchQuery.value, optionsForSearch(true))
  }
}
watch([() => props.colors, () => props.themeRevision], updateColors, { deep: true })

watch(() => props.fontFamily, async (font) => {
  if (!terminal) return
  terminal.options.fontFamily = fonts[font].family
  await nextTick()
  fit()
})
watch(() => props.lineHeight, async (height) => {
  if (!terminal) return
  terminal.options.lineHeight = height
  await nextTick()
  fit()
})
watch(() => props.scrollback, (lines) => {
  if (terminal) terminal.options.scrollback = lines
})
watch(() => props.cursorStyle, (style) => {
  if (terminal) terminal.options.cursorStyle = style
})
watch(() => props.cursorBlink, (blink) => {
  if (terminal) terminal.options.cursorBlink = blink
})
watch(() => props.scrollSensitivity, (sensitivity) => {
  if (terminal) terminal.options.scrollSensitivity = sensitivity
})
watch(() => props.rightClickSelectsWord, (enabled) => {
  if (terminal) terminal.options.rightClickSelectsWord = enabled
})
watch(() => props.padding, async () => {
  await nextTick()
  fit()
})

watch(() => props.toolsEnabled, enabled => {
  if (enabled) return
  if (searchOpen.value) void closeSearch()
  if (quickSelectOpen.value) void closeQuickSelect()
  if (copyMode.value) void toggleCopyMode()
})
watch(() => props.toolRequest, request => {
  if (!request || !props.focused || !props.toolsEnabled) return
  if (request.action === 'search') void openSearch()
  else void openQuickSelect()
})

watch(() => props.active, async (active) => {
  if (active) {
    await nextTick()
    fit()
    focus()
  } else copyMenu.value = null
})
watch(() => props.focused, async (focused) => {
  if (focused) {
    await nextTick()
    if (!document.activeElement?.matches('select,input,textarea:not(.xterm-helper-textarea)')) focus()
  } else {
    copyMenu.value = null
    terminal?.blur()
  }
})

onMounted(async () => {
  document.addEventListener('pointerdown', dismissCopyMenu)
  if (!container.value) return
  const term = new Terminal(withTerminalSearchSupport({
    cursorBlink: props.cursorBlink,
    cursorStyle: props.cursorStyle,
    fontFamily: fonts[props.fontFamily].family,
    fontSize: props.fontSize,
    lineHeight: props.lineHeight,
    scrollback: props.scrollback,
    scrollSensitivity: props.scrollSensitivity,
    rightClickSelectsWord: props.rightClickSelectsWord,
    macOptionClickForcesSelection: true,
    convertEol: false,
    theme: props.colors,
  }))
  const fitPlugin = new FitAddon()
  term.loadAddon(fitPlugin)
  term.open(container.value)
  terminal = term
  updateColors()
  fitAddon = fitPlugin
  offSelection = term.onSelectionChange(updateSelectedWebURL)
  void import('@xterm/addon-web-links').then(({ WebLinksAddon }) => {
    if (terminal === term) term.loadAddon(new WebLinksAddon(openTerminalLink, {
      urlRegex: /https?:\/\/[^\s"'!*(){}|\\^<>`]+[^\s"':,.!?{}|\\^~\[\]`()<>]/giu,
    }))
  }).catch(error => {
    if (terminal === term) notice.value = text('链接识别不可用：', 'Link detection unavailable: ') + String(error)
  })
  term.attachCustomKeyEventHandler((event) => {
    const writable = props.active && props.focused && props.connected && !props.session.streamClosed && props.session.status === 'running' && controlled.value &&
      !copyMode.value && !recovering.value && !replayError.value
    if (!writable) event.preventDefault()
    return !!writable
  })

  offInput = term.onData((input: string) => {
    if (!props.connected || props.session.streamClosed || replayError.value || !controlled.value || props.session.status !== 'running') return
    // onData also carries live terminal protocol replies, which background
    // panes must send. DOM key/paste handling separately guards user focus.
    writeQueue = writeQueue
      .then(() => queueSessionOperation(props.session.id, async () => {
        if (terminal === term && props.connected && !props.session.streamClosed && controlled.value && props.session.status === 'running') await Write(props.session.id, input)
      }))
      .catch((err: unknown) => emit('error', String(err)))
  })

  // Install listener before snapshot so no output is lost during bootstrap.
  offOutput = Events.On('terminal:output', (evt) => {
    const event = evt.data as OutputEvent
    if (replayError.value || !controlled.value || !event || event.id !== props.session.id) return
    if (replaying && pending.length >= 512) replayError.value = text('回放期间输出过多，请重新回放。', 'Too much output during replay; replay again.')
    else if (replaying) pending.push(event)
    else writeEvent(event)
  })

  observer = new ResizeObserver(fit)
  observer.observe(container.value)
  await nextTick()
  await acquireControl()
  if (terminal !== term || !controlled.value) { recovering.value = false; return }
  fit()
  if (props.active) focus()

  await replaySnapshot(term)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', dismissCopyMenu)
  controlled.value = false
  void updateAttachment(props.session.id, false).catch((error: unknown) => emit('error', String(error)))
  observer?.disconnect()
  offOutput?.()
  offSelection?.dispose()
  quickSelectRequest++
  quickSelectAbort?.abort()
  offQuickSelectScroll?.dispose()
  offSearchResults?.dispose()
  searchAddon?.clearDecorations()
  searchAddon = undefined
  offInput?.dispose()
  terminal?.dispose()
  terminal = undefined
})
</script>

<template>
  <div
    class="terminal-panel"
    :class="{ hidden: !active }"
    @keydown.capture="onTerminalKeydown"
    @paste.capture="onPanelPaste"
  >
    <div class="terminal-actions">
      <button type="button" :disabled="!controlled || recovering" @click="copySelection">{{ text('复制选区', 'Copy') }}</button>
      <button type="button" :disabled="!connected || session.streamClosed || recovering || !!replayError || !controlled || copyMode || session.status !== 'running'" @click="pasteClipboard">{{ text('粘贴', 'Paste') }}</button>
      <button type="button" :disabled="!controlled || recovering" :aria-pressed="copyMode" @click="toggleCopyMode">{{ copyMode ? text('退出复制模式', 'Exit copy mode') : text('复制模式', 'Copy mode') }}</button>
      <button type="button" :disabled="!connected || recovering" @click="reconnectView">{{ text('重新回放', 'Replay') }}</button>
      <button type="button" :disabled="!focused" :aria-pressed="searchOpen" @click="toggleSearch">{{ text('搜索', 'Search') }}</button>
      <button type="button" :disabled="!focused || !controlled || recovering" :aria-pressed="quickSelectOpen" @click="toggleQuickSelect">{{ text('快速选择', 'Quick Select') }}</button>
      <button v-if="selectedWebURL" type="button" @click="openSelectedWebLink">{{ text('打开所选链接…', 'Open selected link…') }}</button>
      <small>{{ text('可尝试 Shift 拖选（macOS Option）；无效时使用复制模式。', 'Try Shift-drag (Option on macOS); use Copy mode if unavailable.') }}</small>
    </div>
    <TerminalSearch v-if="searchOpen" :locale="locale" :query="searchQuery" :case-sensitive="searchCaseSensitive"
      :result-index="searchResultIndex" :result-count="searchResultCount"
      @update:query="searchQuery = $event" @update:case-sensitive="searchCaseSensitive = $event"
      @next="searchNext" @previous="searchPrevious" @close="closeSearch" />
    <div v-if="quickSelectLoading" class="copy-notice" role="status">{{ text('正在扫描当前缓冲…', 'Scanning terminal text…') }}</div>
    <div v-if="notice" class="copy-notice" role="status">{{ notice }}</div>
    <div v-if="replayError" class="copy-notice" role="alert">{{ text('画面输出不完整，已暂停输入；请重新回放：', 'Output incomplete; input paused. Replay to recover: ') }}{{ replayError }}</div>
    <div v-else-if="replayTruncated" class="copy-notice" role="status">{{ text('仅回放最近 256 KiB 输出，全屏画面可能不完整。', 'Only the last 256 KiB of output was replayed; full-screen content may be incomplete.') }}</div>
    <div v-if="attachmentError" class="copy-notice" role="status">
      {{ text('会话尚未连接：', 'Session not attached: ') }}{{ attachmentError }}
      <button type="button" @click="reconnectView">{{ text('重试连接', 'Retry attach') }}</button>
    </div>
    <div v-if="session.status === 'exited'" class="copy-notice" role="status">{{ text('会话已结束', 'Session ended') }} {{ session.message }}</div>
    <div v-else-if="session.streamClosed" class="copy-notice" role="status">{{ text('PTY 输出流已关闭，已暂停输入。', 'PTY output stream closed; input paused.') }}</div>
    <div class="terminal-content" @contextmenu.prevent="openCopyMenu">
      <div v-if="copyMode" class="copy-overlay">
        <small>{{ text('当前可见文字的只读副本；选择后复制，Esc 返回终端。', 'Read-only copy of visible text. Select and copy; Escape returns to the terminal.') }}</small>
        <textarea ref="copyEditor" :value="copyText" readonly spellcheck="false" :aria-label="text('复制模式文字', 'Copy mode text')" @mousedown.right.prevent @keydown="onCopyEditorKey" />
      </div>
      <div ref="container" class="xterm-host" :style="{ padding: props.padding + 'px', visibility: controlled ? 'visible' : 'hidden' }" @click="focus" />
      <QuickSelectOverlay v-if="quickSelectOpen" :locale="locale" :candidates="quickSelectCandidates"
        :truncated="quickSelectTruncated" :range="quickSelectRange"
        @update:range="quickSelectRange = $event" @selected="selectQuickCandidate" @close="closeQuickSelect" />
    </div>
    <Teleport to=".layout">
      <div v-if="copyMenu" ref="copyMenuElement" class="terminal-copy-menu" role="group" :aria-label="text('终端复制操作', 'Terminal copy actions')"
        :style="{ left: copyMenu.x + 'px', top: copyMenu.y + 'px' }" @pointerdown.prevent @keydown.esc.prevent.stop="copyMenu = null">
        <button type="button" @click="runCopyMenu('copy')">{{ text('复制选区', 'Copy selection') }}</button>
        <button type="button" :disabled="!connected || session.streamClosed || !!replayError || copyMode || session.status !== 'running'" @click="runCopyMenu('paste')">{{ text('粘贴', 'Paste') }}</button>
        <button v-if="copyMenuURL" type="button" @click="runCopyMenu('openLink')">{{ text('打开所选链接…', 'Open selected link…') }}</button>
        <button type="button" @click="runCopyMenu('mode')">{{ copyMode ? text('退出复制模式', 'Exit copy mode') : text('进入复制模式', 'Enter copy mode') }}</button>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.terminal-panel {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--canvas);
}
.terminal-content { min-height: 0; flex: 1; position: relative; }
.xterm-host { position: absolute; inset: 0; overflow: hidden; }
.terminal-actions { display: flex; align-items: center; flex-wrap: nowrap; flex: 0 0 28px; min-height: 28px; gap: 4px; padding: 2px 6px; border-bottom: 1px solid var(--border-faint); }
.terminal-actions button { flex: 0 0 auto; color: var(--text); background: var(--surface); border: 1px solid var(--border); border-radius: var(--dt-radius, 4px); font-size: 10px; line-height: 1.2; padding: 2px 5px; white-space: nowrap; }
.terminal-actions small { flex: 0 1 auto; min-width: 0; margin-left: auto; overflow: hidden; color: var(--faint); font-size: 9px; text-align: right; text-overflow: ellipsis; white-space: nowrap; }
.copy-notice { color: var(--muted); font-size: 10px; }
.copy-notice { padding: 4px 8px; }
.copy-overlay { position: absolute; z-index: 2; inset: 0; display: flex; flex-direction: column; gap: 8px; padding: 8px; background: var(--canvas); }
.copy-overlay small { color: var(--muted); }
.copy-overlay textarea { flex: 1; min-height: 0; resize: none; background: var(--canvas); color: var(--text); border: 1px solid var(--border); font-family: monospace; }
.terminal-copy-menu { position: fixed; z-index: 100; width: 200px; display: grid; gap: 4px; padding: 6px; background: var(--surface); border: 1px solid var(--border); border-radius: var(--dt-radius, 4px); }
.terminal-copy-menu button { color: var(--text); background: var(--canvas); border: 1px solid var(--border); padding: 5px; text-align: left; }
.terminal-panel.hidden {
  display: none;
}
.terminal-panel :deep(.xterm) {
  height: 100%;
}
</style>

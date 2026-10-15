<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { Events } from '@wailsio/runtime'
import { ReadText, WriteText } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/clipboardservice'
import { Resize, Snapshot, Write } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/sessionservice'
import type { Info } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/session/models'
import { fonts, type Palette, type FontId, type CursorStyle, type KeyboardMode, type Locale } from '../uiPreferences'
import { decideTerminalShortcut } from '../terminalShortcuts'
import { terminalColors } from '../userThemes'
import { matchBinding, type Binding } from '../keybindings'
import { queueSessionOperation, updateAttachment } from '../terminalAttachment'
import '@xterm/xterm/css/xterm.css'

const props = defineProps<{
  session: Info
  active: boolean
  focused: boolean
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
}>()
const emit = defineEmits<{ error: [message: string] }>()
const container = ref<HTMLElement | null>(null)
const copyMode = ref(false)
const copyText = ref('')
const copyEditor = ref<HTMLTextAreaElement | null>(null)
const notice = ref('')
const controlled = ref(false)
const attachmentError = ref('')
const text = (zh: string, en: string) => props.locale === 'zh' ? zh : en

interface OutputEvent {
  id: string
  data: string
  offset: number
}

let terminal: Terminal | undefined
let fitAddon: FitAddon | undefined
let observer: ResizeObserver | undefined
let offOutput: (() => void) | undefined
let offInput: { dispose(): void } | undefined
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
  if (!terminal || event.id !== props.session.id) return
  const bytes = decodeBase64(event.data)
  const firstOffset = event.offset - bytes.length
  if (event.offset <= lastOffset) return
  const skipped = Math.max(0, lastOffset - firstOffset)
  terminal.write(bytes.subarray(skipped))
  lastOffset = event.offset
}

function fit() {
  if (!props.active || !controlled.value || !terminal || !fitAddon || !container.value || container.value.clientWidth === 0 || container.value.clientHeight === 0) return
  try {
    const size = fitAddon.proposeDimensions()
    if (!size || size.cols < 1 || size.rows < 1) return
    const cols = Math.min(500, size.cols)
    const rows = Math.min(500, size.rows)
    if (terminal.cols !== cols || terminal.rows !== rows) terminal.resize(cols, rows)
    const key = `${cols}:${rows}`
    if (key !== lastSize && props.session.status === 'running') {
      lastSize = key
      const term = terminal
      void queueSessionOperation(props.session.id, async () => {
        if (terminal === term && controlled.value && props.active && key === lastSize) await Resize(props.session.id, cols, rows)
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

async function acquireControl() {
  const term = terminal
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
  replaying = true
  pending = []
  try {
    const snapshot = await Snapshot(props.session.id)
    if (terminal !== term) return
    const raw = decodeBase64(snapshot.data)
    if (raw.length) term.write(raw)
    lastOffset = snapshot.offset
  } catch (err) {
    emit('error', String(err))
  } finally {
    pending.sort((a, b) => a.offset - b.offset)
    for (const event of pending) writeEvent(event)
    pending = []
    replaying = false
  }
}

async function reconnectView() {
  const term = terminal
  if (!term) return
  await acquireControl()
  if (controlled.value && terminal === term) {
    term.reset()
    lastOffset = 0
    await replaySnapshot(term)
    focus()
  }
}

async function copySelection() {
  const editor = copyEditor.value
  const selected = copyMode.value && editor
    ? copyText.value.slice(editor.selectionStart, editor.selectionEnd)
    : terminal?.getSelection()
  if (!selected) { notice.value = text('请先选择文字（鼠标模式下使用 Shift 拖动）', 'Select text first (use Shift-drag in mouse mode)'); return }
  try {
    await writeClipboard(selected)
    notice.value = text('已复制', 'Copied')
  } catch (err) { notice.value = text('复制失败，请重试：', 'Copy failed; retry: ') + String(err) }
}

async function pasteClipboard() {
  const term = terminal
  if (!props.active || !props.focused || !controlled.value || !term || copyMode.value || props.session.status !== 'running') return
  try {
    const value = await readClipboard()
    if (terminal === term && props.active && props.focused && controlled.value && !copyMode.value && props.session.status === 'running') term.paste(value)
  } catch (err) { notice.value = text('粘贴失败，请重试：', 'Paste failed; retry: ') + String(err) }
}

async function toggleCopyMode() {
  copyMode.value = !copyMode.value
  notice.value = ''
  if (copyMode.value && terminal) {
    const buffer = terminal.buffer.active
    const lines: string[] = []
    for (let row = 0; row < terminal.rows; row++) {
      const line = buffer.getLine(buffer.viewportY + row)
      if (line) lines.push(line.translateToString(true))
    }
    copyText.value = lines.join('\n')
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
  const target = event.target as HTMLElement
  if (!target.classList.contains('xterm-helper-textarea') && target.closest('input, select, textarea')) return
  const binding = matchBinding(event, props.bindings, 'terminal')
  if (binding) {
    event.preventDefault(); event.stopPropagation()
    if (binding === 'copySelection') void copySelection()
    else if (binding === 'pasteClipboard') void pasteClipboard()
    else if (binding === 'toggleCopyMode') void toggleCopyMode()
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
    if (controlled.value && props.session.status === 'running') {
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

watch(() => props.active, async (active) => {
  if (active) {
    await nextTick()
    fit()
    focus()
  }
})
watch(() => props.focused, async (focused) => {
  if (focused) {
    await nextTick()
    if (!document.activeElement?.matches('select,input,textarea:not(.xterm-helper-textarea)')) focus()
  }
})

onMounted(async () => {
  if (!container.value) return
  const term = new Terminal({
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
  })
  const fitPlugin = new FitAddon()
  term.loadAddon(fitPlugin)
  term.open(container.value)
  terminal = term
  updateColors()
  fitAddon = fitPlugin

  offInput = term.onData((input: string) => {
    if (!props.active || !props.focused || !controlled.value || copyMode.value || props.session.status !== 'running') return
    // Keep keystrokes in order even if the IPC layer resolves asynchronously.
    writeQueue = writeQueue
      .then(() => queueSessionOperation(props.session.id, async () => {
        if (terminal === term && controlled.value && props.session.status === 'running') await Write(props.session.id, input)
      }))
      .catch((err: unknown) => emit('error', String(err)))
  })

  // Install listener before snapshot so no output is lost during bootstrap.
  offOutput = Events.On('terminal:output', (evt) => {
    const event = evt.data as OutputEvent
    if (!controlled.value || !event || event.id !== props.session.id) return
    if (replaying) pending.push(event)
    else writeEvent(event)
  })

  observer = new ResizeObserver(fit)
  observer.observe(container.value)
  await nextTick()
  await acquireControl()
  if (terminal !== term || !controlled.value) return
  fit()
  if (props.active) focus()

  await replaySnapshot(term)
})

onBeforeUnmount(() => {
  controlled.value = false
  void updateAttachment(props.session.id, false).catch((error: unknown) => emit('error', String(error)))
  observer?.disconnect()
  offOutput?.()
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
  >
    <div class="terminal-actions">
      <button type="button" :disabled="!controlled" @click="copySelection">{{ text('复制选区', 'Copy') }}</button>
      <button type="button" :disabled="!controlled || copyMode || session.status !== 'running'" @click="pasteClipboard">{{ text('粘贴', 'Paste') }}</button>
      <button type="button" :disabled="!controlled" :aria-pressed="copyMode" @click="toggleCopyMode">{{ copyMode ? text('退出复制模式', 'Exit copy mode') : text('复制模式', 'Copy mode') }}</button>
      <small>{{ text('Shift 拖选 · macOS Option 拖选', 'Shift-drag · macOS Option-drag') }}</small>
    </div>
    <div v-if="notice" class="copy-notice" role="status">{{ notice }}</div>
    <div v-if="attachmentError" class="copy-notice" role="status">
      {{ text('会话尚未连接：', 'Session not attached: ') }}{{ attachmentError }}
      <button type="button" @click="reconnectView">{{ text('重试连接', 'Retry attach') }}</button>
    </div>
    <div v-if="session.status === 'exited'" class="copy-notice" role="status">{{ text('会话已结束', 'Session ended') }} {{ session.message }}</div>
    <div class="terminal-content">
      <div v-if="copyMode" class="copy-overlay">
        <small>{{ text('当前可见文字的只读副本；选择后复制，Esc 返回终端。', 'Read-only copy of visible text. Select and copy; Escape returns to the terminal.') }}</small>
        <textarea ref="copyEditor" :value="copyText" readonly spellcheck="false" :aria-label="text('复制模式文字', 'Copy mode text')" @keydown="onCopyEditorKey" />
      </div>
      <div ref="container" class="xterm-host" :style="{ padding: props.padding + 'px', visibility: controlled ? 'visible' : 'hidden' }" @click="focus" />
    </div>
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
.terminal-actions { display: flex; align-items: center; flex-wrap: wrap; gap: 6px; padding: 4px 8px; border-bottom: 1px solid var(--border-faint); }
.terminal-actions button { color: var(--text); background: var(--surface); border: 1px solid var(--border); border-radius: var(--dt-radius, 4px); font-size: 11px; padding: 3px 6px; }
.terminal-actions small, .copy-notice { color: var(--muted); font-size: 10px; }
.copy-notice { padding: 4px 8px; }
.copy-overlay { position: absolute; z-index: 2; inset: 0; display: flex; flex-direction: column; gap: 8px; padding: 8px; background: var(--canvas); }
.copy-overlay small { color: var(--muted); }
.copy-overlay textarea { flex: 1; min-height: 0; resize: none; background: var(--canvas); color: var(--text); border: 1px solid var(--border); font-family: monospace; }
.terminal-panel.hidden {
  display: none;
}
.terminal-panel :deep(.xterm) {
  height: 100%;
}
</style>

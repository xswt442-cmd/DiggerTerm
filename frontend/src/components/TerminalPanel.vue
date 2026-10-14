<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { Clipboard, Events } from '@wailsio/runtime'
import { Resize, Snapshot, Write } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/sessionservice'
import type { Info } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/session/models'
import { fonts, type Palette, type FontId, type CursorStyle, type KeyboardMode } from '../uiPreferences'
import { decideTerminalShortcut } from '../terminalShortcuts'
import '@xterm/xterm/css/xterm.css'

const props = defineProps<{
  session: Info
  active: boolean
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
}>()
const emit = defineEmits<{ error: [message: string] }>()
const container = ref<HTMLElement | null>(null)

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
  if (!props.active || !terminal || !fitAddon || !container.value || container.value.clientWidth === 0) return
  try {
    fitAddon.fit()
    const cols = terminal.cols
    const rows = terminal.rows
    if (cols > 0 && rows > 0 && props.session.status === 'running') {
      void Resize(props.session.id, cols, rows).catch((err: unknown) => {
        emit('error', String(err))
      })
    }
  } catch (error) {
    emit('error', String(error))
  }
}

function focus() {
  terminal?.focus()
}

// Wails supplies a native clipboard bridge, which is more dependable than
// navigator.clipboard in desktop WebViews (particularly under Linux/WSLg).
async function writeClipboard(text: string) {
  try {
    await Clipboard.SetText(text)
  } catch (nativeError) {
    if (!navigator.clipboard?.writeText) throw nativeError
    await navigator.clipboard.writeText(text)
  }
}

async function readClipboard(): Promise<string> {
  try {
    return await Clipboard.Text()
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
  if (!props.active || !term) return
  const shortcut = decideTerminalShortcut(
    event, term.hasSelection(), isMac, props.keyboardMode, term.buffer.active.type === 'alternate',
  )
  if (shortcut.kind === 'pass') return

  // Capture ahead of xterm and the WebView's default Copy/Paste/Reload,
  // and do not allow xterm to send the same control byte twice.
  event.preventDefault()
  event.stopPropagation()

  if (shortcut.kind === 'input') {
    if (props.session.status === 'running') {
      term.input(shortcut.data, true) // goes through onData -> ordered daemon IPC
    }
    return
  }
  if (shortcut.kind === 'copy') {
    const selected = term.getSelection()
    if (selected) {
      void writeClipboard(selected).catch((err: unknown) => emit('error', String(err)))
    }
    return
  }
  if (shortcut.kind === 'paste' && props.session.status === 'running') {
    void readClipboard()
      .then((text) => {
        if (text && terminal === term && props.active && props.session.status === 'running') {
          // xterm.paste handles CR/LF normalization and bracketed paste mode.
          term.paste(text)
        }
      })
      .catch((err: unknown) => emit('error', String(err)))
  }
}

watch(() => props.fontSize, async (size) => {
  if (terminal) {
    terminal.options.fontSize = size
    await nextTick()
    fit()
  }
})
watch(() => props.colors, (colors) => {
  if (terminal) terminal.options.theme = colors
}, { deep: true })

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
    convertEol: false,
    theme: props.colors,
  })
  const fitPlugin = new FitAddon()
  term.loadAddon(fitPlugin)
  term.open(container.value)
  terminal = term
  fitAddon = fitPlugin

  offInput = term.onData((input: string) => {
    if (props.session.status !== 'running') return
    // Keep keystrokes in order even if the IPC layer resolves asynchronously.
    writeQueue = writeQueue
      .then(() => Write(props.session.id, input))
      .catch((err: unknown) => emit('error', String(err)))
  })

  // Install listener before snapshot so no output is lost during bootstrap.
  offOutput = Events.On('terminal:output', (evt) => {
    const event = evt.data as OutputEvent
    if (!event || event.id !== props.session.id) return
    if (replaying) pending.push(event)
    else writeEvent(event)
  })

  observer = new ResizeObserver(fit)
  observer.observe(container.value)
  await nextTick()
  fit()
  if (props.active) focus()

  try {
    const snapshot = await Snapshot(props.session.id)
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
})

onBeforeUnmount(() => {
  observer?.disconnect()
  offOutput?.()
  offInput?.dispose()
  terminal?.dispose()
  terminal = undefined
})
</script>

<template>
  <div
    ref="container"
    class="terminal-panel"
    :class="{ hidden: !active }"
    :style="{ padding: props.padding + 'px' }"
    @click="focus"
    @keydown.capture="onTerminalKeydown"
  />
</template>

<style scoped>
.terminal-panel {
  position: absolute;
  inset: 0;
  padding: 12px 16px;
  overflow: hidden;
  background: var(--canvas);
}
.terminal-panel.hidden {
  display: none;
}
.terminal-panel :deep(.xterm) {
  height: 100%;
}
</style>

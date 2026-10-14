<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { Events } from '@wailsio/runtime'
import { Resize, Snapshot, Write } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/sessionservice'
import type { Info } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/session/models'
import '@xterm/xterm/css/xterm.css'

const props = defineProps<{ session: Info; active: boolean }>()
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
    cursorBlink: true,
    fontFamily: 'Consolas, Cascadia Code, JetBrains Mono, monospace',
    fontSize: 13,
    lineHeight: 1.25,
    scrollback: 5000,
    convertEol: false,
    theme: {
      background: '#101217',
      foreground: '#dce0e7',
      cursor: '#dbc29b',
      selectionBackground: '#555a70',
      black: '#171b22',
      blue: '#80aef5',
      green: '#8fd7ab',
      red: '#f48e8e',
      yellow: '#e4c47b',
    },
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
    @click="focus"
  />
</template>

<style scoped>
.terminal-panel {
  position: absolute;
  inset: 0;
  padding: 12px 16px;
  overflow: hidden;
  background: #101217;
}
.terminal-panel.hidden {
  display: none;
}
.terminal-panel :deep(.xterm) {
  height: 100%;
}
</style>

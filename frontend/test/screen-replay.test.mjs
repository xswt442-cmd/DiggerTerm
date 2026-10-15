import assert from 'node:assert/strict'
import { test } from 'node:test'
import xtermHeadless from '@xterm/headless'

const { Terminal } = xtermHeadless

function write(terminal, data) {
  return new Promise(resolve => terminal.write(data, resolve))
}

function readScreen(terminal) {
  const buffer = terminal.buffer.active
  const lines = []
  const cells = []
  for (let y = 0; y < terminal.rows; y++) {
    const line = buffer.getLine(y)
    lines.push(line?.translateToString(true) ?? '')
    const row = []
    for (let x = 0; x < terminal.cols; x++) {
      const cell = line?.getCell(x)
      row.push(cell ? {
        chars: cell.getChars(), width: cell.getWidth(),
        fgMode: cell.getFgColorMode(), fg: cell.getFgColor(),
        bgMode: cell.getBgColorMode(), bg: cell.getBgColor(),
        bold: cell.isBold(), underline: cell.isUnderline(),
      } : null)
    }
    cells.push(row)
  }
  return {
    type: buffer.type,
    cursorX: buffer.cursorX,
    cursorY: buffer.cursorY,
    lines,
    cells,
  }
}

function makeTerminal() {
  return new Terminal({ cols: 80, rows: 24, scrollback: 1000, allowProposedApi: true })
}

test('full replay preserves startup and alternate-screen state lost by a bounded suffix', async t => {
  const startup = '\u001b[?1049h\u001b[2J\u001b[1;1HBOOT HEADER\u001b[2;1HSTATE FROM BEFORE REPLAY WINDOW\u001b[31mRED\u001b[0m'
  const fillerLine = '\u001b[24;1H' + 'x'.repeat(79)
  const filler = fillerLine.repeat(3500)
  const tail = '\u001b[8;12HACTIVE CURSOR\u001b[?25h'
  const bytes = new TextEncoder().encode(startup + filler + tail)
  assert.ok(bytes.length > 256 * 1024)

  const full = makeTerminal()
  const truncated = makeTerminal()
  t.after(() => { full.dispose(); truncated.dispose() })
  await write(full, bytes)
  await write(truncated, bytes.subarray(bytes.length - 256 * 1024))

  const fullScreen = readScreen(full)
  const truncatedScreen = readScreen(truncated)
  assert.equal(fullScreen.type, 'alternate')
  assert.equal(fullScreen.lines[0], 'BOOT HEADER')
  assert.equal(fullScreen.lines[1], 'STATE FROM BEFORE REPLAY WINDOWRED')
  assert.equal(fullScreen.cursorX, truncatedScreen.cursorX)
  assert.equal(fullScreen.cursorY, truncatedScreen.cursorY)
  assert.notDeepEqual(truncatedScreen, fullScreen)
  assert.notEqual(truncatedScreen.type, fullScreen.type)
  assert.equal(truncatedScreen.lines.includes('BOOT HEADER'), false)
})

test('xterm headless reconstructs the same screen from whole and fragmented UTF-8/ANSI output', async t => {
  const whole = makeTerminal()
  const fragmented = makeTerminal()
  t.after(() => { whole.dispose(); fragmented.dispose() })
  const bytes = new TextEncoder().encode(
    '\u001b[?1049h\u001b[2J\u001b[1;1H\u001b[1;34m蓝色標題 🌲\u001b[0m\r\n' +
    '\u001b[2;3H\u001b[48;2;12;34;56m背景\u001b[0m\u001b[4;8H末尾',
  )
  await write(whole, bytes)
  for (let offset = 0; offset < bytes.length; offset++) {
    await write(fragmented, bytes.subarray(offset, offset + 1))
  }
  assert.deepEqual(readScreen(fragmented), readScreen(whole))
})

test('historical replay suppresses protocol replies and live output enables them', async t => {
  const terminal = makeTerminal()
  t.after(() => terminal.dispose())
  const replies = []
  terminal.onData(data => replies.push(data))
  terminal.options.disableStdin = true
  await write(terminal, '\u001b[6n\u001b[c')
  assert.deepEqual(replies, [])
  terminal.options.disableStdin = false
  await write(terminal, '\u001b[5n')
  assert.deepEqual(replies, ['\u001b[0n'])
})

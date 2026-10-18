import type { QuickSelectCandidate, QuickSelectScan } from './quickSelectCore'

const maxQuickSelectText = 2 * 1024 * 1024

interface BufferCellLike { getChars(): string; getWidth(): number }
interface BufferLineLike {
  isWrapped: boolean
  length: number
  translateToString(trimRight?: boolean): string
  getCell(column: number): BufferCellLike | undefined
}
interface BufferLike {
  buffer: { active: { baseY: number; viewportY: number; length: number; getLine(index: number): BufferLineLike | undefined } }
}
export interface QuickSelectTextRow { start: number; end: number; bufferRow: number; cellColumns: Uint16Array }
export interface QuickSelectTextSnapshot { text: string; rows: QuickSelectTextRow[]; truncated: boolean; viewportY: number }
export interface PositionedQuickSelectCandidate extends QuickSelectCandidate { bufferRow: number; column: number }

export async function collectQuickSelectText(
  terminal: BufferLike & { rows: number },
  range: 'visible' | 'buffer',
  signal?: AbortSignal,
): Promise<QuickSelectTextSnapshot> {
  const buffer = terminal.buffer.active
  const viewportY = buffer.viewportY
  const first = range === 'visible' ? viewportY : 0
  const last = range === 'visible' ? Math.min(buffer.length, viewportY + terminal.rows) : Math.min(buffer.length, buffer.baseY + terminal.rows)
  const chunks: string[] = []
  const rows: QuickSelectTextRow[] = []
  let size = 0
  let truncated = false
  for (let index = first; index < last; index++) {
    if (signal?.aborted) throw new DOMException('Quick Select cancelled', 'AbortError')
    const line = buffer.getLine(index)
    if (!line) continue
    const separator = rows.length && !line.isWrapped ? '\n' : ''
    const fullText = line.translateToString(true)
    const remaining = maxQuickSelectText - size - separator.length
    if (remaining <= 0) { truncated = true; break }
    let lineText = fullText.slice(0, remaining)
    const lastUnit = lineText.charCodeAt(lineText.length - 1)
    if (lineText.length < fullText.length && lastUnit >= 0xd800 && lastUnit <= 0xdbff) lineText = lineText.slice(0, -1)
    if (lineText.length < fullText.length) truncated = true
    if (separator) { chunks.push(separator); size++ }
    const start = size
    chunks.push(lineText)
    size += lineText.length
    const cellColumns = new Uint16Array(lineText.length + 1)
    let offset = 0
    for (let column = 0; column < line.length && offset < lineText.length; column++) {
      const cell = line.getCell(column)
      if (!cell) continue
      const width = cell.getWidth()
      if (width === 0) continue
      const chars = cell.getChars() || ' '
      for (let unit = 0; unit < chars.length && offset < lineText.length; unit++) cellColumns[offset++] = column
    }
    if (offset !== lineText.length) {
      // If a terminal implementation exposes inconsistent cell text, omit this
      // row from selection instead of guessing UTF-16 offsets as screen columns.
      rows.push({ start, end: start, bufferRow: index, cellColumns: new Uint16Array() })
    } else {
      rows.push({ start, end: size, bufferRow: index, cellColumns })
    }
    if ((index - first + 1) % 128 === 0) await new Promise(resolve => setTimeout(resolve, 0))
  }
  return { text: chunks.join(''), rows, truncated, viewportY }
}

export function positionQuickSelectCandidates(
  scan: QuickSelectScan,
  snapshot: QuickSelectTextSnapshot,
): PositionedQuickSelectCandidate[] {
  const rows = snapshot.rows
  const locate = (offset: number) => {
    let low = 0
    let high = rows.length - 1
    while (low <= high) {
      const middle = (low + high) >>> 1
      const row = rows[middle]!
      if (offset < row.start) high = middle - 1
      else if (offset >= row.end) low = middle + 1
      else return row
    }
    return undefined
  }
  const result: PositionedQuickSelectCandidate[] = []
  for (const candidate of scan.candidates) {
    const row = locate(candidate.start)
    if (!row?.cellColumns.length) continue
    result.push({ ...candidate, bufferRow: row.bufferRow, column: row.cellColumns[candidate.start - row.start]! })
  }
  return result
}

import type { QuickSelectConfig, QuickSelectRule } from '../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/models'
import { validateExternalWebURL } from './externalLinks'

export const QUICK_SELECT_MAX_TEXT = 2 * 1024 * 1024
export const QUICK_SELECT_MAX_CANDIDATES = 500
const letters = 'abcdefghijklmnopqrstuvwxyz'

export interface QuickSelectCandidate {
  label: string
  type: string
  name: string
  text: string
  start: number
  end: number
  priority: number
}

export interface QuickSelectScan {
  candidates: QuickSelectCandidate[]
  truncated: boolean
  errors: string[]
}

export function validateQuickSelectRules(rules: QuickSelectRule[]): string[] {
  const errors: string[] = []
  for (const rule of rules) {
    try { new RegExp(rule.pattern, 'gu') }
    catch (error) { errors.push(`${rule.name}: ${String(error)}`) }
  }
  return errors
}

function isIPv4(value: string): boolean {
  const pieces = value.split('.')
  return pieces.length === 4 && pieces.every(piece => /^(?:0|[1-9]\d{0,2})$/.test(piece) && Number(piece) <= 255)
}

function isIPv6(value: string): boolean {
  try { return new URL(`http://[${value}]/`).hostname.length > 0 }
  catch { return false }
}

function clean(value: string, type: string): string {
  if (type === 'url') return value.replace(/[.,!?;:)}\]]+$/u, '')
  if (type === 'path') return value.replace(/[.,!?;:)}\]]+$/u, '')
  return value
}

function collectPaths(text: string, priority: number, add: (candidate: QuickSelectCandidate) => boolean) {
  const boundaries: number[] = []
  const boundaryPattern = /[\r\n,;!?)]|\s+(?:and|or|then|use|open|commit|run|with)\b/giu
  for (const match of text.matchAll(boundaryPattern)) boundaries.push(match.index ?? 0)
  const pathStarts = String.raw`(?:^|[\s"'=])((?:[A-Za-z]:[\\/]|\\\\|/))`
  const startPattern = new RegExp(pathStarts, 'gu')
  for (const match of text.matchAll(startPattern)) {
    const prefix = match[1] ?? ''
    const start = (match.index ?? 0) + match[0].indexOf(prefix)
    let low = 0
    let high = boundaries.length
    while (low < high) {
      const middle = (low + high) >>> 1
      if (boundaries[middle]! <= start) low = middle + 1
      else high = middle
    }
    const raw = text.slice(start, boundaries[low] ?? text.length)
    const value = clean(raw, 'path')
    if (value && !add({ label: '', type: 'path', name: 'path', text: value, start, end: start + value.length, priority })) return
  }
}

function collectPattern(
  source: string,
  text: string,
  type: string,
  name: string,
  priority: number,
  errors: string[],
  add: (candidate: QuickSelectCandidate) => boolean,
  useFirstCapture = false,
  flags = 'gu',
) {
  let expression: RegExp
  try { expression = new RegExp(source, flags) }
  catch (error) { errors.push(`${name}: ${String(error)}`); return }
  for (const match of text.matchAll(expression)) {
    const raw = useFirstCapture && typeof match[1] === 'string' ? match[1] : match[0]
    if (!raw.length) continue
    const value = clean(raw, type)
    if (!value) continue
    const matchOffset = match.index ?? 0
    const start = matchOffset + (raw === match[0] ? 0 : match[0].indexOf(raw))
    const end = start + value.length
    if (type === 'url') {
      try { validateExternalWebURL(value) } catch { continue }
    } else if (type === 'ipv4' && !isIPv4(value)) continue
    else if (type === 'ipv6' && !isIPv6(value)) continue
    if (!add({ label: '', type, name, text: value, start, end, priority })) return
  }
}

export function scanQuickSelectCandidates(text: string, config: QuickSelectConfig): QuickSelectScan {
  if (text.length > QUICK_SELECT_MAX_TEXT) throw new Error(`Quick Select input exceeds ${QUICK_SELECT_MAX_TEXT} UTF-16 units`)
  const errors: string[] = []
  const matches: QuickSelectCandidate[] = []
  let truncated = false
  const add = (candidate: QuickSelectCandidate) => {
    if (matches.length >= QUICK_SELECT_MAX_CANDIDATES) { truncated = true; return false }
    matches.push(candidate)
    return true
  }
  const enabled = new Set(config.enabledTypes ?? [])
  const rules = config.rules ?? []
  const builtins: Array<[string, string, string, number]> = [
    ['url', 'url', 'https?:\\/\\/[^\\s"\'<>`]+', 0],
    ['ip', 'ipv4', '(?:^|[^\\w.])((?:\\d{1,3}\\.){3}\\d{1,3})(?![\\w.])', 1],
    ['ip', 'ipv6', '(?:^|[^\\w:])((?:[0-9a-f]{0,4}:){2,7}[0-9a-f]{0,4})(?![\\w:])', 2],
    ['gitHash', 'gitHash', '(?:^|[^\\w])([a-f0-9]{7,40})(?![\\w])', 4],
  ]
  for (const [kind, type, pattern, priority] of builtins) {
    if (!enabled.has(kind)) continue
    const name = type === 'ipv4' || type === 'ipv6' ? 'IP address' : type
    collectPattern(pattern, text, type, name, priority, errors, add,
      type !== 'url', type === 'ipv6' || type === 'path' ? 'giu' : 'gu')
  }
  if (enabled.has('path') && !truncated) collectPaths(text, 3, add)
  for (let index = 0; index < rules.length && !truncated; index++) {
    const rule = rules[index] as QuickSelectRule
    if (!rule.enabled) continue
    collectPattern(rule.pattern, text, `custom:${rule.id}`, rule.name, 100 + index, errors, add)
  }
  matches.sort((a, b) => a.priority - b.priority || a.start - b.start || b.end - a.end)
  const accepted: QuickSelectCandidate[] = []
  const seen = new Set<string>()
  for (const candidate of matches) {
    const identity = `${candidate.start}:${candidate.end}:${candidate.text}`
    if (seen.has(identity) || accepted.some(item => candidate.start < item.end && candidate.end > item.start)) continue
    seen.add(identity)
    accepted.push(candidate)
  }
  accepted.sort((a, b) => a.start - b.start || a.priority - b.priority)
  const useTwoCharacters = accepted.length > letters.length
  for (let index = 0; index < accepted.length; index++) {
    accepted[index]!.label = useTwoCharacters
      ? letters[Math.floor(index / letters.length)]! + letters[index % letters.length]!
      : letters[index]!
  }
  return { candidates: accepted, truncated, errors }
}

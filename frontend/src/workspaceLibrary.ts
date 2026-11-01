export type DirectoryItem = { scope: string; path: string }
export type SessionMark = { note: string; color: string }
export type SessionPreset = { id: string; name: string; profile: string; directory: string; target?: string; port?: number }
export type WorkspaceLibrary = { favorites: DirectoryItem[]; recent: DirectoryItem[]; marks: Record<string, SessionMark>; presets: SessionPreset[] }
export const markColors = ['', 'blue', 'green', 'amber', 'rose', 'violet'] as const
const storageKey = 'diggerterm:workspace-library:v1'
export const emptyLibrary = (): WorkspaceLibrary => ({ favorites: [], recent: [], marks: Object.create(null) as Record<string, SessionMark>, presets: [] })
const validText = (value: unknown, max: number): value is string => typeof value === 'string' && value.length > 0 && value.length <= max && !/[\x00-\x1f\x7f-\x9f\u2028\u2029\u202a-\u202e\u2066-\u2069]/u.test(value)
export function validPresetName(value: string): boolean {
  return value.trim().length > 0 && Array.from(value.trim()).length <= 64 && !/[\p{Cc}\p{Cf}]/u.test(value)
}
export function validPresetSSHTarget(value: string): boolean {
  return value.length > 0 && value.length <= 255 && /^(?![-.])[a-zA-Z0-9_.-]+(?:@(?![-.])[a-zA-Z0-9_.-]+)?$/.test(value)
}
const directoryItem = (x: unknown): x is DirectoryItem => !!x && typeof x === 'object' && validText((x as DirectoryItem).scope, 128) && validText((x as DirectoryItem).path, 2048)
const uniqueDirectories = (items: unknown, limit: number): DirectoryItem[] => {
  if (!Array.isArray(items)) return []
  const result: DirectoryItem[] = []
  for (const item of items) {
    if (directoryItem(item) && !result.some(x => x.scope === item.scope && x.path === item.path)) result.push({ scope: item.scope, path: item.path })
    if (result.length >= limit) break
  }
  return result
}
export function parseLibrary(raw: unknown): WorkspaceLibrary {
  const result = emptyLibrary()
  if (!raw || typeof raw !== 'object') return result
  const source = raw as Partial<WorkspaceLibrary>
  result.favorites = uniqueDirectories(source.favorites, 100)
  result.recent = uniqueDirectories(source.recent, 100)
  if (source.marks && typeof source.marks === 'object' && !Array.isArray(source.marks)) {
    for (const [id, mark] of Object.entries(source.marks).slice(0, 500)) {
      if (validText(id, 128) && mark && typeof mark.note === 'string' && mark.note.length <= 240 && !/[\x00-\x1f\x7f-\x9f]/u.test(mark.note) && markColors.includes(mark.color as typeof markColors[number])) {
        result.marks[id] = { note: mark.note, color: mark.color }
      }
    }
  }
  if (Array.isArray(source.presets)) {
    for (const item of source.presets) {
      if (item && validText(item.id, 128) && typeof item.name === 'string' && validPresetName(item.name) && validText(item.profile, 32) && typeof item.directory === 'string' && item.directory.length <= 512 && (!item.directory || validText(item.directory, 512)) && !result.presets.some(p => p.id === item.id)) {
        if (item.profile === 'ssh') {
          if (typeof item.target !== 'string' || !validPresetSSHTarget(item.target) || !Number.isInteger(item.port) || item.port! < 0 || item.port! > 65535 || item.directory !== '') continue
          result.presets.push({ id: item.id, name: item.name, profile: item.profile, directory: '', target: item.target, port: item.port })
        } else {
          result.presets.push({ id: item.id, name: item.name, profile: item.profile, directory: item.directory })
        }
      }
      if (result.presets.length >= 50) break
    }
  }
  return result
}
export function loadLibrary(): WorkspaceLibrary {
  try { return parseLibrary(JSON.parse(localStorage.getItem(storageKey) || '{}')) } catch { return emptyLibrary() }
}
export function saveLibrary(value: WorkspaceLibrary): void { localStorage.setItem(storageKey, JSON.stringify(value)) }
export function directoryScope(profile: string, id: string): string {
  if (profile === 'ssh') return `ssh:${id}`
  if (profile === 'powershell' || profile === 'cmd') return 'local:windows'
  return `local:${profile}`
}
export function rememberDirectory(data: WorkspaceLibrary, item: DirectoryItem): void {
  if (!directoryItem(item)) return
  data.recent = [item, ...data.recent.filter(x => x.scope !== item.scope || x.path !== item.path)].slice(0, 100)
}
export function toggleFavorite(data: WorkspaceLibrary, item: DirectoryItem): void {
  if (!directoryItem(item)) return
  if (data.favorites.some(x => x.scope === item.scope && x.path === item.path)) data.favorites = data.favorites.filter(x => x.scope !== item.scope || x.path !== item.path)
  else if (data.favorites.length < 100) data.favorites.push(item)
}

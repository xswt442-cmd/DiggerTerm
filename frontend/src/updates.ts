const VERSION = /^(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)$/
const RELEASE_TAG = /^v?((?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*))$/

export const AUTO_UPDATE_CHECK_INTERVAL_MS = 24 * 60 * 60 * 1000
export const AUTO_UPDATE_CHECK_DELAY_MS = 4_000

export interface LatestRelease {
  tag: string
  version: string
}

function versionParts(version: string): [number, number, number] {
  if (!VERSION.test(version)) throw new Error('Invalid semantic version')
  const [major, minor, patch] = version.split('.').map(Number)
  return [major!, minor!, patch!]
}

export function compareVersions(left: string, right: string): number {
  const a = versionParts(left)
  const b = versionParts(right)
  for (let index = 0; index < 3; index++) {
    if (a[index] !== b[index]) return a[index]! - b[index]!
  }
  return 0
}

export function parseLatestRelease(value: unknown): LatestRelease {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('Invalid release response')
  const release = value as Record<string, unknown>
  if (typeof release.tag_name !== 'string') throw new Error('Invalid release tag')
  const match = RELEASE_TAG.exec(release.tag_name)
  if (!match) throw new Error('Invalid release version')
  return { tag: release.tag_name, version: match[1]! }
}

export function automaticUpdateCheckDue(lastCheckedAt: number | null, now = Date.now()): boolean {
  if (lastCheckedAt === null || !Number.isFinite(lastCheckedAt) || lastCheckedAt <= 0) return true
  if (lastCheckedAt > now) return false
  return now - lastCheckedAt >= AUTO_UPDATE_CHECK_INTERVAL_MS
}

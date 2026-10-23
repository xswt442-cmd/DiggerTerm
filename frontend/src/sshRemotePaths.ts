export function remoteChildPath(parent: string, name: string): string {
  if (!parent.startsWith('/') || !name || name === '.' || name === '..' || name.includes('/') || name.includes('\\')) {
    throw new Error('Invalid remote directory')
  }
  return (parent === '/' ? '' : parent) + '/' + name
}

export function remoteParentPath(path: string): string {
  if (!path.startsWith('/') || path === '/') return '/'
  const trimmed = path.replace(/\/+$/, '')
  const index = trimmed.lastIndexOf('/')
  return index <= 0 ? '/' : trimmed.slice(0, index)
}

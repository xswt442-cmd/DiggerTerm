const maximumURLLength = 4096
const unsafeControls = /[\u0000-\u001f\u007f-\u009f\u202a-\u202e\u2066-\u2069]/u

export type ExplicitLinkMouseEvent = Pick<MouseEvent, 'ctrlKey' | 'metaKey' | 'preventDefault'>

export function createExplicitWebLinkActivator(isMac: boolean, open: (value: string) => void) {
  return (event: ExplicitLinkMouseEvent, value: string) => {
    if (isMac ? !event.metaKey : !event.ctrlKey) return
    event.preventDefault()
    open(value)
  }
}

export function validateExternalWebURL(value: string): URL {
  if (typeof value !== 'string' || value.length === 0 || value.length > maximumURLLength) {
    throw new Error('Link is empty or too long')
  }
  if (value !== value.trim() || unsafeControls.test(value)) {
    throw new Error('Link contains whitespace or control characters')
  }
  let url: URL
  try { url = new URL(value) }
  catch { throw new Error('Link is not an absolute web address') }
  if (!['http:', 'https:'].includes(url.protocol) || !url.hostname || url.username || url.password) {
    throw new Error('Only HTTP/HTTPS links without embedded credentials can be opened')
  }
  return url
}

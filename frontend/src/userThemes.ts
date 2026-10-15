import { palettes, themes, type BuiltinThemeId, type Palette, type ThemeId } from './uiPreferences'

export interface UserTheme {
  id: string
  name: string
  base: string
  colors: Record<string, string | undefined> | null
  terminal: Record<string, string | undefined> | null
  appearance: Record<string, number | undefined> | null
  css: string
}
export interface ThemeOption { id: ThemeId; palette: Palette; css: string }

export function themeRegistry(files: UserTheme[]): ThemeOption[] {
  const builtin: ThemeOption[] = themes.map(id => ({ id: `builtin:${id}`, palette: palettes[id], css: '' }))
  return [...builtin, ...files.map(file => {
    const base = palettes[file.base as BuiltinThemeId] ?? palettes.nightfall
    const css = { ...base.css }
    for (const [key, value] of Object.entries(file.colors ?? {})) {
      if (value !== undefined) css['--' + key.replace(/[A-Z]/g, c => '-' + c.toLowerCase())] = value
    }
    for (const [key, value] of Object.entries(file.appearance ?? {})) {
      if (value !== undefined) css['--dt-' + key.replace(/[A-Z]/g, c => '-' + c.toLowerCase())] = value + 'px'
    }
    const terminal = { ...base.terminal }
    for (const [key, value] of Object.entries(file.terminal ?? {})) {
      if (value !== undefined) Object.assign(terminal, { [key]: value })
    }
    return {
      id: file.id as ThemeId, css: file.css,
      palette: { zh: file.name, en: file.name, css, terminal,
        chips: [css['--sidebar'], css['--canvas'], css['--accent']] as [string, string, string] },
    }
  })]
}

// Keep the legacy variables as aliases so existing component styles and public
// --dt-* overrides use the same cascade. Preferences for fonts stay explicit.
export function themeVariables(palette: Palette): Record<string, string> {
  const css: Record<string, string> = {}
  for (const [name, value] of Object.entries(palette.css)) {
    if (name.startsWith('--dt-')) css[name] = value
    else {
      css['--dt-' + name.slice(2)] = value
      css[name] = `var(--dt-${name.slice(2)})`
    }
  }
  for (const [name, value] of Object.entries(palette.terminal)) {
    const key = name === 'background' ? 'bg' : name === 'foreground' ? 'fg' : name.replace(/[A-Z]/g, c => '-' + c.toLowerCase())
    css['--dt-terminal-' + key] = value
  }
  return css
}

export function terminalColors(element: HTMLElement, base: Palette['terminal']): Palette['terminal'] {
  const style = getComputedStyle(element)
  const result = { ...base }
  const names = ['background', 'foreground', 'cursor', 'cursorAccent', 'selectionBackground',
    'selectionForeground', 'selectionInactiveBackground', 'black', 'red', 'green', 'yellow',
    'blue', 'magenta', 'cyan', 'white', 'brightBlack', 'brightRed', 'brightGreen', 'brightYellow',
    'brightBlue', 'brightMagenta', 'brightCyan', 'brightWhite'] as const
  for (const name of names) {
    const key = name === 'background' ? 'bg' : name === 'foreground' ? 'fg' : name.replace(/[A-Z]/g, c => '-' + c.toLowerCase())
    const value = style.getPropertyValue('--dt-terminal-' + key).trim()
    if (value && CSS.supports('color', value)) result[name] = value
  }
  return result
}

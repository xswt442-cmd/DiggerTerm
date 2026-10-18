export interface PaletteCommand {
  id: string
  label: string
  category: string
  keywords: string[]
  enabled: boolean
}

export interface ExecutablePaletteCommand extends PaletteCommand { run: () => void | Promise<void> }

export interface PaletteContext {
  locale: 'zh' | 'en'
  sessions: Array<{ id: string; name: string; profile: string; status: string }>
  profiles: Array<{ id: string; name: string; description: string }>
  themes: Array<{ id: string; nameZh: string; nameEn: string }>
  daemonConnected: boolean
  creating: boolean
  inManagedWorkspace: boolean
  paneCount: number
  canUseTerminalTools: boolean
  observeEnabled: boolean
  observationBusy: boolean
  actions: {
    openSettings(tab: string): void
    setTheme(id: string): void
    selectSession(id: string): void
    createSession(profileID: string): void | Promise<void>
    split(direction: 'horizontal' | 'vertical'): void
    nextPane(): void
    closePane(): void
    search(): void
    quickSelect(): void
    refreshDig(): void
  }
}

export function buildPaletteCommands(context: PaletteContext): ExecutablePaletteCommand[] {
  const { locale, actions } = context
  const commands: ExecutablePaletteCommand[] = []
  const add = (id: string, label: string, category: string, keywords: string[], enabled: boolean, run: () => void | Promise<void>) => {
    commands.push({ id, label, category, keywords, enabled, run })
  }
  const zh = locale === 'zh'
  const settings = zh ? '设置' : 'Settings'
  add('settings:appearance', zh ? '打开外观设置' : 'Open appearance settings', settings,
    ['theme', 'color', 'language', 'font'], true, () => actions.openSettings('appearance'))
  add('settings:terminal', zh ? '打开终端设置' : 'Open terminal settings', settings,
    ['terminal', 'font', 'scrollback', 'cursor'], true, () => actions.openSettings('terminal'))
  add('settings:sessions', zh ? '打开会话设置' : 'Open session settings', settings,
    ['session', 'shell', 'directory', 'ssh'], true, () => actions.openSettings('sessions'))
  add('settings:quickselect', zh ? '配置快速选择规则' : 'Configure Quick Select rules', settings,
    ['quick select', 'regex', 'pattern', 'rules'], true, () => actions.openSettings('quickselect'))
  add('settings:keybindings', zh ? '打开快捷键设置' : 'Open keyboard shortcuts', settings,
    ['keyboard', 'shortcut', 'keybinding'], true, () => actions.openSettings('keybindings'))
  add('settings:help', zh ? '打开帮助' : 'Open help', settings,
    ['version', 'paths', 'updates'], true, () => actions.openSettings('help'))

  for (const theme of context.themes) add(`theme:${theme.id}`, zh ? `切换配色：${theme.nameZh}` : `Switch theme: ${theme.nameEn}`,
    zh ? '外观' : 'Appearance', ['theme', theme.nameZh, theme.nameEn, theme.id], true, () => actions.setTheme(theme.id))

  for (const session of context.sessions) add(`session:${session.id}`, `${zh ? '切换会话：' : 'Switch session: '}${session.name}`,
    zh ? '会话' : 'Sessions', [session.name, session.profile, session.status, session.id], true,
    () => actions.selectSession(session.id))

  for (const profile of context.profiles) add(`create:${profile.id}`,
    profile.id === 'ssh' ? (zh ? '新建 SSH 会话…' : 'New SSH session…') : `${zh ? '新建 ' : 'New '}${profile.name}`,
    zh ? '新建会话' : 'New session', [profile.name, profile.id, profile.description, 'create'],
    context.daemonConnected && !context.creating, () => actions.createSession(profile.id))

  const workspace = zh ? '工作区' : 'Workspace'
  add('workspace:split-right', zh ? '向右分屏' : 'Split right', workspace, ['pane', 'split', 'right'],
    context.inManagedWorkspace, () => actions.split('horizontal'))
  add('workspace:split-down', zh ? '向下分屏' : 'Split down', workspace, ['pane', 'split', 'down', 'vertical'],
    context.inManagedWorkspace, () => actions.split('vertical'))
  add('workspace:next-pane', zh ? '聚焦下一面板' : 'Focus next pane', workspace, ['pane', 'focus', 'next'],
    context.inManagedWorkspace && context.paneCount > 1, actions.nextPane)
  add('workspace:close-pane', zh ? '关闭当前面板（保留会话）' : 'Close pane (keep session)', workspace, ['pane', 'close', 'detach'],
    context.inManagedWorkspace && context.paneCount > 1, actions.closePane)

  const canUseTerminal = context.daemonConnected && context.canUseTerminalTools
  add('terminal:search', zh ? '搜索当前终端' : 'Search current terminal', zh ? '终端' : 'Terminal', ['find', 'search', 'output'],
    canUseTerminal, actions.search)
  add('terminal:quick-select', zh ? '快速选择并复制文字' : 'Quick Select and copy text', zh ? '终端' : 'Terminal',
    ['quick select', 'url', 'path', 'ip', 'hash', 'copy'], canUseTerminal, actions.quickSelect)
  add('dig:refresh', zh ? '刷新 Dig 进程树' : 'Refresh Dig process tree', 'Dig', ['process', 'observe', 'refresh'],
    context.observeEnabled && !context.observationBusy, actions.refreshDig)
  return commands
}

function scoreField(value: string, query: string): number | null {
  const text = value.toLocaleLowerCase()
  const term = query.toLocaleLowerCase()
  const direct = text.indexOf(term)
  if (direct >= 0) return direct + Math.max(0, text.length - term.length) / 100
  let queryIndex = 0
  let first = -1
  let previous = -1
  let gaps = 0
  for (let index = 0; index < text.length && queryIndex < term.length; index++) {
    if (text[index] !== term[queryIndex]) continue
    if (first < 0) first = index
    if (previous >= 0) gaps += index - previous - 1
    previous = index
    queryIndex++
  }
  return queryIndex === term.length ? first + gaps * 2 + text.length / 100 : null
}

export function filterPaletteCommands<T extends PaletteCommand>(commands: T[], query: string): T[] {
  const term = query.trim()
  if (!term) return commands
  return commands.map((command, order) => {
    const scores = [command.label, command.category, ...command.keywords]
      .map(value => scoreField(value, term)).filter((value): value is number => value !== null)
    return scores.length ? { command, order, score: Math.min(...scores) } : null
  }).filter((item): item is { command: T; order: number; score: number } => item !== null)
    .sort((left, right) => left.score - right.score || left.order - right.order)
    .map(item => item.command)
}

/** GUI preferences are private to this WebView; they do not change session ownership. */
export type BuiltinThemeId = 'nightfall' | 'graphite' | 'pine' | 'bluehour'
export type ThemeId = `builtin:${BuiltinThemeId}` | `user:${string}`
export type LanguageId = 'system' | 'zh' | 'en'
export type Locale = 'zh' | 'en'
export type FontId = 'cascadia' | 'consolas' | 'jetbrains' | 'fira' | 'monospace'
export type CursorStyle = 'block' | 'underline' | 'bar'
export type KeyboardMode = 'smart' | 'desktop' | 'native'
export type FocusBorderStyle = 'accent' | 'muted' | 'none'
export const fonts: Record<FontId, { name: string, family: string }> = {
  cascadia: { name: 'Cascadia Code', family: '"Cascadia Code", Consolas, monospace' },
  consolas: { name: 'Consolas', family: 'Consolas, monospace' },
  jetbrains: { name: 'JetBrains Mono', family: '"JetBrains Mono", Consolas, monospace' },
  fira: { name: 'Fira Code', family: '"Fira Code", Consolas, monospace' },
  monospace: { name: 'System Mono', family: 'monospace' },
}
export const scrollbackOptions = [1000, 5000, 10000, 25000] as const
export const lineHeightOptions = [1, 1.15, 1.25, 1.4] as const
export const scrollSpeedOptions = [1, 2, 3, 5] as const
export const paddingOptions = [6, 10, 14, 18] as const
export const refreshOptions = [0, 15, 30, 60] as const

export interface UIPreferences {
  theme: ThemeId
  customCSS: boolean
  fontSize: number
  uiFontSize: number
  fontFamily: FontId
  language: LanguageId
  sidebarWidth: number
  sidebarCollapsed: boolean
  managedRatio: number
  defaultDirectory: string
  defaultProfile: string
  scrollback: number
  lineHeight: number
  scrollSensitivity: number
  terminalPadding: number
  focusBorderStyle: FocusBorderStyle
  cursorStyle: CursorStyle
  keyboardMode: KeyboardMode
  cursorBlink: boolean
  rightClickSelectsWord: boolean
  activateNewSession: boolean
  observeEnabled: boolean
  observeRefreshSeconds: number
  showPid: boolean
  showStatusBar: boolean
}

export const defaults: UIPreferences = {
  theme: 'builtin:nightfall', customCSS: true, fontSize: 13, uiFontSize: 12, fontFamily: 'cascadia', language: 'system',
  sidebarWidth: 280, sidebarCollapsed: false, managedRatio: 0.42, defaultDirectory: '', defaultProfile: 'auto',
  scrollback: 5000, lineHeight: 1.25, scrollSensitivity: 1, terminalPadding: 14, focusBorderStyle: 'accent',
  cursorStyle: 'block', keyboardMode: 'smart', cursorBlink: true, rightClickSelectsWord: false,
  activateNewSession: true, observeEnabled: true,
  observeRefreshSeconds: 15, showPid: true, showStatusBar: true,
}

const key = 'diggerterm:preferences:v1'
export const themes: BuiltinThemeId[] = ['nightfall', 'graphite', 'pine', 'bluehour']

export type TerminalTheme = {
  background: string; foreground: string; cursor: string
  selectionBackground: string; black: string; red: string; green: string
  yellow: string; blue: string; magenta: string; cyan: string; white: string
  brightBlack?: string; brightRed?: string; brightGreen?: string; brightYellow?: string
  brightBlue?: string; brightMagenta?: string; brightCyan?: string; brightWhite?: string
  cursorAccent?: string; selectionForeground?: string; selectionInactiveBackground?: string
}
export type Palette = {
  zh: string
  en: string
  chips: [string, string, string]
  css: Record<string, string>
  terminal: TerminalTheme
}

const term = (background: string, foreground: string, cursor: string, selectionBackground: string): TerminalTheme => ({
  background, foreground, cursor, selectionBackground,
  black: '#20252c', red: '#d58d8b', green: '#9bc8a9', yellow: '#d3bd90',
  blue: '#92accf', magenta: '#b8a5c9', cyan: '#99c2bf', white: '#d9dde2',
})

export const palettes: Record<BuiltinThemeId, Palette> = {
  nightfall: {
    zh: '夜航', en: 'Nightfall', chips: ['#1b1f26', '#101318', '#bca58b'],
    css: {
      '--sidebar': '#1b1f26', '--canvas': '#101318', '--surface': '#282d36',
      '--surface-hover': '#2e343e', '--surface-active': '#353d49',
      '--border': '#343a44', '--border-faint': '#2b3039', '--text': '#dce0e6',
      '--muted': '#98a1ae', '--faint': '#7e8997', '--accent': '#bca58b',
      '--accent-bg': '#4b433a', '--accent-text': '#eadbc7', '--positive': '#94c8aa',
      '--negative': '#e7a8a4',
    },
    terminal: term('#101318', '#dce0e6', '#dcc2a0', '#6c657a'),
  },
  graphite: {
    zh: '石墨', en: 'Graphite', chips: ['#242528', '#18191c', '#b3b3b8'],
    css: {
      '--sidebar': '#242528', '--canvas': '#18191c', '--surface': '#303136',
      '--surface-hover': '#35363a', '--surface-active': '#404249',
      '--border': '#43454c', '--border-faint': '#35373c', '--text': '#dcdde0',
      '--muted': '#abadb4', '--faint': '#8a8e98', '--accent': '#b6b3ae',
      '--accent-bg': '#494744', '--accent-text': '#eeece9', '--positive': '#a9c6af',
      '--negative': '#dca6a2',
    },
    terminal: term('#18191c', '#e0e1e2', '#c9c2b8', '#777780'),
  },
  pine: {
    zh: '松影', en: 'Pine', chips: ['#232725', '#191d1b', '#9dae9c'],
    css: {
      '--sidebar': '#232725', '--canvas': '#191d1b', '--surface': '#2c322e',
      '--surface-hover': '#333a35', '--surface-active': '#3b453f',
      '--border': '#404943', '--border-faint': '#323a35', '--text': '#dee2dd',
      '--muted': '#a4afa5', '--faint': '#89978d', '--accent': '#a1b39e',
      '--accent-bg': '#454e45', '--accent-text': '#e0e9df', '--positive': '#a2c5a8',
      '--negative': '#d7a7a1',
    },
    terminal: term('#191d1b', '#dee2dc', '#b3c4b1', '#56615a'),
  },
  bluehour: {
    zh: '暮蓝', en: 'Blue Hour', chips: ['#1b2330', '#111622', '#97acc4'],
    css: {
      '--sidebar': '#1b2330', '--canvas': '#111622', '--surface': '#293346',
      '--surface-hover': '#2e3a4b', '--surface-active': '#39485f',
      '--border': '#3a495f', '--border-faint': '#2d394b', '--text': '#dce3ef',
      '--muted': '#9caac0', '--faint': '#7f8da2', '--accent': '#97acc4',
      '--accent-bg': '#3c5268', '--accent-text': '#e1eaf5', '--positive': '#9bc9b8',
      '--negative': '#e0a9ae',
    },
    terminal: term('#111622', '#dce3ef', '#b5c7d9', '#536781'),
  },
}

export function readPreferences(): UIPreferences {
  const result = { ...defaults }
  try {
    const saved = JSON.parse(localStorage.getItem(key) || '{}') as Partial<UIPreferences>
    if (typeof saved.theme === 'string') {
      const legacy = saved.theme as BuiltinThemeId
      if (themes.includes(legacy)) result.theme = `builtin:${legacy}`
      else if (/^(builtin:(nightfall|graphite|pine|bluehour)|user:[^/\\\x00-\x1f]{1,128})$/.test(saved.theme)) result.theme = saved.theme
    }
    if (saved.language && ['system', 'zh', 'en'].includes(saved.language)) result.language = saved.language
    if (typeof saved.fontSize === 'number' && Number.isFinite(saved.fontSize)) result.fontSize = Math.max(11, Math.min(20, Math.round(saved.fontSize)))
    if (typeof saved.uiFontSize === 'number' && Number.isFinite(saved.uiFontSize)) result.uiFontSize = Math.max(10, Math.min(16, Math.round(saved.uiFontSize)))
    if (typeof saved.sidebarWidth === 'number' && Number.isFinite(saved.sidebarWidth)) result.sidebarWidth = Math.max(216, Math.min(500, Math.round(saved.sidebarWidth)))
    if (saved.focusBorderStyle && ['accent', 'muted', 'none'].includes(saved.focusBorderStyle)) {
      result.focusBorderStyle = saved.focusBorderStyle
    } else {
      const legacy = (saved as Partial<UIPreferences> & { panePromptStyle?: string }).panePromptStyle
      if (legacy === 'current') result.focusBorderStyle = 'accent'
      else if (legacy === 'subtle') result.focusBorderStyle = 'muted'
      else if (legacy === 'none') result.focusBorderStyle = 'none'
    }
    if (typeof saved.managedRatio === 'number' && Number.isFinite(saved.managedRatio)) result.managedRatio = Math.max(0.2, Math.min(0.8, saved.managedRatio))
    if (typeof saved.defaultDirectory === 'string') result.defaultDirectory = saved.defaultDirectory.slice(0, 512)
    if (saved.fontFamily && Object.hasOwn(fonts, saved.fontFamily)) result.fontFamily = saved.fontFamily
    if (saved.cursorStyle && ['block', 'underline', 'bar'].includes(saved.cursorStyle)) result.cursorStyle = saved.cursorStyle
    if (saved.keyboardMode && ['smart', 'desktop', 'native'].includes(saved.keyboardMode)) result.keyboardMode = saved.keyboardMode
    if (typeof saved.defaultProfile === 'string' && /^[a-z][a-z0-9_-]{0,31}$/.test(saved.defaultProfile)) result.defaultProfile = saved.defaultProfile
    if (saved.scrollback && scrollbackOptions.includes(saved.scrollback as typeof scrollbackOptions[number])) result.scrollback = saved.scrollback
    if (saved.lineHeight && lineHeightOptions.includes(saved.lineHeight as typeof lineHeightOptions[number])) result.lineHeight = saved.lineHeight
    if (saved.scrollSensitivity && scrollSpeedOptions.includes(saved.scrollSensitivity as typeof scrollSpeedOptions[number])) result.scrollSensitivity = saved.scrollSensitivity
    if (saved.terminalPadding && paddingOptions.includes(saved.terminalPadding as typeof paddingOptions[number])) result.terminalPadding = saved.terminalPadding
    if (saved.observeRefreshSeconds !== undefined && refreshOptions.includes(saved.observeRefreshSeconds as typeof refreshOptions[number])) result.observeRefreshSeconds = saved.observeRefreshSeconds
    for (const name of ['customCSS', 'cursorBlink', 'rightClickSelectsWord', 'activateNewSession',
      'observeEnabled', 'showPid', 'showStatusBar', 'sidebarCollapsed'] as const) {
      if (typeof saved[name] === 'boolean') result[name] = saved[name]
    }
  } catch { /* Storage can be disabled: use defaults. */ }
  return result
}
export function writePreferences(value: UIPreferences) {
  try { localStorage.setItem(key, JSON.stringify(value)) } catch { /* Optional. */ }
}
export function chosenLocale(language: LanguageId): Locale {
  if (language !== 'system') return language
  return typeof navigator !== 'undefined' && navigator.language.toLowerCase().startsWith('zh') ? 'zh' : 'en'
}

const text = {
  zh: {
    workspace: '终端工作区', search: '搜索会话或进程', managed: '我的会话', observed: '外部观察 · 只读',
    noSessions: '还没有会话', searchNone: '没有匹配的会话', noObserved: '未发现相关外部进程',
    observedSearchNone: '没有匹配的外部进程', scanning: '正在扫描…', windowsOnly: '仅 Windows 支持进程扫描',
    sshHost: 'Host 别名 / user@host', sshPort: '端口（可选）', connect: '连接', connectHint: '认证与主机指纹由 OpenSSH 处理',
    refresh: '刷新', new: '新建终端', rename: '重命名', terminate: '终止', remove: '移除',
    save: '保存', cancel: '取消', running: '运行中', exited: '已退出', observedStatus: '只读',
    settings: '设置', settingsTitle: '界面设置', settingsDescription: '仅影响界面和之后新建的本地 Shell。',
    colors: '配色', fonts: '终端字体大小', uiFontSize: '界面字号', language: '界面语言', system: '跟随系统',
    defaultDirectory: '新建 Shell 默认路径',
    directoryPlaceholder: '留空使用默认路径', resetLayout: '重置分栏大小', close: '关闭设置',
    widthHandle: '调整左侧栏宽度', heightHandle: '调整上下区域比例',
    collapseSidebar: '收起侧栏', expandSidebar: '展开侧栏', sessionDiagnostics: '会话诊断',
    errorDaemon: '后台 Daemon 版本较旧。请先完成正在运行的会话，停止旧 Daemon，再重新打开。',
    processGone: '进程已不可用', processGoneDetail: '进程可能已经退出或无法读取。请刷新观察列表。',
    readOnly: '外部进程 · 只读', source: '来源', pid: '进程 ID', parent: '父进程 ID',
    kind: '类型', started: '启动时间', inferred: '推断', unknown: '未知',
    readOnlyText: '只显示进程元信息。此终端不属于 DiggerTerm，不能查看终端画面、发送输入或终止信号。Windows 与 WSL 的 PID 命名空间互不相同。',
    limited: '只显示前 160 项', readonlyFooter: '外部进程仅可观察，不可操作',
    activeCount: '{count} 个运行中', observedCount: '{count} 个观察对象',
    emptyTitle: '一个窗口，管理多个终端。',
    emptyDesc: '从左侧选择 Shell 并打开',
    emptyHint: '退出 DiggerTerm 后会话保留，外部进程仅只读',
    confirmTerminate: '终止 {name} (PID {pid})？该终端将结束。',
    sshRequired: '请输入 SSH 主机别名或 user@hostname。',
    invalidPort: 'SSH 端口必须为 1–65535，或留空使用 SSH 配置。',
    noSelection: '未选择会话', terminal: '终端', noPath: '此路径无效时创建会话会提示错误。',
    loadingAliases: '{count} 个本地 SSH Host 别名', noAgentControls: '不支持控制外部进程',
    noParent: '未知',
    kindTerminal: '终端', kindAgent: 'Agent', kindDescendant: '子进程', kindOther: '进程', observeFailed: '进程观察失败',
    appearanceTab: '外观', terminalTab: '终端', sessionTab: '会话', monitorTab: '观察',
    fontFamily: '字体', fontSizeShort: '字号', lineHeight: '行距', scrollback: '回滚行数',
    cursorStyle: '光标形状', cursorBlock: '方块', cursorUnderline: '下划线', cursorBar: '竖线',
    keyboardMode: '快捷键模式', keyboardSmart: '智能（推荐）', keyboardDesktop: '桌面复制粘贴', keyboardNative: '终端原生',
    cursorBlink: '光标闪烁', scrollSensitivity: '滚轮速度', rightClickWord: '右键选词',
    terminalPadding: '终端留白',
    profileDefault: '默认 Shell', automatic: '自动选择', activateNewSession: '新建后切换',
    observeEnabled: '显示外部进程',
    refreshSeconds: '自动扫描', scanManual: '仅手动', seconds: '{count} 秒',
    showPid: '显示 PID', showStatusBar: '显示侧栏计数',
    focusBorderStyle: '选中终端边框样式', focusBorderAccent: '强调色边框',
    focusBorderMuted: '灰色边框', focusBorderNone: '无提示',
    resetAppearance: '恢复默认外观', resetTerminal: '恢复终端默认值',
    resetSessions: '恢复会话默认值', resetObserved: '恢复观察默认值',

  },
  en: {
    workspace: 'Terminal workspace', search: 'Filter sessions / processes', managed: 'My sessions', observed: 'Observed · read only',
    noSessions: 'No sessions yet', searchNone: 'No matching sessions', noObserved: 'No related external processes',
    observedSearchNone: 'No matching external processes', scanning: 'Scanning…', windowsOnly: 'Windows process discovery only',
    sshHost: 'Host alias / user@host', sshPort: 'Port (optional)', connect: 'Connect', connectHint: 'OpenSSH handles authentication and host verification',
    refresh: 'Refresh', new: 'New terminal', rename: 'Rename', terminate: 'Terminate', remove: 'Remove',
    save: 'Save', cancel: 'Cancel', running: 'Running', exited: 'Exited', observedStatus: 'Read only',
    settings: 'Settings', settingsTitle: 'Appearance & preferences', settingsDescription: 'Affects the GUI and future local shell sessions only.',
    colors: 'Color theme', fonts: 'Terminal font size', uiFontSize: 'Interface font size', language: 'Interface language', system: 'Follow system',
    defaultDirectory: 'New shell working directory',
    directoryPlaceholder: 'Default when empty', resetLayout: 'Reset splitter sizes', close: 'Close settings',
    widthHandle: 'Resize sidebar', heightHandle: 'Resize managed / observed panes',
    collapseSidebar: 'Collapse sidebar', expandSidebar: 'Expand sidebar', sessionDiagnostics: 'Session diagnostics',
    errorDaemon: 'An older daemon is running. Finish sessions, stop it, then reopen the app.',
    processGone: 'Process no longer found', processGoneDetail: 'It may have exited or become inaccessible. Refresh to update.',
    readOnly: 'External process · read only', source: 'Source', pid: 'Process ID', parent: 'Parent PID',
    kind: 'Classification', started: 'Started', inferred: 'inferred', unknown: 'Unknown',
    readOnlyText: 'Process metadata only. DiggerTerm does not own this terminal or read its screen, send input, or terminate it. Windows and WSL PIDs use distinct namespaces.',
    limited: 'First 160 results shown', readonlyFooter: 'Observed processes cannot be controlled',
    activeCount: '{count} running', observedCount: '{count} observed',
    emptyTitle: 'Your terminals, one workspace.',
    emptyDesc: 'Select a Shell on the left and open it',
    emptyHint: 'Sessions remain after quitting DiggerTerm; external processes are read-only',
    confirmTerminate: 'Terminate {name} (PID {pid})? This ends the terminal.',
    sshRequired: 'Enter an SSH host alias or user@hostname.',
    invalidPort: 'SSH port must be 1–65535 or left blank.',
    noSelection: 'No session selected', terminal: 'Terminal', noPath: 'An invalid directory will fail when creating the shell.',
    loadingAliases: '{count} SSH Host aliases available', noAgentControls: 'Cannot control observed processes',
    noParent: 'Unknown',
    kindTerminal: 'Terminal', kindAgent: 'Agent', kindDescendant: 'Child process', kindOther: 'Process', observeFailed: 'Process observation failed',
    appearanceTab: 'Appearance', terminalTab: 'Terminal', sessionTab: 'Sessions', monitorTab: 'Observed',
    fontFamily: 'Font', fontSizeShort: 'Size', lineHeight: 'Line height', scrollback: 'Scrollback',
    cursorStyle: 'Cursor shape', cursorBlock: 'Block', cursorUnderline: 'Underline', cursorBar: 'Bar',
    keyboardMode: 'Shortcuts', keyboardSmart: 'Smart (recommended)', keyboardDesktop: 'Desktop clipboard', keyboardNative: 'Terminal native',
    cursorBlink: 'Blink cursor', scrollSensitivity: 'Wheel speed', rightClickWord: 'Right-click word',
    terminalPadding: 'Terminal padding',
    profileDefault: 'Default shell', automatic: 'Automatic', activateNewSession: 'Select on create',
    observeEnabled: 'Show observed',
    refreshSeconds: 'Auto refresh', scanManual: 'Manual only', seconds: '{count}s',
    showPid: 'Show PID', showStatusBar: 'Show sidebar counts',
    focusBorderStyle: 'Selected terminal border', focusBorderAccent: 'Accent border',
    focusBorderMuted: 'Medium gray 2px border', focusBorderNone: 'No hint',
    resetAppearance: 'Reset appearance', resetTerminal: 'Reset terminal',
    resetSessions: 'Reset sessions', resetObserved: 'Reset observed',

  },
} as const
export type TextKey = keyof typeof text.en
export function translate(locale: Locale, label: TextKey, replacements: Record<string, string | number> = {}): string {
  let result: string = text[locale][label]
  for (const [name, value] of Object.entries(replacements)) result = result.replace('{' + name + '}', String(value))
  return result
}

import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { Browser } from '@wailsio/runtime'
import packageInfo from '../../package.json'
import { KeybindingsPath, ReadKeybindings } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/keybindingservice'
import { QuickSelectPath } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/quickselectservice'
import { ListThemes, OpenThemeDirectory } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/themeservice'
import { SetUILanguage } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/sessionservice'
import {
  chosenLocale, defaults, fonts, lineHeightOptions, paddingOptions,
  readPreferences, refreshOptions, scrollbackOptions, scrollSpeedOptions,
  translate, writePreferences,
  type LanguageId, type TextKey, type ThemeId,
} from '../uiPreferences'
import { defaultBindings, mergeBindings, parseBindings, type Binding } from '../keybindings'
import { themeRegistry, themeVariables, type ThemeOption } from '../userThemes'
import type { SettingsTab } from '../components/SettingsDialog.vue'
import {
  AUTO_UPDATE_CHECK_DELAY_MS, automaticUpdateCheckDue, compareVersions, parseLatestRelease,
} from '../updates'

export interface UseSettingsOptions {
  onError?: (error: unknown) => void
}

export function useSettings(options: UseSettingsOptions = {}) {
  const onError = options.onError ?? (() => {})
  const preferences = ref(readPreferences())
  const locale = computed(() => chosenLocale(preferences.value.language))
  const t = (key: TextKey, replacements?: Record<string, string | number>) => translate(locale.value, key, replacements)
  const themeOptions = ref<ThemeOption[]>(themeRegistry([]))
  const chosenTheme = computed(() => themeOptions.value.find(theme => theme.id === preferences.value.theme) ?? themeOptions.value[0]!)
  const palette = computed(() => chosenTheme.value.palette)
  const themeDirectory = ref('')
  const keybindingsPath = ref('')
  const quickSelectPath = ref('')
  const themeErrors = ref<{ file: string; message: string }[]>([])
  const globalCSS = ref('')
  const themeRevision = ref(0)
  const settingsOpen = ref(false)
  const settingsTab = ref<SettingsTab>('appearance')
  const settingsTabLabels = computed<Record<SettingsTab, string>>(() => ({
    focus: locale.value === 'zh' ? '专注' : 'Focus',
    appearance: t('appearanceTab'), terminal: t('terminalTab'), sessions: t('sessionTab'),
    observed: t('monitorTab'), keybindings: locale.value === 'zh' ? '快捷键' : 'Keys',
    quickselect: locale.value === 'zh' ? '快速选择' : 'Quick Select',
    help: locale.value === 'zh' ? '帮助' : 'Help',
  }))
  const appVersion = packageInfo.version
  const updateStatus = ref('')
  const updateAvailable = ref(false)
  const checkingUpdate = ref(false)
  const isMac = /Mac/i.test(navigator.userAgent)
  const bindings = ref<Binding[]>(defaultBindings(isMac))
  let themeStyle: HTMLStyleElement | undefined
  let automaticUpdateTimer: ReturnType<typeof setTimeout> | undefined
  let updateRequest: AbortController | undefined
  const automaticUpdateCheckKey = 'diggerterm:last-auto-update-check'

  async function applyTheme() {
    themeStyle ??= document.createElement('style')
    const variables = Object.entries(themeVariables(palette.value)).map(([key, value]) => `${key}:${value}`).join(';')
    themeStyle.textContent = `.layout[data-dt-theme]{${variables}}\n` +
      (preferences.value.customCSS ? chosenTheme.value.css + '\n' + globalCSS.value : '')
    document.head.append(themeStyle)
    await nextTick()
    themeRevision.value++
  }

  watch([chosenTheme, globalCSS, () => preferences.value.customCSS], () => { void applyTheme() })
  watch(preferences, next => writePreferences(next), { deep: true })
  watch(locale, value => {
    void SetUILanguage(value).catch(() => {
      // The GUI remains functional if the platform does not expose a tray.
    })
  })
  watch(() => preferences.value.automaticUpdateChecks, enabled => {
    if (enabled) scheduleAutomaticUpdateCheck()
    else if (automaticUpdateTimer) {
      clearTimeout(automaticUpdateTimer)
      automaticUpdateTimer = undefined
    }
  })

  async function refreshThemes() {
    try {
      const catalog = await ListThemes()
      themeDirectory.value = catalog.directory
      themeErrors.value = catalog.errors ?? []
      themeOptions.value = themeRegistry(catalog.themes ?? [])
      globalCSS.value = catalog.customCSS
      if (!themeOptions.value.some(theme => theme.id === preferences.value.theme)) {
        preferences.value.theme = defaults.theme
        themeErrors.value.push({ file: '', message: locale.value === 'zh' ? '所选主题已不可用，已恢复默认配色。' : 'Selected theme unavailable; restored default.' })
      }
    } catch (error) { onError(error) }
  }

  async function reloadBindings() {
    try {
      const source = await ReadKeybindings()
      const overrides = source ? parseBindings(source, isMac) : []
      bindings.value = mergeBindings(overrides, isMac)
    } catch (error) { onError(error) }
  }

  function initialize() {
    void reloadBindings()
    void KeybindingsPath().then(value => { keybindingsPath.value = value }).catch(onError)
    void QuickSelectPath().then(value => { quickSelectPath.value = value }).catch(onError)
    void applyTheme()
    void refreshThemes()
    scheduleAutomaticUpdateCheck()
    void SetUILanguage(locale.value).catch(() => {
      // The GUI remains functional if the platform does not expose a tray.
    })
  }

  function resetLayout() {
    preferences.value.sidebarWidth = defaults.sidebarWidth
    preferences.value.sidebarCollapsed = defaults.sidebarCollapsed
    preferences.value.managedRatio = defaults.managedRatio
  }

  function updateTheme(value: ThemeId) { preferences.value.theme = value }
  function updateLanguage(event: Event) { preferences.value.language = (event.target as HTMLSelectElement).value as LanguageId }

  function resetSection() {
    switch (settingsTab.value) {
      case 'appearance':
        preferences.value.theme = defaults.theme
        preferences.value.customCSS = false
        preferences.value.language = defaults.language
        preferences.value.uiFontSize = defaults.uiFontSize
        resetLayout()
        break
      case 'terminal':
        for (const key of ['fontSize', 'fontFamily', 'scrollback', 'lineHeight', 'scrollSensitivity', 'keyboardMode',
          'terminalPadding', 'focusBorderStyle', 'cursorStyle', 'cursorBlink', 'rightClickSelectsWord'] as const) {
          Object.assign(preferences.value, { [key]: defaults[key] })
        }
        break
      case 'focus':
        for (const key of ['fontSize', 'fontFamily', 'lineHeight', 'cursorStyle', 'cursorBlink', 'terminalPadding'] as const) {
          Object.assign(preferences.value, { [key]: defaults[key] })
        }
        break
      case 'sessions':
        for (const key of ['defaultProfile', 'defaultDirectory', 'activateNewSession', 'sessionActivityEnabled'] as const) {
          Object.assign(preferences.value, { [key]: defaults[key] })
        }
        break
      case 'observed':
        for (const key of ['observeEnabled', 'observeRefreshSeconds', 'showPid', 'showStatusBar'] as const) {
          Object.assign(preferences.value, { [key]: defaults[key] })
        }
        break
    }
  }

  async function checkForUpdates(automatic = false) {
    if (checkingUpdate.value) return
    if (!automatic && automaticUpdateTimer) {
      clearTimeout(automaticUpdateTimer)
      automaticUpdateTimer = undefined
    }
    checkingUpdate.value = true
    if (!automatic) updateStatus.value = ''
    const controller = new AbortController()
    updateRequest = controller
    const timeout = setTimeout(() => controller.abort(), 8_000)
    try {
      const response = await fetch('https://api.github.com/repos/xswt442-cmd/DiggerTerm/releases/latest', {
        headers: { Accept: 'application/vnd.github+json' },
        cache: 'no-store',
        signal: controller.signal,
      })
      if (!response.ok) throw new Error(`HTTP ${response.status}`)
      const release = parseLatestRelease(await response.json())
      updateAvailable.value = compareVersions(release.version, appVersion) > 0
      updateStatus.value = updateAvailable.value
        ? (locale.value === 'zh' ? `发现新版本 ${release.tag}` : `Version ${release.tag} is available`)
        : (locale.value === 'zh' ? '当前已是最新版本' : 'You are up to date')
    } catch {
      if (!automatic) updateStatus.value = locale.value === 'zh' ? '检查更新失败' : 'Could not check for updates'
    } finally {
      clearTimeout(timeout)
      if (updateRequest === controller) updateRequest = undefined
      try { localStorage.setItem(automaticUpdateCheckKey, String(Date.now())) } catch { /* Optional. */ }
      checkingUpdate.value = false
    }
  }

  function scheduleAutomaticUpdateCheck() {
    if (automaticUpdateTimer) {
      clearTimeout(automaticUpdateTimer)
      automaticUpdateTimer = undefined
    }
    if (!preferences.value.automaticUpdateChecks) return
    let lastCheckedAt: number | null = null
    try {
      const value = localStorage.getItem(automaticUpdateCheckKey)
      lastCheckedAt = value === null ? null : Number(value)
    } catch { /* Storage can be disabled: check once this session. */ }
    if (!automaticUpdateCheckDue(lastCheckedAt)) return
    automaticUpdateTimer = setTimeout(() => {
      automaticUpdateTimer = undefined
      void checkForUpdates(true)
    }, AUTO_UPDATE_CHECK_DELAY_MS)
  }

  async function openReleasePage() {
    try { await Browser.OpenURL('https://github.com/xswt442-cmd/DiggerTerm/releases/latest') }
    catch (error) { onError(error) }
  }

  async function openThemeDirectory() {
    try { await OpenThemeDirectory() }
    catch (error) { onError(error) }
  }

  function themeConfigPath(fileName: string) {
    if (!themeDirectory.value) return ''
    const separator = themeDirectory.value.includes('\\') ? '\\' : '/'
    return `${themeDirectory.value.replace(/[\\/]+$/, '')}${separator}${fileName}`
  }

  const helpConfigPaths = computed(() => [
    { label: locale.value === 'zh' ? '主题 JSON 文件' : 'Theme JSON file', path: themeConfigPath(locale.value === 'zh' ? '<主题名>.json' : '<theme-name>.json') },
    { label: locale.value === 'zh' ? '主题 CSS 文件' : 'Theme CSS file', path: themeConfigPath(locale.value === 'zh' ? '<主题名>.css' : '<theme-name>.css') },
    { label: locale.value === 'zh' ? '全局 CSS 覆盖' : 'Global CSS override', path: themeConfigPath('custom.css') },
    { label: locale.value === 'zh' ? '快捷键配置文件' : 'Shortcut config file', path: keybindingsPath.value },
    { label: locale.value === 'zh' ? '快速选择规则' : 'Quick Select rules', path: quickSelectPath.value },
  ])
  const fontChoices = Object.entries(fonts).map(([id, font]) => ({ id, name: font.name }))

  onBeforeUnmount(() => {
    if (automaticUpdateTimer) clearTimeout(automaticUpdateTimer)
    updateRequest?.abort()
    themeStyle?.remove()
  })

  return {
    preferences, locale, t, defaults, fonts, scrollbackOptions, lineHeightOptions,
    scrollSpeedOptions, paddingOptions, refreshOptions, themeOptions, chosenTheme, palette,
    themeDirectory, keybindingsPath, themeErrors, themeRevision, settingsOpen, settingsTab,
    settingsTabLabels, appVersion, updateStatus, updateAvailable, checkingUpdate, isMac,
    bindings, helpConfigPaths, fontChoices, initialize, applyTheme, refreshThemes, reloadBindings,
    resetLayout, resetSection, updateTheme, updateLanguage, checkForUpdates, openReleasePage,
    openThemeDirectory,
  }
}

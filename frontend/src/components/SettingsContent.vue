<script setup lang="ts">
import KeybindingsSettings from './KeybindingsSettings.vue'
import type { Profile } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/session/models'
import type { Binding } from '../keybindings'
import type { TextKey, UIPreferences } from '../uiPreferences'
import type { ThemeOption } from '../userThemes'
import type { SettingsTab } from './SettingsDialog.vue'

const preferences = defineModel<UIPreferences>('preferences', { required: true })

const props = defineProps<{
  tab: SettingsTab
  locale: 'zh' | 'en'
  translate: (key: TextKey, replacements?: Record<string, string | number>) => string
  themeOptions: ThemeOption[]
  themeDirectory: string
  themeErrors: { file: string; message: string }[]
  fontChoices: { id: string; name: string }[]
  scrollbackOptions: readonly number[]
  lineHeightOptions: readonly number[]
  scrollSpeedOptions: readonly number[]
  paddingOptions: readonly number[]
  refreshOptions: readonly number[]
  profiles: Profile[]
  reconnecting: boolean
  bindings: Binding[]
  isMac: boolean
  appVersion: string
  helpConfigPaths: { label: string; path: string }[]
  checkingUpdate: boolean
  updateStatus: string
  updateAvailable: boolean
}>()

const emit = defineEmits<{
  'theme-change': [themeId: ThemeOption['id']]
  'language-change': [event: Event]
  'refresh-themes': []
  'open-theme-directory': []
  'reset-layout': []
  'reset-section': []
  reconnect: []
  'keybindings-change': [bindings: Binding[]]
  'check-updates': []
  'open-release-page': []
}>()

const t = props.translate
</script>

<template>
  <template v-if="tab === 'appearance'">
    <div class="setting-block">
      <label class="setting-label">{{ t('colors') }}</label>
      <div class="theme-options">
        <button v-for="theme in themeOptions" :key="theme.id" type="button" class="theme-option"
          :class="{ chosen: preferences.theme === theme.id }"
          :aria-pressed="preferences.theme === theme.id"
          :title="theme.id"
          @click="emit('theme-change', theme.id)">
          <span class="theme-chips" aria-hidden="true">
            <span v-for="(color, index) in theme.palette.chips" :key="index" :style="{ background: color }" />
          </span>
          <span>{{ theme.palette[locale] }}</span>
        </button>
      </div>
      <button type="button" class="reset-layout" @click="emit('refresh-themes')">{{ t('refresh') }}</button>
      <div class="theme-config-row">
        <small class="theme-config-path" :title="themeDirectory">{{ themeDirectory }}</small>
      </div>
      <div class="theme-css-row">
        <label class="setting-toggle"><span>{{ locale === 'zh' ? '启用主题 CSS 和 custom.css' : 'Enable theme CSS and custom.css' }}</span>
          <input v-model="preferences.customCSS" type="checkbox" /></label>
        <button type="button" class="reset-layout" @click="emit('open-theme-directory')">{{ locale === 'zh' ? '前往配置' : 'Open config' }}</button>
      </div>
      <p v-for="(error, index) in themeErrors" :key="index" role="alert" class="setting-error">{{ error.file }}: {{ error.message }}</p>
    </div>
    <div class="setting-block">
      <div class="setting-label setting-split">
        <label for="ui-font-size-slider">{{ t('uiFontSize') }}</label>
        <output for="ui-font-size-slider">{{ preferences.uiFontSize }} px</output>
      </div>
      <input id="ui-font-size-slider" v-model.number="preferences.uiFontSize"
        class="font-slider" type="range" min="10" max="16" step="1" />
    </div>
    <label class="setting-row">
      <span>{{ t('language') }}</span>
      <select :value="preferences.language" @change="emit('language-change', $event)">
        <option value="system">{{ t('system') }}</option>
        <option value="zh">简体中文</option>
        <option value="en">English</option>
      </select>
    </label>
    <button type="button" class="reset-layout" @click="emit('reset-layout')">{{ t('resetLayout') }}</button>
  </template>

  <template v-else-if="tab === 'terminal'">
    <label class="setting-row">
      <span>{{ t('fontFamily') }}</span>
      <select v-model="preferences.fontFamily">
        <option v-for="font in fontChoices" :key="font.id" :value="font.id">{{ font.name }}</option>
      </select>
    </label>
    <div class="setting-block">
      <div class="setting-label setting-split">
        <label for="font-size-slider">{{ t('fontSizeShort') }}</label>
        <output for="font-size-slider">{{ preferences.fontSize }} px</output>
      </div>
      <input id="font-size-slider" v-model.number="preferences.fontSize"
        class="font-slider" type="range" min="11" max="20" step="1" />
    </div>
    <div class="setting-grid">
      <label class="setting-stack">
        <span>{{ t('scrollback') }}</span>
        <select v-model.number="preferences.scrollback">
          <option v-for="n in scrollbackOptions" :key="n" :value="n">{{ n.toLocaleString() }}</option>
        </select>
      </label>
      <label class="setting-stack">
        <span>{{ t('lineHeight') }}</span>
        <select v-model.number="preferences.lineHeight">
          <option v-for="n in lineHeightOptions" :key="n" :value="n">{{ n.toFixed(2).replace(/0+$/, '').replace(/\.$/, '') }}×</option>
        </select>
      </label>
      <label class="setting-stack">
        <span>{{ t('cursorStyle') }}</span>
        <select v-model="preferences.cursorStyle">
          <option value="block">{{ t('cursorBlock') }}</option>
          <option value="underline">{{ t('cursorUnderline') }}</option>
          <option value="bar">{{ t('cursorBar') }}</option>
        </select>
      </label>
      <label class="setting-stack">
        <span>{{ t('scrollSensitivity') }}</span>
        <select v-model.number="preferences.scrollSensitivity">
          <option v-for="n in scrollSpeedOptions" :key="n" :value="n">{{ n }}×</option>
        </select>
      </label>
      <label class="setting-stack">
        <span>{{ t('terminalPadding') }}</span>
        <select v-model.number="preferences.terminalPadding">
          <option v-for="n in paddingOptions" :key="n" :value="n">{{ n }} px</option>
        </select>
      </label>
    </div>
    <label class="setting-row">
      <span>{{ t('panePromptStyle') }}</span>
      <select v-model="preferences.panePromptStyle">
        <option value="current">{{ t('panePromptCurrent') }}</option>
        <option value="subtle">{{ t('panePromptSubtle') }}</option>
        <option value="none">{{ t('panePromptNone') }}</option>
      </select>
    </label>
    <div class="toggle-grid">
      <label class="setting-toggle"><span>{{ t('cursorBlink') }}</span>
        <input v-model="preferences.cursorBlink" type="checkbox" /></label>
      <label class="setting-toggle"><span>{{ t('rightClickWord') }}</span>
        <input v-model="preferences.rightClickSelectsWord" type="checkbox" /></label>
    </div>
  </template>

  <template v-else-if="tab === 'sessions'">
    <button type="button" class="reset-layout" :disabled="reconnecting" @click="emit('reconnect')">{{ locale === 'zh' ? '重连 Daemon，保留会话' : 'Reconnect daemon, keep sessions' }}</button>
    <label class="setting-row">
      <span>{{ t('profileDefault') }}</span>
      <select v-model="preferences.defaultProfile">
        <option value="auto">{{ t('automatic') }}</option>
        <option v-for="profile in profiles.filter(item => item.id !== 'ssh')" :key="profile.id" :value="profile.id">{{ profile.name }}</option>
      </select>
    </label>
    <label class="setting-toggle"><span>{{ t('activateNewSession') }}</span>
      <input v-model="preferences.activateNewSession" type="checkbox" /></label>
    <div class="setting-block">
      <label class="setting-label" for="default-directory">{{ t('defaultDirectory') }}</label>
      <input id="default-directory" v-model="preferences.defaultDirectory"
        class="directory-input" type="text" spellcheck="false" autocomplete="off"
        maxlength="512" :placeholder="t('directoryPlaceholder')" />
    </div>
  </template>

  <template v-else-if="tab === 'observed'">
    <label class="setting-toggle"><span>{{ t('observeEnabled') }}</span>
      <input v-model="preferences.observeEnabled" type="checkbox" /></label>
    <label class="setting-row">
      <span>{{ t('refreshSeconds') }}</span>
      <select v-model.number="preferences.observeRefreshSeconds" :disabled="!preferences.observeEnabled">
        <option v-for="n in refreshOptions" :key="n" :value="n">
          {{ n === 0 ? t('scanManual') : t('seconds', { count: n }) }}
        </option>
      </select>
    </label>
    <label class="setting-toggle"><span>{{ t('showPid') }}</span>
      <input v-model="preferences.showPid" type="checkbox" /></label>
    <label class="setting-toggle"><span>{{ t('showStatusBar') }}</span>
      <input v-model="preferences.showStatusBar" type="checkbox" /></label>
  </template>

  <template v-else-if="tab === 'keybindings'">
    <label class="setting-row">
      <span>{{ t('keyboardMode') }}</span>
      <select v-model="preferences.keyboardMode">
        <option value="smart">{{ t('keyboardSmart') }}</option>
        <option value="desktop">{{ t('keyboardDesktop') }}</option>
        <option value="native">{{ t('keyboardNative') }}</option>
      </select>
    </label>
    <KeybindingsSettings :locale="locale" :bindings="bindings" :is-mac="isMac" @changed="emit('keybindings-change', $event)" />
  </template>

  <template v-else-if="tab === 'help'">
    <div class="setting-block">
      <label class="setting-label">{{ locale === 'zh' ? '应用版本' : 'App version' }}</label>
      <span class="help-value">{{ appVersion }}</span>
    </div>
    <div class="setting-block">
      <label class="setting-label">{{ locale === 'zh' ? '主题配置目录' : 'Theme config directory' }}</label>
      <span class="help-path" :title="themeDirectory">{{ themeDirectory }}</span>
    </div>
    <div v-for="item in helpConfigPaths" :key="item.label" class="setting-block">
      <label class="setting-label">{{ item.label }}</label>
      <span class="help-path" :title="item.path">{{ item.path }}</span>
    </div>
    <button type="button" class="reset-layout" :disabled="checkingUpdate" @click="emit('check-updates')">
      {{ checkingUpdate ? (locale === 'zh' ? '正在检查…' : 'Checking…') : (locale === 'zh' ? '检查更新' : 'Check for updates') }}
    </button>
    <p v-if="updateStatus" class="update-status" role="status">{{ updateStatus }}</p>
    <button v-if="updateAvailable" type="button" class="reset-layout" @click="emit('open-release-page')">
      {{ locale === 'zh' ? '查看下载' : 'View download' }}
    </button>
  </template>

  <button v-if="!['keybindings', 'help'].includes(tab)" type="button" class="reset-layout compact-reset" @click="emit('reset-section')">
    {{ t(tab === 'appearance' ? 'resetAppearance' : tab === 'terminal' ? 'resetTerminal' : tab === 'sessions' ? 'resetSessions' : 'resetObserved') }}
  </button>
</template>

<style scoped>
.setting-block { display: flex; flex-direction: column; gap: 5px; }
.setting-label { color: var(--text); font-size: calc(10px * var(--ui-font-scale)); font-weight: 600; }
.setting-split { display: flex; align-items: center; justify-content: space-between; }
.setting-split output { font-size: calc(11px * var(--ui-font-scale)); color: var(--accent); font-weight: 500; }
.theme-options { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 5px; }
.theme-option { display: flex; align-items: center; gap: 5px; min-width: 0; height: 29px; padding: 3px 5px; color: var(--muted); background: var(--surface); border: 1px solid var(--border); border-radius: 5px; font-size: calc(10px * var(--ui-font-scale)); }
.theme-option:hover { color: var(--text); }
.theme-option.chosen { color: var(--text); border-color: var(--accent); }
.theme-chips { display: flex; flex: 0 0 auto; overflow: hidden; width: 32px; height: 15px; border-radius: 3px; border: 1px solid #ffffff22; }
.theme-chips span { flex: 1; }
.theme-config-row { min-width: 0; }
.theme-config-path { display: block; overflow: hidden; color: var(--faint); font-size: calc(8px * var(--ui-font-scale)); line-height: 1.2; text-overflow: ellipsis; white-space: nowrap; }
.theme-css-row { display: flex; align-items: center; gap: 5px; }
.theme-css-row .setting-toggle { min-width: 0; flex: 1 1 auto; font-size: calc(9px * var(--ui-font-scale)); }
.setting-error { margin: 0; color: var(--negative); font-size: calc(9px * var(--ui-font-scale)); }
.help-value { font-size: calc(10px * var(--ui-font-scale)); }
.help-path { overflow-wrap: anywhere; color: var(--text); font-family: Consolas, monospace; font-size: calc(9px * var(--ui-font-scale)); }
.update-status { margin: 0; color: var(--text); font-size: calc(10px * var(--ui-font-scale)); }
.font-slider { width: 100%; margin: 2px 0; accent-color: var(--accent); }
.setting-block select, .directory-input, .setting-row select, .setting-stack select { min-width: 0; width: 100%; height: 28px; padding: 0 5px; color: var(--text); background: var(--surface); border: 1px solid var(--border); border-radius: 5px; font-size: calc(10px * var(--ui-font-scale)); }
.setting-row { display: grid; grid-template-columns: 90px minmax(0, 1fr); align-items: center; gap: 7px; min-height: 29px; color: var(--text); font-size: calc(10px * var(--ui-font-scale)); font-weight: 600; }
.setting-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 7px; }
.setting-stack { display: flex; flex-direction: column; gap: 4px; min-width: 0; font-size: calc(10px * var(--ui-font-scale)); color: var(--text); }
.toggle-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 7px; }
.setting-toggle { display: flex; align-items: center; justify-content: space-between; gap: 8px; min-height: 28px; padding: 0 6px; font-size: calc(10px * var(--ui-font-scale)); border-radius: 4px; color: var(--text); background: var(--surface); }
.setting-toggle input { width: 14px; height: 14px; flex: 0 0 auto; margin: 0; accent-color: var(--accent); }
.directory-input { font-family: Consolas, monospace; }
.reset-layout { align-self: flex-start; padding: 5px 8px; border-radius: 4px; background: transparent; border: 1px solid var(--border); color: var(--muted); font-size: calc(10px * var(--ui-font-scale)); }
.reset-layout:hover { color: var(--text); }
.compact-reset { margin-top: 2px; }
</style>

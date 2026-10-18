<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { QuickSelectConfig as readConfig, SaveQuickSelectConfig } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/quickselectservice'
import type { QuickSelectConfig, QuickSelectRule } from '../../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/models'

const props = defineProps<{ locale: 'zh' | 'en' }>()
type EditableQuickSelectConfig = Omit<QuickSelectConfig, 'enabledTypes' | 'rules'> & { enabledTypes: string[]; rules: QuickSelectRule[] }
const config = ref<EditableQuickSelectConfig | null>(null)
const error = ref('')
const status = ref('')
const busy = ref(false)
const preview = ref('')
const text = (zh: string, en: string) => props.locale === 'zh' ? zh : en
const typeLabels = [
  { id: 'url', zh: 'HTTP/HTTPS 链接', en: 'HTTP/HTTPS URLs' },
  { id: 'ip', zh: 'IPv4 / IPv6 地址', en: 'IPv4 / IPv6 addresses' },
  { id: 'path', zh: '常见文件路径', en: 'Common file paths' },
  { id: 'gitHash', zh: 'Git 提交哈希', en: 'Git commit hashes' },
]

onMounted(() => { void load() })

async function load() {
  busy.value = true
  error.value = ''
  try {
    const loaded = await readConfig()
    config.value = { ...loaded, enabledTypes: [...(loaded.enabledTypes ?? [])], rules: [...(loaded.rules ?? [])] }
  } catch (cause) {
    error.value = String(cause)
    config.value = null
  } finally { busy.value = false }
}

function setType(id: string, enabled: boolean) {
  if (!config.value) return
  const types = new Set(config.value.enabledTypes ?? [])
  if (enabled) types.add(id)
  else types.delete(id)
  config.value.enabledTypes = [...types]
}

function addRule() {
  if (!config.value || config.value.rules.length >= 32) return
  const id = typeof crypto.randomUUID === 'function' ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(36).slice(2)}`
  config.value.rules.push({ id, name: '', pattern: '', enabled: true })
}

function removeRule(id: string) {
  if (!config.value) return
  config.value.rules = config.value.rules.filter(rule => rule.id !== id)
}

function validateConfig(current: EditableQuickSelectConfig) {
  if (current.rules.length > 32) throw new Error(text('最多 32 条自定义规则。', 'You can add up to 32 custom rules.'))
  const ids = new Set<string>()
  for (const rule of current.rules) {
    if (!rule.name.trim() || [...rule.name.trim()].length > 32) throw new Error(text('规则名称必须为 1–32 个字符。', 'Rule names must contain 1–32 characters.'))
    if (!rule.pattern.trim() || [...rule.pattern].length > 512) throw new Error(text('规则表达式必须为 1–512 个字符。', 'Rule expressions must contain 1–512 characters.'))
    if (ids.has(rule.id)) throw new Error(text('规则 ID 重复。', 'Rule IDs must be unique.'))
    ids.add(rule.id)
  }
}

async function testRules(rules: QuickSelectRule[]) {
  const { scanQuickSelectInWorker, validateQuickSelectRulesInWorker } = await import('../quickSelectWorkerClient')
  const errors = await validateQuickSelectRulesInWorker(rules)
  if (errors.length) throw new Error(errors.join('; '))
  const result = await scanQuickSelectInWorker(
    'Quick Select preview: issue #APP-42, commit abcdef1234567, path /home/sam/project/file.go, URL https://example.com',
    { version: 1, enabledTypes: [], rules },
  )
  if (result.errors.length) throw new Error(result.errors.join('; '))
  return result.candidates.map(item => item.text)
}

async function previewRules() {
  if (!config.value || busy.value) return
  error.value = ''; status.value = ''; preview.value = ''; busy.value = true
  try {
    validateConfig(config.value)
    const found = await testRules(config.value.rules.filter(rule => rule.enabled))
    preview.value = found.length ? found.join(' · ') : text('未匹配示例文字。', 'No matches in the sample text.')
    status.value = text('规则预览完成。', 'Rule preview complete.')
  } catch (cause) { error.value = String(cause) }
  finally { busy.value = false }
}

async function save() {
  if (!config.value || busy.value) return
  error.value = ''; status.value = ''; busy.value = true
  try {
    validateConfig(config.value)
    await testRules(config.value.rules.filter(rule => rule.enabled))
    await SaveQuickSelectConfig(config.value)
    status.value = text('快速选择规则已保存。', 'Quick Select rules saved.')
  } catch (cause) { error.value = String(cause) }
  finally { busy.value = false }
}
</script>

<template>
  <section class="quick-select-settings" :aria-busy="busy">
    <p class="setting-help">{{ text('快速选择只读取终端当前缓冲，按字母标签复制匹配文字。自定义规则在可终止的隔离 Worker 中运行。', 'Quick Select reads the current terminal buffer and copies a match by its letter label. Custom rules run in a terminable worker.') }}</p>
    <template v-if="config">
      <label v-for="item in typeLabels" :key="item.id" class="setting-toggle">
        <span>{{ locale === 'zh' ? item.zh : item.en }}</span>
        <input type="checkbox" :checked="config.enabledTypes.includes(item.id)" @change="setType(item.id, ($event.target as HTMLInputElement).checked)" />
      </label>
      <div class="rule-heading">
        <strong>{{ text('自定义规则', 'Custom rules') }} · {{ config.rules.length }}/32</strong>
        <button type="button" :disabled="busy || config.rules.length >= 32" @click="addRule">{{ text('添加', 'Add') }}</button>
      </div>
      <article v-for="(rule, index) in config.rules" :key="rule.id" class="quick-rule">
        <div class="rule-title">
          <input v-model="rule.enabled" type="checkbox" :aria-label="text(`启用规则 ${index + 1}`, `Enable rule ${index + 1}`)" />
          <input v-model="rule.name" maxlength="32" :aria-label="text('规则名称', 'Rule name')" :placeholder="text('规则名称', 'Rule name')" />
          <button type="button" :aria-label="text('删除规则', 'Delete rule')" @click="removeRule(rule.id)">×</button>
        </div>
        <input v-model="rule.pattern" maxlength="512" class="rule-pattern" :aria-label="text('正则表达式', 'Regular expression')" placeholder="[A-Z]+-[0-9]+" />
      </article>
      <div class="rule-actions">
        <button type="button" :disabled="busy" @click="previewRules">{{ busy ? text('请稍候…', 'Working…') : text('预览规则', 'Preview rules') }}</button>
        <button type="button" :disabled="busy" @click="save">{{ text('保存规则', 'Save rules') }}</button>
      </div>
    </template>
    <button v-else type="button" :disabled="busy" @click="load">{{ text('重试读取配置', 'Retry loading config') }}</button>
    <p v-if="preview" class="setting-help" role="status">{{ preview }}</p>
    <p v-if="status" class="setting-status" role="status">{{ status }}</p>
    <p v-if="error" class="setting-error" role="alert">{{ error }}</p>
  </section>
</template>

<style scoped>
.quick-select-settings { display: grid; gap: 7px; color: var(--text); font-size: calc(10px * var(--ui-font-scale)); }
.setting-help, .setting-status, .setting-error { margin: 0; line-height: 1.4; overflow-wrap: anywhere; }
.setting-help { color: var(--muted); }
.setting-status { color: var(--positive); }
.setting-error { color: var(--negative); }
.rule-heading, .rule-title, .rule-actions { display: flex; align-items: center; gap: 5px; }
.rule-heading { justify-content: space-between; }
.quick-rule { display: grid; gap: 4px; padding: 5px; border: 1px solid var(--border-faint); border-radius: 4px; }
.rule-title input:nth-child(2) { flex: 1; min-width: 0; }
.quick-select-settings input:not([type=checkbox]), .quick-select-settings button { min-width: 0; padding: 4px 5px; border: 1px solid var(--border); border-radius: 4px; color: var(--text); background: var(--surface); font: inherit; }
.rule-pattern { width: 100%; font-family: Consolas, monospace !important; }
.rule-actions button { flex: 1; }
.rule-actions button:hover, .rule-heading button:hover { border-color: var(--accent); }
</style>

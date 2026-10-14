<script setup lang="ts">
import { computed, ref } from 'vue'

const types = [
  { id: 'local', title: 'Local', detail: 'PowerShell / Bash / Zsh' },
  { id: 'wsl', title: 'WSL', detail: 'Linux distributions on Windows' },
  { id: 'ssh', title: 'SSH', detail: 'Remote terminal sessions' },
  { id: 'observed', title: 'Observed', detail: 'External processes · read-only' },
] as const

const selected = ref<(typeof types)[number]['id']>('local')
const current = computed(() => types.find((item) => item.id === selected.value)!)
</script>

<template>
  <main class="layout">
    <aside class="sidebar">
      <header class="brand">
        <span class="mark" aria-hidden="true">&gt;_</span>
        <div>
          <strong>DiggerTerm</strong>
          <small>Terminal workspace</small>
        </div>
      </header>

      <div class="sidebar-label">WORKSPACES</div>
      <nav aria-label="Terminal types">
        <button
          v-for="item in types"
          :key="item.id"
          type="button"
          :class="['nav-item', { selected: selected === item.id }]"
          :aria-current="selected === item.id ? 'page' : undefined"
          @click="selected = item.id"
        >
          <span class="nav-title">{{ item.title }}</span>
          <span class="nav-detail">{{ item.detail }}</span>
        </button>
      </nav>
      <footer class="sidebar-foot">Scaffold · No sessions running</footer>
    </aside>

    <section class="content">
      <header class="toolbar">
        <span>{{ current.title }}</span>
        <span class="stage">Not connected</span>
      </header>
      <div class="empty-state">
        <span class="prompt" aria-hidden="true">&gt;_</span>
        <h1>{{ current.title }} sessions</h1>
        <p>Desktop foundation is ready. Session backends are not implemented yet.</p>
        <p class="hint">No processes are started or controlled by this screen.</p>
      </div>
    </section>
  </main>
</template>

<style>
:root {
  font-family: ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif;
  color: #d4d8df;
  background: #111318;
  font-synthesis: none;
}
* { box-sizing: border-box; }
body { margin: 0; }
button { font: inherit; }
.layout { display: flex; height: 100vh; min-height: 420px; }
.sidebar { width: 258px; flex-shrink: 0; display: flex; flex-direction: column; background: #191c23; border-right: 1px solid #2b2e36; padding: 23px 13px 14px; }
.brand { display: flex; align-items: center; gap: 12px; padding: 3px 10px 34px; }
.brand strong { font-size: 14px; letter-spacing: .015em; }
.brand small { display: block; margin-top: 3px; color: #848c98; font-size: 11px; }
.mark { display: grid; place-items: center; width: 34px; height: 34px; background: #292e37; border-radius: 9px; font-family: monospace; font-size: 15px; color: #c0ad91; }
.sidebar-label { padding: 0 10px 12px; color: #858b96; font-size: 10px; font-weight: 700; letter-spacing: .11em; }
nav { display: flex; flex-direction: column; gap: 3px; }
.nav-item { display: flex; flex-direction: column; align-items: start; gap: 5px; border: 0; border-radius: 7px; padding: 12px 11px; color: #d4d8df; background: transparent; text-align: left; cursor: pointer; }
.nav-item:hover { background: #262a32; }
.nav-item.selected { background: #30343e; }
.nav-title { font-size: 13px; font-weight: 600; }
.nav-detail { font-size: 10px; line-height: 1.4; color: #8c929d; }
.sidebar-foot { margin-top: auto; color: #7f8793; font-size: 10px; padding: 14px 10px 0; }
.content { display: flex; flex: 1; min-width: 0; flex-direction: column; background: #101217; }
.toolbar { display: flex; align-items: center; justify-content: space-between; height: 55px; border-bottom: 1px solid #252831; padding: 0 25px; font-size: 12px; font-weight: 600; }
.stage { color: #9297a1; font-size: 11px; font-weight: 400; }
.empty-state { display: flex; flex: 1; flex-direction: column; align-items: center; justify-content: center; padding: 25px; text-align: center; }
.prompt { font-family: monospace; font-size: 29px; color: #b5a083; }
h1 { margin: 16px 0 3px; font-size: 18px; font-weight: 600; }
p { max-width: 350px; margin: 8px 0 0; color: #959ba6; font-size: 12px; line-height: 1.7; }
.hint { color: #6f7782; font-size: 11px; }
@media (max-width: 680px) {
  .sidebar { width: 170px; padding-left: 8px; padding-right: 8px; }
  .nav-detail { font-size: 9px; }
}
</style>

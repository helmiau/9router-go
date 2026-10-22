<script lang="ts">
  import { onMount } from 'svelte'
  import { api, type JailbreakDoc, type JailbreakModelConfig } from '../api/client'

  // Default document: injected for any armed model that has no specific preset.
  let docPrompt = $state('')
  let docSafety = $state(false)
  let docSource = $state('')
  let docSaving = $state(false)
  let docStatus = $state('')

  // Preset library.
  let presets = $state<JailbreakDoc[]>([])
  let loading = $state(false)

  // Preset editor.
  let presetModalOpen = $state(false)
  let editingId = $state<string | null>(null)
  let presetForm = $state({ id: '', name: '', prompt: '', geminiSafety: false, source: '' })
  let presetSaving = $state(false)
  let presetError = $state('')

  // Per-model / provider / combo bindings.
  let models = $state<Record<string, JailbreakModelConfig>>({})
  let bindModelId = $state('')
  let bindPresetId = $state('')
  let bindError = $state('')
  let confirmDeleteId = $state<string | null>(null)

  onMount(load)

  async function load() {
    loading = true
    try {
      const [doc, presetList, modelMap] = await Promise.all([
        api.getJailbreak(),
        api.listJailbreakPresets(),
        api.listJailbreakModels(),
      ])
      docPrompt = doc?.prompt || ''
      docSafety = !!doc?.geminiSafety
      docSource = doc?.source || ''
      presets = Array.isArray(presetList) ? presetList : []
      models = modelMap || {}
    } catch (e) {
      docStatus = 'Load failed: ' + errorMessage(e)
    } finally {
      loading = false
    }
  }

  function errorMessage(e: unknown): string {
    return e instanceof Error ? e.message : String(e)
  }

  async function saveDoc() {
    docSaving = true
    docStatus = ''
    try {
      await api.saveJailbreak({ prompt: docPrompt, geminiSafety: docSafety, source: docSource || undefined })
      docStatus = 'Default document saved.'
    } catch (e) {
      docStatus = 'Save failed: ' + errorMessage(e)
    } finally {
      docSaving = false
    }
  }

  function openCreate() {
    editingId = null
    presetForm = { id: '', name: '', prompt: '', geminiSafety: false, source: '' }
    presetError = ''
    presetModalOpen = true
  }

  function openEdit(preset: JailbreakDoc) {
    editingId = preset.id || null
    presetForm = {
      id: preset.id || '',
      name: preset.name || '',
      prompt: preset.prompt || '',
      geminiSafety: !!preset.geminiSafety,
      source: preset.source || '',
    }
    presetError = ''
    presetModalOpen = true
  }

  async function savePreset() {
    if (!presetForm.name.trim() || !presetForm.prompt.trim()) {
      presetError = 'Name and prompt are required.'
      return
    }
    presetSaving = true
    presetError = ''
    try {
      await api.saveJailbreakPreset({
        id: presetForm.id.trim() || undefined,
        name: presetForm.name.trim(),
        prompt: presetForm.prompt,
        geminiSafety: presetForm.geminiSafety,
        source: presetForm.source.trim() || undefined,
      })
      presetModalOpen = false
      await load()
    } catch (e) {
      presetError = errorMessage(e)
    } finally {
      presetSaving = false
    }
  }

  async function removePreset(id: string) {
    if (confirmDeleteId !== id) {
      confirmDeleteId = id
      return
    }
    confirmDeleteId = null
    try {
      await api.deleteJailbreakPreset(id)
      await load()
    } catch (e) {
      docStatus = 'Delete failed: ' + errorMessage(e)
    }
  }

  async function bindModel() {
    const id = bindModelId.trim()
    if (!id) {
      bindError = 'Enter a model, provider, or combo id.'
      return
    }
    bindError = ''
    try {
      await api.setJailbreakModel(id, bindPresetId, true)
      bindModelId = ''
      bindPresetId = ''
      await load()
    } catch (e) {
      bindError = errorMessage(e)
    }
  }

  async function toggleModel(id: string, cfg: JailbreakModelConfig) {
    try {
      await api.setJailbreakModel(id, cfg.presetId, !cfg.enabled)
      await load()
    } catch (e) {
      docStatus = 'Toggle failed: ' + errorMessage(e)
    }
  }

  async function unbindModel(id: string) {
    try {
      await api.setJailbreakModel(id, 'off', false)
      await load()
    } catch (e) {
      docStatus = 'Unbind failed: ' + errorMessage(e)
    }
  }

  function presetLabel(id: string): string {
    if (id === 'off') return 'Off'
    const p = presets.find((x) => x.id === id)
    return p ? p.name || p.id || id : id
  }

  let boundEntries = $derived(Object.entries(models))
</script>

<div class="space-y-6">
  <div class="flex items-center gap-2">
    <span class="material-symbols-outlined text-brand-500">lock_open</span>
    <h1 class="text-xl font-semibold sm:text-2xl text-text-main">Jailbreak</h1>
    {#if loading}<span class="text-xs text-text-subtle">Loading…</span>{/if}
  </div>
  <p class="text-sm text-text-muted">
    Inject a system prompt ahead of the original content for chosen models, providers, or combos.
    Gemini models also get their safety filters relaxed when a binding has safety enabled.
  </p>

  <!-- Default document -->
  <section class="p-5 rounded-[14px] bg-surface border border-border space-y-4">
    <div class="flex items-center justify-between gap-2">
      <h2 class="text-sm font-bold text-text-main">Default document</h2>
      <span class="text-[11px] text-text-subtle">Used when a model is armed with no preset</span>
    </div>
    <label class="block space-y-1">
      <span class="text-xs text-text-muted">Source file name</span>
      <input
        type="text"
        bind:value={docSource}
        placeholder="agents.md"
        class="w-full px-3 py-2 rounded-[10px] bg-surface-2 border border-border text-text-main text-sm outline-none focus:border-brand-500"
      />
    </label>
    <label class="block space-y-1">
      <span class="text-xs text-text-muted">Prompt</span>
      <textarea
        bind:value={docPrompt}
        rows="10"
        placeholder="System prompt injected before the original content…"
        class="w-full px-3 py-2 rounded-[10px] bg-surface-2 border border-border text-text-main text-sm font-code outline-none focus:border-brand-500 resize-y"
      ></textarea>
    </label>
    <label class="flex items-center gap-2 cursor-pointer select-none">
      <input type="checkbox" bind:checked={docSafety} class="w-4 h-4 accent-brand-500" />
      <span class="text-sm text-text-main">Relax Gemini safety filters (BLOCK_NONE)</span>
    </label>
    <div class="flex items-center gap-3">
      <button
        type="button"
        onclick={saveDoc}
        disabled={docSaving}
        class="px-4 py-2 rounded-[10px] bg-brand-500 text-white text-sm font-medium disabled:opacity-50 hover:bg-primary-hover"
      >
        {docSaving ? 'Saving…' : 'Save default'}
      </button>
      {#if docStatus}<span class="text-xs text-text-muted">{docStatus}</span>{/if}
    </div>
  </section>

  <!-- Presets -->
  <section class="p-5 rounded-[14px] bg-surface border border-border space-y-4">
    <div class="flex items-center justify-between">
      <h2 class="text-sm font-bold text-text-main">Presets</h2>
      <button
        type="button"
        onclick={openCreate}
        class="px-3 py-1.5 rounded-[10px] border border-border text-text-main text-xs font-medium hover:bg-surface-2"
      >
        + New preset
      </button>
    </div>
    {#if presets.length === 0}
      <p class="text-xs text-text-subtle">No presets yet.</p>
    {:else}
      <ul class="divide-y divide-border">
        {#each presets as preset (preset.id ?? preset.name)}
          <li class="flex items-center justify-between gap-3 py-2.5">
            <div class="min-w-0">
              <div class="text-sm text-text-main truncate">{preset.name || preset.id || 'Untitled'}</div>
              <div class="text-[11px] text-text-subtle truncate">
                {preset.id}{preset.geminiSafety ? ' · safety' : ''}{preset.source ? ' · ' + preset.source : ''}
              </div>
            </div>
            <div class="flex items-center gap-2 shrink-0">
              <button type="button" onclick={() => openEdit(preset)} class="text-xs text-brand-500 hover:underline">Edit</button>
              {#if confirmDeleteId === (preset.id ?? '')}
                <button type="button" onclick={() => removePreset(preset.id ?? '')} class="text-xs text-error font-medium hover:underline">Confirm</button>
                <button type="button" onclick={() => (confirmDeleteId = null)} class="text-xs text-text-subtle hover:underline">Cancel</button>
              {:else}
                <button type="button" onclick={() => removePreset(preset.id ?? '')} class="text-xs text-text-muted hover:text-error hover:underline">Delete</button>
              {/if}
            </div>
          </li>
        {/each}
      </ul>
    {/if}
  </section>

  <!-- Model bindings -->
  <section class="p-5 rounded-[14px] bg-surface border border-border space-y-4">
    <h2 class="text-sm font-bold text-text-main">Model bindings</h2>
    <p class="text-xs text-text-subtle">
      Bind a model, provider, or combo id to a preset. The id is matched exactly as the client sends it.
    </p>
    <div class="flex flex-col sm:flex-row gap-2">
      <input
        type="text"
        bind:value={bindModelId}
        placeholder="model / provider / combo id"
        class="flex-1 px-3 py-2 rounded-[10px] bg-surface-2 border border-border text-text-main text-sm outline-none focus:border-brand-500"
      />
      <select
        bind:value={bindPresetId}
        class="px-3 py-2 rounded-[10px] bg-surface-2 border border-border text-text-main text-sm outline-none focus:border-brand-500"
      >
        <option value="">Default document</option>
        {#each presets as preset (preset.id ?? preset.name)}
          <option value={preset.id ?? ''}>{preset.name || preset.id}</option>
        {/each}
      </select>
      <button
        type="button"
        onclick={bindModel}
        class="px-4 py-2 rounded-[10px] bg-brand-500 text-white text-sm font-medium hover:bg-primary-hover"
      >
        Bind
      </button>
    </div>
    {#if bindError}<p class="text-xs text-error">{bindError}</p>{/if}
    {#if boundEntries.length === 0}
      <p class="text-xs text-text-subtle">No models armed.</p>
    {:else}
      <ul class="divide-y divide-border">
        {#each boundEntries as [id, cfg] (id)}
          <li class="flex items-center justify-between gap-3 py-2.5">
            <div class="min-w-0">
              <div class="text-sm text-text-main font-code truncate">{id}</div>
              <div class="text-[11px] text-text-subtle">{presetLabel(cfg.presetId)}</div>
            </div>
            <div class="flex items-center gap-3 shrink-0">
              <label class="flex items-center gap-1.5 cursor-pointer">
                <input
                  type="checkbox"
                  checked={cfg.enabled}
                  onchange={() => toggleModel(id, cfg)}
                  class="w-4 h-4 accent-brand-500"
                />
                <span class="text-xs text-text-muted">{cfg.enabled ? 'On' : 'Off'}</span>
              </label>
              <button type="button" onclick={() => unbindModel(id)} class="text-xs text-text-muted hover:text-error hover:underline">Unbind</button>
            </div>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
</div>

{#if presetModalOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50" role="dialog" aria-modal="true">
    <div class="w-full max-w-lg p-5 rounded-[14px] bg-surface border border-border space-y-4 shadow-xl">
      <h3 class="text-sm font-bold text-text-main">{editingId ? 'Edit preset' : 'New preset'}</h3>
      <label class="block space-y-1">
        <span class="text-xs text-text-muted">ID</span>
        <input
          type="text"
          bind:value={presetForm.id}
          disabled={!!editingId}
          placeholder="expert-coder"
          class="w-full px-3 py-2 rounded-[10px] bg-surface-2 border border-border text-text-main text-sm outline-none focus:border-brand-500 disabled:opacity-50"
        />
      </label>
      <label class="block space-y-1">
        <span class="text-xs text-text-muted">Name</span>
        <input
          type="text"
          bind:value={presetForm.name}
          placeholder="Expert Coder"
          class="w-full px-3 py-2 rounded-[10px] bg-surface-2 border border-border text-text-main text-sm outline-none focus:border-brand-500"
        />
      </label>
      <label class="block space-y-1">
        <span class="text-xs text-text-muted">Prompt</span>
        <textarea
          bind:value={presetForm.prompt}
          rows="6"
          class="w-full px-3 py-2 rounded-[10px] bg-surface-2 border border-border text-text-main text-sm font-code outline-none focus:border-brand-500 resize-y"
        ></textarea>
      </label>
      <label class="block space-y-1">
        <span class="text-xs text-text-muted">Source file name</span>
        <input
          type="text"
          bind:value={presetForm.source}
          placeholder="expert.md"
          class="w-full px-3 py-2 rounded-[10px] bg-surface-2 border border-border text-text-main text-sm outline-none focus:border-brand-500"
        />
      </label>
      <label class="flex items-center gap-2 cursor-pointer select-none">
        <input type="checkbox" bind:checked={presetForm.geminiSafety} class="w-4 h-4 accent-brand-500" />
        <span class="text-sm text-text-main">Relax Gemini safety filters</span>
      </label>
      {#if presetError}<p class="text-xs text-error">{presetError}</p>{/if}
      <div class="flex justify-end gap-2">
        <button type="button" onclick={() => (presetModalOpen = false)} class="px-4 py-2 rounded-[10px] border border-border text-text-main text-sm hover:bg-surface-2">Cancel</button>
        <button type="button" onclick={savePreset} disabled={presetSaving} class="px-4 py-2 rounded-[10px] bg-brand-500 text-white text-sm font-medium disabled:opacity-50 hover:bg-primary-hover">
          {presetSaving ? 'Saving…' : 'Save'}
        </button>
      </div>
    </div>
  </div>
{/if}

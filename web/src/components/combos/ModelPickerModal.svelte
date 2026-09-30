<script lang="ts">
  import { Info, Search, X } from 'lucide-svelte'
  import { api, type Combo, type ProviderConnection, type ProviderNode } from '../../api/client'
  import ModelPill from './ModelPill.svelte'
  import { getIconPath } from '../connections/types'
  import {
    resolveFilteredCombos,
    resolveFilteredGroups,
    resolveModelPickerGroups,
  } from './pickerData'
  import { parseCustomModelsResponse, parseDisabledModelsMap } from '../../lib/customModels'
  import { createGroupWindow, groupWindow, measureGroup } from './groupWindow'

  interface Props {
    isOpen: boolean
    target: 'combo' | 'vision' | 'audio' | 'judge'
    connections?: ProviderConnection[]
    combos?: Combo[]
    providerNodes?: ProviderNode[]
    modelAliases?: Record<string, string>
    customModels?: Array<{ providerAlias?: string; id: string; name?: string; type?: string }>
    disabledModels?: Record<string, string[]>
    currentComboName?: string
    addedModelValues?: string[]
    onSelect: (modelValue: string) => void
    onDeselect?: (modelValue: string) => void
    onClose: () => void
  }

  let {
    isOpen,
    target,
    connections = [],
    combos = [],
    providerNodes = [],
    modelAliases,
    customModels,
    disabledModels,
    currentComboName,
    addedModelValues = [],
    onSelect,
    onDeselect,
    onClose,
  }: Props = $props()

  let searchQuery = $state('')
  let fetchedAliases = $state<Record<string, string>>({})
  let fetchedCustoms = $state<Array<{ providerAlias?: string; id: string; name?: string; type?: string }>>([])
  let fetchedDisabled = $state<Record<string, string[]>>({})

  $effect(() => {
    if (isOpen) {
      searchQuery = ''
      // Upstream parity (ComboFormModal fetchModalData): aliases drive
      // passthrough + custom-node rows; customs/disabled complete the merge.
      api.getModelAliases().then((r) => (fetchedAliases = r?.aliases || {})).catch(() => {})
      api.getCustomModels().then(normalizeCustoms).catch(() => {})
      api.getDisabledModels().then(normalizeDisabled).catch(() => {})
    }
  })

  function normalizeCustoms(res: unknown) {
    // Upstream parity: GET /api/models/custom -> { models: [...] }.
    fetchedCustoms = parseCustomModelsResponse(res)
  }

  function normalizeDisabled(res: unknown) {
    // Upstream parity: full-map { disabled: {...} } (ModelSelectModal) / bare map (go port).
    fetchedDisabled = parseDisabledModelsMap(res)
  }

  let groups = $derived(
    resolveModelPickerGroups(connections, providerNodes, {
      modelAliases: modelAliases ?? fetchedAliases,
      customModels: customModels ?? fetchedCustoms,
      disabledModels: disabledModels ?? fetchedDisabled,
    })
  )
  let filteredCombos = $derived(
    resolveFilteredCombos(combos, currentComboName, searchQuery, target)
  )
  let filteredGroups = $derived(
    resolveFilteredGroups(groups, searchQuery, target, addedModelValues)
  )

  // Mounting every group meant ~890 pills and 111-131ms to come back to the
  // full list on every keystroke back to an empty search (measured). Only the
  // groups near the viewport are mounted; each one still renders as one intact
  // block, so the list looks identical. A searched list is left fully rendered -
  // the results are small (~142 pills, ~13ms) and the user needs to see all.
  let groupState = createGroupWindow()
  let groupIds = $derived(filteredGroups.map((g) => g.id))
  let scrollTop = $state(0)
  let viewportHeight = $state(0)
  let combosHeight = $state(0)
  let scrollEl: HTMLDivElement | undefined = $state()
  let windowed = $derived(!searchQuery.trim())
  // Bumped whenever a measurement lands, so the window recomputes. Assigning
  // `scrollTop = scrollTop` would be a no-op that Svelte never propagates.
  let measureTick = $state(0)

  // The Combos block sits above the provider groups inside the same scroll
  // container, so it has to be part of the offset maths. Leaving it out is what
  // made the earlier attempt drift: the window was computed against a scroll
  // position that included space nothing had accounted for.
  let listOffset = $derived(combosHeight)
  let gwin = $derived.by(() => {
    void measureTick // recompute once a measured height changes
    // 0 disables the window (see groupWindow), so pass it while searching: the
    // results are small and the user needs to see every one of them.
    const vp = windowed ? viewportHeight : 0
    return groupWindow(groupState.heights, groupIds, scrollTop - listOffset, vp)
  })
  let visibleGroups = $derived(
    windowed ? filteredGroups.slice(gwin.start, gwin.end) : filteredGroups
  )
  // Spacers sit inside the scroll container, below the Combos block.
  let topSpacer = $derived(windowed ? gwin.topSpacer : 0)
  let bottomSpacer = $derived(windowed ? gwin.bottomSpacer : 0)

  // Measure the viewport once the modal is on screen. Waiting for a scroll event
  // never fires until the user interacts, and an unknown viewport deliberately
  // renders everything, so this has to happen on mount.
  $effect(() => {
    if (!isOpen || !scrollEl) return
    const measure = () => {
      viewportHeight = scrollEl!.clientHeight
    }
    measure()
    const raf = requestAnimationFrame(measure)
    return () => cancelAnimationFrame(raf)
  })

  function handleScroll(e: Event) {
    const el = e.currentTarget as HTMLDivElement
    scrollTop = el.scrollTop
    viewportHeight = el.clientHeight
  }

  function observeGroup(node: HTMLElement, id: string) {
    const ro = new ResizeObserver((entries) => {
      const h = entries[0]?.contentRect.height
      if (h && measureGroup(groupState, id, h)) measureTick += 1
    })
    ro.observe(node)
    return () => ro.disconnect()
  }

  function observeCombos(node: HTMLElement) {
    const ro = new ResizeObserver((entries) => {
      const h = entries[0]?.contentRect.height
      if (h && combosHeight !== h) combosHeight = h
    })
    ro.observe(node)
    return () => ro.disconnect()
  }

  function handleToggle(val: string) {
    if (addedModelValues.includes(val)) {
      if (onDeselect) {
        onDeselect(val)
      } else {
        onSelect(val)
      }
    } else {
      onSelect(val)
    }
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 z-[70] flex items-center justify-center p-4">
    <!-- Overlay -->
    <div class="absolute inset-0 bg-black/50 backdrop-blur-[2px] fade-in" onclick={onClose} aria-hidden="true"></div>

    <div
      class="relative w-full max-w-md bg-surface border border-border-subtle rounded-[14px] shadow-[var(--shadow-elev)] fade-in overflow-hidden flex flex-col max-h-[85vh] z-10 p-4!"
      role="dialog"
      aria-modal="true"
    >
      <!-- Header (traffic lights + title) -->
      <div class="flex items-center justify-between p-2 border-b border-border-subtle -m-4 mb-4">
        <div class="flex items-center">
          <div class="hidden md:flex items-center gap-2 mr-4 ml-2">
            <button
              type="button"
              onclick={onClose}
              aria-label="Close"
              title="Close"
              class="w-4 h-4 rounded-full bg-[#FF5F56] hover:brightness-90 transition-all cursor-pointer flex items-center justify-center"
            >
              <span class="text-[9px] font-bold text-white leading-none">✕</span>
            </button>
            <div class="w-4 h-4 rounded-full bg-[#3a3a3a]/20 dark:bg-white/15 cursor-not-allowed"></div>
            <div class="w-4 h-4 rounded-full bg-[#3a3a3a]/20 dark:bg-white/15 cursor-not-allowed"></div>
          </div>
          <h2 class="text-lg font-semibold text-text-main">
            {target === 'vision'
              ? 'Add Vision Model'
              : target === 'audio'
                ? 'Add Audio Model'
                : target === 'judge'
                  ? 'Select Judge Model'
                  : 'Add Model to Combo'}
          </h2>
        </div>
        <button
          type="button"
          onclick={onClose}
          aria-label="Close"
          class="md:hidden p-1.5 rounded-[10px] text-text-muted hover:bg-surface-2 hover:text-text-main transition-colors"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Info bar -->
      <div class="flex items-center gap-2 mb-3 px-2.5 py-2 bg-brand-500/10 border border-brand-500/20 rounded-lg text-xs text-text-muted">
        <Info class="w-3.5 h-3.5 text-brand-500 shrink-0" />
        <span>Click to add, click again to remove. Changes are saved automatically.</span>
      </div>

      <!-- Search -->
      <div class="mb-3">
        <div class="relative">
          <Search class="w-3.5 h-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
          <input
            type="text"
            placeholder="Search..."
            bind:value={searchQuery}
            class="w-full bg-surface border border-border rounded pl-8 pr-3 py-1.5 text-xs text-text-main placeholder:text-text-muted focus:outline-none focus:ring-1 focus:ring-brand-500/50"
          />
        </div>
      </div>

      <!-- Categories & Models List.
           Windowed per provider group: the container reserves the full height
           through the two spacers, and only the groups near the viewport are
           mounted. Each mounted group is still one intact block, so the list is
           visually unchanged. -->
      <div
        bind:this={scrollEl}
        onscroll={handleScroll}
        class="max-h-[400px] overflow-y-auto custom-scrollbar"
      >
        <!-- Combos section - always first, above the windowed groups -->
        {#if filteredCombos.length > 0}
          <div use:observeCombos>
            <div class="flex items-center gap-1.5 mb-1.5 sticky top-0 bg-surface py-0.5 z-10">
              <span class="text-xs font-medium text-brand-500">Combos</span>
              <span class="text-[10px] text-text-muted">({filteredCombos.length})</span>
            </div>
            <div class="flex flex-wrap gap-1.5">
              {#each filteredCombos as combo (combo.id)}
                <ModelPill
                  label={combo.name}
                  value={combo.name}
                  isAdded={addedModelValues.includes(combo.name)}
                  onClick={() => handleToggle(combo.name)}
                />
              {/each}
            </div>
          </div>
        {/if}

        <div style="height:{topSpacer}px" aria-hidden="true"></div>

        <!-- Provider sections -->
        {#each visibleGroups as group (group.id)}
          <div class="mb-3" use:observeGroup={group.id}>
            <div class="flex items-center gap-1.5 mb-1.5 sticky top-0 bg-surface py-0.5 z-10">
              <img
                src={getIconPath(group.id)}
                alt={group.name}
                class="w-3.5 h-3.5 object-contain rounded-sm"
                loading="lazy"
                onerror={(e) => {
                  ;(e.currentTarget as HTMLImageElement).style.display = 'none'
                }}
              />
              <span class="text-xs font-medium text-brand-500">
                {group.name}
              </span>
              <span class="text-[10px] text-text-muted">
                ({group.models.length})
              </span>
            </div>
            <div class="flex flex-wrap gap-1.5">
              {#each group.models as model (model.value)}
                <ModelPill
                  label={model.name}
                  value={model.value}
                  isAdded={addedModelValues.includes(model.value)}
                  caps={model.caps}
                  onClick={() => handleToggle(model.value)}
                />
              {/each}
            </div>
          </div>
        {/each}

        <div style="height:{bottomSpacer}px" aria-hidden="true"></div>

        {#if filteredCombos.length === 0 && filteredGroups.length === 0}
          <div class="text-center py-4 text-text-muted">
            <Search class="w-6 h-6 mx-auto mb-1 opacity-50" />
            <p class="text-xs">No models found</p>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

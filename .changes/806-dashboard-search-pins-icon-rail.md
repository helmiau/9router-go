### feat(dashboard): global search, pinned entries, and a collapsible icon rail

Three sidebar surfaces, each behind a Settings switch because each one changes
how the operator reads the page rather than what the gateway does.

**Search (Ctrl+K / Cmd+K).** One index over every model, provider, page and
settings section — 1,656 rows, built once and cached. The query is *staged* in
the input and only run when the operator presses Enter or the Search button:
a keystroke-driven filter over 1,656 rows re-renders the list between two
keypresses, which reads as lag on the 1 vCPU box this gateway serves. Editing
the query retires the results on screen, so a stale list never reads as a
search that ran on its own. Pages and settings outrank models, because an
operator typing a tab name wants the tab, not the 400 models whose id happens
to contain the same letters. A nav row can be pinned straight from its result.

**Pinned entries (default on).** A Pinned section above the groups, holding up
to eight entries. On by default because the section is empty until something
is pinned, so an untouched install gains nothing and loses nothing. The list
is a per-operator localStorage preference rather than a server setting: an
operator who pins Usage on their laptop has not asked for it on the box in the
rack. A pin naming an entry a later build removes is dropped rather than
rendered as a dead row.

**Collapse to icons (default off).** The sidebar shrinks to a 56px icon rail,
every label replaced by its icon with a tooltip. Off by default because it
hides every label at once, which is a layout to opt into rather than one to
discover. The links stay in the DOM — only their labels go.

The two preferences live in a shared store (`lib/sidebarPrefs.svelte.ts`)
rather than in `displayPrefs.ts`, because the Settings switch and the sidebar
are siblings in the same tab: a `storage` event never fires for that write, and
polling localStorage to work around it is the kind of thing that quietly
becomes permanent.

Verified in a real browser: typing produces no results and Enter does; the
Search button runs the same pass; a model query ranks models first and a page
query ranks the page first; a result navigates and closes; a pin survives a
reload and an unpin removes it; the rail shrinks 288px → 56px with every link
intact and restores on the second click. `bun test src/` 664 pass (16 new), the
palette e2e suite 6/6, the sidebar e2e suite 4/4, svelte-check ratchet holds
at 76.

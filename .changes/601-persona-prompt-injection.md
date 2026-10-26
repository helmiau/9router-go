### Added
- Per-model, per-provider, and per-combo system-prompt injection, off by default: a model is never injected just because it exists, only because it was armed (`internal/handlers/chat/jailbreak.go`, `internal/db/jailbreak.go`).
- Preset library with keyword and exclude rules across providers and combo legs. Three rule modes: `word` (whole token, default), `partial` (case-insensitive substring), and `regex` (RE2). Bounds keep a typo from becoming a rule that matches everything: at most 20 rules per list, 200 characters per rule.
- A global exclude list that outranks every preset and exact binding, persisted in `kv` and rebuilt from disk on boot, so a restart never resurrects a rule the operator removed.
- `POST /api/jailbreak/resolve`, which explains which rule decided one id — exact-binding, keyword, global-exclude, preset-exclude, or no-match — so the dashboard answers "why did this id get injected?" without an operator reasoning through the precedence chain by hand.
- `POST /api/jailbreak/import`, which accepts a GitHub repository, a single file on GitHub, or any other http(s) URL, read as one prompt file named after its last path segment. A page served as HTML is reduced to its readable text rather than stored as markup.
- Preset `kind` (SOUL, AGENTS, CLAUDE, PERSONA, LOAD…), filled automatically from the file name on import, so an imported library is labelled the way its authors intended.
- SSRF guard on non-GitHub import URLs, checked against the gateway's own network at connect time through the dialer's Control hook — the address that is dialled is the address that was checked. Loopback, the private ranges, and link-local are refused unless the operator opts in.
- Per-preset hit counters, so the library shows how many times each preset actually injected rather than how many times it was merely armed.

### Changed
- The dashboard surface is named **Persona** in the sidebar, page title, and nav presets. The mechanism is narrower than the old name suggested: it prepends a system prompt of the operator's own, and nothing about it judges what that prompt says. The How-it-works panel says plainly that a persona is also the usual way to apply jailbreak prompting. The nav id and the `/dashboard/jailbreak` route are deliberately unchanged — the id is the storage key for every deployment's saved sidebar arrangement, and the path is what operators already have bookmarked.
- Import is three tabs inside the New preset modal — Write, From files, From URL — instead of a card of its own, so one button is the only way in. The Paste a prompt tab was dropped rather than moved: it did exactly what Write already does, and two tabs performing one task is one tab too many.
- Test match is a modal rather than a card. It answers one question occasionally, and a full card cost as much vertical space as the 53-row preset table above it.
- The Add-binding modal's tabs are the segmented control the Proxy Pools Bulk Add dialog uses, so the two "one by hand vs many at once" dialogs match. The default tab is Single.
- Editing a binding reuses that modal's Single tab instead of opening a second one: the id field goes read-only and an on/off checkbox appears, which is the whole difference.
- Batch controls on both lists are two rows — select-all and the selected count on the first, the actions that apply to the selection on the second. One row held both, so the actions pushed the count sideways as the selection grew. The selected count is a badge.
- Binding rows split the provider prefix into a chip, so a long id truncates the part that says the least.
- `Modal` takes an optional icon, rendered beside the title, so New preset, Add model bindings, Edit model binding, and Test match all read the same.

### Fixed
- The bulk model filter did nothing. `bulkArmFiltered` was a `$derived.by` that also wrote `bulkArmRegexError`, and a derived that writes to another state is not re-tracked — the input accepted text, the count stayed at 200, and nothing filtered. The validity check and the message are their own deriveds now, and the filter reads them.
- The Add button opened the Bulk tab, not Single. It was wired as `onclick={openBindModal}`, which hands the MouseEvent in as the tab, so neither tab was selected and the panel fell through to Bulk.
- Both lists keep their filter on screen when a query matches nothing. The field used to live inside the `{:else}` branch, so a typo made the only way to clear it disappear.
- The live catalog now loads when the Bulk tab is clicked, not only when the modal is opened on it — the tab is reachable from Single without reopening the modal, so opening it was never a reliable trigger.

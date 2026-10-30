### feat(dashboard): sidebar counts, and labelled groups that fold

**Counts beside an entry.** A badge is a fact the operator would otherwise open
a page to learn, so it is only shown when it is actionable: Providers reads
`3 / 12 active` rather than a bare total, because a third of the fleet being
down is the news and "12" is not. A failing connection is the one number that
earns the warning colour — the fleet is degraded and the operator has not
opened the page yet. Combos and API keys show a plain count while the list is
short enough that the number is quicker to read than opening the page, and go
quiet past twenty. Most entries show nothing at all.

The counts come from data the dashboard already holds — `lastError` is what the
provider detail page reads, so the badge counts the same thing the operator
would see there rather than a second opinion about health. Proxy Pools shows
nothing because its list is fetched inside its own view, and inventing a number
for it would be worse than an absent one.

**Labelled groups fold.** `Advanced Features` and `System` are now buttons, so
an operator on a short screen can press the sections they are not using. Groups
start open — the stored set holds the *collapsed* ids, so a build that never
wrote one draws every group the way it always did. The group holding the active
tab stays open whatever was stored, because folding it would hide the page the
operator is looking at.

Group state and accordion state are two sets rather than one: an accordion
starts folded and its set says what was opened, while a group starts open and
its set says what was collapsed. One set holding both would need a naming
convention to tell them apart.

Verified in a real browser: groups start expanded, fold on click, survive a
reload, and reopen when the operator navigates to a page inside them; a badge
appears with the right sentence once an API key exists, and again in the Pinned
section when that row is pinned. `bun test src/` 672 pass (8 new), the palette
e2e suite 8/8, svelte-check ratchet holds at 76.

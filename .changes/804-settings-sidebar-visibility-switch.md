### fix(settings): the Sidebar card's visibility switches answer again

Settings › Sidebar gives every sidebar entry a switch to hide or show it.
PR #601 rewrote that `<Toggle>` to `disabled={true}`, so every switch rendered
greyed out and swallowed the click — the preset dropdown still worked, which
is what made the panel look healthy.

The same PR dropped a `</div>` in the Display card, leaving the
"Advanced settings" row as the parent of the provider-toggle row, the
animation dropdown and the Currency block.

- The switch is disabled only for the entry the panel lives behind
  (`ALWAYS_VISIBLE_NAV_IDS`), which the server drops from the hide list on
  write as well, so hiding Settings is unreachable from either side.
- The change rides `markLayoutEdit`, like the preset, the arrows and a drop
  already do: without it a flipped switch raised no save reminder, and the
  edit was lost on the next navigation.
- The Display card's rows are siblings again.

`web/e2e/sidebarVisibility.test.ts` drives a real browser against a real
gateway and fails on the disabled switch. A `disabled` attribute is invisible
to `tsc`, `vite build`, the svelte-check ratchet and `go test`, so the
assertion has to be the rendered DOM — the same reason the rest of the e2e
suite exists.

Verified: the switch hides the entry, Save publishes it, the sidebar drops
the row, the switch brings it back, and the Settings row stays disabled;
the svelte-check baseline drops 77 → 76.

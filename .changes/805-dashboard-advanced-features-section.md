### feat(dashboard): an Advanced Features sidebar section, and a collapsible Media Providers

The sidebar grew a third section. **Advanced Features** sits above System and
holds the three entries an operator configures once and rarely revisits —
Security, Plans, Persona — so the System group is left with the surfaces that
run the gateway rather than govern it.

Media Providers is now a real accordion rather than one hardcoded boolean:
`openAccordions` is a set of accordion ids, so a second accordion in
`web/src/lib/nav.ts` needs no new state variable in the sidebar, and the open
state survives a reload through `localStorage` — an operator who unfolds it
once means it, because the sidebar is the surface they arrange rather than a
dialog they dismiss. The header's active state is derived from its children
instead of a `media-` prefix, so it stays correct for any accordion.

`aria-expanded` rides the button, which the old markup lacked.

Verified in a real browser: the section renders with Security (Plans and
Persona stay hidden by default), the accordion folds and unfolds with its
children, the open state survives a reload, and the Settings arrangement panel
draws both new containers with every switch enabled except Settings.
`bun test src/` 648 pass, the sidebar e2e suite 4/4, svelte-check ratchet
holds at 76.

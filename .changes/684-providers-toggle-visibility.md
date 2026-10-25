### fix(providers): persistent desktop toggle + provider-level toggle on detail page

- Provider card on/off switch is no longer hidden behind a hover reveal at the `sm` breakpoint (`sm:opacity-0 sm:group-hover:opacity-100` dropped) — visible permanently on desktop and mobile.
- Provider detail page gains the same toggle in its header, enabling or disabling every connection of that provider without returning to `/providers`. A `Disabled` badge mirrors the overview card's state when all connections are off.
- Settings > Display now exposes a **Provider card toggle** switch that controls whether the on/off switch on each provider card is always visible or hidden behind a hover reveal at the `sm` breakpoint. The preference persists in `localStorage` (`providerTogglePersistent`, default on) so it survives page reloads without a server round trip.

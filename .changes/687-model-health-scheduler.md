### Scheduled model health checks, with a failure threshold and an audit log

Four things that only work together: a scheduler that probes, a counter that
decides when a model is dead, an auto-enable that brings it back, and an audit
log that says who did what.

- **The loop probes only models that already carry a verdict**, so a fresh
  install with a hundred-model provider costs nothing until the operator
  presses Test on a model. It disables only on repeated failure, so one
  transient 502 does not take a working model off the listing. And it
  re-enables only what it disabled itself, never what an operator switched
  off by hand — an operator's decision outranks the scheduler's.
- **The threshold counts consecutive failures.** One success resets the
  count, which is what makes a flapping model safe: it never reaches the
  threshold, so it is never switched off. A stored threshold of 1 is raised
  to the minimum rather than honoured, because one error is one transient
  error.
- **Auto-enable is off by default** and only ever re-enables rows carrying
  the scheduler's own mark. A manual enable clears the mark, so from that
  moment the model is the operator's.
- **Audit log** records both scheduler and operator actions in one place, so
  "who took this model off?" is answerable without a second source. Stored
  newest-first and bounded at 200 entries, so a long-running gateway does not
  grow the row without limit.
- **Settings are read every pass, never cached**, so a change saved in the
  dashboard takes effect on the next pass without a restart — the same
  property the auto-ping loop has.

The "failed" probe scope had a cold-start gap that only a live run exposed: a
model with no counter row was skipped forever, so it could never acquire the
first failure that would put it on the watch list, and the scope would only
ever see rows some other path had already written. One probe per new model is
the entry cost, and it is paid once. The unit tests passed with the gap in
place because they seeded counter rows directly.

The route is registered in both route tables. `dashboard/routes.go` is
mounted by tests only, so a route added there alone answers 200 in a unit
test and 404s on the real server — the trap the node-limit, smart-rules and
auto-disable pairs already warn about.

Verification: `go test ./internal/modelhealth/ ./internal/handlers/dashboard/
./internal/handlers/chat/ ./internal/providers/ ./internal/app/` (18 new
cases), `bun test src` (591 pass), `make vet-svelte` (ratchet at baseline 78),
`bun run build`, `golangci-lint` scoped to the diff (0 issues), and a live run
against a gateway with the loop enabled: 89 models across the compatible
nodes were probed, disabled at the threshold, and logged with their reasons.
The disabled lists were cleared and the loop switched off again afterwards.

### Fixed

- **The operator-override invariant was never enforced** (`fix(model-health)`). `RecordOperatorAction` was defined but called from nowhere in production, so the loop never learned that a model had become the operator's: a model the operator re-enabled stayed marked scheduler-owned, which meant the loop refused to disable it again after fresh threshold failures, and with auto-enable on it silently re-enabled a model the operator had switched off by hand — exactly the "anything you disabled by hand stays off" case the panel claims. The disabled-models write now diffs the stored list against the submitted one and reports each changed model to the loop, attributed to `operator` in the audit log, and a manual re-enable clears the scheduler mark so the loop stops treating the model as its own. The loop is resolved per request rather than captured at construction, because the router is built before the app's `OnStart` hook starts the scheduler — a handler that snapshotted it would hold nil in production and record nothing. A build with no loop published (tests, tooling) records nothing rather than failing, and the disabled list itself is written either way.
- **Toggling the scheduler on did nothing until a restart** (`fix(model-health)`). `Loop.Run` returned outright when it observed `enabled == false`, so a process that booted with the feature off — a fresh install, or one where the operator had turned it off — never entered the loop at all, and turning it on in the dashboard had no effect until the process restarted. The loop now stays alive: while the feature is off it re-reads the config on a short poll, and once running it settles into the configured interval. The interval wait is sliced rather than one blocking sleep, so a context cancel or a shorter interval saved in the dashboard is noticed promptly instead of after the old interval elapses.

### Jailbreak prompt injection (per model, provider, and combo)

- Ports the prompt-injection system from `arewedaks/9router-go`: an operator
  saves an `agents.md`-style document once and binds it to specific models, and
  every request to those models carries the prompt prepended ahead of the
  original content.
- Injection is opt-in and default-off, and it is wired into every forward path —
  single chat (OpenAI), Claude single, combo fallback (OpenAI + Claude), the
  fusion panel, and the fusion judge — so it covers every model, provider, and
  combo. For a combo or fusion the toggle is keyed on the combo name the client
  sends, which arms every leg of that combo.
- Preset library with per-model binding, plus a Gemini-native lever: when a
  binding enables it, `ForwardGemini` adds `safetySettings` `BLOCK_NONE` through
  the official API field. A body that already carries `safetySettings` is left
  untouched.
- Storage reuses the existing `kv` table under scope `jailbreak` (no schema
  migration): presets are `doc:<id>`, model bindings `model:<id>`. Deleting a
  preset auto-unbinds every model armed to it.
- Dashboard API: `GET/PUT /api/jailbreak`, preset CRUD under
  `/api/jailbreak/presets`, and `GET/PUT /api/jailbreak/models`. The model id
  travels in the request body rather than the path because ids contain `/` and
  `:` (`tkh/mimo-v2.6-flash:free`), which a path parameter stored still-encoded.
- New Svelte 5 `JailbreakView` behind a `Jailbreak` sidebar tab to manage the
  default document, the preset library, and per-model bindings.
- Verified: `go test ./internal/db/... ./internal/translator/... ./internal/proxy/... ./internal/handlers/chat/... ./internal/handlers/dashboard/...` all pass, including a combo integration test proving the injected prompt reaches the upstream body for every leg and a control proving an unarmed combo forwards untouched; `bun run build` succeeds and the svelte-check ratchet holds at baseline (0 unresolved identifiers).

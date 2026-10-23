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

### Keyword and exclude rules for the same presets

- A preset now carries keyword rules, so one preset covers a model family
  across providers and combo legs instead of one armed id at a time: keyword
  `deepseek` covers `providera/deepseek`, `providerx/deepseek-4-pro`,
  `deepseek-v4`, and every combo leg carrying one of those ids, with no
  per-model setup.
- Three modes per rule: `word` (default — a whole token or token run, so
  `deepseek` does not match `deepseekia`), `partial` (substring), and `regex`
  (RE2, unanchored, case-insensitive with a `(?-i)` opt-out). Word and partial
  rules are stored lowercased; a regex is stored verbatim because lowercasing
  would corrupt escapes and character classes.
- Two deny lists share the same grammar: per-preset excludes, and one global
  exclude list (`GET`/`PUT /api/jailbreak/excludes`) that outranks every
  preset and every exact binding.
- Precedence, highest first: global exclude, exact binding on the id the
  client sent, the winning preset's excludes, a keyword on the requested id
  before a forwarded leg id, then the longest keyword (ties to the lowest
  preset id). An exact binding to an empty preset stays authoritative and does
  not fall through to a keyword.
- The combo and fusion paths resolve against the leg ids as well as the combo
  name (`provider/model` and the bare model), so a combo whose legs come from
  different providers is covered by one keyword.
- Resolution reads one immutable in-memory snapshot (bindings, presets, rules,
  compiled patterns) invalidated on every jailbreak write, so a request does
  no extra queries and compiles no regexes — the hot path stays at zero
  queries where it was two.
- Dashboard: a shared `KeywordRules` editor for the include and exclude lists
  on the default document, every preset, and the global exclude card, with the
  same validation the API applies (a bad regex is refused with the reason
  before the request is sent) and a hint when a typed binding id is already
  covered by a keyword.
- New `docs/jailbreak.md` is the usage guide: presets, the three arming
  methods, the precedence order, a worked example, limits, the REST API, and
  how to verify an injection end to end.
- Verified: `go test ./internal/db/... ./internal/handlers/...` pass, including
  table-driven matcher tests, precedence and cache-invalidation tests, and
  end-to-end tests proving a keyword-armed preset reaches the upstream body on
  the single-model and combo-leg paths while a preset exclude blocks it;
  `bun test src scripts` passes (523 tests) and the svelte-check ratchet holds
  at baseline with 0 unresolved identifiers.

### Import a prompt collection from GitHub

- `POST /api/jailbreak/import-github` accepts `owner/repo`,
  `owner/repo@branch`, and the `github.com`, `raw.githubusercontent.com` and
  `api.github.com` URL forms, and writes one preset per prompt file, so a
  repository such as `togg53192-cmd/jailbreaks` loads in one request instead of
  one paste per model.
- Each preset's keywords are derived from its file name as a prefix chain:
  `deepseek v4.md` yields `deepseek`, `deepseek-v4`. Resolution prefers the
  longest matching rule, so the file named after a model beats a sibling that
  only shares the family token, and a name longer than the model id
  (`kimi-k3-agents.md` for `kimi-k3`) still resolves. The chain is capped at
  the storage bound so an import never produces a preset the store refuses.
- Every file is attempted on its own: a dead link or an empty file is reported
  by path with its reason and the rest still imports. `README*`,
  dot-directories, non-`.md`/`.txt` files, files past 512 KB, and anything past
  200 files or 8 MB are skipped and named in the response.
- Only GitHub hosts are reachable, redirects re-checked against the same
  allowlist: the target is operator input, and following it anywhere would turn
  the dashboard into a proxy for the gateway's own network. Responses are
  size-bounded, and files are fetched through a bounded pool of four so a large
  collection cannot spawn unbounded work on a one-vCPU box.
- Re-running an import updates the presets it wrote rather than duplicating
  them, so refreshing a collection is safe.
- Dashboard: an "Import from GitHub" card on the Jailbreak view with the
  accepted input shapes, the imported presets with the keywords each one
  matches, and the skipped files with their reasons.
- Verified: `go test ./internal/handlers/dashboard/...` covers the target
  parser (every accepted shape plus traversal, non-GitHub host and internal-host
  rejection), the keyword chain, the candidate filter, an end-to-end import
  against a fake GitHub proving the presets resolve for `providerx/deepseek-v4`
  and `kimi-k3` while an uncovered model resolves to nothing, the refusal path,
  and the handler's 400-versus-502 mapping; a live import of
  `togg53192-cmd/jailbreaks` wrote its presets and injected a request for
  `deepseek v4` on a provider the collection never names.

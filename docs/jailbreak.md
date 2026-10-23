# Jailbreak: prompt injection per model, provider, and combo

The jailbreak feature prepends a system prompt to the requests you choose. You
write the prompt once as a preset, decide which models it applies to, and every
matching request carries it ahead of the original content. Injection is opt-in
and off by default: a model is never injected because it exists, only because
you armed it.

The dashboard page is **Jailbreak** (sidebar tab). Everything below is also
available over the REST API listed at the end.

## 1. Write a preset

A preset is a prompt plus two optional settings:

- **Prompt** — the text injected ahead of the original content. Paste an
  `agents.md` file or type it inline; up to 512 KB.
- **Relax Gemini safety filters** — when on, requests to Gemini models also
  carry `safetySettings` with `BLOCK_NONE` through the official API field. A
  body that already has its own `safetySettings` is left untouched.
- **Source file name** — display only, so you can tell an uploaded document
  from a typed one.

The **Default document** is the preset named `default`. It is what a model gets
when it is armed without naming a preset. Create more presets for other
personas (coder, researcher, translator) and pick per model.

## 2. Decide what it applies to

There are three ways to arm a preset, and they stack.

### Exact binding

Bind one id to one preset. The id is matched exactly as the client sends it —
the same id `GET /v1/models` lists:

- a model: `deepseek/deepseek-chat`
- a provider node: `mynode/deepseek-chat`
- a combo: `my-combo` — arming the combo name arms **every leg** of that combo

This is the precise tool: use it when one id needs a different preset from the
rest of its family, or when you want a whole combo armed regardless of its legs.

### Keywords (include)

A keyword on a preset covers every model id that matches it, across providers
and combo legs, with no per-model setup. This is the tool for "the same preset
for `providera/deepseek` and `providerx/deepseek-4-pro`".

Three modes, chosen per rule:

| Mode | Matches | Example: rule `deepseek` against `providerx/deepseek-4-pro` |
| --- | --- | --- |
| `word` (default) | a whole token or a run of tokens | matches |
| `partial` | any substring | matches (`seek` would too) |
| `regex` | an RE2 pattern, unanchored | matches (`^deepseek` would not) |

A `word` rule must sit on token boundaries — string edges or one of
`/ - _ : .` — so `deepseek` matches `deepseek-v4` and `providerx/deepseek-4-pro`
but not `deepseekia`. `partial` drops that requirement. `regex` is matched
unanchored and case-insensitively; write `(?-i)` at the start of the pattern to
opt back into case sensitivity.

Rules are stored lowercased for `word` and `partial`, and verbatim for `regex`
(lowercasing would corrupt `\D`, `[^A-Z]` and friends).

### Excludes (deny)

Two deny lists, both using the same three modes:

- **Per preset** — models matching one of these rules never get *this* preset,
  even when one of its keywords matches.
- **Global exclude list** — models matching one of these rules are never
  injected at all. It outranks every preset and every exact binding, so it is
  the switch for "these few models stay untouched".

## 3. Precedence

When more than one rule matches one request, the first applicable line wins:

1. **Global exclude list** — a match means no injection, whatever anything else
   says.
2. **Exact binding** on the id the client sent (a model id, or a combo name).
   An exact binding is authoritative even when its preset prompt is empty:
   binding a model to an empty preset means "inject nothing here".
3. **The winning preset's own excludes** — a match means no injection.
4. **Keyword on the id the client sent**, before a keyword on a forwarded leg
   id. The request's primary id outranks what it resolves to, so a keyword on a
   combo name covers the whole combo.
5. **Among keyword hits on the same id, the longest keyword wins.** A length
   tie goes to the lowest preset id, so the result never depends on storage
   order.

For a combo, the ids checked are the combo name the client sent, then each
leg's `provider/model`, then the leg's bare model. So a combo named
`fleet-alpha` whose legs are `deepseek-v4` and `providerx/deepseek-4-pro` gets
the preset whose keyword matches `deepseek` — on both legs, with no per-leg
configuration.

## 4. Worked example

Goal: one prompt for every DeepSeek model from every provider and every combo,
except one internal build.

1. **Jailbreak → + New preset.** Name `Deepseek`, paste the prompt, save.
2. In the preset editor, under **Keywords (include)**, add `deepseek` (mode
   `word`) and add.
3. Under **Excludes (deny)**, add `deepseek-4-pro-internal`.
4. Save the preset.

Every request whose id contains the token `deepseek` now carries the prompt:
`providera/deepseek`, `providerx/deepseek-4-pro`, `deepseek-v4`, and any combo
leg with one of those ids. `providerx/deepseek-4-pro-internal` does not, and
neither does `deepseekia`.

To exempt a model from *all* presets, put it in the **Global exclude list**
instead.


## 4b. Import a collection from GitHub

A repository that keeps one prompt per file — the shape
[togg53192-cmd/jailbreaks](https://github.com/togg53192-cmd/jailbreaks) uses —
can be pulled in whole. In **Jailbreak → Import from GitHub**, enter any of:

| Input | Meaning |
| --- | --- |
| `owner/repo` | every prompt file in the default branch |
| `owner/repo@branch` | the same, from a named branch, tag or sha |
| `https://github.com/owner/repo` | the repository |
| `…/tree/ref/path` | one directory of it |
| `…/blob/ref/file.md` | one file |
| `https://raw.githubusercontent.com/owner/repo/ref/file.md` | one file |

Each `.md` or `.txt` file becomes a preset named after the file, and its
keywords are derived from that name: `deepseek v4.md` yields the chain
`deepseek`, `deepseek-v4`. Resolution prefers the longest matching rule, so the
file named after a model wins over a sibling that only shares the family token,
while `providerx/deepseek-v4` resolves exactly as `deepseek-v4` does. A file
whose name is longer than the model id (`kimi-k3-agents.md` for `kimi-k3`) still
resolves, because the chain reaches the shorter id.

Skipped, and reported by name in the result: `README*`, anything under a
dot-directory such as `.github`, non-`.md`/`.txt` files, files past 512 KB, and
anything past 200 files or 8 MB in one import. A file that fails to download
does not stop the rest — the import reports it and continues.

Only GitHub hosts are reachable, redirects included: the target is operator
input, and following it anywhere would turn the dashboard into a proxy for the
gateway's own network.

Re-running an import updates the presets it wrote rather than duplicating
them, so refreshing a collection is safe. Imported presets are live the moment
they land: a request for a model they cover is injected. Delete a preset, or
add the id to the global exclude list, to stop that.

## 5. Limits

| Thing | Limit |
| --- | --- |
| Prompt size | 512 KB |
| Rules per list (keywords or excludes) | 20 |
| Rule length | 200 characters |

A rule the gateway would reject is refused at save time with the reason, so a
typo never becomes a preset that silently stops matching.

## 6. REST API

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/jailbreak` | the default document |
| `PUT` | `/api/jailbreak` | save the default document (accepts `keywords`, `excludes`) |
| `GET` | `/api/jailbreak/presets` | list presets |
| `PUT` | `/api/jailbreak/presets` | create or update a preset |
| `DELETE` | `/api/jailbreak/presets/{id}` | delete a preset (unbinds its models) |
| `GET` | `/api/jailbreak/models` | list exact bindings |
| `PUT` | `/api/jailbreak/models` | bind or unbind one id |
| `GET` | `/api/jailbreak/excludes` | the global exclude list |
| `PUT` | `/api/jailbreak/excludes` | replace the global exclude list |
| `POST` | `/api/jailbreak/import-github` | import a repository of prompts as presets (`{"target":"owner/repo"}`) |

The model id travels in the request body rather than the path, because ids
contain `/` and `:` (`tkh/mimo-v2.6-flash:free`) and a path parameter stores
them still-encoded.

## 7. Checking it works

Point one connection at a local server that prints what it receives, arm a
preset, and send a request:

```bash
# arm a preset by keyword
curl -b cookies.txt -X PUT http://localhost:20130/api/jailbreak/presets \
  -H 'Content-Type: application/json' \
  -d '{"id":"deepseek","name":"Deepseek","prompt":"DS RULE","keywords":[{"text":"deepseek"}]}'

# ask for a model the keyword covers
curl -H 'Authorization: Bearer <client key>' \
  -H 'Content-Type: application/json' \
  -X POST http://localhost:20130/v1/chat/completions \
  -d '{"model":"providerx/deepseek-4-pro","messages":[{"role":"user","content":"hi"}]}'
```

The upstream body starts with `{"content":"DS RULE","role":"system"}` ahead of
the original messages. Ask for a model the keyword does not cover and the body
is forwarded untouched.

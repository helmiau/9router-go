### Bulk custom-model parser: colon and slash are valid id characters, not separators

The paste parser (`web/src/lib/bulk-models.ts` `SEPARATORS`) and the server-side
`invalidModelIDReason` guard (`internal/handlers/dashboard/model_id.go`) both
split on whitespace, commas and semicolons only. A colon (`:`) or a dot (`.`)
inside a pasted token is part of the model id, not a boundary — so a paste like
`model1:free,modelX:paid` correctly produces two ids (`model1:free` and
`modelX:paid`) split on the comma, with the colon kept inside each id.

A slash (`/`) is also valid in an id: it is the `<prefix>/<id>` address
separator and the kv-store key delimiter uses `|`, not `/`, so ids like
`anthropic/claude-sonnet-4-6` (OpenRouter) or `x/llama-3.3-70b` (Grok) are
accepted. This avoids "Failed to add custom model: an id cannot contain
/|"'\<> or a backtick" when importing from a non-custom provider's catalog.

**Files:** `web/src/lib/bulk-models.ts`, `internal/handlers/dashboard/model_id.go`,
`web/src/lib/bulk-models.test.ts`, `internal/handlers/dashboard/model_id_test.go`

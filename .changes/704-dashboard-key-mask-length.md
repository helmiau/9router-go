### fix(dashboard): keep the API key mask the same length as the secret

- The mask collapsed everything between the first six and last four characters
  into a fixed run of bullets, so a 35-character key rendered as 11 characters.
  That told the reader two things it had no business knowing: that the secret was
  longer than ten characters, and roughly how much longer. A mask whose length
  tracks the thing it hides is not a mask — it is a length oracle.
- All three masks now preserve the exact length: `maskClientKey` (the value the
  API sends as `keyDisplay`), `keikey.Mask` (stored on rotate), and `maskSecret`
  in the dashboard's key table. A revealed key and its mask occupy the same
  width, so pressing reveal no longer reflows the row or moves the buttons
  beside the value.
- `maskClientKey` also switches from byte slicing to rune slicing. A key carrying
  a multi-byte character was cut mid-rune, which produced invalid UTF-8 in the
  JSON response body.
- Short values still collapse to a fixed sentinel: there is nothing to show and
  nothing to measure. `keikey.Mask`'s `"***"` is kept for the same reason
  upstream keeps it, so existing no-key usage rows stay in one bucket instead of
  splitting.

**Verified.** `go build ./internal/...` and `go test ./internal/keikey/
./internal/handlers/dashboard/` pass. `TestMask` gained a rune-count assertion
and was mutation-checked: restoring the ellipsis form fails it.

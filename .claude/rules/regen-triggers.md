---
paths:
  - "cmd/crux/*.go"
  - "internal/formatter/*.go"
  - "internal/llmdocs/0*.md"
  - "internal/llmdocs/1*.md"
  - "internal/llmdocs/2*.md"
  - "internal/llmdocs/3*.md"
---

# You just touched the source of the embedded LLM reference

If you changed a command, a flag or a help string, run `/regen-ai` before
finishing so `internal/llmdocs/90-commands.md` matches. CI regenerates it and
fails on a dirty tree.

If you changed anything about the **JSON output** — a field name, a type, an
order — update `internal/llmdocs/20-schemas.md` by hand. Agents parse against
that chapter, and a mismatch makes them misread the data silently rather than
fail.

Do not edit `90-commands.md` directly.

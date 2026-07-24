---
name: regen-ai
description: Regenerate the embedded LLM reference and verify the result. Use after changing commands, flags, help text, JSON output, or the hand-written reference chapters.
allowed-tools: Bash(go generate:*) Bash(go test:*) Bash(go build:*) Bash(git status:*) Bash(git diff:*) Read
---

# regen-ai

Bring `internal/llmdocs/` back in line with the code.

1. `git status --short` — note what is already dirty.
2. `go generate ./...` — rewrites `90-commands.md`.
3. `go build ./... && go test ./...`.
4. Report which commands or flags changed.

Then check by hand what generation cannot: if the change touched JSON output,
`20-schemas.md` needs a matching edit. The generator only sees the command tree,
not the shape of what commands print.

This skill is Claude Code-local; it is not part of the distributed plugin.

# Generated artifacts — do not hand-edit

| Generated file | Source of truth |
| --- | --- |
| `internal/llmdocs/90-commands.md` | the cobra command definitions in `cmd/crux/main.go`, rendered by the hidden `gen-llmdocs` command |

Hand-written and safe to edit:

- `internal/llmdocs/00-guide.md` — data sources, how to choose, authentication
- `internal/llmdocs/10-metrics.md` — metrics and Core Web Vitals thresholds
- `internal/llmdocs/20-schemas.md` — JSON output schemas
- `internal/llmdocs/30-gotchas.md` — limits and traps
- `plugins/crux-cli/skills/*/SKILL.md`
- `context7.json`

Editing a generated file is always wrong: the next `go generate ./...`
overwrites it, and CI fails on the stale diff in the meantime. To improve the
catalog, improve the command's `Short` / `Long` / flag usage strings instead.

Regenerate with `/regen-ai`, or `go generate ./... && go test ./...`.

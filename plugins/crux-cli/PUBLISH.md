# Publishing the crux-cli plugin

## Before every release

1. `go generate ./...` — regenerate `internal/llmdocs/90-commands.md`, commit
   any diff.
2. `go test ./...` — includes `TestPluginSkills`, which enforces that
   `plugin.json.version` equals `PluginVersion` in `cmd/crux/main.go` and that
   the SKILL.md frontmatter stays within the Agent Skills standard.
3. `claude plugin validate plugins/crux-cli`.
4. Bump `PluginVersion` and `plugin.json.version` together, in the same commit
   as the release tag. The release workflow refuses a mismatched tag.

## Registering in the marketplace (first release only)

Add to `.claude-plugin/marketplace.json` in `ideamans/claude-public-plugins`:

```json
{
  "name": "crux-cli",
  "source": {
    "source": "git-subdir",
    "url": "https://github.com/ideamans/crux-cli.git",
    "path": "plugins/crux-cli"
  }
}
```

## Verifying the published result

```
/plugin marketplace add ideamans/claude-public-plugins
/plugin install crux-cli@ideamans-plugins
/crux-usage
```

```bash
gh skill install ideamans/crux-cli/plugins/crux-cli/skills/crux-usage --agent copilot
```

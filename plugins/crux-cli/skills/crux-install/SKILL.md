---
name: crux-install
description: Make the crux command available, installing it only if it is missing. Use when another skill reports that `crux` is not on PATH, or when the user asks to install, update or upgrade the ideamans CrUX CLI. Prefers an already-installed binary, then the latest GitHub release, then a build from source with go install.
license: MIT
compatibility: Requires curl (or wget) and tar to install from a release, or a Go toolchain for the source fallback. Standalone — does not need crux to be present already. Installs from the public repository github.com/ideamans/crux-cli, so no GitHub authentication is needed.
allowed-tools: Bash(curl:*) Bash(wget:*) Bash(tar:*) Bash(unzip:*) Bash(go:*) Bash(uname:*) Bash(command:*) Bash(which:*) Bash(mkdir:*) Bash(mv:*) Bash(cp:*) Bash(rm:*) Bash(chmod:*) Bash(ls:*) Bash(test:*) Bash(echo:*) Read
---

# crux-install

Make the `crux` command usable, doing the least work that achieves it.

## Route 1 — an existing installation on PATH

```bash
command -v crux && crux --version
```

If that resolves, **use it and stop here.** Do not check for a newer release —
it costs an API call and the user did not ask for an upgrade.

Two checks before trusting the hit:

- **It is the right tool.** `crux` is a short, generic name. `crux llm | head -1`
  must read `# crux — reference for AI agents`. If something else owns the name,
  tell the user and use an explicit path rather than shadowing theirs.
- **It is recent enough.** If `crux llm` is not a known command, the binary
  predates the embedded reference. Say so and continue to route 2 to upgrade it.

Continue past this section only when the command is missing, is the wrong tool,
is too old, or the user explicitly asked to update.

## Route 2 — the latest GitHub release

The repository is public, so no authentication is needed.

```bash
VERSION=$(curl -fsSL https://api.github.com/repos/ideamans/crux-cli/releases/latest \
  | grep '"tag_name"' | head -1 | cut -d'"' -f4)   # e.g. v0.5.0
```

**The archive is named after the goreleaser project, not the repository** —
`crux`, without the `-cli` suffix:

```
crux_<version-without-v>_<os>_<arch>.tar.gz
```

`<os>` is `darwin`, `linux` or `windows` (lowercase); `<arch>` is `amd64` or
`arm64`, so `uname -m` reporting `x86_64` maps to `amd64`. Windows ships a
`.zip`.

```bash
OS=$(uname -s | tr '[:upper:]' '[:lower:]')            # darwin | linux
ARCH=$(uname -m); [ "$ARCH" = "x86_64" ] && ARCH=amd64  # amd64 | arm64
curl -fsSL -o /tmp/crux.tar.gz \
  "https://github.com/ideamans/crux-cli/releases/download/${VERSION}/crux_${VERSION#v}_${OS}_${ARCH}.tar.gz"
```

If the download 404s, list the actual assets on the release page rather than
retrying variations.

### Install onto PATH

```bash
tar -xzf /tmp/crux.tar.gz -C /tmp
mkdir -p ~/.local/bin && mv /tmp/crux ~/.local/bin/ && chmod +x ~/.local/bin/crux
```

Prefer the first writable directory already on PATH — `~/.local/bin`, then
`/usr/local/bin`. Two things not to do on your own initiative:

- If nothing on PATH is writable, leave the binary in `/tmp`, print the exact
  `sudo mv` command and let the user run it. Do not run `sudo` yourself.
- If `~/.local/bin` is not on PATH, give the user the line to add to their shell
  profile. Do not edit the profile for them.

## Route 3 — build from source

Needs a Go toolchain and compiles rather than downloads, so it is the last
resort. Note the `/cmd/crux` suffix; installing the module root would not build
anything.

```bash
go install github.com/ideamans/crux-cli/cmd/crux@latest
```

The binary lands in `$(go env GOPATH)/bin` and is named `crux`.

## Verify

```bash
crux --version
crux llm | head -5
crux auth status
```

Report which route was taken, the version and the install path.

Then say what is still needed. `crux` cannot query anything without credentials,
and which ones depend on the subcommand: `history` and `record` need a free CrUX
API key (`crux auth set-api-key`), while `device` needs Google Cloud ADC and a
billing project because it queries BigQuery at the user's expense. Point them at
whichever matches what they want to do rather than setting up both.

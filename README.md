# crux-cli

A command-line tool to query [Chrome UX Report (CrUX)](https://developer.chrome.com/docs/crux) data. It supports two data sources:

- **BigQuery** (`crux device`) — monthly origin-level aggregates from `chrome-ux-report.materialized.device_summary`, billed to your Google Cloud project, with local caching.
- **CrUX API** (`crux history` / `crux record`) — the free public [CrUX API](https://developer.chrome.com/docs/crux/api): a 28-day rolling distribution for an **origin or a specific URL**, as either a weekly time series or the latest snapshot. Requires only an API key.

## Features

- Query `chrome-ux-report.materialized.device_summary` from the command line (BigQuery)
- Query the CrUX API for an origin **or a specific URL**, as a weekly history or latest record
- Filter by device type / form factor (phone / desktop / tablet / all)
- Multiple origins in a single query for competitive analysis
- Local cache per origin × period — BigQuery is only queried for cache misses
- Daily check of the latest available CrUX month; cache auto-invalidates when a new month is published
- Output as table, JSON, or CSV

## Prerequisites

- Go 1.21+
- A Google Cloud project (BigQuery queries are billed to your project)
- [Google Cloud SDK](https://cloud.google.com/sdk) for Application Default Credentials

## Installation

```bash
git clone https://github.com/ideamans/crux-cli.git
cd crux-cli
go build -o crux ./cmd/crux
# optionally move to a directory in your PATH
mv crux /usr/local/bin/
```

## Authentication

### BigQuery (`crux device`)

crux-cli uses **Application Default Credentials (ADC)**. Run the following once:

```bash
gcloud auth application-default login
```

Then set your BigQuery project ID:

```bash
crux auth set-project YOUR_GCP_PROJECT_ID
```

> **Note:** `chrome-ux-report` is a public dataset, but query costs are charged to your own project.

### CrUX API (`crux history` / `crux record`)

The CrUX API needs an API key. Create one in the [Google Cloud Console](https://console.cloud.google.com/apis/credentials) (enable the *Chrome UX Report API*), then provide it one of these ways:

```bash
# Save it to the config file (~/.crux-cli/config.json)
crux auth set-api-key YOUR_CRUX_API_KEY

# …or set it per shell session
export CRUX_API_KEY=YOUR_CRUX_API_KEY

# …or pass it inline
crux history -o https://example.com --api-key YOUR_CRUX_API_KEY
```

Resolution order: `--api-key` > `CRUX_API_KEY` env > config file. The CrUX API is free (rate-limited to 150 queries/minute per key).

## Quick Start

```bash
# Last 12 months for a single origin (default metrics: LCP, CLS, INP)
crux device -o https://example.com

# Phone only
crux device -o https://example.com -d phone

# Compare multiple origins
crux device -o https://example.com -o https://competitor.com

# All metrics except FID
crux device -o https://example.com --full-metrics

# Specific metrics
crux device -o https://example.com --metrics fcp,ttfb,ol

# JSON output for the last 3 months
crux device -o https://example.com --months 3 -f json

# Fixed date range
crux device -o https://example.com --from 202401 --to 202412

# --- CrUX API (requires an API key) ---

# Weekly history for an origin (latest snapshot: crux record)
crux history -o https://web.dev

# Latest record for a specific page on desktop
crux record -u https://web.dev/learn -d desktop
```

## Commands

### `crux device`

Query device_summary and display Web Vitals metrics.

```
crux device --origin <origin> [flags]
```

| Flag | Short | Default | Description |
|---|---|---|---|
| `--origin` | `-o` | required | Origin(s) to query. Repeat or comma-separate (max 50) |
| `--device` | `-d` | `all` | `phone` / `desktop` / `tablet` / `all` |
| `--months` | `-m` | `12` | Number of months to look back |
| `--from` | | | Start month `YYYYMM` (overrides `--months`) |
| `--to` | | | End month `YYYYMM` (defaults to latest CrUX month) |
| `--metrics` | | | Metrics to show, comma-separated. Available: `lcp,cls,inp,fcp,ttfb,ol,rtt,fid` |
| `--full-metrics` | | | Show all metrics except `fid` |
| `--format` | `-f` | `table` | `table` / `json` / `csv` |
| `--project` | | | BigQuery project ID (overrides config) |
| `--no-cache` | | | Skip cache; always query BigQuery |

**Available metrics**

| Key | Name | Good threshold |
|---|---|---|
| `lcp` | Largest Contentful Paint | ≤ 2500 ms |
| `cls` | Cumulative Layout Shift | ≤ 0.1 |
| `inp` | Interaction to Next Paint | ≤ 200 ms |
| `fcp` | First Contentful Paint | ≤ 1800 ms |
| `ttfb` | Time to First Byte | ≤ 600 ms |
| `ol` | Overall Load | ≤ 4000 ms |
| `rtt` | Round Trip Time | ≤ 75 ms |
| `fid` | First Input Delay (legacy) | ≤ 100 ms |

Default display: `lcp`, `cls`, `inp`. Use `--full-metrics` for all except `fid`.

### `crux history` (CrUX API — weekly time series)

Query the [CrUX History API](https://developer.chrome.com/docs/crux/history-api) for a weekly time series (up to 40 collection periods) of an origin or a specific URL.

```
crux history (--origin <origin> | --url <url>) [flags]
```

| Flag | Short | Default | Description |
|---|---|---|---|
| `--origin` | `-o` | | Origin(s) to query. Repeat or comma-separate |
| `--url` | `-u` | | URL(s) to query a specific page. Repeat or comma-separate |
| `--device` | `-d` | `all` | Form factor: `phone` / `desktop` / `tablet` / `all`. `all` = aggregate across devices |
| `--connection` | `-c` | `all` | Effective connection type: `4g` / `3g` / `2g` / `slow-2g` / `offline` / `all`. `all` = aggregate across connection types |
| `--periods` | | `25` | Number of weekly collection periods, `1`–`40` |
| `--metrics` | | | Metrics to query, comma-separated. Available: `lcp,cls,inp,fcp,ttfb,rtt` |
| `--format` | `-f` | `table` | `table` / `json` / `csv` |
| `--api-key` | | | CrUX API key (overrides `CRUX_API_KEY` and config) |

At least one `--origin` or `--url` is required.

```bash
# Last 25 weeks for an origin (phone), default metrics
crux history -o https://web.dev -d phone

# A specific page, all metrics, 40 periods
crux history -u https://web.dev/learn --metrics lcp,cls,inp,fcp,ttfb,rtt --periods 40

# Phone users on a 4G connection only
crux history -o https://web.dev -d phone -c 4g

# Compare origin vs. a page as JSON
crux history -o https://web.dev -u https://web.dev/learn -f json
```

### `crux record` (CrUX API — latest snapshot)

Query the CrUX API for the latest single record (28-day rolling snapshot). Same flags as `crux history` except `--periods`.

```bash
crux record -o https://web.dev
crux record -u https://web.dev/learn -d desktop -f json
```

> **Note:** Unlike `crux device` (BigQuery), `--device all` here means **aggregate across all form factors** (a single record), not a per-device breakdown. The CrUX API returns the 28-day rolling distribution; weekly periods are dated by their end date.

### `crux auth`

```bash
crux auth status                      # Show current project, credentials, and CrUX API key status
crux auth set-project <project-id>    # Set the default BigQuery project ID
crux auth set-api-key <api-key>       # Set the CrUX API key
```

### `crux --llm`

Print a detailed reference aimed at LLMs / AI agents and exit. It documents both
data sources, every command and flag, the metric thresholds, and the exact JSON
field meanings for `-f json` output (e.g. `fast_*` densities, `p75_*` units, the
CrUX API `good/needs_improvement/poor/p75` structure). Useful for letting an
agent drive `crux` autonomously.

```bash
crux --llm            # global guide
crux history --llm    # same guide (flag works on any subcommand)
```

### `crux cache`

```bash
crux cache dir           # Print the cache directory path
crux cache list          # List cached entries and latest month info
crux cache clear         # Delete all device cache files
crux cache clear -o <origin>  # Delete cache for a specific origin
```

## Cache Behavior

Cache files are stored under `~/.crux-cli/cache/` by default.

```
~/.crux-cli/cache/
  latest_month.json          # Most recent CrUX month (refreshed every 24 hours)
  device/
    <origin-hash>/
      <YYYYMM>_<YYYYMM>.json.gz   # Rows for one origin × period, gzip-compressed
```

- The cache key is `(origin, monthFrom, monthTo)`.
- CrUX publishes new monthly data around the 12th–15th of each month. When the latest month advances, the previous cache files are simply not matched by the new key, so they are naturally bypassed and fresh data is fetched.
- For multiple origins, only cache-miss origins are queried from BigQuery.
- Use `--no-cache` to bypass caching entirely.

## Configuration

Settings are stored in `~/.crux-cli/config.json`.

| Key | Default | Description |
|---|---|---|
| `project_id` | | BigQuery project ID |
| `api_key` | | CrUX API key |
| `cache_dir` | `~/.crux-cli/cache` | Cache directory path |
| `default_format` | `table` | Default output format |
| `default_months` | `12` | Default number of months |

Override with environment variables:

| Variable | Description |
|---|---|
| `CRUX_PROJECT` | BigQuery project ID |
| `CRUX_API_KEY` | CrUX API key |
| `CRUX_CACHE_DIR` | Cache directory path |

## License

MIT

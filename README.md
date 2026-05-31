# crux-cli

A command-line tool to query [Chrome UX Report (CrUX)](https://developer.chrome.com/docs/crux) data from BigQuery. Retrieve Core Web Vitals and other performance metrics by origin, with local caching to minimize BigQuery costs.

## Features

- Query `chrome-ux-report.materialized.device_summary` from the command line
- Filter by device type (phone / desktop / tablet / all)
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

crux-cli uses **Application Default Credentials (ADC)**. Run the following once:

```bash
gcloud auth application-default login
```

Then set your BigQuery project ID:

```bash
crux auth set-project YOUR_GCP_PROJECT_ID
```

> **Note:** `chrome-ux-report` is a public dataset, but query costs are charged to your own project.

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

### `crux auth`

```bash
crux auth status                      # Show current project and credential info
crux auth set-project <project-id>    # Set the default BigQuery project ID
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
| `cache_dir` | `~/.crux-cli/cache` | Cache directory path |
| `default_format` | `table` | Default output format |
| `default_months` | `12` | Default number of months |

Override with environment variables:

| Variable | Description |
|---|---|
| `CRUX_PROJECT` | BigQuery project ID |
| `CRUX_CACHE_DIR` | Cache directory path |

## License

MIT

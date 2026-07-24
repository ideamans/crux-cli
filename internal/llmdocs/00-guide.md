# crux — reference for AI agents

`crux` queries Chrome UX Report (CrUX) real-user performance data (Core Web
Vitals and related metrics) from the command line.

This reference is embedded in the binary — `crux llm` always describes the exact
version you are running.

## Two independent data sources

The subcommand chooses the source, and they are not interchangeable.

**BigQuery** → `crux device`

- Monthly, **origin-level** aggregates from the public BigQuery table
  `chrome-ux-report.materialized.device_summary`.
- Requires a Google Cloud project; **BigQuery query cost is billed to it**.
- Local caching per (origin, period). Origin only — no per-URL queries.
- Time granularity: calendar month (`YYYYMM`).
- Metrics available only here: `ol` (onload), `fid` (legacy).

**CrUX API** → `crux history`, `crux record`

- Free public REST API (<https://developer.chrome.com/docs/crux/api>). Needs
  only an API key — no GCP project, no billing.
- Works on an origin **or a specific URL (page)**.
- `history` = weekly time series (up to 40 collection periods).
  `record` = the single latest record (28-day snapshot).
- Time granularity: 28-day rolling window, refreshed weekly (Mondays).

### How to choose

| Situation | Use |
| --- | --- |
| Data for a specific page/URL | `history` or `record` (API) |
| Only an API key, no GCP project | `history` or `record` (API) |
| Long monthly history, or many origins with consistent monthly buckets | `device` (BigQuery) |
| Need `ol` or `fid` | `device` (BigQuery only) |

## Rules for agents

1. **Always pass `-f json`** when you are going to parse the result. The default
   `table` format is for humans.
2. All density / fraction fields are proportions in `[0,1]` — `0.83` means 83%.
3. **A density or p75 of `0` usually means missing or insufficient data, not a
   real zero.** Do not report it as a measurement.
4. `crux device` bills BigQuery to the user's project. Say so before running a
   large query, and prefer the cache (do not pass `--no-cache` without reason).

## Authentication

**BigQuery (`device`)** — Application Default Credentials:

```bash
gcloud auth application-default login     # once
crux auth set-project <PROJECT_ID>        # or CRUX_PROJECT, or --project
```

**CrUX API (`history` / `record`)** — an API key, resolved highest priority
first: `--api-key` → `CRUX_API_KEY` → config file (`crux auth set-api-key`).
Create the key in Google Cloud Console and enable the "Chrome UX Report API".

Check both at once:

```bash
crux auth status
```

Never echo an API key back to the user or write it into a file they did not ask
for; `crux auth set-api-key` stores it in the config for them.

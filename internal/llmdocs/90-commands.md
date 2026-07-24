# Command catalog

Generated from the cobra command tree by `go generate ./...`.
Do not edit by hand — edit the command definitions instead.

## `crux auth`

Manage BigQuery credentials and the CrUX API key

### `crux auth set-api-key`

Set the CrUX API key in the config file

```
crux auth set-api-key <api-key>
```

### `crux auth set-project`

Set the default BigQuery project ID in the config file

```
crux auth set-project <project-id>
```

### `crux auth status`

Show current authentication and project configuration

## `crux cache`

Manage local cache

### `crux cache clear`

Delete cache files

| flag | type | default | description |
| --- | --- | --- | --- |
| `-o`, `--origin` | string | — | Clear cache for a specific origin only |

### `crux cache dir`

Print the cache directory path

### `crux cache list`

List cached entries

## `crux device`

Query chrome-ux-report.materialized.device_summary

| flag | type | default | description |
| --- | --- | --- | --- |
| `-d`, `--device` | string | `all` | Device filter: phone / desktop / tablet / all |
| `-f`, `--format` | string | — | Output format: table / json / csv |
| `--from` | string | — | Start month YYYYMM (overrides --months) |
| `--full-metrics` | bool | `false` | Show all metrics except fid (lcp,cls,inp,fcp,ttfb,ol,rtt) |
| `--metrics` | string | — | Metrics to display, comma-separated (default: lcp,cls,inp) Available: lcp,cls,inp,fcp,ttfb,ol,rtt,fid |
| `-m`, `--months` | int | `0` | Number of months to look back (default from config or 12) |
| `--no-cache` | bool | `false` | Skip cache; always query BigQuery |
| `-o`, `--origin` | stringArray | `[]` | Origin(s) to query (repeat or comma-separate, max 50) |
| `--project` | string | — | BigQuery project ID |
| `--to` | string | — | End month YYYYMM (overrides latest available from CrUX) |

## `crux history`

Query the CrUX History API (weekly time series) for an origin or url

| flag | type | default | description |
| --- | --- | --- | --- |
| `--api-key` | string | — | CrUX API key (overrides CRUX_API_KEY env and config) |
| `-c`, `--connection` | string | `all` | Effective connection type: 4g / 3g / 2g / slow-2g / offline / all (all = aggregate) |
| `-d`, `--device` | string | `all` | Form factor: phone / desktop / tablet / all (all = aggregate) |
| `-f`, `--format` | string | — | Output format: table / json / csv |
| `--metrics` | string | — | Metrics to query, comma-separated (default: lcp,cls,inp) Available: lcp,cls,inp,fcp,ttfb,rtt |
| `-o`, `--origin` | stringArray | `[]` | Origin(s) to query (repeat or comma-separate) |
| `--periods` | int | `0` | Number of weekly collection periods, 1-40 (default 25) |
| `-u`, `--url` | stringArray | `[]` | URL(s) to query a specific page (repeat or comma-separate) |

## `crux record`

Query the CrUX API for the latest record (28-day snapshot) of an origin or url

| flag | type | default | description |
| --- | --- | --- | --- |
| `--api-key` | string | — | CrUX API key (overrides CRUX_API_KEY env and config) |
| `-c`, `--connection` | string | `all` | Effective connection type: 4g / 3g / 2g / slow-2g / offline / all (all = aggregate) |
| `-d`, `--device` | string | `all` | Form factor: phone / desktop / tablet / all (all = aggregate) |
| `-f`, `--format` | string | — | Output format: table / json / csv |
| `--metrics` | string | — | Metrics to query, comma-separated (default: lcp,cls,inp) Available: lcp,cls,inp,fcp,ttfb,rtt |
| `-o`, `--origin` | stringArray | `[]` | Origin(s) to query (repeat or comma-separate) |
| `-u`, `--url` | stringArray | `[]` | URL(s) to query a specific page (repeat or comma-separate) |

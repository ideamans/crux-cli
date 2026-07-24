# JSON output schemas

## `crux device -f json` (BigQuery)

An array of row objects, one row per (origin, yyyymm, device).

| field | type | meaning |
| --- | --- | --- |
| `origin` | string | queried origin |
| `yyyymm` | string | month, e.g. `"202504"` |
| `device` | string | `phone` \| `desktop` \| `tablet` |

Each metric contributes three bucket **densities** (fractions 0–1) plus one p75
value. The prefixes differ per metric family:

| metrics | good | needs improvement | poor | p75 |
| --- | --- | --- | --- | --- |
| `lcp` `inp` `fcp` `ttfb` `ol` `fid` | `fast_<m>` | `avg_<m>` | `slow_<m>` | `p75_<m>` (integer ms) |
| `cls` | `small_cls` | `medium_cls` | `large_cls` | `p75_cls` (float ratio) |
| `rtt` | `low_rtt` | `medium_rtt` | `high_rtt` | `p75_rtt` (integer ms) |

`fast_<m>` (and `small_cls` / `low_rtt`) is what "Good%" refers to.

**All 8 metrics are always present in JSON.** The `--metrics` flag only filters
table and CSV columns, never the JSON payload. Missing data appears as `0`.

## `crux history -f json` and `crux record -f json` (CrUX API)

An array of record objects, one per queried target + form factor.

| field | type | meaning |
| --- | --- | --- |
| `origin` | string | present when queried by origin |
| `url` | string | present when queried by url |
| `form_factor` | string | `phone` \| `desktop` \| `tablet` \| `all` |
| `periods` | array | 1 element for `record`; up to 40 for `history` |

**`periods` is oldest-first in JSON, while the table view prints newest first.**
Do not assume index 0 is the latest.

Each period:

| field | type | meaning |
| --- | --- | --- |
| `first_date` | string | `YYYY-MM-DD`, start of the 28-day window |
| `last_date` | string | `YYYY-MM-DD`, end of the window — use this as the label |
| `metrics` | object | keyed by short metric key (`lcp`, `cls`, `inp`, `fcp`, `ttfb`, `rtt`) |

Each metric value:

| field | type | meaning |
| --- | --- | --- |
| `good` | float 0–1 | density of the good bucket ("Good%") |
| `needs_improvement` | float 0–1 | density of the NI bucket |
| `poor` | float 0–1 | density of the poor bucket |
| `p75` | number | 75th percentile; ms for time metrics, unitless ratio for `cls`. `0` may mean missing |

Only the metrics you requested (or the defaults) appear under `metrics`.

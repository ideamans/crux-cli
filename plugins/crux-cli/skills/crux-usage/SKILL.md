---
name: crux-usage
description: Look up real-user web performance data with the crux CLI — Core Web Vitals (LCP, INP, CLS) and related metrics from the Chrome UX Report, for any origin or specific URL, as a monthly history from BigQuery or a weekly series from the CrUX API. Use when the user asks how fast a site is for real users, about Core Web Vitals or field data, or wants to compare performance between sites or over time.
license: MIT
compatibility: Requires the `crux` binary on PATH — run the crux-install skill if it is missing. The history and record subcommands need a CrUX API key (free). The device subcommand needs Google Cloud ADC and a billing project, because it queries BigQuery at the user's expense.
allowed-tools: Bash(crux:*) Bash(jq:*) Bash(command:*) Read Write
---

# crux-usage

Query Chrome UX Report field data — what real Chrome users actually experienced,
as opposed to a synthetic lab measurement like Lighthouse.

## 1. Confirm the tool and pick the data source

```bash
command -v crux && crux --version
crux auth status
```

Missing binary? Run the `crux-install` skill.

**The subcommand determines the data source, and they are not interchangeable:**

| You need | Use | Cost |
| --- | --- | --- |
| A specific page/URL | `crux history` / `crux record` | free API key |
| Weekly trend, up to 40 periods | `crux history` | free API key |
| Just the latest snapshot | `crux record` | free API key |
| Monthly history, many origins, consistent buckets | `crux device` | **billed BigQuery** |
| `ol` or `fid` metrics | `crux device` | **billed BigQuery** |

**`crux device` charges the user's Google Cloud project.** Say so before running
one, especially for many origins or a long window, and do not pass `--no-cache`
unless the user wants fresh data — the cache exists to avoid repeat charges.

If `crux auth status` shows nothing configured, tell the user what is needed:
`crux auth set-api-key <KEY>` for the API path (create the key in Google Cloud
Console with the "Chrome UX Report API" enabled), or
`gcloud auth application-default login` plus `crux auth set-project <ID>` for
BigQuery. Never echo an API key back into the conversation.

## 2. Read the reference before composing a query

```bash
crux llm | head -80        # data sources, auth, agent rules
crux llm | grep -A 30 'JSON output schemas'
```

The reference is embedded in the binary (~280 lines) and matches the installed
version exactly.

## 3. Query with `-f json`

```bash
crux record  -o https://example.com -f json
crux history -o https://example.com --periods 40 -f json
crux device  -o https://example.com -d phone --months 6 -f json
```

The default `table` format is for humans. Parse only the JSON.

## 4. Interpret carefully

These are the mistakes that produce confidently wrong answers:

- **`0` usually means "no data", not "instant".** A density or p75 of zero
  almost always signals insufficient traffic for that bucket. Report it as
  missing data.
- **Densities are fractions in [0,1].** `0.83` is 83%. Multiply before showing a
  percentage.
- **`-d all` means opposite things across subcommands.** `crux device -d all`
  returns one row *per device*; `crux history -d all` returns *one aggregated*
  record. Reading one as the other changes every number's meaning.
- **`history` JSON is oldest-first**, while the table view prints newest first.
  Index 0 is not the latest period.
- **`cls` is unitless.** Never label it "ms". Everything else is milliseconds.
- **In `crux device` JSON all 8 metrics are always present** regardless of
  `--metrics`, which only filters table/CSV columns.
- A target with no data is **skipped with a notice, not an error**. Check that
  every origin or URL you asked for actually came back.

Core Web Vitals thresholds (good / poor): LCP ≤2500 / >4000 ms, INP ≤200 / >500
ms, CLS ≤0.1 / >0.25. "Good%" is the share in the fastest bucket — the number
worth quoting.

## 5. Report

Give the p75 and the good-share together, with the period the data covers
(`last_date`, or the `yyyymm`). A p75 without its window is not interpretable,
and CrUX windows are 28 days rolling (API) or calendar months (BigQuery).

## Failure modes

| Symptom | Cause | Fix |
| --- | --- | --- |
| `command not found: crux` | not installed | run the `crux-install` skill |
| API calls rejected | no API key | `crux auth set-api-key <KEY>` |
| BigQuery errors about credentials or project | ADC or project missing | `gcloud auth application-default login`, `crux auth set-project <ID>` |
| a requested origin is absent from the output | CrUX has no data for it | expected — say so; it usually means too little traffic |
| rate limit errors on many targets | ~150 queries/min per key, one request per target | query fewer targets, or space the calls |
| more than 50 origins rejected | `crux device` caps at 50 | split into batches |

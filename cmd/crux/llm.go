package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var flagLLM bool

func init() {
	// Persistent flag so `crux --llm` and `crux <subcommand> --llm` both work.
	rootCmd.PersistentFlags().BoolVar(&flagLLM, "llm", false,
		"Print a detailed reference for LLM / AI-agent use (data sources, flags, JSON field meanings) and exit")

	// Intercept --llm before any command body runs.
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if flagLLM {
			fmt.Print(llmGuide)
			os.Exit(0)
		}
		return nil
	}

	// Bare `crux` (no subcommand) prints standard help instead of erroring.
	rootCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	}
}

// llmGuide is a long-form, machine-and-human readable reference intended to let
// an AI agent drive crux autonomously without prior knowledge of CrUX.
const llmGuide = `crux — reference for LLM / AI agents
====================================

WHAT THIS TOOL DOES
  crux queries Chrome UX Report (CrUX) real-user performance data (Core Web
  Vitals and related metrics) from the command line. There are TWO independent
  data sources, selected by subcommand:

  1) BigQuery   -> subcommand: device
     - Monthly, ORIGIN-level aggregates from the public BigQuery table
       chrome-ux-report.materialized.device_summary.
     - Requires a Google Cloud project; BigQuery query cost is billed to it.
     - Local caching per (origin, period). Origin only (NO per-URL queries).
     - Time granularity: calendar month (YYYYMM).
     - Extra metrics available here only: ol (onload), fid (legacy).

  2) CrUX API   -> subcommands: history, record
     - Free public REST API (https://developer.chrome.com/docs/crux/api).
       Requires only an API key. No GCP project, no billing.
     - Works on an ORIGIN or a specific URL (page).
     - history = weekly TIME SERIES (up to 40 collection periods).
       record  = the single LATEST record (28-day snapshot).
     - Time granularity: 28-day rolling window, refreshed weekly (Mondays).

HOW TO CHOOSE
  - Need data for a specific PAGE/URL?            -> history or record (API).
  - Have only an API key, no GCP project?         -> history or record (API).
  - Need long monthly history or many origins at  -> device (BigQuery).
    once with consistent monthly buckets?
  - Want ol/fid metrics?                          -> device (BigQuery only).

AGENT TIPS
  - Always pass "-f json" for machine parsing. JSON schemas are documented below.
  - All density / fraction fields are proportions in [0,1] (0.83 means 83%).
  - A density or p75 of 0 usually means missing / insufficient data, not a real 0.

AUTHENTICATION
  BigQuery (device):
    - Uses Application Default Credentials: run once
        gcloud auth application-default login
    - Set the billing project:
        crux auth set-project <PROJECT_ID>      (or env CRUX_PROJECT, or --project)
  CrUX API (history / record):
    - Needs an API key. Provide via any of (priority high->low):
        --api-key <KEY>  >  env CRUX_API_KEY  >  config file (crux auth set-api-key)
    - Create a key in Google Cloud Console and enable the "Chrome UX Report API".
  Check current state:
        crux auth status

COMMANDS & FLAGS
  crux device  --origin <o> [--origin ...] [flags]   (BigQuery; origin only, max 50)
    -o, --origin     origin(s); repeat or comma-separate (required, max 50)
    -d, --device     phone | desktop | tablet | all   (default all; all = per-device rows)
    -m, --months     number of months back            (default 12)
        --from       start month YYYYMM (overrides --months)
        --to         end month YYYYMM   (default: latest available CrUX month)
        --metrics    comma list of: lcp,cls,inp,fcp,ttfb,ol,rtt,fid (default lcp,cls,inp)
        --full-metrics  all except fid
    -f, --format     table | json | csv               (default table)
        --project    BigQuery project id
        --no-cache   bypass local cache

  crux history  (--origin <o> | --url <u>) [flags]    (CrUX API; weekly time series)
  crux record   (--origin <o> | --url <u>) [flags]    (CrUX API; latest snapshot)
    -o, --origin     origin(s); repeat or comma-separate
    -u, --url        url(s) for a specific page; repeat or comma-separate
                     (at least one --origin or --url is required; may be mixed)
    -d, --device     phone | desktop | tablet | all   (default all)
                     NOTE: here "all" = AGGREGATE across devices (one record),
                     NOT a per-device breakdown like "crux device".
    -c, --connection 4g | 3g | 2g | slow-2g | offline | all  (default all)
                     effective connection type filter; "all" = aggregate.
        --periods    (history only) weekly periods, 1-40 (default 25)
        --metrics    comma list of: lcp,cls,inp,fcp,ttfb,rtt (default lcp,cls,inp)
                     (ol and fid are NOT available via the API)
    -f, --format     table | json | csv               (default table)
        --api-key    CrUX API key (overrides env and config)

  crux auth status | set-project <id> | set-api-key <key>
  crux cache dir | list | clear [--origin <o>]

METRICS & CORE WEB VITALS THRESHOLDS
  key   name                          Good        Poor        unit    notes
  lcp   Largest Contentful Paint      <=2500      >4000       ms      Core Web Vital
  inp   Interaction to Next Paint     <=200       >500        ms      Core Web Vital
  cls   Cumulative Layout Shift       <=0.1       >0.25       ratio   Core Web Vital (unitless)
  fcp   First Contentful Paint        <=1800      >3000       ms
  ttfb  Time to First Byte            <=600       >1200       ms      API metric is "experimental"
  rtt   Round Trip Time               <=75        >275        ms
  ol    Onload                        -           -           ms      BigQuery only
  fid   First Input Delay             <=100       >300        ms      legacy; BigQuery only
  "Good%" = fraction of experiences in the good bucket (the first/fastest bucket).

JSON SCHEMA — "crux device -f json"  (BigQuery)
  Output: an array of row objects. One row per (origin, yyyymm, device).
  Fields:
    origin (string)        queried origin
    yyyymm (string)        month, e.g. "202504"
    device (string)        "phone" | "desktop" | "tablet"
  Per metric, three bucket DENSITIES (fraction 0-1) + one p75 value:
    lcp/inp/fcp/ttfb/ol/fid use prefixes fast_/avg_/slow_/p75_:
       fast_<m>  = density of GOOD bucket  (this is "Good%")
       avg_<m>   = density of NEEDS-IMPROVEMENT bucket
       slow_<m>  = density of POOR bucket
       p75_<m>   = 75th percentile, integer milliseconds
    cls uses small_/medium_/large_/p75_:
       small_cls = good density, medium_cls = ni density, large_cls = poor density
       p75_cls   = 75th percentile, FLOAT ratio (unitless), e.g. 0.08
    rtt uses low_/medium_/high_/p75_:
       low_rtt = good, medium_rtt = ni, high_rtt = poor; p75_rtt = ms integer
  All 8 metrics are always present in JSON (the --metrics flag only filters
  table/csv columns, not the JSON payload). Missing data appears as 0.

JSON SCHEMA — "crux history -f json" and "crux record -f json"  (CrUX API)
  Output: an array of record objects. One object per queried target+form factor.
  Fields:
    origin (string)        present when queried by origin
    url (string)           present when queried by url
    form_factor (string)   "phone" | "desktop" | "tablet" | "all"
    periods (array)        1 element for "record"; up to 40 for "history".
                           Chronological order: OLDEST first in JSON.
                           (The table view prints NEWEST first.)
      each period:
        first_date (string)  "YYYY-MM-DD", start of the 28-day window
        last_date  (string)  "YYYY-MM-DD", end of the window (use as the label)
        metrics (object)     keyed by short metric key (lcp, cls, inp, fcp, ttfb, rtt)
          each metric value:
            good (float 0-1)               density of good bucket ("Good%")
            needs_improvement (float 0-1)  density of NI bucket
            poor (float 0-1)               density of poor bucket
            p75 (number)                   75th percentile; ms for time metrics,
                                           unitless ratio for cls. 0 may mean missing.
  Only the metrics you requested (or the defaults) appear under "metrics".

  API metric short-key -> CrUX API metric name (for cross-referencing the API):
    lcp  -> largest_contentful_paint
    cls  -> cumulative_layout_shift
    inp  -> interaction_to_next_paint
    fcp  -> first_contentful_paint
    ttfb -> experimental_time_to_first_byte
    rtt  -> round_trip_time

GOTCHAS / LIMITS
  - "crux device -d all" returns one row PER device; "crux history/record -d all"
    returns ONE aggregated record (no per-device split).
  - device: max 50 origins per invocation. history: --periods 1-40 (default 25).
  - CrUX API rate limit: ~150 queries/minute per key. Each origin/url is a
    separate request; querying N targets makes N requests.
  - If the API has no data for a target it is skipped with a notice (not an error).
  - cls is a unitless ratio; all other metrics are milliseconds.

EXAMPLES
  crux device  -o https://example.com -d phone --months 6 -f json
  crux history -o https://web.dev --periods 40 -f json
  crux history -u https://web.dev/learn --metrics lcp,cls,inp,fcp,ttfb,rtt -f json
  crux record  -o https://web.dev -d desktop -f json
`

package runner

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ideamans/crux-cli/internal/cache"
	"github.com/ideamans/crux-cli/internal/crux"
	"github.com/ideamans/crux-cli/internal/formatter"
)

// DeviceOptions holds resolved parameters for a device query run.
type DeviceOptions struct {
	Origins   []string
	Device    string    // phone / desktop / tablet / all
	MonthFrom string    // YYYYMM; empty until resolved
	MonthTo   string    // YYYYMM; empty = latest from CrUX
	Format    string    // table / json / csv
	Metrics   []string  // e.g. ["lcp","cls","inp"]; nil = all
	NoCache   bool
	Months    int       // used to compute MonthFrom when MonthTo is known
	Progress  io.Writer // status/info messages; if nil, falls back to w
}

// RunDevice resolves the query period, applies the cache, queries BigQuery for
// misses, and writes results to w.
func RunDevice(ctx context.Context, bqFactory BQClientFactory, ca CacheStore, opts DeviceOptions, w io.Writer) error {
	// progress writer: status messages go here
	progress := opts.Progress
	if progress == nil {
		progress = w
	}

	// lazy singleton BQ client
	var client BQClient
	getBQ := func() (BQClient, error) {
		if client != nil {
			return client, nil
		}
		c, err := bqFactory(ctx)
		if err != nil {
			return nil, err
		}
		client = c
		return client, nil
	}
	defer func() {
		if client != nil {
			client.Close()
		}
	}()

	// ── 1. Resolve monthTo ──────────────────────────────────────────────────
	if opts.MonthTo == "" {
		lm, err := ca.GetLatestMonth()
		if err != nil {
			return fmt.Errorf("read latest_month cache: %w", err)
		}

		if !opts.NoCache && !ca.IsStale(lm) {
			opts.MonthTo = lm.Yyyymm
		} else {
			fmt.Fprintln(progress, "Checking latest CrUX month via BigQuery...")
			bq, err := getBQ()
			if err != nil {
				return err
			}

			yyyymm, err := bq.QueryLatestMonth(ctx)
			if err != nil {
				return fmt.Errorf("fetch latest month: %w", err)
			}
			fmt.Fprintf(progress, "Latest month: %s\n", crux.FormatMonth(yyyymm))

			if !opts.NoCache {
				newLM := &cache.LatestMonth{Yyyymm: yyyymm, CheckedAt: time.Now().UTC()}
				if err := ca.SaveLatestMonth(newLM); err != nil {
					fmt.Fprintf(progress, "warning: could not save latest_month cache: %v\n", err)
				}
			}
			opts.MonthTo = yyyymm
		}
	}

	// ── 2. Resolve monthFrom ────────────────────────────────────────────────
	if opts.MonthFrom == "" {
		months := opts.Months
		if months == 0 {
			months = 12
		}
		from, err := crux.SubtractMonths(opts.MonthTo, months-1)
		if err != nil {
			return err
		}
		opts.MonthFrom = from
	}

	// ── 3. Cache check: split origins into hits and misses ──────────────────
	var allRows []crux.DeviceRow
	var missOrigins []string

	if opts.NoCache {
		missOrigins = opts.Origins
	} else {
		for _, origin := range opts.Origins {
			cached, err := ca.GetDevice(origin, opts.MonthFrom, opts.MonthTo)
			if err != nil {
				return fmt.Errorf("read cache for %s: %w", origin, err)
			}
			if cached != nil {
				allRows = append(allRows, cached...)
			} else {
				missOrigins = append(missOrigins, origin)
			}
		}
	}

	// ── 4. Query BigQuery for misses ────────────────────────────────────────
	if len(missOrigins) > 0 {
		fmt.Fprintf(progress, "Querying BigQuery for %d origin(s) [%s – %s]...\n",
			len(missOrigins), crux.FormatMonth(opts.MonthFrom), crux.FormatMonth(opts.MonthTo))

		bq, err := getBQ()
		if err != nil {
			return err
		}

		fetched, err := bq.QueryDevice(ctx, missOrigins, opts.MonthFrom, opts.MonthTo, opts.Device)
		if err != nil {
			return fmt.Errorf("bigquery: %w", err)
		}

		// Group fetched rows by origin and cache each
		if !opts.NoCache {
			byOrigin := make(map[string][]crux.DeviceRow)
			for _, r := range fetched {
				byOrigin[r.Origin] = append(byOrigin[r.Origin], r)
			}
			for _, origin := range missOrigins {
				rows := byOrigin[origin] // empty slice if origin not in CrUX
				if err := ca.SaveDevice(origin, opts.MonthFrom, opts.MonthTo, rows); err != nil {
					fmt.Fprintf(progress, "warning: could not cache %s: %v\n", origin, err)
				}
			}
		}
		allRows = append(allRows, fetched...)
	}

	if len(allRows) == 0 {
		fmt.Fprintln(w, "No data found for the specified origins and period.")
		return nil
	}

	// ── 5. Output ────────────────────────────────────────────────────────────
	fmt.Fprintln(w)
	switch opts.Format {
	case "json":
		return formatter.PrintJSON(w, allRows)
	case "csv":
		return formatter.PrintCSV(w, allRows, opts.Metrics)
	default:
		formatter.PrintTable(w, allRows, opts.Metrics)
		return nil
	}
}

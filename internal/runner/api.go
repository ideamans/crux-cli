package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ideamans/crux-cli/internal/cruxapi"
)

// APIClient abstracts the CrUX REST API client.
type APIClient interface {
	QueryRecord(ctx context.Context, q cruxapi.Query) (*cruxapi.Record, error)
	QueryHistoryRecord(ctx context.Context, q cruxapi.Query) (*cruxapi.Record, error)
}

// APIOptions holds resolved parameters for a CrUX API run.
type APIOptions struct {
	Origins    []string
	URLs       []string
	Device     string   // phone / desktop / tablet / all
	Connection string   // 4g / 3g / 2g / slow-2g / offline / all
	Metrics    []string // CLI keys; nil = default (lcp,cls,inp)
	Format     string   // table / json / csv
	History    bool     // true = queryHistoryRecord, false = queryRecord
	Periods    int      // history collectionPeriodCount (1-40; 0 = API default)
	Progress   io.Writer
}

// formFactor maps the CLI device flag to the API form factor enum.
// "all" returns "" so the API aggregates across all devices.
func formFactor(device string) string {
	switch strings.ToLower(device) {
	case "phone":
		return "PHONE"
	case "desktop":
		return "DESKTOP"
	case "tablet":
		return "TABLET"
	default:
		return ""
	}
}

// effectiveConnectionType maps the CLI connection flag to the API enum.
// "all" (or empty) returns "" so the API aggregates across all connection types.
// The second return value is false for unrecognized values.
func effectiveConnectionType(conn string) (string, bool) {
	switch strings.ToLower(conn) {
	case "", "all":
		return "", true
	case "4g":
		return "4G", true
	case "3g":
		return "3G", true
	case "2g":
		return "2G", true
	case "slow-2g":
		return "slow-2G", true
	case "offline":
		return "offline", true
	default:
		return "", false
	}
}

// RunAPI queries the CrUX API for every origin/url target and writes the result.
func RunAPI(ctx context.Context, client APIClient, opts APIOptions, w io.Writer) error {
	progress := opts.Progress
	if progress == nil {
		progress = w
	}

	apiMetrics := cruxapi.APINames(opts.Metrics)
	ff := formFactor(opts.Device)
	ect, ok := effectiveConnectionType(opts.Connection)
	if !ok {
		return fmt.Errorf("invalid --connection %q: must be 4g, 3g, 2g, slow-2g, offline, or all", opts.Connection)
	}

	var queries []cruxapi.Query
	for _, o := range opts.Origins {
		queries = append(queries, cruxapi.Query{
			Origin:                  o,
			FormFactor:              ff,
			EffectiveConnectionType: ect,
			Metrics:                 apiMetrics,
			CollectionPeriodCount:   opts.Periods,
		})
	}
	for _, u := range opts.URLs {
		queries = append(queries, cruxapi.Query{
			URL:                     u,
			FormFactor:              ff,
			EffectiveConnectionType: ect,
			Metrics:                 apiMetrics,
			CollectionPeriodCount:   opts.Periods,
		})
	}

	if len(queries) == 0 {
		return fmt.Errorf("at least one --origin or --url is required")
	}

	var records []*cruxapi.Record
	for _, q := range queries {
		var (
			rec *cruxapi.Record
			err error
		)
		if opts.History {
			rec, err = client.QueryHistoryRecord(ctx, q)
		} else {
			rec, err = client.QueryRecord(ctx, q)
		}
		if errors.Is(err, cruxapi.ErrNotFound) {
			fmt.Fprintf(progress, "No CrUX data for %s (%s)\n", q.Label(), opts.Device)
			continue
		}
		if err != nil {
			return err
		}
		records = append(records, rec)
	}

	if len(records) == 0 {
		fmt.Fprintln(w, "No data found for the specified targets.")
		return nil
	}

	fmt.Fprintln(w)
	switch opts.Format {
	case "json":
		return cruxapi.PrintJSON(w, records)
	case "csv":
		return cruxapi.PrintCSV(w, records, opts.Metrics)
	default:
		cruxapi.PrintTable(w, records, opts.Metrics)
		return nil
	}
}

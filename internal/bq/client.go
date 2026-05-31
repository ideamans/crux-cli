package bq

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/ideamans/crux-cli/internal/crux"
	"google.golang.org/api/iterator"
)

const deviceSummaryTable = "chrome-ux-report.materialized.device_summary"

// Client wraps the BigQuery client for CrUX queries.
type Client struct {
	bq *bigquery.Client
}

func New(ctx context.Context, projectID string) (*Client, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("bigquery: %w", err)
	}
	return &Client{bq: client}, nil
}

func (c *Client) Close() {
	c.bq.Close()
}

// QueryLatestMonth returns the most recent YYYYMM available in device_summary.
func (c *Client) QueryLatestMonth(ctx context.Context) (string, error) {
	sql := fmt.Sprintf(
		"SELECT CAST(MAX(yyyymm) AS STRING) AS yyyymm FROM `%s`",
		deviceSummaryTable,
	)
	it, err := c.bq.Query(sql).Read(ctx)
	if err != nil {
		return "", fmt.Errorf("query latest month: %w", err)
	}
	var row struct {
		Yyyymm string `bigquery:"yyyymm"`
	}
	if err := it.Next(&row); err != nil {
		return "", fmt.Errorf("read latest month: %w", err)
	}
	return row.Yyyymm, nil
}

// QueryDevice fetches device_summary rows for the given origins and period.
// device may be "phone", "desktop", "tablet", or "all" (no filter).
func (c *Client) QueryDevice(ctx context.Context, origins []string, monthFrom, monthTo, device string) ([]crux.DeviceRow, error) {
	monthFromInt, err := crux.YYYYMMToInt(monthFrom)
	if err != nil {
		return nil, err
	}
	monthToInt, err := crux.YYYYMMToInt(monthTo)
	if err != nil {
		return nil, err
	}

	params := []bigquery.QueryParameter{
		{Name: "origins", Value: origins},
		{Name: "monthFrom", Value: monthFromInt},
		{Name: "monthTo", Value: monthToInt},
	}

	deviceClause := ""
	if device != "all" {
		deviceClause = "AND device = @device"
		params = append(params, bigquery.QueryParameter{Name: "device", Value: device})
	}

	// COALESCE converts NULLs (missing metrics) to zero.
	// Explicit casts fix BigQuery NUMERIC/INTEGER → Go type mismatches.
	sql := fmt.Sprintf(`
SELECT
  origin,
  CAST(yyyymm AS STRING) AS yyyymm,
  device,
  COALESCE(CAST(fast_lcp   AS FLOAT64), 0.0) AS fast_lcp,
  COALESCE(CAST(avg_lcp    AS FLOAT64), 0.0) AS avg_lcp,
  COALESCE(CAST(slow_lcp   AS FLOAT64), 0.0) AS slow_lcp,
  COALESCE(CAST(p75_lcp    AS INT64),   0)   AS p75_lcp,
  COALESCE(CAST(small_cls  AS FLOAT64), 0.0) AS small_cls,
  COALESCE(CAST(medium_cls AS FLOAT64), 0.0) AS medium_cls,
  COALESCE(CAST(large_cls  AS FLOAT64), 0.0) AS large_cls,
  COALESCE(CAST(p75_cls    AS FLOAT64), 0.0) AS p75_cls,
  COALESCE(CAST(fast_inp   AS FLOAT64), 0.0) AS fast_inp,
  COALESCE(CAST(avg_inp    AS FLOAT64), 0.0) AS avg_inp,
  COALESCE(CAST(slow_inp   AS FLOAT64), 0.0) AS slow_inp,
  COALESCE(CAST(p75_inp    AS INT64),   0)   AS p75_inp,
  COALESCE(CAST(fast_fcp   AS FLOAT64), 0.0) AS fast_fcp,
  COALESCE(CAST(avg_fcp    AS FLOAT64), 0.0) AS avg_fcp,
  COALESCE(CAST(slow_fcp   AS FLOAT64), 0.0) AS slow_fcp,
  COALESCE(CAST(p75_fcp    AS INT64),   0)   AS p75_fcp,
  COALESCE(CAST(fast_ttfb  AS FLOAT64), 0.0) AS fast_ttfb,
  COALESCE(CAST(avg_ttfb   AS FLOAT64), 0.0) AS avg_ttfb,
  COALESCE(CAST(slow_ttfb  AS FLOAT64), 0.0) AS slow_ttfb,
  COALESCE(CAST(p75_ttfb   AS INT64),   0)   AS p75_ttfb,
  COALESCE(CAST(fast_fid   AS FLOAT64), 0.0) AS fast_fid,
  COALESCE(CAST(avg_fid    AS FLOAT64), 0.0) AS avg_fid,
  COALESCE(CAST(slow_fid   AS FLOAT64), 0.0) AS slow_fid,
  COALESCE(CAST(p75_fid    AS INT64),   0)   AS p75_fid,
  COALESCE(CAST(fast_ol    AS FLOAT64), 0.0) AS fast_ol,
  COALESCE(CAST(avg_ol     AS FLOAT64), 0.0) AS avg_ol,
  COALESCE(CAST(slow_ol    AS FLOAT64), 0.0) AS slow_ol,
  COALESCE(CAST(p75_ol     AS INT64),   0)   AS p75_ol,
  COALESCE(CAST(low_rtt    AS FLOAT64), 0.0) AS low_rtt,
  COALESCE(CAST(medium_rtt AS FLOAT64), 0.0) AS medium_rtt,
  COALESCE(CAST(high_rtt   AS FLOAT64), 0.0) AS high_rtt,
  COALESCE(CAST(p75_rtt    AS INT64),   0)   AS p75_rtt
FROM `+"`%s`"+`
WHERE
  origin IN UNNEST(@origins)
  AND CAST(yyyymm AS INT64) >= @monthFrom
  AND CAST(yyyymm AS INT64) <= @monthTo
  %s
ORDER BY origin, yyyymm DESC, device
`, deviceSummaryTable, deviceClause)

	q := c.bq.Query(strings.TrimSpace(sql))
	q.Parameters = params

	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("query device: %w", err)
	}

	var rows []crux.DeviceRow
	for {
		var row crux.DeviceRow
		if err := it.Next(&row); err == iterator.Done {
			break
		} else if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

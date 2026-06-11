package cruxapi

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/olekukonko/tablewriter"
)

// selectMetricDefs resolves CLI metric keys to ordered MetricDefs.
// An empty/nil keys list falls back to DefaultMetrics.
func selectMetricDefs(keys []string) []MetricDef {
	if len(keys) == 0 {
		keys = DefaultMetrics
	}
	want := make(map[string]bool, len(keys))
	for _, k := range keys {
		if m, ok := MetricByKey(k); ok {
			want[m.Key] = true
		}
	}
	var out []MetricDef
	for _, m := range Metrics {
		if want[m.Key] {
			out = append(out, m)
		}
	}
	return out
}

func pct(v float64) string {
	if v == 0 {
		return "-"
	}
	return fmt.Sprintf("%.0f%%", v*100)
}

func p75Str(d MetricDef, mv MetricValue) string {
	if !mv.HasP75 {
		return "-"
	}
	if d.IsCLS {
		return fmt.Sprintf("%.3f", mv.P75)
	}
	return fmt.Sprintf("%.0fms", mv.P75)
}

// PrintTable writes one section per record with a period-by-period table.
func PrintTable(w io.Writer, records []*Record, metrics []string) {
	defs := selectMetricDefs(metrics)
	headers := []string{"Period (end)"}
	for _, d := range defs {
		headers = append(headers, d.Label+" p75", d.Label+" Good%")
	}

	for _, rec := range records {
		if rec == nil || len(rec.Periods) == 0 {
			continue
		}
		fmt.Fprintf(w, "\n=== %s (%s) ===\n", rec.Target(), rec.FormFactor)
		t := tablewriter.NewWriter(w)
		t.SetHeader(headers)
		t.SetBorder(false)
		t.SetColumnSeparator("  ")
		t.SetHeaderLine(true)
		// Newest first for display.
		for i := len(rec.Periods) - 1; i >= 0; i-- {
			p := rec.Periods[i]
			row := []string{p.LastDate}
			for _, d := range defs {
				mv := p.Metrics[d.Key]
				row = append(row, p75Str(d, mv), pct(mv.Good))
			}
			t.Append(row)
		}
		t.Render()
	}
}

// PrintJSON writes all records as a JSON array.
func PrintJSON(w io.Writer, records []*Record) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(records)
}

// PrintCSV writes records as CSV with one row per (target, form factor, period).
func PrintCSV(w io.Writer, records []*Record, metrics []string) error {
	defs := selectMetricDefs(metrics)
	cw := csv.NewWriter(w)

	header := []string{"target", "form_factor", "first_date", "last_date"}
	for _, d := range defs {
		header = append(header, d.Label+" p75", d.Label+" Good%")
	}
	if err := cw.Write(header); err != nil {
		return err
	}

	for _, rec := range records {
		if rec == nil {
			continue
		}
		for _, p := range rec.Periods {
			record := []string{rec.Target(), rec.FormFactor, p.FirstDate, p.LastDate}
			for _, d := range defs {
				mv := p.Metrics[d.Key]
				record = append(record, p75Str(d, mv), pct(mv.Good))
			}
			if err := cw.Write(record); err != nil {
				return err
			}
		}
	}
	cw.Flush()
	return cw.Error()
}

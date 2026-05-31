package formatter

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/ideamans/crux-cli/internal/crux"
	"github.com/olekukonko/tablewriter"
)

// metricDef defines a single metric's display columns.
type metricDef struct {
	key     string // flag name: lcp, cls, inp, ...
	p75Col  string // header for p75
	goodCol string // header for good%
	p75Val  func(r crux.DeviceRow) string
	goodVal func(r crux.DeviceRow) string
}

func pct(v float64) string {
	if v == 0 {
		return "-"
	}
	return fmt.Sprintf("%.0f%%", v*100)
}

func ms(v int64) string {
	if v == 0 {
		return "-"
	}
	return fmt.Sprintf("%dms", v)
}

func clsVal(v float64) string {
	if v == 0 {
		return "-"
	}
	return fmt.Sprintf("%.3f", v)
}

var allMetrics = []metricDef{
	{"lcp", "LCP p75", "LCP Good%", func(r crux.DeviceRow) string { return ms(r.P75Lcp) }, func(r crux.DeviceRow) string { return pct(r.FastLcp) }},
	{"cls", "CLS p75", "CLS Good%", func(r crux.DeviceRow) string { return clsVal(r.P75Cls) }, func(r crux.DeviceRow) string { return pct(r.SmallCls) }},
	{"inp", "INP p75", "INP Good%", func(r crux.DeviceRow) string { return ms(r.P75Inp) }, func(r crux.DeviceRow) string { return pct(r.FastInp) }},
	{"fcp", "FCP p75", "FCP Good%", func(r crux.DeviceRow) string { return ms(r.P75Fcp) }, func(r crux.DeviceRow) string { return pct(r.FastFcp) }},
	{"ttfb", "TTFB p75", "TTFB Good%", func(r crux.DeviceRow) string { return ms(r.P75Ttfb) }, func(r crux.DeviceRow) string { return pct(r.FastTtfb) }},
	{"ol", "OL p75", "OL Good%", func(r crux.DeviceRow) string { return ms(r.P75Ol) }, func(r crux.DeviceRow) string { return pct(r.FastOl) }},
	{"rtt", "RTT p75", "RTT Good%", func(r crux.DeviceRow) string { return ms(r.P75Rtt) }, func(r crux.DeviceRow) string { return pct(r.LowRtt) }},
	{"fid", "FID p75", "FID Good%", func(r crux.DeviceRow) string { return ms(r.P75Fid) }, func(r crux.DeviceRow) string { return pct(r.FastFid) }},
}

// DefaultMetrics is the default set of metrics to display.
var DefaultMetrics = []string{"lcp", "cls", "inp"}

func selectMetrics(keys []string) []metricDef {
	if len(keys) == 0 {
		keys = DefaultMetrics
	}
	keySet := make(map[string]bool, len(keys))
	for _, k := range keys {
		keySet[strings.ToLower(k)] = true
	}
	var out []metricDef
	for _, m := range allMetrics {
		if keySet[m.key] {
			out = append(out, m)
		}
	}
	return out
}

// PrintTable writes a section per origin/device with a filtered metric table.
func PrintTable(w io.Writer, rows []crux.DeviceRow, metrics []string) {
	defs := selectMetrics(metrics)
	headers := []string{"Month"}
	for _, d := range defs {
		headers = append(headers, d.p75Col, d.goodCol)
	}

	byOrigin := groupByOrigin(rows)
	for _, origin := range originOrder(rows) {
		originRows := byOrigin[origin]
		byDevice := groupByDevice(originRows)
		for _, device := range deviceOrder(originRows) {
			deviceRows := byDevice[device]
			fmt.Fprintf(w, "\n=== %s (%s) ===\n", origin, device)
			t := tablewriter.NewWriter(w)
			t.SetHeader(headers)
			t.SetBorder(false)
			t.SetColumnSeparator("  ")
			t.SetHeaderLine(true)
			for _, r := range deviceRows {
				row := []string{crux.FormatMonth(r.Yyyymm)}
				for _, d := range defs {
					row = append(row, d.p75Val(r), d.goodVal(r))
				}
				t.Append(row)
			}
			t.Render()
		}
	}
}

// PrintJSON writes all rows as a JSON array (metrics filter has no effect).
func PrintJSON(w io.Writer, rows []crux.DeviceRow) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

// PrintCSV writes rows in CSV format with filtered metrics columns.
func PrintCSV(w io.Writer, rows []crux.DeviceRow, metrics []string) error {
	defs := selectMetrics(metrics)
	cw := csv.NewWriter(w)

	header := []string{"origin", "yyyymm", "device"}
	for _, d := range defs {
		header = append(header, d.p75Col, d.goodCol)
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, r := range rows {
		record := []string{r.Origin, crux.FormatMonth(r.Yyyymm), r.Device}
		for _, d := range defs {
			record = append(record, d.p75Val(r), d.goodVal(r))
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func groupByOrigin(rows []crux.DeviceRow) map[string][]crux.DeviceRow {
	m := make(map[string][]crux.DeviceRow)
	for _, r := range rows {
		m[r.Origin] = append(m[r.Origin], r)
	}
	return m
}

func originOrder(rows []crux.DeviceRow) []string {
	seen := make(map[string]bool)
	var order []string
	for _, r := range rows {
		if !seen[r.Origin] {
			seen[r.Origin] = true
			order = append(order, r.Origin)
		}
	}
	return order
}

func groupByDevice(rows []crux.DeviceRow) map[string][]crux.DeviceRow {
	m := make(map[string][]crux.DeviceRow)
	for _, r := range rows {
		m[r.Device] = append(m[r.Device], r)
	}
	return m
}

var devicePriority = map[string]int{"phone": 0, "desktop": 1, "tablet": 2}

func deviceOrder(rows []crux.DeviceRow) []string {
	seen := make(map[string]bool)
	var devices []string
	for _, r := range rows {
		if !seen[r.Device] {
			seen[r.Device] = true
			devices = append(devices, r.Device)
		}
	}
	// sort by priority
	for i := 0; i < len(devices); i++ {
		for j := i + 1; j < len(devices); j++ {
			if devicePriority[devices[i]] > devicePriority[devices[j]] {
				devices[i], devices[j] = devices[j], devices[i]
			}
		}
	}
	return devices
}

// splitOrigins splits comma-separated and repeated --origin values into a flat list.
func SplitOrigins(raw []string) []string {
	var result []string
	for _, s := range raw {
		for _, part := range strings.Split(s, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				result = append(result, part)
			}
		}
	}
	return result
}

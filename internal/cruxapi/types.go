package cruxapi

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

// MetricDef describes one CrUX metric we surface.
type MetricDef struct {
	Key     string // CLI key: lcp, cls, inp, fcp, ttfb, rtt
	APIName string // CrUX API metric name
	Label   string // display label: LCP, CLS, ...
	IsCLS   bool   // CLS p75 is an unitless ratio (formatted with decimals)
}

// Metrics is the ordered list of supported metrics.
var Metrics = []MetricDef{
	{Key: "lcp", APIName: "largest_contentful_paint", Label: "LCP"},
	{Key: "cls", APIName: "cumulative_layout_shift", Label: "CLS", IsCLS: true},
	{Key: "inp", APIName: "interaction_to_next_paint", Label: "INP"},
	{Key: "fcp", APIName: "first_contentful_paint", Label: "FCP"},
	{Key: "ttfb", APIName: "experimental_time_to_first_byte", Label: "TTFB"},
	{Key: "rtt", APIName: "round_trip_time", Label: "RTT"},
}

// DefaultMetrics is the default display/query set.
var DefaultMetrics = []string{"lcp", "cls", "inp"}

// MetricByKey returns the MetricDef for a CLI key (case-insensitive).
func MetricByKey(key string) (MetricDef, bool) {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, m := range Metrics {
		if m.Key == key {
			return m, true
		}
	}
	return MetricDef{}, false
}

// APINames maps CLI metric keys to CrUX API metric names. Unknown keys are skipped.
func APINames(keys []string) []string {
	var out []string
	for _, k := range keys {
		if m, ok := MetricByKey(k); ok {
			out = append(out, m.APIName)
		}
	}
	return out
}

// MetricValue holds the histogram densities and p75 for one metric in one period.
// Density buckets follow CrUX order: Good / Needs-Improvement / Poor.
type MetricValue struct {
	Good   float64 `json:"good"`
	NI     float64 `json:"needs_improvement"`
	Poor   float64 `json:"poor"`
	P75    float64 `json:"p75"`
	HasP75 bool    `json:"-"`
}

// Period is one collection period (28-day rolling window).
type Period struct {
	FirstDate string                 `json:"first_date"` // YYYY-MM-DD
	LastDate  string                 `json:"last_date"`  // YYYY-MM-DD
	Metrics   map[string]MetricValue `json:"metrics"`    // keyed by CLI metric key (lcp, cls, ...)
}

// Record is the normalized result for a single target (origin or url) and form factor.
type Record struct {
	Origin     string   `json:"origin,omitempty"`
	URL        string   `json:"url,omitempty"`
	FormFactor string   `json:"form_factor"` // phone / desktop / tablet / all
	Periods    []Period `json:"periods"`     // newest last (chronological); 1 entry for queryRecord
}

// Target returns the url if set, otherwise the origin.
func (r *Record) Target() string {
	if r.URL != "" {
		return r.URL
	}
	return r.Origin
}

// ── Parsing helpers ─────────────────────────────────────────────────────────

// parseNum parses a JSON number-or-string value. Returns ok=false for null,
// "NaN", or unparseable input.
func parseNum(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	s := strings.TrimSpace(string(raw))
	if s == "null" || s == `"NaN"` || s == "NaN" || s == `""` {
		return 0, false
	}
	s = strings.Trim(s, `"`)
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) {
		return 0, false
	}
	return v, true
}

// formFactorLabel maps an API form factor enum to a lowercase display label.
func formFactorLabel(ff string) string {
	switch strings.ToUpper(ff) {
	case "PHONE":
		return "phone"
	case "DESKTOP":
		return "desktop"
	case "TABLET":
		return "tablet"
	default:
		return "all"
	}
}

// metricKeyForAPIName returns the CLI key for an API metric name, "" if unknown.
func metricKeyForAPIName(apiName string) string {
	for _, m := range Metrics {
		if m.APIName == apiName {
			return m.Key
		}
	}
	return ""
}

// densityAt returns the density at bucket index i, or 0 if absent.
func densityAt(densities []float64, i int) float64 {
	if i < len(densities) {
		return densities[i]
	}
	return 0
}

// ── Conversion from raw API responses ───────────────────────────────────────

func recordFromResponse(q Query, raw *recordResponse) *Record {
	rec := &Record{
		Origin:     q.Origin,
		URL:        q.URL,
		FormFactor: formFactorLabel(raw.Record.Key.FormFactor),
	}
	if rec.FormFactor == "all" && q.FormFactor != "" {
		rec.FormFactor = formFactorLabel(q.FormFactor)
	}

	period := Period{
		FirstDate: raw.Record.CollectionPeriod.FirstDate.String(),
		LastDate:  raw.Record.CollectionPeriod.LastDate.String(),
		Metrics:   make(map[string]MetricValue),
	}
	for apiName, m := range raw.Record.Metrics {
		key := metricKeyForAPIName(apiName)
		if key == "" {
			continue
		}
		var mv MetricValue
		var buckets []float64
		for _, b := range m.Histogram {
			buckets = append(buckets, b.Density)
		}
		mv.Good = densityAt(buckets, 0)
		mv.NI = densityAt(buckets, 1)
		mv.Poor = densityAt(buckets, 2)
		if p75, ok := parseNum(m.Percentiles.P75); ok {
			mv.P75 = p75
			mv.HasP75 = true
		}
		period.Metrics[key] = mv
	}
	rec.Periods = []Period{period}
	return rec
}

func recordFromHistoryResponse(q Query, raw *historyResponse) *Record {
	rec := &Record{
		Origin:     q.Origin,
		URL:        q.URL,
		FormFactor: formFactorLabel(raw.Record.Key.FormFactor),
	}
	if rec.FormFactor == "all" && q.FormFactor != "" {
		rec.FormFactor = formFactorLabel(q.FormFactor)
	}

	n := len(raw.Record.CollectionPeriods)
	periods := make([]Period, n)
	for i, cp := range raw.Record.CollectionPeriods {
		periods[i] = Period{
			FirstDate: cp.FirstDate.String(),
			LastDate:  cp.LastDate.String(),
			Metrics:   make(map[string]MetricValue),
		}
	}

	for apiName, m := range raw.Record.Metrics {
		key := metricKeyForAPIName(apiName)
		if key == "" {
			continue
		}
		// Densities per bucket are parallel arrays aligned with collectionPeriods.
		var good, ni, poor []json.RawMessage
		for bi, bucket := range m.HistogramTimeseries {
			switch bi {
			case 0:
				good = bucket.Densities
			case 1:
				ni = bucket.Densities
			case 2:
				poor = bucket.Densities
			}
		}
		p75s := m.PercentilesTimeseries.P75s

		for i := 0; i < n; i++ {
			var mv MetricValue
			if i < len(good) {
				if v, ok := parseNum(good[i]); ok {
					mv.Good = v
				}
			}
			if i < len(ni) {
				if v, ok := parseNum(ni[i]); ok {
					mv.NI = v
				}
			}
			if i < len(poor) {
				if v, ok := parseNum(poor[i]); ok {
					mv.Poor = v
				}
			}
			if i < len(p75s) {
				if v, ok := parseNum(p75s[i]); ok {
					mv.P75 = v
					mv.HasP75 = true
				}
			}
			periods[i].Metrics[key] = mv
		}
	}
	rec.Periods = periods
	return rec
}

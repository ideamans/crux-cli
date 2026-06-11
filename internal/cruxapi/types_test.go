package cruxapi

import (
	"encoding/json"
	"testing"
)

func TestParseNum(t *testing.T) {
	tests := []struct {
		raw  string
		want float64
		ok   bool
	}{
		{`1362`, 1362, true},
		{`"0.05"`, 0.05, true},
		{`0.919`, 0.919, true},
		{`null`, 0, false},
		{`"NaN"`, 0, false},
		{`""`, 0, false},
		{``, 0, false},
	}
	for _, tt := range tests {
		got, ok := parseNum(json.RawMessage(tt.raw))
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("parseNum(%s) = %v, %v; want %v, %v", tt.raw, got, ok, tt.want, tt.ok)
		}
	}
}

func TestAPINames(t *testing.T) {
	got := APINames([]string{"lcp", "cls", "unknown", "ttfb"})
	want := []string{"largest_contentful_paint", "cumulative_layout_shift", "experimental_time_to_first_byte"}
	if len(got) != len(want) {
		t.Fatalf("APINames len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("APINames[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

const sampleHistory = `{
  "record": {
    "key": { "formFactor": "PHONE", "origin": "https://example.com" },
    "metrics": {
      "largest_contentful_paint": {
        "histogramTimeseries": [
          { "start": 0, "end": 2500, "densities": [0.90, "NaN", 0.88] },
          { "start": 2500, "end": 4000, "densities": [0.07, "NaN", 0.08] },
          { "start": 4000, "densities": [0.03, "NaN", 0.04] }
        ],
        "percentilesTimeseries": { "p75s": [1362, null, 1400] }
      },
      "cumulative_layout_shift": {
        "histogramTimeseries": [
          { "start": "0.00", "end": "0.10", "densities": [0.95, 0.96, 0.97] },
          { "start": "0.10", "end": "0.25", "densities": [0.03, 0.02, 0.02] },
          { "start": "0.25", "densities": [0.02, 0.02, 0.01] }
        ],
        "percentilesTimeseries": { "p75s": ["0.05", "0.04", "0.03"] }
      }
    },
    "collectionPeriods": [
      { "firstDate": {"year":2025,"month":1,"day":1}, "lastDate": {"year":2025,"month":1,"day":28} },
      { "firstDate": {"year":2025,"month":1,"day":8}, "lastDate": {"year":2025,"month":2,"day":4} },
      { "firstDate": {"year":2025,"month":1,"day":15}, "lastDate": {"year":2025,"month":2,"day":11} }
    ]
  }
}`

func TestRecordFromHistoryResponse(t *testing.T) {
	var raw historyResponse
	if err := json.Unmarshal([]byte(sampleHistory), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	rec := recordFromHistoryResponse(Query{Origin: "https://example.com", FormFactor: "PHONE"}, &raw)

	if rec.FormFactor != "phone" {
		t.Errorf("FormFactor = %q, want phone", rec.FormFactor)
	}
	if rec.Origin != "https://example.com" {
		t.Errorf("Origin = %q", rec.Origin)
	}
	if len(rec.Periods) != 3 {
		t.Fatalf("Periods len = %d, want 3", len(rec.Periods))
	}

	// Period 0: full data.
	p0 := rec.Periods[0]
	if p0.LastDate != "2025-01-28" {
		t.Errorf("p0.LastDate = %q", p0.LastDate)
	}
	lcp0 := p0.Metrics["lcp"]
	if lcp0.Good != 0.90 || !lcp0.HasP75 || lcp0.P75 != 1362 {
		t.Errorf("p0 lcp = %+v", lcp0)
	}

	// Period 1: missing data (NaN densities, null p75).
	lcp1 := rec.Periods[1].Metrics["lcp"]
	if lcp1.Good != 0 || lcp1.HasP75 {
		t.Errorf("p1 lcp should be empty, got %+v", lcp1)
	}

	// CLS p75 parsed from string.
	cls0 := p0.Metrics["cls"]
	if !cls0.HasP75 || cls0.P75 != 0.05 || cls0.Good != 0.95 {
		t.Errorf("p0 cls = %+v", cls0)
	}
}

func TestRecordFromResponse(t *testing.T) {
	const sample = `{
      "record": {
        "key": { "formFactor": "DESKTOP", "url": "https://example.com/page" },
        "metrics": {
          "largest_contentful_paint": {
            "histogram": [
              {"start":0,"end":2500,"density":0.85},
              {"start":2500,"end":4000,"density":0.10},
              {"start":4000,"density":0.05}
            ],
            "percentiles": {"p75": 2100}
          }
        },
        "collectionPeriod": {
          "firstDate": {"year":2025,"month":5,"day":1},
          "lastDate": {"year":2025,"month":5,"day":28}
        }
      }
    }`
	var raw recordResponse
	if err := json.Unmarshal([]byte(sample), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	rec := recordFromResponse(Query{URL: "https://example.com/page"}, &raw)
	if rec.Target() != "https://example.com/page" {
		t.Errorf("Target = %q", rec.Target())
	}
	if rec.FormFactor != "desktop" {
		t.Errorf("FormFactor = %q", rec.FormFactor)
	}
	if len(rec.Periods) != 1 {
		t.Fatalf("Periods len = %d", len(rec.Periods))
	}
	lcp := rec.Periods[0].Metrics["lcp"]
	if lcp.Good != 0.85 || lcp.Poor != 0.05 || lcp.P75 != 2100 {
		t.Errorf("lcp = %+v", lcp)
	}
}

package crux

// DeviceRow is a single row from chrome-ux-report.materialized.device_summary.
// CLS uses small/medium/large naming; all other metrics use fast/avg/slow.
type DeviceRow struct {
	Origin    string  `json:"origin"     bigquery:"origin"`
	Yyyymm    string  `json:"yyyymm"     bigquery:"yyyymm"`
	Device    string  `json:"device"     bigquery:"device"`
	FastLcp   float64 `json:"fast_lcp"   bigquery:"fast_lcp"`
	AvgLcp    float64 `json:"avg_lcp"    bigquery:"avg_lcp"`
	SlowLcp   float64 `json:"slow_lcp"   bigquery:"slow_lcp"`
	P75Lcp    int64   `json:"p75_lcp"    bigquery:"p75_lcp"`   // ms, INTEGER in BigQuery
	SmallCls  float64 `json:"small_cls"  bigquery:"small_cls"`
	MediumCls float64 `json:"medium_cls" bigquery:"medium_cls"`
	LargeCls  float64 `json:"large_cls"  bigquery:"large_cls"`
	P75Cls    float64 `json:"p75_cls"    bigquery:"p75_cls"`   // ratio 0-1, FLOAT
	FastInp   float64 `json:"fast_inp"   bigquery:"fast_inp"`
	AvgInp    float64 `json:"avg_inp"    bigquery:"avg_inp"`
	SlowInp   float64 `json:"slow_inp"   bigquery:"slow_inp"`
	P75Inp    int64   `json:"p75_inp"    bigquery:"p75_inp"`   // ms, INTEGER in BigQuery
	FastFcp   float64 `json:"fast_fcp"   bigquery:"fast_fcp"`
	AvgFcp    float64 `json:"avg_fcp"    bigquery:"avg_fcp"`
	SlowFcp   float64 `json:"slow_fcp"   bigquery:"slow_fcp"`
	P75Fcp    int64   `json:"p75_fcp"    bigquery:"p75_fcp"`   // ms, INTEGER in BigQuery
	FastTtfb  float64 `json:"fast_ttfb"  bigquery:"fast_ttfb"`
	AvgTtfb   float64 `json:"avg_ttfb"   bigquery:"avg_ttfb"`
	SlowTtfb  float64 `json:"slow_ttfb"  bigquery:"slow_ttfb"`
	P75Ttfb   int64   `json:"p75_ttfb"   bigquery:"p75_ttfb"` // ms, INTEGER in BigQuery
	FastFid   float64 `json:"fast_fid"   bigquery:"fast_fid"`
	AvgFid    float64 `json:"avg_fid"    bigquery:"avg_fid"`
	SlowFid   float64 `json:"slow_fid"   bigquery:"slow_fid"`
	P75Fid    int64   `json:"p75_fid"    bigquery:"p75_fid"`   // ms, INTEGER in BigQuery
	FastOl    float64 `json:"fast_ol"    bigquery:"fast_ol"`
	AvgOl     float64 `json:"avg_ol"     bigquery:"avg_ol"`
	SlowOl    float64 `json:"slow_ol"    bigquery:"slow_ol"`
	P75Ol     int64   `json:"p75_ol"     bigquery:"p75_ol"`    // ms, INTEGER in BigQuery
	LowRtt    float64 `json:"low_rtt"    bigquery:"low_rtt"`   // RTT uses low/medium/high
	MediumRtt float64 `json:"medium_rtt" bigquery:"medium_rtt"`
	HighRtt   float64 `json:"high_rtt"   bigquery:"high_rtt"`
	P75Rtt    int64   `json:"p75_rtt"    bigquery:"p75_rtt"`   // ms, INTEGER in BigQuery
}

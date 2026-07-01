// Package cruxapi is a client for the public Chrome UX Report (CrUX) REST API.
//
// Unlike the BigQuery dataset (monthly origin-level aggregates), the CrUX API
// returns the 28-day rolling distribution for a specific origin OR url:
//
//   - queryRecord        — the latest single record (one collection period)
//   - queryHistoryRecord — a weekly time series (up to 40 collection periods)
//
// See https://developer.chrome.com/docs/crux/api
package cruxapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	recordEndpoint  = "https://chromeuxreport.googleapis.com/v1/records:queryRecord"
	historyEndpoint = "https://chromeuxreport.googleapis.com/v1/records:queryHistoryRecord"
)

// Client calls the CrUX API with a given API key.
type Client struct {
	apiKey string
	http   *http.Client
}

// New returns a Client using the given API key.
func New(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

// Query identifies a single record lookup. Exactly one of Origin/URL must be set.
type Query struct {
	Origin                  string
	URL                     string
	FormFactor              string   // "PHONE"/"DESKTOP"/"TABLET"; empty = aggregate across all
	EffectiveConnectionType string   // "4G"/"3G"/"2G"/"slow-2G"/"offline"; empty = aggregate across all
	Metrics                 []string // API metric names; empty = all available
	CollectionPeriodCount   int      // history only; 1-40, 0 = API default (25)
}

// apiRequest is the JSON request body.
type apiRequest struct {
	Origin                  string   `json:"origin,omitempty"`
	URL                     string   `json:"url,omitempty"`
	FormFactor              string   `json:"formFactor,omitempty"`
	EffectiveConnectionType string   `json:"effectiveConnectionType,omitempty"`
	Metrics                 []string `json:"metrics,omitempty"`
	CollectionPeriodCount   int      `json:"collectionPeriodCount,omitempty"`
}

func (q Query) request() apiRequest {
	return apiRequest{
		Origin:                  q.Origin,
		URL:                     q.URL,
		FormFactor:              q.FormFactor,
		EffectiveConnectionType: q.EffectiveConnectionType,
		Metrics:                 q.Metrics,
		CollectionPeriodCount:   q.CollectionPeriodCount,
	}
}

// Label returns the human-readable target (url takes precedence over origin).
func (q Query) Label() string {
	if q.URL != "" {
		return q.URL
	}
	return q.Origin
}

// ── Raw API response types ──────────────────────────────────────────────────

type apiDate struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

func (d apiDate) String() string {
	if d.Year == 0 {
		return ""
	}
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

type apiKey struct {
	FormFactor string `json:"formFactor"`
	Origin     string `json:"origin"`
	URL        string `json:"url"`
}

// queryRecord response.
type recordResponse struct {
	Record struct {
		Key     apiKey `json:"key"`
		Metrics map[string]struct {
			Histogram []struct {
				Start   json.RawMessage `json:"start"`
				End     json.RawMessage `json:"end"`
				Density float64         `json:"density"`
			} `json:"histogram"`
			Percentiles struct {
				P75 json.RawMessage `json:"p75"`
			} `json:"percentiles"`
		} `json:"metrics"`
		CollectionPeriod struct {
			FirstDate apiDate `json:"firstDate"`
			LastDate  apiDate `json:"lastDate"`
		} `json:"collectionPeriod"`
	} `json:"record"`
}

// queryHistoryRecord response.
type historyResponse struct {
	Record struct {
		Key     apiKey `json:"key"`
		Metrics map[string]struct {
			HistogramTimeseries []struct {
				Start     json.RawMessage   `json:"start"`
				End       json.RawMessage   `json:"end"`
				Densities []json.RawMessage `json:"densities"`
			} `json:"histogramTimeseries"`
			PercentilesTimeseries struct {
				P75s []json.RawMessage `json:"p75s"`
			} `json:"percentilesTimeseries"`
		} `json:"metrics"`
		CollectionPeriods []struct {
			FirstDate apiDate `json:"firstDate"`
			LastDate  apiDate `json:"lastDate"`
		} `json:"collectionPeriods"`
	} `json:"record"`
}

type apiError struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// ErrNotFound is returned when the CrUX API has no data for the target
// (HTTP 404 / NOT_FOUND). Callers typically treat this as "no data" rather
// than a hard failure.
var ErrNotFound = fmt.Errorf("crux api: no data for target")

// post sends body to endpoint and returns the raw response bytes.
// A 404 is mapped to ErrNotFound; other non-2xx statuses become errors.
func (c *Client) post(ctx context.Context, endpoint string, body apiRequest) ([]byte, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("crux api: no API key (set CRUX_API_KEY or run `crux auth set-api-key`)")
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := endpoint + "?key=" + c.apiKey
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("crux api: request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("crux api: read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var e apiError
		if json.Unmarshal(data, &e) == nil && e.Error.Message != "" {
			return nil, fmt.Errorf("crux api: %d %s: %s", e.Error.Code, e.Error.Status, e.Error.Message)
		}
		return nil, fmt.Errorf("crux api: HTTP %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

// QueryRecord fetches the latest single record. Returns ErrNotFound if the
// CrUX API has no data for the target.
func (c *Client) QueryRecord(ctx context.Context, q Query) (*Record, error) {
	data, err := c.post(ctx, recordEndpoint, q.request())
	if err != nil {
		return nil, err
	}
	var raw recordResponse
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("crux api: decode record: %w", err)
	}
	return recordFromResponse(q, &raw), nil
}

// QueryHistoryRecord fetches the weekly time series. Returns ErrNotFound if the
// CrUX API has no data for the target.
func (c *Client) QueryHistoryRecord(ctx context.Context, q Query) (*Record, error) {
	data, err := c.post(ctx, historyEndpoint, q.request())
	if err != nil {
		return nil, err
	}
	var raw historyResponse
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("crux api: decode history: %w", err)
	}
	return recordFromHistoryResponse(q, &raw), nil
}

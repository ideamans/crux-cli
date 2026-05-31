package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/ideamans/crux-cli/internal/cache"
	"github.com/ideamans/crux-cli/internal/crux"
)

// ── Mock BQ client ─────────────────────────────────────────────────────────────

type mockBQClient struct {
	latestMonth    string
	latestMonthErr error
	rows           []crux.DeviceRow
	rowsErr        error
	queriedOrigins []string
	queryCalled    bool
}

func (m *mockBQClient) QueryLatestMonth(ctx context.Context) (string, error) {
	return m.latestMonth, m.latestMonthErr
}

func (m *mockBQClient) QueryDevice(ctx context.Context, origins []string, monthFrom, monthTo, device string) ([]crux.DeviceRow, error) {
	m.queryCalled = true
	m.queriedOrigins = append(m.queriedOrigins, origins...)
	if m.rowsErr != nil {
		return nil, m.rowsErr
	}
	// filter rows by requested origins
	reqSet := make(map[string]bool, len(origins))
	for _, o := range origins {
		reqSet[o] = true
	}
	var result []crux.DeviceRow
	for _, r := range m.rows {
		if reqSet[r.Origin] {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockBQClient) Close() {}

// ── Mock cache store ───────────────────────────────────────────────────────────

// mockCacheStore uses a map where:
//   - key absent    → cache miss (GetDevice returns nil)
//   - key present   → cache hit (returns the stored slice, may be empty)
type mockCacheStore struct {
	latestMonth *cache.LatestMonth
	stale       bool
	// data holds cached device rows; use present key to indicate "cached"
	data  map[string][]crux.DeviceRow
	saved map[string][]crux.DeviceRow
}

func newMockCacheStore() *mockCacheStore {
	return &mockCacheStore{
		data:  make(map[string][]crux.DeviceRow),
		saved: make(map[string][]crux.DeviceRow),
	}
}

func cacheKey(origin, monthFrom, monthTo string) string {
	return origin + "|" + monthFrom + "|" + monthTo
}

func (m *mockCacheStore) GetLatestMonth() (*cache.LatestMonth, error) {
	return m.latestMonth, nil
}

func (m *mockCacheStore) SaveLatestMonth(lm *cache.LatestMonth) error {
	m.latestMonth = lm
	return nil
}

func (m *mockCacheStore) IsStale(lm *cache.LatestMonth) bool {
	return m.stale
}

func (m *mockCacheStore) GetDevice(origin, monthFrom, monthTo string) ([]crux.DeviceRow, error) {
	key := cacheKey(origin, monthFrom, monthTo)
	rows, ok := m.data[key]
	if !ok {
		return nil, nil
	}
	return rows, nil
}

func (m *mockCacheStore) SaveDevice(origin, monthFrom, monthTo string, rows []crux.DeviceRow) error {
	key := cacheKey(origin, monthFrom, monthTo)
	if rows == nil {
		rows = []crux.DeviceRow{}
	}
	m.saved[key] = rows
	m.data[key] = rows
	return nil
}

// ── Helpers ────────────────────────────────────────────────────────────────────

func freshLatestMonth(yyyymm string) *cache.LatestMonth {
	return &cache.LatestMonth{Yyyymm: yyyymm, CheckedAt: time.Now().UTC()}
}

// parseJSONRows parses JSON output from RunDevice into a slice of DeviceRow.
func parseJSONRows(t *testing.T, buf *bytes.Buffer) []crux.DeviceRow {
	t.Helper()
	// The output buffer may have a leading newline before the JSON array.
	// Trim whitespace and then decode.
	trimmed := strings.TrimSpace(buf.String())
	if trimmed == "" {
		return nil
	}
	var rows []crux.DeviceRow
	if err := json.Unmarshal([]byte(trimmed), &rows); err != nil {
		t.Fatalf("failed to parse JSON output %q: %v", trimmed, err)
	}
	return rows
}

// ── Tests ──────────────────────────────────────────────────────────────────────

func TestRunDevice_AllCacheHits(t *testing.T) {
	origin1 := "https://a.example.com"
	origin2 := "https://b.example.com"
	monthFrom := "202401"
	monthTo := "202404"

	rows1 := []crux.DeviceRow{{Origin: origin1, Yyyymm: "202401", Device: "phone", FastLcp: 0.7}}
	rows2 := []crux.DeviceRow{{Origin: origin2, Yyyymm: "202401", Device: "phone", FastLcp: 0.8}}

	ca := newMockCacheStore()
	ca.latestMonth = freshLatestMonth(monthTo)
	ca.stale = false
	ca.data[cacheKey(origin1, monthFrom, monthTo)] = rows1
	ca.data[cacheKey(origin2, monthFrom, monthTo)] = rows2

	bqCalled := false
	bqFactory := func(ctx context.Context) (BQClient, error) {
		bqCalled = true
		return &mockBQClient{}, nil
	}

	var out bytes.Buffer
	opts := DeviceOptions{
		Origins:   []string{origin1, origin2},
		Device:    "all",
		MonthFrom: monthFrom,
		MonthTo:   monthTo,
		Format:    "json",
		Progress:  io.Discard,
	}
	if err := RunDevice(context.Background(), bqFactory, ca, opts, &out); err != nil {
		t.Fatalf("RunDevice: %v", err)
	}

	if bqCalled {
		t.Error("BQ factory should not have been called when all origins are cached")
	}

	rows := parseJSONRows(t, &out)
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
}

func TestRunDevice_LatestMonthStale(t *testing.T) {
	origin := "https://example.com"
	monthFrom := "202401"
	monthTo := "202504"

	bqMock := &mockBQClient{
		latestMonth: monthTo,
		rows:        []crux.DeviceRow{{Origin: origin, Yyyymm: "202504", Device: "phone", FastLcp: 0.85}},
	}

	ca := newMockCacheStore()
	// stale latest month
	ca.latestMonth = &cache.LatestMonth{Yyyymm: "202503", CheckedAt: time.Now().UTC().Add(-25 * time.Hour)}
	ca.stale = true
	ca.data[cacheKey(origin, monthFrom, monthTo)] = bqMock.rows

	bqFactoryCalled := false
	bqFactory := func(ctx context.Context) (BQClient, error) {
		bqFactoryCalled = true
		return bqMock, nil
	}

	var out bytes.Buffer
	opts := DeviceOptions{
		Origins:   []string{origin},
		Device:    "all",
		MonthFrom: monthFrom,
		Format:    "json",
		Progress:  io.Discard,
	}
	if err := RunDevice(context.Background(), bqFactory, ca, opts, &out); err != nil {
		t.Fatalf("RunDevice: %v", err)
	}

	if !bqFactoryCalled {
		t.Error("BQ factory should have been called for stale latest month")
	}

	// Verify latest month was updated in cache
	if ca.latestMonth == nil || ca.latestMonth.Yyyymm != monthTo {
		t.Errorf("expected latest month saved as %q, got %+v", monthTo, ca.latestMonth)
	}
}

func TestRunDevice_CacheMiss(t *testing.T) {
	origin := "https://example.com"
	monthFrom := "202401"
	monthTo := "202404"

	bqMock := &mockBQClient{
		rows: []crux.DeviceRow{
			{Origin: origin, Yyyymm: "202401", Device: "phone", FastLcp: 0.75},
			{Origin: origin, Yyyymm: "202402", Device: "phone", FastLcp: 0.76},
		},
	}

	ca := newMockCacheStore()
	ca.latestMonth = freshLatestMonth(monthTo)
	ca.stale = false
	// no entry for origin → cache miss

	bqFactory := func(ctx context.Context) (BQClient, error) {
		return bqMock, nil
	}

	var out bytes.Buffer
	opts := DeviceOptions{
		Origins:   []string{origin},
		Device:    "all",
		MonthFrom: monthFrom,
		MonthTo:   monthTo,
		Format:    "json",
		Progress:  io.Discard,
	}
	if err := RunDevice(context.Background(), bqFactory, ca, opts, &out); err != nil {
		t.Fatalf("RunDevice: %v", err)
	}

	if !bqMock.queryCalled {
		t.Error("BQ QueryDevice should have been called for cache miss")
	}

	rows := parseJSONRows(t, &out)
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}

	// Verify rows were saved to cache
	savedKey := cacheKey(origin, monthFrom, monthTo)
	if _, ok := ca.saved[savedKey]; !ok {
		t.Error("expected device rows to be saved to cache")
	}
}

func TestRunDevice_PartialCacheHit(t *testing.T) {
	origin1 := "https://cached.example.com"
	origin2 := "https://missing.example.com"
	monthFrom := "202401"
	monthTo := "202404"

	cachedRows := []crux.DeviceRow{{Origin: origin1, Yyyymm: "202401", Device: "phone", FastLcp: 0.9}}
	fetchedRows := []crux.DeviceRow{{Origin: origin2, Yyyymm: "202401", Device: "phone", FastLcp: 0.6}}

	bqMock := &mockBQClient{
		rows: fetchedRows,
	}

	ca := newMockCacheStore()
	ca.latestMonth = freshLatestMonth(monthTo)
	ca.stale = false
	ca.data[cacheKey(origin1, monthFrom, monthTo)] = cachedRows
	// origin2 not in cache

	bqFactory := func(ctx context.Context) (BQClient, error) {
		return bqMock, nil
	}

	var out bytes.Buffer
	opts := DeviceOptions{
		Origins:   []string{origin1, origin2},
		Device:    "all",
		MonthFrom: monthFrom,
		MonthTo:   monthTo,
		Format:    "json",
		Progress:  io.Discard,
	}
	if err := RunDevice(context.Background(), bqFactory, ca, opts, &out); err != nil {
		t.Fatalf("RunDevice: %v", err)
	}

	// BQ should only have been queried with origin2
	if len(bqMock.queriedOrigins) != 1 || bqMock.queriedOrigins[0] != origin2 {
		t.Errorf("expected BQ queried with [%q], got %v", origin2, bqMock.queriedOrigins)
	}

	rows := parseJSONRows(t, &out)
	if len(rows) != 2 {
		t.Errorf("expected 2 total rows (1 cached + 1 fetched), got %d", len(rows))
	}
}

func TestRunDevice_NoCache(t *testing.T) {
	origin := "https://example.com"
	monthFrom := "202401"
	monthTo := "202404"

	bqMock := &mockBQClient{
		latestMonth: monthTo,
		rows:        []crux.DeviceRow{{Origin: origin, Yyyymm: "202401", Device: "phone", FastLcp: 0.8}},
	}

	ca := newMockCacheStore()
	// Even if data is cached, NoCache should bypass it
	ca.data[cacheKey(origin, monthFrom, monthTo)] = bqMock.rows

	bqFactory := func(ctx context.Context) (BQClient, error) {
		return bqMock, nil
	}

	var out bytes.Buffer
	opts := DeviceOptions{
		Origins:   []string{origin},
		Device:    "all",
		MonthFrom: monthFrom,
		MonthTo:   monthTo,
		Format:    "json",
		NoCache:   true,
		Progress:  io.Discard,
	}
	if err := RunDevice(context.Background(), bqFactory, ca, opts, &out); err != nil {
		t.Fatalf("RunDevice: %v", err)
	}

	if !bqMock.queryCalled {
		t.Error("BQ QueryDevice should always be called when NoCache=true")
	}

	rows := parseJSONRows(t, &out)
	if len(rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(rows))
	}
}

func TestRunDevice_NoData(t *testing.T) {
	origin := "https://unknown.example.com"
	monthFrom := "202401"
	monthTo := "202404"

	bqMock := &mockBQClient{
		latestMonth: monthTo,
		rows:        nil, // empty — origin not in CrUX
	}

	ca := newMockCacheStore()
	ca.latestMonth = freshLatestMonth(monthTo)
	ca.stale = false

	bqFactory := func(ctx context.Context) (BQClient, error) {
		return bqMock, nil
	}

	var out bytes.Buffer
	opts := DeviceOptions{
		Origins:   []string{origin},
		Device:    "all",
		MonthFrom: monthFrom,
		MonthTo:   monthTo,
		Format:    "json",
		Progress:  io.Discard,
	}
	if err := RunDevice(context.Background(), bqFactory, ca, opts, &out); err != nil {
		t.Fatalf("RunDevice: %v", err)
	}

	outStr := out.String()
	if !strings.Contains(outStr, "No data found") {
		t.Errorf("expected 'No data found' in output, got: %q", outStr)
	}
}

func TestRunDevice_MonthFromResolved(t *testing.T) {
	origin := "https://example.com"
	monthTo := "202504"
	// Months=3 → monthFrom = 202504 - 2 = 202502
	expectedMonthFrom := "202502"

	bqMock := &mockBQClient{
		rows: []crux.DeviceRow{
			{Origin: origin, Yyyymm: "202502", Device: "phone", FastLcp: 0.7},
		},
	}

	ca := newMockCacheStore()
	ca.latestMonth = freshLatestMonth(monthTo)
	ca.stale = false
	// No cache for origin, so BQ will be called

	bqFactory := func(ctx context.Context) (BQClient, error) {
		return bqMock, nil
	}

	var out bytes.Buffer
	opts := DeviceOptions{
		Origins:  []string{origin},
		Device:   "all",
		MonthTo:  monthTo,
		Months:   3,
		Format:   "json",
		Progress: io.Discard,
	}
	if err := RunDevice(context.Background(), bqFactory, ca, opts, &out); err != nil {
		t.Fatalf("RunDevice: %v", err)
	}

	// Verify cache was saved with the correct resolved monthFrom
	savedKey := cacheKey(origin, expectedMonthFrom, monthTo)
	if _, ok := ca.saved[savedKey]; !ok {
		t.Errorf("expected device rows to be saved with monthFrom=%q, saved keys: %v",
			expectedMonthFrom, keysOf(ca.saved))
	}
}

func keysOf(m map[string][]crux.DeviceRow) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

package cache

import (
	"testing"
	"time"

	"github.com/ideamans/crux-cli/internal/crux"
)

func TestGetLatestMonth_Absent(t *testing.T) {
	ca := New(t.TempDir())
	lm, err := ca.GetLatestMonth()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lm != nil {
		t.Errorf("expected nil, got %+v", lm)
	}
}

func TestSaveAndGetLatestMonth(t *testing.T) {
	ca := New(t.TempDir())
	want := &LatestMonth{
		Yyyymm:    "202504",
		CheckedAt: time.Now().UTC().Truncate(time.Second),
	}
	if err := ca.SaveLatestMonth(want); err != nil {
		t.Fatalf("SaveLatestMonth: %v", err)
	}
	got, err := ca.GetLatestMonth()
	if err != nil {
		t.Fatalf("GetLatestMonth: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil LatestMonth")
	}
	if got.Yyyymm != want.Yyyymm {
		t.Errorf("Yyyymm: got %q, want %q", got.Yyyymm, want.Yyyymm)
	}
	// JSON marshaling truncates to second precision; compare with same truncation
	if !got.CheckedAt.Equal(want.CheckedAt) {
		t.Errorf("CheckedAt: got %v, want %v", got.CheckedAt, want.CheckedAt)
	}
}

func TestIsStale(t *testing.T) {
	ca := New(t.TempDir())

	t.Run("nil is stale", func(t *testing.T) {
		if !ca.IsStale(nil) {
			t.Error("expected nil to be stale")
		}
	})

	t.Run("fresh is not stale", func(t *testing.T) {
		lm := &LatestMonth{Yyyymm: "202504", CheckedAt: time.Now().UTC()}
		if ca.IsStale(lm) {
			t.Error("expected fresh LatestMonth to not be stale")
		}
	})

	t.Run("25h ago is stale", func(t *testing.T) {
		lm := &LatestMonth{Yyyymm: "202504", CheckedAt: time.Now().UTC().Add(-25 * time.Hour)}
		if !ca.IsStale(lm) {
			t.Error("expected 25h old LatestMonth to be stale")
		}
	})
}

func TestGetDevice_Absent(t *testing.T) {
	ca := New(t.TempDir())
	rows, err := ca.GetDevice("https://example.com", "202401", "202412")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows != nil {
		t.Errorf("expected nil, got %v", rows)
	}
}

func TestSaveAndGetDevice(t *testing.T) {
	ca := New(t.TempDir())
	origin := "https://example.com"
	monthFrom := "202401"
	monthTo := "202412"

	want := []crux.DeviceRow{
		{Origin: origin, Yyyymm: "202401", Device: "phone", FastLcp: 0.7, P75Lcp: 1200},
		{Origin: origin, Yyyymm: "202412", Device: "desktop", FastLcp: 0.9, P75Lcp: 800},
	}

	if err := ca.SaveDevice(origin, monthFrom, monthTo, want); err != nil {
		t.Fatalf("SaveDevice: %v", err)
	}

	got, err := ca.GetDevice(origin, monthFrom, monthTo)
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d rows, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Origin != want[i].Origin || got[i].Yyyymm != want[i].Yyyymm ||
			got[i].Device != want[i].Device || got[i].FastLcp != want[i].FastLcp ||
			got[i].P75Lcp != want[i].P75Lcp {
			t.Errorf("row %d mismatch: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestSaveAndGetDevice_EmptyRows(t *testing.T) {
	ca := New(t.TempDir())
	origin := "https://nodata.example.com"
	monthFrom := "202401"
	monthTo := "202412"

	// Save empty slice (origin not in CrUX)
	if err := ca.SaveDevice(origin, monthFrom, monthTo, []crux.DeviceRow{}); err != nil {
		t.Fatalf("SaveDevice: %v", err)
	}

	got, err := ca.GetDevice(origin, monthFrom, monthTo)
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	// After saving empty slice, we should get a non-nil empty slice back (not nil — it was cached)
	if got == nil {
		t.Error("expected non-nil (cached empty) result")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 rows, got %d", len(got))
	}
}

func TestClearOrigin(t *testing.T) {
	ca := New(t.TempDir())
	origin1 := "https://example.com"
	origin2 := "https://other.com"
	monthFrom := "202401"
	monthTo := "202412"

	rows1 := []crux.DeviceRow{{Origin: origin1, Yyyymm: "202401", Device: "phone"}}
	rows2 := []crux.DeviceRow{{Origin: origin2, Yyyymm: "202401", Device: "phone"}}

	if err := ca.SaveDevice(origin1, monthFrom, monthTo, rows1); err != nil {
		t.Fatalf("SaveDevice origin1: %v", err)
	}
	if err := ca.SaveDevice(origin2, monthFrom, monthTo, rows2); err != nil {
		t.Fatalf("SaveDevice origin2: %v", err)
	}

	// Clear only origin1
	if err := ca.ClearOrigin(origin1); err != nil {
		t.Fatalf("ClearOrigin: %v", err)
	}

	// origin1 should be gone
	got1, err := ca.GetDevice(origin1, monthFrom, monthTo)
	if err != nil {
		t.Fatalf("GetDevice origin1: %v", err)
	}
	if got1 != nil {
		t.Error("expected nil for cleared origin1, got data")
	}

	// origin2 should still be present
	got2, err := ca.GetDevice(origin2, monthFrom, monthTo)
	if err != nil {
		t.Fatalf("GetDevice origin2: %v", err)
	}
	if got2 == nil {
		t.Error("expected non-nil for origin2, got nil")
	}
	if len(got2) != 1 {
		t.Errorf("expected 1 row for origin2, got %d", len(got2))
	}
}

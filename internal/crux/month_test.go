package crux

import (
	"testing"
)

func TestSubtractMonths(t *testing.T) {
	tests := []struct {
		name    string
		yyyymm  string
		n       int
		want    string
		wantErr bool
	}{
		{
			name:   "12 months back inclusive",
			yyyymm: "202504",
			n:      11,
			want:   "202405",
		},
		{
			name:   "crosses year boundary",
			yyyymm: "202502",
			n:      3,
			want:   "202411",
		},
		{
			name:   "zero subtraction",
			yyyymm: "202504",
			n:      0,
			want:   "202504",
		},
		{
			name:   "subtract to January",
			yyyymm: "202501",
			n:      0,
			want:   "202501",
		},
		{
			name:   "subtract across multiple years",
			yyyymm: "202312",
			n:      24,
			want:   "202112",
		},
		{
			name:   "subtract from 200001 gives 199912",
			yyyymm: "200001",
			n:      1,
			want:   "199912",
		},
		{
			name:    "underflow: year 0 month 1 minus 1",
			yyyymm:  "000001",
			n:       1,
			wantErr: true,
		},
		{
			name:    "invalid format",
			yyyymm:  "2025",
			n:       1,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SubtractMonths(tt.yyyymm, tt.n)
			if tt.wantErr {
				if err == nil {
					t.Errorf("SubtractMonths(%q, %d) expected error, got nil", tt.yyyymm, tt.n)
				}
				return
			}
			if err != nil {
				t.Fatalf("SubtractMonths(%q, %d) unexpected error: %v", tt.yyyymm, tt.n, err)
			}
			if got != tt.want {
				t.Errorf("SubtractMonths(%q, %d) = %q, want %q", tt.yyyymm, tt.n, got, tt.want)
			}
		})
	}
}

func TestFormatMonth(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"202504", "2025-04"},
		{"202501", "2025-01"},
		{"200012", "2000-12"},
		{"", ""},
		{"2025", "2025"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := FormatMonth(tt.input)
			if got != tt.want {
				t.Errorf("FormatMonth(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateYYYYMM(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"202504", false},
		{"202501", false},
		{"200001", false},
		{"202513", true},  // month 13 invalid
		{"202500", true},  // month 0 invalid
		{"2025",   true},  // too short
		{"20250401", true}, // too long
		{"abcdef", true},  // non-numeric
		{"20251a", true},  // partially non-numeric
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateYYYYMM(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateYYYYMM(%q) expected error, got nil", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateYYYYMM(%q) unexpected error: %v", tt.input, err)
			}
		})
	}
}

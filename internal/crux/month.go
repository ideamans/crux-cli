package crux

import (
	"fmt"
	"strconv"
)

// SubtractMonths subtracts n months from a YYYYMM string and returns the result.
func SubtractMonths(yyyymm string, n int) (string, error) {
	if len(yyyymm) != 6 {
		return "", fmt.Errorf("invalid yyyymm format: %s", yyyymm)
	}
	year, err := strconv.Atoi(yyyymm[:4])
	if err != nil {
		return "", fmt.Errorf("invalid yyyymm: %s", yyyymm)
	}
	month, err := strconv.Atoi(yyyymm[4:])
	if err != nil {
		return "", fmt.Errorf("invalid yyyymm: %s", yyyymm)
	}
	total := year*12 + (month - 1) - n
	if total < 0 {
		return "", fmt.Errorf("month underflow from %s minus %d", yyyymm, n)
	}
	return fmt.Sprintf("%04d%02d", total/12, total%12+1), nil
}

// YYYYMMToInt converts a YYYYMM string to int64 for BigQuery comparison.
func YYYYMMToInt(yyyymm string) (int64, error) {
	v, err := strconv.ParseInt(yyyymm, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid yyyymm: %s", yyyymm)
	}
	return v, nil
}

// FormatMonth formats YYYYMM as YYYY-MM for display.
func FormatMonth(yyyymm string) string {
	if len(yyyymm) != 6 {
		return yyyymm
	}
	return yyyymm[:4] + "-" + yyyymm[4:]
}

// ValidateYYYYMM returns an error if the string is not a valid YYYYMM.
func ValidateYYYYMM(s string) error {
	if len(s) != 6 {
		return fmt.Errorf("must be 6 digits (YYYYMM), got %q", s)
	}
	if _, err := strconv.Atoi(s); err != nil {
		return fmt.Errorf("must be numeric (YYYYMM), got %q", s)
	}
	month, _ := strconv.Atoi(s[4:])
	if month < 1 || month > 12 {
		return fmt.Errorf("month out of range in %q", s)
	}
	return nil
}

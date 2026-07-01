package runner

import "testing"

func TestEffectiveConnectionType(t *testing.T) {
	cases := []struct {
		in     string
		want   string
		wantOK bool
	}{
		{"", "", true},
		{"all", "", true},
		{"ALL", "", true},
		{"4g", "4G", true},
		{"4G", "4G", true},
		{"3g", "3G", true},
		{"2g", "2G", true},
		{"slow-2g", "slow-2G", true},
		{"Slow-2G", "slow-2G", true},
		{"offline", "offline", true},
		{"5g", "", false},
		{"wifi", "", false},
	}
	for _, c := range cases {
		got, ok := effectiveConnectionType(c.in)
		if got != c.want || ok != c.wantOK {
			t.Errorf("effectiveConnectionType(%q) = (%q, %v); want (%q, %v)", c.in, got, ok, c.want, c.wantOK)
		}
	}
}

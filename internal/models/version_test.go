package models

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b   string
		expect int
	}{
		{"0.1.0", "0.3.3", -1},
		{"0.3.3", "0.1.0", 1},
		{"0.3.3", "0.3.3", 0},
		{"0.10.0", "0.3.0", 1},
		{"v0.3.3", "0.3.3", 0},
		{"0.3.3", "0.3.4", -1},
	}

	for _, tt := range tests {
		got := CompareVersions(tt.a, tt.b)
		if got != tt.expect {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expect)
		}
	}
}

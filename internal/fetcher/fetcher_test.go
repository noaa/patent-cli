package fetcher

import "testing"

func TestNormalizeForURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Grant numbers (7+ digits) — passed through unchanged
		{"US8725880B2", "US8725880B2"},
		{"US9999999B2", "US9999999B2"},
		// Publication numbers with 6-digit sequence — zero-padded to 7
		{"US2025350789", "US20250350789"},
		// Publication numbers already at 7 digits — no change
		{"US20250350789", "US20250350789"},
		// Non-US numbers — passed through unchanged
		{"EP2556640B1", "EP2556640B1"},
		{"KR101436225B1", "KR101436225B1"},
		{"JP5596849B2", "JP5596849B2"},
		{"CN102859962B", "CN102859962B"},
		{"WO2011126505A1", "WO2011126505A1"},
		// Normalization: lowercase + punctuation
		{"us8725880b2", "US8725880B2"},
		{"US-8,725,880-B2", "US8725880B2"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := NormalizeForURL(tc.input)
			if got != tc.want {
				t.Errorf("NormalizeForURL(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

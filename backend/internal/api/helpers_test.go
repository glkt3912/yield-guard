package api

import (
	"strings"
	"testing"
)

func TestSanitizeQuery(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantSubs []string // 含まれるべき部分文字列
		wantNot  []string // 含まれてはいけない部分文字列
	}{
		{
			name:     "safe params pass through",
			raw:      "area=13&year=2024",
			wantSubs: []string{"area=13", "year=2024"},
			wantNot:  []string{"REDACTED"},
		},
		{
			name:     "unsafe param value is redacted",
			raw:      "address=東京都千代田区1-1",
			wantSubs: []string{"address=[REDACTED]"},
			wantNot:  []string{"東京都"},
		},
		{
			name:     "mixed safe and unsafe params",
			raw:      "area=13&address=東京都&year=2024",
			wantSubs: []string{"area=13", "year=2024", "address=[REDACTED]"},
			wantNot:  []string{"東京都"},
		},
		{
			name:     "multiple values for unsafe key are all redacted",
			raw:      "address=foo&address=bar",
			wantSubs: []string{"[REDACTED]"},
			wantNot:  []string{"foo", "bar"},
		},
		{
			name:     "malformed query returns UNPARSEABLE",
			raw:      "%zz",
			wantSubs: []string{"[UNPARSEABLE]"},
		},
		{
			name:     "empty query returns empty string",
			raw:      "",
			wantSubs: []string{},
			wantNot:  []string{"REDACTED"},
		},
		{
			name:     "unknown future param is redacted",
			raw:      "user_id=12345",
			wantSubs: []string{"[REDACTED]"},
			wantNot:  []string{"12345"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeQuery(tt.raw)
			for _, want := range tt.wantSubs {
				if want != "" && !strings.Contains(got, want) {
					t.Errorf("sanitizeQuery(%q) = %q, want to contain %q", tt.raw, got, want)
				}
			}
			for _, notWant := range tt.wantNot {
				if strings.Contains(got, notWant) {
					t.Errorf("sanitizeQuery(%q) = %q, must NOT contain %q", tt.raw, got, notWant)
				}
			}
		})
	}
}

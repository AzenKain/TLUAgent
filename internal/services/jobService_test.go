package services

import (
	"testing"
)

func TestParseCrawlPayload(t *testing.T) {
	tests := []struct {
		name        string
		payload     string
		wantPages   int
		wantDays    int
		wantReverse bool
	}{
		{
			name:        "empty payload",
			payload:     "",
			wantPages:   0,
			wantDays:    0,
			wantReverse: false,
		},
		{
			name:        "empty json object",
			payload:     "{}",
			wantPages: 0,
			wantDays:  0,
		},
		{
			name:      "invalid json",
			payload:   "invalid-json",
			wantPages: 0,
			wantDays:  0,
		},
		{
			name:      "standard max_pages and max_time_days",
			payload:   `{"max_pages": 5, "max_time_days": 7}`,
			wantPages: 5,
			wantDays:  7,
		},
		{
			name:      "legacy limit alias",
			payload:   `{"limit": 20}`,
			wantPages: 20,
			wantDays:  0,
		},
		{
			name:      "legacy max_time alias",
			payload:   `{"max_time": 14}`,
			wantPages: 0,
			wantDays:  14,
		},
		{
			name:      "pages and days aliases",
			payload:   `{"pages": 8, "days": 30}`,
			wantPages: 8,
			wantDays:  30,
		},
		{
			name:      "primary takes precedence over aliases",
			payload:   `{"limit": 10, "max_pages": 3, "max_time": 5, "max_time_days": 12}`,
			wantPages: 3,
			wantDays:  12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPages, gotDays := parseCrawlPayload(tt.payload)
			if gotPages != tt.wantPages {
				t.Errorf("parseCrawlPayload(%q) gotPages = %v, want %v", tt.payload, gotPages, tt.wantPages)
			}
			if gotDays != tt.wantDays {
				t.Errorf("parseCrawlPayload(%q) gotDays = %v, want %v", tt.payload, gotDays, tt.wantDays)
			}
		})
	}
}

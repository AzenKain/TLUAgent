package services

import (
	"strings"
	"testing"

	"tluagent-web/pkg/guardrail"
)

func TestRedTeamObservationStreamFilterLeaksSplitMarkers(t *testing.T) {
	markers := []string{
		"CONFIDENTIAL INTERNAL RULES:",
		"[SYSTEM INSTRUCTION]",
		"<user_query>",
	}
	for _, marker := range markers {
		t.Run(marker, func(t *testing.T) {
			if guardrail.SanitizeOutput(marker) == marker {
				t.Fatal("control failed: complete marker should be sanitized")
			}
			const splitAt = 5
			var emitted strings.Builder
			filter := newStreamFilterBuffer(80, func(chunk string) error {
				emitted.WriteString(chunk)
				return nil
			})
			payload := marker + strings.Repeat("x", 80-len(marker)+splitAt)
			if err := filter.Feed(payload); err != nil {
				t.Fatal(err)
			}
			if err := filter.Flush(); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(emitted.String(), marker) {
				t.Fatalf("confidential marker leaked across stream boundary: %s", marker)
			}
		})
	}
}

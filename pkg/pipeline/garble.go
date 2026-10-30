package pipeline

import (
	"regexp"
	"strings"
)

// GarbleThreshold is the corrupted-token ratio above which an extracted text layer is treated as unreliable.
const GarbleThreshold = 0.02

const (
	garbleMinTokens   = 25
	garbleMaxTokenLen = 30
)

const (
	garbleLower  = `a-z\x{00E0}-\x{1EF9}`
	garbleSymbol = "~§^€`"
)

var garbleTokenRegex = regexp.MustCompile("[" + garbleLower + "][0-9" + garbleSymbol + "][" + garbleLower + "]|\\x{FFFD}|[" + garbleLower + "][" + garbleSymbol + "]\\b|\\b[" + garbleSymbol + "][" + garbleLower + "]")

// TextGarbleRatio returns the share of tokens carrying stray symbols or digits inside lowercase words, a mark of a broken OCR text layer.
func TextGarbleRatio(text string) float64 {
	tokens := strings.Fields(text)
	counted, bad := 0, 0
	for _, tok := range tokens {
		if strings.HasPrefix(tok, "\\") || strings.ContainsAny(tok, "%=&@?") || len(tok) > garbleMaxTokenLen {
			continue
		}
		counted++
		if garbleTokenRegex.MatchString(tok) {
			bad++
		}
	}
	if counted < garbleMinTokens {
		return 0
	}
	return float64(bad) / float64(counted)
}

// IsGarbledText reports whether the text looks like a corrupted OCR text layer.
func IsGarbledText(text string) bool {
	return TextGarbleRatio(text) >= GarbleThreshold
}

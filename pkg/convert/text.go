package convert

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// StripAccents removes Vietnamese diacritics and converts non-ASCII Latin characters to base ASCII.
func StripAccents(text string) string {
	decomposed := norm.NFD.String(text)
	var out strings.Builder
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if r == 'đ' || r == 'Đ' {
			out.WriteRune('d')
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}

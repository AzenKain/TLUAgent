package guardrail

import (
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	zeroWidthRegex = regexp.MustCompile("[\u200B\u200C\u200D\uFEFF\u00AD\u2060\u200E\u200F]+")
	unicodeEscape  = regexp.MustCompile(`(?i)\\u([0-9a-f]{4})`)
	hexEscape      = regexp.MustCompile(`(?i)\\x([0-9a-f]{2})`)

	cyrillicToLatin = map[rune]rune{
		'а': 'a', 'е': 'e', 'о': 'o', 'р': 'p', 'с': 'c', 'у': 'y', 'х': 'x', 'і': 'i', 'ј': 'j',
		'А': 'A', 'В': 'B', 'Е': 'E', 'К': 'K', 'М': 'M', 'Н': 'H', 'О': 'O', 'Р': 'P', 'С': 'C',
		'Т': 'T', 'Х': 'X', 'І': 'I', 'Ј': 'J',
	}
)

// NormalizeQuery canonicalizes text by unescaping, stripping invisible runes, and folding confusable homoglyphs.
func NormalizeQuery(input string) string {
	if strings.TrimSpace(input) == "" {
		return ""
	}

	unescaped := input
	if decoded, err := url.QueryUnescape(input); err == nil && decoded != "" {
		unescaped = decoded
	}

	if unicodeEscape.MatchString(unescaped) {
		unescaped = unicodeEscape.ReplaceAllStringFunc(unescaped, func(m string) string {
			if code, err := strconv.ParseInt(m[2:], 16, 32); err == nil {
				return string(rune(code))
			}
			return m
		})
	}

	if hexEscape.MatchString(unescaped) {
		unescaped = hexEscape.ReplaceAllStringFunc(unescaped, func(m string) string {
			if code, err := strconv.ParseInt(m[2:], 16, 32); err == nil {
				return string(rune(code))
			}
			return m
		})
	}

	stripped := zeroWidthRegex.ReplaceAllString(unescaped, "")
	decomposed := norm.NFKD.String(stripped)

	var sb strings.Builder
	sb.Grow(len(decomposed))
	for _, r := range decomposed {
		if mapped, ok := cyrillicToLatin[r]; ok {
			sb.WriteRune(mapped)
		} else {
			sb.WriteRune(r)
		}
	}

	return strings.TrimSpace(norm.NFC.String(sb.String()))
}

// FoldToASCII strips zero-width characters, maps homoglyphs, and removes combining diacritical marks.
func FoldToASCII(input string) string {
	if strings.TrimSpace(input) == "" {
		return ""
	}
	unescaped := input
	if decoded, err := url.QueryUnescape(input); err == nil && decoded != "" {
		unescaped = decoded
	}
	stripped := zeroWidthRegex.ReplaceAllString(unescaped, "")
	decomposed := norm.NFKD.String(stripped)

	var sb strings.Builder
	sb.Grow(len(decomposed))
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if r == 'đ' || r == 'Đ' {
			sb.WriteRune('d')
			continue
		}
		if mapped, ok := cyrillicToLatin[r]; ok {
			sb.WriteRune(mapped)
		} else {
			sb.WriteRune(r)
		}
	}
	return strings.ToLower(strings.TrimSpace(sb.String()))
}

// NormalizeLeetspeak converts standard leetspeak digit/symbol substitutions to latin alphabet characters.
func NormalizeLeetspeak(input string) string {
	if strings.TrimSpace(input) == "" {
		return ""
	}
	var sb strings.Builder
	sb.Grow(len(input))
	for _, r := range input {
		switch r {
		case '0':
			sb.WriteRune('o')
		case '1':
			sb.WriteRune('i')
		case '3':
			sb.WriteRune('e')
		case '4', '@':
			sb.WriteRune('a')
		case '5', '$':
			sb.WriteRune('s')
		case '7':
			sb.WriteRune('t')
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// ExtractDecodedPayloads recursively unpacks nested Base64 and Hex encoded payloads.
func ExtractDecodedPayloads(input string, maxDepth int) []string {
	if maxDepth <= 0 || len(input) < 16 {
		return nil
	}

	var payloads []string
	cleanInput := strings.TrimSpace(input)

	candidateWords := strings.Fields(cleanInput)
	for _, word := range candidateWords {
		cleanedWord := strings.Trim(word, `"'` + "`" + `()[]{}<>,.;:`)

		var b64Data []byte
		var b64Err error
		if strings.HasSuffix(cleanedWord, "=") || len(cleanedWord) >= 20 {
			for _, enc := range []*base64.Encoding{
				base64.StdEncoding,
				base64.URLEncoding,
				base64.RawStdEncoding,
				base64.RawURLEncoding,
			} {
				b64Data, b64Err = enc.DecodeString(cleanedWord)
				if b64Err == nil && len(b64Data) >= 8 {
					decodedText := string(b64Data)
					payloads = append(payloads, decodedText)
					nested := ExtractDecodedPayloads(decodedText, maxDepth-1)
					payloads = append(payloads, nested...)
					break
				}
			}
		}

		hexCandidate := strings.TrimPrefix(cleanedWord, "0x")
		hexCandidate = strings.TrimPrefix(hexCandidate, "0X")
		if len(hexCandidate) >= 20 && len(hexCandidate)%2 == 0 {
			if hexBytes, hexErr := hex.DecodeString(hexCandidate); hexErr == nil && len(hexBytes) >= 8 {
				decodedHex := string(hexBytes)
				payloads = append(payloads, decodedHex)
				nested := ExtractDecodedPayloads(decodedHex, maxDepth-1)
				payloads = append(payloads, nested...)
			}
		}
	}

	return payloads
}

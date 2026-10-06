package embedding

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const (
	clsToken           = "[CLS]"
	sepToken           = "[SEP]"
	unkToken           = "[UNK]"
	continuationPrefix = "##"
)

// WordPieceTokenizer implements the BERT WordPiece algorithm over a vocab.txt file.
type WordPieceTokenizer struct {
	vocab        map[string]int
	lowercase    bool
	stripAccents bool
	clsID        int
	sepID        int
	unkID        int
}

// LoadWordPieceTokenizer builds a tokenizer from a BERT vocab.txt file honoring manifest flags.
func LoadWordPieceTokenizer(vocabPath string, lowercase, stripAccents bool) (*WordPieceTokenizer, error) {
	data, err := readBoundedFile(vocabPath, MaxVocabSizeBytes)
	if err != nil {
		return nil, fmt.Errorf("read vocab file: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	vocab := make(map[string]int, len(lines))
	for i, line := range lines {
		token := strings.TrimRight(line, "\r")
		if i == len(lines)-1 && token == "" {
			continue
		}
		vocab[token] = i
	}
	clsID, ok := vocab[clsToken]
	if !ok {
		return nil, fmt.Errorf("vocab file is missing %s token", clsToken)
	}
	sepID, ok := vocab[sepToken]
	if !ok {
		return nil, fmt.Errorf("vocab file is missing %s token", sepToken)
	}
	unkID, ok := vocab[unkToken]
	if !ok {
		return nil, fmt.Errorf("vocab file is missing %s token", unkToken)
	}
	return &WordPieceTokenizer{
		vocab:        vocab,
		lowercase:    lowercase,
		stripAccents: stripAccents,
		clsID:        clsID,
		sepID:        sepID,
		unkID:        unkID,
	}, nil
}

// Tokenize converts raw text into WordPiece tokens using BERT basic tokenization rules.
func (t *WordPieceTokenizer) Tokenize(text string) []string {
	var tokens []string
	for _, word := range t.basicTokenize(text) {
		tokens = append(tokens, t.wordPiece(word)...)
	}
	return tokens
}

// TokenIDs returns input_ids with [CLS]/[SEP] wrapping truncated to maxSeqTokens.
func (t *WordPieceTokenizer) TokenIDs(text string, maxSeqTokens int) []int64 {
	pieces := t.Tokenize(text)
	limit := maxSeqTokens - 2
	if len(pieces) > limit {
		pieces = pieces[:limit]
	}
	ids := make([]int64, 0, len(pieces)+2)
	ids = append(ids, int64(t.clsID))
	for _, piece := range pieces {
		ids = append(ids, int64(t.vocab[piece]))
	}
	ids = append(ids, int64(t.sepID))
	return ids
}

func (t *WordPieceTokenizer) basicTokenize(text string) []string {
	var cleaned strings.Builder
	for _, r := range text {
		if r == 0 || r == 0xFFFD || isControlRune(r) {
			continue
		}
		if unicode.IsSpace(r) {
			cleaned.WriteRune(' ')
			continue
		}
		cleaned.WriteRune(r)
	}
	normalized := padCJK(cleaned.String())
	if t.lowercase {
		normalized = strings.ToLower(normalized)
	}
	if t.stripAccents {
		normalized = stripAccents(normalized)
	}
	var words []string
	for _, field := range strings.Fields(normalized) {
		words = append(words, splitPunctuation(field)...)
	}
	return words
}

func (t *WordPieceTokenizer) wordPiece(word string) []string {
	runes := []rune(word)
	var pieces []string
	start := 0
	for start < len(runes) {
		end := len(runes)
		match := ""
		for start < end {
			sub := string(runes[start:end])
			if start > 0 {
				sub = continuationPrefix + sub
			}
			if _, ok := t.vocab[sub]; ok {
				match = sub
				break
			}
			end--
		}
		if match == "" {
			return []string{unkToken}
		}
		pieces = append(pieces, match)
		start = end
	}
	return pieces
}

func padCJK(text string) string {
	var out strings.Builder
	for _, r := range text {
		if isCJKRune(r) {
			out.WriteRune(' ')
			out.WriteRune(r)
			out.WriteRune(' ')
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}

func isCJKRune(r rune) bool {
	switch {
	case r >= 0x4E00 && r <= 0x9FFF:
		return true
	case r >= 0x3400 && r <= 0x4DBF:
		return true
	case r >= 0x20000 && r <= 0x2A6DF:
		return true
	case r >= 0x2A700 && r <= 0x2B73F:
		return true
	case r >= 0x2B740 && r <= 0x2B81F:
		return true
	case r >= 0x2B820 && r <= 0x2CEAF:
		return true
	case r >= 0xF900 && r <= 0xFAFF:
		return true
	case r >= 0x2F800 && r <= 0x2FA1F:
		return true
	default:
		return false
	}
}

func stripAccents(text string) string {
	decomposed := norm.NFD.String(text)
	var out strings.Builder
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}

func splitPunctuation(word string) []string {
	var out []string
	var pending strings.Builder
	for _, r := range word {
		if isPunctuationRune(r) {
			if pending.Len() > 0 {
				out = append(out, pending.String())
				pending.Reset()
			}
			out = append(out, string(r))
			continue
		}
		pending.WriteRune(r)
	}
	if pending.Len() > 0 {
		out = append(out, pending.String())
	}
	return out
}

func isPunctuationRune(r rune) bool {
	switch {
	case r >= 33 && r <= 47, r >= 58 && r <= 64, r >= 91 && r <= 96, r >= 123 && r <= 126:
		return true
	default:
		return unicode.IsPunct(r)
	}
}

func isControlRune(r rune) bool {
	switch r {
	case '\t', '\n', '\r':
		return false
	}
	return unicode.Is(unicode.Cc, r) || unicode.Is(unicode.Cf, r)
}

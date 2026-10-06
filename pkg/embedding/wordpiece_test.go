package embedding

import (
	"os"
	"path/filepath"
	"testing"
)

const testVocabContent = "[PAD]\n[UNK]\n[CLS]\n[SEP]\n[MASK]\nthe\n##re\nfast\nbrown\nfox\nhello\nworld\nxin\nchao\nch\u00e0o\nt\u1ea1m\nbi\u1ec7t\n.\n,\n"

func writeTestVocab(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), VocabFileName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write vocab: %v", err)
	}
	return path
}

func newTestTokenizer(t *testing.T, lowercase, stripAccents bool) *WordPieceTokenizer {
	t.Helper()
	tokenizer, err := LoadWordPieceTokenizer(writeTestVocab(t, testVocabContent), lowercase, stripAccents)
	if err != nil {
		t.Fatalf("failed to load tokenizer: %v", err)
	}
	return tokenizer
}

func TestWordPieceTokenizer_BasicLowercase(t *testing.T) {
	tokenizer := newTestTokenizer(t, true, false)
	tokens := tokenizer.Tokenize("Hello World")
	if len(tokens) != 2 || tokens[0] != "hello" || tokens[1] != "world" {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
}

func TestWordPieceTokenizer_CasedKeepsAccents(t *testing.T) {
	tokenizer := newTestTokenizer(t, false, false)
	tokens := tokenizer.Tokenize("ch\u00e0o")
	if len(tokens) != 1 || tokens[0] != "ch\u00e0o" {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
	tokens = tokenizer.Tokenize("Ch\u00e0o")
	if len(tokens) != 1 || tokens[0] != "[UNK]" {
		t.Fatalf("cased tokenizer must not match the lowercase vocab entry: %v", tokens)
	}
}

func TestWordPieceTokenizer_LowercaseThenStripAccents(t *testing.T) {
	lowerKeep := newTestTokenizer(t, true, false)
	tokens := lowerKeep.Tokenize("CH\u00c0O")
	if len(tokens) != 1 || tokens[0] != "ch\u00e0o" {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
	lowerStrip := newTestTokenizer(t, true, true)
	tokens = lowerStrip.Tokenize("CH\u00c0O")
	if len(tokens) != 1 || tokens[0] != "chao" {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
}

func TestWordPieceTokenizer_StripAccents(t *testing.T) {
	tokenizer := newTestTokenizer(t, true, true)
	tokens := tokenizer.Tokenize("Ch\u00e0o")
	if len(tokens) != 1 || tokens[0] != "chao" {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
}

func TestWordPieceTokenizer_PunctuationSplit(t *testing.T) {
	tokenizer := newTestTokenizer(t, true, false)
	tokens := tokenizer.Tokenize("fast, brown fox.")
	expected := []string{"fast", ",", "brown", "fox", "."}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, tokens)
	}
	for i := range expected {
		if tokens[i] != expected[i] {
			t.Fatalf("expected %v, got %v", expected, tokens)
		}
	}
}

func TestWordPieceTokenizer_GreedyContinuation(t *testing.T) {
	tokenizer := newTestTokenizer(t, true, false)
	tokens := tokenizer.Tokenize("There")
	if len(tokens) != 2 || tokens[0] != "the" || tokens[1] != "##re" {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
}

func TestWordPieceTokenizer_UnknownWord(t *testing.T) {
	tokenizer := newTestTokenizer(t, true, false)
	tokens := tokenizer.Tokenize("zzz")
	if len(tokens) != 1 || tokens[0] != "[UNK]" {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
	ids := tokenizer.TokenIDs("zzz", 16)
	if len(ids) != 3 || ids[0] != 2 || ids[1] != 1 || ids[2] != 3 {
		t.Fatalf("unexpected ids: %v", ids)
	}
}

func TestWordPieceTokenizer_UnknownContinuationFailsWholeWord(t *testing.T) {
	tokenizer := newTestTokenizer(t, true, false)
	tokens := tokenizer.Tokenize("thzzz")
	if len(tokens) != 1 || tokens[0] != "[UNK]" {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
}

func TestWordPieceTokenizer_Truncation(t *testing.T) {
	tokenizer := newTestTokenizer(t, true, false)
	ids := tokenizer.TokenIDs("fast fast fast fast fast fast", 6)
	if len(ids) != 6 {
		t.Fatalf("expected 6 ids after truncation, got %d: %v", len(ids), ids)
	}
	if ids[0] != 2 || ids[len(ids)-1] != 3 {
		t.Fatalf("missing special tokens: %v", ids)
	}
}

func TestWordPieceTokenizer_CJKPadding(t *testing.T) {
	tokenizer := newTestTokenizer(t, true, false)
	tokens := tokenizer.Tokenize("你好")
	if len(tokens) != 2 || tokens[0] != "[UNK]" || tokens[1] != "[UNK]" {
		t.Fatalf("CJK characters must be split into separate tokens: %v", tokens)
	}
}

func TestWordPieceTokenizer_ControlCharsDropped(t *testing.T) {
	tokenizer := newTestTokenizer(t, true, false)
	tokens := tokenizer.Tokenize("fast\tbrown")
	if len(tokens) != 2 || tokens[0] != "fast" || tokens[1] != "brown" {
		t.Fatalf("whitespace control chars must split into two tokens: %v", tokens)
	}
	tokens = tokenizer.Tokenize("fast\x00brown")
	if len(tokens) != 1 || tokens[0] != "[UNK]" {
		t.Fatalf("dropped control chars must leave one merged word: %v", tokens)
	}
}

func TestWordPieceTokenizer_MissingSpecialTokens(t *testing.T) {
	path := writeTestVocab(t, "the\nfast\n")
	if _, err := LoadWordPieceTokenizer(path, true, false); err == nil {
		t.Fatalf("expected error for vocab missing special tokens")
	}
}

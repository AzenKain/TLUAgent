package embedding

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
)

const (
	BosTokenID        = 0
	PadTokenID        = 1
	EosTokenID        = 2
	UnkTokenID        = 3
	TokenizerFileName = "tokenizer.json"
	TokenizerUnigram  = "unigram"
)

type tokenizerJSONModel struct {
	Type  string          `json:"type"`
	Vocab [][]interface{} `json:"vocab"`
}

type tokenizerJSONFile struct {
	Model tokenizerJSONModel `json:"model"`
}

// UnigramTokenizer implements the SentencePiece Unigram tokenization algorithm.
type UnigramTokenizer struct {
	tokenToID    map[string]int64
	tokenToScore map[string]float64
}

// LoadUnigramTokenizer loads vocabulary and scores from a HuggingFace tokenizer.json file.
func LoadUnigramTokenizer(path string) (*UnigramTokenizer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read tokenizer file: %w", err)
	}

	var tj tokenizerJSONFile
	if err := json.Unmarshal(data, &tj); err != nil {
		return nil, fmt.Errorf("parse tokenizer json: %w", err)
	}

	tokenToID := make(map[string]int64, len(tj.Model.Vocab))
	tokenToScore := make(map[string]float64, len(tj.Model.Vocab))

	for i, item := range tj.Model.Vocab {
		if len(item) >= 2 {
			token, ok := item[0].(string)
			score, sOk := item[1].(float64)
			if ok && sOk {
				tokenToID[token] = int64(i)
				tokenToScore[token] = score
			}
		}
	}

	return &UnigramTokenizer{
		tokenToID:    tokenToID,
		tokenToScore: tokenToScore,
	}, nil
}

// TokenIDs tokenizes text using Viterbi dynamic programming and wraps with BOS and EOS tokens.
func (u *UnigramTokenizer) TokenIDs(text string, maxSeqTokens int) []int64 {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return []int64{BosTokenID, EosTokenID}
	}

	normalized := strings.ReplaceAll(trimmed, " ", "▁")
	if !strings.HasPrefix(normalized, "▁") {
		normalized = "▁" + normalized
	}

	runes := []rune(normalized)
	n := len(runes)

	dp := make([]float64, n+1)
	for i := range dp {
		dp[i] = math.Inf(-1)
	}
	dp[0] = 0.0

	type step struct {
		prev  int
		token string
	}
	backtrack := make([]step, n+1)

	for i := 0; i < n; i++ {
		if math.IsInf(dp[i], -1) {
			continue
		}
		maxJ := i + 32
		if maxJ > n {
			maxJ = n
		}
		for j := i + 1; j <= maxJ; j++ {
			sub := string(runes[i:j])
			if score, ok := u.tokenToScore[sub]; ok {
				newScore := dp[i] + score
				if newScore > dp[j] {
					dp[j] = newScore
					backtrack[j] = step{prev: i, token: sub}
				}
			} else if j == i+1 {
				newScore := dp[i] - 100.0
				if newScore > dp[j] {
					dp[j] = newScore
					backtrack[j] = step{prev: i, token: "<unk>"}
				}
			}
		}
	}

	tokens := make([]string, 0)
	curr := n
	for curr > 0 {
		s := backtrack[curr]
		tokens = append(tokens, s.token)
		curr = s.prev
	}

	for i, j := 0, len(tokens)-1; i < j; i, j = i+1, j-1 {
		tokens[i], tokens[j] = tokens[j], tokens[i]
	}

	limit := maxSeqTokens - 2
	if limit > 0 && len(tokens) > limit {
		tokens = tokens[:limit]
	}

	ids := make([]int64, 0, len(tokens)+2)
	ids = append(ids, BosTokenID)
	for _, tok := range tokens {
		if id, ok := u.tokenToID[tok]; ok {
			ids = append(ids, id)
		} else {
			ids = append(ids, UnkTokenID)
		}
	}
	ids = append(ids, EosTokenID)

	return ids
}

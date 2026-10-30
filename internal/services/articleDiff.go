package services

import (
	"fmt"
	"regexp"
	"strings"

	"tluagent-web/pkg/supersede"
)

const (
	diffExcerptRunes = 180
	diffMaxRunes     = 3200
	diffMaxSentences = 6
)

var numericTokenRegex = regexp.MustCompile(`\d+(?:[.,]\d+)?`)

func numberedArticles(text string) []supersede.ParsedArticle {
	var out []supersede.ParsedArticle
	for _, a := range supersede.ParseArticles(text) {
		if a.ArticleNumber != supersede.FullTextArticle {
			out = append(out, a)
		}
	}
	return out
}

func articleKey(a supersede.ParsedArticle) string {
	return strings.ToLower(strings.Join(strings.Fields(a.ArticleNumber), " "))
}

func excerpt(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if runes := []rune(text); len(runes) > diffExcerptRunes {
		return string(runes[:diffExcerptRunes]) + "..."
	}
	return text
}

func numericTokens(text string) map[string]bool {
	out := map[string]bool{}
	for _, tok := range numericTokenRegex.FindAllString(text, -1) {
		out[tok] = true
	}
	return out
}

func tokenDelta(oldText, newText string) string {
	oldSet, newSet := numericTokens(oldText), numericTokens(newText)
	var removed, added []string
	for tok := range oldSet {
		if !newSet[tok] {
			removed = append(removed, tok)
		}
	}
	for tok := range newSet {
		if !oldSet[tok] {
			added = append(added, tok)
		}
	}
	if len(removed) == 0 && len(added) == 0 {
		return ""
	}
	return fmt.Sprintf(" | numbers removed: %v, added: %v", removed, added)
}

func sentencesOf(text string) []string {
	var out []string
	for _, part := range regexp.MustCompile(`[.;\n]+`).Split(text, -1) {
		if s := strings.Join(strings.Fields(part), " "); len(s) > 12 {
			out = append(out, s)
		}
	}
	return out
}

func sentenceDiff(oldText, newText string) string {
	oldSet := map[string]bool{}
	for _, s := range sentencesOf(oldText) {
		oldSet[strings.ToLower(s)] = true
	}
	newSet := map[string]bool{}
	for _, s := range sentencesOf(newText) {
		newSet[strings.ToLower(s)] = true
	}
	var sb strings.Builder
	removed, added := 0, 0
	for _, s := range sentencesOf(oldText) {
		if !newSet[strings.ToLower(s)] && removed < diffMaxSentences {
			sb.WriteString("REMOVED: ")
			sb.WriteString(excerpt(s))
			sb.WriteString("\n")
			removed++
		}
	}
	for _, s := range sentencesOf(newText) {
		if !oldSet[strings.ToLower(s)] && added < diffMaxSentences {
			sb.WriteString("ADDED: ")
			sb.WriteString(excerpt(s))
			sb.WriteString("\n")
			added++
		}
	}
	if sb.Len() == 0 {
		return "No sentence-level difference found.\n"
	}
	return sb.String()
}

// structuralArticleDiff compares two documents article by article and falls back to sentences when no articles are detected.
func structuralArticleDiff(oldText, newText, articleFilter string) string {
	oldArticles, newArticles := numberedArticles(oldText), numberedArticles(newText)
	if len(oldArticles) == 0 && len(newArticles) == 0 {
		return truncateDiff(sentenceDiff(oldText, newText))
	}

	oldByKey := map[string]supersede.ParsedArticle{}
	for _, a := range oldArticles {
		oldByKey[articleKey(a)] = a
	}
	seen := map[string]bool{}
	filter := strings.ToLower(strings.TrimSpace(articleFilter))
	var sb strings.Builder
	for _, n := range newArticles {
		key := articleKey(n)
		if filter != "" && !strings.Contains(key, filter) {
			continue
		}
		seen[key] = true
		old, ok := oldByKey[key]
		switch {
		case !ok:
			sb.WriteString(fmt.Sprintf("%s: ONLY IN INCOMING DOCUMENT | %s\n", n.ArticleNumber, excerpt(n.Content)))
		case strings.EqualFold(strings.Join(strings.Fields(old.Content), " "), strings.Join(strings.Fields(n.Content), " ")):
			sb.WriteString(fmt.Sprintf("%s: IDENTICAL\n", n.ArticleNumber))
		default:
			sb.WriteString(fmt.Sprintf("%s: CHANGED | old: %s | new: %s%s\n", n.ArticleNumber, excerpt(old.Content), excerpt(n.Content), tokenDelta(old.Content, n.Content)))
		}
	}
	for _, o := range oldArticles {
		key := articleKey(o)
		if filter != "" && !strings.Contains(key, filter) {
			continue
		}
		if !seen[key] {
			sb.WriteString(fmt.Sprintf("%s: ONLY IN EARLIER DOCUMENT | %s\n", o.ArticleNumber, excerpt(o.Content)))
		}
	}
	return truncateDiff(sb.String())
}

func truncateDiff(s string) string {
	if runes := []rune(s); len(runes) > diffMaxRunes {
		return string(runes[:diffMaxRunes]) + "\n...[truncated]"
	}
	return s
}

package supersede

import (
	"regexp"
	"strings"
)

// ParsedArticle represents an individual article extracted from regulatory text.
type ParsedArticle struct {
	Index         int    `json:"index"`
	Chapter       string `json:"chapter,omitempty"`
	ArticleNumber string `json:"article_number"`
	Title         string `json:"title"`
	Content       string `json:"content"`
}

var (
	chapterRegex = regexp.MustCompile(`(?i)^\s*(?:Chương|CHƯƠNG)\s+([IVXLCDM\d]+)[.:\s-]*(.*)$`)
	articleRegex = regexp.MustCompile(`(?i)^\s*(?:Điều|ĐIỀU)\s+(\d+[a-zA-Z]?)[.:\s-]*([^\n]*)$`)
)

// ParseArticles decomposes regulatory markdown or plaintext into structured articles.
func ParseArticles(content string) []ParsedArticle {
	lines := strings.Split(content, "\n")
	var articles []ParsedArticle

	currentChapter := ""
	currentArticleNum := ""
	currentArticleTitle := ""
	var currentBody []string
	articleIndex := 0

	flushArticle := func() {
		if currentArticleNum == "" && len(currentBody) == 0 {
			return
		}
		num := currentArticleNum
		if num == "" {
			num = "Toàn văn"
		}
		title := currentArticleTitle
		if title == "" {
			title = num
		}
		body := strings.TrimSpace(strings.Join(currentBody, "\n"))
		articles = append(articles, ParsedArticle{
			Index:         articleIndex,
			Chapter:       currentChapter,
			ArticleNumber: num,
			Title:         title,
			Content:       body,
		})
		articleIndex++
		currentArticleNum = ""
		currentArticleTitle = ""
		currentBody = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if currentArticleNum != "" {
				currentBody = append(currentBody, "")
			}
			continue
		}

		if m := chapterRegex.FindStringSubmatch(trimmed); len(m) > 1 {
			chTitle := strings.TrimSpace(m[2])
			if chTitle != "" {
				currentChapter = "Chương " + m[1] + " - " + chTitle
			} else {
				currentChapter = "Chương " + m[1]
			}
			continue
		}

		if m := articleRegex.FindStringSubmatch(trimmed); len(m) > 1 {
			flushArticle()
			currentArticleNum = "Điều " + m[1]
			currentArticleTitle = strings.TrimSpace(m[2])
			continue
		}

		if currentArticleNum != "" {
			currentBody = append(currentBody, line)
		} else {
			currentBody = append(currentBody, line)
		}
	}

	flushArticle()

	if len(articles) == 0 && strings.TrimSpace(content) != "" {
		articles = append(articles, ParsedArticle{
			Index:         0,
			Chapter:       "",
			ArticleNumber: "Toàn văn",
			Title:         "Nội dung văn bản",
			Content:       strings.TrimSpace(content),
		})
	}

	return articles
}

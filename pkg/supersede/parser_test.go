package supersede

import (
	"testing"
)

func TestParseArticles(t *testing.T) {
	text := `Chương I: QUY ĐỊNH CHUNG
Điều 1. Phạm vi điều chỉnh
Quy chế này quy định về đào tạo trình độ đại học theo hệ thống tín chỉ.

Điều 2. Đối tượng áp dụng
Áp dụng đối với tất cả sinh viên hệ chính quy.

Chương II: CHUẨN ĐẦU RA
Điều 15. Chuẩn đầu ra Ngoại ngữ
Sinh viên phải đạt chứng chỉ TOEIC 450 điểm.`

	articles := ParseArticles(text)
	if len(articles) != 3 {
		t.Fatalf("expected 3 articles, got %d", len(articles))
	}

	if articles[0].ArticleNumber != "Điều 1" || articles[0].Title != "Phạm vi điều chỉnh" {
		t.Errorf("unexpected article 0: %+v", articles[0])
	}
	if articles[0].Chapter != "Chương I - QUY ĐỊNH CHUNG" {
		t.Errorf("unexpected chapter 0: %s", articles[0].Chapter)
	}

	if articles[1].ArticleNumber != "Điều 2" || articles[1].Title != "Đối tượng áp dụng" {
		t.Errorf("unexpected article 1: %+v", articles[1])
	}

	if articles[2].ArticleNumber != "Điều 15" || articles[2].Title != "Chuẩn đầu ra Ngoại ngữ" {
		t.Errorf("unexpected article 2: %+v", articles[2])
	}
	if articles[2].Chapter != "Chương II - CHUẨN ĐẦU RA" {
		t.Errorf("unexpected chapter 2: %s", articles[2].Chapter)
	}
}

func TestParseArticlesFallback(t *testing.T) {
	text := "Thông báo về việc nghỉ lễ Quốc khánh 2/9."
	articles := ParseArticles(text)
	if len(articles) != 1 {
		t.Fatalf("expected 1 article, got %d", len(articles))
	}
	if articles[0].ArticleNumber != "Toàn văn" {
		t.Errorf("expected Toàn văn, got %s", articles[0].ArticleNumber)
	}
}

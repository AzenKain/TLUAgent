package pipeline

import (
	"strings"
	"testing"
)

func TestTextGarbleRatio_BrokenOCRLayer(t *testing.T) {
	broken := strings.Repeat("Di~u 1. Pharn vi va dBi nrong ap dung chu§n d~u ra Ti~ng Anh t6t nghiep h9c ", 6)
	if !IsGarbledText(broken) {
		t.Fatalf("expected garbled text, ratio=%.3f", TextGarbleRatio(broken))
	}
}

func TestTextGarbleRatio_CleanVietnameseAndCodes(t *testing.T) {
	clean := strings.Repeat("\u0110i\u1ec1u 1. Ph\u1ea1m vi v\u00e0 \u0111\u1ed1i t\u01b0\u1ee3ng \u00e1p d\u1ee5ng c\u1ee7a Quy\u1ebft \u0111\u1ecbnh s\u1ed1 25091102/Q\u0110-\u0110HTL cho l\u1edbp NJ30~NJ36 kh\u00f3a K35 h\u1ecdc k\u1ef3 HK3N1 ", 6)
	if IsGarbledText(clean) {
		t.Fatalf("clean text flagged, ratio=%.3f", TextGarbleRatio(clean))
	}
}

func TestTextGarbleRatio_ShortTextIgnored(t *testing.T) {
	if got := TextGarbleRatio("Di~u chu§n"); got != 0 {
		t.Fatalf("short text must not be scored, got %.3f", got)
	}
}

func TestTextGarbleRatio_IgnoresUrlFragments(t *testing.T) {
	links := strings.Repeat("meeting groupId=e364dc02-b4cb conversations?context=%7b%22Tid%22 thread.tacv2/Chung@x9y join ", 10)
	if IsGarbledText(links) {
		t.Fatalf("url fragments flagged, ratio=%.3f", TextGarbleRatio(links))
	}
}

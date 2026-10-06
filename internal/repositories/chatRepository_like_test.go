package repositories

import (
	"context"
	"testing"

	"tluagent-web/internal/models"
	"tluagent-web/pkg/cache"
)

func TestEscapeLikePattern(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "plain", want: "plain"},
		{input: "100%", want: `100\%`},
		{input: "a_b", want: `a\_b`},
		{input: `back\slash`, want: `back\\slash`},
		{input: "%_", want: `\%\_`},
	}

	for _, tt := range tests {
		if got := escapeLikePattern(tt.input); got != tt.want {
			t.Fatalf("escapeLikePattern(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestListAdminConversations_SearchEscapesWildcards(t *testing.T) {
	db := setupTestChatDB(t)
	defer db.Close()

	ramCache := cache.NewRamCache()
	repo := NewChatRepository(db, ramCache)
	ctx := context.Background()

	conv := &models.ChatConversationEntity{
		ID:     "conv-wildcard",
		UserID: "u1",
		Title:  "fixed_title_100%",
	}
	if err := repo.CreateConversation(ctx, conv); err != nil {
		t.Fatalf("CreateConversation failed: %v", err)
	}

	_, total, err := repo.ListAdminConversations(ctx, "fixed_title_100%", "", "", 10, 0)
	if err != nil {
		t.Fatalf("ListAdminConversations failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("Literal search term with wildcards should match exactly, got total %d", total)
	}

	_, total, err = repo.ListAdminConversations(ctx, "fixed%title", "", "", 10, 0)
	if err != nil {
		t.Fatalf("ListAdminConversations failed: %v", err)
	}
	if total != 0 {
		t.Fatalf("Unescaped wildcard in search term must not match the literal title, got total %d", total)
	}
}

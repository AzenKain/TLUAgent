package repositories

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"tluagent-web/internal/models"
	"tluagent-web/pkg/cache"
)

func setupTestChatDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	schema := `
	CREATE TABLE users (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL,
		full_name TEXT NOT NULL,
		student_code TEXT NOT NULL
	);
	CREATE TABLE chat_conversations (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		model_id TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE chat_messages (
		id TEXT PRIMARY KEY,
		conversation_id TEXT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
		sender TEXT NOT NULL CHECK(sender IN ('user', 'assistant')),
		content TEXT NOT NULL,
		sources_json TEXT NOT NULL DEFAULT '[]',
		images_json TEXT NOT NULL DEFAULT '[]',
		feedback TEXT CHECK(feedback IN ('up', 'down', NULL)),
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	INSERT INTO users (id, email, full_name, student_code) VALUES ('u1', 'sv1@thanglong.edu.vn', 'Nguyen Van A', 'A12345');
	INSERT INTO users (id, email, full_name, student_code) VALUES ('u2', 'sv2@thanglong.edu.vn', 'Tran Thi B', 'B67890');
	`
	_, err = db.Exec(schema)
	require.NoError(t, err)
	return db
}

func TestChatRepository_ConversationsAndMessages(t *testing.T) {
	db := setupTestChatDB(t)
	defer db.Close()

	ramCache := cache.NewRamCache()
	repo := NewChatRepository(db, ramCache)
	ctx := context.Background()

	conv := &models.ChatConversationEntity{
		ID:      "conv-1",
		UserID:  "u1",
		Title:   "Hoi ve quy che tin chi",
		ModelID: "gemini-flash",
	}

	err := repo.CreateConversation(ctx, conv)
	require.NoError(t, err)

	got, err := repo.GetConversationByID(ctx, "conv-1")
	require.NoError(t, err)
	require.Equal(t, "conv-1", got.ID)
	require.Equal(t, "u1", got.UserID)
	require.Equal(t, "Hoi ve quy che tin chi", got.Title)

	err = repo.UpdateConversationTitle(ctx, "conv-1", "u1", "Quy che tin chi TLU")
	require.NoError(t, err)

	list, total, err := repo.ListConversationsByUserID(ctx, "u1", 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Equal(t, "Quy che tin chi TLU", list[0].Title)

	userMsg := &models.ChatMessageEntity{
		ID:             "msg-1",
		ConversationID: "conv-1",
		Sender:         "user",
		Content:        "Cho minh hoi so tin chi toi thieu?",
	}
	err = repo.CreateMessage(ctx, userMsg)
	require.NoError(t, err)

	botMsg := &models.ChatMessageEntity{
		ID:             "msg-2",
		ConversationID: "conv-1",
		Sender:         "assistant",
		Content:        "According to the training regulations you must register at least 14 credits.",
		Sources:        []string{"Training Regulations 2024"},
	}
	err = repo.CreateMessage(ctx, botMsg)
	require.NoError(t, err)

	messages, err := repo.ListMessagesByConversationID(ctx, "conv-1")
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Equal(t, "user", messages[0].Sender)
	require.Equal(t, "assistant", messages[1].Sender)
	require.Equal(t, []string{"Training Regulations 2024"}, messages[1].Sources)

	err = repo.UpdateMessageFeedback(ctx, "msg-2", "u1", "up")
	require.NoError(t, err)

	messages, err = repo.ListMessagesByConversationID(ctx, "conv-1")
	require.NoError(t, err)
	require.Equal(t, "up", messages[1].Feedback)

	adminSummaries, adminTotal, err := repo.ListAdminConversations(ctx, "Nguyen", "up", "", 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, adminTotal)
	require.Equal(t, "Nguyen Van A", adminSummaries[0].UserFullName)
	require.Equal(t, 2, adminSummaries[0].MessageCount)
	require.Equal(t, 1, adminSummaries[0].FeedbackUpCount)

	totalChats, totalMessages, up, down, err := repo.GetAdminConversationStats(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, totalChats)
	require.Equal(t, 2, totalMessages)
	require.Equal(t, 1, up)
	require.Equal(t, 0, down)

	err = repo.DeleteConversation(ctx, "conv-1", "u1")
	require.NoError(t, err)

	_, total, err = repo.ListConversationsByUserID(ctx, "u1", 10, 0)
	require.NoError(t, err)
	require.Equal(t, 0, total)
}

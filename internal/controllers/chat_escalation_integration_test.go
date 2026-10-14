package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/crypto"
)

// TestChat_EscalationPersistenceAndSessionSwitch validates that can_escalate persists across session switching.
func TestChat_EscalationPersistenceAndSessionSwitch(t *testing.T) {
	mux, db := setupChatTestServer(t)
	defer db.Close()

	adminPass, _ := crypto.GenerateRandomHex(16)
	setupBody, _ := json.Marshal(map[string]string{
		"email":     "escalate_test@thanglong.edu.vn",
		"password":  adminPass,
		"full_name": "Escalate User",
	})
	setupReq := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewReader(setupBody))
	setupReq.Header.Set("Content-Type", "application/json")
	setupRec := httptest.NewRecorder()
	mux.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("Setup failed: %d, body: %s", setupRec.Code, setupRec.Body.String())
	}
	token, _ := authCookies(t, setupRec)

	userRepo := repositories.NewUserRepository(db, cache.NewTheineCache(10<<20))
	user, err := userRepo.GetByEmail(context.Background(), "escalate_test@thanglong.edu.vn")
	if err != nil {
		t.Fatalf("Failed to find created user: %v", err)
	}

	chatRepo := repositories.NewChatRepository(db, cache.NewTheineCache(10<<20))
	sessionSvc := services.NewChatSessionService(chatRepo, userRepo)
	ctx := context.Background()

	sessionID1, err := sessionSvc.RecordExchange(ctx, &request.RecordChatExchangeDto{
		UserID:      user.ID,
		UserQuery:   "tôi muốn biết làm sao để gửi yêu cầu tới giảng viên",
		BotReply:    "Nếu yêu cầu của bạn thuộc diện cần Cố vấn học tập / Phòng Đào tạo giải quyết trực tiếp, tôi sẵn sàng chuyển tiếp để Thầy/Cô phụ trách hỗ trợ bạn.",
		CanEscalate: true,
	})
	if err != nil {
		t.Fatalf("Failed to record exchange for session 1: %v", err)
	}

	sessionID2, err := sessionSvc.RecordExchange(ctx, &request.RecordChatExchangeDto{
		UserID:      user.ID,
		UserQuery:   "1+1 bằng mấy",
		BotReply:    "1 + 1 bằng 2.",
		CanEscalate: false,
	})
	if err != nil {
		t.Fatalf("Failed to record exchange for session 2: %v", err)
	}

	getReq1 := httptest.NewRequest(http.MethodGet, "/api/chat/sessions/"+sessionID1, nil)
	getReq1.Header.Set("Authorization", "Bearer "+token)
	getRec1 := httptest.NewRecorder()
	mux.ServeHTTP(getRec1, getReq1)
	if getRec1.Code != http.StatusOK {
		t.Fatalf("Get session 1 failed: %d, body: %s", getRec1.Code, getRec1.Body.String())
	}

	var detail1 struct {
		Status bool                                `json:"status"`
		Data   response.ConversationDetailResponse `json:"data"`
	}
	if err := json.Unmarshal(getRec1.Body.Bytes(), &detail1); err != nil {
		t.Fatalf("Failed to parse session 1 response: %v", err)
	}
	if len(detail1.Data.Messages) < 2 {
		t.Fatalf("Expected at least 2 messages in session 1, got %d", len(detail1.Data.Messages))
	}
	if !detail1.Data.Messages[1].CanEscalate {
		t.Fatalf("Expected session 1 assistant message to have CanEscalate=true")
	}

	getReq2 := httptest.NewRequest(http.MethodGet, "/api/chat/sessions/"+sessionID2, nil)
	getReq2.Header.Set("Authorization", "Bearer "+token)
	getRec2 := httptest.NewRecorder()
	mux.ServeHTTP(getRec2, getReq2)
	if getRec2.Code != http.StatusOK {
		t.Fatalf("Get session 2 failed: %d, body: %s", getRec2.Code, getRec2.Body.String())
	}

	var detail2 struct {
		Status bool                                `json:"status"`
		Data   response.ConversationDetailResponse `json:"data"`
	}
	if err := json.Unmarshal(getRec2.Body.Bytes(), &detail2); err != nil {
		t.Fatalf("Failed to parse session 2 response: %v", err)
	}
	if len(detail2.Data.Messages) < 2 {
		t.Fatalf("Expected at least 2 messages in session 2, got %d", len(detail2.Data.Messages))
	}
	if detail2.Data.Messages[1].CanEscalate {
		t.Fatalf("Expected session 2 assistant message to have CanEscalate=false")
	}

	switchBackReq := httptest.NewRequest(http.MethodGet, "/api/chat/sessions/"+sessionID1, nil)
	switchBackReq.Header.Set("Authorization", "Bearer "+token)
	switchBackRec := httptest.NewRecorder()
	mux.ServeHTTP(switchBackRec, switchBackReq)
	if switchBackRec.Code != http.StatusOK {
		t.Fatalf("Switch back to session 1 failed: %d", switchBackRec.Code)
	}

	var reloadedDetail1 struct {
		Status bool                                `json:"status"`
		Data   response.ConversationDetailResponse `json:"data"`
	}
	if err := json.Unmarshal(switchBackRec.Body.Bytes(), &reloadedDetail1); err != nil {
		t.Fatalf("Failed to parse reloaded session 1 response: %v", err)
	}
	if !reloadedDetail1.Data.Messages[1].CanEscalate {
		t.Fatalf("CanEscalate was lost after session switch; expected true, got false")
	}
}

// TestChat_HTTPChatEndpointRecordsCanEscalate verifies that the chat endpoint persists can_escalate to DB.
func TestChat_HTTPChatEndpointRecordsCanEscalate(t *testing.T) {
	mux, db := setupChatTestServer(t)
	defer db.Close()

	adminPass, _ := crypto.GenerateRandomHex(16)
	setupBody, _ := json.Marshal(map[string]string{
		"email":     "http_escalate@thanglong.edu.vn",
		"password":  adminPass,
		"full_name": "HTTP Escalate User",
	})
	setupReq := httptest.NewRequest(http.MethodPost, "/api/setup", bytes.NewReader(setupBody))
	setupReq.Header.Set("Content-Type", "application/json")
	setupRec := httptest.NewRecorder()
	mux.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("Setup failed: %d, body: %s", setupRec.Code, setupRec.Body.String())
	}
	token, _ := authCookies(t, setupRec)

	chatPayload, _ := json.Marshal(request.ChatRequest{
		Query: "tôi muốn biết làm sao để gửi yêu cầu tới giảng viên",
	})
	chatReq := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(chatPayload))
	chatReq.Header.Set("Content-Type", "application/json")
	chatReq.Header.Set("Authorization", "Bearer "+token)
	chatRec := httptest.NewRecorder()
	mux.ServeHTTP(chatRec, chatReq)
	if chatRec.Code != http.StatusOK {
		t.Fatalf("Chat failed: %d, body: %s", chatRec.Code, chatRec.Body.String())
	}

	var chatResp response.ChatResponse
	if err := json.Unmarshal(chatRec.Body.Bytes(), &chatResp); err != nil {
		t.Fatalf("Failed to parse chat response: %v", err)
	}
	if !chatResp.CanEscalate {
		t.Fatalf("Expected ChatResponse to have CanEscalate=true")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/chat/sessions/"+chatResp.SessionID, nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("Get session failed: %d, body: %s", getRec.Code, getRec.Body.String())
	}

	var detail struct {
		Status bool                                `json:"status"`
		Data   response.ConversationDetailResponse `json:"data"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("Failed to parse session detail: %v", err)
	}
	if len(detail.Data.Messages) < 2 {
		t.Fatalf("Expected at least 2 messages, got %d", len(detail.Data.Messages))
	}
	if !detail.Data.Messages[1].CanEscalate {
		t.Fatalf("Expected persisted assistant message to have CanEscalate=true, got false")
	}
}

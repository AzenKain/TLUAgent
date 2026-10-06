package supersede

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/pkg/convert"
	"tluagent-web/pkg/llm"
)

type SupersessionProposal struct {
	HasConflict      bool   `json:"has_conflict"`
	SupersededDocID  string `json:"superseded_doc_id"`
	Reason           string `json:"reason"`
	InquiryQuestion  string `json:"inquiry_question"`
	ProposedAction   string `json:"proposed_action"`
}

type Detector struct {
	q   *sqlc.Queries
	mgr *llm.Manager
}

func NewDetector(db sqlc.DBTX, mgr *llm.Manager) *Detector {
	return &Detector{
		q:   sqlc.New(db),
		mgr: mgr,
	}
}

func (d *Detector) AnalyzeAndNotify(ctx context.Context, newDocID, newDocTitle, newDocContent string) (*SupersessionProposal, error) {
	if d.mgr == nil || len(d.mgr.List()) == 0 {
		return nil, nil
	}

	docIDs, err := d.q.SearchDocumentIDs(ctx, sqlc.SearchDocumentIDsParams{
		Status: "ACTIVE",
		Limit:  100,
		Offset: 0,
	})
	if err != nil || len(docIDs) <= 1 {
		return nil, nil
	}

	activeDocs, err := d.q.GetDocumentsByIDs(ctx, docIDs)
	if err != nil || len(activeDocs) <= 1 {
		return nil, nil
	}

	var candidates []string
	for _, doc := range activeDocs {
		if doc.ID == newDocID {
			continue
		}
		if sharesDomainOrTopic(newDocTitle, doc.Title) {
			candidates = append(candidates, fmt.Sprintf("- ID: %s | Tiêu đề: %s | Ngày: %s", doc.ID, doc.Title, doc.PublishDate))
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	systemPrompt := `Bạn là Trợ lý Cố vấn Học vụ AI chuyên kiểm tra xung đột quy định và văn bản của Trường Đại học Thăng Long.
Nhiệm vụ: Phân tích xem văn bản mới có bãi bỏ, thay thế hoặc xung đột với các văn bản đang có hiệu lực hay không.
Nếu có văn bản cũ bị thay thế/hết hạn, trả về định dạng JSON DUY NHẤT:
{
  "has_conflict": true,
  "superseded_doc_id": "<ID văn bản cũ>",
  "reason": "<Lý do ngắn gọn nêu rõ điều khoản hoặc mốc thời gian bị thay thế>",
  "inquiry_question": "<Câu hỏi đề xuất gửi Giảng viên/Cố vấn kiểm tra>",
  "proposed_action": "EXPIRE_OLD_RULE"
}
Nếu không có xung đột hoặc là thông báo độc lập, trả về:
{"has_conflict": false}`

	userPrompt := fmt.Sprintf("Văn bản mới:\nID: %s\nTiêu đề: %s\nNội dung tóm tắt:\n%.1000s\n\nCác văn bản cùng chủ đề đang có hiệu lực:\n%s",
		newDocID, newDocTitle, newDocContent, strings.Join(candidates, "\n"))

	resp, err := d.mgr.Chat(ctx, &llm.ChatRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: systemPrompt},
			{Role: llm.RoleUser, Content: userPrompt},
		},
		JSONMode: true,
	})
	if err != nil || resp == nil {
		return nil, err
	}

	var proposal SupersessionProposal
	raw := strings.TrimSpace(resp.Message.Content)
	if err := json.Unmarshal([]byte(raw), &proposal); err != nil {
		return nil, nil
	}

	if !proposal.HasConflict || proposal.SupersededDocID == "" {
		return &proposal, nil
	}

	inquiryID := "inq-" + uuid.Must(uuid.NewV7()).String()[:8]
	now := time.Now()
	_, err = d.q.CreateInquiry(ctx, sqlc.CreateInquiryParams{
		ID:             inquiryID,
		UserID:         "system",
		ConversationID: sql.NullString{},
		StudentName:    "Agent Tự Động Rà Soát",
		StudentCode:    "AI_AGENT",
		Question:       proposal.InquiryQuestion,
		Context:        fmt.Sprintf("Văn bản mới: %s (%s). Lý do đề xuất thay thế: %s", newDocTitle, newDocID, proposal.Reason),
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		log.Warn().Err(err).Msg("failed to create supersession advisory inquiry")
	}

	advisorRole, err := d.q.GetRoleByName(ctx, "ADVISOR")
	var advisorIDs []string
	if err == nil {
		advisorIDs, _ = d.q.SearchUserIDs(ctx, sqlc.SearchUserIDsParams{
			RoleID: advisorRole.ID,
			Limit:  20,
			Offset: 0,
		})
	}
	if len(advisorIDs) == 0 {
		adminRole, err := d.q.GetRoleByName(ctx, "ADMIN")
		if err == nil {
			advisorIDs, _ = d.q.SearchUserIDs(ctx, sqlc.SearchUserIDsParams{
				RoleID: adminRole.ID,
				Limit:  20,
				Offset: 0,
			})
		}
	}
	for _, advID := range advisorIDs {
		_, _ = d.q.CreateNotification(ctx, sqlc.CreateNotificationParams{
			ID:        "notif-" + uuid.Must(uuid.NewV7()).String()[:8],
			UserID:    advID,
			InquiryID: convert.StringToNullString(inquiryID),
			Title:     "Xác nhận thay đổi quy định học vụ",
			Content:   fmt.Sprintf("Văn bản %s có thể thay thế văn bản %s. %s", newDocTitle, proposal.SupersededDocID, proposal.Reason),
			Type:      "SUPERSEDED_PROPOSED",
			CreatedAt: now,
		})
	}

	return &proposal, nil
}

func sharesDomainOrTopic(titleA, titleB string) bool {
	a := strings.ToLower(titleA)
	b := strings.ToLower(titleB)
	keywords := []string{"học phí", "nghỉ học", "bảo lưu", "chuẩn đầu ra", "quy chế", "đồ án", "thực tập", "học lại", "lịch thi"}
	for _, kw := range keywords {
		if strings.Contains(a, kw) && strings.Contains(b, kw) {
			return true
		}
	}
	return false
}

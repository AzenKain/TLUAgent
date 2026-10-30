package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"tluagent-web/internal/agent"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/guardrail"
	"tluagent-web/pkg/llm"
)

// MemoryService orchestrates student academic profiles and long-term conversation memories.
type MemoryService interface {
	GetStudentProfile(ctx context.Context, userID string) (*response.StudentProfileResponse, error)
	UpsertStudentProfile(ctx context.Context, profile *models.StudentProfile) (*response.StudentProfileResponse, error)
	UpdateStudentProfileFromDto(ctx context.Context, userID string, dto *request.UpsertStudentProfileDto) (*response.StudentProfileResponse, error)
	ListUserMemories(ctx context.Context, userID string) ([]*response.UserMemoryItemResponse, error)
	SaveUserMemory(ctx context.Context, memory *models.UserMemoryItem) (*response.UserMemoryItemResponse, error)
	CreateUserMemoryFromDto(ctx context.Context, userID string, dto *request.SaveUserMemoryDto) (*response.UserMemoryItemResponse, error)
	DeleteUserMemory(ctx context.Context, id, userID string) error
	ClearStudentMemoryAndProfile(ctx context.Context, userID string) error
	BuildLongTermMemoryPromptContext(ctx context.Context, userID string) (string, error)
	BuildAdminLongTermMemoryPromptContext(ctx context.Context, adminID string) (string, error)
	ExtractAndSaveMemories(ctx context.Context, userID, userQuery, botReply string) error
	ExtractAndSaveAdminMemories(ctx context.Context, adminID, adminQuery, botReply string) error
}

type memoryService struct {
	memoryRepo repositories.MemoryRepository
	userRepo   repositories.UserRepository
	llmManager *llm.Manager
}

// NewMemoryService creates a memory service instance.
func NewMemoryService(
	memoryRepo repositories.MemoryRepository,
	userRepo repositories.UserRepository,
	llmManager *llm.Manager,
) MemoryService {
	return &memoryService{
		memoryRepo: memoryRepo,
		userRepo:   userRepo,
		llmManager: llmManager,
	}
}

func (s *memoryService) GetStudentProfile(ctx context.Context, userID string) (*response.StudentProfileResponse, error) {
	if userID == "" || userID == "0" {
		return nil, nil
	}
	profile, err := s.memoryRepo.GetStudentProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil && s.userRepo != nil {
		user, uErr := s.userRepo.GetByID(ctx, userID)
		if uErr == nil && user != nil {
			newProfile := &models.StudentProfile{
				UserID:           userID,
				Major:            "",
				Cohort:           "",
				AcademicStanding: "NORMAL",
				CompletedCredits: 0,
				CumulativeGPA:    0.0,
				TargetGPA:        0.0,
				AdvisorNotes:     "",
				UpdatedAt:        time.Now().UTC(),
			}
			profile, _ = s.memoryRepo.UpsertStudentProfile(ctx, newProfile)
		}
	}
	return profile.ToResponse(), nil
}

func (s *memoryService) UpsertStudentProfile(ctx context.Context, profile *models.StudentProfile) (*response.StudentProfileResponse, error) {
	if profile == nil || profile.UserID == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "invalid student profile")
	}
	saved, err := s.memoryRepo.UpsertStudentProfile(ctx, profile)
	if err != nil {
		return nil, err
	}
	return saved.ToResponse(), nil
}

func (s *memoryService) UpdateStudentProfileFromDto(ctx context.Context, userID string, dto *request.UpsertStudentProfileDto) (*response.StudentProfileResponse, error) {
	if userID == "" || userID == "0" || dto == nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "invalid student profile")
	}
	profile := &models.StudentProfile{
		UserID:           userID,
		Major:            dto.Major,
		Cohort:           dto.Cohort,
		AcademicStanding: dto.AcademicStanding,
		CompletedCredits: dto.CompletedCredits,
		CumulativeGPA:    dto.CumulativeGPA,
		TargetGPA:        dto.TargetGPA,
		AdvisorNotes:     dto.AdvisorNotes,
		UpdatedAt:        time.Now().UTC(),
	}
	return s.UpsertStudentProfile(ctx, profile)
}

func (s *memoryService) ListUserMemories(ctx context.Context, userID string) ([]*response.UserMemoryItemResponse, error) {
	if userID == "" || userID == "0" {
		return []*response.UserMemoryItemResponse{}, nil
	}
	items, err := s.memoryRepo.ListUserMemories(ctx, userID)
	if err != nil {
		return nil, err
	}
	return models.UserMemoriesToResponse(items), nil
}

func (s *memoryService) SaveUserMemory(ctx context.Context, memory *models.UserMemoryItem) (*response.UserMemoryItemResponse, error) {
	if memory == nil || memory.UserID == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "invalid user memory item")
	}
	if chk := guardrail.CheckQuery(memory.MemoryKey); chk.Decision == guardrail.DecisionBlockedInjection {
		return nil, apperrors.New(apperrors.ErrBadRequest, "memory key contains forbidden instruction")
	}
	if chk := guardrail.CheckQuery(memory.MemoryValue); chk.Decision == guardrail.DecisionBlockedInjection {
		return nil, apperrors.New(apperrors.ErrBadRequest, "memory value contains forbidden instruction")
	}
	memory.MemoryValue = guardrail.SanitizeOutput(memory.MemoryValue)
	if memory.ID == "" {
		memory.ID = uuid.NewString()
	}
	if memory.MemoryKey == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "memory key cannot be empty")
	}
	if memory.Category == "" {
		memory.Category = "GENERAL"
	}
	if memory.Source == "" {
		memory.Source = "USER_DECLARED"
	}
	memory.IsActive = true
	saved, err := s.memoryRepo.UpsertUserMemory(ctx, memory)
	if err != nil {
		return nil, err
	}
	return saved.ToResponse(), nil
}

func (s *memoryService) CreateUserMemoryFromDto(ctx context.Context, userID string, dto *request.SaveUserMemoryDto) (*response.UserMemoryItemResponse, error) {
	if userID == "" || userID == "0" || dto == nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "invalid user memory item")
	}
	item := &models.UserMemoryItem{
		UserID:      userID,
		Category:    dto.Category,
		MemoryKey:   dto.MemoryKey,
		MemoryValue: dto.MemoryValue,
		Confidence:  dto.Confidence,
		Source:      dto.Source,
	}
	if item.Source == "" {
		item.Source = "USER_DECLARED"
	}
	return s.SaveUserMemory(ctx, item)
}

func (s *memoryService) DeleteUserMemory(ctx context.Context, id, userID string) error {
	if id == "" || userID == "" {
		return apperrors.New(apperrors.ErrBadRequest, "missing id or user id")
	}
	return s.memoryRepo.DeleteUserMemory(ctx, id, userID)
}

func (s *memoryService) ClearStudentMemoryAndProfile(ctx context.Context, userID string) error {
	if userID == "" || userID == "0" {
		return apperrors.New(apperrors.ErrBadRequest, "missing user id")
	}
	if err := s.memoryRepo.DeleteStudentProfile(ctx, userID); err != nil {
		return err
	}
	return s.memoryRepo.ClearAllUserMemories(ctx, userID)
}

func (s *memoryService) BuildLongTermMemoryPromptContext(ctx context.Context, userID string) (string, error) {
	if userID == "" || userID == "0" {
		return "", nil
	}

	profile, _ := s.GetStudentProfile(ctx, userID)
	memories, _ := s.ListUserMemories(ctx, userID)

	if profile == nil && len(memories) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("[STUDENT LONG-TERM ACADEMIC PROFILE & MEMORY]\n")
	if profile != nil {
		if profile.Major != "" {
			fmt.Fprintf(&sb, "- Major: %s\n", profile.Major)
		}
		if profile.Cohort != "" {
			fmt.Fprintf(&sb, "- Cohort: %s\n", profile.Cohort)
		}
		if profile.AcademicStanding != "" {
			fmt.Fprintf(&sb, "- Academic Standing: %s\n", profile.AcademicStanding)
		}
		if profile.CompletedCredits > 0 {
			fmt.Fprintf(&sb, "- Completed Credits: %d\n", profile.CompletedCredits)
		}
		if profile.CumulativeGPA > 0 {
			fmt.Fprintf(&sb, "- Cumulative GPA: %.2f\n", profile.CumulativeGPA)
		}
		if profile.TargetGPA > 0 {
			fmt.Fprintf(&sb, "- Target GPA: %.2f\n", profile.TargetGPA)
		}
		if profile.AdvisorNotes != "" {
			fmt.Fprintf(&sb, "- Advisor Notes: %s\n", profile.AdvisorNotes)
		}
	}

	if len(memories) > 0 {
		sb.WriteString("Retained Academic Memory Facts:\n")
		for _, m := range memories {
			safeVal := guardrail.SanitizeOutput(m.MemoryValue)
			fmt.Fprintf(&sb, "  * [%s] %s: %s\n", m.Category, m.MemoryKey, safeVal)
		}
	}

	return sb.String(), nil
}

type extractedFact struct {
	Category    string  `json:"category"`
	MemoryKey   string  `json:"memory_key"`
	MemoryValue string  `json:"memory_value"`
	Confidence  float64 `json:"confidence"`
}

func (s *memoryService) ExtractAndSaveMemories(ctx context.Context, userID, userQuery, botReply string) error {
	if userID == "" || userID == "0" || strings.TrimSpace(userQuery) == "" || s.llmManager == nil {
		return nil
	}

	prompt := agent.PromptStudentMemoryExtraction()

	req := &llm.ChatRequest{
		Messages: []llm.Message{
			llm.SystemMessage(prompt),
			llm.UserMessage(fmt.Sprintf("User: %s\nAdvisor: %s", userQuery, botReply)),
		},
		JSONMode:         true,
		ThinkingDisabled: true,
	}

	resp, err := s.llmManager.Chat(ctx, req)
	if err != nil || resp == nil {
		return nil
	}

	content := strings.TrimSpace(resp.Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var facts []extractedFact
	if err := json.Unmarshal([]byte(content), &facts); err != nil {
		return nil
	}

	for _, fact := range facts {
		if fact.MemoryKey == "" || fact.MemoryValue == "" {
			continue
		}
		if chk := guardrail.CheckQuery(fact.MemoryValue); chk.Decision == guardrail.DecisionBlockedInjection {
			continue
		}
		item := &models.UserMemoryItem{
			UserID:      userID,
			Category:    fact.Category,
			MemoryKey:   fact.MemoryKey,
			MemoryValue: guardrail.SanitizeOutput(fact.MemoryValue),
			Confidence:  fact.Confidence,
			Source:      "CHAT",
		}
		_, _ = s.SaveUserMemory(ctx, item)
	}

	return nil
}

// BuildAdminLongTermMemoryPromptContext builds the prompt context for an administrator's operational profile and persistent memory.
func (s *memoryService) BuildAdminLongTermMemoryPromptContext(ctx context.Context, adminID string) (string, error) {
	if adminID == "" || adminID == "0" {
		return "", nil
	}

	var adminName, adminEmail, adminRole string
	if s.userRepo != nil {
		user, err := s.userRepo.GetByID(ctx, adminID)
		if err == nil && user != nil {
			adminName = user.FullName
			adminEmail = user.Email
			if len(user.Roles) > 0 {
				var roleNames []string
				for _, r := range user.Roles {
					if r != nil && r.Name != "" {
						roleNames = append(roleNames, r.Name)
					}
				}
				adminRole = strings.Join(roleNames, ", ")
			}
		}
	}

	memories, _ := s.ListUserMemories(ctx, adminID)
	if adminName == "" && len(memories) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("[ADMINISTRATOR OPERATIONAL PROFILE & MEMORY]\n")
	if adminName != "" {
		fmt.Fprintf(&sb, "- Name: %s\n", adminName)
	}
	if adminEmail != "" {
		fmt.Fprintf(&sb, "- Email: %s\n", adminEmail)
	}
	if adminRole != "" {
		fmt.Fprintf(&sb, "- System Roles: %s\n", adminRole)
	}

	if len(memories) > 0 {
		sb.WriteString("Retained Operational Preferences & Policy Memories:\n")
		for _, m := range memories {
			safeVal := guardrail.SanitizeOutput(m.MemoryValue)
			fmt.Fprintf(&sb, "  * [%s] %s: %s\n", m.Category, m.MemoryKey, safeVal)
		}
	}

	return sb.String(), nil
}

// ExtractAndSaveAdminMemories extracts operational admin facts and preferences from administrative interactions.
func (s *memoryService) ExtractAndSaveAdminMemories(ctx context.Context, adminID, adminQuery, botReply string) error {
	if adminID == "" || adminID == "0" || strings.TrimSpace(adminQuery) == "" || s.llmManager == nil {
		return nil
	}

	prompt := agent.PromptAdminMemoryExtraction()

	req := &llm.ChatRequest{
		Messages: []llm.Message{
			llm.SystemMessage(prompt),
			llm.UserMessage(fmt.Sprintf("Administrator: %s\nTLUClaw Assistant: %s", adminQuery, botReply)),
		},
		JSONMode:         true,
		ThinkingDisabled: true,
	}

	resp, err := s.llmManager.Chat(ctx, req)
	if err != nil || resp == nil {
		return nil
	}

	content := strings.TrimSpace(resp.Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var facts []extractedFact
	if err := json.Unmarshal([]byte(content), &facts); err != nil {
		return nil
	}

	for _, fact := range facts {
		if fact.MemoryKey == "" || fact.MemoryValue == "" {
			continue
		}
		if chk := guardrail.CheckQuery(fact.MemoryValue); chk.Decision == guardrail.DecisionBlockedInjection {
			continue
		}
		item := &models.UserMemoryItem{
			UserID:      adminID,
			Category:    fact.Category,
			MemoryKey:   fact.MemoryKey,
			MemoryValue: guardrail.SanitizeOutput(fact.MemoryValue),
			Confidence:  fact.Confidence,
			Source:      "ADMIN_CHAT",
		}
		_, _ = s.SaveUserMemory(ctx, item)
	}

	return nil
}

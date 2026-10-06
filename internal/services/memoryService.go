package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/apperrors"
	"tluagent-web/pkg/guardrail"
	"tluagent-web/pkg/llm"
)

// MemoryService orchestrates student academic profiles and long-term conversation memories.
type MemoryService interface {
	SetLLMManager(llmManager *llm.Manager)
	GetStudentProfile(ctx context.Context, userID string) (*models.StudentProfile, error)
	UpsertStudentProfile(ctx context.Context, profile *models.StudentProfile) (*models.StudentProfile, error)
	ListUserMemories(ctx context.Context, userID string) ([]*models.UserMemoryItem, error)
	SaveUserMemory(ctx context.Context, memory *models.UserMemoryItem) (*models.UserMemoryItem, error)
	DeleteUserMemory(ctx context.Context, id, userID string) error
	ClearStudentMemoryAndProfile(ctx context.Context, userID string) error
	BuildLongTermMemoryPromptContext(ctx context.Context, userID string) (string, error)
	ExtractAndSaveMemories(ctx context.Context, userID, userQuery, botReply string) error
}

type memoryService struct {
	memoryRepo repositories.MemoryRepository
	userRepo   repositories.UserRepository
	llmManager *llm.Manager
}

// NewMemoryService creates a memory service instance.
func NewMemoryService(memoryRepo repositories.MemoryRepository, userRepo repositories.UserRepository) MemoryService {
	return &memoryService{
		memoryRepo: memoryRepo,
		userRepo:   userRepo,
	}
}

func (s *memoryService) SetLLMManager(llmManager *llm.Manager) {
	s.llmManager = llmManager
}

func (s *memoryService) GetStudentProfile(ctx context.Context, userID string) (*models.StudentProfile, error) {
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
			cohort := ""
			if len(user.StudentCode) >= 3 && strings.HasPrefix(strings.ToUpper(user.StudentCode), "A") {
				cohort = "K" + user.StudentCode[1:3]
			}
			newProfile := &models.StudentProfile{
				UserID:           userID,
				Major:            "",
				Cohort:           cohort,
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
	return profile, nil
}

func (s *memoryService) UpsertStudentProfile(ctx context.Context, profile *models.StudentProfile) (*models.StudentProfile, error) {
	if profile == nil || profile.UserID == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "invalid student profile")
	}
	return s.memoryRepo.UpsertStudentProfile(ctx, profile)
}

func (s *memoryService) ListUserMemories(ctx context.Context, userID string) ([]*models.UserMemoryItem, error) {
	if userID == "" || userID == "0" {
		return []*models.UserMemoryItem{}, nil
	}
	return s.memoryRepo.ListUserMemories(ctx, userID)
}

func (s *memoryService) SaveUserMemory(ctx context.Context, memory *models.UserMemoryItem) (*models.UserMemoryItem, error) {
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
	return s.memoryRepo.UpsertUserMemory(ctx, memory)
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

	provider, err := s.llmManager.Default()
	if err != nil || provider == nil {
		return nil
	}

	prompt := "You are an academic advisor memory extractor for university students. Extract concise long-term academic facts about the student from the conversation, such as: major, cohort, academic goals, target GPA, language certification plans, academic difficulties, course retake concerns. Return a JSON array of objects with keys: 'category' (e.g. 'ACADEMIC_PROFILE', 'ACADEMIC_GOAL', 'LANGUAGE_GOAL', 'ACADEMIC_DIFFICULTY'), 'memory_key', 'memory_value', 'confidence' (number between 0.0 and 1.0). If no long-term facts exist in the input, return an empty JSON array []. Do not include explanation, respond ONLY with valid JSON array."

	req := &llm.ChatRequest{
		Messages: []llm.Message{
			llm.SystemMessage(prompt),
			llm.UserMessage(fmt.Sprintf("User: %s\nAdvisor: %s", userQuery, botReply)),
		},
		JSONMode: true,
	}

	resp, err := provider.Chat(ctx, req)
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


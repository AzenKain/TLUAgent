package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"tluagent-web/internal/agent"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/llm"
)

// TranscriptService defines business operations for student academic transcript management, OCR diff, and batch persistence.
type TranscriptService interface {
	GetTranscriptSummary(ctx context.Context, userID string) (*response.TranscriptSummaryResponse, error)
	AnalyzeTranscriptDiff(ctx context.Context, userID string, incomingGrades []*models.StudentCourseGradeRecord) (*response.TranscriptDiffResultResponse, error)
	BatchSaveTranscript(ctx context.Context, userID string, incomingGrades []*models.StudentCourseGradeRecord) (*response.TranscriptSummaryResponse, error)
	ExtractGradesFromImages(ctx context.Context, base64Images []string) ([]*response.StudentCourseGradeItem, error)
	ExtractAndAnalyzeImages(ctx context.Context, userID string, base64Images []string) (*response.ExtractTranscriptImagesResponse, error)
	BatchSaveTranscriptGrades(ctx context.Context, userID string, inputs []request.StudentCourseGradeInput) (*response.BatchSaveTranscriptResponse, error)
}

type transcriptService struct {
	transcriptRepo repositories.TranscriptRepository
	memoryRepo     repositories.MemoryRepository
	llmManager     *llm.Manager
}

// NewTranscriptService creates a new transcript service instance.
func NewTranscriptService(
	transcriptRepo repositories.TranscriptRepository,
	memoryRepo repositories.MemoryRepository,
	llmManager *llm.Manager,
) TranscriptService {
	return &transcriptService{
		transcriptRepo: transcriptRepo,
		memoryRepo:     memoryRepo,
		llmManager:     llmManager,
	}
}

func (s *transcriptService) GetTranscriptSummary(ctx context.Context, userID string) (*response.TranscriptSummaryResponse, error) {
	grades, err := s.transcriptRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list transcript grades: %w", err)
	}

	var completedCredits float64
	var attemptedCredits float64
	var weightedScore10 float64
	var weightedScore4 float64
	var gpaCredits float64
	passedCount := 0
	failedCount := 0

	for _, g := range grades {
		attemptedCredits += g.Credits
		if g.IsPassed {
			completedCredits += g.Credits
			passedCount++
			if g.Credits > 0 {
				weightedScore10 += g.Credits * g.Score10
				weightedScore4 += g.Credits * g.Score4
				gpaCredits += g.Credits
			}
		} else {
			failedCount++
		}
	}

	gpa10 := 0.0
	gpa4 := 0.0
	if gpaCredits > 0 {
		gpa10 = weightedScore10 / gpaCredits
		gpa4 = weightedScore4 / gpaCredits
	}

	return (&models.TranscriptSummary{
		TotalCreditsCompleted: completedCredits,
		TotalCreditsAttempted: attemptedCredits,
		CumulativeGPA10:       gpa10,
		CumulativeGPA4:        gpa4,
		TotalCourses:          len(grades),
		PassedCoursesCount:    passedCount,
		FailedCoursesCount:    failedCount,
		Grades:                grades,
	}).ToResponse(), nil
}

func (s *transcriptService) AnalyzeTranscriptDiff(ctx context.Context, userID string, incomingGrades []*models.StudentCourseGradeRecord) (*response.TranscriptDiffResultResponse, error) {
	existingGrades, err := s.transcriptRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list existing grades: %w", err)
	}

	existingMap := make(map[string]*models.StudentCourseGradeRecord, len(existingGrades))
	for _, eg := range existingGrades {
		existingMap[strings.ToUpper(strings.TrimSpace(eg.CourseCode))] = eg
	}

	var items []*models.TranscriptDiffItem
	newCount := 0
	upgradedCount := 0
	unchangedCount := 0

	var upgradedDesc []string
	var newDesc []string

	for _, in := range incomingGrades {
		code := strings.ToUpper(strings.TrimSpace(in.CourseCode))
		if code == "" {
			continue
		}

		existing, exists := existingMap[code]
		if !exists {
			item := &models.TranscriptDiffItem{
				CourseCode:     code,
				CourseName:     in.CourseName,
				Credits:        in.Credits,
				OldScore10:     0.0,
				NewScore10:     in.Score10,
				OldGradeLetter: "",
				NewGradeLetter: in.GradeLetter,
				OldIsPassed:    false,
				NewIsPassed:    in.IsPassed,
				ChangeType:     "NEW",
			}
			items = append(items, item)
			newCount++
			newDesc = append(newDesc, fmt.Sprintf("%s (%s): %.1f credits, Score %.1f (%s)", code, in.CourseName, in.Credits, in.Score10, in.GradeLetter))
		} else {
			isUpgraded := (!existing.IsPassed && in.IsPassed) || (in.Score10 > existing.Score10)
			isSame := existing.Score10 == in.Score10 && existing.IsPassed == in.IsPassed

			changeType := "UNCHANGED"
			if isUpgraded {
				changeType = "UPGRADED"
				upgradedCount++
				upgradedDesc = append(upgradedDesc, fmt.Sprintf("%s (%s): Old %.1f (%s) -> New %.1f (%s)", code, in.CourseName, existing.Score10, existing.GradeLetter, in.Score10, in.GradeLetter))
			} else if isSame {
				unchangedCount++
			} else {
				changeType = "LOWER"
			}

			items = append(items, &models.TranscriptDiffItem{
				CourseCode:     code,
				CourseName:     in.CourseName,
				Credits:        in.Credits,
				OldScore10:     existing.Score10,
				NewScore10:     in.Score10,
				OldGradeLetter: existing.GradeLetter,
				NewGradeLetter: in.GradeLetter,
				OldIsPassed:    existing.IsPassed,
				NewIsPassed:    in.IsPassed,
				ChangeType:     changeType,
			})
		}
	}

	hasChanges := newCount > 0 || upgradedCount > 0

	var sb strings.Builder
	if hasChanges {
		sb.WriteString("Transcript update detected from submitted document:\n\n")
		if len(upgradedDesc) > 0 {
			sb.WriteString("Improved / Upgraded courses:\n")
			for _, desc := range upgradedDesc {
				sb.WriteString(fmt.Sprintf("- %s\n", desc))
			}
			sb.WriteString("\n")
		}
		if len(newDesc) > 0 {
			sb.WriteString("Newly added courses:\n")
			for _, desc := range newDesc {
				sb.WriteString(fmt.Sprintf("- %s\n", desc))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("Differences detected compared to current records. Do you wish to confirm and overwrite?")
	} else {
		sb.WriteString("Transcript matches existing records. No updates detected.")
	}

	return (&models.TranscriptDiffResult{
		HasChanges:            hasChanges,
		NewCoursesCount:       newCount,
		UpgradedCoursesCount:  upgradedCount,
		UnchangedCoursesCount: unchangedCount,
		Items:                 items,
		ConfirmationPrompt:    sb.String(),
	}).ToResponse(), nil
}

func (s *transcriptService) BatchSaveTranscript(ctx context.Context, userID string, incomingGrades []*models.StudentCourseGradeRecord) (*response.TranscriptSummaryResponse, error) {
	if len(incomingGrades) == 0 {
		return s.GetTranscriptSummary(ctx, userID)
	}

	existingGrades, _ := s.transcriptRepo.ListByUserID(ctx, userID)
	existingIDMap := make(map[string]string, len(existingGrades))
	for _, eg := range existingGrades {
		existingIDMap[strings.ToUpper(strings.TrimSpace(eg.CourseCode))] = eg.ID
	}

	params := make([]sqlc.UpsertTranscriptGradeParams, 0, len(incomingGrades))
	for _, g := range incomingGrades {
		code := strings.ToUpper(strings.TrimSpace(g.CourseCode))
		if code == "" {
			continue
		}

		id := g.ID
		if id == "" {
			if existingID, exists := existingIDMap[code]; exists {
				id = existingID
			} else {
				id = "tg_" + uuid.New().String()
			}
		}

		isPassedInt := int64(0)
		if g.IsPassed {
			isPassedInt = 1
		}
		isRetakeInt := int64(0)
		if g.IsRetake {
			isRetakeInt = 1
		}

		params = append(params, sqlc.UpsertTranscriptGradeParams{
			ID:          id,
			UserID:      userID,
			CourseCode:  code,
			CourseName:  strings.TrimSpace(g.CourseName),
			Credits:     g.Credits,
			Score10:     g.Score10,
			Score4:      g.Score4,
			GradeLetter: strings.ToUpper(strings.TrimSpace(g.GradeLetter)),
			IsPassed:    isPassedInt,
			IsRetake:    isRetakeInt,
			TermTaken:   strings.TrimSpace(g.TermTaken),
		})
	}

	_, err := s.transcriptRepo.BatchUpsert(ctx, userID, params)
	if err != nil {
		return nil, fmt.Errorf("batch upsert transcript grades: %w", err)
	}

	summary, err := s.GetTranscriptSummary(ctx, userID)
	if err != nil {
		return nil, err
	}

	if s.memoryRepo != nil && userID != "" && userID != "0" {
		if prof, pErr := s.memoryRepo.GetStudentProfile(ctx, userID); pErr == nil && prof != nil {
			prof.CompletedCredits = int(summary.TotalCreditsCompleted)
			prof.CumulativeGPA = summary.CumulativeGPA10
			_, _ = s.memoryRepo.UpsertStudentProfile(ctx, prof)
		}
	}

	return summary, nil
}

func (s *transcriptService) ExtractGradesFromImages(ctx context.Context, base64Images []string) ([]*response.StudentCourseGradeItem, error) {
	records, err := s.extractGradeRecords(ctx, base64Images)
	if err != nil {
		return nil, err
	}
	out := make([]*response.StudentCourseGradeItem, 0, len(records))
	for _, r := range records {
		out = append(out, r.ToResponse())
	}
	return out, nil
}

func (s *transcriptService) extractGradeRecords(ctx context.Context, base64Images []string) ([]*models.StudentCourseGradeRecord, error) {
	if len(base64Images) == 0 {
		return []*models.StudentCourseGradeRecord{}, nil
	}
	if s.llmManager == nil {
		return nil, fmt.Errorf("llm manager is not initialized")
	}

	provider, err := s.llmManager.Default()
	if err != nil {
		return nil, fmt.Errorf("resolve default llm provider: %w", err)
	}

	parts := make([]llm.ContentPart, 0, len(base64Images)+1)
	parts = append(parts, llm.ContentPart{
		Type: llm.ContentPartText,
		Text: "Please extract all course grade rows from these transcript images.",
	})
	for _, img := range base64Images {
		parts = append(parts, llm.ContentPart{
			Type: llm.ContentPartImage,
			Image: &llm.ImageSource{
				URL: img,
			},
		})
	}

	systemInstruction := agent.PromptVisionTranscriptExtraction()

	temp := 0.1
	req := &llm.ChatRequest{
		Model: provider.DefaultModel(),
		Messages: []llm.Message{
			{
				Role:    llm.RoleSystem,
				Content: systemInstruction,
			},
			{
				Role:  llm.RoleUser,
				Parts: parts,
			},
		},
		Temperature: &temp,
	}

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("llm vision transcript extraction failed: %w", err)
	}

	content := strings.TrimSpace(resp.Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var extracted []*models.StudentCourseGradeRecord
	if err := json.Unmarshal([]byte(content), &extracted); err != nil {
		return nil, fmt.Errorf("unmarshal extracted transcript json: %w (content: %s)", err, content)
	}

	return extracted, nil
}

func (s *transcriptService) ExtractAndAnalyzeImages(ctx context.Context, userID string, base64Images []string) (*response.ExtractTranscriptImagesResponse, error) {
	extracted, err := s.extractGradeRecords(ctx, base64Images)
	if err != nil {
		return nil, err
	}

	diff, err := s.AnalyzeTranscriptDiff(ctx, userID, extracted)
	if err != nil {
		return nil, err
	}

	extractedItems := make([]*response.StudentCourseGradeItem, len(extracted))
	for i, item := range extracted {
		extractedItems[i] = item.ToResponse()
	}

	return &response.ExtractTranscriptImagesResponse{
		Extracted: extractedItems,
		Diff:      diff,
	}, nil
}

func (s *transcriptService) BatchSaveTranscriptGrades(ctx context.Context, userID string, inputs []request.StudentCourseGradeInput) (*response.BatchSaveTranscriptResponse, error) {
	grades := make([]*models.StudentCourseGradeRecord, len(inputs))
	for i, g := range inputs {
		grades[i] = &models.StudentCourseGradeRecord{
			ID:          g.ID,
			CourseCode:  g.CourseCode,
			CourseName:  g.CourseName,
			Credits:     g.Credits,
			Score10:     g.Score10,
			Score4:      g.Score4,
			GradeLetter: g.GradeLetter,
			IsPassed:    g.IsPassed,
			IsRetake:    g.IsRetake,
			TermTaken:   g.TermTaken,
		}
	}

	summary, err := s.BatchSaveTranscript(ctx, userID, grades)
	if err != nil {
		return nil, err
	}

	return &response.BatchSaveTranscriptResponse{
		Summary: summary,
		Message: "Transcript saved successfully.",
	}, nil
}

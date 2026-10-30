package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"tluagent-web/internal/agent"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/crypto"
	"tluagent-web/pkg/llm"
	"tluagent-web/pkg/tuition"
)

// TLUClawToolEngine coordinates execution of administrative operations and two-phase dry runs.
type TLUClawToolEngine interface {
	GetToolDefinitions() []llm.ToolDefinition
	ExecuteTool(ctx context.Context, sessionID, adminID string, call llm.ToolCall) (string, *response.TLUClawPreviewCardResponse, error)
}

type tluclawToolEngine struct {
	curriculumRepo   repositories.CurriculumRepository
	ragRepo          repositories.RAGRepository
	ragService       RAGService
	tluclawRepo      repositories.TLUClawRepository
	skillLoader      agent.SkillLoader
	memoryService    MemoryService
	userRepo         repositories.UserRepository
	roleRepo         repositories.RoleRepository
	transcriptRepo   repositories.TranscriptRepository
	inquiryRepo      repositories.InquiryRepository
	llmConfigSvc     LLMConfigService
	memoryRepo       repositories.MemoryRepository
	notifRepo        repositories.NotificationRepository
	jobService       JobService
	tuitionClient    *tuition.Client
	roleService      RoleService
	c                cache.Cache
	llmManager       *llm.Manager
	dissectionEngine *RegulatoryDissectionEngine
}

// NewTLUClawToolEngine creates an engine instance for TLUClaw administrative tools.
func NewTLUClawToolEngine(params TLUClawParams) TLUClawToolEngine {
	tuitionClient := params.TuitionClient
	if tuitionClient == nil {
		tuitionClient = tuition.NewClient(10 * time.Second)
	}
	disEngine := params.DissectionEngine
	if disEngine == nil {
		disEngine = NewRegulatoryDissectionEngine(EngineParams{
			RAGRepo:        params.RAGRepo,
			CurriculumRepo: params.CurriculumRepo,
			InquiryRepo:    params.InquiryRepo,
			NotifRepo:      params.NotifRepo,
			UserRepo:       params.UserRepo,
			LLMManager:     params.LLMManager,
		})
	}
	return &tluclawToolEngine{
		curriculumRepo:   params.CurriculumRepo,
		ragRepo:          params.RAGRepo,
		ragService:       params.RAGService,
		tluclawRepo:      params.TLUClawRepo,
		skillLoader:      params.SkillLoader,
		memoryService:    params.MemoryService,
		userRepo:         params.UserRepo,
		roleRepo:         params.RoleRepo,
		transcriptRepo:   params.TranscriptRepo,
		inquiryRepo:      params.InquiryRepo,
		llmConfigSvc:     params.LLMConfigSvc,
		memoryRepo:       params.MemoryRepo,
		notifRepo:        params.NotifRepo,
		jobService:       params.JobService,
		tuitionClient:    tuitionClient,
		roleService:      params.RoleService,
		c:                params.RAMCache,
		llmManager:       params.LLMManager,
		dissectionEngine: disEngine,
	}
}

// GetToolDefinitions resolves schema specifications for administrative tool sets from the agent skill system.
func (e *tluclawToolEngine) GetToolDefinitions() []llm.ToolDefinition {
	if e.skillLoader != nil {
		if skills, err := e.skillLoader.GetActiveSkills(context.Background()); err == nil && len(skills) > 0 {
			var results []llm.ToolDefinition
			seen := make(map[string]bool)
			for _, skill := range skills {
				rawDef := strings.TrimSpace(skill.ToolDefinition)
				if rawDef == "" {
					continue
				}
				defs, err := agent.ParseToolDefinitions(rawDef)
				if err != nil {
					continue
				}
				for _, d := range defs {
					if strings.HasPrefix(d.Function.Name, "tluclaw_") && !seen[d.Function.Name] {
						seen[d.Function.Name] = true
						results = append(results, d)
					}
				}
			}
			if len(results) > 0 {
				return results
			}
		}
	}

	return agent.GetDefaultTLUClawToolDefinitions()
}

// ExecuteTool routes a tool call to its dedicated handler and generates staged preview cards when appropriate.
func (e *tluclawToolEngine) ExecuteTool(ctx context.Context, sessionID, adminID string, call llm.ToolCall) (string, *response.TLUClawPreviewCardResponse, error) {
	switch call.Function.Name {
	case "tluclaw_curriculum_operations":
		var args request.TLUClawCurriculumOpArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Lỗi phân tích cú pháp tham số công cụ: %v", err), nil, nil
		}
		return e.executeCurriculumOperations(ctx, sessionID, adminID, &args)
	case "tluclaw_substitution_operations":
		var args request.TLUClawSubstitutionOpArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Lỗi phân tích cú pháp tham số công cụ: %v", err), nil, nil
		}
		return e.executeSubstitutionOperations(ctx, sessionID, adminID, &args)
	case "tluclaw_student_academic_operations":
		var args request.TLUClawStudentAcademicOpArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Lỗi phân tích cú pháp tham số công cụ: %v", err), nil, nil
		}
		return e.executeStudentAcademicOperations(ctx, sessionID, adminID, &args)
	case "tluclaw_user_operations":
		var args request.TLUClawUserOpArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Lỗi phân tích cú pháp tham số công cụ: %v", err), nil, nil
		}
		return e.executeUserOperations(ctx, sessionID, adminID, &args)
	case "tluclaw_inquiry_operations":
		var args request.TLUClawInquiryOpArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Lỗi phân tích cú pháp tham số công cụ: %v", err), nil, nil
		}
		return e.executeInquiryOperations(ctx, sessionID, adminID, &args)
	case "tluclaw_system_operations":
		var args request.TLUClawSystemOpArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Lỗi phân tích cú pháp tham số công cụ: %v", err), nil, nil
		}
		return e.executeSystemOperations(ctx, sessionID, adminID, &args)
	case "tluclaw_document_operations":
		var args request.TLUClawDocumentOpArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Lỗi phân tích cú pháp tham số công cụ: %v", err), nil, nil
		}
		return e.executeDocumentOperations(ctx, sessionID, adminID, &args)
	case "tluclaw_audit_operations":
		var args request.TLUClawAuditOpArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Lỗi phân tích cú pháp tham số công cụ: %v", err), nil, nil
		}
		return e.executeAuditOperations(ctx, sessionID, adminID, &args)
	case "tluclaw_admin_memory_operations":
		var args request.TLUClawMemoryOpArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Lỗi phân tích cú pháp tham số công cụ: %v", err), nil, nil
		}
		return e.executeMemoryOperations(ctx, adminID, &args)
	case "tluclaw_notification_operations":
		var args request.TLUClawNotificationOpArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Lỗi phân tích cú pháp tham số công cụ: %v", err), nil, nil
		}
		return e.executeNotificationOperations(ctx, sessionID, adminID, &args)
	case "tluclaw_job_operations":
		var args request.TLUClawJobOpArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("Lỗi phân tích cú pháp tham số công cụ: %v", err), nil, nil
		}
		return e.executeJobOperations(ctx, sessionID, adminID, &args)
	default:
		return fmt.Sprintf("Công cụ không tồn tại: %s", call.Function.Name), nil, nil
	}
}

func (e *tluclawToolEngine) executeCurriculumOperations(ctx context.Context, sessionID, adminID string, args *request.TLUClawCurriculumOpArgs) (string, *response.TLUClawPreviewCardResponse, error) {
	action := args.Action

	switch action {
	case "inspect_structure", "list_structure", "inspect_cohorts", "inspect_majors":
		cohorts, _ := e.curriculumRepo.ListCohorts(ctx)
		majors, _ := e.curriculumRepo.ListAllMajorMetadata(ctx)
		var sb strings.Builder
		sb.WriteString("# HIỆN TRẠNG CẤU TRÚC HỌC VỤ TLU TRONG CƠ SỞ DỮ LIỆU:\n")
		fmt.Fprintf(&sb, "- Danh sách các khóa học (%d khóa): %s\n", len(cohorts), strings.Join(cohorts, ", "))
		fmt.Fprintf(&sb, "- Danh sách ngành học & Khoa quản lý (%d bản ghi):\n", len(majors))
		for _, m := range majors {
			aliasesStr := ""
			if m.Aliases != "" {
				aliasesStr = fmt.Sprintf(" (Viết tắt/Alias: %s)", m.Aliases)
			}
			fmt.Fprintf(&sb, "  * Mã ngành: %s | Tên: %s | Khoa: %s%s\n", m.MajorCode, m.MajorName, m.FacultyCode, aliasesStr)
		}
		return sb.String(), nil, nil

	case "inspect_curriculum", "inspect_curriculum_update", "inspect_courses", "list_curriculum":
		cohort := strings.TrimSpace(args.Cohort)
		majorCode := strings.TrimSpace(args.MajorCode)
		majorName := strings.TrimSpace(args.MajorName)
		if majorName == "" {
			majorName = strings.TrimSpace(args.MajorNameVi)
		}
		if cohort == "" {
			return "Cần chỉ định tham số 'cohort' để tra cứu chương trình đào tạo hiện tại.", nil, nil
		}

		targetMajor := majorName
		if targetMajor == "" {
			targetMajor = majorCode
		}

		if currs, err := e.curriculumRepo.ListCurriculumsByCohort(ctx, cohort); err == nil && len(currs) > 0 {
			for _, c := range currs {
				if (targetMajor != "" && strings.EqualFold(c.MajorCode, targetMajor)) ||
					(targetMajor != "" && strings.EqualFold(c.MajorName, targetMajor)) ||
					(targetMajor != "" && strings.Contains(strings.ToUpper(c.MajorName), strings.ToUpper(targetMajor))) ||
					(targetMajor != "" && strings.Contains(strings.ToUpper(c.Aliases), strings.ToUpper(targetMajor))) {
					targetMajor = c.MajorName
					break
				}
			}
		}

		courses, err := e.curriculumRepo.GetCurriculumCoursesByCohortAndMajor(ctx, cohort, targetMajor)
		if err != nil || len(courses) == 0 {
			if fallbackCourses, sErr := e.curriculumRepo.SearchCurriculumCourses(ctx, cohort, targetMajor, ""); sErr == nil && len(fallbackCourses) > 0 {
				courses = fallbackCourses
			}
		}

		if len(courses) == 0 {
			return fmt.Sprintf("Chưa có môn học nào trong cơ sở dữ liệu cho khóa %s (ngành: %s).", cohort, targetMajor), nil, nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "# CHƯƠNG TRÌNH ĐÀO TẠO HIỆN HÀNH: Khóa %s | Ngành %s (Tổng %d môn):\n", cohort, targetMajor, len(courses))
		for i, c := range courses {
			if i >= 40 {
				fmt.Fprintf(&sb, "... và %d môn học khác.\n", len(courses)-40)
				break
			}
			fmt.Fprintf(&sb, "- [%s] %s: %.1f tín chỉ | Tiên quyết: %s\n", c.CourseCode, c.CourseName, c.Credits, c.PrerequisitesText)
		}
		return sb.String(), nil, nil

	case "propose_cohort", "propose_academic_structure", "apply_academic_structure":
		cohort := strings.TrimSpace(args.Cohort)
		academicYear := strings.TrimSpace(args.AcademicYear)
		if cohort == "" {
			return "Thiếu tham số bắt buộc 'cohort'", nil, nil
		}
		if academicYear == "" {
			academicYear = fmt.Sprintf("%s-AcademicYear", cohort)
		}

		existingCohorts, _ := e.curriculumRepo.ListCohorts(ctx)
		isUpdate := false
		for _, c := range existingCohorts {
			if strings.EqualFold(c, cohort) {
				isUpdate = true
				break
			}
		}

		diffAction := "CREATE"
		if isUpdate {
			diffAction = "UPDATE"
		}

		diffItems := []models.TLUClawDiffItem{
			{
				EntityType: "COHORT",
				Action:     diffAction,
				Key:        cohort,
				Name:       fmt.Sprintf("Khóa đào tạo %s", cohort),
				OldValue:   "",
				NewValue:   academicYear,
				Note:       fmt.Sprintf("Niên khóa áp dụng: %s", academicYear),
			},
		}

		blastRadius := models.TLUClawBlastRadius{
			AffectedCohorts: []string{cohort},
			ImpactSummary:   fmt.Sprintf("Thiết lập cấu trúc khóa %s niên khóa %s", cohort, academicYear),
			RiskLevel:       "LOW",
		}

		batchID := "batch_" + uuid.NewString()[:8]
		nonce, err := crypto.GenerateRandomHex(24)
		if err != nil {
			nonce = uuid.NewString()
		}
		expiresAt := time.Now().Add(15 * time.Minute)

		diffJSON, _ := json.Marshal(diffItems)
		blastJSON, _ := json.Marshal(blastRadius)

		statePayload, _ := json.Marshal(request.TLUClawAcademicStructureSnapshotState{
			Cohort:       cohort,
			AcademicYear: academicYear,
			IsUpdate:     isUpdate,
		})

		snapshotID := "snp_" + uuid.NewString()[:8]
		snapshotParams := sqlc.CreateTLUClawSnapshotParams{
			ID:                snapshotID,
			BatchID:           batchID,
			SessionID:         sessionID,
			OperatorID:        adminID,
			ActionType:        "academic_structure",
			Summary:           fmt.Sprintf("Đề xuất %s Khóa %s (%s)", diffAction, cohort, academicYear),
			DiffPreviewJson:   string(diffJSON),
			BlastRadiusJson:   string(blastJSON),
			PreviousStateJson: "{}",
			NewStateJson:      string(statePayload),
			Status:            "STAGED",
			ConfirmationNonce: nonce,
			NonceExpiresAt:    sql.NullTime{Time: expiresAt, Valid: true},
			AppliedAt:         sql.NullTime{Valid: false},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if _, err := e.tluclawRepo.CreateSnapshot(ctx, snapshotParams); err != nil {
			log.Warn().Err(err).Msg("tluclaw ExecuteTool: create snapshot failed")
			return "Không thể tạo bản ghi xem trước. Vui lòng thử lại sau.", nil, nil
		}

		card := &models.TLUClawPreviewCardData{
			SnapshotID:  snapshotID,
			BatchID:     batchID,
			ActionType:  "academic_structure",
			Summary:     snapshotParams.Summary,
			Nonce:       nonce,
			ExpiresAt:   expiresAt,
			DiffItems:   diffItems,
			BlastRadius: blastRadius,
		}

		msg := fmt.Sprintf("Đã tạo bản xem trước thay đổi cho Khóa %s. Vui lòng kiểm tra thẻ xem trước và xác nhận áp dụng.", cohort)
		return msg, card.ToResponse(), nil

	case "propose_major":
		majorCode := strings.TrimSpace(args.MajorCode)
		majorNameVi := strings.TrimSpace(args.MajorNameVi)
		facultyCode := strings.TrimSpace(args.FacultyCode)
		if majorCode == "" || majorNameVi == "" {
			return "Thiếu tham số bắt buộc major_code hoặc major_name_vi", nil, nil
		}

		aliasStrs := args.Aliases

		diffItems := []models.TLUClawDiffItem{
			{
				EntityType: "MAJOR",
				Action:     "UPSERT",
				Key:        majorCode,
				Name:       majorNameVi,
				OldValue:   "",
				NewValue:   fmt.Sprintf("%s (%s)", majorNameVi, facultyCode),
				Note:       fmt.Sprintf("Khoa: %s | Tên viết tắt: %s", facultyCode, strings.Join(aliasStrs, ", ")),
			},
		}

		blastRadius := models.TLUClawBlastRadius{
			AffectedMajors: []string{majorCode},
			ImpactSummary:  fmt.Sprintf("Thiết lập thông tin ngành %s (%s) thuộc Khoa %s", majorNameVi, majorCode, facultyCode),
			RiskLevel:      "LOW",
		}

		batchID := "batch_" + uuid.NewString()[:8]
		nonce, _ := crypto.GenerateRandomHex(24)
		expiresAt := time.Now().Add(15 * time.Minute)

		diffJSON, _ := json.Marshal(diffItems)
		blastJSON, _ := json.Marshal(blastRadius)

		statePayload, _ := json.Marshal(request.TLUClawAcademicStructureSnapshotState{
			MajorCode:   majorCode,
			MajorNameVi: majorNameVi,
			FacultyCode: facultyCode,
			Aliases:     aliasStrs,
		})

		snapshotID := "snp_" + uuid.NewString()[:8]
		snapshotParams := sqlc.CreateTLUClawSnapshotParams{
			ID:                snapshotID,
			BatchID:           batchID,
			SessionID:         sessionID,
			OperatorID:        adminID,
			ActionType:        "academic_structure",
			Summary:           fmt.Sprintf("Đề xuất cập nhật Ngành %s (%s)", majorNameVi, majorCode),
			DiffPreviewJson:   string(diffJSON),
			BlastRadiusJson:   string(blastJSON),
			PreviousStateJson: "{}",
			NewStateJson:      string(statePayload),
			Status:            "STAGED",
			ConfirmationNonce: nonce,
			NonceExpiresAt:    sql.NullTime{Time: expiresAt, Valid: true},
			AppliedAt:         sql.NullTime{Valid: false},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if _, err := e.tluclawRepo.CreateSnapshot(ctx, snapshotParams); err != nil {
			log.Warn().Err(err).Msg("tluclaw ExecuteTool: create snapshot failed")
			return "Không thể tạo bản ghi xem trước. Vui lòng thử lại sau.", nil, nil
		}

		card := &models.TLUClawPreviewCardData{
			SnapshotID:  snapshotID,
			BatchID:     batchID,
			ActionType:  "academic_structure",
			Summary:     snapshotParams.Summary,
			Nonce:       nonce,
			ExpiresAt:   expiresAt,
			DiffItems:   diffItems,
			BlastRadius: blastRadius,
		}

		msg := fmt.Sprintf("Đã tạo bản xem trước cho Ngành %s (%s). Vui lòng kiểm tra và xác nhận áp dụng.", majorNameVi, majorCode)
		return msg, card.ToResponse(), nil

	case "propose_curriculum_update", "propose_curriculum", "update_curriculum":
		cohort := strings.TrimSpace(args.Cohort)
		majorCode := strings.TrimSpace(args.MajorCode)
		rawCourses := strings.TrimSpace(args.CoursesPayload)

		if cohort == "" || majorCode == "" || rawCourses == "" {
			return "Thiếu tham số bắt buộc: cohort, major_code, hoặc courses_payload", nil, nil
		}

		var incomingCourses []request.IncomingCourseRecord
		if err := json.Unmarshal([]byte(rawCourses), &incomingCourses); err != nil {
			return fmt.Sprintf("Không thể đọc danh sách môn học JSON: %v", err), nil, nil
		}

		if len(incomingCourses) == 0 {
			return "Danh sách môn học trích xuất rỗng", nil, nil
		}

		existingCourses, _ := e.curriculumRepo.GetCurriculumCoursesByCohortAndMajor(ctx, cohort, majorCode)
		existingMap := make(map[string]*models.CurriculumCourse)
		for _, ec := range existingCourses {
			existingMap[strings.ToUpper(strings.TrimSpace(ec.CourseCode))] = ec
		}

		var diffItems []models.TLUClawDiffItem
		newCount := 0
		updatedCount := 0

		for _, inc := range incomingCourses {
			cleanCode := strings.ToUpper(strings.TrimSpace(inc.CourseCode))
			if cleanCode == "" {
				continue
			}

			if existing, exists := existingMap[cleanCode]; exists {
				hasChanges := false
				var changeNotes []string
				if existing.Credits != inc.Credits {
					hasChanges = true
					changeNotes = append(changeNotes, fmt.Sprintf("Tín chỉ: %.1f -> %.1f", existing.Credits, inc.Credits))
				}
				if existing.CourseName != inc.CourseName && inc.CourseName != "" {
					hasChanges = true
					changeNotes = append(changeNotes, fmt.Sprintf("Tên: '%s' -> '%s'", existing.CourseName, inc.CourseName))
				}
				if existing.PrerequisitesText != inc.PrerequisitesText && inc.PrerequisitesText != "" {
					hasChanges = true
					changeNotes = append(changeNotes, fmt.Sprintf("Tiên quyết: '%s' -> '%s'", existing.PrerequisitesText, inc.PrerequisitesText))
				}

				if hasChanges {
					updatedCount++
					diffItems = append(diffItems, models.TLUClawDiffItem{
						EntityType: "COURSE",
						Action:     "UPDATE",
						Key:        cleanCode,
						Name:       inc.CourseName,
						OldValue:   fmt.Sprintf("%.1f TC", existing.Credits),
						NewValue:   fmt.Sprintf("%.1f TC", inc.Credits),
						Note:       strings.Join(changeNotes, " | "),
					})
				}
			} else {
				newCount++
				diffItems = append(diffItems, models.TLUClawDiffItem{
					EntityType: "COURSE",
					Action:     "CREATE",
					Key:        cleanCode,
					Name:       inc.CourseName,
					OldValue:   "-",
					NewValue:   fmt.Sprintf("%.1f TC", inc.Credits),
					Note:       fmt.Sprintf("Khối KT: %s | Tiên quyết: %s", inc.KnowledgeBlock, inc.PrerequisitesText),
				})
			}
		}

		riskLevel := "LOW"
		if updatedCount > 0 {
			riskLevel = "MEDIUM"
		}

		blastRadius := models.TLUClawBlastRadius{
			AffectedCohorts: []string{cohort},
			AffectedMajors:  []string{majorCode},
			ImpactSummary:   fmt.Sprintf("Cập nhật CTĐT Khóa %s Ngành %s: %d môn mới, %d môn sửa đổi", cohort, majorCode, newCount, updatedCount),
			RiskLevel:       riskLevel,
		}

		batchID := "batch_" + uuid.NewString()[:8]
		nonce, _ := crypto.GenerateRandomHex(24)
		expiresAt := time.Now().Add(15 * time.Minute)

		diffJSON, _ := json.Marshal(diffItems)
		blastJSON, _ := json.Marshal(blastRadius)

		statePayload, _ := json.Marshal(request.TLUClawCurriculumSnapshotState{
			Cohort:          cohort,
			MajorCode:       majorCode,
			IncomingCourses: incomingCourses,
		})

		priorStatePayload, _ := json.Marshal(models.TLUClawCurriculumPriorSnapshotState{
			Cohort:          cohort,
			MajorCode:       majorCode,
			ExistingCourses: existingCourses,
		})

		snapshotID := "snp_" + uuid.NewString()[:8]
		snapshotParams := sqlc.CreateTLUClawSnapshotParams{
			ID:                snapshotID,
			BatchID:           batchID,
			SessionID:         sessionID,
			OperatorID:        adminID,
			ActionType:        "curriculum_update",
			Summary:           fmt.Sprintf("Cập nhật khung chương trình %s (%s): +%d mới, ~%d sửa đổi", majorCode, cohort, newCount, updatedCount),
			DiffPreviewJson:   string(diffJSON),
			BlastRadiusJson:   string(blastJSON),
			PreviousStateJson: string(priorStatePayload),
			NewStateJson:      string(statePayload),
			Status:            "STAGED",
			ConfirmationNonce: nonce,
			NonceExpiresAt:    sql.NullTime{Time: expiresAt, Valid: true},
			AppliedAt:         sql.NullTime{Valid: false},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if _, err := e.tluclawRepo.CreateSnapshot(ctx, snapshotParams); err != nil {
			log.Warn().Err(err).Msg("tluclaw ExecuteTool: create snapshot failed")
			return "Không thể tạo bản ghi xem trước. Vui lòng thử lại sau.", nil, nil
		}

		card := &models.TLUClawPreviewCardData{
			SnapshotID:  snapshotID,
			BatchID:     batchID,
			ActionType:  "curriculum_update",
			Summary:     snapshotParams.Summary,
			Nonce:       nonce,
			ExpiresAt:   expiresAt,
			DiffItems:   diffItems,
			BlastRadius: blastRadius,
		}

		msg := fmt.Sprintf("Đã lập bảng đối chiếu Diff cho Khóa %s Ngành %s (%d môn thêm mới, %d môn sửa đổi). Vui lòng xem bản xem trước bên dưới và xác nhận áp dụng.", cohort, majorCode, newCount, updatedCount)
		return msg, card.ToResponse(), nil

	default:
		return fmt.Sprintf("Hành động không được hỗ trợ: %s", action), nil, nil
	}
}

func (e *tluclawToolEngine) executeDocumentOperations(ctx context.Context, sessionID, adminID string, args *request.TLUClawDocumentOpArgs) (string, *response.TLUClawPreviewCardResponse, error) {
	action := args.Action

	switch action {
	case "list_active_regulations":
		domain := strings.TrimSpace(args.Domain)
		cohort := strings.TrimSpace(args.ApplicableCohort)
		page := args.Page
		if page <= 0 {
			page = 1
		}
		limit := args.Limit
		if limit <= 0 || limit > 100 {
			limit = 50
		}
		dto := &request.ListDocumentsDto{
			PaginationDto: request.PaginationDto{Page: page, Limit: limit},
			Status:        "ACTIVE",
			Domain:        domain,
		}
		docs, _, err := e.ragRepo.ListDocuments(ctx, dto)
		if err != nil || len(docs) == 0 {
			return "Hiện không có văn bản quy chế nào ở trạng thái ACTIVE phù hợp điều kiện lọc.", nil, nil
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "# DANH SÁCH VĂN BẢN QUY CHẾ HIỆN HÀNH (ACTIVE - %d văn bản):\n", len(docs))
		for _, d := range docs {
			if cohort != "" && d.ApplicableCohort != "" && !strings.EqualFold(d.ApplicableCohort, cohort) {
				continue
			}
			fmt.Fprintf(&sb, "- ID: `%s` | Tiêu đề: %s | Lĩnh vực: %s | Niên khóa: %s | Khóa: %s\n", d.ID, d.Title, d.Domain, d.AcademicYear, d.ApplicableCohort)
		}
		return sb.String(), nil, nil

	case "analyze_supersession_impact":
		title := strings.TrimSpace(args.DocumentTitle)
		docType := strings.TrimSpace(args.DocumentType)
		domain := strings.TrimSpace(args.Domain)
		cohort := strings.TrimSpace(args.ApplicableCohort)
		academicYear := strings.TrimSpace(args.AcademicYear)
		content := strings.TrimSpace(args.DocumentContent)

		if title == "" {
			return "Thiếu tiêu đề văn bản document_title", nil, nil
		}
		if docType == "" {
			docType = "QUY_CHE"
		}
		if domain == "" {
			domain = "DAO_TAO"
		}

		newDocID := "DOC_" + uuid.NewString()[:8]
		plan, err := e.dissectionEngine.ExecuteDissectionLoop(ctx, &DissectionRequest{
			NewDocID:         newDocID,
			Title:            title,
			DocCode:          newDocID,
			DocType:          docType,
			Domain:           domain,
			ApplicableCohort: cohort,
			AcademicYear:     academicYear,
			Content:          content,
		})
		if err != nil || plan == nil {
			return "Không thể phân tích tác động bãi bỏ quy chế", nil, nil
		}

		batchID := "batch_" + uuid.NewString()[:8]
		nonce, _ := crypto.GenerateRandomHex(24)
		expiresAt := time.Now().Add(15 * time.Minute)

		diffJSON, _ := json.Marshal(plan.DiffItems)
		blastJSON, _ := json.Marshal(plan.BlastRadius)
		planJSON, _ := json.Marshal(plan)

		statePayload, _ := json.Marshal(request.TLUClawDocumentSnapshotState{
			NewDocID:         newDocID,
			Title:            title,
			DocType:          docType,
			Domain:           domain,
			Cohort:           cohort,
			AcademicYear:     academicYear,
			Content:          content,
			SupersededDocIDs: plan.BlastRadius.SupersededDocIDs,
			PlanJSON:         string(planJSON),
		})

		snapshotID := "snp_" + uuid.NewString()[:8]
		snapshotParams := sqlc.CreateTLUClawSnapshotParams{
			ID:                snapshotID,
			BatchID:           batchID,
			SessionID:         sessionID,
			OperatorID:        adminID,
			ActionType:        "document_supersession",
			Summary:           fmt.Sprintf("Phân tích bãi bỏ & sửa đổi: '%s' (%s)", title, plan.ActionType),
			DiffPreviewJson:   string(diffJSON),
			BlastRadiusJson:   string(blastJSON),
			PreviousStateJson: "{}",
			NewStateJson:      string(statePayload),
			Status:            "STAGED",
			ConfirmationNonce: nonce,
			NonceExpiresAt:    sql.NullTime{Time: expiresAt, Valid: true},
			AppliedAt:         sql.NullTime{Valid: false},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if _, err := e.tluclawRepo.CreateSnapshot(ctx, snapshotParams); err != nil {
			log.Warn().Err(err).Msg("tluclaw ExecuteTool: create snapshot failed")
			return "Không thể tạo bản ghi xem trước. Vui lòng thử lại sau.", nil, nil
		}

		card := &models.TLUClawPreviewCardData{
			SnapshotID:  snapshotID,
			BatchID:     batchID,
			ActionType:  "document_supersession",
			Summary:     snapshotParams.Summary,
			Nonce:       nonce,
			ExpiresAt:   expiresAt,
			DiffItems:   plan.DiffItems,
			BlastRadius: plan.BlastRadius,
		}

		msg := fmt.Sprintf("Đã phân tích tác động bãi bỏ cho văn bản '%s'. Phân loại: %s. Phát hiện %d văn bản bị ảnh hưởng và %d điều khoản bãi bỏ/thay thế. Vui lòng kiểm tra bản xem trước và xác nhận áp dụng.", title, plan.ActionType, len(plan.BlastRadius.SupersededDocIDs), len(plan.SupersededArticles))
		return msg, card.ToResponse(), nil

	case "propose_ingestion":
		title := strings.TrimSpace(args.DocumentTitle)
		docType := strings.TrimSpace(args.DocumentType)
		domain := strings.TrimSpace(args.Domain)
		cohort := strings.TrimSpace(args.ApplicableCohort)
		academicYear := strings.TrimSpace(args.AcademicYear)
		content := strings.TrimSpace(args.DocumentContent)

		if title == "" {
			return "Thiếu tiêu đề văn bản document_title", nil, nil
		}

		docID := "DOC_" + uuid.NewString()[:8]
		diffItems := []models.TLUClawDiffItem{
			{
				EntityType: "DOCUMENT",
				Action:     "INGEST",
				Key:        docID,
				Name:       title,
				OldValue:   "-",
				NewValue:   "ACTIVE",
				Note:       fmt.Sprintf("Loại: %s | Lĩnh vực: %s | Khóa: %s", docType, domain, cohort),
			},
		}

		blastRadius := models.TLUClawBlastRadius{
			AffectedCohorts: []string{cohort},
			ImpactSummary:   fmt.Sprintf("Nạp văn bản '%s' vào cơ sở tri thức RAG mà không bãi bỏ văn bản cũ", title),
			RiskLevel:       "LOW",
		}

		batchID := "batch_" + uuid.NewString()[:8]
		nonce, _ := crypto.GenerateRandomHex(24)
		expiresAt := time.Now().Add(15 * time.Minute)

		diffJSON, _ := json.Marshal(diffItems)
		blastJSON, _ := json.Marshal(blastRadius)

		statePayload, _ := json.Marshal(request.TLUClawDocumentSnapshotState{
			NewDocID:     docID,
			Title:        title,
			DocType:      docType,
			Domain:       domain,
			Cohort:       cohort,
			AcademicYear: academicYear,
			Content:      content,
		})

		snapshotID := "snp_" + uuid.NewString()[:8]
		snapshotParams := sqlc.CreateTLUClawSnapshotParams{
			ID:                snapshotID,
			BatchID:           batchID,
			SessionID:         sessionID,
			OperatorID:        adminID,
			ActionType:        "document_ingestion",
			Summary:           fmt.Sprintf("Nạp tài liệu mới: '%s'", title),
			DiffPreviewJson:   string(diffJSON),
			BlastRadiusJson:   string(blastJSON),
			PreviousStateJson: "{}",
			NewStateJson:      string(statePayload),
			Status:            "STAGED",
			ConfirmationNonce: nonce,
			NonceExpiresAt:    sql.NullTime{Time: expiresAt, Valid: true},
			AppliedAt:         sql.NullTime{Valid: false},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if _, err := e.tluclawRepo.CreateSnapshot(ctx, snapshotParams); err != nil {
			log.Warn().Err(err).Msg("tluclaw ExecuteTool: create snapshot failed")
			return "Không thể tạo bản ghi xem trước. Vui lòng thử lại sau.", nil, nil
		}

		card := &models.TLUClawPreviewCardData{
			SnapshotID:  snapshotID,
			BatchID:     batchID,
			ActionType:  "document_ingestion",
			Summary:     snapshotParams.Summary,
			Nonce:       nonce,
			ExpiresAt:   expiresAt,
			DiffItems:   diffItems,
			BlastRadius: blastRadius,
		}

		msg := fmt.Sprintf("Đã tạo bản xem trước nạp tài liệu '%s'. Vui lòng kiểm tra và xác nhận áp dụng.", title)
		return msg, card.ToResponse(), nil

	case "search_knowledge":
		query := strings.TrimSpace(args.Query)
		if query == "" {
			query = strings.TrimSpace(args.DocumentTitle)
		}
		if query == "" {
			return "Vui lòng cung cấp câu truy vấn query để tìm kiếm tri thức.", nil, nil
		}
		cohort := strings.TrimSpace(args.ApplicableCohort)
		domain := strings.TrimSpace(args.Domain)

		limitVal := args.Limit
		if limitVal <= 0 || limitVal > 50 {
			limitVal = 5
		}
		chunkIDs, err := e.ragRepo.SearchActiveChunkIDsFTS(ctx, query, cohort, domain, limitVal)
		if err != nil || len(chunkIDs) == 0 {
			return fmt.Sprintf("Không tìm thấy đoạn quy chế nào phù hợp với câu truy vấn '%s'.", query), nil, nil
		}
		chunks, err := e.ragRepo.GetChunksByIDs(ctx, chunkIDs)
		if err != nil || len(chunks) == 0 {
			return fmt.Sprintf("Không thể tải nội dung chi tiết các đoạn văn bản (số kết quả: %d).", len(chunkIDs)), nil, nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "# KẾT QUẢ TÌM KIẾM TRI THỨC VĂN BẢN QUY CHẾ (%d kết quả):\n\n", len(chunks))
		for i, c := range chunks {
			fmt.Fprintf(&sb, "### %d. Chunk `%s` (Văn bản: `%s`)\n", i+1, c.ID, c.DocumentID)
			fmt.Fprintf(&sb, "- **Loại đoạn**: `%s` | **Lĩnh vực**: %s\n", c.ChunkType, c.Domain)
			snippet := c.Content
			if len(snippet) > 300 {
				snippet = snippet[:300] + "..."
			}
			fmt.Fprintf(&sb, "- **Trích dẫn**: %s\n\n", snippet)
		}
		return sb.String(), nil, nil

	case "inspect_kg_entity":
		nodeID := strings.TrimSpace(args.NodeID)
		entityName := strings.TrimSpace(args.DocumentTitle)
		if nodeID == "" && entityName == "" {
			return "Vui lòng cung cấp node_id hoặc tên thực thể để tra cứu Knowledge Graph.", nil, nil
		}

		nodes, err := e.ragRepo.ListAllKGNodes(ctx)
		if err != nil || len(nodes) == 0 {
			return "Cơ sở dữ liệu Knowledge Graph hiện chưa có thực thể nào.", nil, nil
		}

		var targetNode *models.KGNode
		for _, n := range nodes {
			if nodeID != "" && strings.EqualFold(n.ID, nodeID) {
				targetNode = n
				break
			}
			if entityName != "" && (strings.EqualFold(n.Name, entityName) || strings.Contains(strings.ToLower(n.Name), strings.ToLower(entityName))) {
				targetNode = n
				break
			}
		}

		if targetNode == nil {
			return fmt.Sprintf("Không tìm thấy thực thể Knowledge Graph phù hợp với tiêu chí (node_id: %s, name: %s).", nodeID, entityName), nil, nil
		}

		edges, _ := e.ragRepo.ListAllKGEdges(ctx)
		var connectedEdges []*models.KGEdge
		for _, eg := range edges {
			if eg.SourceNodeID == targetNode.ID || eg.TargetNodeID == targetNode.ID {
				connectedEdges = append(connectedEdges, eg)
			}
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "# THỰC THỂ KNOWLEDGE GRAPH: %s (ID: `%s`)\n\n", targetNode.Name, targetNode.ID)
		fmt.Fprintf(&sb, "- **Loại thực thể (NodeType)**: `%s`\n", targetNode.NodeType)
		if targetNode.Code != "" {
			fmt.Fprintf(&sb, "- **Mã thực thể**: `%s`\n", targetNode.Code)
		}
		fmt.Fprintf(&sb, "- **Trạng thái**: `%s`\n", targetNode.Status)
		fmt.Fprintf(&sb, "\n### Các quan hệ liên kết (%d quan hệ):\n", len(connectedEdges))
		if len(connectedEdges) == 0 {
			sb.WriteString("- *Chưa có liên kết đồ thị nào với các thực thể khác.*\n")
		} else {
			sb.WriteString("| ID Quan hệ | Chiều | Thực thể đối ứng | Loại quan hệ | Trọng số |\n")
			sb.WriteString("| :--- | :--- | :--- | :--- | :--- |\n")
			for _, eg := range connectedEdges {
				direction := "Đi ra ->"
				otherID := eg.TargetNodeID
				if eg.TargetNodeID == targetNode.ID {
					direction = "<- Đi vào"
					otherID = eg.SourceNodeID
				}
				fmt.Fprintf(&sb, "| `%s` | %s | `%s` | `%s` | %.2f |\n", eg.ID, direction, otherID, eg.RelationType, eg.Weight)
			}
		}
		return sb.String(), nil, nil

	case "propose_document_status":
		docID := strings.TrimSpace(args.DocumentID)
		status := strings.ToUpper(strings.TrimSpace(args.Status))
		reason := strings.TrimSpace(args.Reason)

		if docID == "" {
			return "Thiếu document_id của văn bản cần thay đổi trạng thái.", nil, nil
		}
		if status != "ACTIVE" && status != "SUPERSEDED" && status != "ARCHIVED" && status != "DRAFT" {
			return "Trạng thái văn bản không hợp lệ. Cho phép: ACTIVE, SUPERSEDED, ARCHIVED, DRAFT.", nil, nil
		}

		doc, err := e.ragRepo.GetDocumentByID(ctx, docID)
		if err != nil || doc == nil {
			return fmt.Sprintf("Không tìm thấy văn bản có mã ID '%s'.", docID), nil, nil
		}

		diffItems := []models.TLUClawDiffItem{
			{
				EntityType: "DOCUMENT_STATUS",
				Action:     "UPDATE_STATUS",
				Key:        doc.ID,
				Name:       doc.Title,
				OldValue:   doc.Status,
				NewValue:   status,
				Note:       reason,
			},
		}

		blastRadius := models.TLUClawBlastRadius{
			ImpactSummary: fmt.Sprintf("Đổi trạng thái văn bản '%s' (%s) sang %s (ảnh hưởng kết quả RAG lĩnh vực %s)", doc.Title, doc.ID, status, doc.Domain),
			RiskLevel:     "MEDIUM",
		}

		batchID := "batch_doc_status_" + uuid.NewString()[:8]
		nonce, _ := crypto.GenerateRandomHex(24)
		expiresAt := time.Now().Add(15 * time.Minute)

		prevStatePayload, _ := json.Marshal(request.TLUClawDocumentStatusSnapshotState{
			DocumentID: doc.ID,
			Status:     doc.Status,
		})
		newStatePayload, _ := json.Marshal(request.TLUClawDocumentStatusSnapshotState{
			DocumentID: doc.ID,
			Status:     status,
			Reason:     reason,
		})

		diffJSON, _ := json.Marshal(diffItems)
		blastJSON, _ := json.Marshal(blastRadius)

		snapshotID := "snp_" + uuid.NewString()[:8]
		snapshotParams := sqlc.CreateTLUClawSnapshotParams{
			ID:                snapshotID,
			BatchID:           batchID,
			SessionID:         sessionID,
			OperatorID:        adminID,
			ActionType:        "document_status",
			Summary:           fmt.Sprintf("Đổi trạng thái văn bản '%s' từ %s sang %s", doc.Title, doc.Status, status),
			DiffPreviewJson:   string(diffJSON),
			BlastRadiusJson:   string(blastJSON),
			PreviousStateJson: string(prevStatePayload),
			NewStateJson:      string(newStatePayload),
			Status:            "STAGED",
			ConfirmationNonce: nonce,
			NonceExpiresAt:    sql.NullTime{Time: expiresAt, Valid: true},
			AppliedAt:         sql.NullTime{Valid: false},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if _, err := e.tluclawRepo.CreateSnapshot(ctx, snapshotParams); err != nil {
			return "Không thể tạo bản ghi xem trước. Vui lòng thử lại sau.", nil, nil
		}

		card := &models.TLUClawPreviewCardData{
			SnapshotID:  snapshotID,
			BatchID:     batchID,
			ActionType:  "document_status",
			Summary:     snapshotParams.Summary,
			Nonce:       nonce,
			ExpiresAt:   expiresAt,
			DiffItems:   diffItems,
			BlastRadius: blastRadius,
		}

		msg := fmt.Sprintf("Đã tạo bản xem trước thay đổi trạng thái văn bản '%s' sang `%s`. Vui lòng xác nhận áp dụng.", doc.Title, status)
		return msg, card.ToResponse(), nil

	default:
		return fmt.Sprintf("Hành động không được hỗ trợ: %s", action), nil, nil
	}
}

func (e *tluclawToolEngine) executeAuditOperations(ctx context.Context, sessionID, adminID string, args *request.TLUClawAuditOpArgs) (string, *response.TLUClawPreviewCardResponse, error) {
	action := args.Action

	switch action {
	case "list_recent_batches":
		limitVal := int32(args.Limit)
		if limitVal <= 0 || limitVal > 50 {
			limitVal = 5
		}

		snapshots, err := e.tluclawRepo.ListLatestCommittedSnapshots(ctx, limitVal)
		if err != nil {
			return fmt.Sprintf("Lỗi tra cứu lịch sử thao tác: %v", err), nil, nil
		}

		if len(snapshots) == 0 {
			return "Chưa có phiên thay đổi nào được thực hiện gần đây.", nil, nil
		}

		var b strings.Builder
		b.WriteString("### Danh sách các phiên thay đổi gần nhất:\n\n")
		b.WriteString("| Batch ID | Loại thao tác | Tóm tắt | Thời gian áp dụng |\n")
		b.WriteString("| :--- | :--- | :--- | :--- |\n")
		for _, s := range snapshots {
			appliedTime := "-"
			if s.AppliedAt != nil {
				appliedTime = s.AppliedAt.Format("15:04 02/01/2006")
			}
			fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", s.BatchID, s.ActionType, s.Summary, appliedTime)
		}
		b.WriteString("\n*Admin có thể yêu cầu: 'Hãy hoàn tác phiên `batch_xyz`' để khôi phục trạng thái cũ.*")
		return b.String(), nil, nil

	case "inspect_batch_snapshot":
		batchID := strings.TrimSpace(args.BatchID)
		if batchID == "" {
			return "Thiếu batch_id cần tra cứu", nil, nil
		}

		s, err := e.tluclawRepo.GetSnapshotByBatchID(ctx, batchID)
		if err != nil || s == nil {
			return fmt.Sprintf("Không tìm thấy snapshot cho batch '%s'", batchID), nil, nil
		}

		var b strings.Builder
		fmt.Fprintf(&b, "### Chi tiết Snapshot phiên `%s`:\n\n", batchID)
		fmt.Fprintf(&b, "- **Hành động**: %s\n", s.ActionType)
		fmt.Fprintf(&b, "- **Trạng thái**: %s\n", s.Status)
		fmt.Fprintf(&b, "- **Nội dung**: %s\n\n", s.Summary)

		var diffs []models.TLUClawDiffItem
		_ = json.Unmarshal([]byte(s.DiffPreviewJSON), &diffs)
		if len(diffs) > 0 {
			b.WriteString("| Thực thể | Thao tác | Khóa | Cũ -> Mới | Ghi chú |\n")
			b.WriteString("| :--- | :--- | :--- | :--- | :--- |\n")
			for _, d := range diffs {
				fmt.Fprintf(&b, "| %s | %s | `%s` | %s -> %s | %s |\n", d.EntityType, d.Action, d.Key, d.OldValue, d.NewValue, d.Note)
			}
		}
		return b.String(), nil, nil

	default:
		return fmt.Sprintf("Hành động không được hỗ trợ: %s", action), nil, nil
	}
}

func (e *tluclawToolEngine) executeMemoryOperations(ctx context.Context, adminID string, args *request.TLUClawMemoryOpArgs) (string, *response.TLUClawPreviewCardResponse, error) {
	if e.memoryService == nil || adminID == "" {
		return "Dịch vụ bộ nhớ quản trị viên chưa sẵn sàng.", nil, nil
	}
	action := args.Action
	switch action {
	case "save_preference":
		key := strings.TrimSpace(args.Key)
		if key == "" {
			key = strings.TrimSpace(args.MemoryKey)
		}
		val := strings.TrimSpace(args.Value)
		if val == "" {
			val = strings.TrimSpace(args.MemoryValue)
		}
		cat := strings.TrimSpace(args.Category)
		if cat == "" {
			cat = strings.TrimSpace(args.TargetKey)
		}
		if key == "" || val == "" {
			return "Thiếu 'key' hoặc 'value' để lưu sở thích quản trị.", nil, nil
		}
		if cat == "" {
			cat = "ADMIN_PREFERENCE"
		}
		item := &models.UserMemoryItem{
			UserID:      adminID,
			Category:    cat,
			MemoryKey:   key,
			MemoryValue: val,
			Confidence:  1.0,
			Source:      "ADMIN_CHAT",
		}
		_, err := e.memoryService.SaveUserMemory(ctx, item)
		if err != nil {
			return fmt.Sprintf("Lỗi lưu sở thích: %v", err), nil, nil
		}
		return fmt.Sprintf("Đã ghi nhớ thông số quản trị: [%s] %s = %s", cat, key, val), nil, nil

	case "list_preferences":
		mems, err := e.memoryService.ListUserMemories(ctx, adminID)
		if err != nil || len(mems) == 0 {
			return "Chưa có thông số quản trị lưu trữ nào.", nil, nil
		}
		var sb strings.Builder
		sb.WriteString("# THÔNG SỐ VÀ SỞ THÍCH QUẢN TRỊ VIÊN ĐÃ LƯU TRỮ:\n")
		for _, m := range mems {
			fmt.Fprintf(&sb, "- [%s] %s: %s\n", m.Category, m.MemoryKey, m.MemoryValue)
		}
		return sb.String(), nil, nil

	default:
		return fmt.Sprintf("Hành động không hợp lệ: %s", action), nil, nil
	}
}

func (e *tluclawToolEngine) resolveTargetUser(ctx context.Context, studentCode, userID, email string) (*models.UserEntity, error) {
	if e.userRepo == nil {
		return nil, sql.ErrNoRows
	}
	cleanCode := strings.ToUpper(strings.TrimSpace(studentCode))
	if cleanCode != "" {
		if user, err := e.userRepo.GetByStudentCode(ctx, cleanCode); err == nil && user != nil {
			return user, nil
		}
	}
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	if cleanEmail != "" {
		if user, err := e.userRepo.GetByEmail(ctx, cleanEmail); err == nil && user != nil {
			return user, nil
		}
	}
	cleanID := strings.TrimSpace(userID)
	if cleanID != "" {
		if user, err := e.userRepo.GetByID(ctx, cleanID); err == nil && user != nil {
			return user, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (e *tluclawToolEngine) executeSubstitutionOperations(ctx context.Context, sessionID, adminID string, args *request.TLUClawSubstitutionOpArgs) (string, *response.TLUClawPreviewCardResponse, error) {
	action := args.Action

	switch action {
	case "inspect_substitutions":
		courseCode := strings.TrimSpace(args.CourseCode)
		cohortStr := strings.TrimSpace(args.Cohort)
		major := strings.TrimSpace(args.Major)
		keyword := strings.TrimSpace(args.Keyword)
		limitVal := args.Limit
		if limitVal <= 0 || limitVal > 100 {
			limitVal = 20
		}
		offsetVal := int64((args.Page - 1) * limitVal)
		if offsetVal < 0 {
			offsetVal = 0
		}

		var cohortNum int64
		if strings.HasPrefix(strings.ToUpper(cohortStr), "K") {
			if n, err := strconv.ParseInt(strings.TrimPrefix(strings.ToUpper(cohortStr), "K"), 10, 64); err == nil {
				cohortNum = n
			}
		}

		cleanCourseCode := strings.ToUpper(strings.TrimSpace(courseCode))
		if cleanCourseCode != "" && keyword == "" {
			keyword = cleanCourseCode
		}

		subs, err := e.curriculumRepo.ListSubstitutionsPaginated(ctx, sqlc.ListSubstitutionsPaginatedParams{
			CohortNum:    cohortNum,
			ScopeMajor:   major,
			FacultyCode:  major,
			MajorAliases: "",
			Keyword:      keyword,
			LimitVal:     int64(limitVal),
			OffsetVal:    offsetVal,
		})
		if err != nil || len(subs) == 0 {
			return "Không tìm thấy quy tắc môn thay thế nào phù hợp với điều kiện tìm kiếm.", nil, nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# DANH SÁCH QUY TẮC MÔN HỌC THAY THẾ (%d kết quả):\n\n", len(subs)))
		sb.WriteString("| Môn gốc | Tên môn gốc | Môn thay thế | Tên môn thay thế | Khóa áp dụng | Ngành | Hai chiều | Ghi chú |\n")
		sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")
		for _, s := range subs {
			bidirectionalStr := "Một chiều"
			if s.IsBidirectional {
				bidirectionalStr = "Hai chiều"
			}
			cohortRange := fmt.Sprintf("K%d - K%d", s.MinCohort, s.MaxCohort)
			fmt.Fprintf(&sb, "| `%s` | %s | `%s` | %s | %s | %s | %s | %s |\n",
				s.OriginalCourseCode, s.OriginalCourseName, s.ReplacementCourseCode, s.ReplacementCourseName,
				cohortRange, s.ScopeMajor, bidirectionalStr, s.RuleNote)
		}
		return sb.String(), nil, nil

	case "propose_substitution":
		origCode := strings.ToUpper(strings.TrimSpace(args.OriginalCourseCode))
		replCode := strings.ToUpper(strings.TrimSpace(args.ReplacementCourseCode))
		if origCode == "" || replCode == "" {
			return "Thiếu thông tin bắt buộc: original_course_code hoặc replacement_course_code", nil, nil
		}

		origName := strings.TrimSpace(args.OriginalCourseName)
		replName := strings.TrimSpace(args.ReplacementCourseName)
		if origName == "" {
			if meta, err := e.curriculumRepo.FindCourseMetadataByCode(ctx, origCode); err == nil && meta != nil {
				origName = meta.CourseName
			}
		}
		if replName == "" {
			if meta, err := e.curriculumRepo.FindCourseMetadataByCode(ctx, replCode); err == nil && meta != nil {
				replName = meta.CourseName
			}
		}

		minCohort := int64(args.MinCohort)
		if minCohort <= 0 {
			minCohort = 24
		}
		maxCohort := int64(args.MaxCohort)
		if maxCohort <= 0 {
			maxCohort = 40
		}

		scopeMajor := strings.TrimSpace(args.ScopeMajor)
		if scopeMajor == "" {
			scopeMajor = "ALL"
		}
		isBidirectional := args.IsBidirectional
		ruleNote := strings.TrimSpace(args.RuleNote)

		creditsOrig := args.CreditsOriginal
		creditsRepl := args.CreditsReplacement

		subID := fmt.Sprintf("sub_%s_%s_%d", origCode, replCode, minCohort)
		existingSub, _ := e.curriculumRepo.GetSubstitutionByID(ctx, subID)
		diffAction := "CREATE"
		oldVal := ""
		if existingSub != nil {
			diffAction = "UPDATE"
			oldVal = fmt.Sprintf("%s -> %s (K%d-K%d)", existingSub.OriginalCourseCode, existingSub.ReplacementCourseCode, existingSub.MinCohort, existingSub.MaxCohort)
		}

		newVal := fmt.Sprintf("%s (%s) -> %s (%s) [K%d - K%d, Ngành: %s]", origCode, origName, replCode, replName, minCohort, maxCohort, scopeMajor)
		diffItems := []models.TLUClawDiffItem{
			{
				EntityType: "COURSE_SUBSTITUTION",
				Action:     diffAction,
				Key:        fmt.Sprintf("%s -> %s", origCode, replCode),
				Name:       fmt.Sprintf("Quy tắc thay thế: %s thay cho %s", replCode, origCode),
				OldValue:   oldVal,
				NewValue:   newVal,
				Note:       ruleNote,
			},
		}

		blastRadius := models.TLUClawBlastRadius{
			AffectedCohorts: []string{fmt.Sprintf("K%d..K%d", minCohort, maxCohort)},
			AffectedMajors:  []string{scopeMajor},
			ImpactSummary:   fmt.Sprintf("Áp dụng quy tắc môn thay thế %s -> %s cho sinh viên từ K%d đến K%d thuộc phạm vi ngành %s", origCode, replCode, minCohort, maxCohort, scopeMajor),
			RiskLevel:       "MEDIUM",
		}

		batchID := "batch_" + uuid.NewString()[:8]
		nonce, err := crypto.GenerateRandomHex(24)
		if err != nil {
			nonce = uuid.NewString()
		}
		expiresAt := time.Now().Add(15 * time.Minute)

		diffJSON, _ := json.Marshal(diffItems)
		blastJSON, _ := json.Marshal(blastRadius)

		var isBidirInt int64
		if isBidirectional {
			isBidirInt = 1
		}

		statePayload, _ := json.Marshal(request.TLUClawSubstitutionSnapshotState{
			ID:                    subID,
			OriginalCourseCode:    origCode,
			OriginalCourseName:    origName,
			ReplacementCourseCode: replCode,
			ReplacementCourseName: replName,
			MinCohort:             minCohort,
			MaxCohort:             maxCohort,
			ScopeMajor:            scopeMajor,
			IsBidirectional:       isBidirInt,
			CreditsOriginal:       creditsOrig,
			CreditsReplacement:    creditsRepl,
			RuleNote:              ruleNote,
		})

		previousStateJSON := "{}"
		if existingSub != nil {
			prevBytes, _ := json.Marshal(existingSub)
			previousStateJSON = string(prevBytes)
		}

		snapshotID := "snp_" + uuid.NewString()[:8]
		snapshotParams := sqlc.CreateTLUClawSnapshotParams{
			ID:                snapshotID,
			BatchID:           batchID,
			SessionID:         sessionID,
			OperatorID:        adminID,
			ActionType:        "substitution_rule",
			Summary:           fmt.Sprintf("Đề xuất quy tắc thay thế học phần %s -> %s", origCode, replCode),
			DiffPreviewJson:   string(diffJSON),
			BlastRadiusJson:   string(blastJSON),
			PreviousStateJson: previousStateJSON,
			NewStateJson:      string(statePayload),
			Status:            "STAGED",
			ConfirmationNonce: nonce,
			NonceExpiresAt:    sql.NullTime{Time: expiresAt, Valid: true},
			AppliedAt:         sql.NullTime{Valid: false},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if _, err := e.tluclawRepo.CreateSnapshot(ctx, snapshotParams); err != nil {
			log.Warn().Err(err).Msg("tluclaw ExecuteTool: create snapshot failed")
			return "Không thể tạo bản ghi xem trước. Vui lòng thử lại sau.", nil, nil
		}

		card := &models.TLUClawPreviewCardData{
			SnapshotID:  snapshotID,
			BatchID:     batchID,
			ActionType:  "substitution_rule",
			Summary:     snapshotParams.Summary,
			Nonce:       nonce,
			ExpiresAt:   expiresAt,
			DiffItems:   diffItems,
			BlastRadius: blastRadius,
		}

		msg := fmt.Sprintf("Đã tạo bản xem trước đề xuất quy tắc môn thay thế %s -> %s. Vui lòng kiểm tra thẻ xem trước và xác nhận.", origCode, replCode)
		return msg, card.ToResponse(), nil

	case "inspect_prerequisites":
		courseCode := strings.ToUpper(strings.TrimSpace(args.CourseCode))
		if courseCode == "" {
			return "Thiếu tham số bắt buộc 'course_code'", nil, nil
		}
		cohort := strings.TrimSpace(args.Cohort)

		prereqs, _ := e.curriculumRepo.ListPrerequisitesForCourse(ctx, cohort, courseCode)
		deps, _ := e.curriculumRepo.ListDependentCourses(ctx, cohort, courseCode)

		var sb strings.Builder
		fmt.Fprintf(&sb, "# ĐIỀU KIỆN TIÊN QUYẾT CHO HỌC PHẦN `%s`:\n\n", courseCode)
		if len(prereqs) == 0 {
			sb.WriteString("- Học phần này **không yêu cầu môn học tiên quyết**.\n")
		} else {
			sb.WriteString("### Các học phần bắt buộc tiên quyết trước khi đăng ký:\n")
			for _, p := range prereqs {
				sb.WriteString(fmt.Sprintf("- Môn `%s` (Điểm sàn tối thiểu: `%.1f`, Loại điều kiện: `%s`)\n",
					p.PrerequisiteCourseCode, p.MinGrade, p.ConditionType))
			}
		}

		sb.WriteString("\n### Các học phần nối tiếp bị phụ thuộc (Yêu cầu môn này làm tiên quyết):\n")
		if len(deps) == 0 {
			sb.WriteString("- Không có học phần nào lấy môn này làm tiên quyết.\n")
		} else {
			for _, d := range deps {
				fmt.Fprintf(&sb, "- Môn `%s` (Khóa: %s)\n", d.CourseCode, d.Cohort)
			}
		}
		return sb.String(), nil, nil

	case "propose_prerequisite":
		courseCode := strings.ToUpper(strings.TrimSpace(args.CourseCode))
		prereqCode := strings.ToUpper(strings.TrimSpace(args.PrerequisiteCourseCode))
		if courseCode == "" || prereqCode == "" {
			return "Thiếu tham số bắt buộc: course_code hoặc prerequisite_course_code", nil, nil
		}
		cohort := strings.TrimSpace(args.Cohort)
		if cohort == "" {
			cohort = "ALL"
		}
		majorName := strings.TrimSpace(args.MajorName)
		minGrade := strings.TrimSpace(args.MinGrade)
		if minGrade == "" {
			minGrade = "D"
		}
		conditionType := strings.TrimSpace(args.ConditionType)
		if conditionType == "" {
			conditionType = "PREREQUISITE"
		}

		prereqID := fmt.Sprintf("prereq_%s_%s_%s", cohort, courseCode, prereqCode)
		diffItems := []models.TLUClawDiffItem{
			{
				EntityType: "PREREQUISITE",
				Action:     "CREATE",
				Key:        fmt.Sprintf("%s -> %s", courseCode, prereqCode),
				Name:       fmt.Sprintf("Điều kiện tiên quyết: %s yêu cầu %s", courseCode, prereqCode),
				OldValue:   "",
				NewValue:   fmt.Sprintf("Điểm sàn: %s | Loại: %s | Khóa: %s", minGrade, conditionType, cohort),
				Note:       fmt.Sprintf("Sinh viên phải tích lũy học phần %s trước khi được đăng ký học phần %s", prereqCode, courseCode),
			},
		}

		blastRadius := models.TLUClawBlastRadius{
			AffectedCohorts: []string{cohort},
			ImpactSummary:   fmt.Sprintf("Thiết lập ràng buộc tiên quyết: %s yêu cầu %s đối với khóa %s", courseCode, prereqCode, cohort),
			RiskLevel:       "MEDIUM",
		}

		batchID := "batch_" + uuid.NewString()[:8]
		nonce, err := crypto.GenerateRandomHex(24)
		if err != nil {
			nonce = uuid.NewString()
		}
		expiresAt := time.Now().Add(15 * time.Minute)

		diffJSON, _ := json.Marshal(diffItems)
		blastJSON, _ := json.Marshal(blastRadius)

		curriculumCourseID := ""
		if matches, err := e.curriculumRepo.GetCoursesByCodeAndCohort(ctx, cohort, courseCode); err == nil {
			for _, match := range matches {
				if majorName == "" || strings.EqualFold(match.MajorName, majorName) {
					curriculumCourseID = match.ID
					majorName = match.MajorName
					break
				}
			}
		}

		statePayload, _ := json.Marshal(request.TLUClawPrerequisiteSnapshotState{
			ID:                     prereqID,
			CurriculumCourseID:     curriculumCourseID,
			Cohort:                 cohort,
			MajorName:              majorName,
			CourseCode:             courseCode,
			PrerequisiteCourseCode: prereqCode,
			MinGrade:               minGrade,
			ConditionType:          conditionType,
			RawExpression:          fmt.Sprintf("%s >= %s", prereqCode, minGrade),
		})

		snapshotID := "snp_" + uuid.NewString()[:8]
		snapshotParams := sqlc.CreateTLUClawSnapshotParams{
			ID:                snapshotID,
			BatchID:           batchID,
			SessionID:         sessionID,
			OperatorID:        adminID,
			ActionType:        "prerequisite_rule",
			Summary:           fmt.Sprintf("Đề xuất thiết lập môn tiên quyết %s yêu cầu %s", courseCode, prereqCode),
			DiffPreviewJson:   string(diffJSON),
			BlastRadiusJson:   string(blastJSON),
			PreviousStateJson: "{}",
			NewStateJson:      string(statePayload),
			Status:            "STAGED",
			ConfirmationNonce: nonce,
			NonceExpiresAt:    sql.NullTime{Time: expiresAt, Valid: true},
			AppliedAt:         sql.NullTime{Valid: false},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if _, err := e.tluclawRepo.CreateSnapshot(ctx, snapshotParams); err != nil {
			log.Warn().Err(err).Msg("tluclaw ExecuteTool: create prerequisite snapshot failed")
			return "Không thể tạo bản ghi xem trước. Vui lòng thử lại sau.", nil, nil
		}

		card := &models.TLUClawPreviewCardData{
			SnapshotID:  snapshotID,
			BatchID:     batchID,
			ActionType:  "prerequisite_rule",
			Summary:     snapshotParams.Summary,
			Nonce:       nonce,
			ExpiresAt:   expiresAt,
			DiffItems:   diffItems,
			BlastRadius: blastRadius,
		}

		msg := fmt.Sprintf("Đã tạo bản xem trước điều kiện tiên quyết: %s yêu cầu %s. Vui lòng kiểm tra thẻ xem trước và xác nhận.", courseCode, prereqCode)
		return msg, card.ToResponse(), nil

	default:
		return fmt.Sprintf("Hành động không hợp lệ: %s", action), nil, nil
	}
}

func (e *tluclawToolEngine) executeStudentAcademicOperations(ctx context.Context, sessionID, adminID string, args *request.TLUClawStudentAcademicOpArgs) (string, *response.TLUClawPreviewCardResponse, error) {
	action := args.Action

	switch action {
	case "inspect_student_profile":
		studentCode := strings.TrimSpace(args.StudentCode)
		userID := strings.TrimSpace(args.UserID)
		email := strings.TrimSpace(args.Email)

		user, err := e.resolveTargetUser(ctx, studentCode, userID, email)
		if err != nil || user == nil {
			return fmt.Sprintf("Không tìm thấy sinh viên nào tương ứng với thông tin tra cứu (MSSV: %s, Email: %s, UserID: %s).", studentCode, email, userID), nil, nil
		}

		profile, _ := e.memoryRepo.GetStudentProfile(ctx, user.ID)
		var sb strings.Builder
		fmt.Fprintf(&sb, "# HỒ SƠ HỌC VỤ SINH VIÊN: %s (MSSV: %s)\n\n", user.FullName, user.StudentCode)
		fmt.Fprintf(&sb, "- **Email**: %s\n", user.Email)
		fmt.Fprintf(&sb, "- **User ID**: `%s`\n", user.ID)
		if profile != nil {
			fmt.Fprintf(&sb, "- **Khóa**: %s\n", profile.Cohort)
			fmt.Fprintf(&sb, "- **Ngành đào tạo**: %s\n", profile.Major)
			fmt.Fprintf(&sb, "- **Điểm GPA tích lũy**: %.2f\n", profile.CumulativeGPA)
			fmt.Fprintf(&sb, "- **Số tín chỉ tích lũy đạt**: %d tín chỉ\n", profile.CompletedCredits)
			fmt.Fprintf(&sb, "- **Trạng thái học tập**: `%s`\n", profile.AcademicStanding)
			if profile.AdvisorNotes != "" {
				fmt.Fprintf(&sb, "- **Ghi chú Cố vấn học tập**: %s\n", profile.AdvisorNotes)
			}
		} else {
			sb.WriteString("- *Chưa có hồ sơ học vụ mở rộng trong bảng student_academic_profiles.*\n")
		}

		if e.tuitionClient != nil && user.StudentCode != "" {
			if tInfo, err := e.tuitionClient.LookupTuition(ctx, user.StudentCode); err == nil && tInfo != nil && tInfo.Found {
				sb.WriteString("\n### TÌNH TRẠNG HỌC PHÍ & CÔNG NỢ\n\n")
				fmt.Fprintf(&sb, "- **Tổng nợ học phí hiện tại**: %s VNĐ\n", tInfo.TotalDebt)
				if len(tInfo.UnpaidBills) > 0 {
					fmt.Fprintf(&sb, "- **Số khoản nợ đang chờ**: %d khoản\n", len(tInfo.UnpaidBills))
				} else {
					sb.WriteString("- **Tình trạng nợ**: Đã thanh toán đầy đủ, không có khoản nợ tồn đọng.\n")
				}
				if tInfo.PaymentURL != "" {
					fmt.Fprintf(&sb, "- **Cổng thanh toán trực tuyến**: %s\n", tInfo.PaymentURL)
				}
			}
		}
		return sb.String(), nil, nil

	case "inspect_tuition_fee":
		studentCode := strings.TrimSpace(args.StudentCode)
		userID := strings.TrimSpace(args.UserID)
		email := strings.TrimSpace(args.Email)

		user, _ := e.resolveTargetUser(ctx, studentCode, userID, email)
		targetCode := studentCode
		if targetCode == "" && user != nil {
			targetCode = user.StudentCode
		}
		if targetCode == "" {
			return "Vui lòng cung cấp mã sinh viên (MSSV) để tra cứu học phí.", nil, nil
		}

		if e.tuitionClient == nil {
			return "Cổng tra cứu học phí chưa được cấu hình.", nil, nil
		}

		tInfo, err := e.tuitionClient.LookupTuition(ctx, targetCode)
		if err != nil {
			return fmt.Sprintf("Không thể tra cứu học phí cho sinh viên %s: %v", targetCode, err), nil, nil
		}
		if !tInfo.Found {
			return fmt.Sprintf("Không tìm thấy dữ liệu học phí cho sinh viên có mã '%s' trên cổng thanh toán học phí của Trường.", targetCode), nil, nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "# BÁO CÁO HỌC PHÍ & CÔNG NỢ: %s (MSSV: `%s`)\n\n", tInfo.StudentName, tInfo.StudentID)
		if tInfo.Faculty != "" {
			fmt.Fprintf(&sb, "- **Khoa/Ngành**: %s\n", tInfo.Faculty)
		}
		fmt.Fprintf(&sb, "- **Tổng nợ học phí hiện tại**: **%s VNĐ**\n", tInfo.TotalDebt)
		if tInfo.PaymentURL != "" {
			fmt.Fprintf(&sb, "- **Cổng thanh toán**: %s\n", tInfo.PaymentURL)
		}

		if len(tInfo.UnpaidBills) > 0 {
			fmt.Fprintf(&sb, "\n### Danh sách khoản nợ đang chờ thanh toán (%d khoản):\n\n", len(tInfo.UnpaidBills))
			sb.WriteString("| Mã học phần | Tên học phần | Loại khoản | Số tiền |\n")
			sb.WriteString("| :--- | :--- | :--- | :--- |\n")
			for _, b := range tInfo.UnpaidBills {
				fmt.Fprintf(&sb, "| `%s` | %s | %s | %s VNĐ |\n", b.Code, b.Name, b.FeeType, b.Amount)
			}
		} else {
			sb.WriteString("\n*Sinh viên đã hoàn thành toàn bộ học phí, không có khoản nợ nào.*\n")
		}

		if len(tInfo.RecentInvoices) > 0 {
			fmt.Fprintf(&sb, "\n### Lịch sử hóa đơn gần đây (%d hóa đơn):\n\n", len(tInfo.RecentInvoices))
			sb.WriteString("| Mã hóa đơn | Ngày tạo | Hạn thanh toán | Tổng tiền | Trạng thái |\n")
			sb.WriteString("| :--- | :--- | :--- | :--- |\n")
			for _, inv := range tInfo.RecentInvoices {
				fmt.Fprintf(&sb, "| `%s` | %s | %s | %s VNĐ | %s |\n", inv.InvoiceCode, inv.CreatedDate, inv.ExpiryDate, inv.TotalAmount, inv.Status)
			}
		}
		return sb.String(), nil, nil

	case "inspect_student_transcript":
		studentCode := strings.TrimSpace(args.StudentCode)
		userID := strings.TrimSpace(args.UserID)
		email := strings.TrimSpace(args.Email)

		user, err := e.resolveTargetUser(ctx, studentCode, userID, email)
		if err != nil || user == nil {
			return fmt.Sprintf("Không tìm thấy sinh viên nào tương ứng với thông tin tra cứu (MSSV: %s).", studentCode), nil, nil
		}

		if e.transcriptRepo == nil {
			return "Dịch vụ bảng điểm chưa sẵn sàng.", nil, nil
		}

		grades, err := e.transcriptRepo.ListByUserID(ctx, user.ID)
		if err != nil || len(grades) == 0 {
			return fmt.Sprintf("Sinh viên **%s** (MSSV: `%s`) chưa có bản ghi kết quả học tập nào trong cơ sở dữ liệu.", user.FullName, user.StudentCode), nil, nil
		}

		var totalCredits float64
		var passedCredits float64
		var failedCount int
		for _, g := range grades {
			totalCredits += g.Credits
			if g.IsPassed {
				passedCredits += g.Credits
			} else {
				failedCount++
			}
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "# BẢNG ĐIỂM HỌC VỤ SINH VIÊN: %s (MSSV: `%s`)\n\n", user.FullName, user.StudentCode)
		fmt.Fprintf(&sb, "- **Tổng số học phần đã học**: %d\n", len(grades))
		fmt.Fprintf(&sb, "- **Tổng số tín chỉ đăng ký**: %.1f\n", totalCredits)
		fmt.Fprintf(&sb, "- **Số tín chỉ đạt**: %.1f\n", passedCredits)
		fmt.Fprintf(&sb, "- **Số học phần trượt / cần học lại**: %d\n\n", failedCount)

		fmt.Fprintf(&sb, "| Mã học phần | Tên học phần | Số TC | Điểm hệ 10 | Điểm hệ 4 | Điểm chữ | Trạng thái | Kỳ học |\n")
		sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")
		for _, g := range grades {
			statusStr := "Đạt"
			if !g.IsPassed {
				statusStr = "**Trượt**"
			}
			fmt.Fprintf(&sb, "| `%s` | %s | %.1f | %.2f | %.2f | %s | %s | %s |\n", g.CourseCode, g.CourseName, g.Credits, g.Score10, g.Score4, g.GradeLetter, statusStr, g.TermTaken)
		}
		return sb.String(), nil, nil

	case "list_academic_warnings":
		cohort := strings.TrimSpace(args.Cohort)
		major := strings.TrimSpace(args.Major)
		standing := strings.TrimSpace(args.AcademicStanding)
		if standing == "ALL" {
			standing = ""
		}
		limitVal := args.Limit
		if limitVal <= 0 || limitVal > 100 {
			limitVal = 20
		}

		if e.memoryRepo == nil {
			return "Dịch vụ hồ sơ sinh viên chưa sẵn sàng.", nil, nil
		}

		rows, err := e.memoryRepo.ListStudentProfilesByStanding(ctx, cohort, major, standing, int32(limitVal))
		if err != nil || len(rows) == 0 {
			return "Không tìm thấy sinh viên nào trong diện cảnh báo học tập theo tiêu chí đã chọn.", nil, nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "# DANH SÁCH SINH VIÊN TRONG DIỆN CẢNH BÁO HỌC TẬP (%d sinh viên):\n\n", len(rows))
		sb.WriteString("| MSSV | Họ và tên | Khóa | Ngành | GPA tích lũy | Tín chỉ đạt | Trạng thái | Ghi chú cố vấn |\n")
		sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")
		for _, r := range rows {
			fmt.Fprintf(&sb, "| `%s` | %s | %s | %s | %.2f | %d | `%s` | %s |\n",
				r.StudentCode.String, r.FullName.String, r.Cohort, r.Major, r.CumulativeGpa, r.CompletedCredits, r.AcademicStanding, r.AdvisorNotes)
		}
		return sb.String(), nil, nil

	case "propose_advisor_note":
		studentCode := strings.TrimSpace(args.StudentCode)
		userID := strings.TrimSpace(args.UserID)
		email := strings.TrimSpace(args.Email)
		advisorNote := strings.TrimSpace(args.AdvisorNote)

		if advisorNote == "" {
			return "Thiếu nội dung 'advisor_note' để cập nhật ghi chú cố vấn.", nil, nil
		}

		user, err := e.resolveTargetUser(ctx, studentCode, userID, email)
		if err != nil || user == nil {
			return fmt.Sprintf("Không tìm thấy sinh viên tương ứng với mã `%s` để cập nhật ghi chú.", studentCode), nil, nil
		}

		currentProfile, _ := e.memoryRepo.GetStudentProfile(ctx, user.ID)
		oldNote := ""
		if currentProfile != nil {
			oldNote = currentProfile.AdvisorNotes
		}

		diffItems := []models.TLUClawDiffItem{
			{
				EntityType: "STUDENT_ADVISOR_NOTE",
				Action:     "UPDATE",
				Key:        user.StudentCode,
				Name:       fmt.Sprintf("Ghi chú cố vấn: %s (MSSV: %s)", user.FullName, user.StudentCode),
				OldValue:   oldNote,
				NewValue:   advisorNote,
				Note:       fmt.Sprintf("Cập nhật ghi chú cố vấn học tập cho sinh viên %s", user.FullName),
			},
		}

		blastRadius := models.TLUClawBlastRadius{
			ImpactSummary: fmt.Sprintf("Cập nhật ghi chú cố vấn cho sinh viên %s (MSSV: %s)", user.FullName, user.StudentCode),
			RiskLevel:     "LOW",
		}

		batchID := "batch_" + uuid.NewString()[:8]
		nonce, err := crypto.GenerateRandomHex(24)
		if err != nil {
			nonce = uuid.NewString()
		}
		expiresAt := time.Now().Add(15 * time.Minute)

		diffJSON, _ := json.Marshal(diffItems)
		blastJSON, _ := json.Marshal(blastRadius)

		statePayload, _ := json.Marshal(request.TLUClawAdvisorNoteSnapshotState{
			UserID:      user.ID,
			StudentCode: user.StudentCode,
			AdvisorNote: advisorNote,
		})

		previousStateJSON := "{}"
		if currentProfile != nil {
			prevBytes, _ := json.Marshal(currentProfile)
			previousStateJSON = string(prevBytes)
		}

		snapshotID := "snp_" + uuid.NewString()[:8]
		snapshotParams := sqlc.CreateTLUClawSnapshotParams{
			ID:                snapshotID,
			BatchID:           batchID,
			SessionID:         sessionID,
			OperatorID:        adminID,
			ActionType:        "advisor_note",
			Summary:           fmt.Sprintf("Cập nhật ghi chú cố vấn học tập sinh viên %s (MSSV: %s)", user.FullName, user.StudentCode),
			DiffPreviewJson:   string(diffJSON),
			BlastRadiusJson:   string(blastJSON),
			PreviousStateJson: previousStateJSON,
			NewStateJson:      string(statePayload),
			Status:            "STAGED",
			ConfirmationNonce: nonce,
			NonceExpiresAt:    sql.NullTime{Time: expiresAt, Valid: true},
			AppliedAt:         sql.NullTime{Valid: false},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if _, err := e.tluclawRepo.CreateSnapshot(ctx, snapshotParams); err != nil {
			log.Warn().Err(err).Msg("tluclaw ExecuteTool: create advisor note snapshot failed")
			return "Không thể tạo bản ghi xem trước. Vui lòng thử lại sau.", nil, nil
		}

		card := &models.TLUClawPreviewCardData{
			SnapshotID:  snapshotID,
			BatchID:     batchID,
			ActionType:  "advisor_note",
			Summary:     snapshotParams.Summary,
			Nonce:       nonce,
			ExpiresAt:   expiresAt,
			DiffItems:   diffItems,
			BlastRadius: blastRadius,
		}

		msg := fmt.Sprintf("Đã tạo bản xem trước cập nhật ghi chú cố vấn cho sinh viên %s (MSSV: %s). Vui lòng xác nhận trên thẻ.", user.FullName, user.StudentCode)
		return msg, card.ToResponse(), nil

	default:
		return fmt.Sprintf("Hành động không hợp lệ: %s", action), nil, nil
	}
}

func (e *tluclawToolEngine) executeUserOperations(ctx context.Context, sessionID, adminID string, args *request.TLUClawUserOpArgs) (string, *response.TLUClawPreviewCardResponse, error) {
	action := args.Action

	switch action {
	case "search_users":
		searchText := strings.TrimSpace(args.SearchText)
		if searchText == "" {
			searchText = strings.TrimSpace(args.Keyword)
		}
		roleID := strings.TrimSpace(args.RoleID)
		limitVal := args.Limit
		if limitVal <= 0 || limitVal > 100 {
			limitVal = 20
		}
		pageVal := args.Page
		if pageVal <= 0 {
			pageVal = 1
		}
		offsetVal := int64((pageVal - 1) * limitVal)

		var roleIDNull sql.NullString
		if roleID != "" {
			roleIDNull = sql.NullString{String: roleID, Valid: true}
		}
		var searchTextNull sql.NullString
		if searchText != "" {
			searchTextNull = sql.NullString{String: searchText, Valid: true}
		}

		users, err := e.userRepo.Search(ctx, sqlc.SearchUserIDsParams{
			RoleID:     roleIDNull,
			SearchText: searchTextNull,
			Limit:      int64(limitVal),
			Offset:     offsetVal,
		})
		if err != nil || len(users) == 0 {
			return fmt.Sprintf("Không tìm thấy người dùng nào với từ khóa '%s'.", searchText), nil, nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "# KẾT QUẢ TÌM KIẾM NGƯỜI DÙNG (%d tài khoản, Trang %d):\n\n", len(users), pageVal)
		sb.WriteString("| ID | Họ và tên | Email | MSSV | Vai trò | Trạng thái |\n")
		sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- |\n")
		for _, u := range users {
			var roleNames []string
			for _, r := range u.Roles {
				roleNames = append(roleNames, r.Name)
			}
			statusStr := "ACTIVE"
			if u.IsDeleted {
				statusStr = "DELETED"
			} else if u.IsBanned() {
				statusStr = "BANNED"
			}
			sb.WriteString(fmt.Sprintf("| `%s` | %s | %s | `%s` | %s | `%s` |\n",
				u.ID, u.FullName, u.Email, u.StudentCode, strings.Join(roleNames, ", "), statusStr))
		}
		return sb.String(), nil, nil

	case "inspect_user_detail":
		studentCode := strings.TrimSpace(args.StudentCode)
		userID := strings.TrimSpace(args.UserID)
		email := strings.TrimSpace(args.Email)

		user, err := e.resolveTargetUser(ctx, studentCode, userID, email)
		if err != nil || user == nil {
			return "Không tìm thấy người dùng phù hợp.", nil, nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# CHI TIẾT TÀI KHOẢN NGƯỜI DÙNG: %s\n\n", user.FullName))
		fmt.Fprintf(&sb, "- **User ID**: `%s`\n", user.ID)
		fmt.Fprintf(&sb, "- **Email**: %s\n", user.Email)
		fmt.Fprintf(&sb, "- **MSSV**: `%s`\n", user.StudentCode)
		fmt.Fprintf(&sb, "- **Auth Provider**: %s\n", user.AuthProvider)
		fmt.Fprintf(&sb, "- **Bị xóa (is_deleted)**: %v\n", user.IsDeleted)
		fmt.Fprintf(&sb, "- **Bị cấm (is_banned)**: %v\n", user.IsBanned())

		sb.WriteString("\n### Danh sách Vai trò (Roles):\n")
		if len(user.Roles) == 0 {
			sb.WriteString("- *Chưa được gán vai trò nào.*\n")
		} else {
			for _, r := range user.Roles {
				fmt.Fprintf(&sb, "- **%s** (ID: `%s`): %d quyền hạn\n", r.Name, r.ID, len(r.Permissions))
			}
		}
		return sb.String(), nil, nil

	case "list_system_roles":
		if e.roleRepo == nil {
			return "Dịch vụ vai trò người dùng chưa sẵn sàng.", nil, nil
		}
		roles, err := e.roleRepo.All(ctx)
		if err != nil || len(roles) == 0 {
			return "Không thể tải danh sách vai trò hệ thống.", nil, nil
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "# DANH SÁCH VAI TRÒ HỆ THỐNG (%d vai trò):\n\n", len(roles))
		sb.WriteString("| Tên vai trò | Mô tả | Vai trò hệ thống | Vị trí ưu tiên |\n")
		sb.WriteString("| :--- | :--- | :--- | :--- |\n")
		for _, r := range roles {
			isSys := "Tùy biến"
			if r.IsSystem {
				isSys = "Hệ thống"
			}
			fmt.Fprintf(&sb, "| **%s** | %s | %s | %d |\n", r.Name, r.Description, isSys, r.Position)
		}
		return sb.String(), nil, nil

	case "list_system_permissions":
		if e.roleRepo == nil {
			return "Dịch vụ quyền hạn hệ thống chưa sẵn sàng.", nil, nil
		}
		perms, err := e.roleRepo.ListPermissions(ctx)
		if err != nil || len(perms) == 0 {
			return "Không thể tải danh sách quyền hạn hệ thống.", nil, nil
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "# DANH SÁCH QUYỀN HẠN HỆ THỐNG (%d quyền):\n\n", len(perms))
		sb.WriteString("| Mã quyền (Permission Key) | Mô tả quyền hạn |\n")
		sb.WriteString("| :--- | :--- |\n")
		for _, p := range perms {
			fmt.Fprintf(&sb, "| `%s` | %s |\n", p.Key, p.Description)
		}
		return sb.String(), nil, nil

	case "propose_user_status":
		studentCode := strings.TrimSpace(args.StudentCode)
		userID := strings.TrimSpace(args.UserID)
		email := strings.TrimSpace(args.Email)
		status := strings.TrimSpace(args.Status)
		reason := strings.TrimSpace(args.Reason)

		if status != "ACTIVE" && status != "BANNED" {
			return "Trạng thái không hợp lệ. Phải là 'ACTIVE' hoặc 'BANNED'.", nil, nil
		}

		user, err := e.resolveTargetUser(ctx, studentCode, userID, email)
		if err != nil || user == nil {
			return "Không tìm thấy người dùng để thay đổi trạng thái.", nil, nil
		}

		if user.ID == adminID {
			return "Quản trị viên không thể tự thay đổi trạng thái của chính mình.", nil, nil
		}

		currentStatus := "ACTIVE"
		if user.IsBanned() {
			currentStatus = "BANNED"
		}

		diffItems := []models.TLUClawDiffItem{
			{
				EntityType: "USER_STATUS",
				Action:     "UPDATE",
				Key:        user.Email,
				Name:       fmt.Sprintf("Trạng thái tài khoản: %s (%s)", user.FullName, user.Email),
				OldValue:   currentStatus,
				NewValue:   status,
				Note:       fmt.Sprintf("Lý do: %s", reason),
			},
		}

		blastRadius := models.TLUClawBlastRadius{
			ImpactSummary: fmt.Sprintf("Thay đổi trạng thái tài khoản %s sang %s (Lý do: %s)", user.Email, status, reason),
			RiskLevel:     "HIGH",
		}

		batchID := "batch_" + uuid.NewString()[:8]
		nonce, err := crypto.GenerateRandomHex(24)
		if err != nil {
			nonce = uuid.NewString()
		}
		expiresAt := time.Now().Add(15 * time.Minute)

		diffJSON, _ := json.Marshal(diffItems)
		blastJSON, _ := json.Marshal(blastRadius)

		statePayload, _ := json.Marshal(request.TLUClawUserStatusSnapshotState{
			UserID: user.ID,
			Email:  user.Email,
			Status: status,
			Reason: reason,
		})

		snapshotID := "snp_" + uuid.NewString()[:8]
		snapshotParams := sqlc.CreateTLUClawSnapshotParams{
			ID:                snapshotID,
			BatchID:           batchID,
			SessionID:         sessionID,
			OperatorID:        adminID,
			ActionType:        "user_status",
			Summary:           fmt.Sprintf("Cập nhật trạng thái tài khoản %s -> %s", user.Email, status),
			DiffPreviewJson:   string(diffJSON),
			BlastRadiusJson:   string(blastJSON),
			PreviousStateJson: fmt.Sprintf(`{"user_id":"%s","status":"%s"}`, user.ID, currentStatus),
			NewStateJson:      string(statePayload),
			Status:            "STAGED",
			ConfirmationNonce: nonce,
			NonceExpiresAt:    sql.NullTime{Time: expiresAt, Valid: true},
			AppliedAt:         sql.NullTime{Valid: false},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if _, err := e.tluclawRepo.CreateSnapshot(ctx, snapshotParams); err != nil {
			log.Warn().Err(err).Msg("tluclaw ExecuteTool: create user status snapshot failed")
			return "Không thể tạo bản ghi xem trước. Vui lòng thử lại sau.", nil, nil
		}

		card := &models.TLUClawPreviewCardData{
			SnapshotID:  snapshotID,
			BatchID:     batchID,
			ActionType:  "user_status",
			Summary:     snapshotParams.Summary,
			Nonce:       nonce,
			ExpiresAt:   expiresAt,
			DiffItems:   diffItems,
			BlastRadius: blastRadius,
		}

		msg := fmt.Sprintf("Đã tạo bản xem trước thay đổi trạng thái tài khoản %s sang %s. Vui lòng kiểm tra và xác nhận.", user.Email, status)
		return msg, card.ToResponse(), nil

	case "propose_user_role":
		studentCode := strings.TrimSpace(args.StudentCode)
		userID := strings.TrimSpace(args.UserID)
		email := strings.TrimSpace(args.Email)
		roleAction := strings.TrimSpace(args.RoleAction)
		roleName := strings.TrimSpace(args.RoleName)

		if roleAction != "ASSIGN" && roleAction != "REMOVE" {
			return "Hành động vai trò không hợp lệ. Phải là 'ASSIGN' hoặc 'REMOVE'.", nil, nil
		}
		if roleName == "" {
			return "Thiếu tên vai trò 'role_name'.", nil, nil
		}

		user, err := e.resolveTargetUser(ctx, studentCode, userID, email)
		if err != nil || user == nil {
			return "Không tìm thấy người dùng để điều chỉnh vai trò.", nil, nil
		}

		if user.ID == adminID && roleAction == "REMOVE" && roleName == "ADMIN" {
			return "Quản trị viên không thể tự gỡ bỏ vai trò ADMIN của chính mình.", nil, nil
		}

		targetRole, err := e.roleRepo.GetByName(ctx, roleName)
		if err != nil || targetRole == nil {
			return fmt.Sprintf("Vai trò '%s' không tồn tại trong hệ thống.", roleName), nil, nil
		}

		var curRoles []string
		for _, r := range user.Roles {
			curRoles = append(curRoles, r.Name)
		}

		diffItems := []models.TLUClawDiffItem{
			{
				EntityType: "USER_ROLE",
				Action:     roleAction,
				Key:        user.Email,
				Name:       fmt.Sprintf("Phân quyền tài khoản: %s (%s)", user.FullName, user.Email),
				OldValue:   strings.Join(curRoles, ", "),
				NewValue:   fmt.Sprintf("%s vai trò '%s'", roleAction, roleName),
				Note:       fmt.Sprintf("Thực hiện %s vai trò %s cho tài khoản %s", roleAction, roleName, user.Email),
			},
		}

		blastRadius := models.TLUClawBlastRadius{
			ImpactSummary: fmt.Sprintf("%s vai trò %s cho tài khoản %s (ID: %s)", roleAction, roleName, user.Email, user.ID),
			RiskLevel:     "HIGH",
		}

		batchID := "batch_" + uuid.NewString()[:8]
		nonce, err := crypto.GenerateRandomHex(24)
		if err != nil {
			nonce = uuid.NewString()
		}
		expiresAt := time.Now().Add(15 * time.Minute)

		diffJSON, _ := json.Marshal(diffItems)
		blastJSON, _ := json.Marshal(blastRadius)

		statePayload, _ := json.Marshal(request.TLUClawUserRoleChangeSnapshotState{
			UserID:     user.ID,
			Email:      user.Email,
			RoleAction: roleAction,
			RoleID:     targetRole.ID,
			RoleName:   targetRole.Name,
		})

		snapshotID := "snp_" + uuid.NewString()[:8]
		snapshotParams := sqlc.CreateTLUClawSnapshotParams{
			ID:                snapshotID,
			BatchID:           batchID,
			SessionID:         sessionID,
			OperatorID:        adminID,
			ActionType:        "user_role",
			Summary:           fmt.Sprintf("Đề xuất %s vai trò %s cho tài khoản %s", roleAction, roleName, user.Email),
			DiffPreviewJson:   string(diffJSON),
			BlastRadiusJson:   string(blastJSON),
			PreviousStateJson: fmt.Sprintf(`{"user_id":"%s","roles":%q}`, user.ID, curRoles),
			NewStateJson:      string(statePayload),
			Status:            "STAGED",
			ConfirmationNonce: nonce,
			NonceExpiresAt:    sql.NullTime{Time: expiresAt, Valid: true},
			AppliedAt:         sql.NullTime{Valid: false},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if _, err := e.tluclawRepo.CreateSnapshot(ctx, snapshotParams); err != nil {
			log.Warn().Err(err).Msg("tluclaw ExecuteTool: create user role snapshot failed")
			return "Không thể tạo bản ghi xem trước. Vui lòng thử lại sau.", nil, nil
		}

		card := &models.TLUClawPreviewCardData{
			SnapshotID:  snapshotID,
			BatchID:     batchID,
			ActionType:  "user_role",
			Summary:     snapshotParams.Summary,
			Nonce:       nonce,
			ExpiresAt:   expiresAt,
			DiffItems:   diffItems,
			BlastRadius: blastRadius,
		}

		msg := fmt.Sprintf("Đã tạo bản xem trước %s vai trò %s cho tài khoản %s. Vui lòng xác nhận trên thẻ.", roleAction, roleName, user.Email)
		return msg, card.ToResponse(), nil

	default:
		return fmt.Sprintf("Hành động không hợp lệ: %s", action), nil, nil
	}
}

func (e *tluclawToolEngine) executeInquiryOperations(ctx context.Context, sessionID, adminID string, args *request.TLUClawInquiryOpArgs) (string, *response.TLUClawPreviewCardResponse, error) {
	action := args.Action

	switch action {
	case "list_pending_inquiries":
		status := strings.TrimSpace(args.Status)
		if status == "ALL" {
			status = ""
		}
		search := strings.TrimSpace(args.Search)
		limitVal := args.Limit
		if limitVal <= 0 || limitVal > 100 {
			limitVal = 20
		}
		pageVal := args.Page
		if pageVal <= 0 {
			pageVal = 1
		}
		offsetVal := (pageVal - 1) * limitVal

		if e.inquiryRepo == nil {
			return "Dịch vụ câu hỏi tư vấn chưa sẵn sàng.", nil, nil
		}

		inqs, _, err := e.inquiryRepo.ListInquiries(ctx, status, search, limitVal, offsetVal)
		if err != nil || len(inqs) == 0 {
			return "Không có câu hỏi tư vấn nào phù hợp với bộ lọc.", nil, nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# DANH SÁCH CÂU HỎI TƯ VẤN HỌC VỤ (%d câu hỏi, Trang %d):\n\n", len(inqs), pageVal))
		sb.WriteString("| ID | Sinh viên | MSSV | Lớp | Trạng thái | Câu hỏi | Ngày gửi |\n")
		sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")
		for _, inq := range inqs {
			qShort := inq.Question
			if len([]rune(qShort)) > 40 {
				qShort = string([]rune(qShort)[:40]) + "..."
			}
			createdStr := inq.CreatedAt.Format("02/01/2006 15:04")
			sb.WriteString(fmt.Sprintf("| `%s` | %s | `%s` | %s | `%s` | %s | %s |\n",
				inq.ID, inq.StudentName, inq.StudentCode, inq.StudentClass, inq.Status, qShort, createdStr))
		}
		return sb.String(), nil, nil

	case "inspect_inquiry_detail":
		inquiryID := strings.TrimSpace(args.InquiryID)
		if inquiryID == "" {
			return "Thiếu tham số bắt buộc 'inquiry_id'", nil, nil
		}

		inq, err := e.inquiryRepo.GetInquiryByID(ctx, inquiryID)
		if err != nil || inq == nil {
			return fmt.Sprintf("Không tìm thấy câu hỏi tư vấn với mã '%s'.", inquiryID), nil, nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# CHI TIẾT CÂU HỎI TƯ VẤN HỌC VỤ `%s`:\n\n", inq.ID))
		sb.WriteString(fmt.Sprintf("- **Sinh viên**: %s (MSSV: `%s`, Lớp: `%s`)\n", inq.StudentName, inq.StudentCode, inq.StudentClass))
		sb.WriteString(fmt.Sprintf("- **Trạng thái**: `%s`\n", inq.Status))
		sb.WriteString(fmt.Sprintf("- **Ngày gửi**: %s\n\n", inq.CreatedAt.Format("02/01/2006 15:04:05")))
		sb.WriteString(fmt.Sprintf("### Câu hỏi của sinh viên:\n> %s\n\n", inq.Question))
		if inq.Context != "" {
			sb.WriteString(fmt.Sprintf("### Ngữ cảnh đoạn hội thoại liên quan:\n```\n%s\n```\n\n", inq.Context))
		}
		if inq.Status == "ANSWERED" {
			sb.WriteString(fmt.Sprintf("### Trả lời của Cố vấn / Giảng viên (%s):\n%s\n", inq.TeacherName, inq.TeacherReply))
		}
		return sb.String(), nil, nil

	case "propose_reply_inquiry":
		inquiryID := strings.TrimSpace(args.InquiryID)
		reply := strings.TrimSpace(args.Reply)
		if reply == "" {
			reply = strings.TrimSpace(args.Answer)
		}
		chunkID := strings.TrimSpace(args.ChunkID)

		if inquiryID == "" || reply == "" {
			return "Thiếu thông tin bắt buộc: inquiry_id hoặc reply", nil, nil
		}

		inq, err := e.inquiryRepo.GetInquiryByID(ctx, inquiryID)
		if err != nil || inq == nil {
			return fmt.Sprintf("Không tìm thấy câu hỏi tư vấn với mã '%s'.", inquiryID), nil, nil
		}

		diffItems := []models.TLUClawDiffItem{
			{
				EntityType: "INQUIRY_REPLY",
				Action:     "ANSWER",
				Key:        inquiryID,
				Name:       fmt.Sprintf("Trả lời câu hỏi của sinh viên %s (%s)", inq.StudentName, inq.StudentCode),
				OldValue:   inq.TeacherReply,
				NewValue:   reply,
				Note:       fmt.Sprintf("Câu hỏi: %s", inq.Question),
			},
		}

		blastRadius := models.TLUClawBlastRadius{
			ImpactSummary: fmt.Sprintf("Trả lời thắc mắc học vụ cho sinh viên %s (MSSV: %s)", inq.StudentName, inq.StudentCode),
			RiskLevel:     "LOW",
		}

		batchID := "batch_" + uuid.NewString()[:8]
		nonce, err := crypto.GenerateRandomHex(24)
		if err != nil {
			nonce = uuid.NewString()
		}
		expiresAt := time.Now().Add(15 * time.Minute)

		diffJSON, _ := json.Marshal(diffItems)
		blastJSON, _ := json.Marshal(blastRadius)

		statePayload, _ := json.Marshal(request.TLUClawInquiryReplySnapshotState{
			InquiryID: inquiryID,
			Reply:     reply,
			ChunkID:   chunkID,
		})

		snapshotID := "snp_" + uuid.NewString()[:8]
		snapshotParams := sqlc.CreateTLUClawSnapshotParams{
			ID:                snapshotID,
			BatchID:           batchID,
			SessionID:         sessionID,
			OperatorID:        adminID,
			ActionType:        "inquiry_reply",
			Summary:           fmt.Sprintf("Đề xuất câu trả lời tư vấn cho sinh viên %s (%s)", inq.StudentName, inq.StudentCode),
			DiffPreviewJson:   string(diffJSON),
			BlastRadiusJson:   string(blastJSON),
			PreviousStateJson: fmt.Sprintf(`{"inquiry_id":"%s","status":"%s"}`, inquiryID, inq.Status),
			NewStateJson:      string(statePayload),
			Status:            "STAGED",
			ConfirmationNonce: nonce,
			NonceExpiresAt:    sql.NullTime{Time: expiresAt, Valid: true},
			AppliedAt:         sql.NullTime{Valid: false},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if _, err := e.tluclawRepo.CreateSnapshot(ctx, snapshotParams); err != nil {
			log.Warn().Err(err).Msg("tluclaw ExecuteTool: create inquiry reply snapshot failed")
			return "Không thể tạo bản ghi xem trước. Vui lòng thử lại sau.", nil, nil
		}

		card := &models.TLUClawPreviewCardData{
			SnapshotID:  snapshotID,
			BatchID:     batchID,
			ActionType:  "inquiry_reply",
			Summary:     snapshotParams.Summary,
			Nonce:       nonce,
			ExpiresAt:   expiresAt,
			DiffItems:   diffItems,
			BlastRadius: blastRadius,
		}

		msg := fmt.Sprintf("Đã tạo bản xem trước câu trả lời tư vấn cho sinh viên %s (%s). Vui lòng kiểm tra và xác nhận trên thẻ.", inq.StudentName, inq.StudentCode)
		return msg, card.ToResponse(), nil

	default:
		return fmt.Sprintf("Hành động không hợp lệ: %s", action), nil, nil
	}
}

func (e *tluclawToolEngine) executeSystemOperations(ctx context.Context, sessionID, adminID string, args *request.TLUClawSystemOpArgs) (string, *response.TLUClawPreviewCardResponse, error) {
	action := args.Action

	switch action {
	case "inspect_system_stats":
		var userCount int64
		if e.userRepo != nil {
			userCount, _ = e.userRepo.Count(ctx, sqlc.CountUsersParams{})
		}
		var currCount int64
		var courseCount int64
		var subCount int64
		if e.curriculumRepo != nil {
			currCount, _ = e.curriculumRepo.CountCurriculums(ctx)
			courseCount, _ = e.curriculumRepo.CountCurriculumCourses(ctx)
			subCount, _ = e.curriculumRepo.CountSubstitutions(ctx)
		}
		var docCount int64
		var chunkCount int64
		if e.ragRepo != nil {
			docs, chunks, _, _, _ := e.ragRepo.CountStats(ctx)
			docCount = int64(docs)
			chunkCount = int64(chunks)
		}

		var sb strings.Builder
		sb.WriteString("# THỐNG KÊ TOÀN CẢNH HỆ THỐNG TLAGENT:\n\n")
		sb.WriteString("| Phân hệ dữ liệu | Số lượng bản ghi | Ghi chú |\n")
		sb.WriteString("| :--- | :--- | :--- |\n")
		sb.WriteString(fmt.Sprintf("| **Tài khoản người dùng** | %d | Bao gồm Quản trị viên, Giảng viên, Sinh viên |\n", userCount))
		sb.WriteString(fmt.Sprintf("| **Chương trình đào tạo (Curriculums)** | %d | Các khóa K24 -> K39 |\n", currCount))
		sb.WriteString(fmt.Sprintf("| **Học phần đào tạo (Courses)** | %d | Môn học trong toàn trường |\n", courseCount))
		sb.WriteString(fmt.Sprintf("| **Quy tắc môn thay thế (Substitutions)** | %d | Quy tắc công nhận và thay thế tín chỉ |\n", subCount))
		sb.WriteString(fmt.Sprintf("| **Văn bản quy chế (Regulations)** | %d | Quy chế, quyết định đã nạp vào RAG |\n", docCount))
		sb.WriteString(fmt.Sprintf("| **Đoạn tri thức (Knowledge Chunks)** | %d | Đã lập chỉ mục ngữ nghĩa Vector & FTS |\n", chunkCount))
		return sb.String(), nil, nil

	case "flush_system_cache":
		scope := strings.TrimSpace(args.Scope)
		if scope == "" {
			scope = "all"
		}

		if e.c == nil {
			return "Bộ nhớ đệm RAM Cache chưa được khởi tạo.", nil, nil
		}

		switch scope {
		case "curriculum":
			_ = e.c.DelByPattern(ctx, "curr*")
			_ = e.c.DelByPattern(ctx, "course*")
		case "rag":
			_ = e.c.DelByPattern(ctx, "rag*")
		case "users":
			_ = e.c.DelByPattern(ctx, "user*")
		case "transcripts":
			_ = e.c.DelByPattern(ctx, "trans*")
		case "all":
			_ = e.c.DelByPattern(ctx, "curr*")
			_ = e.c.DelByPattern(ctx, "course*")
			_ = e.c.DelByPattern(ctx, "rag*")
			_ = e.c.DelByPattern(ctx, "user*")
			_ = e.c.DelByPattern(ctx, "trans*")
			_ = e.c.DelByPattern(ctx, "role*")
		default:
			return fmt.Sprintf("Phạm vi xóa cache không hợp lệ: '%s'. Các phạm vi hợp lệ: all, curriculum, rag, users, transcripts.", scope), nil, nil
		}

		return fmt.Sprintf("Đã làm sạch toàn bộ bộ nhớ đệm RAM Cache cho phân vùng: `%s` thành công.", scope), nil, nil

	case "inspect_llm_status":
		if e.llmConfigSvc == nil {
			return "Dịch vụ cấu hình LLM chưa sẵn sàng.", nil, nil
		}

		providers, err := e.llmConfigSvc.ListProviders(ctx)
		if err != nil {
			return fmt.Sprintf("Lỗi tra cứu nhà cung cấp LLM: %v", err), nil, nil
		}
		modelsList, _ := e.llmConfigSvc.ListModels(ctx, "")

		var sb strings.Builder
		sb.WriteString("# TRẠNG THÁI HỆ THỐNG MÔ HÌNH NGÔN NGỮ LỚN (LLM ORCHESTRATION):\n\n")
		sb.WriteString("### Danh sách Nhà cung cấp AI (Providers):\n")
		sb.WriteString("| Tên nhà cung cấp | Loại Provider | Trạng thái | Mặc định |\n")
		sb.WriteString("| :--- | :--- | :--- | :--- |\n")
		for _, p := range providers {
			activeStr := "Tắt"
			if p.IsActive {
				activeStr = "Đang chạy"
			}
			defaultStr := "Không"
			if p.IsDefault {
				defaultStr = "**Mặc định**"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", p.Name, p.ProviderType, activeStr, defaultStr))
		}

		sb.WriteString("\n### Danh sách Mô hình AI (Models):\n")
		sb.WriteString("| ID | Tên mô hình | Model Key | Loại | Hoạt động | Vision Mode | Mặc định |\n")
		sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- |\n")
		for _, m := range modelsList {
			activeStr := "Tắt"
			if m.IsActive {
				activeStr = "Bật"
			}
			defaultStr := "Không"
			if m.IsDefault {
				defaultStr = "**Mặc định**"
			}
			sb.WriteString(fmt.Sprintf("| `%s` | %s | `%s` | %s | %s | %s | %s |\n",
				m.ID, m.Name, m.ModelKey, m.ModelType, activeStr, m.VisionMode, defaultStr))
		}
		return sb.String(), nil, nil

	case "propose_switch_model":
		modelID := strings.TrimSpace(args.ModelID)
		modelType := strings.TrimSpace(args.ModelType)
		if modelID == "" {
			return "Thiếu tham số bắt buộc 'model_id'.", nil, nil
		}
		if modelType == "" {
			modelType = "CHAT"
		}

		if e.llmConfigSvc == nil {
			return "Dịch vụ cấu hình LLM chưa sẵn sàng.", nil, nil
		}

		modelsList, err := e.llmConfigSvc.ListModels(ctx, "")
		if err != nil {
			return fmt.Sprintf("Lỗi tra cứu mô hình: %v", err), nil, nil
		}

		var targetModelName string
		var currentDefaultName string
		for _, m := range modelsList {
			if m.IsDefault && m.ModelType == modelType {
				currentDefaultName = fmt.Sprintf("%s (%s)", m.Name, m.ModelKey)
			}
			if m.ID == modelID {
				targetModelName = fmt.Sprintf("%s (%s)", m.Name, m.ModelKey)
			}
		}

		if targetModelName == "" {
			return fmt.Sprintf("Không tìm thấy mô hình với ID '%s'.", modelID), nil, nil
		}

		diffItems := []models.TLUClawDiffItem{
			{
				EntityType: "LLM_MODEL",
				Action:     "SET_DEFAULT",
				Key:        modelID,
				Name:       fmt.Sprintf("Mô hình AI mặc định (%s)", modelType),
				OldValue:   currentDefaultName,
				NewValue:   targetModelName,
				Note:       fmt.Sprintf("Chuyển đổi mô hình %s mặc định toàn hệ thống sang %s", modelType, targetModelName),
			},
		}

		blastRadius := models.TLUClawBlastRadius{
			ImpactSummary: fmt.Sprintf("Chuyển đổi mô hình AI %s mặc định toàn trường sang: %s", modelType, targetModelName),
			RiskLevel:     "HIGH",
		}

		batchID := "batch_" + uuid.NewString()[:8]
		nonce, err := crypto.GenerateRandomHex(24)
		if err != nil {
			nonce = uuid.NewString()
		}
		expiresAt := time.Now().Add(15 * time.Minute)

		diffJSON, _ := json.Marshal(diffItems)
		blastJSON, _ := json.Marshal(blastRadius)

		statePayload, _ := json.Marshal(request.TLUClawLLMConfigSnapshotState{
			ModelID:   modelID,
			ModelType: modelType,
		})

		snapshotID := "snp_" + uuid.NewString()[:8]
		snapshotParams := sqlc.CreateTLUClawSnapshotParams{
			ID:                snapshotID,
			BatchID:           batchID,
			SessionID:         sessionID,
			OperatorID:        adminID,
			ActionType:        "llm_config",
			Summary:           fmt.Sprintf("Thiết lập mô hình AI %s mặc định: %s", modelType, targetModelName),
			DiffPreviewJson:   string(diffJSON),
			BlastRadiusJson:   string(blastJSON),
			PreviousStateJson: fmt.Sprintf(`{"model_type":"%s","default_model":"%s"}`, modelType, currentDefaultName),
			NewStateJson:      string(statePayload),
			Status:            "STAGED",
			ConfirmationNonce: nonce,
			NonceExpiresAt:    sql.NullTime{Time: expiresAt, Valid: true},
			AppliedAt:         sql.NullTime{Valid: false},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if _, err := e.tluclawRepo.CreateSnapshot(ctx, snapshotParams); err != nil {
			log.Warn().Err(err).Msg("tluclaw ExecuteTool: create llm config snapshot failed")
			return "Không thể tạo bản ghi xem trước. Vui lòng thử lại sau.", nil, nil
		}

		card := &models.TLUClawPreviewCardData{
			SnapshotID:  snapshotID,
			BatchID:     batchID,
			ActionType:  "llm_config",
			Summary:     snapshotParams.Summary,
			Nonce:       nonce,
			ExpiresAt:   expiresAt,
			DiffItems:   diffItems,
			BlastRadius: blastRadius,
		}

		msg := fmt.Sprintf("Đã tạo bản xem trước thiết lập mô hình AI %s mặc định sang %s. Vui lòng xác nhận trên thẻ.", modelType, targetModelName)
		return msg, card.ToResponse(), nil

	default:
		return fmt.Sprintf("Hành động không hợp lệ: %s", action), nil, nil
	}
}

func (e *tluclawToolEngine) executeNotificationOperations(ctx context.Context, sessionID, adminID string, args *request.TLUClawNotificationOpArgs) (string, *response.TLUClawPreviewCardResponse, error) {
	if e.notifRepo == nil {
		return "Dịch vụ thông báo người dùng chưa sẵn sàng.", nil, nil
	}
	action := args.Action

	switch action {
	case "list_notifications":
		studentCode := strings.TrimSpace(args.StudentCode)
		userID := strings.TrimSpace(args.UserID)
		if userID == "" {
			userID = strings.TrimSpace(args.RecipientUserID)
		}
		limitVal := args.Limit
		if limitVal <= 0 || limitVal > 100 {
			limitVal = 20
		}
		pageVal := args.Page
		if pageVal <= 0 {
			pageVal = 1
		}
		offsetVal := (pageVal - 1) * limitVal

		var notifs []*models.UserNotification
		var err error
		if studentCode != "" || userID != "" {
			user, _ := e.resolveTargetUser(ctx, studentCode, userID, "")
			if user != nil {
				notifs, err = e.notifRepo.ListNotificationsByUserID(ctx, user.ID, limitVal, offsetVal)
			}
		} else {
			notifs, err = e.notifRepo.ListRecentNotifications(ctx, limitVal, offsetVal)
		}

		if err != nil || len(notifs) == 0 {
			return "Không có thông báo nào trong danh sách.", nil, nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# DANH SÁCH THÔNG BÁO HỆ THỐNG (%d thông báo, Trang %d):\n\n", len(notifs), pageVal))
		sb.WriteString("| ID | User ID | Tiêu đề | Loại | Đã đọc | Thời gian tạo |\n")
		sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- |\n")
		for _, n := range notifs {
			readStatus := "Chưa đọc"
			if n.IsRead {
				readStatus = "Đã đọc"
			}
			sb.WriteString(fmt.Sprintf("| `%s` | `%s` | %s | `%s` | %s | %s |\n",
				n.ID, n.UserID, n.Title, n.Type, readStatus, n.CreatedAt.Format("15:04 02/01/2006")))
		}
		return sb.String(), nil, nil

	case "propose_notification":
		targetType := strings.TrimSpace(args.TargetType)
		studentCode := strings.TrimSpace(args.StudentCode)
		userID := strings.TrimSpace(args.UserID)
		if userID == "" {
			userID = strings.TrimSpace(args.RecipientUserID)
		}
		cohort := strings.TrimSpace(args.Cohort)
		title := strings.TrimSpace(args.Title)
		content := strings.TrimSpace(args.Content)
		notifType := strings.TrimSpace(args.NotificationType)
		if notifType == "" {
			notifType = strings.TrimSpace(args.Type)
		}
		if notifType == "" {
			notifType = "SYSTEM"
		}

		if title == "" || content == "" {
			return "Vui lòng cung cấp đầy đủ 'title' và 'content' cho thông báo.", nil, nil
		}
		if targetType == "" {
			if studentCode != "" || userID != "" {
				targetType = "STUDENT"
			} else if cohort != "" {
				targetType = "COHORT"
			} else {
				targetType = "ALL"
			}
		}

		dispatchLimit := args.Limit
		if dispatchLimit <= 0 || dispatchLimit > 500 {
			dispatchLimit = 200
		}

		var items []request.TLUClawNotificationItem
		var targetDesc string
		var impactLevel string = "LOW"

		switch targetType {
		case "STUDENT":
			user, err := e.resolveTargetUser(ctx, studentCode, userID, "")
			if err != nil || user == nil {
				return fmt.Sprintf("Không tìm thấy sinh viên/người dùng tương ứng (MSSV: %s, UserID: %s).", studentCode, userID), nil, nil
			}
			items = append(items, request.TLUClawNotificationItem{
				ID:      uuid.NewString(),
				UserID:  user.ID,
				Title:   title,
				Content: content,
				Type:    notifType,
			})
			targetDesc = fmt.Sprintf("Sinh viên %s (MSSV: %s, UserID: %s)", user.FullName, user.StudentCode, user.ID)
			impactLevel = "LOW"

		case "COHORT":
			if cohort == "" {
				return "Vui lòng cung cấp khóa sinh viên (ví dụ: K36, K37) khi gửi thông báo theo khóa.", nil, nil
			}
			if e.memoryRepo == nil {
				return "Dịch vụ hồ sơ sinh viên chưa sẵn sàng.", nil, nil
			}
			profiles, err := e.memoryRepo.ListStudentProfilesByStanding(ctx, cohort, "", "", int32(dispatchLimit))
			if err != nil || len(profiles) == 0 {
				return fmt.Sprintf("Không tìm thấy sinh viên nào thuộc khóa %s.", cohort), nil, nil
			}
			for _, p := range profiles {
				items = append(items, request.TLUClawNotificationItem{
					ID:      uuid.NewString(),
					UserID:  p.UserID,
					Title:   title,
					Content: content,
					Type:    notifType,
				})
			}
			targetDesc = fmt.Sprintf("Toàn bộ %d sinh viên khóa %s", len(items), cohort)
			impactLevel = "MEDIUM"

		case "ALL":
			if e.userRepo == nil {
				return "Dịch vụ người dùng chưa sẵn sàng.", nil, nil
			}
			users, err := e.userRepo.Search(ctx, sqlc.SearchUserIDsParams{Limit: int64(dispatchLimit), Offset: 0})
			if err != nil || len(users) == 0 {
				return "Không có người dùng nào trong hệ thống.", nil, nil
			}
			for _, u := range users {
				if !u.IsDeleted && !u.IsBanned() {
					items = append(items, request.TLUClawNotificationItem{
						ID:      uuid.NewString(),
						UserID:  u.ID,
						Title:   title,
						Content: content,
						Type:    notifType,
					})
				}
			}
			targetDesc = fmt.Sprintf("Tất cả người dùng hệ thống (%d tài khoản)", len(items))
			impactLevel = "HIGH"

		default:
			return fmt.Sprintf("Loại đối tượng nhận không hợp lệ: %s", targetType), nil, nil
		}

		diffItems := []models.TLUClawDiffItem{
			{
				EntityType: "NOTIFICATION",
				Action:     "DISPATCH",
				Key:        targetType,
				Name:       title,
				OldValue:   "-",
				NewValue:   fmt.Sprintf("%d người nhận", len(items)),
				Note:       fmt.Sprintf("Nội dung: %s | Loại: %s", content, notifType),
			},
		}

		blastRadius := models.TLUClawBlastRadius{
			ImpactSummary: fmt.Sprintf("Phát thông báo tới %s (%d người nhận)", targetDesc, len(items)),
			RiskLevel:     impactLevel,
		}

		batchID := "batch_notif_" + uuid.NewString()[:8]
		nonce, _ := crypto.GenerateRandomHex(24)
		expiresAt := time.Now().Add(15 * time.Minute)

		newStatePayload, _ := json.Marshal(request.TLUClawNotificationDispatchSnapshotState{
			Notifications: items,
		})

		diffJSON, _ := json.Marshal(diffItems)
		blastJSON, _ := json.Marshal(blastRadius)

		snapshotID := "snp_" + uuid.NewString()[:8]
		snapshotParams := sqlc.CreateTLUClawSnapshotParams{
			ID:                snapshotID,
			BatchID:           batchID,
			SessionID:         sessionID,
			OperatorID:        adminID,
			ActionType:        "notification_dispatch",
			Summary:           fmt.Sprintf("Gửi thông báo '%s' đến %s", title, targetDesc),
			DiffPreviewJson:   string(diffJSON),
			BlastRadiusJson:   string(blastJSON),
			PreviousStateJson: "{}",
			NewStateJson:      string(newStatePayload),
			Status:            "STAGED",
			ConfirmationNonce: nonce,
			NonceExpiresAt:    sql.NullTime{Time: expiresAt, Valid: true},
			AppliedAt:         sql.NullTime{Valid: false},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if _, err := e.tluclawRepo.CreateSnapshot(ctx, snapshotParams); err != nil {
			return "Không thể tạo bản ghi xem trước thông báo.", nil, nil
		}

		card := &models.TLUClawPreviewCardData{
			SnapshotID:  snapshotID,
			BatchID:     batchID,
			ActionType:  "notification_dispatch",
			Summary:     snapshotParams.Summary,
			Nonce:       nonce,
			ExpiresAt:   expiresAt,
			DiffItems:   diffItems,
			BlastRadius: blastRadius,
		}

		msg := fmt.Sprintf("Đã tạo bản xem trước gửi thông báo tới %s. Vui lòng xác nhận để phát thông báo.", targetDesc)
		return msg, card.ToResponse(), nil

	default:
		return fmt.Sprintf("Hành động không được hỗ trợ: %s", action), nil, nil
	}
}

func (e *tluclawToolEngine) executeJobOperations(ctx context.Context, sessionID, adminID string, args *request.TLUClawJobOpArgs) (string, *response.TLUClawPreviewCardResponse, error) {
	if e.jobService == nil {
		return "Dịch vụ tác vụ nền (JobService) chưa sẵn sàng.", nil, nil
	}
	action := args.Action

	switch action {
	case "list_available_tasks":
		tasks := e.jobService.ListTasks()
		if len(tasks) == 0 {
			return "Hệ thống chưa đăng ký tác vụ nền nào.", nil, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# DANH SÁCH TÁC VỤ NỀN HỆ THỐNG (%d tác vụ):\n\n", len(tasks)))
		sb.WriteString("| Mã tác vụ (Task ID) | Tên / Mô tả tác vụ |\n")
		sb.WriteString("| :--- | :--- |\n")
		for _, t := range tasks {
			sb.WriteString(fmt.Sprintf("| `%s` | %s |\n", t.Type, t.Description))
		}
		return sb.String(), nil, nil

	case "list_recent_jobs":
		limitVal := args.Limit
		if limitVal <= 0 || limitVal > 100 {
			limitVal = 15
		}
		pageVal := args.Page
		if pageVal <= 0 {
			pageVal = 1
		}
		offsetVal := int64((pageVal - 1) * limitVal)

		jobs, total, err := e.jobService.ListJobs(ctx, &request.ListJobsDto{
			Limit:  int64(limitVal),
			Offset: offsetVal,
		})
		if err != nil || len(jobs) == 0 {
			return "Chưa có tác vụ nền nào trong lịch sử thực thi.", nil, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# LỊCH SỬ TÁC VỤ NỀN GẦN ĐÂY (Tổng: %d, Hiển thị: %d, Trang %d):\n\n", total, len(jobs), pageVal))
		sb.WriteString("| ID | Loại tác vụ | Trạng thái | Tiến độ | Lỗi | Bắt đầu | Cập nhật |\n")
		sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- | :--- |\n")
		for _, j := range jobs {
			errMsg := "-"
			if j.ErrorMsg != nil && *j.ErrorMsg != "" {
				errMsg = *j.ErrorMsg
				if len(errMsg) > 25 {
					errMsg = errMsg[:25] + "..."
				}
			}
			updated := j.UpdatedAt.Format("15:04 02/01")
			sb.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | %d%% | %s | %s | %s |\n",
				j.ID, j.Type, j.Status, j.Progress, errMsg, j.CreatedAt.Format("15:04 02/01"), updated))
		}
		return sb.String(), nil, nil

	case "inspect_job_status":
		jobID := strings.TrimSpace(args.JobID)
		if jobID == "" {
			return "Vui lòng cung cấp job_id cần tra cứu.", nil, nil
		}
		job, err := e.jobService.GetJob(ctx, jobID)
		if err != nil || job == nil {
			return fmt.Sprintf("Không tìm thấy tác vụ có ID `%s`.", jobID), nil, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# CHI TIẾT TÁC VỤ NỀN: `%s`\n\n", job.ID))
		fmt.Fprintf(&sb, "- **Loại tác vụ**: `%s`\n", job.Type)
		fmt.Fprintf(&sb, "- **Trạng thái**: `%s`\n", job.Status)
		fmt.Fprintf(&sb, "- **Tiến độ**: %d%%\n", job.Progress)
		if job.PayloadJSON != nil && *job.PayloadJSON != "" {
			fmt.Fprintf(&sb, "- **Dữ liệu đầu vào (Payload)**: `%s`\n", *job.PayloadJSON)
		}
		if job.ErrorMsg != nil && *job.ErrorMsg != "" {
			fmt.Fprintf(&sb, "- **Thông báo lỗi**: `%s`\n", *job.ErrorMsg)
		}
		fmt.Fprintf(&sb, "- **Thời điểm tạo**: %s\n", job.CreatedAt.Format("15:04:05 02/01/2006"))
		fmt.Fprintf(&sb, "- **Thời điểm cập nhật**: %s\n", job.UpdatedAt.Format("15:04:05 02/01/2006"))
		return sb.String(), nil, nil

	case "propose_trigger_job":
		taskType := strings.TrimSpace(args.TaskType)
		payload := strings.TrimSpace(args.Payload)

		if taskType == "" {
			return "Vui lòng chỉ định loại tác vụ task_type (ví dụ: crawl_notices, crawl_news, crawl_static_pages, clean_corpus, supersede_check).", nil, nil
		}

		diffItems := []models.TLUClawDiffItem{
			{
				EntityType: "ASYNC_JOB",
				Action:     "TRIGGER",
				Key:        taskType,
				Name:       taskType,
				OldValue:   "-",
				NewValue:   "QUEUED",
				Note:       fmt.Sprintf("Payload: %s", payload),
			},
		}

		blastRadius := models.TLUClawBlastRadius{
			ImpactSummary: fmt.Sprintf("Kích hoạt tác vụ nền '%s' đưa vào hàng đợi worker", taskType),
			RiskLevel:     "MEDIUM",
		}

		batchID := "batch_job_" + uuid.NewString()[:8]
		nonce, _ := crypto.GenerateRandomHex(24)
		expiresAt := time.Now().Add(15 * time.Minute)

		newStatePayload, _ := json.Marshal(request.TLUClawJobTriggerSnapshotState{
			TaskType: taskType,
			Payload:  payload,
		})

		diffJSON, _ := json.Marshal(diffItems)
		blastJSON, _ := json.Marshal(blastRadius)

		snapshotID := "snp_" + uuid.NewString()[:8]
		snapshotParams := sqlc.CreateTLUClawSnapshotParams{
			ID:                snapshotID,
			BatchID:           batchID,
			SessionID:         sessionID,
			OperatorID:        adminID,
			ActionType:        "job_trigger",
			Summary:           fmt.Sprintf("Kích hoạt tác vụ nền: '%s'", taskType),
			DiffPreviewJson:   string(diffJSON),
			BlastRadiusJson:   string(blastJSON),
			PreviousStateJson: "{}",
			NewStateJson:      string(newStatePayload),
			Status:            "STAGED",
			ConfirmationNonce: nonce,
			NonceExpiresAt:    sql.NullTime{Time: expiresAt, Valid: true},
			AppliedAt:         sql.NullTime{Valid: false},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if _, err := e.tluclawRepo.CreateSnapshot(ctx, snapshotParams); err != nil {
			return "Không thể tạo bản ghi xem trước kích hoạt tác vụ.", nil, nil
		}

		card := &models.TLUClawPreviewCardData{
			SnapshotID:  snapshotID,
			BatchID:     batchID,
			ActionType:  "job_trigger",
			Summary:     snapshotParams.Summary,
			Nonce:       nonce,
			ExpiresAt:   expiresAt,
			DiffItems:   diffItems,
			BlastRadius: blastRadius,
		}

		msg := fmt.Sprintf("Đã tạo bản xem trước kích hoạt tác vụ nền `%s`. Vui lòng xác nhận để đưa vào hàng đợi.", taskType)
		return msg, card.ToResponse(), nil

	default:
		return fmt.Sprintf("Hành động không được hỗ trợ: %s", action), nil, nil
	}
}

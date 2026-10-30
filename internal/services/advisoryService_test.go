package services_test

import (
	"context"
	"database/sql"
	"io"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/database"
	"tluagent-web/pkg/llm"
)

// TestAdvisoryTranscriptAndRoadmapIntegration verifies transcript context injection and DAG roadmap recommendation.
func TestAdvisoryTranscriptAndRoadmapIntegration(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("Failed to apply schema: %v", err)
	}

	ramCache := cache.NewTheineCache(10 << 20)
	userRepo := repositories.NewUserRepository(db, ramCache)
	transcriptRepo := repositories.NewTranscriptRepository(db, ramCache)
	memoryRepo := repositories.NewMemoryRepository(db, ramCache)
	curriculumRepo := repositories.NewCurriculumRepository(db, ramCache)
	ruleRepo := repositories.NewMemoryRuleRepository()
	llmRepo := repositories.NewLLMRepository(db, ramCache)

	transcriptSvc := services.NewTranscriptService(transcriptRepo, memoryRepo, nil)
	curriculumSvc := services.NewCurriculumService(curriculumRepo)
	roadmapOptimizer := services.NewRoadmapOptimizer(curriculumRepo)
	advisorySvc := services.NewAdvisoryService(services.AdvisoryParams{
		RuleRepo:         ruleRepo,
		UserRepo:         userRepo,
		LLMRepo:          llmRepo,
		TranscriptSvc:    transcriptSvc,
		CurriculumSvc:    curriculumSvc,
		RoadmapOptimizer: roadmapOptimizer,
		RamCache:         ramCache,
	})

	ctx := context.Background()

	testUser, err := userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           "test-adv-user",
		Email:        "adv_student@thanglong.edu.vn",
		FullName:     sql.NullString{String: "Nguyen Van A", Valid: true},
		StudentCode:  sql.NullString{String: "A41234", Valid: true},
		AuthProvider: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	grades := []*models.StudentCourseGradeRecord{
		{
			CourseCode:  "CS101",
			CourseName:  "Lap trinh Co ban",
			Credits:     3.0,
			Score10:     8.5,
			Score4:      3.5,
			GradeLetter: "B+",
			IsPassed:    true,
			TermTaken:   "HK1-2023",
		},
		{
			CourseCode:  "MA101",
			CourseName:  "Giai tich 1",
			Credits:     3.0,
			Score10:     3.0,
			Score4:      0.0,
			GradeLetter: "F",
			IsPassed:    false,
			TermTaken:   "HK1-2023",
		},
	}
	if _, err := transcriptSvc.BatchSaveTranscript(ctx, testUser.ID, grades); err != nil {
		t.Fatalf("Failed to save transcript: %v", err)
	}

	var events []services.StreamEvent
	chatReq := &request.ChatRequest{
		Query: "Tư vấn lộ trình học kỳ tới và xem lại bảng điểm của tôi",
	}

	resp, err := advisorySvc.ConsultStream(ctx, testUser.ID, chatReq, func(evt services.StreamEvent) error {
		events = append(events, evt)
		return nil
	})
	if err != nil {
		t.Fatalf("ConsultStream failed: %v", err)
	}
	if resp == nil {
		t.Fatal("Expected non-nil ChatResponse")
	}

	var foundTranscriptTool bool
	for _, evt := range events {
		if evt.Type == "tool_call" && evt.ToolCall != nil && evt.ToolCall.Name == "lookup_student_transcript_and_roadmap" {
			foundTranscriptTool = true
			break
		}
	}
	if !foundTranscriptTool {
		t.Errorf("Expected tool_call event for lookup_student_transcript_and_roadmap, got events: %+v", events)
	}

	var hasTranscriptSource bool
	for _, s := range resp.Sources {
		if strings.Contains(s, "bảng điểm") || strings.Contains(s, "Bảng điểm") {
			hasTranscriptSource = true
			break
		}
	}
	if !hasTranscriptSource {
		t.Errorf("Expected transcript source in response sources, got: %v", resp.Sources)
	}
}

type mockTestStreamReader struct {
	chunks []*llm.StreamChunk
	pos    int
}

func (r *mockTestStreamReader) Recv() (*llm.StreamChunk, error) {
	if r.pos >= len(r.chunks) {
		return nil, io.EOF
	}
	c := r.chunks[r.pos]
	r.pos++
	return c, nil
}

func (r *mockTestStreamReader) Close() error {
	return nil
}

type mockTestLLMProvider struct {
	chatFunc   func(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error)
	streamFunc func(ctx context.Context, req *llm.ChatRequest) (llm.StreamReader, error)
}

func (m *mockTestLLMProvider) Name() string         { return "mock" }
func (m *mockTestLLMProvider) DefaultModel() string { return "mock-model" }
func (m *mockTestLLMProvider) Chat(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, req)
	}
	return &llm.ChatResponse{
		Message: llm.AssistantMessage("Mock reply"),
	}, nil
}
func (m *mockTestLLMProvider) ChatStream(ctx context.Context, req *llm.ChatRequest) (llm.StreamReader, error) {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, req)
	}
	return nil, nil
}

// TestAdvisoryDynamicRoadmapPreferencesAndMemory tests collaborative roadmap preference persistence and memory hydration.
func TestAdvisoryDynamicRoadmapPreferencesAndMemory(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("Failed to apply schema: %v", err)
	}

	ramCache := cache.NewTheineCache(10 << 20)
	userRepo := repositories.NewUserRepository(db, ramCache)
	memoryRepo := repositories.NewMemoryRepository(db, ramCache)
	curriculumRepo := repositories.NewCurriculumRepository(db, ramCache)
	ruleRepo := repositories.NewMemoryRuleRepository()
	llmRepo := repositories.NewLLMRepository(db, ramCache)

	memorySvc := services.NewMemoryService(memoryRepo, userRepo, nil)
	curriculumSvc := services.NewCurriculumService(curriculumRepo)
	roadmapOptimizer := services.NewRoadmapOptimizer(curriculumRepo)

	llmManager := llm.NewManager()
	mockProv := &mockTestLLMProvider{}
	llmManager.Register("mock", mockProv)

	advisorySvc := services.NewAdvisoryService(services.AdvisoryParams{
		RuleRepo:         ruleRepo,
		LLMManager:       llmManager,
		UserRepo:         userRepo,
		LLMRepo:          llmRepo,
		MemoryService:    memorySvc,
		CurriculumSvc:    curriculumSvc,
		RoadmapOptimizer: roadmapOptimizer,
		RamCache:         ramCache,
	})

	ctx := context.Background()

	testUser, err := userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           "test-dyn-student",
		Email:        "dyn_student@thanglong.edu.vn",
		FullName:     sql.NullString{String: "Tran Van B", Valid: true},
		StudentCode:  sql.NullString{String: "A42001", Valid: true},
		AuthProvider: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	_, err = curriculumRepo.UpsertCurriculum(ctx, sqlc.UpsertCurriculumParams{
		ID:           "curr_k35_cntt",
		Cohort:       "K35",
		AcademicYear: "2022-2023",
		MajorCode:    "CNTT",
		MajorName:    "Công nghệ thông tin",
		TotalCourses: 3,
		TotalCredits: 9.0,
	})
	if err != nil {
		t.Fatalf("UpsertCurriculum failed: %v", err)
	}

	_, _ = curriculumRepo.UpsertCurriculumCourse(ctx, sqlc.UpsertCurriculumCourseParams{
		ID:                "c_cs102",
		CurriculumID:      "curr_k35_cntt",
		Cohort:            "K35",
		MajorName:         "Công nghệ thông tin",
		CourseCode:        "CS102",
		CourseName:        "Tin hoc van phong",
		Credits:           3.0,
		KnowledgeBlock:    "Co so",
		SuggestedSemester: 1,
		IsElearning:       1,
	})
	_, _ = curriculumRepo.UpsertCurriculumCourse(ctx, sqlc.UpsertCurriculumCourseParams{
		ID:                "c_ge111",
		CurriculumID:      "curr_k35_cntt",
		Cohort:            "K35",
		MajorName:         "Công nghệ thông tin",
		CourseCode:        "GE111",
		CourseName:        "Tieng Anh 1",
		Credits:           3.0,
		KnowledgeBlock:    "Dai cuong",
		SuggestedSemester: 1,
		IsElearning:       0,
	})
	_, _ = curriculumRepo.UpsertCurriculumCourse(ctx, sqlc.UpsertCurriculumCourseParams{
		ID:                "c_cs101",
		CurriculumID:      "curr_k35_cntt",
		Cohort:            "K35",
		MajorName:         "Công nghệ thông tin",
		CourseCode:        "CS101",
		CourseName:        "Lap trinh Co ban",
		Credits:           3.0,
		KnowledgeBlock:    "Chuyen nganh",
		SuggestedSemester: 2,
		IsElearning:       0,
	})

	turnCounter := 0
	mockProv.chatFunc = func(c context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		turnCounter++
		if turnCounter == 1 {
			return &llm.ChatResponse{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							ID:   "tc-save-pref",
							Type: "function",
							Function: llm.FunctionCall{
								Name: "update_student_academic_preferences",
								Arguments: `{"target_semesters": 7, "pace_preference": "fast", "summer_willingness": true, "work_schedule": "Part-time 15h/w", "focus_track": "AI & Data Science", "custom_notes": "Hoan MI201 sang he"}`,
							},
						},
					},
				},
			}, nil
		}
		return &llm.ChatResponse{
			Message: llm.AssistantMessage("Toi da luu cac thiet lap hoc tap vao bo nho dai han cua ban."),
		}, nil
	}

	saveReq := &request.ChatRequest{
		Query: "Em muốn đăng ký tốt nghiệp trong 3.5 năm, nhịp độ học nhanh và có đi làm thêm 15h/tuần",
	}
	saveResp, err := advisorySvc.Consult(ctx, testUser.ID, saveReq)
	if err != nil {
		t.Fatalf("Consult for saving preferences failed: %v", err)
	}
	if saveResp == nil {
		t.Fatal("Expected non-nil save response")
	}

	mems, err := memorySvc.ListUserMemories(ctx, testUser.ID)
	if err != nil {
		t.Fatalf("ListUserMemories failed: %v", err)
	}
	if len(mems) < 5 {
		t.Fatalf("Expected at least 5 memory facts saved, got %d", len(mems))
	}

	var hasTargetSemesters, hasPace, hasSummer bool
	for _, m := range mems {
		if m.MemoryKey == "TARGET_SEMESTERS" && strings.Contains(m.MemoryValue, "7 kỳ") {
			hasTargetSemesters = true
		}
		if m.MemoryKey == "PACE_PREFERENCE" && m.MemoryValue == "fast" {
			hasPace = true
		}
		if m.MemoryKey == "SUMMER_WILLINGNESS" && m.MemoryValue == "true" {
			hasSummer = true
		}
	}
	if !hasTargetSemesters || !hasPace || !hasSummer {
		t.Errorf("Missing expected memory entries: target=%v, pace=%v, summer=%v", hasTargetSemesters, hasPace, hasSummer)
	}

	turnCounter = 0
	mockProv.chatFunc = func(c context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
		turnCounter++
		if turnCounter == 1 {
			return &llm.ChatResponse{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							ID:   "tc-opt-road",
							Type: "function",
							Function: llm.FunctionCall{
								Name:      "optimize_study_roadmap",
								Arguments: `{"cohort": "K35", "major": "Công nghệ thông tin"}`,
							},
						},
					},
				},
			}, nil
		}
		return &llm.ChatResponse{
			Message: llm.AssistantMessage("Day la ban thao lo trinh de xuat cho ban."),
		}, nil
	}

	optReq := &request.ChatRequest{
		Query: "Em K35 ngành Công nghệ thông tin, tối ưu lại lộ trình học tập cho em với",
	}
	optResp, err := advisorySvc.Consult(ctx, testUser.ID, optReq)
	if err != nil {
		t.Fatalf("Consult for roadmap optimization failed: %v", err)
	}
	if optResp == nil || optResp.Roadmap == nil {
		t.Fatal("Expected non-nil optResp with Roadmap")
	}

	if optResp.Roadmap.TargetGraduationTerm != 7 {
		t.Errorf("Expected target graduation term 7 hydrated from memory, got %d", optResp.Roadmap.TargetGraduationTerm)
	}
	if len(optResp.Roadmap.Semesters) == 0 {
		t.Error("Expected scheduled semesters in roadmap preview")
	}
}

// TestAdvisoryRealtimeConsultStream verifies token, reasoning, and tool call streaming in ConsultStream.
func TestAdvisoryRealtimeConsultStream(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("Failed to apply schema: %v", err)
	}

	ramCache := cache.NewTheineCache(10 << 20)
	userRepo := repositories.NewUserRepository(db, ramCache)
	memoryRepo := repositories.NewMemoryRepository(db, ramCache)
	curriculumRepo := repositories.NewCurriculumRepository(db, ramCache)
	ruleRepo := repositories.NewMemoryRuleRepository()
	llmRepo := repositories.NewLLMRepository(db, ramCache)

	curriculumSvc := services.NewCurriculumService(curriculumRepo)
	roadmapOptimizer := services.NewRoadmapOptimizer(curriculumRepo)
	transcriptSvc := services.NewTranscriptService(repositories.NewTranscriptRepository(db, ramCache), memoryRepo, nil)

	llmManager := llm.NewManager()
	mockProv := &mockTestLLMProvider{}
	llmManager.Register("mock", mockProv)

	advisorySvc := services.NewAdvisoryService(services.AdvisoryParams{
		RuleRepo:         ruleRepo,
		LLMManager:       llmManager,
		UserRepo:         userRepo,
		LLMRepo:          llmRepo,
		TranscriptSvc:    transcriptSvc,
		CurriculumSvc:    curriculumSvc,
		RoadmapOptimizer: roadmapOptimizer,
		RamCache:         ramCache,
	})

	ctx := context.Background()

	testUser, err := userRepo.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           "test-adv-user-stream",
		Email:        "stream_student@thanglong.edu.vn",
		FullName:     sql.NullString{String: "Tran Van Stream", Valid: true},
		StudentCode:  sql.NullString{String: "A41999", Valid: true},
		AuthProvider: "local",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	streamTurn := 0
	mockProv.streamFunc = func(c context.Context, req *llm.ChatRequest) (llm.StreamReader, error) {
		streamTurn++
		if streamTurn == 1 {
			return &mockTestStreamReader{
				chunks: []*llm.StreamChunk{
					{
						Delta: llm.MessageDelta{
							ReasoningContent: "Toi dang suy nghi can tra cuu bang diem...",
						},
					},
					{
						Delta: llm.MessageDelta{
							ToolCalls: []llm.ToolCallChunk{
								{
									Index: 0,
									ID:    "call_transcript_stream_1",
									Type:  "function",
									Function: llm.FunctionCallChunk{
										Name:      "lookup_student_transcript_and_roadmap",
										Arguments: "{}",
									},
								},
							},
						},
					},
				},
			}, nil
		}

		return &mockTestStreamReader{
			chunks: []*llm.StreamChunk{
				{
					Delta: llm.MessageDelta{
						ReasoningContent: "Da co ket qua tra cuu, bat dau viet phan hoi...",
					},
				},
				{
					Delta: llm.MessageDelta{
						Content: "Dua vao ket qua tra cuu, ",
					},
				},
				{
					Delta: llm.MessageDelta{
						Content: "chuan dau ra tieng Anh TOEIC cua ban la 450 diem.",
					},
				},
			},
		}, nil
	}

	var events []services.StreamEvent
	chatReq := &request.ChatRequest{
		Query: "Chuan dau ra tieng anh cua toi la bao nhieu?",
	}

	resp, err := advisorySvc.ConsultStream(ctx, testUser.ID, chatReq, func(evt services.StreamEvent) error {
		events = append(events, evt)
		return nil
	})
	if err != nil {
		t.Fatalf("ConsultStream failed: %v", err)
	}
	if resp == nil {
		t.Fatal("Expected non-nil ChatResponse")
	}

	var hasReasoning bool
	var hasToolRunning bool
	var hasToolCompleted bool
	var chunkCount int
	var fullChunkText strings.Builder

	for _, evt := range events {
		switch evt.Type {
		case "reasoning":
			hasReasoning = true
		case "tool_call":
			if evt.ToolCall != nil {
				if evt.ToolCall.Status == "running" {
					hasToolRunning = true
				}
				if evt.ToolCall.Status == "completed" {
					hasToolCompleted = true
				}
			}
		case "chunk":
			chunkCount++
			fullChunkText.WriteString(evt.Content)
		}
	}

	if !hasReasoning {
		t.Error("Expected reasoning stream events to be emitted to client")
	}
	if !hasToolRunning {
		t.Error("Expected tool_call running event to be emitted")
	}
	if !hasToolCompleted {
		t.Error("Expected tool_call completed event to be emitted")
	}
	if chunkCount == 0 {
		t.Error("Expected chunk stream events to be emitted")
	}
	if !strings.Contains(fullChunkText.String(), "450 diem") {
		t.Errorf("Expected streamed text to contain '450 diem', got: %s", fullChunkText.String())
	}
}



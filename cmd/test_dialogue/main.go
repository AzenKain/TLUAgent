package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/config"
	"tluagent-web/pkg/database"
	"tluagent-web/pkg/llm"
)

func main() {
	if err := config.LoadEnv(); err != nil {
		fmt.Printf("Error loading .env: %v\n", err)
	}

	dbPath := "data/tluagent.db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		dbPath = "web/data/tluagent.db"
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)")
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.ApplySchema(db); err != nil {
		fmt.Printf("Schema migration error: %v\n", err)
	}

	ramCache := cache.NewRamCache()

	userRepo := repositories.NewUserRepository(db, ramCache)
	chatRepo := repositories.NewChatRepository(db, ramCache)
	ragRepo := repositories.NewRAGRepository(db, ramCache)
	agentRepo := repositories.NewAgentRepository(db, ramCache)
	memoryRepo := repositories.NewMemoryRepository(db, ramCache)
	llmRepo := repositories.NewLLMRepository(db, ramCache)
	ruleRepo := repositories.NewMemoryRuleRepository()

	llmManager := llm.NewManager()
	llmConfigSvc := services.NewLLMConfigService(llmRepo, llmManager)
	_ = llmConfigSvc.SyncLLMManager(context.Background())

	token := os.Getenv("TOKEN")
	if token == "" {
		token = os.Getenv("OPENAI_API_KEY")
	}
	apiURL := os.Getenv("API")
	if apiURL == "" {
		apiURL = "https://api.vilao.ai"
	}
	model := os.Getenv("MODEL")
	if model == "" {
		model = "chib/deepseek-v4.1-flash"
	}

	if token != "" {
		vilaoProv := llm.NewOpenAIProvider(llm.OpenAIConfig{
			BaseURL:      apiURL,
			APIKey:       token,
			DefaultModel: model,
			HTTPClient:   llm.NewRetryClient(120*time.Second, 2),
		})
		llmManager.Register("vilao", vilaoProv)
		llmManager.Register("openai", vilaoProv)
		_ = llmManager.SetDefault("vilao")
		fmt.Printf("[CONFIG] Connected Vilao provider: %s (model: %s)\n", apiURL, model)
	} else {
		fmt.Println("[CONFIG] No TOKEN found in .env, checking default provider...")
	}

	agentSvc := services.NewAgentService(agentRepo)
	_ = agentSvc.SeedDefaultsIfEmpty(context.Background())

	memorySvc := services.NewMemoryService(memoryRepo, userRepo)
	memorySvc.SetLLMManager(llmManager)

	ragSvc := services.NewRAGService(ragRepo)
	ragSvc.SetLLMManager(llmManager)

	advisorySvc := services.NewAdvisoryService(ruleRepo, llmManager, userRepo, llmRepo)
	advisorySvc.SetRAGService(ragSvc)
	advisorySvc.SetAgentService(agentSvc)
	advisorySvc.SetMemoryService(memorySvc)
	advisorySvc.SetChatRepository(chatRepo)

	ctx := context.Background()

	testUserID := "student-test-vilao-01"
	existingUser, _ := userRepo.GetByID(ctx, testUserID)
	if existingUser == nil {
		_, err = userRepo.CreateUser(ctx, sqlc.CreateUserParams{
			ID:           testUserID,
			Email:        "sv_vilao_test@thanglong.edu.vn",
			FullName:     sql.NullString{String: "Nguyen Hai Dang", Valid: true},
			StudentCode:  sql.NullString{String: "A36999", Valid: true},
			AuthProvider: "local",
		})
		if err != nil {
			fmt.Printf("Create user note: %v\n", err)
		}
	}

	initialProfile := &models.StudentProfile{
		UserID:           testUserID,
		Major:            "Information Technology",
		Cohort:           "K36",
		AcademicStanding: "NORMAL",
		CompletedCredits: 100,
		CumulativeGPA:    3.20,
		TargetGPA:        3.50,
		AdvisorNotes:     "Sinh vien du kien tot nghiep dot 3 nam 2026",
		UpdatedAt:        time.Now(),
	}
	_, _ = memorySvc.UpsertStudentProfile(ctx, initialProfile)

	_ = memoryRepo.DeleteUserMemory(ctx, "mem-target-goal", testUserID)
	_, _ = memorySvc.SaveUserMemory(ctx, &models.UserMemoryItem{
		ID:          "mem-target-goal",
		UserID:      testUserID,
		Category:    "ACADEMIC_GOAL",
		MemoryKey:   "graduation_plan",
		MemoryValue: "Mục tiêu tốt nghiệp loại Giỏi (GPA >= 3.2) vào kỳ 3 năm 2026 ngành CNTT",
		Confidence:  1.0,
		Source:      "INITIAL_ADVISORY",
	})

	fmt.Println("\n================================================================================")
	fmt.Println("             TLUAGENT MULTI-TURN DIALOGUE & KNOWLEDGE VERIFICATION              ")
	fmt.Println("================================================================================")

	sessionID := uuid.NewString()
	var history []request.ChatMessageDTO

	turns := []struct {
		turnNum     int
		title       string
		query       string
		newSession  bool
		verifyNotes string
	}{
		{
			turnNum:     1,
			title:       "Turn 1: Giới thiệu, hỏi chuẩn đầu ra TOEIC cho Khóa 36 CNTT",
			query:       "Chào Cố vấn, em là sinh viên K36 ngành Công nghệ thông tin (mã SV A36999). Hiện tại em đã tích lũy 100 tín chỉ, GPA 3.2. Mục tiêu của em là tốt nghiệp loại Giỏi năm 2026. Em muốn hỏi: Với K36 ngành CNTT, chuẩn đầu ra tiếng Anh TOEIC yêu cầu bao nhiêu điểm?",
			newSession:  false,
			verifyNotes: "Kỳ vọng: Chuẩn TOEIC K36 CNTT là 450 điểm (hoặc theo Quyết định CĐR ngoại ngữ). Có trích dẫn nguồn văn bản.",
		},
		{
			turnNum:     2,
			title:       "Turn 2: Hỏi xử lý môn nợ điểm F (Giải tích 2) & Short-term Context Memory",
			query:       "Kỳ này em lỡ bị điểm F môn Giải tích 2 (3 tín chỉ). Theo quy chế trường mình, em có bắt buộc phải học lại môn này không? Điểm F này có tính vào GPA không và sau khi học lại qua môn thì điểm mới có thay thế điểm F cũ không?",
			newSession:  false,
			verifyNotes: "Kỳ vọng: Điểm F môn bắt buộc phải học lại; giải thích cách tính điểm tích lũy theo quy chế tín chỉ TLU. Có nguồn trích dẫn.",
		},
		{
			turnNum:     3,
			title:       "Turn 3: Kiểm tra Long-term Memory (Mở phiên mới, Cố vấn có nhớ thông tin K36 & Mục tiêu không?)",
			query:       "Thầy cô nhắc lại giúp em: với mục tiêu tốt nghiệp mà em đã chia sẻ từ đầu và khóa học của em, em cần hoàn thành những điều kiện học vụ và chuẩn đầu ra nào nữa ngoài TOEIC?",
			newSession:  true,
			verifyNotes: "Kỳ vọng: Cố vấn nhận diện sinh viên K36 CNTT, nhớ mục tiêu tốt nghiệp Giỏi 2026 từ Long-term Memory, nêu các CĐR (Tin học, GDTC, GDQP-AN, Tín chỉ tích lũy, Đồ án).",
		},
		{
			turnNum:     4,
			title:       "Turn 4: Kiểm tra Chống Bịa Đặt / Anti-Hallucination (Bẫy câu hỏi sai sự thật)",
			query:       "Em nghe bạn em bảo sinh viên K36 được miễn học phí 100% nếu có chứng chỉ HSK tiếng Trung cấp 3 và trường mình có chính sách cộng 2.0 GPA cho sinh viên tham gia câu lạc bộ nhảy, có đúng không ạ?",
			newSession:  false,
			verifyNotes: "Kỳ vọng: Bác bỏ hoàn toàn tin đồn bịa đặt, khẳng định không có quy định miễn 100% học phí bằng HSK3 hay cộng 2.0 GPA cho CLB nhảy.",
		},
		{
			turnNum:     5,
			title:       "Turn 5: Hỏi thủ tục nghỉ học tạm thời (bảo lưu kết quả học tập) & Nơi tiếp nhận",
			query:       "Em hiểu rồi. Thế nếu em muốn làm thủ tục nghỉ học tạm thời (bảo lưu) 1 học kỳ vì lý do sức khỏe thì điều kiện, giấy tờ cần chuẩn bị và nộp tại phòng ban nào của trường mình?",
			newSession:  false,
			verifyNotes: "Kỳ vọng: Nêu rõ điều kiện bảo lưu, hồ sơ bệnh án, địa điểm nộp (Bộ phận tiếp sinh viên / Phòng Đào tạo Nhà T). Có trích dẫn nguồn.",
		},
	}

	for _, tCase := range turns {
		fmt.Printf("\n--------------------------------------------------------------------------------\n")
		fmt.Printf("▶ %s\n", tCase.title)
		fmt.Printf("Target: %s\n", tCase.verifyNotes)
		fmt.Printf("User Query: \"%s\"\n\n", tCase.query)

		if tCase.newSession {
			sessionID = uuid.NewString()
			history = nil
			fmt.Printf("[SESSION RESET] Starting fresh session: %s to test LONG-TERM MEMORY...\n", sessionID)
		}

		startTime := time.Now()
		reqDto := &request.ChatRequest{
			Query:     tCase.query,
			History:   history,
			SessionID: sessionID,
			ModelID:   "vilao-default",
		}

		resp, err := advisorySvc.Consult(ctx, testUserID, reqDto)
		duration := time.Since(startTime)

		if err != nil {
			fmt.Printf("❌ ERROR: %v\n", err)
			continue
		}

		fmt.Printf("Agent Reply (%s):\n%s\n\n", duration.Round(time.Millisecond), resp.Reply)

		hasSourcesInText := strings.Contains(resp.Reply, "Nguồn") || strings.Contains(resp.Reply, "Căn cứ") || strings.Contains(resp.Reply, "Quy chế") || strings.Contains(resp.Reply, "Quyết định") || len(resp.Sources) > 0
		fmt.Printf("[AUDIT VERIFICATION]\n")
		fmt.Printf("- Trả về Sources Metadata: %d nguồn (%v)\n", len(resp.Sources), resp.Sources)
		fmt.Printf("- Có trích dẫn nguồn ở cuối bài/trong bài: %v\n", hasSourcesInText)

		history = append(history,
			request.ChatMessageDTO{Role: "user", Content: tCase.query},
			request.ChatMessageDTO{Role: "assistant", Content: resp.Reply},
		)

		time.Sleep(1 * time.Second)
	}

	fmt.Println("\n================================================================================")
	fmt.Println("             INSPECTING STORED LONG-TERM USER MEMORIES                          ")
	fmt.Println("================================================================================")
	memList, err := memorySvc.ListUserMemories(ctx, testUserID)
	if err != nil {
		fmt.Printf("Error listing memories: %v\n", err)
	} else {
		fmt.Printf("Total stored long-term memory items for %s: %d\n", testUserID, len(memList))
		for idx, m := range memList {
			fmt.Printf("  [%d] Category: %s | Key: %s | Value: %s (Source: %s, Conf: %.2f)\n",
				idx+1, m.Category, m.MemoryKey, m.MemoryValue, m.Source, m.Confidence)
		}
	}

	prof, err := memorySvc.GetStudentProfile(ctx, testUserID)
	if err == nil && prof != nil {
		fmt.Printf("\nStudent Academic Profile: Cohort=%s, Major=%s, GPA=%.2f, Standing=%s, Notes=%s\n",
			prof.Cohort, prof.Major, prof.CumulativeGPA, prof.AcademicStanding, prof.AdvisorNotes)
	}
	fmt.Println("================================================================================")
}

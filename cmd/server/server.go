package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"

	"os"
	"tluagent-web/internal/agent"
	"tluagent-web/internal/controllers"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/routes"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/config"
	"tluagent-web/pkg/database"
	"tluagent-web/pkg/embedding"
	"tluagent-web/pkg/guardrail"
	"tluagent-web/pkg/llm"
	"tluagent-web/pkg/tuition"
	"tluagent-web/pkg/worker"
)

//go:embed all:dist
var embeddedDist embed.FS

type Server struct {
	httpServer     *http.Server
	db             *sql.DB
	addr           string
	jobScheduleSvc services.JobScheduleService
	jobQueue       *worker.Queue
}

func NewServer(addr string) *Server {
	db, err := database.NewSQLiteDB()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize SQLite database")
	}

	if err := database.ApplySchema(db); err != nil {
		log.Fatal().Err(err).Msg("Failed to apply database schema migrations")
	}

	ramCache := cache.NewRamCache()

	txManager := database.NewTxManager(db)
	roleRepo := repositories.NewRoleRepository(db, ramCache)
	userRepo := repositories.NewUserRepository(db, ramCache)
	settingsRepo := repositories.NewSettingsRepository(db, ramCache)
	llmRepo := repositories.NewLLMRepository(db, ramCache)
	ruleRepo := repositories.NewMemoryRuleRepository()
	chatRepo := repositories.NewChatRepository(db, ramCache)
	ragRepo := repositories.NewRAGRepository(db, ramCache)
	agentRepo := repositories.NewAgentRepository(db, ramCache)
	memoryRepo := repositories.NewMemoryRepository(db, ramCache)
	inquiryRepo := repositories.NewInquiryRepository(db, ramCache)
	notificationRepo := repositories.NewNotificationRepository(db, ramCache)
	curriculumRepo := repositories.NewCurriculumRepository(db, ramCache)
	transcriptRepo := repositories.NewTranscriptRepository(db, ramCache)
	tluclawRepo := repositories.NewTLUClawRepository(db, ramCache)
	jobRepo := repositories.NewJobRepository(db, ramCache)
	jobScheduleRepo := repositories.NewJobScheduleRepository(db, ramCache)

	llmManager := llm.NewManager()
	tuitionClient := tuition.NewClient(10 * time.Second)

	llmConfigSvc := services.NewLLMConfigService(llmRepo, llmManager, settingsRepo)
	if err := llmConfigSvc.SeedDefaultProvidersIfEmpty(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed to seed default LLM providers")
	}
	if err := llmConfigSvc.SyncLLMManager(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed initial LLM manager synchronization")
	}

	permCache := services.NewPermissionCache(roleRepo)
	if err := permCache.Reload(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed initial permission cache load")
	}

	memorySvc := services.NewMemoryService(memoryRepo, userRepo, llmManager)
	inquirySvc := services.NewInquiryService(inquiryRepo, ragRepo, userRepo, notificationRepo, llmManager)
	notifSvc := services.NewNotificationService(notificationRepo)
	ragSvc := services.NewRAGService(ragRepo, ramCache, llmManager, inquirySvc)

	agentSvc := services.NewAgentService(agentRepo, ragSvc, settingsRepo)
	if err := agentSvc.SeedDefaultsIfEmpty(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed to seed default agent prompts and skills")
	}

	roleSvc := services.NewRoleService(roleRepo, permCache, txManager)
	authSvc := services.NewAuthService(userRepo, roleRepo, settingsRepo, txManager)
	userSvc := services.NewUserService(userRepo, roleRepo, settingsRepo, txManager, permCache)
	curriculumSvc := services.NewCurriculumService(curriculumRepo)
	roadmapOptimizer := services.NewRoadmapOptimizer(curriculumRepo)
	transcriptSvc := services.NewTranscriptService(transcriptRepo, memoryRepo, llmManager)

	advisorySvc := services.NewAdvisoryService(services.AdvisoryParams{
		RuleRepo:         ruleRepo,
		LLMManager:       llmManager,
		UserRepo:         userRepo,
		LLMRepo:          llmRepo,
		RAGService:       ragSvc,
		AgentService:     agentSvc,
		MemoryService:    memorySvc,
		ChatRepo:         chatRepo,
		SettingsRepo:     settingsRepo,
		TuitionClient:    tuitionClient,
		RamCache:         ramCache,
		CurriculumSvc:    curriculumSvc,
		TranscriptSvc:    transcriptSvc,
		RoadmapOptimizer: roadmapOptimizer,
	})

	chatSessionSvc := services.NewChatSessionService(chatRepo, userRepo)
	chatQuotaSvc := services.NewChatQuotaService(ramCache)
	streamLimiter := services.NewStreamLimiter(
		config.GetIntConfigWithDefault("CHAT_MAX_STREAMS", 50),
		config.GetIntConfigWithDefault("CHAT_MAX_STREAMS_PER_CLIENT", 2),
	)

	jobQueue := worker.NewQueue(4)
	dissectionEngine := services.NewRegulatoryDissectionEngine(services.EngineParams{
		DB:             db,
		RAGRepo:        ragRepo,
		CurriculumRepo: curriculumRepo,
		InquiryRepo: inquiryRepo,
		NotifRepo:   notificationRepo,
		UserRepo:    userRepo,
		LLMManager:  llmManager,
	})
	detector := services.NewDetector(db, llmManager)
	detector.SetEngine(dissectionEngine)
	jobSvc := services.NewJobService(jobRepo, jobQueue, llmManager, ragSvc, detector)
	jobQueue.Start()
	_ = jobSvc.Recover(context.Background())
	jobScheduleSvc := services.NewJobScheduleService(jobScheduleRepo, jobSvc)
	if err := jobScheduleSvc.SeedDefaultSchedulesIfEmpty(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed to seed default job schedules")
	}
	jobScheduleSvc.Start()

	tluclawSvc := services.NewTLUClawService(services.TLUClawParams{
		TLUClawRepo:      tluclawRepo,
		CurriculumRepo:   curriculumRepo,
		RAGRepo:          ragRepo,
		RAGService:       ragSvc,
		RAMCache:         ramCache,
		LLMManager:       llmManager,
		SkillLoader:      agent.NewDBSkillLoader(agentRepo),
		MemoryService:    memorySvc,
		UserRepo:         userRepo,
		RoleRepo:         roleRepo,
		TranscriptRepo:   transcriptRepo,
		InquiryRepo:      inquiryRepo,
		LLMConfigSvc:     llmConfigSvc,
		MemoryRepo:       memoryRepo,
		NotifRepo:        notificationRepo,
		JobService:       jobSvc,
		TuitionClient:    tuitionClient,
		RoleService:      roleSvc,
		SettingsRepo:     settingsRepo,
		DissectionEngine: dissectionEngine,
	})

	healthCtrl := controllers.NewHealthController()
	authCtrl := controllers.NewAuthController(authSvc)
	roleCtrl := controllers.NewRoleController(roleSvc)
	userCtrl := controllers.NewUserController(userSvc)
	chatCtrl := controllers.NewChatController(advisorySvc, chatSessionSvc, streamLimiter)
	adminChatCtrl := controllers.NewAdminChatController(chatSessionSvc)
	llmCtrl := controllers.NewLLMController(llmConfigSvc)
	ragCtrl := controllers.NewRAGController(ragSvc)
	agentCtrl := controllers.NewAgentController(agentSvc)
	memoryCtrl := controllers.NewMemoryController(memorySvc)
	inquiryCtrl := controllers.NewInquiryController(inquirySvc)
	notifCtrl := controllers.NewNotificationController(notifSvc)
	curriculumCtrl := controllers.NewCurriculumController(curriculumSvc, roadmapOptimizer, transcriptSvc, userRepo, memorySvc)
	transcriptCtrl := controllers.NewTranscriptController(transcriptSvc)
	tluclawCtrl := controllers.NewTLUClawController(tluclawSvc)
	jobCtrl := controllers.NewJobController(jobSvc, jobScheduleSvc)

	embeddingRoot := filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "onnx")
	embeddingSvc := services.NewEmbeddingService(settingsRepo, embeddingRoot, llmManager)
	if err := embeddingSvc.SyncActiveEmbedder(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed initial active ONNX embedder synchronization")
	}
	embeddingCtrl := controllers.NewEmbeddingController(embeddingSvc)

	bgeM3Dir := filepath.Join(embeddingRoot, "bge-m3")
	if _, err := os.Stat(bgeM3Dir); err == nil {
		if manifest, err := embedding.LoadManifest(bgeM3Dir); err == nil {
			if modelPath, err := embedding.FindModelFile(bgeM3Dir); err == nil {
				if libPath, err := embedding.CurrentRuntimeLibraryPath(embeddingRoot); err == nil {
					if onnxEmbedder, err := embedding.NewOnnxEmbedder(bgeM3Dir, manifest, modelPath, libPath); err == nil {
						guardrail.SetGlobalClassifier(guardrail.NewBGEM3Classifier(onnxEmbedder))
						log.Info().Msg("Initialized BGE-M3 dual-anchor semantic classifier for guardrail")
					}
				}
			}
		}
	}
	if guardrail.GetGlobalClassifier() == nil && llmManager.Embedder() != nil {
		guardrail.SetGlobalClassifier(guardrail.NewBGEM3Classifier(llmManager.Embedder()))
	}

	mux := http.NewServeMux()
	routes.RegisterAPIRoutes(mux, healthCtrl, chatCtrl, adminChatCtrl, userRepo, permCache, chatQuotaSvc)
	routes.RegisterAuthRoutes(mux, authCtrl, userRepo)
	routes.RegisterRoleRoutes(mux, roleCtrl, userRepo, permCache)
	routes.RegisterUserRoutes(mux, userCtrl, userRepo, permCache)
	routes.RegisterLLMRoutes(mux, llmCtrl, userRepo, permCache)
	routes.RegisterEmbeddingRoutes(mux, embeddingCtrl, userRepo, permCache)
	routes.RegisterRAGRoutes(mux, ragCtrl, userRepo, permCache)
	routes.RegisterAgentRoutes(mux, agentCtrl, userRepo, permCache)
	routes.RegisterMemoryRoutes(mux, memoryCtrl, userRepo, permCache)
	routes.RegisterInquiryRoutes(mux, inquiryCtrl, notifCtrl, userRepo, permCache)
	routes.RegisterCurriculumRoutes(mux, curriculumCtrl, userRepo, permCache)
	routes.RegisterTranscriptRoutes(mux, transcriptCtrl, userRepo, permCache)
	routes.RegisterTLUClawRoutes(mux, tluclawCtrl, userRepo, permCache)
	routes.RegisterJobRoutes(mux, jobCtrl, userRepo, permCache)
	routes.RegisterSPARoutes(mux, embeddedDist)

	rateLimiter := middlewares.NewRateLimiter(120, time.Minute)
	handler := middlewares.Recovery(
		middlewares.RequestLogger(
			middlewares.SecurityHeaders()(
				middlewares.CORS(
					middlewares.BodyLimit(10 << 20)(
						rateLimiter.Middleware()(
							middlewares.CSRFProtection()(mux),
						),
					),
				),
			),
		),
	)

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       120 * time.Second,
	}

	return &Server{
		httpServer:     srv,
		db:             db,
		addr:           addr,
		jobScheduleSvc: jobScheduleSvc,
		jobQueue:       jobQueue,
	}
}

func (s *Server) Start() error {
	log.Info().
		Str("addr", s.addr).
		Msg("TLUAgent Web Server is listening")

	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("HTTP server error: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Info().Msg("Closing HTTP server...")
	serverErr := s.httpServer.Shutdown(ctx)

	if s.jobScheduleSvc != nil {
		s.jobScheduleSvc.Stop()
	}
	if s.jobQueue != nil {
		s.jobQueue.Stop()
	}

	log.Info().Msg("Performing SQLite WAL checkpoint and closing database...")
	dbErr := database.CheckpointWALAndClose(s.db)

	if serverErr != nil {
		return serverErr
	}
	return dbErr
}

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

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/middlewares"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/routes"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/config"
	"tluagent-web/pkg/database"
	"tluagent-web/pkg/llm"
	"tluagent-web/pkg/supersede"
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

	llmManager := llm.NewManager()
	llmConfigSvc := services.NewLLMConfigService(llmRepo, llmManager)
	llmConfigSvc.SetSettingsRepository(settingsRepo)
	if err := llmConfigSvc.SeedDefaultProvidersIfEmpty(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed to seed default LLM providers")
	}
	if err := llmConfigSvc.SyncLLMManager(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed initial LLM manager synchronization")
	}

	agentSvc := services.NewAgentService(agentRepo)
	if err := agentSvc.SeedDefaultsIfEmpty(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed to seed default agent prompts and skills")
	}

	permCache := services.NewPermissionCache(roleRepo)
	if err := permCache.Reload(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed initial permission cache load")
	}

	memoryRepo := repositories.NewMemoryRepository(db, ramCache)
	memorySvc := services.NewMemoryService(memoryRepo, userRepo)
	memorySvc.SetLLMManager(llmManager)

	inquiryRepo := repositories.NewInquiryRepository(db, ramCache)
	notificationRepo := repositories.NewNotificationRepository(db, ramCache)
	inquirySvc := services.NewInquiryService(inquiryRepo, ragRepo, userRepo, notificationRepo)
	inquirySvc.SetLLMManager(llmManager)
	notifSvc := services.NewNotificationService(notificationRepo)

	ragSvc := services.NewRAGService(ragRepo)
	ragSvc.SetLLMManager(llmManager)
	ragSvc.SetInquiryService(inquirySvc)
	agentSvc.SetRAGService(ragSvc)
	agentSvc.SetSettingsRepository(settingsRepo)
	roleSvc := services.NewRoleService(roleRepo, permCache, txManager)
	authSvc := services.NewAuthService(userRepo, roleRepo, settingsRepo, txManager)
	userSvc := services.NewUserService(userRepo, roleRepo, settingsRepo, txManager)
	advisorySvc := services.NewAdvisoryService(ruleRepo, llmManager, userRepo, llmRepo)
	advisorySvc.SetRAGService(ragSvc)
	advisorySvc.SetAgentService(agentSvc)
	advisorySvc.SetMemoryService(memorySvc)
	advisorySvc.SetChatRepository(chatRepo)
	advisorySvc.SetSettingsRepository(settingsRepo)
	advisorySvc.SetRamCache(ramCache)
	chatSessionSvc := services.NewChatSessionService(chatRepo, userRepo)
	chatQuotaSvc := services.NewChatQuotaService(ramCache)
	streamLimiter := services.NewStreamLimiter(
		config.GetIntConfigWithDefault("CHAT_MAX_STREAMS", 50),
		config.GetIntConfigWithDefault("CHAT_MAX_STREAMS_PER_CLIENT", 2),
	)

	healthCtrl := controllers.NewHealthController()
	authCtrl := controllers.NewAuthController(authSvc)
	roleCtrl := controllers.NewRoleController(roleSvc)
	userCtrl := controllers.NewUserController(userSvc)
	chatCtrl := controllers.NewChatController(advisorySvc, chatSessionSvc, streamLimiter)
	adminChatCtrl := controllers.NewAdminChatController(chatSessionSvc)
	llmCtrl := controllers.NewLLMController(llmConfigSvc)
	ragCtrl := controllers.NewRAGController(ragSvc, ragRepo)
	agentCtrl := controllers.NewAgentController(agentSvc)
	memoryCtrl := controllers.NewMemoryController(memorySvc)
	inquiryCtrl := controllers.NewInquiryController(inquirySvc)
	notifCtrl := controllers.NewNotificationController(notifSvc)

	jobRepo := repositories.NewJobRepository(db, ramCache)
	jobScheduleRepo := repositories.NewJobScheduleRepository(db, ramCache)
	jobQueue := worker.NewQueue(4)
	detector := supersede.NewDetector(db, llmManager)
	jobSvc := services.NewJobService(jobRepo, jobQueue, llmManager, ragSvc, detector)
	_ = jobSvc.Recover(context.Background())
	jobScheduleSvc := services.NewJobScheduleService(jobScheduleRepo, jobSvc)
	if err := jobScheduleSvc.SeedDefaultSchedulesIfEmpty(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed to seed default job schedules")
	}
	jobScheduleSvc.Start()
	jobCtrl := controllers.NewJobController(jobSvc, jobScheduleSvc)

	embeddingRoot := filepath.Join(config.GetConfigWithDefault("DATA_DIR", "./data"), "onnx")
	embeddingSvc := services.NewEmbeddingService(settingsRepo, embeddingRoot)
	embeddingSvc.SetLLMManager(llmManager)
	if err := embeddingSvc.SyncActiveEmbedder(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed initial active ONNX embedder synchronization")
	}
	embeddingCtrl := controllers.NewEmbeddingController(embeddingSvc)

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
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
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

package services_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tluagent-web/internal/controllers"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/repositories"
	"tluagent-web/internal/routes"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/database"
)

type isolationRedteamHarness struct {
	mux        *http.ServeMux
	users      repositories.UserRepository
	roles      repositories.RoleRepository
	auth       services.AuthService
	transcript services.TranscriptService
	memory     services.MemoryService
	inquiries  services.InquiryService
	notifs     repositories.NotificationRepository
}

func isolationRedteamRandom(t *testing.T) string {
	t.Helper()
	var data [32]byte
	if _, err := rand.Read(data[:]); err != nil {
		t.Fatal("generate ephemeral test value failed")
	}
	return hex.EncodeToString(data[:])
}

func newIsolationRedteamHarness(t *testing.T, cached bool) *isolationRedteamHarness {
	t.Helper()
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("JWT_SECRET", isolationRedteamRandom(t))
	t.Setenv("JWT_REFRESH_SECRET", isolationRedteamRandom(t))
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	var ram cache.Cache
	if cached {
		ram = cache.NewTheineCache(10 << 20)
	}
	users := repositories.NewUserRepository(db, ram)
	roles := repositories.NewRoleRepository(db, ram)
	settings := repositories.NewSettingsRepository(db, ram)
	for roleName, keys := range map[string][]string{
		constants.RoleTypeStudent.String(): {constants.PermInquiryRead, constants.PermInquiryCreate, constants.PermRAGRead},
		constants.RoleTypeAdvisor.String(): {constants.PermInquiryRead, constants.PermInquiryAnswer},
	} {
		role, err := roles.GetByName(context.Background(), roleName)
		if err != nil {
			t.Fatal(err)
		}
		permissions, err := roles.GetRolePermissions(context.Background(), role.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range keys {
			permissions = append(permissions, &models.RolePermissionEntity{PermissionKey: key, Effect: "allow"})
		}
		if err := roles.ReplaceRolePermissions(context.Background(), role.ID, permissions); err != nil {
			t.Fatal(err)
		}
	}
	perms := services.NewPermissionCache(roles)
	if err := perms.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	memory := services.NewMemoryService(repositories.NewMemoryRepository(db, ram), users, nil)
	transcript := services.NewTranscriptService(repositories.NewTranscriptRepository(db, ram), nil, nil)
	notifs := repositories.NewNotificationRepository(db, ram)
	ragRepo := repositories.NewRAGRepository(db, ram)
	inquiries := services.NewInquiryService(repositories.NewInquiryRepository(db, ram), ragRepo, users, notifs, nil)
	rag := services.NewRAGService(ragRepo, ram, nil, inquiries)
	curriculumRepo := repositories.NewCurriculumRepository(db, ram)
	curriculum := controllers.NewCurriculumController(services.NewCurriculumService(curriculumRepo), services.NewRoadmapOptimizer(curriculumRepo), transcript, users, memory)
	mux := http.NewServeMux()
	routes.RegisterTranscriptRoutes(mux, controllers.NewTranscriptController(transcript), users, perms)
	routes.RegisterCurriculumRoutes(mux, curriculum, users, perms)
	routes.RegisterMemoryRoutes(mux, controllers.NewMemoryController(memory), users, perms)
	routes.RegisterInquiryRoutes(mux, controllers.NewInquiryController(inquiries), controllers.NewNotificationController(services.NewNotificationService(notifs)), users, perms)
	routes.RegisterRAGRoutes(mux, controllers.NewRAGController(rag), users, perms)
	return &isolationRedteamHarness{
		mux: mux, users: users, roles: roles,
		auth:       services.NewAuthService(users, roles, settings, database.NewTxManager(db)),
		transcript: transcript, memory: memory, inquiries: inquiries, notifs: notifs,
	}
}

func (h *isolationRedteamHarness) user(t *testing.T, roleType constants.RoleType) (string, string) {
	t.Helper()
	ctx := context.Background()
	id := isolationRedteamRandom(t)
	if _, err := h.users.CreateUser(ctx, sqlc.CreateUserParams{ID: id, Email: id + "@example.invalid", AuthProvider: "LOCAL"}); err != nil {
		t.Fatal(err)
	}
	role, err := h.roles.GetByName(ctx, roleType.String())
	if err != nil {
		t.Fatal(err)
	}
	if err := h.roles.CreateUserRole(ctx, id, role.ID); err != nil {
		t.Fatal(err)
	}
	user, err := h.users.GetByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	tokens, err := h.auth.GenToken(user)
	if err != nil {
		t.Fatal(err)
	}
	return id, tokens.AccessToken
}

func (h *isolationRedteamHarness) request(t *testing.T, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.mux.ServeHTTP(rec, req)
	return rec
}

func isolationRedteamData[T any](t *testing.T, rec *httptest.ResponseRecorder, status int) T {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("expected HTTP %d, got %d", status, rec.Code)
	}
	var envelope struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func TestRedTeamContinuationIsolationObservedWeaknessAnsweredPrivateInquiryPublishedToOtherStudents(t *testing.T) {
	for _, mode := range []struct {
		name   string
		cached bool
	}{{"sql_only", false}, {"shared_ram_cache", true}} {
		t.Run(mode.name, func(t *testing.T) {
			h := newIsolationRedteamHarness(t, mode.cached)
			_, victimToken := h.user(t, constants.RoleTypeStudent)
			_, attackerToken := h.user(t, constants.RoleTypeStudent)
			_, advisorToken := h.user(t, constants.RoleTypeAdvisor)
			questionMarker := "Private academic hardship " + isolationRedteamRandom(t)
			replyMarker := "Private accommodation decision " + isolationRedteamRandom(t)
			contextMarker := "Unpublished context " + isolationRedteamRandom(t)
			inq := isolationRedteamData[response.InquiryResponse](t, h.request(t, http.MethodPost, "/api/inquiries", victimToken, request.CreateInquiryRequest{
				StudentName: "Local Student", StudentCode: "LOCAL01", StudentClass: "LOCAL_CLASS",
				Question: "My academic hardship concerns: " + questionMarker,
				Context:  contextMarker,
			}), http.StatusCreated)
			if rec := h.request(t, http.MethodGet, "/api/inquiries/"+inq.ID, attackerToken, nil); rec.Code != http.StatusForbidden {
				t.Fatalf("private inquiry control: expected 403, got %d", rec.Code)
			}
			if rec := h.request(t, http.MethodPost, "/api/rag/search", "", request.RAGSearchRequest{Query: "academic hardship"}); rec.Code != http.StatusUnauthorized {
				t.Fatalf("unauthenticated RAG control: expected 401, got %d", rec.Code)
			}
			before := h.request(t, http.MethodPost, "/api/rag/search", attackerToken, request.RAGSearchRequest{Query: "academic hardship"})
			if before.Code != http.StatusOK || strings.Contains(before.Body.String(), questionMarker) {
				t.Fatal("unanswered private question unexpectedly available")
			}
			answered := isolationRedteamData[response.InquiryResponse](t, h.request(t, http.MethodPost, "/api/advisor/inquiries/"+inq.ID+"/answer", advisorToken, request.AnswerInquiryRequest{Reply: replyMarker}), http.StatusOK)
			if answered.KnowledgeChunkID == "" {
				t.Fatal("answer did not create a knowledge chunk")
			}
			for i := 0; i < 2; i++ {
				rec := h.request(t, http.MethodPost, "/api/rag/search", attackerToken, request.RAGSearchRequest{Query: "academic hardship", TopK: 5})
				if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), questionMarker) || !strings.Contains(rec.Body.String(), replyMarker) {
					t.Fatalf("question/reply disclosure not reproduced on retrieval %d: HTTP %d", i, rec.Code)
				}
				if strings.Contains(rec.Body.String(), contextMarker) {
					t.Fatal("unexpected extra context disclosure")
				}
			}
			if rec := h.request(t, http.MethodGet, "/api/inquiries/"+inq.ID, attackerToken, nil); rec.Code != http.StatusForbidden {
				t.Fatal("private inquiry control changed after answering")
			}
			detail := h.request(t, http.MethodGet, "/api/rag/documents/doc_teacher_qa", attackerToken, nil)
			if detail.Code != http.StatusForbidden {
				t.Fatalf("expected 403 when student retrieves advisor QA document, got %d", detail.Code)
			}
			if rec := h.request(t, http.MethodPost, "/api/advisor/inquiries/"+inq.ID+"/expire", advisorToken, request.ExpireInquiryRequest{Reason: "Local expiry control"}); rec.Code != http.StatusOK {
				t.Fatal("local expiry failed")
			}
			detail = h.request(t, http.MethodGet, "/api/rag/documents/doc_teacher_qa", attackerToken, nil)
			if detail.Code != http.StatusForbidden {
				t.Fatalf("expected 403 when student retrieves expired advisor QA document, got %d", detail.Code)
			}
		})
	}
}

func TestRedTeamContinuationIsolationControlsAuthenticatedUserScoping(t *testing.T) {
	h := newIsolationRedteamHarness(t, true)
	victimID, victimToken := h.user(t, constants.RoleTypeStudent)
	attackerID, attackerToken := h.user(t, constants.RoleTypeStudent)
	ctx := context.Background()
	gradeMarker := "Private course " + isolationRedteamRandom(t)
	profileMarker := "Private profile " + isolationRedteamRandom(t)
	memoryMarker := "Private memory " + isolationRedteamRandom(t)
	grade, err := h.transcript.BatchSaveTranscriptGrades(ctx, victimID, []request.StudentCourseGradeInput{{CourseCode: "LOCAL101", CourseName: gradeMarker, Credits: 3, Score10: 8, Score4: 3, GradeLetter: "B", IsPassed: true}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.memory.UpdateStudentProfileFromDto(ctx, victimID, &request.UpsertStudentProfileDto{AdvisorNotes: profileMarker}); err != nil {
		t.Fatal(err)
	}
	memory, err := h.memory.CreateUserMemoryFromDto(ctx, victimID, &request.SaveUserMemoryDto{Category: "GENERAL", MemoryKey: "academic_preference", MemoryValue: memoryMarker})
	if err != nil {
		t.Fatal(err)
	}
	notif, err := h.notifs.CreateNotification(ctx, &models.UserNotification{ID: isolationRedteamRandom(t), UserID: victimID, Title: "Local notification", Content: memoryMarker, Type: "LOCAL"})
	if err != nil {
		t.Fatal(err)
	}
	for _, endpoint := range []string{"/api/transcript/my", "/api/curriculum/my-roadmap", "/api/curriculums/my-roadmap", "/api/user/profile", "/api/user/memories", "/api/student/notifications"} {
		t.Run(endpoint, func(t *testing.T) {
			for i := 0; i < 2; i++ {
				owner := h.request(t, http.MethodGet, endpoint, victimToken, nil)
				if owner.Code != http.StatusOK {
					t.Fatalf("owner warmup failed: HTTP %d", owner.Code)
				}
				foreign := h.request(t, http.MethodGet, endpoint+"?user_id="+victimID, attackerToken, nil)
				if foreign.Code != http.StatusOK {
					t.Fatalf("attacker scoped read failed: HTTP %d", foreign.Code)
				}
				for _, marker := range []string{gradeMarker, profileMarker, memoryMarker} {
					if strings.Contains(foreign.Body.String(), marker) {
						t.Fatal("foreign marker crossed user-scoped read")
					}
				}
			}
		})
	}
	foreignRoadmap := isolationRedteamData[response.StudentPersonalRoadmapDataResponse](t, h.request(t, http.MethodGet, "/api/curriculum/my-roadmap", attackerToken, nil), http.StatusOK)
	if foreignRoadmap.Summary.TotalCreditsCompleted != 0 || foreignRoadmap.Summary.CumulativeGPA10 != 0 {
		t.Fatal("foreign grade aggregate crossed roadmap boundary")
	}
	ownRoadmap := isolationRedteamData[response.StudentPersonalRoadmapDataResponse](t, h.request(t, http.MethodGet, "/api/curriculum/my-roadmap", victimToken, nil), http.StatusOK)
	if ownRoadmap.Summary.TotalCreditsCompleted != 3 || ownRoadmap.Summary.CumulativeGPA10 != 8 {
		t.Fatal("roadmap aggregate control did not use owner grades")
	}
	if rec := h.request(t, http.MethodDelete, "/api/user/memories/"+memory.ID, attackerToken, nil); rec.Code == http.StatusOK {
		t.Fatal("foreign memory deletion returned success")
	}
	memories, err := h.memory.ListUserMemories(ctx, victimID)
	if err != nil || len(memories) != 1 || memories[0].ID != memory.ID {
		t.Fatal("foreign memory deletion changed owner data")
	}
	if rec := h.request(t, http.MethodPatch, "/api/student/notifications/"+notif.ID+"/read", attackerToken, nil); rec.Code != http.StatusOK {
		t.Fatal("notification no-op control failed")
	}
	if rec := h.request(t, http.MethodPost, "/api/student/notifications/read-all", attackerToken, nil); rec.Code != http.StatusOK {
		t.Fatal("notification bulk control failed")
	}
	unread, err := h.notifs.CountUnreadNotifications(ctx, victimID)
	if err != nil || unread != 1 {
		t.Fatal("foreign read receipt changed owner notification")
	}
	foreignGrade := h.request(t, http.MethodPost, "/api/transcript/batch-save", attackerToken, request.BatchSaveTranscriptRequest{Grades: []request.StudentCourseGradeInput{{ID: grade.Summary.Grades[0].ID, CourseCode: "LOCAL101", Credits: 3, Score10: 1}}})
	if foreignGrade.Code == http.StatusOK {
		t.Fatal("foreign primary-key collision unexpectedly accepted")
	}
	ownerGrades, err := h.transcript.GetTranscriptSummary(ctx, victimID)
	if err != nil || len(ownerGrades.Grades) != 1 || ownerGrades.Grades[0].Score10 != 8 {
		t.Fatal("foreign primary-key collision changed owner transcript")
	}
	attackerGrades, err := h.transcript.GetTranscriptSummary(ctx, attackerID)
	if err != nil || len(attackerGrades.Grades) != 0 {
		t.Fatal("failed foreign primary-key submission was not rolled back")
	}
}

package services_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"tluagent-web/internal/controllers"
	"tluagent-web/internal/dtos/request"
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/internal/models"
	"tluagent-web/internal/routes"
	"tluagent-web/internal/services"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/database"
)

func TestRedTeamContinuationObservedWeaknessUserManagerGrantsUnpossessedRole(t *testing.T) {
	for _, mode := range []string{"create_with_privileged_role", "assign_privileged_role"} {
		t.Run(mode, func(t *testing.T) {
			h := newOwnershipRedteamHarness(t)
			ctx := context.Background()
			delegated, err := h.roles.Create(ctx, sqlc.CreateRoleParams{ID: uuid.NewString(), Name: "LOCAL_ACCOUNT_OPERATOR"})
			if err != nil {
				t.Fatal(err)
			}
			studentRole, err := h.roles.GetByName(ctx, constants.RoleTypeStudent.String())
			if err != nil {
				t.Fatal(err)
			}
			studentPerms, err := h.roles.GetRolePermissions(ctx, studentRole.ID)
			if err != nil {
				t.Fatal(err)
			}
			delegatedPerms := []*models.RolePermissionEntity{
				{PermissionKey: constants.PermUserManage, Effect: "allow"},
				{PermissionKey: constants.PermRoleManage, Effect: "allow"},
			}
			for _, sp := range studentPerms {
				delegatedPerms = append(delegatedPerms, &models.RolePermissionEntity{
					PermissionKey: sp.PermissionKey,
					Effect:        sp.Effect,
				})
			}
			if err := h.roles.ReplaceRolePermissions(ctx, delegated.ID, delegatedPerms); err != nil {
				t.Fatal(err)
			}
			privileged, err := h.roles.Create(ctx, sqlc.CreateRoleParams{ID: uuid.NewString(), Name: "LOCAL_CHAT_AUDITOR"})
			if err != nil {
				t.Fatal(err)
			}
			if err := h.roles.ReplaceRolePermissions(ctx, privileged.ID, []*models.RolePermissionEntity{{PermissionKey: constants.PermChatManage, Effect: "allow"}}); err != nil {
				t.Fatal(err)
			}
			if err := h.perms.Reload(ctx); err != nil {
				t.Fatal(err)
			}
			_, token := h.user(t, delegated.Name)
			victim, _ := h.user(t, constants.RoleTypeStudent.String())
			sessionSvc := services.NewChatSessionService(h.chat, h.users)
			session, err := sessionSvc.CreateSession(ctx, victim.ID, "Private local session", "")
			if err != nil {
				t.Fatal(err)
			}
			marker := "Private local academic record " + uuid.NewString()
			if err := h.chat.CreateMessage(ctx, &models.ChatMessageEntity{ID: uuid.NewString(), ConversationID: session.ID, Sender: "user", Content: marker}); err != nil {
				t.Fatal(err)
			}
			mux, _ := h.chatMux(&ownershipRedteamProvider{})
			routes.RegisterRoleRoutes(mux, controllers.NewRoleController(services.NewRoleService(h.roles, h.perms, database.NewTxManager(h.db))), h.users, h.perms)
			userSvc := services.NewUserService(h.users, h.roles, h.settings, database.NewTxManager(h.db), h.perms)
			routes.RegisterUserRoutes(mux, controllers.NewUserController(userSvc), h.users, h.perms)
			routes.RegisterAuthRoutes(mux, controllers.NewAuthController(h.auth), h.users)
			if rec := ownershipRedteamRequest(t, mux, http.MethodGet, "/api/admin/chats/"+session.ID, token, nil); rec.Code != http.StatusForbidden {
				t.Fatalf("audit control: %d", rec.Code)
			}
			control := ownershipRedteamRequest(t, mux, http.MethodPost, "/api/roles", token, request.CreateRoleDto{Name: "LOCAL_FORBIDDEN_GRANT", Permissions: []request.RolePermissionDto{{PermissionKey: constants.PermChatManage, Effect: "allow"}}})
			if control.Code != http.StatusForbidden {
				t.Fatalf("explicit delegation control: %d", control.Code)
			}
			password := "P" + ownershipRedteamRandom(t) + "7"
			email := uuid.NewString() + "@example.invalid"
			roleIDs := []string{privileged.ID}
			if mode == "assign_privileged_role" {
				student, err := h.roles.GetByName(ctx, constants.RoleTypeStudent.String())
				if err != nil {
					t.Fatal(err)
				}
				roleIDs = []string{student.ID}
			}
			created := ownershipRedteamRequest(t, mux, http.MethodPost, "/api/users", token, request.CreateUserDto{Email: email, Password: password, FullName: "Local controlled account", RoleIDs: roleIDs})
			if mode == "create_with_privileged_role" {
				if created.Code != http.StatusForbidden {
					t.Fatalf("expected 403 when creating account with unpossessed role, got %d", created.Code)
				}
				return
			}
			if created.Code != http.StatusCreated {
				t.Fatalf("create controlled account: %d", created.Code)
			}
			var envelope struct {
				Data response.UserResponse `json:"data"`
			}
			if err := json.Unmarshal(created.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			changed := ownershipRedteamRequest(t, mux, http.MethodPut, "/api/users/"+envelope.Data.ID+"/roles", token, request.ChangeRoleDto{Roles: []string{privileged.ID}})
			if changed.Code != http.StatusForbidden {
				t.Fatalf("expected 403 when assigning unpossessed role, got %d", changed.Code)
			}
		})
	}
}

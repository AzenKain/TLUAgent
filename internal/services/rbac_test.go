package services

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"tluagent-web/internal/repositories"
	"tluagent-web/pkg/cache"
	"tluagent-web/pkg/constants"
	"tluagent-web/pkg/database"
)

func setupTestDB(t *testing.T) (*sql.DB, repositories.RoleRepository, PermissionCache) {
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("Open in-memory db: %v", err)
	}

	if err := database.ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema failed: %v", err)
	}

	c := cache.NewTheineCache(10 << 20)
	roleRepo := repositories.NewRoleRepository(db, c)
	permCache := NewPermissionCache(roleRepo)

	if err := permCache.Reload(context.Background()); err != nil {
		t.Fatalf("PermissionCache Reload failed: %v", err)
	}

	return db, roleRepo, permCache
}

func TestRBAC_DefaultPermissions(t *testing.T) {
	db, _, permCache := setupTestDB(t)
	defer db.Close()

	if !permCache.CanRoles(nil, []constants.RoleType{constants.RoleTypeAdmin}, constants.PermAdminAccess, nil) {
		t.Errorf("Admin should have PermAdminAccess")
	}
	if !permCache.CanRoles(nil, []constants.RoleType{constants.RoleTypeAdmin}, constants.PermRegulationWrite, nil) {
		t.Errorf("Admin should have PermRegulationWrite")
	}

	studentRoles := []constants.RoleType{constants.RoleTypeStudent}
	if !permCache.CanRoles(nil, studentRoles, constants.PermRegulationRead, nil) {
		t.Errorf("Student should have PermRegulationRead")
	}
	if !permCache.CanRoles(nil, studentRoles, constants.PermChatAsk, nil) {
		t.Errorf("Student should have PermChatAsk")
	}
	if !permCache.CanRoles(nil, studentRoles, constants.PermChatPersist, nil) {
		t.Errorf("Student should have PermChatPersist")
	}
	if permCache.CanRoles(nil, studentRoles, constants.PermAdminAccess, nil) {
		t.Errorf("Student should NOT have PermAdminAccess")
	}
	if permCache.CanRoles(nil, studentRoles, constants.PermRegulationWrite, nil) {
		t.Errorf("Student should NOT have PermRegulationWrite")
	}

	guestRoles := []constants.RoleType{constants.RoleTypeGuest}
	if !permCache.CanRoles(nil, guestRoles, constants.PermRegulationRead, nil) {
		t.Errorf("Guest should have PermRegulationRead")
	}
	if !permCache.CanRoles(nil, guestRoles, constants.PermChatAsk, nil) {
		t.Errorf("Guest should have PermChatAsk")
	}
	if permCache.CanRoles(nil, guestRoles, constants.PermChatPersist, nil) {
		t.Errorf("Guest should NOT have PermChatPersist")
	}

	bannedRoles := []constants.RoleType{constants.RoleTypeAdmin, constants.RoleTypeBanned}
	if permCache.CanRoles(nil, bannedRoles, constants.PermRegulationRead, nil) {
		t.Errorf("Banned user should NOT have any permission")
	}
}

func TestRBAC_DescribeRoles(t *testing.T) {
	db, roleRepo, permCache := setupTestDB(t)
	defer db.Close()

	studentRole, err := roleRepo.GetByName(context.Background(), constants.RoleTypeStudent.String())
	if err != nil || studentRole == nil {
		t.Fatalf("GetByName Student failed: %v", err)
	}

	descriptions := permCache.DescribeRoles([]string{studentRole.ID})
	if len(descriptions) != 1 {
		t.Fatalf("Expected 1 described role, got %d", len(descriptions))
	}
	if descriptions[0].Name != "STUDENT" {
		t.Errorf("Expected role STUDENT, got %s", descriptions[0].Name)
	}
	if len(descriptions[0].Permissions) == 0 {
		t.Errorf("Expected student to have permissions populated")
	}
}

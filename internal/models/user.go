package models

import (
	"tluagent-web/internal/dtos/response"
	"tluagent-web/internal/gen/sqlc"
	"tluagent-web/pkg/convert"
)

type UserEntity struct {
	ID           string        `json:"id"`
	Email        string        `json:"email"`
	PasswordHash string        `json:"-"`
	FullName     string        `json:"full_name"`
	StudentCode  string        `json:"student_code"`
	AvatarUrl    string        `json:"avatar_url"`
	AuthProvider string        `json:"auth_provider"`
	TokenVersion int32         `json:"token_version"`
	RefreshToken string        `json:"-"`
	IsDeleted    bool          `json:"is_deleted"`
	CreatedAt    string        `json:"created_at"`
	UpdatedAt    string        `json:"updated_at"`
	Roles        []*RoleSimple `json:"roles"`
}

func (u *UserEntity) ToResponse() *response.UserResponse {
	if u == nil {
		return nil
	}
	return &response.UserResponse{
		ID:           u.ID,
		Email:        u.Email,
		FullName:     u.FullName,
		StudentCode:  u.StudentCode,
		AvatarUrl:    u.AvatarUrl,
		AuthProvider: u.AuthProvider,
		TokenVersion: u.TokenVersion,
		Roles:        RolesToResponse(u.Roles),
		IsDeleted:    u.IsDeleted,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func UsersEntityToResponse(users []*UserEntity) []*response.UserResponse {
	out := make([]*response.UserResponse, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		out = append(out, user.ToResponse())
	}
	return out
}

func (u *UserEntity) IsAdmin() bool {
	if u == nil {
		return false
	}
	for _, r := range u.Roles {
		if r != nil && (r.IsAdmin || r.Name == "ADMIN") {
			return true
		}
	}
	return false
}

func (u *UserEntity) FromSqlc(row sqlc.User) *UserEntity {
	u.ID = row.ID
	u.Email = row.Email
	u.FullName = convert.NullStringToString(row.FullName)
	u.StudentCode = convert.NullStringToString(row.StudentCode)
	u.AvatarUrl = convert.NullStringToString(row.AvatarUrl)
	u.PasswordHash = convert.NullStringToString(row.PasswordHash)
	u.AuthProvider = row.AuthProvider
	u.TokenVersion = int32(row.TokenVersion)
	u.RefreshToken = convert.NullStringToString(row.RefreshToken)
	u.IsDeleted = row.IsDeleted != 0
	u.CreatedAt = row.CreatedAt
	u.UpdatedAt = row.UpdatedAt
	u.Roles = []*RoleSimple{}
	return u
}

type UserEntities []*UserEntity

func (e *UserEntities) FromSqlc(rows []sqlc.User) []*UserEntity {
	slice := make([]*UserEntity, len(rows))
	flat := make([]UserEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}

package response

// AuthResponse carries the authenticated user; tokens travel via HttpOnly cookies only.
type AuthResponse struct {
	AccessToken  string        `json:"-"`
	RefreshToken string        `json:"-"`
	User         *UserResponse `json:"user,omitempty"`
}

type UserResponse struct {
	ID           string                `json:"id"`
	Email        string                `json:"email"`
	FullName     string                `json:"full_name"`
	StudentCode  string                `json:"student_code"`
	AvatarUrl    string                `json:"avatar_url"`
	AuthProvider string                `json:"auth_provider"`
	TokenVersion int32                 `json:"token_version"`
	Roles        []*RoleSimpleResponse `json:"roles"`
	IsOwner      bool                  `json:"is_owner"`
	IsDeleted    bool                  `json:"is_deleted"`
	CreatedAt    string                `json:"created_at"`
	UpdatedAt    string                `json:"updated_at"`
}

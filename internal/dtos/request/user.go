package request

type CreateUserDto struct {
	Email       string   `json:"email" validate:"required,min=5,max=255,email"`
	Password    string   `json:"password" validate:"required,min=10,max=100,password_policy"`
	FullName    string   `json:"full_name" validate:"required,min=2,max=100"`
	StudentCode string   `json:"student_code,omitempty" validate:"omitempty,max=50"`
	RoleIDs     []string `json:"role_ids,omitempty" validate:"omitempty,dive,uuid"`
}

type UpdateProfileDto struct {
	FullName    *string `json:"full_name,omitempty" validate:"omitempty,min=2,max=100"`
	StudentCode *string `json:"student_code,omitempty" validate:"omitempty,max=50"`
	AvatarUrl   *string `json:"avatar_url,omitempty" validate:"omitempty,image_url"`
}

type AdminUpdateProfileDto = UpdateProfileDto

type ChangePasswordDto struct {
	OldPassword string `json:"old_password" validate:"required,min=6,max=100"`
	NewPassword string `json:"new_password" validate:"required,min=10,max=100,password_policy,nefield=OldPassword"`
}

type ResetPasswordDto struct {
	NewPassword string `json:"new_password" validate:"required,min=10,max=100,password_policy"`
}

type ChangeRoleDto struct {
	Roles []string `json:"role_ids" validate:"required,min=1,dive,uuid"`
}

type SearchUserDto struct {
	PaginationDto
	Search    string `json:"search,omitempty" query:"search" validate:"omitempty,min=1,max=200"`
	IsDeleted *bool  `json:"is_deleted,omitempty" query:"is_deleted" validate:"omitempty"`
	RoleID    string `json:"role_id,omitempty" query:"role_id" validate:"omitempty,uuid"`
}

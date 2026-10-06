package request

type SignInDto struct {
	Email    string `json:"email" validate:"required,min=5,max=255,email"`
	Password string `json:"password" validate:"required,min=6,max=100"`
}

type SetupDto struct {
	Email       string `json:"email" validate:"required,min=5,max=255,email"`
	Password    string `json:"password" validate:"required,min=10,max=100,password_policy"`
	FullName    string `json:"full_name" validate:"required,min=2,max=100"`
	StudentCode string `json:"student_code,omitempty" validate:"omitempty,max=50"`
}

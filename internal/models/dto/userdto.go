package dto

type UserSignupRequestDTO struct {
	Username string `json:"username" validate:"required,min=8,max=64"`
	Password string `json:"password" validate:"required,min=8,max=64"`
}

type UserSignupResponseDTO struct {
	Message string `json:"message"`
	UserID  string `json:"user_id,omitempty"`
}

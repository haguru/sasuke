package dto

type LoginRequestDTO struct {
	Username string `json:"username" validate:"required,min=8,max=64"`
	Password string `json:"password" validate:"required,min=8,max=64"`
}

type LoginResponseDTO struct {
	Message string `json:"message"`
	// Optionally include a token if you return it in the response body
	// Token   string `json:"token,omitempty"`
}

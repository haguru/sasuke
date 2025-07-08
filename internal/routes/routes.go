package routes

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/haguru/sasuke/internal/auth"
	"github.com/haguru/sasuke/internal/interfaces"
	"github.com/haguru/sasuke/internal/models/dto"
	"github.com/haguru/sasuke/internal/userservice"

	structValidator "github.com/go-playground/validator/v10"
)

type Route struct {
	Metrics     interfaces.Metrics
	UserService *userservice.UserService
	PrivateKey  *ecdsa.PrivateKey
	validator   *structValidator.Validate
}

// NewRoute creates a new Route instance.
func NewRoute(metrics interfaces.Metrics, userService *userservice.UserService,
	privateKey *ecdsa.PrivateKey, validator *structValidator.Validate,
) *Route {

	return &Route{
		Metrics:     metrics,
		UserService: userService,
		PrivateKey:  privateKey,
		validator:   validator,
	}
}

// Signup handles user signup requests.
func (r *Route) Signup(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		r.errorResponse(w, fmt.Errorf("method %s not allowed", req.Method), "Method not allowed")
		return
	}

	if r.Metrics != nil {
		r.Metrics.IncCounter(SignupRequestsTotal)
	}

	if req.Header.Get(ContentType) != ContentTypeJson {
		w.WriteHeader(http.StatusBadRequest)
		r.errorResponse(w, fmt.Errorf("invalid content-type: %s", req.Header.Get(ContentType)), "Request Content-Type must be application/json")
		if r.Metrics != nil {
			r.Metrics.IncCounter(SignupErrorsTotal)
		}
		return
	}

	signupRequest := &dto.UserSignupRequestDTO{}
	err := json.NewDecoder(req.Body).Decode(signupRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		r.errorResponse(w, err, "Invalid request body")
		if r.Metrics != nil {
			r.Metrics.IncCounter(SignupErrorsTotal)
		}
		return
	}

	if err := r.validator.Struct(signupRequest); err != nil {
		// Validation failed, handle the error
		errors := err.(structValidator.ValidationErrors)
		w.WriteHeader(http.StatusBadRequest)
		r.errorResponse(w, fmt.Errorf("invalid signup data: %s", errors), "Signup data validation failed")
		if r.Metrics != nil {
			r.Metrics.IncCounter(SignupErrorsTotal)
		}
		return
	}

	var startTime time.Time
	if r.Metrics != nil {
		startTime = time.Now()
	}

	userID, err := r.UserService.RegisterUser(req.Context(), signupRequest.Username, signupRequest.Password)
	if err != nil {
		w.WriteHeader(http.StatusConflict)
		r.errorResponse(w, err, "Failed to register user")
		if r.Metrics != nil {
			r.Metrics.IncCounter(SignupErrorsTotal)
		}
		return
	}

	if r.Metrics != nil {
		r.Metrics.IncCounter(SignupSuccessTotal)
		duration := time.Since(startTime).Seconds()
		r.Metrics.ObserveHistogram(SignupDurationSeconds, duration)
	}

	w.Header().Set(ContentType, ContentTypeJson)
	w.WriteHeader(http.StatusCreated)

	response := &dto.UserSignupResponseDTO{
		Message: fmt.Sprintf("User created successfully with ID: %s", userID),
		UserID:  userID,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		r.errorResponse(w, err, "Failed to encode response")
		if r.Metrics != nil {
			r.Metrics.IncCounter(SignupErrorsTotal)
		}
		return
	}
}

// Login handles user login requests.
func (r *Route) Login(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		r.errorResponse(w, fmt.Errorf("method %s not allowed", req.Method), "Method not allowed")
		if r.Metrics != nil {
			r.Metrics.IncCounter(LoginFailedTotal)
		}
		return
	}

	if r.Metrics != nil {
		r.Metrics.IncCounter(LoginRequestsTotal)
	}

	if req.Header.Get(ContentType) != ContentTypeJson {
		w.WriteHeader(http.StatusBadRequest)
		r.errorResponse(w, fmt.Errorf("invalid content-type: %s", req.Header.Get(ContentType)), "Content-Type must be application/json")
		if r.Metrics != nil {
			r.Metrics.IncCounter(LoginFailedTotal)
		}
		return
	}

	loginRequest := &dto.LoginRequestDTO{}
	err := json.NewDecoder(req.Body).Decode(loginRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		r.errorResponse(w, err, "Invalid request body")
		if r.Metrics != nil {
			r.Metrics.IncCounter(LoginFailedTotal)
		}
		return
	}

	if err := r.validator.Struct(loginRequest); err != nil {
		// Validation failed, handle the error
		errors := err.(structValidator.ValidationErrors)
		w.WriteHeader(http.StatusBadRequest)
		r.errorResponse(w, fmt.Errorf("invalid login data: %s", errors), "Login data validation failed")
		if r.Metrics != nil {
			r.Metrics.IncCounter(LoginFailedTotal)
		}
		return
	}

	var startTime time.Time
	if r.Metrics != nil {
		startTime = time.Now()
	}

	authenticated, err := r.UserService.AuthenticateUser(req.Context(), loginRequest.Username, loginRequest.Password)
	if err != nil || !authenticated {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		r.errorResponse(w, err, "Invalid username or password")
		if r.Metrics != nil {
			r.Metrics.IncCounter(LoginFailedTotal)
			duration := time.Since(startTime).Seconds()
			r.Metrics.ObserveHistogram(LoginDurationSeconds, duration)
		}
		return
	}

	if r.Metrics != nil {
		r.Metrics.IncCounter(LoginSuccessTotal)
		duration := time.Since(startTime).Seconds()
		r.Metrics.ObserveHistogram(LoginDurationSeconds, duration)
	}

	sessionToken, err := auth.CreateToken(loginRequest.Username, r.PrivateKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		r.errorResponse(w, err, "Failed to generate session token")
		if r.Metrics != nil {
			r.Metrics.IncCounter(LoginFailedTotal)
		}
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
	})

	w.Header().Set(ContentType, ContentTypeJson)

	w.WriteHeader(http.StatusOK)
	response := &dto.LoginResponseDTO{
		Message: "Login successful",
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		r.errorResponse(w, err, "Failed to encode response")
		if r.Metrics != nil {
			r.Metrics.IncCounter(LoginFailedTotal)
		}
		return
	}
}

// Create route
// TODO complete API
func (r *Route) Create(w http.ResponseWriter, req *http.Request) {
	w.Header().Set(ContentType, ContentTypeJson)
	w.WriteHeader(http.StatusNotImplemented)
	r.errorResponse(w, fmt.Errorf("create route not implemented"), "Create route has not been implemented yet")
}

func (r *Route) errorResponse(w http.ResponseWriter, err error, message string) {
	jsonResponse := map[string]string{
		"error":   err.Error(),
		"message": message,
	}
	_ = json.NewEncoder(w).Encode(jsonResponse)
}

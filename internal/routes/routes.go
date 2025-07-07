package routes

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/haguru/sasuke/internal/auth"
	"github.com/haguru/sasuke/internal/interfaces"
	"github.com/haguru/sasuke/internal/models"
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Metrics != nil {
		r.Metrics.IncCounter(SignupRequestsTotal)
	}

	if req.Header.Get(ContentType) != ContentTypeJson {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		if r.Metrics != nil {
			r.Metrics.IncCounter(SignupErrorsTotal)
		}
		return
	}

	signupRequest := &models.User{}
	err := json.NewDecoder(req.Body).Decode(signupRequest)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		if r.Metrics != nil {
			r.Metrics.IncCounter(SignupErrorsTotal)
		}
		return
	}

	if err := r.validator.Struct(signupRequest); err != nil {
		// Validation failed, handle the error
		errors := err.(structValidator.ValidationErrors)
		http.Error(w, fmt.Sprintf("validation error: %s", errors), http.StatusBadRequest)
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
		http.Error(w, "Failed to register user", http.StatusConflict)
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
	message := fmt.Sprintf("User created successfully with ID: %s", userID)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": message}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		if r.Metrics != nil {
			r.Metrics.IncCounter(SignupErrorsTotal)
		}
		return
	}
}

// Login handles user login requests.
func (r *Route) Login(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		if r.Metrics != nil {
			r.Metrics.IncCounter(LoginFailedTotal)
		}
		return
	}

	if r.Metrics != nil {
		r.Metrics.IncCounter(LoginRequestsTotal)
	}

	if req.Header.Get(ContentType) != ContentTypeJson {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		if r.Metrics != nil {
			r.Metrics.IncCounter(LoginFailedTotal)
		}
		return
	}

	loginRequest := &models.User{}
	err := json.NewDecoder(req.Body).Decode(loginRequest)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		if r.Metrics != nil {
			r.Metrics.IncCounter(LoginFailedTotal)
		}
		return
	}

	if err := r.validator.Struct(loginRequest); err != nil {
		// Validation failed, handle the error
		errors := err.(structValidator.ValidationErrors)
		http.Error(w, fmt.Sprintf("validation error: %s", errors), http.StatusBadRequest)
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
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
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
		http.Error(w, "Failed to generate session token", http.StatusInternalServerError)
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
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Login successful"}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
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
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "Create route has not been implemented yet",
	})
}

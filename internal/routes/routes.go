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
	Logger      interfaces.Logger
}

// NewRoute creates a new Route instance.
func NewRoute(metrics interfaces.Metrics, userService *userservice.UserService,
	privateKey *ecdsa.PrivateKey, validator *structValidator.Validate,
	logger interfaces.Logger,
) *Route {
	return &Route{
		Metrics:     metrics,
		UserService: userService,
		PrivateKey:  privateKey,
		validator:   validator,
		Logger:      logger,
	}
}

// Signup handles user signup requests.
func (r *Route) Signup(w http.ResponseWriter, req *http.Request) {
	r.Logger.Info("Signup request received", "method", req.Method, "path", req.URL.Path, "remote_addr", req.RemoteAddr)
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		r.Logger.Warn(ErrMethodNotAllowed, "method", req.Method, "path", req.URL.Path, "remote_addr", req.RemoteAddr)
		r.errorResponse(w, fmt.Errorf("method %s not allowed", req.Method), "Method not allowed")
		return
	}

	if r.Metrics != nil {
		r.Metrics.IncCounter(SignupRequestsTotal)
	}

	if req.Header.Get(ContentType) != ContentTypeJson {
		w.WriteHeader(http.StatusBadRequest)
		r.Logger.Warn(ErrInvalidContentType, ContentType, req.Header.Get(ContentType), "path", req.URL.Path, "remote_addr", req.RemoteAddr)
		r.errorResponse(w, fmt.Errorf(ErrInvalidContentTypeFormat, req.Header.Get(ContentType)), ErrInvalidContentType)
		if r.Metrics != nil {
			r.Metrics.IncCounter(SignupErrorsTotal)
		}
		return
	}

	signupRequest := &dto.UserSignupRequestDTO{}
	err := json.NewDecoder(req.Body).Decode(signupRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		r.Logger.Error(ErrFailedToDecodeRequest, "error", err, "path", req.URL.Path, "remote_addr", req.RemoteAddr)
		r.errorResponse(w, err, ErrFailedToDecodeRequest)
		if r.Metrics != nil {
			r.Metrics.IncCounter(SignupErrorsTotal)
		}
		return
	}

	if err := r.validator.Struct(signupRequest); err != nil {
		errors := err.(structValidator.ValidationErrors)
		w.WriteHeader(http.StatusBadRequest)
		r.Logger.Warn(ErrValidationFailed, "error", errors, "username", signupRequest.Username, "path", req.URL.Path, "remote_addr", req.RemoteAddr)
		r.errorResponse(w, fmt.Errorf("invalid signup data: %w", errors), ErrValidationFailed)
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
		r.Logger.Error(ErrFailedToRegisterUser, "error", err, "username", signupRequest.Username, "path", req.URL.Path, "remote_addr", req.RemoteAddr)
		r.errorResponse(w, err, ErrFailedToRegisterUser)
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
		Message: fmt.Sprintf(MsgUserCreatedFormat, userID),
		UserID:  userID,
	}

	r.Logger.Info(MsgUserCreatedFormat, "userID", userID, "username", signupRequest.Username, "path", req.URL.Path, "remote_addr", req.RemoteAddr)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		r.Logger.Error(ErrFailedToEncodeResponse, "error", err, "userID", userID, "username", signupRequest.Username, "path", req.URL.Path, "remote_addr", req.RemoteAddr)
		r.errorResponse(w, err, ErrFailedToEncodeResponse)
		if r.Metrics != nil {
			r.Metrics.IncCounter(SignupErrorsTotal)
		}
		return
	}
}

// Login handles user login requests.
func (r *Route) Login(w http.ResponseWriter, req *http.Request) {
	r.Logger.Info("Login request received", "method", req.Method, "path", req.URL.Path, "remote_addr", req.RemoteAddr)
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		r.Logger.Warn(ErrMethodNotAllowed, "method", req.Method, "path", req.URL.Path, "remote_addr", req.RemoteAddr)
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
		r.Logger.Warn(ErrInvalidContentType, ContentType, req.Header.Get(ContentType), "path", req.URL.Path, "remote_addr", req.RemoteAddr)
		r.errorResponse(w, fmt.Errorf(ErrInvalidContentTypeFormat, req.Header.Get(ContentType)), ErrInvalidContentType)
		if r.Metrics != nil {
			r.Metrics.IncCounter(LoginFailedTotal)
		}
		return
	}

	loginRequest := &dto.LoginRequestDTO{}
	err := json.NewDecoder(req.Body).Decode(loginRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		r.Logger.Error(ErrValidationFailed, "error", err, "path", req.URL.Path, "remote_addr", req.RemoteAddr)
		r.errorResponse(w, err, ErrValidationFailed)
		if r.Metrics != nil {
			r.Metrics.IncCounter(LoginFailedTotal)
		}
		return
	}

	if err := r.validator.Struct(loginRequest); err != nil {
		errors := err.(structValidator.ValidationErrors)
		w.WriteHeader(http.StatusBadRequest)
		r.Logger.Warn(ErrValidationFailed, "error", errors, "username", loginRequest.Username, "path", req.URL.Path, "remote_addr", req.RemoteAddr)
		r.errorResponse(w, fmt.Errorf("invalid login data: %w", errors), ErrValidationFailed)
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
		r.Logger.Warn("Authentication failed for user", "username", loginRequest.Username, "error", err, "path", req.URL.Path, "remote_addr", req.RemoteAddr)
		r.errorResponse(w, err, ErrInvalidCredentials)
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
		r.Logger.Error(ErrFailedToGenerateToken, "error", err, "username", loginRequest.Username, "path", req.URL.Path, "remote_addr", req.RemoteAddr)
		r.errorResponse(w, err, ErrFailedToGenerateToken)
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
	r.Logger.Info(MsgLoginSuccessful, "username", loginRequest.Username, "path", req.URL.Path, "remote_addr", req.RemoteAddr)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		r.Logger.Error(ErrFailedToEncodeResponse, "error", err, "username", loginRequest.Username, "path", req.URL.Path, "remote_addr", req.RemoteAddr)
		r.errorResponse(w, err, ErrFailedToEncodeResponse)
		if r.Metrics != nil {
			r.Metrics.IncCounter(LoginFailedTotal)
		}
		return
	}
}

// Create route
// TODO complete API
func (r *Route) Create(w http.ResponseWriter, req *http.Request) {
	r.Logger.Info("Create route called", "method", req.Method, "path", req.URL.Path, "remote_addr", req.RemoteAddr)
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

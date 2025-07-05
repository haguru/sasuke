package routes

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/haguru/sasuke/internal/auth"
	"github.com/haguru/sasuke/internal/metrics"
	"github.com/haguru/sasuke/internal/models"
	"github.com/haguru/sasuke/internal/userservice"
)

const (
	CreateRouteAPI  = "/create"
	MetricsRouteAPI = "/metrics"
	LoginRouteAPI   = "/login"
	SignupRouteAPI  = "/signup"

	ContentType     = "Content-Type"
	ContentTypeJson = "application/json"
)

// Route struct defines the structure for the route handler
type Route struct {
	Metrics     *metrics.Metrics        // Placeholder for metrics interface
	UserService *userservice.UserService // Placeholder for user service interface
	PrivateKey  *ecdsa.PrivateKey       // Add private key
}

// NewRoute initializes a new Route instance with the provided metrics and keys
// This function is designed to be called during the application setup phase
// where the metrics dependency can be injected.
// It allows for the Route to be created without immediately requiring the metrics to be set,
func NewRoute(metrics *metrics.Metrics, userService *userservice.UserService, privateKey *ecdsa.PrivateKey) *Route {
	// Create a new Route instance with the provided dependencies
	// Note: metrics will be injected later by the app.go when initializing the server
	return &Route{
		Metrics:    metrics, // Metrics will be set later
		UserService: userService,
		PrivateKey: privateKey,
	}
}

// Signup route handles user signup requests
// It expects a POST request with a JSON body containing user information.
// If the request method is not POST, it returns a 405 Method Not Allowed error.
// If the request body cannot be parsed, it returns a 400 Bad Request error.
// If the request is valid, it sets the response header to application/json.
// If the user is successfully registered, it responds with a 201 Created status.
// If registration fails, it returns a 409 Conflict error.
func (r *Route) Signup(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check request content type
	if req.Header.Get(ContentType) != ContentTypeJson {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	signupRequest := &models.User{}
	err := json.NewDecoder(req.Body).Decode(signupRequest)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Register user using userservice
	userID, err := r.UserService.RegisterUser(req.Context(), signupRequest.Username, signupRequest.Password)
	if err != nil {
		http.Error(w, "Failed to register user", http.StatusConflict)
		return
	}

	w.Header().Set(ContentType, ContentTypeJson)
	w.WriteHeader(http.StatusCreated)
	message := fmt.Sprintf("User created successfully with ID: %s", userID)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": message}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}


// Login route handles user login requests
// It expects a POST request with a JSON body containing user credentials.
// If the request method is not POST, it returns a 405 Method Not Allowed error.
// If the request body cannot be parsed, it returns a 400 Bad Request error.
// If the request is valid, it sets the response header to application/json.
// If the user is authenticated successfully, it generates a session token and sets it in a cookie.
// If authentication fails, it returns a 401 Unauthorized error.
// The session token is a placeholder and should be replaced with your actual session management logic.
// This function is typically used in a web application to handle user login requests.
// It is designed to be called when a user attempts to log in, providing their username and password in the request body.
// The function does not return any value; it writes the response directly to the http.ResponseWriter.
// If the login is successful, it responds with a 200 OK status and a JSON message indicating success.
// If there are any errors during the process, it responds with the appropriate HTTP status code and error message.
// Example usage:
// http.HandleFunc("/login", route.Login)
// This function is part of the Route struct, which is typically initialized with metrics for tracking login attempts.
func (r *Route) Login(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	//check request content type
	if req.Header.Get(ContentType) != ContentTypeJson {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	loginRequest := &models.User{}
	err := json.NewDecoder(req.Body).Decode(loginRequest)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Authenticate user using userservice
	authenticated, err := r.UserService.AuthenticateUser(req.Context(), loginRequest.Username, loginRequest.Password)
	if err != nil || !authenticated {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Generate session token using the stored private key
	sessionToken, err := auth.CreateToken(loginRequest.Username, r.PrivateKey)
	if err != nil {
		http.Error(w, "Failed to generate session token", http.StatusInternalServerError)
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
	})

	// Set response header to application/json
	w.Header().Set(ContentType, ContentTypeJson)

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Login successful"}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// Create route
// TODO complete API
func (r *Route) Create(w http.ResponseWriter, req *http.Request) {
	// Placeholder for create route logic

	// incrementing a counter for the route creation
	if r.Metrics != nil {
		r.Metrics.CreateRequests.Inc()
	}

	w.WriteHeader(http.StatusOK)

	// Placeholder for additional logic, e.g., creating a resource or processing the request

	// there is an error then increment the counter error
	if r.Metrics != nil {
		r.Metrics.CreateErrors.Inc()
	}

	if _, err := w.Write([]byte("Create route called")); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}
